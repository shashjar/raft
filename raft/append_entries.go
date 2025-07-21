package main

import (
	"log"
	"net/http"
)

type AppendEntriesArgs struct {
	Term int `json:"term"` // leader's term
	// TODO: not used yet, but will be used for client redirection. store the current leader ID in the server state
	LeaderID     int        `json:"leaderId"`     // so follower can redirect clients
	PrevLogIndex int        `json:"prevLogIndex"` // index of log entry immediately preceding new ones
	PrevLogTerm  int        `json:"prevLogTerm"`  // term of prevLogIndex entry
	Entries      []LogEntry `json:"entries"`      // log entries to store (empty for heartbeat, may send more than one for efficiency)
	LeaderCommit int        `json:"leaderCommit"` // leader's commitIndex
}

type AppendEntriesResults struct {
	Term    int  `json:"term"`    // currentTerm, for leader to update itself
	Success bool `json:"success"` // true if follower contained entry matching prevLogIndex and prevLogTerm
}

/*
AppendEntries RPC handler

Invoked by leader to replicate log entries; also used as heartbeat

Receiver implementation:
1. Reply false if term < currentTerm
2. Reply false if log doesn't contain an entry at prevLogIndex whose term matches prevLogTerm
3. If an existing entry conflicts with a new one (same index but different terms), delete the existing entry and all that follow it
4. Append any new entries not already in the log
5. If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
*/
func (s *RaftServer) handleAppendEntries(w http.ResponseWriter, r *http.Request) {
	log.Println("AppendEntries RPC received")

	var args AppendEntriesArgs
	if err := parseIncomingRequest(r, &args); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	results := s.executeAppendEntries(args)
	if err := sendOutgoingResponse(w, &results); err != nil {
		http.Error(w, "Failed to marshal and send response", http.StatusInternalServerError)
		return
	}
}

func (s *RaftServer) executeAppendEntries(args AppendEntriesArgs) AppendEntriesResults {
	if args.Term >= s.currentTerm {
		s.currentTerm = args.Term
		s.votedFor = -1
		s.serverState = Follower
	}

	results := AppendEntriesResults{Term: s.currentTerm, Success: false}

	// (1)
	if args.Term < s.currentTerm {
		return results
	}

	// (2)
	if !s.logEntryExists(args.PrevLogIndex, args.PrevLogTerm) {
		return results
	}

	// Reset election timer when receiving valid AppendEntries (heartbeat)
	s.startOrResetElectionTimer()

	// (3) & (4)
	s.updateLog(args.PrevLogIndex, args.Entries)

	// (5)
	if args.LeaderCommit > s.commitIndex {
		lastNewEntryIndex := max(len(s.log)-1, 0)
		s.commitIndex = min(args.LeaderCommit, lastNewEntryIndex)
	}

	results.Success = true
	return results
}
