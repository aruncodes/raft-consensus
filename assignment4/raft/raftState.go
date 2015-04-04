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
	Killed    = "Killed" //For testing
)

const (
	followerTimeout  = /*3 * time.Second // */ 1000 * time.Millisecond
	heartbeatTimeout = /*2 * time.Second // */ 750 * time.Millisecond
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

		case Killed: //Testing purposes
			raft.Killed()

		default:
			raft.LogState("Unknown state")
			break
		}
	}
}

func (raft *Raft) Killed() {

	for {
		event := <-raft.eventCh

		switch event.(type) {

		case ErrorSimulation:
			ev := event.(ErrorSimulation)
			raft.State = ev.State
			return

		default:
			raft.LogState("Received something while dead")
			continue
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
			logItem := LogItem{0, ev.command, false, raft.Term}
			ev.responseCh <- logItem

		case AppendRPC:
			ev := event.(AppendRPC)

			if len(ev.args.Log) == 0 {
				raft.LogState("Heartbeat received")
			} else {
				raft.LogState("AppendRPC received")
			}

			success := raft.appendEntries(ev.args)

			if success && len(ev.args.Log) > 0 {
				//Disk write if log was updated
				err := raft.WriteStateToFile(FILENAME)
				checkError(err)
			}

			//Respond to RPC
			reply := AppendRPCResults{raft.Term, success}
			ev.responseCh <- reply

			r := time.Duration(rand.Intn(100)) * time.Millisecond
			timer.Reset(followerTimeout + r)

		case VoteRequest:
			// raft.LogState("Vote request received")

			ev := event.(VoteRequest)

			voted := raft.shouldIVote(ev.args)

			if voted {
				raft.Term = ev.args.Term
				raft.VotedFor = int(ev.args.CandidateID)
				raft.LogState("Voted ")

				//Disk write
				err := raft.WriteStateToFile(FILENAME)
				checkError(err)

			} else {
				raft.LogState("Vote request rejected")
			}

			ev.responseCh <- RequestVoteResult{raft.Term, voted} //Actual vote

			//Again wait since someone is a candidate
			r := time.Duration(rand.Intn(100)) * time.Millisecond
			timer.Reset(followerTimeout + r)

		case Timeout:
			raft.LogState("Time out received")
			raft.State = Candidate
			return

		case ErrorSimulation:
			ev := event.(ErrorSimulation)
			raft.State = ev.State
			timer.Stop()
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
	timer := time.AfterFunc(0, timeoutFunc) //For first time,start immediately

	//Update raft state of followers known to leader
	for i, _ := range raft.NextIndex {
		raft.NextIndex[i] = raft.LastLsn + 1
		raft.MatchIndex[i] = 0
	}

	for {

		event := <-raft.eventCh

		switch event.(type) {

		case ClientAppend:
			raft.LogState("Append received")

			ev := event.(ClientAppend)

			logItem := LogItem{raft.LastLsn + 1, ev.command, false, raft.Term}
			raft.Log = append(raft.Log, logItem)
			raft.LastLsn++

			ev.responseCh <- logItem

		case AppendRPC:
			raft.LogState("AppendRPC received")
			//TODO: Someone else is leader (or thinks)
			ev := event.(AppendRPC)
			// log.Print("AppendRPC received from ", ev.args.LeaderId, " term ", ev.args.Term)

			if ev.args.Term > raft.Term {
				//He is actually the leader
				raft.State = Follower
				raft.Term = ev.args.Term
				raft.VotedFor = -1

				ev.responseCh <- AppendRPCResults{raft.Term, true}

				timer.Stop()
				return // return as follower
			} else {
				//We are actually the leader
				ev.responseCh <- AppendRPCResults{raft.Term, false}
			}

		case VoteRequest:
			// raft.LogState("Vote request received")
			//Some one became a candidate, network problem?
			ev := event.(VoteRequest)

			voted := raft.shouldIVote(ev.args)

			if voted {
				//Am I mistakenly thought I am leader?
				raft.State = Follower
				raft.Term = ev.args.Term
				raft.VotedFor = int(ev.args.CandidateID)
				raft.LogState("Voted ")

				//Disk write
				err := raft.WriteStateToFile(FILENAME)
				checkError(err)

			} else {
				raft.LogState("Vote request rejected")
			}

			ev.responseCh <- RequestVoteResult{raft.Term, voted} //Actual vote

			if voted {
				timer.Stop()
				return // Since state changed
			}

		case Timeout:
			raft.LogState("Heartbeat time out")
			//Send append RPCs
			raft.heartBeat()
			timer.Reset(heartbeatTimeout)

			if raft.State == Leader {
				continue //Wait for next event/timeout
			} else {
				timer.Stop()
				return //State could be changed while heart beating
			}

		case ErrorSimulation:
			ev := event.(ErrorSimulation)
			raft.State = ev.State
			timer.Stop()
			return

		default:
			raft.LogState("Unknown received")
		}
	}
}

func (raft *Raft) Candidate() {

	//Increment term and Request votes
	raft.Term++
	raft.VotedFor = -1

	raft.LogState("")

	//Send vote request to all
	votes := 0
	for _, server := range ClusterInfo.Servers {

		if raft.ServerID == server.Id {
			// The current running server
			votes++ // self vote
			raft.VotedFor = raft.ServerID
			continue
		}

		//Create args and reply
		lastLogTerm := raft.Log[raft.LastLsn].Term
		args := RequestVoteArgs{raft.Term, uint64(raft.ServerID), raft.LastLsn, lastLogTerm}
		reply := RequestVoteResult{}

		//Request vote by RPC
		// err := raft.requestVote(server, args, &reply) //fake
		err := raft.voteRequestRPC(server, args, &reply) //

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

	timeoutFunc := func() {
		raft.eventCh <- Timeout{}
	}

	timeoutDelta := int(0.2 * float32(followerTimeout)) //20% of follower timeout
	randDelta := time.Duration(rand.Intn(timeoutDelta))
	timer := time.AfterFunc(followerTimeout+randDelta, timeoutFunc)

	for {

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
			//This must be from a new leader.
			//Verfiy his term and respond
			ev := event.(AppendRPC)

			if ev.args.Term > raft.Term {
				//He is a leader
				//Change to follower state and resend this
				//to event channel so that this will be
				//handled while being a follower

				raft.Term = ev.args.Term
				raft.VotedFor = -1
				raft.State = Follower

				//Resend
				raft.eventCh <- event

				timer.Stop()
				return //return as follower
			} else {
				//Not from a valid leader
				reply := AppendRPCResults{raft.Term, false}
				ev.responseCh <- reply
			}

		case VoteRequest:
			//Vote if eligible
			ev := event.(VoteRequest)

			voted := raft.shouldIVote(ev.args)

			if voted {
				raft.Term = ev.args.Term
				raft.VotedFor = int(ev.args.CandidateID)
				raft.LogState("Voted ")

				//Disk write
				err := raft.WriteStateToFile(FILENAME)
				checkError(err)

			} else {
				raft.LogState("Vote request rejected")
			}

			ev.responseCh <- RequestVoteResult{raft.Term, voted} //Actual vote

			if voted {
				//Go back to follower state
				raft.State = Follower
				timer.Stop()
				return
			}

		case Timeout:
			raft.LogState("Time out received")
			//Stand again as candidate
			//Increment term and Request votes
			if votes > 1 {
				//Some one else also voted other than me
				raft.Term++
				raft.VotedFor = -1
			} else {
				//Donot increment term as this will cause
				//the server to increment terms while being
				//isolated
			}
			return //Come back as candidate since state is not changed

		case ErrorSimulation:
			ev := event.(ErrorSimulation)
			raft.State = ev.State
			timer.Stop()
			return

		default:
			raft.LogState("Unknown received")
		}
	}
}

func (raft *Raft) LogState(msg string) {

	if raft.State == Leader {
		//A line to distinguish rounds of new leaders in log
		log.Print("\r---------------------------------")
	}

	log.Print("S", raft.ServerID, " ", raft.State, " (T", raft.Term, ")", " (L", raft.LeaderID, ")", ": ", msg)
}

func checkError(e error) {
	if e != nil {
		log.Println(e.Error())
	}
}
