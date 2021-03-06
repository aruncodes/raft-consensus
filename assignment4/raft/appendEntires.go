package raft

import ()

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

		if len(args.Log) > 0 {
			//It is an appendEnties, not a heartBeat

			if int(args.PrevLogIndex) >= len(raft.Log) {
				//I dont have an entry at that index
				return false
			}

			if raft.Log[args.PrevLogIndex].Term != args.PrevLogTerm {
				//Log doesnt contain an entry at PrevLogIndex whose term matches
				// prevLogTerm

				//Remove that entry and everything after it
				raft.Log = raft.Log[:args.PrevLogIndex]

				return false
			}

			//If everything is alright, append the entries
			raft.Lock.Lock()
			raft.Log = append(raft.Log, args.Log...)
			raft.Lock.Unlock()
		}

		if args.LeaderCommit > raft.CommitIndex {
			//update commit index to min of leader commit and last entry added

			lastIndex := raft.LastLsn() //raft.Log[len(raft.Log)-1].Lsn()
			min := uint64(lastIndex)
			if min > args.LeaderCommit {
				min = args.LeaderCommit
			}

			raft.CommitIndex = min
		}

		//Apply to state machine if commitIndex > lastApplied
		if raft.CommitIndex > raft.LastApplied {
			for i := raft.LastApplied + 1; i <= raft.CommitIndex; i++ {
				//Apply from last applied to current commit index
				raft.Lock.Lock()
				raft.Log[i].COMMITTED = true
				raft.Lock.Unlock()

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
