package handshake_test

import (
	"context"
	"p2p_module/pkg/p2p/handshake"
	"testing"

	"github.com/google/uuid"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const (
	networkID = "test"
)

func TestHandshake(t *testing.T) {
	logger := log.WithField("test", t.Name())
	mesh, err := mocknet.FullMeshConnected(3)
	require.NoError(t, err)

	hs1, err := handshake.NewHandshake(logger, mesh.Hosts()[0], networkID, uuid.NewString())
	require.NoError(t, err)
	t.Cleanup(hs1.Stop)

	hs2, err := handshake.NewHandshake(logger, mesh.Hosts()[1], networkID, uuid.NewString())
	require.NoError(t, err)
	t.Cleanup(hs2.Stop)

	hs3, err := handshake.NewHandshake(logger, mesh.Hosts()[2], "apage_satana", uuid.NewString())
	require.NoError(t, err)
	t.Cleanup(hs3.Stop)

	require.NoError(t, hs1.Request(context.TODO(), mesh.Hosts()[1].ID()))
	require.Error(t, hs1.Request(context.TODO(), mesh.Hosts()[2].ID()))
}
