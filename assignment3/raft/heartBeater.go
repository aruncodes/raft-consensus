package raft

import (
	"log"
)

func (raft *Raft) heartBeat() {

	votes := 0 //Votes from majority
	startLogIndex := raft.getLatestLogIndex()

	//Send appendRPC to all server
	//starting from startLogIndex
	for _, server := range ClusterInfo.Servers {

		if raft.ServerID == server.Id {
			// The current running server
			continue
		}

		//Create args to call RPC
		var logSlice []LogItem

		if startLogIndex > raft.CommitIndex {
			//Add if more entires are added
			logSlice = raft.Log[startLogIndex : startLogIndex+1] //Only one entry for now
		}

		args := AppendRPCArgs{raft.Term, raft.LeaderID, logSlice} //Send slice with new entires
		var reply AppendRPCResults                                //reply from RPC
		// log.Print("sending rpc to ", server.Id)
		err := raft.appendRPC(server, args, &reply) //Make RPC
		// log.Print("recevied rpc")

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

		//Count votes
		if reply.Success == true {
			votes++
		}
	}

	if votes >= len(ClusterInfo.Servers)/2 { //Majority
		// log.Print("Got majority, log entry:", startLogIndex)
		if startLogIndex != -1 && !raft.Log[startLogIndex].Committed() {
			//If that log entry actually exists and not committed
			//Send to commit channel, KV Store will be waiting
			raft.kvChan <- raft.Log[startLogIndex]

			//Update status as commited
			raft.Lock.Lock()
			raft.Log[startLogIndex].COMMITTED = true
			raft.Lock.Unlock()

			raft.CommitIndex = startLogIndex
		}
	}
}

func (raft *Raft) getLatestLogIndex() int64 {

	for i, item := range raft.Log {
		if !item.Committed() {
			return int64(i) //First uncommitted entry
		}
	}

	return -1
}
