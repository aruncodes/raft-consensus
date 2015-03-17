package raft

import (
	"errors"
	// "log"
	// "net"
	// "net/rpc"
	// "strconv"
)

type AppendRPCArgs struct {
	Term     uint64
	LeaderId int
	// PrevLogIndex	Lsn
	// prevLogTerm	uint64
	Log []LogItem
	// leaderCommit	uint64
}

type AppendRPCResults struct {
	Term    uint64
	Success bool
}

//Append call from client
func (r *Raft) Append(data Command) (LogEntry, error) {

	//Check if leader. If not, send redirect
	if r.ServerID != r.LeaderID {
		return LogItem{}, ErrRedirect(r.LeaderID)
	}

	responseCh := make(chan LogEntry)           //Response channel
	r.eventCh <- ClientAppend{data, responseCh} //Send a clientAppend event
	logItem := <-responseCh                     //Get back response logentry

	if logItem.Lsn() == 0 {
		//Append was to a follower
		return LogItem{}, ErrRedirect(r.LeaderID)
	}

	return logItem, nil
}

func (raft *Raft) appendRPC(server ServerConfig, args AppendRPCArgs, reply *AppendRPCResults) error {
	//Should be actual RPC
	//Actual code commented below with same function name

	//Error simulator code starts
	//State of this server
	switch serverState[raft.ServerID] {

	case KILLED:
		return nil
		// <-raft.eventCh //A timeout before going to killed state (ignoring this will create byzantine situation)

	case DROP_MSG:
		return nil

	case NORMAL:
		break
	}

	//State of server to which append is sent
	switch serverState[server.Id] {

	case KILLED:
		return errors.New("Server down")

	case DROP_MSG:
		return nil

	case NORMAL:
		break
	}
	//Error simulator code ends
	raftMapLock.Lock()
	remoteRaft, exists := raftMap[server.Id]
	raftMapLock.Unlock()

	if !exists {
		return errors.New("Server unavailable")
	}

	responseCh := make(chan AppendRPCResults, 5)
	remoteRaft.eventCh <- AppendRPC{args, responseCh}
	*reply = <-responseCh

	return nil
}

/*

//Actual RPC code
type AppendEntries struct{}

func (raft *Raft) appendRPC(server ServerConfig, args AppendRPCArgs, reply *AppendRPCResults) error {

	client, err := rpc.Dial("tcp", ":"+strconv.Itoa(server.LogPort))
	if err != nil {
		// log.Print("AppendRPC Dial error on port:" + strconv.Itoa(server.LogPort))
		// log.Print("Server ", server.Id, " down")
		return errors.New("Server down")
	}

	err = client.Call("AppendEntries.AppendEntriesRPC", args, reply)

	if err != nil {
		log.Print("RPC fail :" + err.Error())
		return errors.New("RPC fail")
	}

	return nil
}

func (ap *AppendEntries) AppendEntriesRPC(args AppendRPCArgs, result *AppendRPCResults) error {

	if len(args.Log) > 0 {
		log.Print("AppendRPC received")
	} else {
		// log.Print("Heartbeat received")
	}

	//Append to our local log
	// for _, log := range args.Log {
	// 	command := log.DATA

	// 	// raft.Append(command)
	// }

	result.Success = true
	return nil
}

func appendRPCListener(raft *Raft) {
	appendRPC := new(AppendEntries)
	rpc.Register(appendRPC)

	listener, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(raft.LogPort), 10))
	if err != nil {
		log.Print("AppendRCP error : " + err.Error())
		return
	}

	for {
		if conn, err := listener.Accept(); err != nil {
			log.Print("Accept error : " + err.Error())
		} else {
			go rpc.ServeConn(conn)
		}
	}
}
*/
