package main

import (
	"fmt"
	"net"
	)

func getValueMeta(clientConn net.Conn,command []string,mode string) {
	if len(command) < 2 {
		debug("Insufficient arguments")
		clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}

	key := command[1]

	//Check if key available in data store
	val,ok := m[key]

	if ok == false {
		clientConn.Write([]byte("ERR_NOT_FOUND\r\n"))
		return
	}

	// If value is present print it
	if mode == "value" {
		clientConn.Write([]byte("VALUE "+fmt.Sprintf("%d",val.numbytes)+"\r\n"))
	} else if mode == "meta" {
		clientConn.Write([]byte("VALUE "+fmt.Sprintf("%d %d %d",val.version,val.exptime,val.numbytes)+"\r\n"))
	}
		
	clientConn.Write([]byte(string(val.val)+"\r\n"))

}