package main

import (
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

Invoked by candidates to gather votes.

Receiver implementation:
1. Reply false if term < currentTerm
2. If votedFor is null or candidateId, and candidate's log is at least as up-to-date as receiver's log, grant vote
*/
func (s *RaftServer) HandleRequestVote(w http.ResponseWriter, r *http.Request) {
	var args RequestVoteArgs
	if err := parseIncomingRequest(r, &args); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	s.dlog("RequestVote received: %+v", args)

	results := s.executeRequestVote(args)
	if err := sendOutgoingResponse(w, &results); err != nil {
		http.Error(w, "Failed to marshal and send response", http.StatusInternalServerError)
		return
	}

	s.dlog("RequestVote response: %+v", results)
}

func (s *RaftServer) executeRequestVote(args RequestVoteArgs) RequestVoteResults {
	s.mu.Lock()
	defer s.mu.Unlock()

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
		s.StartOrResetElectionTimer()
	}

	return results
}

func (s *RaftServer) SendRequestVote(serverAddr ServerAddress, electionTerm int) {
	s.mu.Lock()

	var lastLogTerm int
	if len(s.log) == 0 {
		lastLogTerm = INITIAL_TERM
	} else {
		lastLogTerm = s.log[len(s.log)-1].term
	}

	args := RequestVoteArgs{
		Term:         electionTerm,
		CandidateID:  s.serverID,
		LastLogIndex: len(s.log) - 1,
		LastLogTerm:  lastLogTerm,
	}

	s.mu.Unlock()

	var results RequestVoteResults
	if err := makeFullRequest(serverAddr.host+":"+strconv.Itoa(serverAddr.port), &args, &results); err != nil {
		panic(err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if results.Term > s.currentTerm {
		s.convertToFollower(results.Term)
	} else if results.VoteGranted {
		s.votesReceived++
		if s.votesReceived > s.clusterSize/2 {
			s.promoteToLeader()
		}
	}
}
