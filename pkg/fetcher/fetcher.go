package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"p2p_module/pkg/p2p"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	updateInterval = 10 * time.Second
)

type Measurer struct {
	ID    string
	Nodes []peer.ID
	Start time.Time
}

// Fetcher is a struct that fetches data from other Nodes in p2p network.
type Fetcher struct {
	echoMeasure   Measurer
	echoMeasureMU sync.Mutex
	echoInterval  *time.Ticker
	updateTimeout *time.Ticker
	log           log.FieldLogger
	host          *p2p.Host
	cancel        context.CancelFunc
	eg            errgroup.Group
	peerStateMu   sync.Mutex
	peerState     map[string][]string
}

func NewFetcher(logger log.FieldLogger, host *p2p.Host) (*Fetcher, error) {
	ctx, cancel := context.WithCancel(context.Background())
	f := &Fetcher{
		log:           logger,
		host:          host,
		cancel:        cancel,
		updateTimeout: time.NewTicker(updateInterval),
		echoInterval:  time.NewTicker(echoInterval),
		peerState:     make(map[string][]string),
	}
	ps, err := pubsub.NewGossipSub(ctx, f.host.GetHost())
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}
	topic, err := ps.Join(echoProtocol)
	if err != nil {
		return nil, fmt.Errorf("failed to join topic: %w", err)
	}
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe topic: %w", err)
	}
	go f.start(ctx)
	go f.startEcho(ctx, topic)
	go f.loop(ctx, sub)
	return f, nil
}

func (f *Fetcher) start(ctx context.Context) {
	f.eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-f.updateTimeout.C:
				f.fetchNodesState()
			}
		}
	})
}

func (f *Fetcher) Stop() {
	f.log.Info("stopping fetcher")
	f.updateTimeout.Stop()
	f.cancel()
	if err := f.eg.Wait(); err != nil {
		f.log.WithError(err).Error("error while stopping fetcher")
	}
}

func (f *Fetcher) fetchNodesState() {
	f.log.Info("fetching Nodes state")
	addrBook := f.host.GetAddressBook()
	peers := f.host.GetHost().Network().Peers()
	for _, pr := range peers {
		ka, err := addrBook.GetPeerAddress(pr)
		if err != nil {
			f.log.WithError(err).Error("failed to get peer address")
			continue
		}
		res, err := f.fetchNodeState(ka.NodeURL)
		if err != nil {
			f.log.WithError(err).Error("failed to fetch node state")
			continue
		}

		f.peerStateMu.Lock()
		if _, ok := f.peerState[pr.String()]; !ok {
			f.peerState[pr.String()] = make([]string, 0)
		}
		f.peerState[pr.String()] = append(f.peerState[pr.String()], res)
		f.peerStateMu.Unlock()
	}
	f.log.Info("fetching Nodes state finished")
}

func (f *Fetcher) ServeKnownPeers() []string {
	addresses := f.host.GetAddressBook().GetKnownPeers()
	result := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		result = append(result, fmt.Sprintf("%s/p2p/%s", addr.Addr.Addrs[0].String(), addr.Addr.ID.String()))
	}
	return result
}

func (f *Fetcher) fetchNodeState(url string) (string, error) {
	client := http.DefaultClient
	client.Timeout = 5 * time.Second
	res, err := client.Get(url + "/stat")
	if err != nil {
		return "", fmt.Errorf("failed to get node state: %w", err)
	}
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	return string(resBody), nil
}

func (f *Fetcher) GetNodeID() string {
	return f.host.GetHost().ID().String()
}

func (f *Fetcher) GetPeerState() map[string][]string {
	f.peerStateMu.Lock()
	defer f.peerStateMu.Unlock()
	return f.peerState
}
