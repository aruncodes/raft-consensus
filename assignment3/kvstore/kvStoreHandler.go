package main

import (
	"assignment3/raft"
	"fmt"
	"log"
	"math/rand"
	"time"
)

func kvStoreHandler(commitCh chan raft.LogEntry, kvResponse chan KVResponse) {

	//Create kv store
	kvstore := make(map[string]value)

	for {
		logEntry := <-commitCh //Receive from raft
		command := Command(logEntry.Data())

		response := ""
		switch command.Cmd {
		case "set", "cas":
			response = setCas(command, kvstore, commitCh)
		case "get", "getm":
			response = getValueMeta(command, kvstore)
		case "delete":
			response = deleteKey(command, kvstore)
		case "expire":
			response = expiryHandler(command, kvstore)
			continue //No one is waiting for response
		default:
			continue
		}

		if logEntry.Committed() {
			//Sent while being a follower, so client is not waiting
			//No response need to be sent
			continue
		}

		//Send back response to kvResponse channel
		//clientConnManager will be waiting
		kvResponse <- KVResponse{logEntry.Lsn(), response}
	}
}

func setCas(command Command, kvstore map[string]value, commitCh chan raft.LogEntry) string {

	key := command.Key
	val := command.Value
	numbytes := command.Length
	exptime := command.ExpiryTime
	version := command.Version

	//Check if already exist
	data, ok := kvstore[key]

	if command.Cmd == "set" {
		if ok == true {
			log.Print("Key already exists")
			return ERR_VERSION
		}
		//Get a random number as version
		version = int64(rand.Intn(10000))

	} else { //CAS
		if ok == false {
			log.Print("Key not found")
			return ERR_NOT_FOUND
		}

		if data.version != version {
			log.Print("Version mismatch")
			return ERR_VERSION
		}

		//Increment Version
		version++
	}

	//Add value to keystore
	kvstore[key] = value{[]byte(val), numbytes, version, exptime}

	//Inform KV-Handler by sending a fake log entry to expire
	sendExpiry := func() {
		command := raft.Command{"expire", key, 0, 0, version, ""}
		fakeLogEntry := raft.LogItem{raft.Lsn(0), command, true, 0}

		commitCh <- fakeLogEntry
	}

	//Set expiry timer
	if exptime > 0 {
		time.AfterFunc(time.Duration(exptime)*time.Second, sendExpiry)
	}

	return fmt.Sprintf("OK %d", version)
}

func getValueMeta(command Command, kvstore map[string]value) string {
	key := command.Key

	//Check if already exist
	val, ok := kvstore[key]

	if ok == false {
		log.Print("Key not found")
		return ERR_NOT_FOUND
	}

	retStr := "VALUE "

	if command.Cmd == "get" {
		retStr += fmt.Sprintf("%d", val.numbytes) + "\r\n"
	} else if command.Cmd == "getm" {
		retStr += fmt.Sprintf("%d %d %d", val.version, val.exptime, val.numbytes) + "\r\n"
	}

	retStr += string(val.val)

	return retStr
}

func deleteKey(command Command, kvstore map[string]value) string {
	key := command.Key

	//Check if already exist
	_, ok := kvstore[key]

	if ok == false {
		log.Print("Key not found")
		return ERR_NOT_FOUND
	}

	// If value is present delete it
	delete(kvstore, key)

	return "DELETED"
}

func expiryHandler(command Command, kvstore map[string]value) string {
	key := command.Key
	version := command.Version

	//Check if exist
	val, ok := kvstore[key]

	if ok == false {
		//Key already removed
		return ERR_NOT_FOUND
	}

	if val.version == version {
		// If same version, remove it
		delete(kvstore, key)
		log.Print(key + " expired.")
	}

	return "EXPIRY_SUCCESS"
}
