package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

func kvStoreHandler() {

	//Create kv store
	kvstore = make(map[string]value)

	for {
		bundle := <-kvQueue	//Receive bundle from client
		command := bundle.command

		response := ""
		switch command.Cmd {
		case "set", "cas":
			response = setCas(command)
		case "get", "getm":
			response = getValueMeta(command)
		case "delete":
			response = deleteKey(command)
		case "expire":
			response = expiryHandler(command)
			continue //No one is waiting for response
		}

		//Send back response
		bundle.responseChan <- response
	}
}

func setCas(command Command) string {

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

	//Inform expiryHandler
	sendExpiry := func() {
		command := Command{"expire", key, 0, 0, version, ""}
		bundle := KVBundle{command, nil}
		// bundle := KVBundle{encodeCommand(command), nil}

		kvQueue <- bundle
	}

	//Set expiry timer
	if exptime > 0 {
		time.AfterFunc(time.Duration(exptime)*time.Second, sendExpiry)
	}

	return fmt.Sprintf("OK %d", version)
}

func getValueMeta(command Command) string {
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

func deleteKey(command Command) string {
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

func expiryHandler(command Command) string {
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
