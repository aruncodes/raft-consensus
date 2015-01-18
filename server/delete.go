package main

import (
	"net"
	)

func deleteValue(clientConn net.Conn,command []string) {
	if len(command) < 2 {
		debug("Insufficient arguments")
		clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}

	key := command[1]
	
	//Check if key available in data store
	_,ok := m[key]

	if ok == false {
		clientConn.Write([]byte("ERR_NOT_FOUND\r\n"))
		return
	}

	// If value is present delete it
	delete(m,key)
		
	clientConn.Write([]byte("DELETED\r\n"))
}