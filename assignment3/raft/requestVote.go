package raft

import (
	"errors"
	// "log"
)

type RequestVoteArgs struct {
	Term        uint64
	CandidateID uint64
	// LastLogIndex Lsn
	// LastLogTerm  uint64
}

type RequestVoteResult struct {
	Term        uint64
	VoteGranted bool
}

func (raft *Raft) requestVote(server ServerConfig, args RequestVoteArgs, reply *RequestVoteResult) error {
	//Should actually do RPC
	//Here, it communicates using channels of remote raft server

	//Error simulator code starts
	//State of server to which vote is requesting
	switch serverState[server.Id] {
	case KILLED, DROP_MSG:
		return nil
	}

	//State of this server
	switch serverState[raft.ServerID] {
	case KILLED, DROP_MSG:
		return nil
	}
	//Error simulator code ends

	raftMapLock.Lock()
	remoteRaft, exists := raftMap[server.Id]
	raftMapLock.Unlock()

	if !exists {
		return errors.New("Server unavailable")
	}
	responseCh := make(chan RequestVoteResult, 5)
	remoteRaft.eventCh <- VoteRequest{args, responseCh}
	*reply = <-responseCh

	return nil
}
