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

// var raft Raft       //Raft object

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
var nServers int
var raftMap map[int]*Raft  //To get raft reference of all 5 server, required to fake RPCs
var raftMapLock sync.Mutex //For accessing global raft map

// Raft implements the SharedLog interface.
type Raft struct {
	Log                      []LogItem
	LastLsn                  Lsn
	ServerID, LeaderID       int
	ClientPort, LogPort      int
	State                    string
	Term                     uint64
	CommitIndex, LastApplied int64
	VotedFor                 int //Voted for whom in this term
	Lock                     sync.Mutex
	kvChan                   chan LogEntry //Commit channel to kvStore
	eventCh                  chan interface{}
}

// Creates a raft object. This implements the SharedLog interface.
// commitCh is the channel that the kvstore waits on for committed messages.
// When the process starts, the local disk log is read and all committed
// entries are recovered and replayed
func NewRaft(config *ClusterConfig, thisServerId int, commitCh chan LogEntry) (*Raft, error) {

	raft := Raft{} // empty raft object
	for _, server := range config.Servers {

		if server.Id == thisServerId { //Config for this server
			raft.ServerID = thisServerId
			raft.ClientPort = server.ClientPort
			raft.LogPort = server.LogPort
			raft.LastLsn = 0
			raft.CommitIndex = -1
			raft.State = Follower
			raft.Term = 0
			raft.VotedFor = -1
			break
		}
	}

	raft.kvChan = commitCh //Store commit channel to KV-Store
	raft.eventCh = make(chan interface{}, 5)

	go raft.loop()

	if raftMap == nil {
		raftMap = make(map[int]*Raft)
	}
	raftMap[thisServerId] = &raft

	log.Print("Raft init, Server id:" + strconv.Itoa(raft.ServerID))
	return &raft, nil
}

func ReadConfig() error {
	//Get config.json file from $GOPATH/src/assignment3/
	filePath := os.Getenv("GOPATH") + "/src/assignment3/config.json"
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.New("Couldn't open config file from :" + filePath)
	}

	err = json.Unmarshal(file, &ClusterInfo)
	if err != nil {
		return errors.New("Wrong format of config file")
	}

	if nServers == 0 {
		nServers = len(ClusterInfo.Servers)
	}

	return nil
}

// ErrRedirect as an Error object
func (e ErrRedirect) Error() string {
	return "Redirect to server " + strconv.Itoa(int(e))
}
