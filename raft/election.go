package main

import (
	"math/rand"
	"time"
)

const (
	MIN_ELECTION_TIMEOUT_MS = 150
	MAX_ELECTION_TIMEOUT_MS = 300
	HEARTBEAT_TIMEOUT_MS    = 50
)

func (s *RaftServer) StartOrResetElectionTimer() {
	// Select a random election timeout
	s.electionTimeout = time.Duration(MIN_ELECTION_TIMEOUT_MS+rand.Intn(MAX_ELECTION_TIMEOUT_MS-MIN_ELECTION_TIMEOUT_MS+1)) * time.Millisecond

	// Stop existing timer if any
	if s.electionTimer != nil {
		s.electionTimer.Stop()
	}

	// Start new timer
	s.electionTimer = time.AfterFunc(s.electionTimeout, func() {
		s.onElectionTimeout()
	})
}

func (s *RaftServer) onElectionTimeout() {
	s.dlog("Election timeout occurred")

	s.mu.Lock()

	// An election timeout only matters if we're a follower or candidate
	if s.serverState == Leader {
		s.mu.Unlock()
		return
	}

	s.mu.Unlock()

	s.startElection()
}

// Conver to candidate and start election
func (s *RaftServer) startElection() {
	s.mu.Lock()

	s.serverState = Candidate
	s.currentTerm++
	s.votedFor = s.serverID

	s.votesReceived = 1

	s.StartOrResetElectionTimer()

	s.mu.Unlock()

	s.campaignForElection()
}

/*
Send RequestVote RPCs to all other servers in the cluster.

If votes received from majority of servers: become leader.
*/
func (s *RaftServer) campaignForElection() {
	for serverID, serverAddr := range s.clusterAddrs {
		if serverID == s.serverID {
			continue
		}

		go s.SendRequestVote(serverAddr)
	}
}

func (s *RaftServer) promoteToLeader() {
	// Only a candidate can be promoted to leader
	if s.serverState != Candidate {
		return
	}

	s.dlog("Promoting to leader")

	s.serverState = Leader

	nextIndex := make(map[int]int)
	matchIndex := make(map[int]int)
	for serverID := range s.clusterAddrs {
		if serverID == s.serverID {
			continue
		}
		nextIndex[serverID] = len(s.log)
		matchIndex[serverID] = -1
	}
	s.nextIndex = nextIndex
	s.matchIndex = matchIndex

	s.StartOrResetHeartbeatTimer()

	if s.electionTimer != nil {
		s.electionTimer.Stop()
	}
	s.electionTimer = nil

	// Send initial empty AppendEntries RPCs (heartbeat) to all other servers in the cluster
	s.onHeartbeatTimeout()
}
