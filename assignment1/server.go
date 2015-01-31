/*
	A Simple Memcached Clone written in Go

	Author: Arun Babu (143050032)
*/

package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Port in which the server should listen to
const PORT = "9000"

//Should the server print debug messages while handling client
const DEBUG = false

//Value of the key-value pair to be stored in datastore
type value struct {
	val                        []byte
	numbytes, version, exptime int64
}

/* This bundle is sent into queue for datastore requests*/
type dataStoreBundle struct {
	clientConn net.Conn
	command    []string
	data       string
	ack        chan bool // The handle client will wait on this channel for ack
}

//request queue
var requestQueue chan dataStoreBundle

//Data store as a map
var m map[string]value

func main() {

	debug("Starting server..")

	//Listen to TCP connection on specified port
	conn, err := net.Listen("tcp", ":"+PORT)
	if err != nil {
		debug("Error listening to port:" + err.Error())
		return
	}

	// Close listener when application closes
	defer closeConn(conn)

	//Initialize datastore
	m = make(map[string]value)

	//Initialize request queue
	requestQueue = make(chan dataStoreBundle, 100)

	//Wake up request handler
	go dataStoreHandler()

	debug("Server started..")

	for {
		//Wait for connections from clients
		client, err := conn.Accept()

		if err != nil {
			debug("Error accepting connection :" + err.Error())
			continue
		}

		//Handle each client in a seperate
		go handleClient(client)
	}

}

func closeConn(c net.Listener) {
	debug("Closing server..")
	c.Close()
}

func handleClient(clientConn net.Conn) {

	//Close connection when client is done
	defer clientConn.Close()

	//Input scanner
	scanner := bufio.NewScanner(clientConn)

	// Server the client till he exits
	for {

		scanner.Split(bufio.ScanLines) //Treat command as string with \n
		scanner.Scan()
		buf := scanner.Text()

		if scanner.Err() != nil {
			debug("Command Read Error")
			WriteTCP(clientConn, "ERR_INTERNAL\r\n")
			return
		}
		// debug("Read Msg: |"+string(buf)+" |")

		command := strings.Split(strings.Trim(buf, "\n \r\000"), " ")
		switch command[0] {

		case "get", "getm", "delete":
			ack := make(chan bool)
			requestQueue <- dataStoreBundle{clientConn, command, "", ack}
			<-ack // Wait for ack after operation

		case "set", "cas":
			//Read data line for set and cas
			dataLength := getDataLen(command)
			if dataLength < 1 {
				WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
				debug("Invalid number of bytes specified.")
				continue
			}
			scanner.Split(bufio.ScanBytes) //Treat each byte as a token

			dataBytes := make([]byte, dataLength)
			for i := int64(0); i < dataLength; i++ {
				scanner.Scan() // Read bytes of specified length

				if scanner.Err() != nil {
					debug("Data Read Error")
					WriteTCP(clientConn, "ERR_INTERNAL\r\n")
					return
				}
				dataBytes[i] = scanner.Bytes()[0]
			}
			data := string(dataBytes)

			ack := make(chan bool)
			//Send the bundle into request queue
			requestQueue <- dataStoreBundle{clientConn, command, data, ack}
			<-ack // Wait for ack after operation

		default:
			WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
		}

		if DEBUG { //Print map as debug
			fmt.Println(m)
		}
	}
}

func WriteTCP(clientConn net.Conn, data string) {
	//Write to TCP connection
	_, err := clientConn.Write([]byte(data))
	if err != nil {
		debug("Write Error:" + err.Error())
	}
}

//Make sure the requests are sequential
func dataStoreHandler() {

	for {

		//Receive request bundle from queue and process sequentially
		requestBundle := <-requestQueue

		debug("Request received : " + requestBundle.command[0])
		switch requestBundle.command[0] {

		case "get":
			getValueMeta(requestBundle.clientConn, requestBundle.command, "value")
		case "getm":
			getValueMeta(requestBundle.clientConn, requestBundle.command, "meta")
		case "set":
			setValue(requestBundle.clientConn, requestBundle.command, requestBundle.data)
		case "cas":
			casValue(requestBundle.clientConn, requestBundle.command, requestBundle.data)
		case "delete":
			deleteValue(requestBundle.clientConn, requestBundle.command)
		case "expire":
			expiryHandler(requestBundle.command)
		}

		//Send ACK
		requestBundle.ack <- true
	}
}

//Get the length of the data to be read for set and cas
func getDataLen(command []string) int64 {

	var len_string string = "0"

	switch command[0] {
	case "set":
		if len(command) > 3 {
			len_string = command[3]
		}
	case "cas":
		if len(command) > 4 {
			len_string = command[4]
		}
	}

	numbytes, err := strconv.ParseInt(len_string, 10, 64)
	if err != nil {
		debug("Invalid number of bytes specified.")
		return 0
	}
	if numbytes < 1 {
		debug("Number of bytes must be positive.")
		return 0
	}

	return numbytes
}

func debug(msg string) {
	if DEBUG {
		fmt.Println(msg)
	}
}
