package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
	)

const PORT = "9000"

const DEBUG = true

type value struct { 
	val []byte
	numbytes,version, exptime int64
	addTime time.Time
}

var m map[string] value

func main() {
	
	fmt.Println("Starting server..")

	//Listen to TCP connection on specified port
	conn,err := net.Listen("tcp",":"+PORT)
	if err != nil {
		fmt.Println("Error listening to port:",err.Error())
		os.Exit(1)
	}

	// Close listener when application closes
	defer closeConn(conn)

	//Initialize datastore
	m = make(map[string] value)

	fmt.Println("Server started..")

	//Wake up expiry handler
	go expiryHandler()
	
	for {
		//Wait for connections from clients
		client,err := conn.Accept()

		if err != nil {
			fmt.Println("Error accepting connection :",err.Error())
			os.Exit(1)
		}

		go handleClient(client)
		// break
	}

}

func closeConn(c net.Listener) {
	fmt.Println("Closing server..")
	c.Close()
}

func handleClient(clientConn net.Conn) {

	//Close connection when client is done
	defer clientConn.Close()


	// Server the client till he exits
	for {
		buf := make([]byte, 512)
		_, err := clientConn.Read(buf)

		if err != nil {
			debug("Read Error:"+err.Error())
			clientConn.Write([]byte("ERR_INTERNAL\r\n"))
			break
		}

		// debug("Read Msg: |",string(buf)," |")

		command := strings.Split(strings.Trim(string(buf),"\n \r\000")," ")

		switch command[0] {
			case "set" : //fmt.Println("Command : set | Arguments :",command[1:])
						setValue(clientConn,command)
			case "get" : //fmt.Println("Command : get | Arguments :",command[1:])
						getValueMeta(clientConn,command,"value")
			case "getm" : //fmt.Println("Command : getm | Arguments :",command[1:])
						getValueMeta(clientConn,command,"meta")
			case "cas" : //fmt.Println("Command : cas | Arguments :",command[1:])
						casValue(clientConn,command)
			case "delete" : //fmt.Println("Command : delete | Arguments :",command[1:])
						deleteValue(clientConn,command)

			default: clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
		}

		fmt.Println(m);
	}
}

func debug(msg string) {
	if DEBUG {
		fmt.Println(msg)
	}
}
