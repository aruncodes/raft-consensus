package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

/*Compare and Swap values if versions match*/
func casValue(clientConn net.Conn, command []string, data string) {
	if len(command) < 5 {
		debug("Insufficient arguments")
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

	version, err := strconv.ParseInt(command[3], 10, 64)
	if err != nil {
		WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
		debug("Invalid version specified :" + command[3])
		return
	}

	if val.version == version {
		//Same version, so proceed
		// Validation

		//No reply
		noreply := false

		if len(command) == 6 {
			if command[5] == "noreply" {
				noreply = true
			} else {
				//Invalid syntax
				WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
				return
			}
		} else if len(command) > 6 {
			//Invalid syntax
			WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
			return
		}

		//Expiry Time
		exptime, err := strconv.ParseInt(command[2], 10, 64)
		if err != nil {
			WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
			debug("Invalid expiry time specified.")
			return
		}
		if exptime < 0 {
			WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
			debug("Expiry time cannot be negative.")
			return
		}

		//Number of Bytes
		numbytes, err := strconv.ParseInt(command[4], 10, 64)
		if err != nil {
			WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
			debug("Invalid number of bytes specified.")
			return
		}
		if numbytes < 1 {
			WriteTCP(clientConn, "ERR_CMD_ERR\r\n")
			debug("Number of bytes must be positive.")
			return
		}

		// Validation completed

		//Trim \r\n from end
		datastring := strings.TrimRight(data, "\n\r\000")

		//Trim datastring to numbytes length
		if int64(len(datastring)) > numbytes {
			datastring = datastring[:numbytes]
		}

		//Increment version
		version++

		//Add value to keystore
		m[key] = value{[]byte(datastring), numbytes, version, exptime}

		//Inform expiryHandler
		sendExpiry := func() {
			ack := make(chan bool)
			writeQueue <- dataStoreWriteBundle{nil, []string{"expire", key, fmt.Sprintf("%d", version)}, "", ack}
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

	} else {
		//Versions do not match
		WriteTCP(clientConn, "ERR_VERSION\r\n")
		return
	}
}
