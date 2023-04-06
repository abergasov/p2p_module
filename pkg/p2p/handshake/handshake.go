package handshake

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	// handshakeProtocol is the protocol used for handshake between nodes.
	handshakeProtocol = "/cere/handshake/1.0.0"
	streamTimeout     = 10 * time.Second
)

// Handshake encapsulates handshake protocol.
// It is used to exchange information between nodes during connection establishment.
type Handshake struct {
	log       log.FieldLogger
	networkID string // used to identify network in handshake. later we can use this to identify network version.
	nodeURL   string // used to provide node's url to other nodes over handshake
	cancel    context.CancelFunc
	emitter   event.Emitter
	h         host.Host
	eg        errgroup.Group
}

// NewHandshake creates new handshake protocol.
func NewHandshake(log log.FieldLogger, h host.Host, networkID, nodeURL string) (*Handshake, error) {
	emitter, err := h.EventBus().Emitter(new(EventHandshakeComplete))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize emitter for handshake: %w", err)
	}
	sub, err := h.EventBus().Subscribe(new(event.EvtPeerIdentificationCompleted))
	if err != nil {
		return nil, fmt.Errorf("failed to subsribe for events: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	hs := &Handshake{
		networkID: networkID,
		nodeURL:   nodeURL,
		emitter:   emitter,
		h:         h,
		cancel:    cancel,
		log:       log,
	}

	h.SetStreamHandler(handshakeProtocol, hs.handler)
	hs.start(ctx, sub)
	return hs, nil
}

// Stop closes any background workers.
func (h *Handshake) Stop() {
	h.log.Info("stopping handshake")
	h.cancel()
	if err := h.eg.Wait(); err != nil {
		h.log.WithError(err).Error("failed to stop handshake")
	}
	if err := h.emitter.Close(); err != nil {
		h.log.WithError(err).Error("failed to close emitter")
	}
}

// start starts background workers.
func (h *Handshake) start(ctx context.Context, sub event.Subscription) {
	h.eg.Go(func() error {
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				if ctx.Err() != nil {
					return fmt.Errorf("handshake ctx: %w", ctx.Err())
				}
				return nil
			case evt := <-sub.Out():
				id, ok := evt.(event.EvtPeerIdentificationCompleted)
				if !ok {
					h.log.WithError(fmt.Errorf("unexpected event type: %T", evt)).Error("unexpected event type")
					continue
				}
				logger := h.log.WithField("pid", id.Peer.String())
				h.eg.Go(func() error {
					logger.Debug("handshake with peer")
					if err := h.Request(ctx, id.Peer); err != nil {
						logger.WithError(err).Warning("failed to complete handshake with peer")
						return h.h.Network().ClosePeer(id.Peer)
					}
					logger.Debug("handshake completed")
					return nil
				})
			}
		}
	})
}

// Request handshake with a peer.
func (h *Handshake) Request(ctx context.Context, peerID peer.ID) error {
	stream, err := h.h.NewStream(network.WithNoDial(ctx, "existing connection"), peerID, handshakeProtocol)
	if err != nil {
		return fmt.Errorf("failed to init stream: %w", err)
	}
	defer stream.Close()

	if err = json.NewEncoder(stream).Encode(&Message{
		NetworkID: h.networkID,
	}); err != nil {
		return fmt.Errorf("failed to send handshake msg: %w", err)
	}
	var ack Ack
	if err = json.NewDecoder(stream).Decode(&ack); err != nil {
		return fmt.Errorf("failed to receive handshake ack: %w", err)
	}
	if len(ack.Error) > 0 {
		return errors.New(ack.Error)
	}
	if err = h.emitter.Emit(EventHandshakeComplete{
		PID:       peerID,
		Direction: stream.Conn().Stat().Direction,
		NodeURL:   ack.NodeURL,
	}); err != nil {
		h.log.WithError(err).Error("failed to emit handshake event")
	}
	return nil
}

// handler handles incoming handshake requests.
func (h *Handshake) handler(stream network.Stream) {
	defer stream.Close()
	var msg Message

	if err := json.NewDecoder(stream).Decode(&msg); err != nil {
		return
	}
	if h.networkID != msg.NetworkID {
		h.log.WithField("networkID", h.networkID).
			WithField("peer networkID", msg.NetworkID).
			WithField("peerID", stream.Conn().RemotePeer().String()).
			WithField("peerAddress", stream.Conn().LocalMultiaddr().String()).
			Warning("network id mismatch")
		return
	}

	if err := json.NewEncoder(stream).Encode(&Ack{NodeURL: h.nodeURL}); err != nil {
		return
	}
}
