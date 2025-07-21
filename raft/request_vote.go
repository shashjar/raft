package main

import (
	"log"
	"net/http"
	"strconv"
)

type RequestVoteArgs struct {
	Term         int `json:"term"`         // candidate's term
	CandidateID  int `json:"candidateId"`  // candidate requesting vote
	LastLogIndex int `json:"lastLogIndex"` // index of candidate's last log entry
	LastLogTerm  int `json:"lastLogTerm"`  // term of candidate's last log entry
}

type RequestVoteResults struct {
	Term        int  `json:"term"`        // currentTerm, for candidate to update itself
	VoteGranted bool `json:"voteGranted"` // true means candidate received vote
}

/*
RequestVote RPC handler

# Invoked by candidates to gather votes

Receiver implementation:
1. Reply false if term < currentTerm
2. If votedFor is null or candidateId, and candidate's log is at least as up-to-date as receiver's log, grant vote
*/
func (s *RaftServer) handleRequestVote(w http.ResponseWriter, r *http.Request) {
	log.Println("RequestVote RPC received")

	var args RequestVoteArgs
	if err := parseIncomingRequest(r, &args); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	results := s.executeRequestVote(args)
	if err := sendOutgoingResponse(w, &results); err != nil {
		http.Error(w, "Failed to marshal and send response", http.StatusInternalServerError)
		return
	}
}

func (s *RaftServer) executeRequestVote(args RequestVoteArgs) RequestVoteResults {
	if args.Term > s.currentTerm {
		s.convertToFollower(args.Term)
	}

	results := RequestVoteResults{Term: s.currentTerm, VoteGranted: false}

	// (1)
	if args.Term < s.currentTerm {
		return results
	}

	// (2)
	if (s.votedFor == -1 || s.votedFor == args.CandidateID) && s.candidateLogIsUpToDate(args.LastLogIndex, args.LastLogTerm) {
		results.VoteGranted = true
		s.votedFor = args.CandidateID
		s.startOrResetElectionTimer()
	}

	return results
}

func (s *RaftServer) sendRequestVote(serverAddr ServerAddress) {
	var lastLogTerm int
	if len(s.log) == 0 {
		lastLogTerm = INITIAL_TERM
	} else {
		lastLogTerm = s.log[len(s.log)-1].term
	}

	args := RequestVoteArgs{
		Term:         s.currentTerm,
		CandidateID:  s.serverID,
		LastLogIndex: len(s.log) - 1,
		LastLogTerm:  lastLogTerm,
	}

	var results RequestVoteResults
	if err := makeFullRequest(serverAddr.host+":"+strconv.Itoa(serverAddr.port), &args, &results); err != nil {
		panic(err)
	}

	if results.Term > s.currentTerm {
		s.convertToFollower(results.Term)
	} else if results.VoteGranted {
		s.votesMutex.Lock()
		defer s.votesMutex.Unlock()

		s.votesReceived++
		if s.votesReceived > s.clusterSize/2 {
			s.promoteToLeader()
		}
	}
}
