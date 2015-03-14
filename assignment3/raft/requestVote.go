package raft

import (
	"errors"
)

type RequestVoteArgs struct {
	Term        uint64
	CandidateID uint64
}

type RequestVoteResult struct {
	Term        uint64
	VoteGranted bool
}

func (raft *Raft) requestVote(server ServerConfig, args RequestVoteArgs, reply *RequestVoteResult) error {
	//Should actually do RPC
	//Here, it communicates using channels of remote raft server

	remoteRaft, exists := raftMap[server.Id]

	if !exists {
		return errors.New("Server down")
	}
	responseCh := make(chan RequestVoteResult)
	remoteRaft.eventCh <- VoteRequest{args, responseCh}
	*reply = <-responseCh

	return nil
}
