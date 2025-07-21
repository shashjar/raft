package main

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type ServerState int

const (
	Leader ServerState = iota
	Follower
	Candidate
)

type RaftServer struct {
	serverID     int
	clusterSize  int
	clusterAddrs map[int]ServerAddress

	serverState ServerState

	// TODO: this state needs to be persisted to disk before responding to RPCs
	// Persistent state on all servers (updated on stable storage before responding to RPCs)
	currentTerm int        // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	votedFor    int        // candidateId that received vote in current term (-1 if none)
	log         []LogEntry // log entries (zero-indexed)

	// TODO: commit entries to state machine once possible
	// Volatile state on all servers
	commitIndex int // index of highest log entry known to be committed (initialized to 0, increases monotonically)
	lastApplied int // index of highest log entry applied to state machine (initialized to 0, increases monotonically)

	// Volatile state on candidates (reinitialized at the start of each election)
	votesReceived int        // counter of votes that the candidate has received in the current election
	votesMutex    sync.Mutex // mutex to protect votesReceived

	// Volatile state on leaders (reinitialized after election)
	nextIndex  []int // for each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	matchIndex []int // for each server, index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)

	// State for election timer
	electionTimeout    time.Duration // duration of the election timeout
	electionTimer      *time.Timer   // timer for the election timeout
	electionTimerMutex sync.Mutex    // mutex to protect the election timer
}

func initializeRaftServer(serverID int, clusterMembers map[int]ServerAddress) *RaftServer {
	clusterSize := len(clusterMembers)
	nextIndex := make([]int, clusterSize)
	matchIndex := make([]int, clusterSize)

	// Initialize nextIndex to last log index + 1 (which is 1 since log starts empty)
	for i := range nextIndex {
		nextIndex[i] = 1
	}

	server := &RaftServer{
		serverID:     serverID,
		clusterSize:  clusterSize,
		clusterAddrs: clusterMembers,

		serverState: Follower,

		currentTerm: 0,
		votedFor:    -1,
		log:         []LogEntry{},

		commitIndex: 0,
		lastApplied: 0,

		votesReceived: 0,
		votesMutex:    sync.Mutex{},

		nextIndex:  nextIndex,
		matchIndex: matchIndex,

		electionTimeout:    0,
		electionTimer:      nil,
		electionTimerMutex: sync.Mutex{},
	}

	server.startOrResetElectionTimer()

	return server
}

func startServer(serverID int, clusterMembers map[int]ServerAddress) {
	// Create a Raft server instance with some initial state
	server := initializeRaftServer(serverID, clusterMembers)

	// Register HTTP handlers for Raft RPCs
	http.HandleFunc("/appendEntries", server.handleAppendEntries)
	http.HandleFunc("/requestVote", server.handleRequestVote)

	// Bring up the server to begin listening for RPCs
	addr := clusterMembers[serverID]
	log.Println("Raft server listening on port", addr.port)
	err := http.ListenAndServe(addr.host+":"+strconv.Itoa(addr.port), nil)
	if err != nil {
		log.Fatalf("Failed to start Raft server: %v", err)
	}
}
