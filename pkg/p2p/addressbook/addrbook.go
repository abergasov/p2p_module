package addressbook

import (
	"context"
	"fmt"
	"p2p_module/pkg/config"
	"path/filepath"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	peersFileName = "peers.json"
)

// AddrBook stores peer addresses.
type AddrBook struct {
	bookPath       string // place where peers.json is stored
	addrMU         sync.RWMutex
	addrIndex      map[peer.ID]*KnownAddress
	addrCandidates map[peer.ID]*peer.AddrInfo
	cancel         context.CancelFunc
	eg             errgroup.Group
	log            log.FieldLogger
}

// NewAddrBook creates new address book.
func NewAddrBook(log log.FieldLogger, cfg config.P2P) *AddrBook {
	ab := &AddrBook{
		bookPath:       filepath.Join(cfg.DataDir, peersFileName),
		addrIndex:      make(map[peer.ID]*KnownAddress),
		addrCandidates: make(map[peer.ID]*peer.AddrInfo),
		log:            log,
	}
	ab.loadPeers()
	ctx, cancel := context.WithCancel(context.Background())
	ab.cancel = cancel
	go ab.periodicallyPersistPeers(ctx)
	return ab
}

// Stop closes any background workers.
func (a *AddrBook) Stop() {
	a.log.Info("stopping address book")
	a.cancel()
	if err := a.eg.Wait(); err != nil {
		a.log.WithError(err).Error("failed to stop address book")
	}
}

// AddAddressCandidate adds address candidate to address book.
func (a *AddrBook) AddAddressCandidate(peerInfo peer.AddrInfo) {
	a.log.WithField("peer", peerInfo.ID).Info("adding address candidate")
	a.addrMU.Lock()
	defer a.addrMU.Unlock()
	if _, ok := a.addrIndex[peerInfo.ID]; ok {
		// address already in address book, it can happen if exchange is super fast
		a.addrIndex[peerInfo.ID].Addr = &peerInfo
	}
	a.addrCandidates[peerInfo.ID] = &peerInfo
}

// MarkAddressValid marks address as valid and store node's public URL.
func (a *AddrBook) MarkAddressValid(id peer.ID, peerPublicID string) {
	a.log.WithField("peer", id).Info("marking address as valid")
	a.addrMU.Lock()
	defer a.addrMU.Unlock()
	addr := a.addrCandidates[id]
	delete(a.addrCandidates, id)

	// add address to address book
	a.addrIndex[id] = &KnownAddress{
		Addr:        addr,
		Attempts:    1,
		NodeURL:     peerPublicID,
		LastAttempt: time.Now(),
		LastSeen:    time.Now(),
		LastSuccess: time.Now(),
	}
}

func (a *AddrBook) GetPeerAddress(id peer.ID) (*KnownAddress, error) {
	a.addrMU.RLock()
	defer a.addrMU.RUnlock()
	if _, ok := a.addrIndex[id]; !ok {
		return nil, fmt.Errorf("peer %s not found", id)
	}
	return a.addrIndex[id], nil
}

func (a *AddrBook) CountPeers() int {
	a.addrMU.RLock()
	defer a.addrMU.RUnlock()
	return len(a.addrIndex)
}

func (a *AddrBook) GetKnownPeers() []*KnownAddress {
	a.addrMU.RLock()
	defer a.addrMU.RUnlock()
	peers := make([]*KnownAddress, 0, len(a.addrIndex))
	for _, p := range a.addrIndex {
		if p.Addr == nil || len(p.Addr.Addrs) == 0 {
			continue
		}
		peers = append(peers, p)
	}
	return peers
}
