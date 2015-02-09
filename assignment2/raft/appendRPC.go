package raft

import (
	"log"
	"net"
	"net/rpc"
	"strconv"
)

type AppendRPCArgs struct {
	// term 		uint64
	// leaderId 	uint64
	// PrevLogIndex	Lsn
	// prevLogTerm	uint64
	Log []LogItem
	// leaderCommit	uint64
}

type AppendRPCResults struct {
	// term uint64
	Success bool
}

type AppendEntries struct{}

func (ap *AppendEntries) AppendEntriesRPC(args AppendRPCArgs, result *AppendRPCResults) error {
	log.Print("AppendRPC call received")

	// raft.Append(args.Log) //Append to our local log
	for _, log := range args.Log {
		command := log.DATA

		raft.Append(command)
	}

	result.Success = true
	return nil
}

func appendRPCListener(port int) {
	appendRPC := new(AppendEntries)
	rpc.Register(appendRPC)

	listener, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(port), 10))
	if err != nil {
		log.Print("AppendRCP error : " + err.Error())
		return
	}

	for {
		if conn, err := listener.Accept(); err != nil {
			log.Print("Accept error : " + err.Error())
		} else {
			log.Print("New RPC connection established")
			go rpc.ServeConn(conn)
		}
	}
}

func RequestAppendEntriesRPC(config ClusterConfig, raft *Raft) bool {

	var votes int

	for _, server := range config.Servers {

		if raft.ServerID == server.Id {
			// The current running server
			continue
		}

		client, err := rpc.Dial("tcp", ":"+strconv.Itoa(server.LogPort))
		if err != nil {
			log.Print("AppendRPC Dial error on port: " + strconv.Itoa(server.LogPort))
			continue
		}

		args := &AppendRPCArgs{raft.Log} //Send all log items for now
		var reply AppendRPCResults

		err = client.Call("AppendEntries.AppendEntriesRPC", args, &reply)

		if err != nil {
			log.Print("RPC call fail :" + err.Error())
			continue
		}

		if reply.Success == true {
			votes++
		}
	}

	if votes >= len(config.Servers)/2 { //Majority
		return true
	}
	return false
}
