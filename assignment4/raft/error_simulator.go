package raft

import (
	"errors"
	"fmt"
)

var appendRPCState int
var voteRPCState int

var serverState [5]int

type ErrorSimulation struct {
	State string
}

const ( //Server States
	NORMAL   = 0
	KILLED   = 1
	DROP_MSG = 2
)

func KillServer(serverId int) {

	raftMap[serverId].eventCh <- ErrorSimulation{Killed}
	serverState[serverId] = KILLED
}

func ResurrectServer(serverId int) {
	serverState[serverId] = NORMAL
	raftMap[serverId].eventCh <- ErrorSimulation{Follower}
}

func ResurrectServerAsLeader(serverId int) {
	serverState[serverId] = NORMAL
	raftMap[serverId].eventCh <- ErrorSimulation{Leader}
}

func KillLeader() int {

	//Find leader
	for _, serverRaft := range raftMap {

		if serverRaft.State == Leader {
			KillServer(serverRaft.ServerID)
			return serverRaft.ServerID
		}
	}
	return -1
}

func GetLeaderId() int {

	//Find leader
	for _, serverRaft := range raftMap {
		if serverRaft.State == Leader {
			return serverRaft.ServerID
		}
	}
	return -1
}

func GetLogAsString(serverId int) string {
	var retStr string

	if serverId == -1 {
		return "none"
	}

	logArray := raftMap[serverId].Log

	for index, logItem := range logArray {
		retStr += fmt.Sprintf("%d %d ", index, logItem.Term)
	}

	return retStr
}

func InsertFakeLogEntry(serverId int) {

	if serverId == -1 {
		return
	}
	raftObj := raftMap[serverId]

	respCh := make(chan LogEntry)
	raftObj.eventCh <- ClientAppend{Command{}, respCh}
	<-respCh
}

//Old methods, not used as of now

var raftMapBackup map[int]*Raft

func MakeServerUnavailable(serverID int) error {

	raftMapLock.Lock()

	if raftMapBackup == nil {
		//Create if not yet initialized
		raftMapBackup = make(map[int]*Raft)
	}

	remoteRaft, exists := raftMap[serverID]

	if !exists {
		raftMapLock.Unlock()
		return errors.New("Raft not available")
	}

	//Make a backup
	raftMapBackup[serverID] = remoteRaft

	//Remove from raft Map
	delete(raftMap, serverID)

	raftMapLock.Unlock()

	return nil
}

func MakeServerAvailable(serverID int) error {

	raftMapLock.Lock()

	if raftMapBackup == nil {
		//This should not happen
		raftMapLock.Unlock()
		return errors.New("Not made unavailable first before making available")
	}

	remoteRaft, exists := raftMapBackup[serverID]

	if !exists {
		//This should not happen
		return errors.New("Raft not available")
	}

	//Restore to raftMap
	raftMap[serverID] = remoteRaft

	raftMapLock.Unlock()

	return nil
}
