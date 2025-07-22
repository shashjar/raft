# TODO

## Preparation

- [x] Read and take notes on the Raft white paper
- [x] Set up Go codebase
- [x] Plan out structure of Raft implementation

## Implementation

- [x] Implement program entrypoint for starting up a single server given the overall Raft cluster information
- [x] Implement bringing up a single server on HTTP
- [x] Implement the HTTP handlers for the AppendEntries and RequestVote RPCs
- [ ] Implement the core Raft algorithm
  - [x] Actual execution of the RequestVote RPC
  - [x] Actual execution of the AppendEntries RPC
  - [x] Implement basic logic for the election timeout
- [x] Implement the sending of the RequestVote RPC to all other servers in the cluster as part of the election start process (`campaignForElection`)
- [x] Implement promotion of a candidate to leader once the vote results are in (`promoteToLeader`)
  - [x] The leader needs to send initial heartbeat messages to all other servers in the cluster as soon as it is elected
  - [x] The leader also needs to send a regular heartbeat during idle periods
- [x] Update the mutex on the `RaftServer` struct to protect all writes/reads of server state
- [ ] Add an endpoint to the server for clients to make requests on
- [ ] Implement sending of actual AppendEntries RPCs (not just heartbeats) by the leader
- [ ] Make sure that persistent state on each server is persisted to stable storage before responding to RPCs

## Verification/Testing

- [ ] Implement a simulator to verify the correctness of the Raft implementation

## Cleanup

- [ ] Add sufficient documentation/comments throughout code
- [ ] Write `README.md`
