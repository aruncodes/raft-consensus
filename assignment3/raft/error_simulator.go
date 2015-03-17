package raft

import (
	"errors"
)

var appendRPCState int
var voteRPCState int

var raftMapBackup map[int]*Raft
var serverState [5]int

const ( //Server States
	NORMAL   = 0
	KILLED   = 1
	DROP_MSG = 2
)

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

func KillServer(serverId int) {

	raftMap[serverId].eventCh <- true
	serverState[serverId] = KILLED
}

func ResurrectServer(serverId int) {
	serverState[serverId] = NORMAL
	raftMap[serverId].eventCh <- true
}

func KillLeader() int {

	//Find leader
	for _, serverRaft := range raftMap {

		if serverRaft.State == Leader {
			// MakeServerUnavailable(serverRaft.ServerID)
			KillServer(serverRaft.ServerID)
			return serverRaft.ServerID
		}
	}
	return -1
}
