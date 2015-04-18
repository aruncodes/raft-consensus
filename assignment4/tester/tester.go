package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	STATE_FILENAME = "saved"
	SERVER_NAME    = "./kvstore"
	NUM_SERVERS    = 5
	START_PORT     = 9000
	LOG_TEST       = true
	LOG_TEST_LABEL = true
	LOG_MSG        = true
	LOG_TCP_MSGS   = true
	LOG_VERBOSE    = true
)

var liveServers []*exec.Cmd
var ERROR_HAPPENED = false
var waitSec = 2 * time.Second //Time to wait for leader election

func main() {
	LogVerbose("Tester starting..")

	liveServers = make([]*exec.Cmd, NUM_SERVERS)
	ResetServerState()

	TestElectionAndReplication()
	TestStress()

	LogVerbose("Tester finished!")

	fmt.Println("----------------------")
	if ERROR_HAPPENED {
		fmt.Println("Final Result : FAIL")
	} else {
		fmt.Println("Final Result : PASS")
	}
}

func ResetServerState() {
	//Remove any state recovery files
	for i := 0; i < NUM_SERVERS; i++ {
		os.Remove(fmt.Sprintf("%s_S%d.state", STATE_FILENAME, i))
	}
}

//At least majority server should be up for getting any response to client
func TestElectionAndReplication() {

	Log("Testing Leader Election")
	//Start majority
	for i := 0; i < (NUM_SERVERS/2)+1; i++ {
		startSingleServer(i)
	}

	time.Sleep(waitSec)                  //Wait for all to start up
	conn, leaderId := connectToLeader(0) //Connect to leader by trying first server

	if conn != nil {
		Log(fmt.Sprintf("Connected to Leader (Id:%v)", leaderId))

		PrintTestLabel("SET Key")
		TCPWrite(conn, "set something 0 10\r\nasgbdtsdhh")
		response := TCPRead(conn)
		checkIfExpected(response[:3], "OK ")

		time.Sleep(1 * time.Second)

		PrintTestLabel("GET Key")
		TCPWrite(conn, "get something\r\n")
		response = TCPRead(conn)
		checkIfExpected(response[:5], "VALUE")

		conn.Close() //Close previous client

		Log("Kill current leader")
		killServer(leaderId)

		time.Sleep(500 * time.Millisecond)
		Log("Restart ")
		startSingleServer(leaderId)

		time.Sleep(500 * time.Millisecond)
		Log("Start another server")
		startSingleServer((NUM_SERVERS / 2) + 1)

		Log("Wait for leader election")
		time.Sleep(waitSec)

		conn, leaderId = connectToLeader((NUM_SERVERS / 2) + 1) //Connect to leader by trying first server
		if conn != nil {
			Log(fmt.Sprintf("Connected to new Leader (Id:%v)", leaderId))
			Log("Leader election SUCCESSFUL")

			Log("Testing Log Replication")
			PrintTestLabel("GET Key")
			TCPWrite(conn, "get something\r\n")
			response = TCPRead(conn)
			if checkIfExpected(response[:5], "VALUE") {
				Log("Log replication SUCCESSFUL")
			} else {
				Log("Log replication FAILED")
				Log("Leader Election FAILED")
				LogVerbose("Leader was not elected after ", waitSec)
			}

			conn.Close() //Close previous client
		} else {
			PrintError("Leader Election Failed")
			PrintError("Couldn't connect to Leader")
		}

	} else {
		PrintError("Couldn't connect to Leader")
	}

	killAllServers()
}

//
func TestStress() {
	Log("Stress test")
	startAllServers()

	//Wait for some time for election
	time.Sleep(waitSec)

	//Connect to leader
	conn, leaderId := connectToLeader(0)
	if conn != nil {
		Log("Connected to leader (Id:", leaderId, ")")
	} else {
		PrintError("Couldn't connect to leader!")
		killAllServers()
		return
	}

	LogVerbose("Add items")
	for i := 0; i < 20; i++ {
		//Add 20 items
		PrintTestLabel("SET Key ", i)
		TCPWrite(conn, fmt.Sprintf("set key%d 0 10\r\n%2dgbdtsdhh", i, i))
		response := TCPRead(conn)
		checkIfExpected(response[:3], "OK ")
	}

	conn.Close() //CLose write connection

	//Wait for some time for log replication
	time.Sleep(time.Second)

	//Kill different server in 4 rounds
	for round := 0; round < 4; round++ {

		//Kill leader
		LogVerbose("Kill leader (Server:", leaderId, ")")
		killServer(leaderId)

		//Wait for next leader
		time.Sleep(waitSec)

		//Restart Server
		LogVerbose("Restart Server ", leaderId)
		startSingleServer(leaderId)
		time.Sleep(2 * time.Second)

		//Connect to new leader
		conn, leaderId = connectToLeader(0)
		if conn != nil {
			Log("Connected to leader (Id:", leaderId, ")")
		} else {
			PrintError("Couldn't connect to leader!")
			killAllServers()
			return
		}

		//Get items
		LogVerbose("Read items")
		for i := round * 5; i < round*5+5; i++ {
			//Add 20 items
			PrintTestLabel("GET Key ", i)
			TCPWrite(conn, fmt.Sprintf("get key%d\r\n", i))
			response := TCPRead(conn)
			checkIfExpected(response[:5], "VALUE")
		}

		conn.Close()
	}

	killAllServers()
}

//Connect to current leader
func connectToLeader(startId int) (net.Conn, int) {

	//Try to connect until we get ERR_NOT_FOUND
	for {

		conn := startClient(startId) //Connect to first server

		if conn == nil {
			PrintError("Couldn't connect to server")
			return nil, -1
		}

		TCPWrite(conn, "get nonExistingKey\r\n")
		response := TCPRead(conn)
		//Response should be either redirect or ERR_NOT
		redirect, sid := parseServerResponse(response)

		if redirect {
			startId = sid
			time.Sleep(250 * time.Millisecond)
			LogVerbose("Redirected to server ", sid)
			continue
		} else {
			return conn, startId
		}
	}
}

//Tries to parse the message if redirect and return port number
func parseServerResponse(response string) (redirect bool, serverId int) {

	if response[:8] == "REDIRECT" {
		serverId, err := strconv.ParseInt(response[9:], 10, 64)

		if err != nil {
			PrintError("Couldn't parse server REDIRECT")
			return false, -1
		}

		return true, int(serverId)
	}

	return false, -2
}

//Connect to server with serverId as id
func startClient(id int) net.Conn {
	port := fmt.Sprintf("%d", START_PORT+id)

	conn, err := net.Dial("tcp", "localhost:"+port)

	if err != nil {
		fmt.Println("Cannot connect to server.. Port:" + port)
		return nil
	}
	return conn
}

func startSingleServer(id int) {
	liveServers[id] = exec.Command(SERVER_NAME, strconv.Itoa(id))
	// liveServers[id].Stdout = os.Stdout
	// liveServers[id].Stderr = os.Stderr
	err := liveServers[id].Start()
	LogVerbose("Started server", id)
	if err != nil {
		panic(err)
	}
}

func killServer(id int) {

	if liveServers[id] == nil {
		return
	}
	liveServers[id].Process.Kill()
	liveServers[id].Wait()
	LogVerbose("Killed server ", id)
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

func TCPWrite(conn net.Conn, buf string) {
	_, err := conn.Write([]byte(buf + "\r\n"))
	PrintTCP(buf)
	if err != nil {
		fmt.Print("Cannot write to TCP socket..")
	}
}

func TCPRead(conn net.Conn) string {
	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	buf := scanner.Text()

	if scanner.Err() != nil {
		fmt.Print("Cannot read from TCP socket..")
	}
	PrintTCP(buf)
	return buf
}
func checkIfExpected(actual string, expected string) bool {
	result := false
	if actual != expected {
		PrintTest(fmt.Sprintf("FAIL : Expected '%s' but got '%s'\n", expected, actual))
		ERROR_HAPPENED = true
	} else {
		PrintTest("PASS")
		result = true
	}
	PrintTest("----------------------------")
	return result

}
func checkIfExpected2(actual string, expected1 string, expected2 string) bool {
	result := false
	if actual != expected1 || actual != expected2 {
		PrintTest(fmt.Sprintf("FAIL : Expected '%s' or '%s' but got '%s'\n", expected1, expected2, actual))
		ERROR_HAPPENED = true
	} else {
		PrintTest("PASS")
		result = true
	}
	PrintTest("----------------------------")
	return result

}

func PrintTCP(str string) {
	if LOG_TCP_MSGS {
		fmt.Println("TCP: ", strings.Trim(str, "\r\n "))
	}
}
func PrintTestLabel(str ...interface{}) {
	if LOG_TEST_LABEL {
		fmt.Print("TEST: ")
		fmt.Println(str...)
	}
}
func Log(str ...interface{}) {
	if LOG_MSG {
		fmt.Println(str...)
		fmt.Println("----------------------------")
	}
}
func PrintError(str string) {
	fmt.Println("FAIL: ", str)
	ERROR_HAPPENED = true
}
func PrintTest(str ...interface{}) {
	if LOG_TEST {
		fmt.Println(str...)
	}
}
func LogVerbose(vargs ...interface{}) {
	if LOG_VERBOSE {
		fmt.Println(vargs...)
	}
}
