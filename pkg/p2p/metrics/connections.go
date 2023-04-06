package metrics

import (
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	connections        = NewGauge("total_connections", "Total number of connections", nil)
	streamsPerProtocol = NewGauge(
		"streams_per_protocol",
		"Number of streams per protocol",
		[]string{"protocol"},
	)
	durationHistogram = NewHistogram(
		"requests_duration",
		"Histogram of server request durations per protocol (seconds)",
		[]string{"protocol"},
	)
)

// ConnectionsMeeter stores the number of connections for node.
// number of connections
// number of streams per each protocol
// histogram for server request durations for each protocol.
type ConnectionsMeeter struct{}

// NewConnectionsMeeter returns a new ConnectionsMeeter.
func NewConnectionsMeeter() *ConnectionsMeeter {
	return &ConnectionsMeeter{}
}

// Listen called when network starts listening on an addr.
func (c *ConnectionsMeeter) Listen(network.Network, ma.Multiaddr) {}

// ListenClose called when network stops listening on an addr.
func (c *ConnectionsMeeter) ListenClose(network.Network, ma.Multiaddr) {}

// Connected called when a connection opened.
func (c *ConnectionsMeeter) Connected(network.Network, network.Conn) {
	connections.WithLabelValues().Inc()
}

// Disconnected called when a connection closed.
func (c *ConnectionsMeeter) Disconnected(network.Network, network.Conn) {
	connections.WithLabelValues().Dec()
}

// OpenedStream called when a stream opened.
func (c *ConnectionsMeeter) OpenedStream(_ network.Network, str network.Stream) {
	streamsPerProtocol.WithLabelValues(string(str.Protocol())).Inc()
}

// ClosedStream called when a stream closed.
func (c *ConnectionsMeeter) ClosedStream(_ network.Network, str network.Stream) {
	protocolID := string(str.Protocol())
	streamsPerProtocol.WithLabelValues(protocolID).Dec()

	// log stream duration
	duration := time.Since(str.Stat().Opened)
	durationHistogram.WithLabelValues(protocolID).Observe(duration.Seconds())
}
