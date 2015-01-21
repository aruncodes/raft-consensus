package main

import (
	"testing"
	"net"
	"fmt"
	"strings"
	"time"
)

type RequestResponsePair struct {
	req string
	data string
	resp string
}

var rrp = []RequestResponsePair {

	{ "get arun","", "ERR_NOT_FOUND" },			//Get if key not set
	{ "set arun 10 100","my name is arun", "OK "},		//Set a key-value
	{ "set arun 14 110","my name is babu", "ERR_VERSION"},		//Overwrite already set variable
	{ "cas arun 10 1000 100","wowo","ERR_VERSION"},		//CAS without correct version
	{ "get arun","", "VALUE 100\r\nmy name is arun"},	//Get the set vairable
	{ "delete babu","","ERR_NOT_FOUND"},		//Delete a non existing key
	{ "delete arun","","DELETED"},				//Delete existing key

}

func TestServer(t *testing.T) {

	//Start server
	go main()

	//Test connectivity
	conn, err := net.Dial("tcp", "localhost:9000")
    if err != nil {
        fmt.Println(err)
        t.Error("Cannot initialize TCP connection")
    }

    //Run test cases
    performRRPairs(t,conn)

    //Check version dependent commands
    checkVersionDependentCommands(t,conn)

    //Test expiry Handler
    checkExpiryHandler(t,conn)

    //Close
    conn.Close()
}

//Run predefined pairs in []rrp
func performRRPairs(t *testing.T,conn net.Conn) {

    for _,pair := range rrp {
    	t.Log(pair.req,pair.resp)

    	TCPWrite(t,conn,pair.req)

    	if pair.data != "" {
    		//Let server read it
    		time.Sleep(10 * time.Millisecond)

    		TCPWrite(t,conn,pair.data)
    	}

    	response := TCPRead(t,conn)
    	
    	response = strings.Trim(response,"\n \r\000")

    	//Strip off version number
    	if (pair.req[:3] == "set" || pair.req[:3] == "cas") && response[:2] == "OK"{
    		response = response[:3]
    	}

    	checkIfExpected(t,response,pair.resp);
    }
}

func checkVersionDependentCommands(t *testing.T,conn net.Conn) {
	//Check 'cas' and 'getm' since they depend of version

	//Set a variable
	TCPWrite(t,conn,"set key1 100 128")
	time.Sleep(10 * time.Millisecond)
	TCPWrite(t,conn,"value1")

	reply := TCPRead(t,conn)
	response := strings.Split(reply," ")

	checkIfExpected(t,response[0],"OK")
	version := strings.TrimRight(response[1],"\r\n\000")

	//Perform CAS
	TCPWrite(t,conn,"cas key1 15 "+version+" 120")
	time.Sleep(10 * time.Millisecond)
	TCPWrite(t,conn,"updated_value")

	reply = TCPRead(t,conn)
	response = strings.Split(reply," ")

	checkIfExpected(t,response[0],"OK")
	version = strings.TrimRight(response[1],"\r\n\000")

	//Check getm
	TCPWrite(t,conn,"getm key1")
	reply = TCPRead(t,conn)
	checkIfExpected(t,strings.Trim(reply,"\000"),"VALUE "+version+" 15 120\r\nupdated_value\r\n")
}

//Check if expiry handler is working
//set a vaue with expiry as 1 second and read it back after 1 second
func checkExpiryHandler(t *testing.T,conn net.Conn) {
	
	TCPWrite(t,conn,"set sam 1 128")
	time.Sleep(10 * time.Millisecond)
	TCPWrite(t,conn,"value_sam")

	reply := TCPRead(t,conn)
	response := strings.Split(reply," ")

	checkIfExpected(t,response[0],"OK")

	//Wait for 1 second
	time.Sleep(1 * time.Second)
	TCPWrite(t,conn,"get sam")
	reply = TCPRead(t,conn)
	checkIfExpected(t,strings.Trim(reply,"\000"),"ERR_NOT_FOUND\r\n")
}

func TCPWrite(t *testing.T,conn net.Conn,buf string) {
	_,err := conn.Write([]byte(buf + "\r\n"))
	if err != nil {
		t.Error("Cannot write to TCP socket..")
	}
}


func TCPRead(t *testing.T,conn net.Conn) string {
	buf := make([]byte, 1024)
	_,err := conn.Read(buf)
	if err != nil {
		t.Error("Cannot read from TCP socket..")
	}

	return string(buf)
}

func checkIfExpected(t *testing.T, actual string, expected string) {
	if actual != expected {
		t.Error(fmt.Sprintf("Expected '%s' but got '%s'\n",expected,actual));
	}
}