package main

import (
	"fmt"
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

const (
	INITIAL_TERM  = 0
	INITIAL_INDEX = -1
)

const (
	APPEND_ENTRIES_PATH = "/appendEntries"
	REQUEST_VOTE_PATH   = "/requestVote"
	SUBMIT_COMMAND_PATH = "/submitCommand"
)

const DEBUG = true

type RaftServer struct {
	mu sync.Mutex // mutex to protect concurrent access to the server's state

	serverID     int                   // this server's unique (across the cluster) ID
	clusterSize  int                   // number of total servers in the cluster
	clusterAddrs map[int]ServerAddress // map of server ID to server address

	serverState ServerState // current state of this server

	// TODO: this state needs to be persisted to disk before responding to RPCs
	// Persistent state on all servers (updated on stable storage before responding to RPCs)
	currentTerm int        // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	votedFor    int        // candidateId that received vote in current term (-1 if none)
	log         []LogEntry // log entries (zero-indexed)

	// TODO: commit entries to state machine once possible
	// Volatile state on all servers
	commitIndex int // index of highest log entry known to be committed (initialized to -1, increases monotonically)
	lastApplied int // index of highest log entry applied to state machine (initialized to -1, increases monotonically)

	// Volatile state on candidates (reinitialized at the start of each election)
	votesReceived int // counter of votes that the candidate has received in the current election

	// Volatile state on leaders (reinitialized after promotion to leader)
	nextIndex       map[int]int       // map of server ID to index of the next log entry to send to that server (initialized to leader last log index + 1, a.k.a. the length of the leader's log)
	matchIndex      map[int]int       // map of server ID to index of highest log entry known to be replicated on server (initialized to -1, increases monotonically)
	pendingCommands map[int]chan bool // map of log index to channel for signaling when entry is committed
	heartbeatTimer  *time.Timer       // timer for the heartbeat timeout

	// State for election timer
	electionTimeout time.Duration // duration of the election timeout
	electionTimer   *time.Timer   // timer for the election timeout
}

func InitializeRaftServer(serverID int, clusterMembers map[int]ServerAddress) *RaftServer {
	clusterSize := len(clusterMembers)

	server := &RaftServer{
		mu: sync.Mutex{},

		serverID:     serverID,
		clusterSize:  clusterSize,
		clusterAddrs: clusterMembers,

		serverState: Follower,

		currentTerm: INITIAL_TERM,
		votedFor:    -1,
		log:         []LogEntry{},

		commitIndex: INITIAL_INDEX,
		lastApplied: INITIAL_INDEX,

		votesReceived: 0,

		nextIndex:       nil,
		matchIndex:      nil,
		pendingCommands: nil,
		heartbeatTimer:  nil,

		electionTimeout: 0,
		electionTimer:   nil,
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	server.StartOrResetElectionTimer()

	return server
}

func StartServer(serverID int, clusterMembers map[int]ServerAddress) {
	// Create a Raft server instance with some initial state
	server := InitializeRaftServer(serverID, clusterMembers)

	// Register HTTP handlers for Raft RPCs
	http.HandleFunc(APPEND_ENTRIES_PATH, server.HandleAppendEntries)
	http.HandleFunc(REQUEST_VOTE_PATH, server.HandleRequestVote)

	// Register HTTP handler for client commands
	http.HandleFunc(SUBMIT_COMMAND_PATH, server.HandleSubmitCommand)

	// Bring up the server to begin listening for RPCs
	addr := clusterMembers[serverID]
	server.dlog("Raft server listening on port %d", addr.port)
	err := http.ListenAndServe(addr.host+":"+strconv.Itoa(addr.port), nil)
	if err != nil {
		server.dlog("Failed to begin serving Raft server: %v", err)
		panic(err)
	}
}

func (s *RaftServer) convertToFollower(newTerm int) {
	s.dlog("Converting to follower with new term %d", newTerm)

	s.serverState = Follower

	s.currentTerm = newTerm
	s.votedFor = -1

	s.nextIndex = nil
	s.matchIndex = nil

	s.pendingCommands = nil

	if s.heartbeatTimer != nil {
		s.heartbeatTimer.Stop()
	}
	s.heartbeatTimer = nil

	s.StartOrResetElectionTimer()
}

func (s *RaftServer) dlog(format string, args ...interface{}) {
	if DEBUG {
		format = fmt.Sprintf("[%d] ", s.serverID) + format
		log.Printf(format, args...)
	}
}
