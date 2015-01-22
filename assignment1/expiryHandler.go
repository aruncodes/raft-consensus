package main

import (
	"strconv"
)

func expiryHandler(command []string) {

	key := command[1]
	version, err := strconv.ParseInt(command[2], 10, 64)
	if err != nil {
		debug("Invalid version specified by expiry Handler :" + command[2])
		return
	}

	//Check if key available in data store
	val, ok := m[key]

	if ok == false {
		//Key already removed
		return
	}

	if val.version == version {
		//Remove key
		delete(m, key)
		debug(key + " expired.")
	}
}
