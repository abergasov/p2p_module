package addressbook

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// KnownAddress tracks information about a known network address that is used
// to determine how viable an address is.
type KnownAddress struct {
	Addr        *peer.AddrInfo
	NodeURL     string
	Attempts    int
	LastSeen    time.Time
	LastAttempt time.Time
	LastSuccess time.Time
}
