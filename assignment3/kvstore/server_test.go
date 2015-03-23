package main

import (
	"assignment3/raft"
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	// "os/exec"
	// "strconv"
	"testing"
	"time"
)

// const NUM_SERVERS = 5
// const SERVER_NAME = "kvstore" //Must be inside GOPATH
// var proc []*exec.Cmd

//Test procedure:
//	Kill one leader, check if new leader gets elected
//	Kill another, check the same
//	Resurrect previous as follower, check if it sinks in
//  Resurrect first one as leader, check if it sinks in
func TestRaft1(t *testing.T) {

	//Remove any state recovery files
	for i := 0; i < 5; i++ {
		os.Remove(fmt.Sprintf("%s_S%d.state", raft.FILENAME, i))
	}

	go main() //Starts all servers

	var leaderId1, leaderId2 int
	ack := make(chan bool)

	time.AfterFunc(1*time.Second, func() {
		checkIfLeaderExist(t)
		leaderId1 = raft.KillLeader() //Kill leader
		log.Print("Killed leader:", leaderId1)
	})

	time.AfterFunc(2*time.Second, func() {
		checkIfLeaderExist(t)
		leaderId2 = raft.KillLeader() //Kill current leader again
		log.Print("Killed leader:", leaderId2)
	})

	time.AfterFunc(3*time.Second, func() {
		checkIfLeaderExist(t)
		raft.ResurrectServer(leaderId2) //Resurrect last killed as follower
		log.Print("Resurrected previous leader:", leaderId2)
	})

	time.AfterFunc(4*time.Second, func() {
		checkIfLeaderExist(t)
		raft.ResurrectServerAsLeader(leaderId1) //Ressurect first one as leader
		log.Print("Resurrected previous leader:", leaderId1)
	})

	time.AfterFunc(5*time.Second, func() {
		checkIfLeaderExist(t)
		ack <- true
	})

	<-ack
}

//Kill 3 servers and make sure no leader gets elected
func TestRaft2(t *testing.T) {
	ack := make(chan bool)

	leader := raft.GetLeaderId()

	//Kill some one who is not a leader
	s1 := (leader + 1) % 5
	raft.KillServer(s1)
	t.Log("Killed ", s1)

	//Once more
	s2 := (s1 + 1) % 5
	raft.KillServer(s2)
	t.Log("Killed ", s2)

	//Kill leader now
	leader = raft.KillLeader()

	//Make sure new leader doesn't get elected
	time.AfterFunc(1*time.Second, func() {
		leaderId := raft.GetLeaderId()
		if leaderId != -1 {
			t.Error("Leader should not get elected!")
		}
		ack <- true
	})
	<-ack

	//Resurrect for next test cases
	raft.ResurrectServer(leader)
	raft.ResurrectServer(s1)
	raft.ResurrectServer(s2)

	//Wait for 1 second for new leader to get elected
	time.AfterFunc(1*time.Second, func() { ack <- true })
	<-ack
}

func checkIfLeaderExist(t *testing.T) bool {
	if raft.GetLeaderId() == -1 {
		t.Error("New leader not elected in time!")
		return false
	}
	t.Log("Leader election successful")
	return true
}

//Test if log after append is same accross all servers
//Check everyones log with leader's
func TestLogReplication1(t *testing.T) {

	ack := make(chan bool)

	//Get leader
	leaderId := raft.GetLeaderId()

	//Append a log entry to leader as client
	raft.InsertFakeLogEntry(leaderId)

	leaderLog := raft.GetLogAsString(leaderId)
	// log.Println(leaderLog)

	time.AfterFunc(1*time.Second, func() { ack <- true })
	<-ack //Wait for 1 second for log replication to happen

	//Get logs of all others and compare with each
	for i := 0; i < 5; i++ {
		checkIfExpected(t, raft.GetLogAsString(i), leaderLog)
	}

}

//After crashing and recovering,
//test if log is replicated after some time.
//
//Procedure: Kill a server, insert a log entry,
//resurrect after some time, see if log match with leader now
func TestLogReplication2(t *testing.T) {
	ack := make(chan bool)

	//Kill one server
	raft.KillServer(1)

	//Append log to server
	time.AfterFunc(1*time.Second, func() {
		//Get leader
		leaderId := raft.GetLeaderId()

		//Append a log entry to leader as client
		raft.InsertFakeLogEntry(leaderId)
	})

	//Resurrect old server after enough time for other to move on
	time.AfterFunc(2*time.Second, func() {
		//Resurrect old server
		raft.ResurrectServer(1)
	})

	//Check log after some time to see if it matches with current leader
	time.AfterFunc(3*time.Second, func() {
		leaderId := raft.GetLeaderId()
		leaderLog := raft.GetLogAsString(leaderId)
		serverLog := raft.GetLogAsString(1)

		checkIfExpected(t, serverLog, leaderLog)

		ack <- true
	})

	<-ack

}

/*
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
*/
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
