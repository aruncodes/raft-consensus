package main

import (
	"fmt"
	"net"
)

func getValueMeta(clientConn net.Conn, command []string, mode string) {
	if len(command) != 2 {
		debug("Arguments invalid")
		WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
		return
	}

	key := command[1]

	//Check if key available in data store
	val, ok := m[key]

	if ok == false {
		WriteTCP(clientConn, "ERR_NOT_FOUND\r\n")
		return
	}

	// If value is present print it
	if mode == "value" {
		WriteTCP(clientConn, "VALUE "+fmt.Sprintf("%d", val.numbytes)+"\r\n")
	} else if mode == "meta" {
		WriteTCP(clientConn, "VALUE "+fmt.Sprintf("%d %d %d", val.version, val.exptime, val.numbytes)+"\r\n")
	}

	WriteTCP(clientConn, string(val.val)+"\r\n")

}
