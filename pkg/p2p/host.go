package p2p

import (
	"context"
	"fmt"
	"p2p_module/pkg/config"
	"p2p_module/pkg/p2p/addressbook"
	"p2p_module/pkg/p2p/discovery"
	"p2p_module/pkg/p2p/handshake"
	"p2p_module/pkg/p2p/metrics"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	networkID = "cere-ddc" // used to identify network in handshake. later we can use this to identify network version.
)

// Host is a wrapper for libp2p host. Additional services for ddc are added to host.
type Host struct {
	log        log.FieldLogger
	conf       config.P2P
	host       host.Host
	handshaker *handshake.Handshake      // using to receive and send handshake messages and to get peer's public key
	discoverer *discovery.NodeDiscoverer // using to keep network nodes list in actual state
	addrBook   *addressbook.AddrBook     // vault of node's peers
	eg         errgroup.Group
	cancel     context.CancelFunc
}

// CreateHost initializes libp2p host configured for ddc.
func CreateHost(log log.FieldLogger, nodeURL string, cfg config.P2P) (*Host, error) {
	log.WithField("bootnode", cfg.BootNode).Info("starting libp2p host")
	cm, err := connmgr.NewConnManager(cfg.LowPeers, cfg.HighPeers, connmgr.WithGracePeriod(cfg.GracePeersShutdown))
	if err != nil {
		return nil, fmt.Errorf("p2p create conn mgr: %w", err)
	}
	ps, err := pstoremem.NewPeerstore()
	if err != nil {
		return nil, fmt.Errorf("can't create peer store: %w", err)
	}
	key, err := EnsureIdentity(cfg.DataDir)

	if err != nil {
		return nil, err
	}
	streamer := *yamux.DefaultTransport
	lopts := []libp2p.Option{
		libp2p.Identity(key),
		libp2p.ListenAddrStrings(cfg.Listen),
		libp2p.UserAgent("go-ddc-cdn-node"),
		libp2p.DisableRelay(),

		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Muxer("/yamux/1.0.0", &streamer),

		libp2p.ConnectionManager(cm),
		libp2p.Peerstore(ps),
		libp2p.BandwidthReporter(metrics.NewBandwidthCollector()),
	}
	h, err := libp2p.New(lopts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize libp2p host: %w", err)
	}
	h.Network().Notify(metrics.NewConnectionsMeeter())
	log.WithField("identity", h.ID().String()).WithField("address", cfg.Listen).Info("start p2p node with identity")

	// init additional services for host
	hs, err := handshake.NewHandshake(log.WithField("module", "handshake"), h, networkID, nodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize handshake: %w", err)
	}
	sub, err := h.EventBus().Subscribe(new(handshake.EventHandshakeComplete))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe for events: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	hst := &Host{
		log:        log,
		cancel:     cancel,
		conf:       cfg,
		host:       h,
		handshaker: hs,
		discoverer: discovery.NewNodeDiscoverer(log.WithField("module", "discover"), cfg.BootNode),
		addrBook:   addressbook.NewAddrBook(log.WithField("module", "address_book"), cfg),
	}
	go hst.handleHandshakes(ctx, sub)
	go hst.Start(ctx)
	return hst, nil
}

// Start starts host services.
func (h *Host) Start(ctx context.Context) {
	h.log.Info("starting p2p bootnode detection")
	for peers := range h.discoverer.Start() {
		for _, peer := range peers {
			if peer.ID == h.host.ID() {
				continue // no need to connect to self
			}
			h.addrBook.AddAddressCandidate(peer)
			if err := h.host.Connect(ctx, peer); err != nil {
				h.log.WithError(err).WithField("peer", peer.ID).Warning("failed to connect to peer")
			}
		}
	}
}

// Stop stops host services.
func (h *Host) Stop() {
	h.log.Info("stopping p2p node")
	h.cancel()
	h.addrBook.Stop()
	h.discoverer.Stop()
	h.handshaker.Stop()
	if err := h.host.Close(); err != nil {
		h.log.WithError(err).Error("failed to close host")
	}
	if err := h.eg.Wait(); err != nil {
		h.log.WithError(err).Error("failed to wait for goroutines")
	}
}

// handleHandshakes handles handshake events.
func (h *Host) handleHandshakes(ctx context.Context, sub event.Subscription) {
	h.eg.Go(func() error {
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				return nil
			case evt := <-sub.Out():
				handshakeEvent, ok := evt.(handshake.EventHandshakeComplete)
				if !ok {
					h.log.WithField("event", evt).Error("unexpected event type")
					continue
				}
				h.log.WithField("peer", handshakeEvent.PID).WithField("nodeURL", handshakeEvent.NodeURL).Info("handshake complete")
				h.addrBook.MarkAddressValid(handshakeEvent.PID, handshakeEvent.NodeURL)
				h.addrBook.PersistPeers()
			}
		}
	})
}

// GetHostID returns host id.
func (h *Host) GetHostID() string {
	return h.host.ID().String()
}

func (h *Host) GetAddressBook() *addressbook.AddrBook {
	return h.addrBook
}

func (h *Host) GetHost() host.Host {
	return h.host
}
