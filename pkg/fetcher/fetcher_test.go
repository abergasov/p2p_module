package fetcher_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testing that the fetcher can broadcast and handle an echo measure
func TestFetcher_NodesCollectState(t *testing.T) {
	// given
	bN := spawnBootNode(t)
	nodes := make([]*testFetcherNode, 0, p2pNodesCount)
	for i := 0; i < p2pNodesCount; i++ {
		fetcherNodeContainer := spawnFetcherNode(t, bN.bootNodeAddress, i)
		nodes = append(nodes, fetcherNodeContainer)
		bN.peersList = append(bN.peersList, fetcherNodeContainer.p2pAddress)
	}

	// when
	require.Eventually(t, func() bool {
		for _, node := range nodes {
			if len(node.fetcherNode.ServeKnownPeers()) != p2pNodesCount-1 {
				return false
			}
		}
		return true
	}, 20*time.Second, 2*time.Second, "Nodes should have all peers in address book")

	// then that Nodes collect state from each other
	require.Eventually(t, func() bool {
		// each node should have state from all other nodes
		for _, node := range nodes {
			peersState := node.fetcherNode.GetPeerState()
			for _, otherNode := range nodes {
				if node.fetcherNode.GetNodeID() == otherNode.fetcherNode.GetNodeID() {
					continue // skip self
				}
				eventsLog := otherNode.evt.GetLog()
				if len(eventsLog) == 0 {
					// node does not have any events, it means nobody request it state
					return false
				}
				if _, ok := peersState[otherNode.fetcherNode.GetNodeID()]; !ok {
					// node does not have state from other node
					return false
				}
				found := false
				for _, log := range peersState[otherNode.fetcherNode.GetNodeID()] {
					if contains(eventsLog, log) {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
		}
		return true // all check are passed
	}, 15*time.Second, 100*time.Millisecond, "Nodes should have state of each node")
}

func contains(src []string, str string) bool {
	for _, s := range src {
		if s == str {
			return true
		}
	}
	return false
}
