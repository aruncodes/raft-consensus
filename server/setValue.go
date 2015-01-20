package main

import (
	"net"
	"strings"
	"strconv"
	"time"
	"math/rand"
	"fmt"
	)

/* Add Values to datastore*/
func setValue(clientConn net.Conn,command []string,data string) {

	if len(command) < 4 {
		debug("Insufficient arguments")
		clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}
	key := command[1]

	// Validation

	//No reply
	noreply := false	

	if len(command) == 5 {
		if command[4] == "noreply" {
			noreply = true
		} else {
			//Invalid syntax
			clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
			return
		}
	} else if len(command) > 5 {
		//Invalid syntax
		clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
		return	
	}

	//Expiry Time
	exptime, err := strconv.ParseInt(command[2],10,64)
	if err != nil {
		debug("Invalid expiry time specified.")
		clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}
	if exptime < 0 {
		debug("Expiry time cannot be negative.")
		clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
		return	
	}

	//Number of Bytes
	numbytes, err := strconv.ParseInt(command[3],10,64)
	if err != nil {
		debug("Invalid number of bytes specified.")
		clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}
	if numbytes < 1 {
		debug("Number of bytes must be positive.")
		clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
		return	
	}

	// Validation completed

	//Trim \r\n from end
	datastring := strings.TrimRight(data,"\n\r\000")

	//Get a random number as version
	version := int64(rand.Intn(10000))

	//Add value to keystore
	m[key] = value{[]byte(datastring),numbytes,version,exptime,time.Now()}

	//Inform expiryHandler
	go dataStoreChanged(key,ADD)

	//Reply if required
	if !noreply {
		clientConn.Write([]byte(fmt.Sprintf("OK %d\r\n",version)))
	}
}