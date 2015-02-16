package raft

import (
	"log"
	"net/rpc"
	"strconv"
	"time"
)

func heartBeater() {

	prevLogIndex := -1
	for {

		votes := 0 //Votes from majority
		startLogIndex := getLatestLogIndex()

		//Send appendRPC to all server
		//starting from startLogIndex
		for _, server := range ClusterInfo.Servers {

			if raft.ServerID == server.Id {
				// The current running server
				continue
			}

			client, err := rpc.Dial("tcp", ":"+strconv.Itoa(server.LogPort))
			if err != nil {
				// log.Print("AppendRPC Dial error on port:" + strconv.Itoa(server.LogPort))
				log.Print("Server ", server.Id, " down")
				continue
			}

			//Create args to call RPC
			var logSlice []LogItem

			if startLogIndex > prevLogIndex {
				//Add if more entires are added
				logSlice = raft.Log[startLogIndex : startLogIndex+1] //Only one entry for now
			}

			args := &AppendRPCArgs{logSlice} //Send slice with new entires
			var reply AppendRPCResults

			err = client.Call("AppendEntries.AppendEntriesRPC", args, &reply)

			if err != nil {
				log.Print("RPC fail :" + err.Error())
				continue
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
				kvChan <- raft.Log[startLogIndex]

				//Update status as commited
				lock.Lock()
				raft.Log[startLogIndex].COMMITTED = true
				lock.Unlock()

				prevLogIndex = startLogIndex
			}
		}

		//Sleep for some time before sending next heart beat
		// time.Sleep(200 * time.Millisecond)
		time.Sleep(2 * time.Second) //for debuging
	}
}

func getLatestLogIndex() int {

	for i, item := range raft.Log {
		if !item.Committed() {
			return i //First uncommitted entry
		}
	}

	return -1
}
