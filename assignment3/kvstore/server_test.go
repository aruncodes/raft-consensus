package main

import (
	"assignment3/raft"
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

const NUM_SERVERS = 5
const SERVER_NAME = "kvstore" //Must be inside GOPATH

var proc []*exec.Cmd

func TestRaft(t *testing.T) {

	go main() //Starts all servers

	var leaderId1, leaderId2 int
	ack := make(chan bool)

	time.AfterFunc(1*time.Second, func() {
		leaderId1 = raft.KillLeader()
		log.Print("Killed leader:", leaderId1)
		// raft.MakeServerUnavailable(2)
	})

	time.AfterFunc(2*time.Second, func() {
		leaderId2 = raft.KillLeader()
		log.Print("Killed leader:", leaderId2)
		// raft.MakeServerAvailable(leaderId)
	})

	time.AfterFunc(5*time.Second, func() {
		raft.ResurrectServerAsLeader(leaderId1)
		log.Print("Resurrected previous leader:", leaderId1)
		// raft.MakeServerAvailable(leaderId)
	})

	time.AfterFunc(3*time.Second, func() {
		raft.ResurrectServer(leaderId2)
		log.Print("Resurrected previous leader:", leaderId2)
		// raft.MakeServerAvailable(leaderId)
	})

	time.AfterFunc(10*time.Second, func() { ack <- true })

	<-ack
}

func _TestCluster(t *testing.T) {
	proc = make([]*exec.Cmd, NUM_SERVERS)

	startSingleServer(1) //Server that is not a leader
	time.Sleep(1 * time.Second)
	conn := startClient(t, "9001") //Connect

	TCPWrite(t, conn, "set something 0 10\r\nasgbdtsdhh")
	response := TCPRead(t, conn)
	checkIfExpected(t, response[:8], "REDIRECT") //Expect a redirect to leader

	TCPWrite(t, conn, "get something\r\n")
	response = TCPRead(t, conn)
	checkIfExpected(t, response[:8], "REDIRECT") //Expect a redirect to leader

	conn.Close() //Close previous client

	startSingleServer(0) //Start leader
	time.Sleep(1 * time.Second)
	conn = startClient(t, "9000") //Connect to leader

	//2 servers will not get majority
	//So Start the next server after 5 seconds to get majority
	//Leader keep sending RPCs till it gets majority
	time.AfterFunc(time.Duration(5*time.Second), func() { startSingleServer(2) })

	TCPWrite(t, conn, "set something 0 10\r\nasgbdtsdhh")
	response = TCPRead(t, conn)
	checkIfExpected(t, response[:3], "OK ")

	TCPWrite(t, conn, "get something\r\n")
	response = TCPRead(t, conn)
	checkIfExpected(t, response[:5], "VALUE")

	killAllServers()
}

func startClient(t *testing.T, port string) net.Conn {
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		fmt.Println(err)
		t.Error("Cannot initialize TCP connection")
		return nil
	}
	return conn
}

func startSingleServer(id int) {
	proc[id] = exec.Command(os.Getenv("GOPATH")+"/bin/"+SERVER_NAME, strconv.Itoa(id))
	proc[id].Stdout = os.Stdout
	proc[id].Stderr = os.Stderr
	err := proc[id].Start()
	if err != nil {
		panic(err)
	}
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
