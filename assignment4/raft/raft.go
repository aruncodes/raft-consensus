package raft

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	// "os"
	"strconv"
	"sync"
)

type Lsn uint64      //Log sequence number, unique for all time.
type ErrRedirect int // Implements Error interface.

var raft Raft //Raft object

const FILENAME = "saved"

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
	Term      uint64
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

var ClusterInfo ClusterConfig //Struct with all raft configs
var nServers int              //Number of servers in cluster
var raftMap map[int]*Raft     //To get raft reference of all 5 server, required to fake RPCs
var raftMapLock sync.Mutex    //For accessing global raft map

// Raft implements the SharedLog interface.
type Raft struct {
	ServerID, LeaderID  int
	ClientPort, LogPort int
	State               string
	Lock                sync.Mutex
	kvChan              chan LogEntry //Commit channel to kvStore
	eventCh             chan interface{}

	//Raft specific
	Log                      []LogItem
	Term                     uint64
	CommitIndex, LastApplied uint64
	NextIndex                []Lsn
	MatchIndex               []Lsn
	VotedFor                 int //Voted for whom in this term
}

// Creates a raft object. This implements the SharedLog interface.
// commitCh is the channel that the kvstore waits on for committed messages.
// When the process starts, the local disk log is read and all committed
// entries are recovered and replayed
func NewRaft(config *ClusterConfig, thisServerId int, commitCh chan LogEntry) (*Raft, error) {

	raft = Raft{} // empty raft object
	for _, server := range config.Servers {

		if server.Id == thisServerId { //Config for this server
			raft.ServerID = thisServerId
			raft.ClientPort = server.ClientPort
			raft.LogPort = server.LogPort
			break
		}
	}

	raft.Log = append(raft.Log, LogItem{}) //Dummy item to make log start from index 1

	//Log indices
	raft.CommitIndex = 0
	raft.LastApplied = 0

	//Raft states
	raft.State = Follower
	raft.Term = 0
	raft.VotedFor = -1

	raft.kvChan = commitCh                          //Store commit channel to KV-Store
	raft.eventCh = make(chan interface{}, nServers) //Event channel for state loop

	//Restore state if state file exists
	if raft.FileExist(FILENAME) {
		//Server crashed last time
		raft.ReadStateFromFile(FILENAME)

		//Add changes to state machine
		for i := 1; i < len(raft.Log); i++ {
			if raft.Log[i].Committed() {
				//Add if already commited
				raft.kvChan <- raft.Log[i]

				raft.LastApplied = uint64(i)
				raft.CommitIndex = uint64(i)
			}
		}
	}

	//Other server states
	raft.NextIndex = make([]Lsn, nServers, nServers)
	raft.MatchIndex = make([]Lsn, nServers, nServers)

	go raft.loop() //Raft state loop

	go raft.RPCListener() //Start RPC Listener

	// if raftMap == nil {
	// 	raftMap = make(map[int]*Raft)
	// }
	// raftMap[thisServerId] = &raft

	log.Print("Raft init, Server id:" + strconv.Itoa(raft.ServerID))
	return &raft, nil
}

func ReadConfig() error {
	//Get config.json file current working dir
	// filePath := os.Getenv("GOPATH") + "/src/assignment3/config.json"
	filePath := "config.json"
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

func (raft *Raft) LastLsn() Lsn {
	return raft.Log[len(raft.Log)-1].Lsn()
}

// ErrRedirect as an Error object
func (e ErrRedirect) Error() string {
	return "Redirect to server " + strconv.Itoa(int(e))
}
