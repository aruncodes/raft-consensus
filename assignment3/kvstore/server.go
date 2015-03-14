package main

import (
	"assignment3/raft"
	"io/ioutil"
	"log"
	"net"
	// "os"
	"strconv"
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

type Command raft.Command //A command from client

//Response bundle from kv store to connection handler
type KVResponse struct {
	lsn      raft.Lsn //Log sequence number
	response string   //Response
}

//Value of the key-value pair to be stored in datastore
type value struct {
	val                        []byte
	numbytes, version, exptime int64
}

// var kvstore map[string]value        //KV Store
// var commitCh chan raft.LogEntry     //Channel from raft to KV Store
// var kvResponse chan KVResponse      //Response channel
// var clientMap map[raft.Lsn]net.Conn //Save all client connections with their Lsn
// var raftObj *raft.Raft //Raft

func main() {

	if !LOG_MESSAGES { //Disable debuging
		log.SetOutput(ioutil.Discard)
	}
	/*
		//Server should get server id as an argument
		if len(os.Args) < 2 {
			log.Print(os.Args[0] + " <server id>")
			return
		}

		//Parse server id
		serverID, err := strconv.ParseInt(os.Args[1], 10, 32)
		if err != nil {
			log.Print("Server ID not valid")
			return
		}
	*/

	for i := 0; i <= 4; i++ { //Create 5 go routines representing 5 raft servers

		//Start server
		if i == 4 {
			startServer(i)
		} else {
			go startServer(i)
		}
	}
}

func startServer(serverID int) {
	log.Print("Starting server..")

	commitCh := make(chan raft.LogEntry, 10) //Commit channel from raft to kvstore
	kvResponse := make(chan KVResponse, 10)  //Response channel from kvstore to clientManger
	clientMap := make(map[raft.Lsn]net.Conn) //Create client map,Saves all client connections with their Lsn

	//Create a new raft(s) and pass commit channel
	raftObj, err := raft.NewRaft(&raft.ClusterInfo, serverID, commitCh)

	if err != nil {
		log.Print(err.Error())
		return
	}

	//Listen to TCP connection on specified port
	conn, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(raftObj.ClientPort), 10))
	if err != nil {
		log.Print("Error listening to port:" + err.Error())
		return
	}

	defer conn.Close() //Close connection when function exits

	go kvStoreHandler(commitCh, kvResponse)              //Start kv store handler
	go clientConnManager(raftObj, clientMap, kvResponse) //Manage client connections

	log.Print("Server started..")

	for {
		//Wait for connections from clients
		client, err := conn.Accept()

		if err != nil {
			log.Print("Error accepting connection :" + err.Error())
			continue
		}

		//Handle one command for now
		//Client manager will deal with more after one
		go handleOneCommand(client, KVResponse{}, raftObj, clientMap) //No response if first time
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
