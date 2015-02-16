package raft

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
)

type Lsn uint64      //Log sequence number, unique for all time.
type ErrRedirect int // Implements Error interface.

var raft Raft       //Raft object
var lock sync.Mutex //Lock for appending to log

type LogEntry interface {
	Lsn() Lsn
	Data() Command
	Committed() bool
}

//A command from client
type Command struct {
	Cmd        string
	Key        string
	ExpiryTime int64
	Length     int64
	Version    int64
	Value      string
}

type LogItem struct {
	LSN       Lsn
	DATA      Command
	COMMITTED bool
}

func (l LogItem) Lsn() Lsn {
	return l.LSN
}
func (l LogItem) Data() Command {
	return l.DATA
}
func (l LogItem) Committed() bool {
	return l.COMMITTED
}

type SharedLog interface {
	// Each data item is wrapped in a LogEntry with a unique
	// lsn. The only error that will be returned is ErrRedirect,
	// to indicate the server id of the leader. Append initiates
	// a local disk write and a broadcast to the other replicas,
	// and returns without waiting for the result.
	Append(data Command) (LogEntry, error)
}

// --------------------------------------
// Raft setup
type ServerConfig struct {
	Id         int    //Id of server. Must be unique
	Hostname   string //name or ip of host
	ClientPort int    //port at which server listens to client messages.
	LogPort    int    // tcp port for inter-replica protocol messages.
}

type ClusterConfig struct {
	Path    string         // Directory for persistent log
	Servers []ServerConfig // All servers in this cluster
}

var ClusterInfo ClusterConfig

// Raft implements the SharedLog interface.
type Raft struct {
	// .... fill
	Log        []LogItem
	LastLsn    Lsn
	ServerID   int
	ClientPort int
	LogPort    int
	LeaderID   int
}

//Commit channel to kvStore
var kvChan chan LogEntry

// Creates a raft object. This implements the SharedLog interface.
// commitCh is the channel that the kvstore waits on for committed messages.
// When the process starts, the local disk log is read and all committed
// entries are recovered and replayed
func NewRaft(config *ClusterConfig, thisServerId int, commitCh chan LogEntry) (*Raft, error) {

	//Get config.json file from $GOPATH/src/assignment2/
	filePath := os.Getenv("GOPATH") + "/src/assignment2/config.json"
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, errors.New("Couldn't open config file from :" + filePath)
	}

	err = json.Unmarshal(file, &ClusterInfo)
	if err != nil {
		return nil, errors.New("Wrong format of config file")
	}

	for _, server := range config.Servers {

		if server.Id == thisServerId { //Config for this server
			raft.ServerID = thisServerId
			raft.LeaderID = 0 //First server is leader by default
			raft.ClientPort = server.ClientPort
			raft.LogPort = server.LogPort
			raft.LastLsn = 0
			break
		}
	}

	kvChan = commitCh //Store channel

	if thisServerId == raft.LeaderID {
		//We are the leader, so start hearbeater
		go heartBeater()
	} else {
		//Just a follower, listen for RPC from leader
		go appendRPCListener(raft.LogPort)
	}

	log.Print("Raft init, Server id:" + strconv.Itoa(raft.ServerID))
	return &raft, nil
}

func (r *Raft) Append(data Command) (LogEntry, error) {

	//Check if leader. If not, send redirect
	if r.ServerID != r.LeaderID {
		return LogItem{}, ErrRedirect(r.LeaderID)
	}

	logItem := LogItem{r.LastLsn + 1, data, false}

	lock.Lock()
	r.Log = append(r.Log, logItem)
	r.LastLsn++
	lock.Unlock()

	return &logItem, nil
}

// ErrRedirect as an Error object
func (e ErrRedirect) Error() string {
	return "Redirect to server " + strconv.Itoa(int(e))
}
