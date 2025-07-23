package main

import (
	"fmt"
	"strconv"
	"time"
)

const HEARTBEAT_TIMEOUT_MS = 50

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
	s.dlog("Heartbeat timeout occurred")

	s.mu.Lock()

	// Only a leader can send heartbeats
	if s.serverState != Leader {
		s.mu.Unlock()
		return
	}

	heartbeatTerm := s.currentTerm

	s.mu.Unlock()

	for serverID, serverAddr := range s.clusterAddrs {
		if serverID == s.serverID {
			continue
		}

		go s.sendHeartbeat(serverAddr, heartbeatTerm)
	}

	s.StartOrResetHeartbeatTimer()
}

func (s *RaftServer) sendHeartbeat(serverAddr ServerAddress, heartbeatTerm int) {
	s.mu.Lock()

	args := AppendEntriesArgs{
		Term:         heartbeatTerm,
		LeaderID:     s.serverID,
		PrevLogIndex: INITIAL_INDEX,
		PrevLogTerm:  INITIAL_TERM,
		Entries:      []LogEntry{},
		LeaderCommit: s.commitIndex,
	}

	s.mu.Unlock()

	var results AppendEntriesResults
	if err := makeFullRequest(serverAddr.host+":"+strconv.Itoa(serverAddr.port), APPEND_ENTRIES_PATH, &args, &results); err != nil {
		panic(err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if results.Term > heartbeatTerm {
		s.convertToFollower(results.Term)
	} else if !results.Success { // A heartbeat should never fail
		panic(fmt.Errorf("heartbeat failed"))
	}
}
