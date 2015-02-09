package main

import (
	"os/exec"
	"strconv"
	"os"
	"testing"
	"time"
	"net"
	"fmt"
	"bufio"
)

const LEADER_PORT = 9000
const NUM_SERVERS = 5
const SERVER_NAME = "server" //Must be inside GOPATH

var proc []*exec.Cmd

func TestCluster(t *testing.T) {
	proc = make([] *exec.Cmd,NUM_SERVERS)

	startSingleServer(1) //Server that is not a leader
	time.Sleep(1 * time.Second)
	conn := startClient(t,"9001") //Connect

	TCPWrite(t,conn,"set something 0 10\r\nasgbdtsdhh")
	response := TCPRead(t,conn)
	checkIfExpected(t,response[:8],"REDIRECT") //Expect a redirect to leader

	TCPWrite(t,conn,"get something\r\n")
	response = TCPRead(t,conn)
	checkIfExpected(t,response[:8],"REDIRECT") //Expect a redirect to leader
	
	conn.Close() //Close previous client 

	startSingleServer(0) //Start leader
	time.Sleep(1 * time.Second)
	conn = startClient(t,"9000") //Connect to leader

	//2 servers will not get majority
	//So Start the next server after 5 seconds to get majority
	//Leader keep sending RPCs till it gets majority
	time.AfterFunc(time.Duration(5 * time.Second), func() {startSingleServer(2)})

	TCPWrite(t,conn,"set something 0 10\r\nasgbdtsdhh")
	response = TCPRead(t,conn)
	checkIfExpected(t,response[:3],"OK ") 

	TCPWrite(t,conn,"get something\r\n")
	response = TCPRead(t,conn)
	checkIfExpected(t,response[:5],"VALUE")

	killAllServers()
}

func startClient(t *testing.T,port string ) net.Conn {
	conn, err := net.Dial("tcp", "localhost:" + port)
	if err != nil {
		fmt.Println(err)
		t.Error("Cannot initialize TCP connection")
		return nil
	}
	return conn
}

func startSingleServer(id int) {
	proc[id] = exec.Command(os.Getenv("GOPATH")+"/bin/" + SERVER_NAME,strconv.Itoa(id))
	proc[id].Stdout = os.Stdout
    proc[id].Stderr = os.Stderr
	err := proc[id].Start()
	if err != nil {panic(err)}
}

func killServer(id int) {

	if proc[id] == nil {
		return
	}
	proc[id].Process.Kill()
	proc[id].Wait()
}

func startAllServers() {

	for i := 0; i < NUM_SERVERS; i++ {
		startSingleServer(i)
	}
}

func killAllServers() {
	for i := 0; i < NUM_SERVERS; i++ {
		killServer(i)
	}
}

func TCPWrite(t *testing.T, conn net.Conn, buf string) {
	_, err := conn.Write([]byte(buf + "\r\n"))
	if err != nil {
		t.Error("Cannot write to TCP socket..")
	}
}

func TCPRead(t *testing.T, conn net.Conn) string {
	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	buf := scanner.Text()

	if scanner.Err() != nil {
		t.Error("Cannot read from TCP socket..")
	}
	return buf
}

func checkIfExpected(t *testing.T, actual string, expected string) {
	if actual != expected {
		t.Error(fmt.Sprintf("Expected '%s' but got '%s'\n", expected, actual))
	}
}
