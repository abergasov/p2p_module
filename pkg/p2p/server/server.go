package server

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	log "github.com/sirupsen/logrus"
)

// ErrNotConnected is returned when peer is not connected.
var ErrNotConnected = errors.New("peer is not connected")

// Server for the Handler.
type Server struct {
	protocol string
	handler  Handler
	timeout  time.Duration

	h Host

	ctx context.Context
}

// WithTimeout configures stream timeout.
func WithTimeout(timeout time.Duration) Opt {
	return func(s *Server) {
		s.timeout = timeout
	}
}

// WithContext configures parent context for contexts that are passed to the handler.
func WithContext(ctx context.Context) Opt {
	return func(s *Server) {
		s.ctx = ctx
	}
}

// New server for the handler.
func New(h Host, proto string, handler Handler, opts ...Opt) *Server {
	srv := &Server{
		ctx:      context.Background(),
		protocol: proto,
		handler:  handler,
		h:        h,
		timeout:  10 * time.Second,
	}
	for _, opt := range opts {
		opt(srv)
	}
	h.SetStreamHandler(protocol.ID(srv.protocol), srv.streamHandler)
	return srv
}

// streamHandler handles incoming stream.
func (s *Server) streamHandler(stream network.Stream) {
	defer stream.Close()
	_ = stream.SetDeadline(time.Now().Add(s.timeout))
	rd := bufio.NewReader(stream)
	size, err := binary.ReadUvarint(rd)
	if err != nil {
		return
	}
	buf := make([]byte, size)
	if _, err = io.ReadFull(rd, buf); err != nil {
		return
	}
	start := time.Now()
	buf, err = s.handler(s.ctx, buf)
	log.WithField("protocol", s.protocol).WithField("duration", time.Since(start)).Debug("protocol handler execution time")

	var resp Response
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Data = buf
	}

	wr := bufio.NewWriter(stream)
	if err = json.NewEncoder(wr).Encode(&resp); err != nil {
		log.WithError(err).Warning("failed to write response")
		return
	}
	if err = wr.Flush(); err != nil {
		log.WithError(err).Warning("failed to flush stream")
	}
}

// Request sends a binary request to the peer. Request is executed in the background, one of the callbacks
// is guaranteed to be called on success/error.
func (s *Server) Request(ctx context.Context, pid peer.ID, req []byte, resp RespHandler, failure RespErrHandler) error {
	if s.h.Network().Connectedness(pid) != network.Connected {
		return fmt.Errorf("%w: %s", ErrNotConnected, pid)
	}
	go s.request(ctx, pid, req, resp, failure)
	return nil
}

// request sends a binary request to the peer.
func (s *Server) request(ctx context.Context, pid peer.ID, req []byte, resp RespHandler, failure RespErrHandler) {
	start := time.Now()
	defer func() {
		log.WithContext(ctx).
			WithField("protocol", s.protocol).
			WithField("duration", time.Since(start)).
			Debug("request execution time")
	}()
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	stream, err := s.h.NewStream(network.WithNoDial(ctx, "existing connection"), pid, protocol.ID(s.protocol))
	if err != nil {
		failure(pid, err)
		return
	}
	defer stream.Close()
	_ = stream.SetDeadline(time.Now().Add(s.timeout))

	wr := bufio.NewWriter(stream)
	sz := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(sz, uint64(len(req)))

	if _, err = wr.Write(sz[:n]); err != nil {
		failure(pid, err)
		return
	}
	if _, err = wr.Write(req); err != nil {
		failure(pid, err)
		return
	}
	if err = wr.Flush(); err != nil {
		failure(pid, err)
		return
	}

	rd := bufio.NewReader(stream)
	var r Response
	if err = json.NewDecoder(rd).Decode(&r); err != nil {
		failure(pid, err)
		return
	}
	if len(r.Error) > 0 {
		failure(pid, errors.New(r.Error))
	} else {
		resp(pid, r.Data)
	}
}
