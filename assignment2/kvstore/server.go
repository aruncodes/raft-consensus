package main

import (
	"assignment2/raft"
	"bufio"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

//Make it true if server should log to STDOUT
const LOG_MESSAGES = true

//Errors
const (
	ERR_INTERNAL  = "ERR_INTERNAL"
	ERR_CMD_ERR   = "ERR_CMD_ERR"
	ERR_NOT_FOUND = "ERR_NOT_FOUND"
	ERR_VERSION   = "ERR_VERSION"
)

//A command from client
type Command raft.Command

//Bundle for kv store handler
type KVBundle struct {
	command Command //Command to be executed
	// command      []byte
	responseChan chan string //Channel to send response
}

//Value of the key-value pair to be stored in datastore
type value struct {
	val                        []byte
	numbytes, version, exptime int64
}

//KV Store
var kvstore map[string]value

//Command queue
var kvQueue chan KVBundle

//Raft
var raftObj *raft.Raft

func main() {

	if !LOG_MESSAGES {
		log.SetOutput(ioutil.Discard)
	}

	//Server should get server id as an argument
	if len(os.Args) < 2 {
		log.Print(os.Args[0] + " <server id>")
		return
	}

	serverID, err := strconv.ParseInt(os.Args[1], 10, 32)

	if err != nil {
		log.Print("Server ID not valid")
		return
	}

	//Create a new raft
	raftObj, err = raft.NewRaft(&raft.ClusterInfo, int(serverID))

	if err != nil {
		log.Print(err.Error())
		return
	}

	//Start server
	startServer(raftObj.ClientPort)
}

func startServer(port int) {
	log.Print("Starting server..")

	//Listen to TCP connection on specified port
	conn, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(port), 10))
	if err != nil {
		log.Print("Error listening to port:" + err.Error())
		return
	}

	//Close connection when function exits
	defer conn.Close()

	//Init queue
	kvQueue = make(chan KVBundle)

	//Start kv store handler
	go kvStoreHandler()

	log.Print("Server started..")

	for {
		//Wait for connections from clients
		client, err := conn.Accept()

		if err != nil {
			log.Print("Error accepting connection :" + err.Error())
			continue
		}

		//Handle each client in a separate go routine
		go handleClient(client)
	}
}

func handleClient(clientConn net.Conn) {

	//Close connection when client is done
	defer clientConn.Close()

	// Server the client till he exits
	for {

		//Input scanner
		scanner := bufio.NewScanner(clientConn)
		scanner.Split(bufio.ScanLines) //Treat command as string with \n
		scanner.Scan()
		buf := scanner.Text()

		if scanner.Err() != nil {
			log.Print("Command Read Error")
			sendError(clientConn, ERR_INTERNAL)
			return
		}

		command, err := parseInput(buf)

		if err != "" {
			ret := sendError(clientConn, err)
			if !ret {
				log.Print("Client disconnected/broken pipe")
				return
			}
			continue
		}
		if command.Cmd == "set" || command.Cmd == "cas" {
			//Read data bytes of command.Length length
			scanner.Split(bufio.ScanBytes) //Treat each byte as a token
			dataBytes := make([]byte, command.Length)

			for i := int64(0); i < command.Length; i++ {
				scanner.Scan() // Read bytes of specified length
				if scanner.Err() != nil {
					log.Print("Data Read Error")
					sendError(clientConn, ERR_INTERNAL)
					return
				}
				dataBytes[i] = scanner.Bytes()[0]
			}
			command.Value = string(dataBytes)
		}

		response := "NO_RESPONSE"

		//Raft code starts
		_, er := raftObj.Append(raft.Command(command))

		if er != nil {
			log.Print(er.Error())
			response = "REDIRECT " + strconv.Itoa(raftObj.LeaderID)
		} else {
			if command.Cmd != "get" && command.Cmd != "getm" {

				for { //Wait till we get majority
					majority := raft.RequestAppendEntriesRPC(raft.ClusterInfo, raftObj)
					if majority {
						break
					}
					time.Sleep(1 * time.Second)
				}
			}

			ackChannel := make(chan string)
			kvBundle := KVBundle{command, ackChannel}

			kvQueue <- kvBundle     //Send bundle
			response = <-ackChannel //Wait for response

		}
		//Raft code ends

		ret := WriteTCP(clientConn, response+"\r\n") //Write response to client

		if !ret {
			log.Print("Client disconnected/broken pipe")
			break
		}
	}

}

func WriteTCP(clientConn net.Conn, data string) bool {
	//Write to TCP connection
	_, err := clientConn.Write([]byte(data))
	if err != nil {
		log.Print("Write Error:" + err.Error())
		return false
	}
	return true
}

func sendError(clientConn net.Conn, error string) bool {
	return WriteTCP(clientConn, error+"\r\n")
}
