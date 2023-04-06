p2p module example

### Disclaimer
This is a simple example of how to use the p2p module.

### Description
Initially all peers of network are wellknown and stored in smart contract. 

As entry point for p2p need bootnode. It can be reading of smart contract or some url which cache serves list of peers.

For simplify - bootnode requesting with some period of time from some mocked url.

Main package which is responsible for p2p module is `pkg/p2p`. It contains:
* `p2p` - main struct which is responsible for p2p module.
* `p2p/metrics` - prometheus metrics for p2p module.
* `p2p/handshage` - handshake protocol for p2p module, for backward compatibility with old nodes. For simplify - it is just verify network id
* `p2p/addresbook` - address book for p2p module with periodic disk persistence
* `p2p/discovery` - periodic discovery of new peers. For simplify - it is just request list of peers from bootnode. Later can be replaced to smart contract event subscription on state change.
* `p2p/server` - package for handle custom business logic messages exchanges on top of p2p network.

Custom logic over the p2p module can be found in `pkg/fetcher`:
* measure messages propagation time
* get state of the network. <- this allows for every node to be as bootnode, and share information about network with other nodes.

### Run
Tests are spawn multiply nodes and check that they are connected to each other. After all check state of every node and verify that it match expected.
```bash
make test-quick
```

Tests are check measure propagation time of the message and validate each node has the same state.