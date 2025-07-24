package main

import (
	"net/http"
	"time"
)

type LogEntry struct {
	// TODO: Command is just a string for now - add actual commands later
	Command string // command for state machine
	Term    int    // term when entry was received by leader
}

type SubmitCommandArgs struct {
	// TODO: for now, just a string
	Command string `json:"command"`
}

type SubmitCommandResults struct {
	Success  bool `json:"success"`
	LeaderID int  `json:"leaderId"`
}

func (s *RaftServer) HandleSubmitCommand(w http.ResponseWriter, r *http.Request) {
	var args SubmitCommandArgs
	if err := parseIncomingRequest(r, &args); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}
	s.dlog("SubmitCommand received: %+v", args)

	results := s.executeSubmitCommand(args)

	if err := s.persistToStableStorage(); err != nil {
		s.dlog("Error persisting to stable storage: %v", err)
		results.Success = false
	}

	if err := sendOutgoingResponse(w, &results); err != nil {
		http.Error(w, "Failed to marshal and send response", http.StatusInternalServerError)
		return
	}
	s.dlog("SubmitCommand response: %+v", results)
}

func (s *RaftServer) executeSubmitCommand(args SubmitCommandArgs) SubmitCommandResults {
	s.mu.Lock()

	results := SubmitCommandResults{Success: false, LeaderID: -1}

	// Redirect the client to the current leader if this server is not the leader
	if s.serverState != Leader {
		results.LeaderID = s.leaderID
		s.mu.Unlock()
		return results
	}

	s.log = append(s.log, LogEntry{Command: args.Command, Term: s.currentTerm})

	logIndex := len(s.log) - 1
	commandChan := make(chan bool, 1)
	s.pendingCommands[logIndex] = commandChan

	for serverID, serverAddr := range s.clusterAddrs {
		if serverID == s.serverID {
			continue
		}

		go s.SendAppendEntries(serverAddr)
	}

	s.checkAndUpdateCommitIndex()

	s.mu.Unlock()

	// Wait for the command to be committed
	select {
	case <-commandChan:
		results.Success = true
	case <-time.After(5 * time.Second): // 5 second timeout
		s.mu.Lock()
		delete(s.pendingCommands, logIndex)
		s.mu.Unlock()
		results.Success = false
	}

	return results
}

// Increase the commitIndex on the leader if possible, and apply any newly committed
// entries to the state machine.
// If there exists an N such that N > commitIndex, a majority of matchIndex[i] ≥ N,
// and log[N].term == currentTerm: set commitIndex = N.
func (s *RaftServer) checkAndUpdateCommitIndex() {
	if s.serverState != Leader {
		return
	}

	for i := s.commitIndex + 1; i < len(s.log); i++ {
		replicatedCount := 1 // Leader has the entry

		for serverID, matchIndex := range s.matchIndex {
			if serverID != s.serverID && matchIndex >= i {
				replicatedCount++
			}
		}

		majority := (s.clusterSize / 2) + 1
		if replicatedCount >= majority && s.log[i].Term == s.currentTerm {
			oldCommitIndex := s.commitIndex
			s.commitIndex = i

			// Apply any newly committed entries to the state machine
			if s.commitIndex > oldCommitIndex {
				s.applyCommittedEntries()
			}

			// Signal on the pending command channel that this entry has been committed
			if ch, exists := s.pendingCommands[i]; exists {
				ch <- true
				close(ch)
				delete(s.pendingCommands, i)
			}
		} else {
			break
		}
	}
}

// Applies all committed entries which haven't been applied yet to the state machine.
func (s *RaftServer) applyCommittedEntries() {
	for s.lastApplied < s.commitIndex {
		s.lastApplied++
		entry := s.log[s.lastApplied]
		s.applyCommandToStateMachine(entry.Command)
	}
}

// Applies a command to the state machine.
func (s *RaftServer) applyCommandToStateMachine(command string) {
	// TODO: Implement actual state machine logic here; for now, just logging the command
	s.dlog("Applying command to state machine: %s", command)
}

/*
Raft determines which of two logs is more up-to-date by comparing the
index and term of the last entries in the logs.

If the logs have last entries with different terms, then the log with
the later term is more up-to-date.

If the logs end with the same term, then whichever log is longer is
more up-to-date.
*/
func (s *RaftServer) candidateLogIsUpToDate(candidateLastLogIndex int, candidateLastLogTerm int) bool {
	receiverLastLogIndex := len(s.log) - 1
	var receiverLastLogTerm int

	if receiverLastLogIndex == INITIAL_INDEX {
		receiverLastLogTerm = INITIAL_TERM
	} else {
		receiverLastLogTerm = s.log[receiverLastLogIndex].Term
	}

	return candidateLastLogTerm > receiverLastLogTerm || (candidateLastLogTerm == receiverLastLogTerm && candidateLastLogIndex >= receiverLastLogIndex)
}

func (s *RaftServer) logEntryExists(logIndex int, logTerm int) bool {
	if logIndex == INITIAL_INDEX {
		return true
	}

	if logIndex >= len(s.log) {
		return false
	}

	entry := s.log[logIndex]
	return entry.Term == logTerm
}

func (s *RaftServer) updateLog(prevLogIndex int, newEntries []LogEntry) {
	// If no new entries, this is just a heartbeat
	if len(newEntries) == 0 {
		return
	}

	// Find the first conflicting entry
	conflictIndex := -1
	for i, entry := range newEntries {
		logIndex := prevLogIndex + 1 + i
		if logIndex < len(s.log) && s.log[logIndex].Term != entry.Term {
			conflictIndex = logIndex
			break
		}
	}

	// If no conflict found, just append new entries
	if conflictIndex == -1 {
		// Append only the new entries we don't already have
		numExistingEntries := len(s.log) - (prevLogIndex + 1)
		if numExistingEntries < len(newEntries) {
			s.log = append(s.log, newEntries[numExistingEntries:]...)
		}
		return
	}

	// Delete the first conflicting entry and all that follow it, then append the new entries not already in the log
	s.log = s.log[:conflictIndex]
	s.log = append(s.log, newEntries[conflictIndex-prevLogIndex-1:]...)
}
