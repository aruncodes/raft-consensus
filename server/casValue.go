package main

import (
	"fmt"
	"net"
	"strings"
	"strconv"
	"time"
	)

/*Compare and Swap values if versions match*/
func casValue(clientConn net.Conn,command []string) {
	if len(command) < 5 {
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

	version, err := strconv.ParseInt(command[3],10,64)
	if err != nil {
		clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
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
				clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
				return
			}
		} else if len(command) > 6 {
			//Invalid syntax
			clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
			return	
		}

		//Expiry Time
		exptime, err := strconv.ParseInt(command[2],10,64)
		if err != nil {
			clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
			debug("Invalid expiry time specified.")
			return
		}
		if exptime < 0 {
			clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
			debug("Expiry time cannot be negative.")
			return	
		}

		//Number of Bytes
		numbytes, err := strconv.ParseInt(command[4],10,64)
		if err != nil {
			clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
			debug("Invalid number of bytes specified.")
			return
		}
		if numbytes < 1 {
			clientConn.Write([]byte("ERR_CMD_ERR\r\n"))
			debug("Number of bytes must be positive.")
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

		//Increment version
		version++

		//Add value to keystore
		m[key] = value{[]byte(datastring),numbytes,version,exptime,time.Now()}
	
		//Inform expiryHandler
		go dataStoreChanged(key,MODIFY)

		//Reply if required
		if !noreply {
			clientConn.Write([]byte(fmt.Sprintf("OK %d\r\n",version)))
		}

	} else {
		//Versions do not match
		clientConn.Write([]byte("ERR_VERSION\r\n"))
		return
	}
}