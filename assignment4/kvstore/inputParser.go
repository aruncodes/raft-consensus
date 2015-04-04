package main

import (
	"strconv"
	"strings"
)

func parseInput(command string) (Command, string) {

	fields := strings.Fields(command)
	length := len(fields)

	if length <= 0 {
		return Command{}, ERR_CMD_ERR
	}

	reqLen := 0
	switch fields[0] {
	case "set":
		reqLen = 4
	case "cas":
		reqLen = 5
	case "get":
		reqLen = 2
	case "getm":
		reqLen = 2
	case "delete":
		reqLen = 2
	default:
		reqLen = -1
	}

	//Length didn't match
	if reqLen != length {
		return Command{}, ERR_CMD_ERR
	}

	//Get, Getm and Delete requires no more validations while parsing
	if fields[0] == "get" || fields[0] == "getm" || fields[0] == "delete" {
		return Command{fields[0], fields[1], 0, 0, 0, ""}, ""
	}

	//Validate expiry timer
	expiryTime, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || expiryTime < 0 {
		return Command{}, ERR_CMD_ERR
	}

	//Validate number of bytes
	var numBytes int64
	if fields[0] == "set" {
		numBytes, err = strconv.ParseInt(fields[3], 10, 64)
	} else { //Cas
		numBytes, err = strconv.ParseInt(fields[4], 10, 64)
	}
	if err != nil || numBytes < 1 {
		return Command{}, ERR_CMD_ERR
	}

	//All validations for set completed
	if fields[0] == "set" {
		return Command{"set", fields[1], expiryTime, numBytes, 0, ""}, ""
	}

	//Version number for cas
	version, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return Command{}, ERR_CMD_ERR
	}

	//Return cas
	return Command{"cas", fields[1], expiryTime, numBytes, version, ""}, ""
}
