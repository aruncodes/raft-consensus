package main

import (
	"net"
	"strings"
	"strconv"
	)

/* Add Values to datastore*/
func setValue(clientConn net.Conn,command []string) {

	if len(command) < 4 {
		debug("Insufficient arguments")
		clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
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
	numbytes, err := strconv.ParseInt(command[3],10,10)
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

	//Read value
	buf := make([]byte,numbytes)
	_, err = clientConn.Read(buf)

	if err != nil {
		debug("Read Error:"+err.Error())
		clientConn.Write([]byte("ERR_INTERNAL\r\n"))
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