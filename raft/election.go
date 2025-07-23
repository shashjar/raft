package main

import (
	"math/rand"
	"time"
)

const (
	MIN_ELECTION_TIMEOUT_MS = 150
	MAX_ELECTION_TIMEOUT_MS = 300
)

func (s *RaftServer) StartOrResetElectionTimer() {
	termAtElectionTimerStart := s.currentTerm

	// Select a random election timeout
	s.electionTimeout = time.Duration(MIN_ELECTION_TIMEOUT_MS+rand.Intn(MAX_ELECTION_TIMEOUT_MS-MIN_ELECTION_TIMEOUT_MS+1)) * time.Millisecond

	// Stop existing timer if any
	if s.electionTimer != nil {
		s.electionTimer.Stop()
	}

	// Start new timer
	s.electionTimer = time.AfterFunc(s.electionTimeout, func() {
		s.onElectionTimeout(termAtElectionTimerStart)
	})

	s.dlog("Election timer started (%dms), term=%d", s.electionTimeout/time.Millisecond, termAtElectionTimerStart)
}

func (s *RaftServer) onElectionTimeout(termAtElectionTimerStart int) {
	s.dlog("Election timeout occurred")

	s.mu.Lock()

	// An election timeout only matters if we're a follower or candidate
	if s.serverState == Leader {
		s.mu.Unlock()
		return
	}

	if termAtElectionTimerStart != s.currentTerm {
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
	electionTerm := s.currentTerm
	s.votedFor = s.serverID

	s.votesReceived = 1

	s.StartOrResetElectionTimer()

	s.mu.Unlock()

	s.campaignForElection(electionTerm)
}

/*
Send RequestVote RPCs to all other servers in the cluster.

If votes received from majority of servers: become leader.
*/
func (s *RaftServer) campaignForElection(electionTerm int) {
	for serverID, serverAddr := range s.clusterAddrs {
		if serverID == s.serverID {
			continue
		}

		go s.SendRequestVote(serverAddr, electionTerm)
	}
}

func (s *RaftServer) promoteToLeader() {
	// Only a candidate can be promoted to leader
	if s.serverState != Candidate {
		return
	}

	s.dlog("Promoting to leader in term %d", s.currentTerm)

	s.serverState = Leader

	nextIndex := make(map[int]int)
	matchIndex := make(map[int]int)
	for serverID := range s.clusterAddrs {
		if serverID == s.serverID {
			continue
		}
		nextIndex[serverID] = len(s.log)
		matchIndex[serverID] = INITIAL_INDEX
	}
	s.nextIndex = nextIndex
	s.matchIndex = matchIndex

	s.pendingCommands = make(map[int]chan bool)

	if s.electionTimer != nil {
		s.electionTimer.Stop()
	}
	s.electionTimer = nil

	// Send initial empty AppendEntries RPCs (heartbeat) to all other servers in the cluster, and then continue regular heartbeats after that
	s.onHeartbeatTimeout()
}
