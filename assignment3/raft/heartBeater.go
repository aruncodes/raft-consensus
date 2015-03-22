package raft

import (
	"log"
)

func (raft *Raft) heartBeat() {

	//Send appendRPC to all server
	//starting from startLogIndex
	for _, server := range ClusterInfo.Servers {

		if raft.ServerID == server.Id {
			// The current running server
			continue
		}

		//Create args to call RPC
		var logSlice []LogItem

		if raft.LastLsn >= raft.NextIndex[server.Id] {
			//Followers log needs to be filled up
			nextIndex := raft.NextIndex[server.Id]
			logSlice = raft.Log[nextIndex:]
		}

		prevLogIndex := raft.NextIndex[server.Id] - 1
		prevLogTerm := raft.Log[prevLogIndex].Term

		args := AppendRPCArgs{raft.Term, raft.LeaderID,
			prevLogIndex, prevLogTerm, logSlice, uint64(raft.CommitIndex)} //Send slice with new entires

		var reply AppendRPCResults                  //reply from RPC
		err := raft.appendRPC(server, args, &reply) //Make RPC

		if err != nil {
			log.Print(err.Error())
			continue
		}

		if reply.Term > raft.Term {
			//There is new leader with a higher term
			//Revert to follower
			raft.State = Follower
			raft.Term = reply.Term
			raft.VotedFor = -1
			break
		}

		if reply.Success {
			//Update nextIndex and matchIndex
			raft.NextIndex[server.Id] = raft.LastLsn + 1
			raft.MatchIndex[server.Id] = raft.LastLsn + 1
		} else {
			//Log inconsistency
			//Decrement nextIndex and retry
			raft.NextIndex[server.Id]--
		}

	}

	//If majority of servers are with matching log , commit till that point
	for i := raft.CommitIndex + 1; i <= uint64(raft.LastLsn); i++ {
		votes := 0
		for j := 0; j < nServers; j++ {
			if uint64(raft.MatchIndex[j]) >= i && raft.Log[i].Term == raft.Term {
				votes++
			}
		}

		if votes > nServers/2 {
			//Got majority for that entry, so commit
			raft.kvChan <- raft.Log[i]

			//Update status as commited
			raft.Lock.Lock()
			raft.Log[i].COMMITTED = true
			raft.Lock.Unlock()

			//Update commit index
			raft.CommitIndex = i
		}
	}

}
