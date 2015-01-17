package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"strconv"
	)

const PORT = "9000"

type value struct { 
	val []byte
	numbytes int64
	version, exptime int64
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

	fmt.Println("Server running and listening for incomming connections..")

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
	// fmt.Println("Need to handle client here..")

	defer clientConn.Close()

	buf := make([]byte, 1024)

	_, err := clientConn.Read(buf)

	if err != nil {
		fmt.Println("Read Error:",err.Error())
	}

	// fmt.Println("Read Msg: |",string(buf)," |")

	command := strings.Split(strings.Trim(string(buf),"\n \r\000")," ")

	switch command[0] {
		case "set" : //fmt.Println("Command : set | Arguments :",command[1:])
					setValue(clientConn,command);
		case "get" : //fmt.Println("Command : get | Arguments :",command[1:])
					getValue(clientConn,command);
		case "getm" : fmt.Println("Command : getm | Arguments :",command[1:])
		case "cas" : fmt.Println("Command : cas | Arguments :",command[1:])
		case "delete" : fmt.Println("Command : delete | Arguments :",command[1:])
		default: fmt.Println("ERRCMDERR")
	}

	/*
	msgLen, err := clientConn.Write(buf)
	
	if err != nil {
		fmt.Println("Write Error:",err.Error())
	}

	fmt.Println("Wrote ",msgLen," bytes")
	*/
	fmt.Println(m);
}

func setValue(clientConn net.Conn,command []string) {

	if len(command) < 4 {
		fmt.Println("Insufficient arguments")
		return
	}
	key := command[1]

	// Validation

	//No reply
	noreply := false	

	if len(command) == 5 && command[4] == "noreply" {
		noreply = true
	}

	//Expiry Time
	exptime, err := strconv.ParseInt(command[2],10,10)
	if err != nil {
		fmt.Println("Invalid expiry time specified.")
		return
	}
	if exptime < 0 {
		fmt.Println("Expiry time cannot be negative.")
		return	
	}

	//Number of Bytes
	numbytes, err := strconv.ParseInt(command[3],10,10)
	if err != nil {
		fmt.Println("Invalid number of bytes specified.")
		return
	}
	if numbytes < 1 {
		fmt.Println("Number of bytes must be positive.")
		return	
	}

	// Validation completed

	//Read value
	buf := make([]byte,numbytes)
	_, err = clientConn.Read(buf)

	if err != nil {
		fmt.Println("Read Error:",err.Error())
	}
	
	//Trim \r\n from end
	datastring := strings.TrimRight(string(buf),"\n\r\000")
	//Add value to keystore
	m[key] = value{[]byte(datastring),numbytes,1000,exptime}

	//Reply if required
	if !noreply {
		clientConn.Write([]byte("OK 1000\r\n"))
	}
}

func getValue(clientConn net.Conn,command []string) {
	if len(command) < 2 {
		fmt.Println("Insufficient arguments")
		return
	}

	key := command[1]

	//Check if key available in data store
	val,ok := m[key]

	if ok == false {
		clientConn.Write([]byte("ERRNOTFOUND\r\n"))
		return
	}

	// If value is present print it
	clientConn.Write([]byte("VALUE "+fmt.Sprintf("%d",val.numbytes)+"\r\n"))
	clientConn.Write([]byte(string(val.val)+"\r\n"))

}