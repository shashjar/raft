# Raft

[![Go 1.23](https://img.shields.io/badge/go-1.23-9cf.svg)](https://golang.org/dl/)

An implementation of the [Raft consensus algorithm](https://raft.github.io/raft.pdf) in Go, featuring leader election, log replication, and persistence.

## Overview

This project implements the Raft consensus algorithm as described in the original paper by Diego Ongaro and John Ousterhout. Raft is a consensus algorithm designed to be understandable and equivalent to Paxos, but more understandable. It achieves consensus through leader election, log replication, and safety mechanisms.

## Features

- **Leader Election**: Automatic leader election with randomized timeouts, implemented using the `RequestVote` RPC
- **Log Replication**: `AppendEntries` RPC for log replication and heartbeats
- **Persistence**: Persistent state storage to disk for crash recovery
- **HTTP API**: `RESTful` endpoints for Raft operations and client commands
- **Client Redirection**: Automatic redirection to current leader

## Architecture

This implementation consists of several key components:

- **`main.go`**: The entrypoint for running a single Raft server as part of a cluster
- **`server.go`**: Core Raft server implementation with state management
- **`persistence.go`**: Persistent state storage and recovery
- **`http.go`**: HTTP client and server utilities
- **`election.go`**: Leader election logic and vote handling
- **`request_vote.go`**: Logic for sending and handling `RequestVote` RPCs (election operation)
- **`append_entries.go`**: Logic for sending and handling `AppendEntries` RPCs (log replication)
- **`heartbeat.go`**: Periodic heartbeat management
- **`log.go`**: Log entry management and command processing

## References

- [Raft Paper](https://raft.github.io/raft.pdf)
- [Raft Website](https://raft.github.io/)
