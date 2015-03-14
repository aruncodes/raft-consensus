package main

import (
	"assignment3/raft"
	"bufio"
	"log"
	"net"
	"strconv"
	"sync"
)

var lock sync.Mutex //Lock for adding to client map

//Always runing go routine
//Receive response and lsn from kvstore,
//get the corresponding client connection object,
//send response to it and serve another command
func clientConnManager(raftObj *raft.Raft, clientMap map[raft.Lsn]net.Conn, kvResponse chan KVResponse) {

	for {
		resp := <-kvResponse //Receive response from kv store

		lsnKey := resp.lsn

		//Access connection object and remove from client map
		lock.Lock()
		conn := clientMap[lsnKey]
		delete(clientMap, lsnKey) //Not required anymore, so remove from map
		lock.Unlock()

		//Send connection and response to actual client handler
		go handleOneCommand(conn, resp, raftObj, clientMap)
	}
}

//Handle exactly one command
//If parsed succesfully, append to raft log
//add to global clientMap and quit
//clientConnManager() will call again for next command
func handleOneCommand(clientConn net.Conn, response KVResponse, raftObj *raft.Raft, clientMap map[raft.Lsn]net.Conn) {

	//If there is some response of previous command, then
	//send that first before serving new one

	if response.lsn != 0 { //When called for first time, it will be 0
		ret := WriteTCP(clientConn, response.response+"\r\n") //Write response to client

		if !ret {
			log.Print("Client disconnected/broken pipe")
			clientConn.Close()
			return
		}
	}

	// Server the client for one command
	for {
		//Input scanner
		scanner := bufio.NewScanner(clientConn)
		scanner.Split(bufio.ScanLines) //Treat command as string with \n
		scanner.Scan()
		buf := scanner.Text()

		if scanner.Err() != nil {
			log.Print("Command Read Error")
			sendError(clientConn, ERR_INTERNAL)
			clientConn.Close()
			break
		}

		command, err := parseInput(buf)

		if err != "" {
			ret := sendError(clientConn, err)
			if !ret {
				log.Print("Client disconnected/broken pipe")
				clientConn.Close()
				break
			}
			continue // Something is wrong, retry command
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
		// Everything upto here (parsing command) is fine

		//Future TODO: encode command struct to byte stream

		//Append to log , add to client map and wait for client handler to
		// continue handling when the response comes back

		logEntry, er := raftObj.Append(raft.Command(command))

		if er != nil { //Possibly not the leader, so redirect
			log.Print(er.Error())

			response := "REDIRECT " + strconv.Itoa(raftObj.LeaderID)
			ret := WriteTCP(clientConn, response+"\r\n") //Write response to client

			if !ret {
				log.Print("Client disconnected/broken pipe")
				clientConn.Close()
				break
			}

		}

		//Add to client map
		lsn := logEntry.Lsn()

		lock.Lock()
		clientMap[lsn] = clientConn
		lock.Unlock()

		//Stop and wait for clientConnManger() to do something
		break

	}

}
