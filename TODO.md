# TODO

## Preparation

- [x] Read and take notes on the Raft white paper
- [x] Set up Go codebase
- [x] Plan out structure of Raft implementation

## Implementation

- [x] Implement program entrypoint for starting up a single server given the overall Raft cluster information
- [ ] Implement bringing up a single server on HTTP, with handlers for the AppendEntries and RequestVote RPCs
- [ ] Implement election timer, heartbeat timer, log replication, and other basic components of the Raft algorithm

## Verification

- [ ] Implement a simulator to verify the correctness of the Raft implementation

## Cleanup

- [ ] Add sufficient documentation/comments throughout code
- [ ] Write `README.md`
