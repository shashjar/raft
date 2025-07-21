package main

type LogEntry struct {
	// TODO: command is just a string for now - add actual commands later
	command string // command for state machine
	term    int    // term when entry was received by leader
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

	if receiverLastLogIndex < 0 {
		receiverLastLogTerm = 0
	} else {
		receiverLastLogTerm = s.log[receiverLastLogIndex].term
	}

	return candidateLastLogTerm > receiverLastLogTerm || (candidateLastLogTerm == receiverLastLogTerm && candidateLastLogIndex >= receiverLastLogIndex)
}

func (s *RaftServer) logEntryExists(logIndex int, logTerm int) bool {
	if logIndex < 0 || logIndex >= len(s.log) {
		return false
	}

	entry := s.log[logIndex]
	return entry.term == logTerm
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
		if logIndex < len(s.log) && s.log[logIndex].term != entry.term {
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
