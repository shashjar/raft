package main

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

const (
	MIN_ELECTION_TIMEOUT_MS = 150
	MAX_ELECTION_TIMEOUT_MS = 300
)

func (s *RaftServer) startOrResetElectionTimer() {
	s.electionTimerMutex.Lock()
	defer s.electionTimerMutex.Unlock()

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
	// An election timeout only matters if we're a follower or candidate
	if s.serverState == Leader {
		return
	}

	log.Printf("Server %d: Election timeout occurred", s.serverID)
	s.startElection()
}

// Conver to candidate and start election
func (s *RaftServer) startElection() {
	s.serverState = Candidate
	s.currentTerm++
	s.votedFor = s.serverID

	s.votesReceived = 1
	s.votesMutex = sync.Mutex{}

	s.startOrResetElectionTimer()

	s.campaignForElection()
}

// TODO: implement
/*
Send RequestVote RPCs to all other servers in the cluster.

If votes received from majority of servers: become leader.
*/
func (s *RaftServer) campaignForElection() {
	for serverID, serverAddr := range s.clusterAddrs {
		if serverID == s.serverID {
			continue
		}

		go s.sendRequestVote(serverAddr)
	}
}

// TODO: implement
func (s *RaftServer) promoteToLeader() {
	// Only a candidate can be promoted to leader
	if s.serverState != Candidate {
		return
	}

	// TODO: send initial empty AppendEntries RPCs to all other servers in the cluster; repeat during idle periods to prevent election timeouts
}
