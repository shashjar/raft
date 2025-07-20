package main

import (
	"log"
	"net/http"
	"strconv"
)

type ServerState int

const (
	Leader ServerState = iota
	Follower
	Candidate
)

type LogEntry struct {
	// TODO: command is just a string for now - add actual commands later
	command string // command for state machine
	term    int    // term when entry was received by leader
}

type RaftServer struct {
	serverID     int
	clusterSize  int
	clusterAddrs map[int]ServerAddress

	serverState ServerState

	// TODO: this state needs to be persisted to disk
	// Persistent state on all servers (updated on stable storage before responding to RPCs)
	currentTerm int        // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	votedFor    int        // candidateId that received vote in current term (-1 if none)
	log         []LogEntry // log entries

	// Volatile state on all servers
	commitIndex int // index of highest log entry known to be committed (initialized to 0, increases monotonically)
	lastApplied int // index of highest log entry applied to state machine (initialized to 0, increases monotonically)

	// Volatile state on leaders (reinitialized after election)
	nextIndex  []int // for each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	matchIndex []int // for each server, index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)
}

func initializeRaftServer(serverID int, clusterMembers map[int]ServerAddress) *RaftServer {
	clusterSize := len(clusterMembers)
	nextIndex := make([]int, clusterSize)
	matchIndex := make([]int, clusterSize)

	// Initialize nextIndex to last log index + 1 (which is 1 since log starts empty)
	for i := range nextIndex {
		nextIndex[i] = 1
	}

	return &RaftServer{
		serverID:     serverID,
		clusterSize:  clusterSize,
		clusterAddrs: clusterMembers,

		serverState: Follower,

		currentTerm: 0,
		votedFor:    -1,
		log:         []LogEntry{},

		commitIndex: 0,
		lastApplied: 0,

		nextIndex:  nextIndex,
		matchIndex: matchIndex,
	}
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

Invoked by leader to replicate log entries; also used as heartbeat

Receiver implementation:
1. Reply false if term < currentTerm
2. Reply false if log doesn't contain an entry at prevLogIndex whose term matches prevLogTerm
3. If an existing entry conflicts with a new one (same index but different terms), delete the existing entry and all that follow it
4. Append any new entries not already in the log
5. If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
*/
func (s *RaftServer) handleAppendEntries(w http.ResponseWriter, r *http.Request) {
	log.Println("AppendEntries RPC received")

	var args AppendEntriesArgs
	if err := parseRequestBody(r, &args); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	results := s.executeAppendEntries(args)
	sendJSONResponse(w, results)
}

// TODO: implement
func (s *RaftServer) executeAppendEntries(args AppendEntriesArgs) AppendEntriesResults {
	results := AppendEntriesResults{Term: s.currentTerm, Success: false}
	return results
}

type RequestVoteArgs struct {
	Term         int `json:"term"`         // candidate's term
	CandidateID  int `json:"candidateId"`  // candidate requesting vote
	LastLogIndex int `json:"lastLogIndex"` // index of candidate's last log entry
	LastLogTerm  int `json:"lastLogTerm"`  // term of candidate's last log entry
}

type RequestVoteResults struct {
	Term        int  `json:"term"`        // currentTerm, for candidate to update itself
	VoteGranted bool `json:"voteGranted"` // true means candidate received vote
}

/*
RequestVote RPC handler

# Invoked by candidates to gather votes

Receiver implementation:
1. Reply false if term < currentTerm
2. If votedFor is null or candidateId, and candidate's log, and candidate's log is at least as up-to-date as receiver's log, grant vote
*/
func (s *RaftServer) handleRequestVote(w http.ResponseWriter, r *http.Request) {
	log.Println("RequestVote RPC received")

	var args RequestVoteArgs
	if err := parseRequestBody(r, &args); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	results := s.executeRequestVote(args)
	sendJSONResponse(w, results)
}

// TODO: implement
func (s *RaftServer) executeRequestVote(args RequestVoteArgs) RequestVoteResults {
	results := RequestVoteResults{Term: s.currentTerm, VoteGranted: false}
	return results
}
