package raft

import (
	"errors"
	"log"
)

type RequestVoteArgs struct {
	Term         uint64
	CandidateID  uint64
	LastLogIndex Lsn
	LastLogTerm  uint64
}

type RequestVoteResult struct {
	Term        uint64
	VoteGranted bool
}

func (raft *Raft) shouldIVote(args RequestVoteArgs) bool {

	candidateTerm := args.Term
	shouldVote := false

	if raft.Term < candidateTerm {
		//Candidate in higher term
		if args.LastLogIndex > raft.LastLsn() {
			//Candidates log is more complete
			shouldVote = true
		} else if args.LastLogIndex == raft.LastLsn() &&
			args.LastLogTerm >= raft.Log[raft.LastLsn()].Term {
			//Candidates log is atleast up to date as me
			shouldVote = true
		} else {
			//Log not upto date
			shouldVote = false
		}
	} else if raft.Term == candidateTerm {
		//We are in same term
		if (raft.VotedFor == -1) || (raft.VotedFor == int(args.CandidateID)) {
			//Not voted in this term or already voted for this server
			shouldVote = true
		} else {
			//Already voted
			shouldVote = false
		}
	} else {
		//Lesser term
		shouldVote = false
	}
	return shouldVote
}

//Per server vote request
func (raft *Raft) sendVoteRequest(server ServerConfig, ackChannel chan bool) {
	//Create args and reply
	lastLogTerm := raft.Log[raft.LastLsn()].Term
	args := RequestVoteArgs{raft.Term, uint64(raft.ServerID), raft.LastLsn(), lastLogTerm}
	reply := RequestVoteResult{}

	//Request vote by RPC
	// err := raft.requestVote(server, args, &reply) //fake
	err := raft.voteRequestRPC(server, args, &reply) //

	if err != nil {
		log.Println(err.Error())
		ackChannel <- false
		return
	}

	//Send ack
	ackChannel <- reply.VoteGranted
}

//Fake RPC, not used anymore
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
