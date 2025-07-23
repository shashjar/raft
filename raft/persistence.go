package main

import (
	"encoding/gob"
	"os"
)

const STABLE_STORAGE_FILENAME = "raft_state.gob"

type PersistentState struct {
	currentTerm int
	votedFor    int
	log         []LogEntry
}

func (s *RaftServer) persistToStableStorage() error {
	s.mu.Lock()
	state := PersistentState{
		currentTerm: s.currentTerm,
		votedFor:    s.votedFor,
		log:         append([]LogEntry(nil), s.log...),
	}
	s.mu.Unlock()

	tmpFilename := STABLE_STORAGE_FILENAME + ".tmp"
	file, err := os.Create(tmpFilename)
	if err != nil {
		return err
	}

	// Clean up on any early exit
	defer func() {
		file.Close()
		_ = os.Remove(tmpFilename)
	}()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(state); err != nil {
		return err
	}

	if err := file.Sync(); err != nil {
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpFilename, STABLE_STORAGE_FILENAME); err != nil {
		return err
	}

	return nil
}

func loadFromStableStorage() (*PersistentState, error) {
	file, err := os.Open(STABLE_STORAGE_FILENAME)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	var state PersistentState
	err = decoder.Decode(&state)
	return &state, err
}
