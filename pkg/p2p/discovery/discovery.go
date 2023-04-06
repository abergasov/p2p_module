package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

const (
	updateInterval = 10 * time.Second
)

// NodeDiscoverer loads peers from boot node. At first stage - simple http request to boot node.
// It updates state of peers with some interval.
// Later it will migrate to smartContract state loading with events subscription.
type NodeDiscoverer struct {
	nodeURL       string
	updateTimeout *time.Ticker
	lastPeers     []string
	peers         chan []peer.AddrInfo

	shutdownCtx context.Context
	cancel      context.CancelFunc
	log         log.FieldLogger
}

// NewNodeDiscoverer creates new instance of NodeDiscoverer.
func NewNodeDiscoverer(log log.FieldLogger, nodeURL string) *NodeDiscoverer {
	shutdownCtx, cancel := context.WithCancel(context.Background())
	return &NodeDiscoverer{
		nodeURL:       nodeURL,
		updateTimeout: time.NewTicker(updateInterval),
		cancel:        cancel,
		shutdownCtx:   shutdownCtx,
		peers:         make(chan []peer.AddrInfo, 1),
		lastPeers:     make([]string, 0),
		log:           log,
	}
}

// Start starts discovery.
func (b *NodeDiscoverer) Start() <-chan []peer.AddrInfo {
	go b.observeChanges()
	return b.peers
}

// Stop stops discovery.
func (b *NodeDiscoverer) Stop() {
	b.log.Info("stopping discovery")
	b.updateTimeout.Stop()
	b.cancel()
}

// observeChanges observes changes in peers state and sends them to peers channel.
func (b *NodeDiscoverer) observeChanges() {
	b.loadPeers()
	b.log.Info("starting fetch peers state")
	for {
		select {
		case <-b.updateTimeout.C:
			b.loadPeers()
		case <-b.shutdownCtx.Done():
			return
		}
	}
}

// loadPeers loads peers from boot node and sends them to peers channel.
func (b *NodeDiscoverer) loadPeers() {
	peers, err := b.loadState()
	if err != nil {
		b.log.WithError(err).Error("failed to load peers from boot node")
		return
	}
	if !b.checkPeersChanged(peers) {
		return // peers are not changed since last observation, skip rest of the logic
	}
	peersAddr := make([]peer.AddrInfo, 0, len(peers))
	for _, rawPeer := range peers {
		ma, err := multiaddr.NewMultiaddr(rawPeer)
		if err != nil {
			b.log.WithError(err).Error("failed to parse peer address")
			continue
		}
		addresses, err := peer.AddrInfosFromP2pAddrs(ma)
		if err != nil {
			b.log.WithError(err).Error("failed to parse peer address")
			continue
		}
		peersAddr = append(peersAddr, addresses...)
	}
	rand.Shuffle(len(peersAddr), func(i, j int) {
		peersAddr[i], peersAddr[j] = peersAddr[j], peersAddr[i]
	})
	b.peers <- peersAddr
}

// checkPeersChanged checks if peers are changed since last observation.
func (b *NodeDiscoverer) checkPeersChanged(peers []string) bool {
	if len(b.lastPeers) != len(peers) {
		b.lastPeers = peers
		return true
	}
	res, _ := lo.Difference[string](b.lastPeers, peers)
	return len(res) > 0
}

// loadState loads peers from boot node.
func (b *NodeDiscoverer) loadState() ([]string, error) {
	client := http.DefaultClient
	client.Timeout = 5 * time.Second
	res, err := client.Get(b.nodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get peers from boot node: %w", err)
	}
	defer res.Body.Close()
	var result []string
	if err = json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode peers from boot node: %w", err)
	}
	return result, nil
}
