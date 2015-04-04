package raft

import (
	"errors"
	// "log"
	// "net"
	// "net/rpc"
	// "strconv"
)

type AppendRPCArgs struct {
	Term         uint64
	LeaderId     int
	PrevLogIndex Lsn
	PrevLogTerm  uint64
	Log          []LogItem
	LeaderCommit uint64
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

//Append from leader
func (raft *Raft) appendEntries(args AppendRPCArgs) bool {

	if raft.Term <= args.Term {
		//Must be the new leader

		//Update term
		if raft.Term < args.Term {
			raft.Term = args.Term
			raft.VotedFor = -1
		}

		//Update Leader ID
		raft.LeaderID = args.LeaderId

		if raft.Log[args.PrevLogIndex].Term != args.PrevLogTerm {
			//Log doesnt contain an entry at PrevLogIndex whose term matches
			// prevLogTerm

			//Remove that entry and everything after it
			raft.Log = raft.Log[:args.PrevLogIndex]

			return false
		}

		//Else append the entries
		raft.Lock.Lock()
		raft.Log = append(raft.Log, args.Log...)
		raft.Lock.Unlock()

		if args.LeaderCommit > raft.CommitIndex {
			//update commit index to min of leader commit and last entry added

			lastIndex := raft.Log[len(raft.Log)-1].Lsn()
			min := uint64(lastIndex)
			if min > args.LeaderCommit {
				min = args.LeaderCommit
			}

			raft.CommitIndex = min
		}

		//Apply to state machine if commitIndex > lastApplied
		if raft.CommitIndex > raft.LastApplied {
			for i, _ := range raft.Log[raft.LastApplied+1 : raft.CommitIndex+1] {
				//Apply from last applied to current commit index
				raft.Log[i].COMMITTED = true
				raft.kvChan <- raft.Log[i]
			}

			//Update lastApplied
			raft.LastApplied = raft.CommitIndex
		}

		return true
	} else {
		//I have another leader
		return false
	}

}

//RPC call to followers
func (raft *Raft) appendRPC(server ServerConfig, args AppendRPCArgs, reply *AppendRPCResults) error {
	//Should be actual RPC
	//Actual code commented below with same function name

	//Error simulator code starts
	//State of this server
	switch serverState[raft.ServerID] {

	case KILLED, DROP_MSG:
		return nil
	}

	//State of server to which append is sent
	switch serverState[server.Id] {

	case KILLED, DROP_MSG:
		return errors.New("Server down")
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
