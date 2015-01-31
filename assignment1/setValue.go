package main

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

/* Add Values to datastore*/
func setValue(clientConn net.Conn, command []string, data string) {

	if len(command) < 4 {
		debug("Insufficient arguments")
		WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
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
			WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
			return
		}
	} else if len(command) > 5 {
		//Invalid syntax
		WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
		return
	}

	//Expiry Time
	exptime, err := strconv.ParseInt(command[2], 10, 64)
	if err != nil {
		debug("Invalid expiry time specified.")
		WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
		return
	}
	if exptime < 0 {
		debug("Expiry time cannot be negative.")
		WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
		return
	}

	//Number of Bytes
	numbytes, err := strconv.ParseInt(command[3], 10, 64)
	if err != nil {
		debug("Invalid number of bytes specified.")
		WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
		return
	}
	if numbytes < 1 {
		debug("Number of bytes must be positive.")
		WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
		return
	}

	//Check i already exist and rewriting
	_, ok := m[key]

	if ok == true {
		debug("Value already exists.")
		WriteTCP(clientConn, "ERR_VERSION\r\n")
		return
	}

	// Validation completed

	//Trim \r\n from end
	datastring := strings.TrimRight(data, "\n\r\000")

	//Get a random number as version
	version := int64(rand.Intn(10000))

	//Trim datastring to numbytes length
	if int64(len(datastring)) > numbytes {
		datastring = datastring[:numbytes]
	}

	//Add value to keystore
	m[key] = value{[]byte(datastring), numbytes, version, exptime}

	//Inform expiryHandler
	sendExpiry := func() {
		ack := make(chan bool)
		requestQueue <- dataStoreBundle{nil, []string{"expire", key, fmt.Sprintf("%d", version)}, "", ack}
		<-ack
	}

	//Set expiry timer
	if exptime > 0 {
		time.AfterFunc(time.Duration(exptime)*time.Second, sendExpiry)
	}

	//Reply if required
	if !noreply {
		WriteTCP(clientConn, fmt.Sprintf("OK %d\r\n", version))
	}
}
