package raft

import (
	"log"
	"math/rand"
	"time"
)

const (
	Follower  = "Follower"
	Leader    = "Leader"
	Candidate = "Candidate"
)

const (
	followerTimeout  = 3 * time.Second
	heartbeatTimeout = 2 * time.Second
)

type ClientAppend struct {
	command    Command
	responseCh chan LogEntry
}

type VoteRequest struct {
	args       RequestVoteArgs
	responseCh chan RequestVoteResult
}

type AppendRPC struct {
	args       AppendRPCArgs
	responseCh chan AppendRPCResults
}

type Timeout struct {
}

func (raft *Raft) loop() {

	for {
		switch raft.State {

		case Follower:
			raft.Follower()

		case Candidate:
			raft.Candidate()

		case Leader:
			raft.Leader()

		default:
			break
		}
	}
}

func (raft *Raft) Follower() {

	//Start timer
	timeoutFunc := func() {
		raft.eventCh <- Timeout{}
	}
	r := time.Duration(rand.Intn(100)) * time.Millisecond
	timer := time.AfterFunc(followerTimeout+r, timeoutFunc) //debug: 3 seconds + some millis

	for {

		event := <-raft.eventCh

		switch event.(type) {

		case ClientAppend:
			raft.LogState("Append received")

			//Sent false to make it redirect
			ev := event.(ClientAppend)
			logItem := LogItem{0, ev.command, false}
			ev.responseCh <- logItem

		case AppendRPC:
			ev := event.(AppendRPC)

			if len(ev.args.Log) == 0 {
				raft.LogState("Heartbeat received")
			} else {
				raft.LogState("AppendRPC received")
			}

			reply := AppendRPCResults{raft.Term, true} //Send true for append now
			ev.responseCh <- reply
			//TODO: actual append

			//Update Leader ID
			raft.LeaderID = ev.args.LeaderId

			r := time.Duration(rand.Intn(100)) * time.Millisecond
			timer.Reset(followerTimeout + r)

		case VoteRequest:
			raft.LogState("Vote request received")

			ev := event.(VoteRequest)

			reply := RequestVoteResult{raft.Term, true} //Vote to any requests for now
			ev.responseCh <- reply
			//TODO: Actual voting

		case Timeout:
			raft.LogState("Time out received")
			raft.State = Candidate
			return

		default:
			raft.LogState("Unknown received")
		}
	}
}
func (raft *Raft) Leader() {

	raft.LogState("")

	//Start timer
	timeoutFunc := func() {
		raft.eventCh <- Timeout{}
	}
	timer := time.AfterFunc(heartbeatTimeout, timeoutFunc) //debug: 3 seconds

	for {

		raft.heartBeat()

		event := <-raft.eventCh

		switch event.(type) {

		case ClientAppend:
			raft.LogState("Append received")

			ev := event.(ClientAppend)

			logItem := LogItem{raft.LastLsn + 1, ev.command, false}
			raft.Log = append(raft.Log, logItem)
			raft.LastLsn++

			ev.responseCh <- logItem

		case AppendRPC:
			raft.LogState("AppendRPC received")
			//Someone else is leader

		case VoteRequest:
			raft.LogState("Vote request received")
			//Some one became a candidate, network problem?

		case Timeout:
			raft.LogState("Heartbeat time out")
			//Send append RPCs
			timer.Reset(heartbeatTimeout)
			continue

		default:
			raft.LogState("Unknown received")
		}
	}
}
func (raft *Raft) Candidate() {

	raft.LogState("")

	timeoutFunc := func() {
		raft.eventCh <- Timeout{}
	}

	for {

		//Send vote request to all
		votes := 0
		for _, server := range ClusterInfo.Servers {

			if raft.ServerID == server.Id {
				// The current running server
				votes++ // self vote
				continue
			}

			//Create args and reply
			args := RequestVoteArgs{raft.Term, uint64(raft.ServerID)}
			reply := RequestVoteResult{}

			//Request vote by RPC (fake)
			err := raft.requestVote(server, args, &reply)

			if err != nil {
				log.Println(err.Error())
				continue
			}

			if reply.VoteGranted == true {
				votes++
			}
		}

		if votes > len(ClusterInfo.Servers)/2 {
			//Got majority of votes
			raft.State = Leader
			raft.LeaderID = raft.ServerID
			return
		}

		//Reached here means didn't get majority of votes,
		//So start a timer and wait to see if we get any append RPC from
		//new leader

		r := time.Duration(rand.Intn(100)) * time.Millisecond
		timer := time.AfterFunc(followerTimeout+r, timeoutFunc) //debug: 3 seconds + some millis

		event := <-raft.eventCh

		switch event.(type) {

		case ClientAppend:
			raft.LogState("Append received")
			//When a candidate receives a client append, it cannot do anything
			//since there is no leader and we cannot send redirect

			//My solution: Create a timer to send same request to event channel
			//after a timeout hoping there will be a leader at that time
			resendEvent := func() {
				raft.eventCh <- event
			}
			time.AfterFunc(followerTimeout, resendEvent)

		case AppendRPC:
			raft.LogState("AppendRPC received")
			//This must from a new leader.
			//Verfiy his term and respond
			ev := event.(AppendRPC)

			success := false
			if ev.args.Term > raft.Term {
				//He is a leader
				raft.Term = ev.args.Term
				raft.State = Follower
				success = true
			}
			reply := AppendRPCResults{raft.Term, success} //Send true for append now
			ev.responseCh <- reply
			//TODO: actual append
			if success {
				timer.Stop()
				return
			}
		case VoteRequest:
			raft.LogState("Vote request received")
			//Do not vote as i am a candidate
			ev := event.(VoteRequest)

			reply := RequestVoteResult{raft.Term, false}
			ev.responseCh <- reply

		case Timeout:
			raft.LogState("Time out received")
			//Stand again as candidate
			//Increment term and Request votes
			raft.Term++
			//TODO: Fix isolated server increments term problem
			continue

		default:
			raft.LogState("Unknown received")
		}
	}
}

func (raft *Raft) LogState(msg string) {
	log.Print("Server:", raft.ServerID, " - ", raft.State, ": ", msg)
}
