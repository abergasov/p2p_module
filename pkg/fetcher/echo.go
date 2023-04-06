package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	// echoProtocol for measure network delivery speed
	echoProtocol = "/cere/ddc/echo/1.0.0"
	echoInterval = 5 * time.Second
)

type EchoMessage struct {
	PeerID peer.ID `json:"peer_id"` // peer who Start measure
	ID     string  `json:"id"`      // uniqueID of measure
}

func (f *Fetcher) startEcho(ctx context.Context, ps *pubsub.Topic) {
	f.eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-f.echoInterval.C:
				if err := f.echoNetwork(ps); err != nil {
					f.log.WithError(err).Error("echo network failed")
				}
			}
		}
	})
}

func (f *Fetcher) loop(ctx context.Context, sub *pubsub.Subscription) {
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			f.log.WithError(err).Error("failed to get next message")
			return
		}

		// only forward messages delivered by others
		if msg.ReceivedFrom == f.host.GetHost().ID() {
			f.log.Info("skip message from self")
			continue
		}
		var em EchoMessage
		if err = json.Unmarshal(msg.Data, &em); err != nil {
			f.log.WithError(err).Error("failed to unmarshal message")
			continue
		}
		go f.echoHandler(em)
	}
}

// echoNetwork sends echo message to all peers and measure delivery speed
// generate ID and broadcast to all peers
func (f *Fetcher) echoNetwork(ps *pubsub.Topic) error {
	eID := uuid.NewString()
	f.echoMeasureMU.Lock()
	f.echoMeasure = Measurer{
		ID:    eID,
		Nodes: make([]peer.ID, 0),
		Start: time.Now(),
	}
	f.echoMeasureMU.Unlock()
	payload, err := json.Marshal(EchoMessage{
		PeerID: f.host.GetHost().ID(),
		ID:     eID,
	})
	if err != nil {
		return fmt.Errorf("marshal echo message failed: %w", err)
	}
	if err = ps.Publish(context.Background(), payload); err != nil {
		return fmt.Errorf("publish echo message failed: %w", err)
	}
	return nil
}

// echoHandler notify node about echo message
func (f *Fetcher) echoHandler(msg EchoMessage) {
	addr, err := f.host.GetAddressBook().GetPeerAddress(msg.PeerID)
	if err != nil {
		f.log.WithError(err).Error("failed to get peer address for process echo")
		return
	}

	data, err := json.Marshal(struct {
		ID  peer.ID `json:"id"`
		MID string  `json:"mid"`
	}{
		ID:  f.host.GetHost().ID(),
		MID: msg.ID,
	})
	if err != nil {
		f.log.WithError(err).Error("failed to marshal echo payload")
		return
	}
	url := fmt.Sprintf("%s/api/v1/state", addr.NodeURL)
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		f.log.WithError(err).Error("failed to create echo request")
		return
	}
	client := http.DefaultClient
	client.Timeout = 5 * time.Second
	res, err := client.Do(request)
	if err != nil {
		f.log.WithError(err).Error("failed to send echo request")
		return
	}
	defer res.Body.Close()
}

// HandleEchoMeasure handles echo measure request
func (f *Fetcher) HandleEchoMeasure(peerID peer.ID, eID string) string {
	f.echoMeasureMU.Lock()
	defer f.echoMeasureMU.Unlock()
	if f.echoMeasure.ID != eID {
		return "expired"
	}
	f.echoMeasure.Nodes = append(f.echoMeasure.Nodes, peerID)
	f.log.WithField("Nodes", len(f.echoMeasure.Nodes)).Info("echo measure Nodes")
	if len(f.echoMeasure.Nodes) == f.host.GetAddressBook().CountPeers() {
		// all Nodes received echo message
		f.log.WithField("duration ms", time.Since(f.echoMeasure.Start).Milliseconds()).Info("echo measure finished")
	}
	return "ok"
}

func (f *Fetcher) ServeMeasureState() Measurer {
	f.echoMeasureMU.Lock()
	defer f.echoMeasureMU.Unlock()
	return f.echoMeasure
}
