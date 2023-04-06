package handshake

import (
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// EventHandshakeComplete is emitted after handshake is completed.
type EventHandshakeComplete struct {
	PID       peer.ID
	NodeURL   string
	Direction network.Direction
}

// Ack is a handshake ack.
type Ack struct {
	Error   string `json:"error"`
	NodeURL string `json:"node_url"`
}

// Message is a handshake message.
type Message struct {
	NetworkID string `json:"network_id"`
}
