package raft

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
)

func (raft *Raft) StateToBytes() []byte {

	//Initialize encoder
	w := bytes.Buffer{}
	enc := gob.NewEncoder(&w)

	//Write current term
	err := enc.Encode(raft.Term)
	if err != nil {
		log.Println(err)
	}

	//Write voted for
	err = enc.Encode(raft.VotedFor)
	if err != nil {
		log.Println(err)
	}

	//Write log
	err = enc.Encode(raft.Log)
	if err != nil {
		log.Println(err)
	}

	return w.Bytes()
}

func (raft *Raft) BytesToState(b []byte) (uint64, int, []LogItem) {

	//Initialize decoder
	r := bytes.Buffer{}
	r.Write(b)
	dec := gob.NewDecoder(&r)

	//Read term
	var term uint
	err := dec.Decode(&term)
	if err != nil {
		log.Println(err)
	}

	//Read VotedFor
	var votedFor int
	err = dec.Decode(&votedFor)
	if err != nil {
		log.Println(err)
	}

	//Read log
	var logArray []LogItem
	err = dec.Decode(&logArray)
	if err != nil {
		log.Println(err)
	}

	return uint64(term), votedFor, logArray
}

func (raft *Raft) WriteStateToFile(filePath string) error {

	fileName := fmt.Sprintf("%s_S%d.state", filePath, raft.ServerID)
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)

	if err != nil {
		log.Println(err)
		return err
	}

	defer file.Close()

	if err != nil {
		log.Println(err)
		return err
	}

	//Get state byte array
	b := raft.StateToBytes()

	//Write state to file
	_, err = file.Write(b)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (raft *Raft) FileExist(filePath string) bool {
	fileName := fmt.Sprintf("%s_S%d.state", filePath, raft.ServerID)
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0666)

	if err != nil {
		return false
	}
	file.Close()

	return true
}

func (raft *Raft) ReadStateFromFile(filePath string) error {

	fileName := fmt.Sprintf("%s_S%d.state", filePath, raft.ServerID)
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0666)

	if err != nil {
		log.Println(err)
		return err
	}

	defer file.Close()

	if err != nil {
		log.Println(err)
		return err
	}

	//Read bytes from file
	data := make([]byte, 1024)
	n, err := file.Read(data)
	if err != nil {
		log.Println(err)
		return err
	}

	data = data[:n] //Trim data

	//Decode data
	Term, VotedFor, Log := raft.BytesToState(data)

	//Restore persistant raft state
	raft.Lock.Lock()
	raft.Term = Term
	raft.VotedFor = VotedFor
	raft.Log = Log
	raft.Lock.Unlock()

	log.Println("Restored state: Term:", Term, "Voted for:", VotedFor /*, "Log:", Log*/)

	return nil
}

func (raft *Raft) CommandToBytes(cmd Command) []byte {

	w := bytes.Buffer{}
	enc := gob.NewEncoder(&w)

	err := enc.Encode(cmd)
	if err != nil {
		log.Println(err)
	}
	return w.Bytes()
}

func (raft *Raft) BytesToCommand(b []byte) Command {

	r := bytes.Buffer{}
	r.Write(b)

	dec := gob.NewDecoder(&r)

	cmd := Command{}
	err := dec.Decode(&cmd)
	if err != nil {
		log.Println(err)
	}
	return cmd
}
