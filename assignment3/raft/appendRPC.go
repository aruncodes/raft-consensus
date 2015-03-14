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

	if len(args.Log) > 0 {
		log.Print("AppendRPC received")
	} else {
		log.Print("Heartbeat received")
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
