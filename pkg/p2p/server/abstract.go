package server

import (
	"context"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// Opt is a type to configure a server.
type Opt func(s *Server)

// Handler is the handler to be defined by the application.
type Handler func(context.Context, []byte) ([]byte, error)

// RespHandler is the handler which is called when a request is sent.
type RespHandler func(peer.ID, []byte)

// RespErrHandler is the handler which is called when an error occurs while performing request.
type RespErrHandler func(peer.ID, error)

// Response is a server response.
type Response struct {
	Data  []byte
	Error string
}

// Host is a subset of libp2p Host interface that needs to be implemented to be usable with server.
type Host interface {
	SetStreamHandler(protocol.ID, network.StreamHandler)
	NewStream(context.Context, peer.ID, ...protocol.ID) (network.Stream, error)
	Network() network.Network
}

// Requester is object that can send requests to other nodes over p2p.
type Requester interface {
	Request(context.Context, peer.ID, []byte, RespHandler, RespErrHandler) error
}
