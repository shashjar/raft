package main

import (
	"fmt"
	"net/http"
	"strconv"
)

type AppendEntriesArgs struct {
	Term         int        `json:"term"`         // leader's term
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

Invoked by leader to replicate log entries; also used as heartbeat.

Receiver implementation:
1. Reply false if term < currentTerm
2. Reply false if log doesn't contain an entry at prevLogIndex whose term matches prevLogTerm
3. If an existing entry conflicts with a new one (same index but different terms), delete the existing entry and all that follow it
4. Append any new entries not already in the log
5. If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
*/
func (s *RaftServer) HandleAppendEntries(w http.ResponseWriter, r *http.Request) {
	var args AppendEntriesArgs
	if err := parseIncomingRequest(r, &args); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}
	s.dlog("AppendEntries received: %+v", args)

	results := s.executeAppendEntries(args)

	if err := s.persistToStableStorage(); err != nil {
		s.dlog("Error persisting to stable storage: %v", err)
		results.Success = false
	}

	if err := sendOutgoingResponse(w, &results); err != nil {
		http.Error(w, "Failed to marshal and send response", http.StatusInternalServerError)
		return
	}
	s.dlog("AppendEntries response: %+v", results)
}

func (s *RaftServer) executeAppendEntries(args AppendEntriesArgs) AppendEntriesResults {
	s.mu.Lock()
	defer s.mu.Unlock()

	if args.Term > s.currentTerm || (args.Term == s.currentTerm && s.serverState != Follower) {
		s.convertToFollower(args.Term)
	}

	results := AppendEntriesResults{Term: s.currentTerm, Success: false}

	// (1)
	if args.Term < s.currentTerm {
		return results
	}

	// When receiving a valid AppendEntries RPC: set leader ID & reset election timer
	s.leaderID = args.LeaderID
	s.StartOrResetElectionTimer()

	// (2)
	// TODO: in this case, could respond to the leader with the actual previous log index and term to make the retry more efficient (described at the end of section 5.3)
	if !s.logEntryExists(args.PrevLogIndex, args.PrevLogTerm) {
		return results
	}

	// (3) & (4)
	s.updateLog(args.PrevLogIndex, args.Entries)

	// (5)
	if args.LeaderCommit > s.commitIndex {
		oldCommitIndex := s.commitIndex
		lastNewEntryIndex := len(s.log) - 1
		s.commitIndex = min(args.LeaderCommit, lastNewEntryIndex)
		if s.commitIndex > oldCommitIndex {
			s.applyCommittedEntries()
		}
	}

	results.Success = true
	return results
}

func (s *RaftServer) SendAppendEntries(serverAddr ServerAddress) {
	for {
		s.mu.Lock()

		nextIndex, exists := s.nextIndex[serverAddr.id]
		if !exists {
			panic(fmt.Errorf("nextIndex for server %d not found", serverAddr.id))
		}

		prevLogIndex := nextIndex - 1
		prevLogTerm := INITIAL_TERM
		if prevLogIndex >= 0 {
			prevLogTerm = s.log[prevLogIndex].Term
		}

		newEntries := s.log[nextIndex:]

		args := AppendEntriesArgs{
			Term:         s.currentTerm,
			LeaderID:     s.serverID,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			Entries:      newEntries,
			LeaderCommit: s.commitIndex,
		}

		s.mu.Unlock()

		var results AppendEntriesResults
		if err := makeFullRequest(serverAddr.host+":"+strconv.Itoa(serverAddr.port), APPEND_ENTRIES_PATH, &args, &results); err != nil {
			panic(err)
		}

		s.mu.Lock()

		if results.Term > s.currentTerm {
			s.convertToFollower(results.Term)
			s.mu.Unlock()
			return
		}

		if results.Success { // If successful: update nextIndex and matchIndex
			s.nextIndex[serverAddr.id] = nextIndex + len(newEntries)
			s.matchIndex[serverAddr.id] = s.nextIndex[serverAddr.id] - 1
			s.checkAndUpdateCommitIndex() // Check if any new entries can be committed
			s.mu.Unlock()
			return
		} else { // If failed because of log inconsistency: decrement nextIndex and try again
			if nextIndex > 0 {
				s.nextIndex[serverAddr.id] = nextIndex - 1
			}
			s.mu.Unlock()
		}
	}
}
