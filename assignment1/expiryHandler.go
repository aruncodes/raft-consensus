package main

import (
	"time"
	"fmt"
)

var expiryTimer *time.Timer
var timerExpiryTime time.Time

var nextKeyToExpire string = ""
var nextKeyVersion int64 = -1

var running bool = false
var wakeup chan bool

func expiryHandler() {

	wakeup = make(chan bool)

	for {
		nextK, nextKV, duration := findNextKeyToExpire()

		nextKeyToExpire, nextKeyVersion = nextK, nextKV

		// if there is some values to expire
		if duration != -1 {
			expiryTimer = time.NewTimer( duration );

			//Set expiry time for others reference
			timerExpiryTime = time.Now().Add(duration)

			//Wait for the duration
			running = true
			debug(fmt.Sprintf("ExpiryHandler: sleeping for %v",duration))
			<- expiryTimer.C // Sleep 

			//Check if version changed just in case something goes wrong
			if m[nextKeyToExpire].version == nextKeyVersion {
				delete(m,nextKeyToExpire)
				debug("ExpiryHandler : "+nextKeyToExpire + " expired and removed")
			}
			//Reset expiry for finding new
			nextKeyToExpire = ""
			nextKeyVersion = -1
		} else {
			//There are no values to expire
			// Wait for dataStoreChanged() to wake up
			debug("ExpiryHandler going to sleep..")
			running = false
			<- wakeup
			debug("ExpiryHandler woke up..")
		}
	}
	
}

//Mode of data store changes
const (
	ADD = 1 << iota
	MODIFY
	DELETE
)

func dataStoreChanged(key string, mode int) {

	debug("Data store changed :"+key)
	//If handler is not running, just wake it up and exit
	if ! running {
		wakeup <- true
		return
	}

	//expiryHandler is waiting for timer to expire,
	// so update the next key to expire and change the timer

	if mode == ADD || mode == MODIFY{
		val := m[key]

		//Expiry time of value
		expiryTime := val.addTime.Add(time.Duration(val.exptime)*time.Second)

		//If the value expires before current timer, update timer
		if expiryTime.Before(timerExpiryTime) {
			//Remaning time for the value to expire
			remainingTime := expiryTime.Sub(time.Now())

			//update timer with remaning time for this key-value pair
			debug("Timer reset for "+key)
			expiryTimer.Reset(remainingTime)

			//update parameters
			nextKeyToExpire = key
			nextKeyVersion = val.version
		}
	}

	if mode == DELETE {
		// if the next key to expire is already deleted, force find next key
		if nextKeyToExpire == key {

			//Reset expiry for finding new
			nextKeyToExpire = ""
			nextKeyVersion = -1

			//reset timer
			expiryTimer.Reset(0)
		}
	}
}

func findNextKeyToExpire() (key string,version int64,exptime time.Duration) {
	
	//Iterate and find out least expiry key

	var nextKeyToExpire string = ""
	var keyVersion int64 = -1
	var shortestDuration time.Duration = -1

	currentTime := time.Now()
	for key,val := range m {

		if val.exptime == 0 {
			//Never expire
			continue
		}
		key = key
		//Expiry time of value
		expiryTime := val.addTime.Add(time.Duration(val.exptime)*time.Second)

		//Remaning time for the value to expire
		remainingTime := expiryTime.Sub(currentTime)

		//Update shortest duration
		if shortestDuration == -1 {
			shortestDuration = remainingTime
			nextKeyToExpire = key
			keyVersion = val.version
		} else if shortestDuration > remainingTime {
			shortestDuration = remainingTime
			nextKeyToExpire = key
			keyVersion = val.version
		}

		debug(fmt.Sprintf("Shortest :%s Time: %v",key,shortestDuration))
	}	

	return nextKeyToExpire,keyVersion,shortestDuration
}