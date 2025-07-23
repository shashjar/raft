# TODO

## Preparation

- [x] Read and take notes on the Raft white paper
- [x] Set up Go codebase
- [x] Plan out structure of Raft implementation

## Implementation

### Setup

- [x] Implement program entrypoint for starting up a single server given the overall Raft cluster information
- [x] Implement bringing up a single server on HTTP

### Basic RPC Logic

- [x] Implement the HTTP handlers for the AppendEntries and RequestVote RPCs
- [ ] Implement the core Raft algorithm
  - [x] Actual execution of the RequestVote RPC
  - [x] Actual execution of the AppendEntries RPC
  - [x] Implement basic logic for the election timeout

### Elections

- [x] Implement the sending of the RequestVote RPC to all other servers in the cluster as part of the election start process (`campaignForElection`)
- [x] Implement promotion of a candidate to leader once the vote results are in (`promoteToLeader`)
  - [x] The leader needs to send initial heartbeat messages to all other servers in the cluster as soon as it is elected
  - [x] The leader also needs to send a regular heartbeat during idle periods

### Some Cleanup

- [x] Update the mutex on the `RaftServer` struct to protect all writes/reads of server state
- [x] Improve debug logging

### Commands Received from Clients

- [x] Add an endpoint to the server for clients to submit commands on
- [x] Implement sending of actual AppendEntries RPCs (not just heartbeats) by the leader in order to replicate log entries
- [x] Implement determination of when individual commands are committed (replicated across a majority of the cluster)
- [x] Apply commands to the state machine once they are committed (can be something basic like just a debug log for now)

### Persistence

- [x] Make sure that persistent state on each server is persisted to stable storage before responding to RPCs
- [x] On startup, load the persistent state from disk if it's available

## Verification/Testing

- [ ] Write a test harness & unit tests to verify the correctness of individual components of the Raft implementation
- [ ] Implement a simulator to verify the correctness of the Raft implementation
  - [ ] In simulator, it would be worth delaying and/or dropping some messages randomly (and regular messages should be simulated with a small delay in transmission)

## Cleanup

- [ ] Add sufficient documentation/comments throughout code
- [ ] Write `README.md`
