package main

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

func (s *RaftServer) StartOrResetHeartbeatTimer() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.heartbeatTimer != nil {
		s.heartbeatTimer.Stop()
	}

	s.heartbeatTimer = time.AfterFunc(time.Duration(HEARTBEAT_TIMEOUT_MS)*time.Millisecond, func() {
		s.onHeartbeatTimeout()
	})
}

func (s *RaftServer) onHeartbeatTimeout() {
	log.Printf("Server %d: Heartbeat timeout occurred", s.serverID)

	s.mu.Lock()

	// Only a leader can send heartbeats
	if s.serverState != Leader {
		s.mu.Unlock()
		return
	}

	s.mu.Unlock()

	for serverID, serverAddr := range s.clusterAddrs {
		if serverID == s.serverID {
			continue
		}

		go s.sendHeartbeat(serverAddr)
	}

	s.StartOrResetHeartbeatTimer()
}

// TODO: replace this with a call to a sendAppendEntries function that can send any slice of log entries
func (s *RaftServer) sendHeartbeat(serverAddr ServerAddress) {
	s.mu.Lock()

	args := AppendEntriesArgs{
		Term:         s.currentTerm,
		LeaderID:     s.serverID,
		PrevLogIndex: INITIAL_INDEX,
		PrevLogTerm:  INITIAL_TERM,
		Entries:      []LogEntry{},
		LeaderCommit: s.commitIndex,
	}

	s.mu.Unlock()

	var results AppendEntriesResults
	if err := makeFullRequest(serverAddr.host+":"+strconv.Itoa(serverAddr.port), &args, &results); err != nil {
		panic(err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if results.Term > s.currentTerm {
		s.convertToFollower(results.Term)
	} else if !results.Success { // A heartbeat should never fail
		panic(fmt.Errorf("heartbeat failed"))
	}
}
