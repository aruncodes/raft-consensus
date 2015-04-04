package raft

import (
	"errors"
	"log"
	"net"
	"net/rpc"
	"strconv"
	"time"
)

//Actual RPC code
type RPC struct{}

//RPC listening server on every server
func (raft *Raft) RPCListener() {
	rpcObj := new(RPC)
	err := rpc.Register(rpcObj)

	if err != nil {
		log.Print("RCP register error : " + err.Error())
		return
	}

	listener, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(raft.LogPort), 10))
	if err != nil {
		log.Print("RCP error : " + err.Error())
		return
	}
	log.Println("RPC listner started:", raft.LogPort)

	for {
		if conn, err := listener.Accept(); err != nil {
			log.Print("Accept error : " + err.Error())
		} else {
			go rpc.ServeConn(conn)
		}
	}
}

//Function being called in follower when leader issues append
func (r *RPC) AppendEntriesRPC(args AppendRPCArgs, reply *AppendRPCResults) error {

	//Send to event channel of this server
	responseCh := make(chan AppendRPCResults, 5)
	raft.eventCh <- AppendRPC{args, responseCh}
	*reply = <-responseCh

	return nil
}

//Function called by leader
func (raft *Raft) appendEntiresRPC(server ServerConfig, args AppendRPCArgs, reply *AppendRPCResults) error {

	client, err := rpc.Dial("tcp", ":"+strconv.Itoa(server.LogPort))
	if err != nil {
		// log.Print("AppendRPC Dial error on port:" + strconv.Itoa(server.LogPort))
		// log.Print("Server ", server.Id, " down")
		return errors.New("Server " + strconv.Itoa(server.Id) + " down")
	}

	// err = client.Call("RPC.AppendEntriesRPC", args, reply) //Blocking RPC

	//Create a timeout timer so that we will not wait for ever
	timerChan := make(chan bool)
	timer := time.AfterFunc(heartbeatTimeout/2, func() { timerChan <- true })

	//Done channel for async rpc.Go()
	done := make(chan *rpc.Call, nServers)
	client.Go("RPC.AppendEntriesRPC", args, reply, done) //Non blocking RPC

	select {
	case <-timerChan:
		*reply = AppendRPCResults{raft.Term, false}
		return errors.New("Server " + strconv.Itoa(server.Id) + " down (timeout)")

	case response := <-done:

		if response.Error != nil {
			log.Print("RPC fail :" + response.Error.Error())
			return errors.New("RPC fail")
		}
		rep := response.Reply.(*AppendRPCResults)
		*reply = *rep
		timer.Stop()
	}
	//Async RPC end

	return nil
}

//Function being called in follower when candidate issues vote request
func (r *RPC) VoteRequestRPC(args RequestVoteArgs, reply *RequestVoteResult) error {

	//Send to event channel of this server
	responseCh := make(chan RequestVoteResult, 5)
	raft.eventCh <- VoteRequest{args, responseCh}
	*reply = <-responseCh

	return nil
}

//Function called by candidate
func (raft *Raft) voteRequestRPC(server ServerConfig, args RequestVoteArgs, reply *RequestVoteResult) error {

	client, err := rpc.Dial("tcp", ":"+strconv.Itoa(server.LogPort))
	if err != nil {
		// log.Print("VoteRPC Dial error on port:" + strconv.Itoa(server.LogPort))
		// log.Print("Server ", server.Id, " down")
		return errors.New("Server " + strconv.Itoa(server.Id) + " down")
	}

	// err = client.Call("RPC.VoteRequestRPC", args, reply) //Blocking RPC

	//Async RPC

	//Create a timeout timer so that we will not wait for ever
	timerChan := make(chan bool)
	timer := time.AfterFunc(heartbeatTimeout/2, func() { timerChan <- true })

	//Done channel for async rpc.Go()
	done := make(chan *rpc.Call, nServers)
	client.Go("RPC.VoteRequestRPC", args, reply, done) //Non blocking RPC

	select {
	case <-timerChan:
		*reply = RequestVoteResult{raft.Term, false}
		return errors.New("Server " + strconv.Itoa(server.Id) + " down (timeout)")

	case response := <-done:

		if response.Error != nil {
			log.Print("RPC fail :" + response.Error.Error())
			return errors.New("RPC fail")
		}
		rep := response.Reply.(*RequestVoteResult)
		*reply = *rep
		timer.Stop()
	}
	//Async RPC end

	return nil
}
