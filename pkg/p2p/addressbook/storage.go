package addressbook

import (
	"context"
	"encoding/json"
	"fmt"
	"p2p_module/pkg/p2p/utils"
	"time"

	log "github.com/sirupsen/logrus"
)

// periodicallyPersistPeers periodically persists peers to disk.
func (a *AddrBook) periodicallyPersistPeers(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	a.eg.Go(func() error {
		for {
			select {
			case <-ticker.C:
				if err := a.persistPeers(); err != nil {
					log.WithError(err).Error("failed to persist peers")
				}
			case <-ctx.Done():
				return nil
			}
		}
	})
}

// PersistPeers persists peers to disk.
func (a *AddrBook) PersistPeers() {
	if err := a.persistPeers(); err != nil {
		log.WithError(err).Error("failed to persist peers")
	}
}

// persistPeers persists peers to disk.
func (a *AddrBook) persistPeers() error {
	a.addrMU.Lock()
	peers := make([]*KnownAddress, 0, len(a.addrIndex))
	for _, addr := range a.addrIndex {
		peers = append(peers, addr)
	}
	a.addrMU.Unlock()
	data, err := json.Marshal(peers)
	if err != nil {
		return fmt.Errorf("failed to encode data: %w", err)
	}
	if err = utils.AtomicallySaveToFile(a.bookPath, data); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}
	return nil
}

// loadPeers load peers from file
// loadPeers loads the known address from the saved file. If empty, missing, or
// malformed file, just don't load anything and start fresh.
func (a *AddrBook) loadPeers() {
	if len(a.bookPath) == 0 {
		return
	}
	data, err := utils.LoadFromFile(a.bookPath)
	if err != nil {
		log.WithError(err).WithField("path", a.bookPath).Warning("failed to load peers file")
		return
	}
	var addresses []*KnownAddress
	if err = json.Unmarshal(data, &addresses); err != nil {
		log.WithError(err).WithField("path", a.bookPath).Warning("failed to parse file")
		return
	}
	a.addrMU.Lock()
	defer a.addrMU.Unlock()
	for _, addr := range addresses {
		a.addrIndex[addr.Addr.ID] = addr
	}
}
