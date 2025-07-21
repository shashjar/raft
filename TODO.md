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
  - [ ] Implement the election timeout
  - [ ] Extend the algorithm logic to account for differences in behavior between followers, candidates, and leaders
- [ ] Make sure that persistent state on each server is persisted to stable storage before responding to RPCs
- [ ] Add an endpoint to the server for clients to make requests on

## Verification

- [ ] Implement a simulator to verify the correctness of the Raft implementation

## Cleanup

- [ ] Add sufficient documentation/comments throughout code
- [ ] Write `README.md`
