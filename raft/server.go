package main

type ServerType int

const (
	Leader ServerType = iota
	Follower
	Candidate
)

type LogEntry struct {
	// TODO: command is just a string for now - add actual commands later
	command string // command for state machine
	term    int    // term when entry was received by leader
}

type ServerState struct {
	serverID     int
	clusterSize  int
	clusterAddrs map[int]ServerAddress

	serverType ServerType

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

func initializeServerState(serverID int, clusterMembers map[int]ServerAddress) *ServerState {
	clusterSize := len(clusterMembers)
	nextIndex := make([]int, clusterSize)
	matchIndex := make([]int, clusterSize)

	// Initialize nextIndex to last log index + 1 (which is 1 since log starts empty)
	for i := range nextIndex {
		nextIndex[i] = 1
	}

	return &ServerState{
		serverID:     serverID,
		clusterSize:  clusterSize,
		clusterAddrs: clusterMembers,

		serverType: Follower,

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
	state := initializeServerState(serverID, clusterMembers)

	// TODO: Start the Raft server with the initialized state
}
