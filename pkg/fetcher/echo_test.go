package fetcher_test

import (
	"p2p_module/pkg/fetcher"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	p2pNodesCount = 4
)

// testing that the fetcher can broadcast and handle an echo measure
func TestFetcher_NetworkEchoMeasure(t *testing.T) {
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

	// then check that Nodes measure each other
	measuresHistory := make(map[string][]fetcher.Measurer)
	require.Eventually(t, func() bool {
		for _, node := range nodes {
			measure := node.fetcherNode.ServeMeasureState()
			if len(measure.Nodes) != p2pNodesCount-1 {
				continue
			}
			// we have complete network measure from node
			if _, ok := measuresHistory[node.fetcherNode.GetNodeID()]; !ok {
				measuresHistory[node.fetcherNode.GetNodeID()] = make([]fetcher.Measurer, 0)
			}
			measuresHistory[node.fetcherNode.GetNodeID()] = append(measuresHistory[node.fetcherNode.GetNodeID()], measure)
		}
		// check that all nodes have completed measure
		return len(measuresHistory) == p2pNodesCount
	}, 10*time.Second, 100*time.Millisecond, "All nodes should have completed measures")
}
