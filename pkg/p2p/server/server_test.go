package server_test

import (
	"context"
	"errors"
	"p2p_module/pkg/p2p/server"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/require"
)

const (
	testProtocol = "test"
)

func TestServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	mesh, err := mocknet.FullMeshConnected(4)
	require.NoError(t, err)
	request := []byte("test request")
	testErr := errors.New("test error")
	errCh := make(chan error, 1)
	respCh := make(chan []byte, 1)

	handler := func(_ context.Context, msg []byte) ([]byte, error) {
		return msg, nil
	}
	errHandler := func(_ context.Context, _ []byte) ([]byte, error) {
		return nil, testErr
	}
	opts := []server.Opt{
		server.WithTimeout(100 * time.Millisecond),
		server.WithContext(ctx),
	}
	client := server.New(mesh.Hosts()[0], testProtocol, handler, opts...)
	_ = server.New(mesh.Hosts()[1], testProtocol, handler, opts...)
	_ = server.New(mesh.Hosts()[2], testProtocol, errHandler, opts...)

	respHandler := func(_ peer.ID, msg []byte) {
		select {
		case <-ctx.Done():
		case respCh <- msg:
		}
	}
	respErrHandler := func(_ peer.ID, err error) {
		select {
		case <-ctx.Done():
		case errCh <- err:
		}
	}
	t.Run("ReceiveMessage", func(t *testing.T) {
		require.NoError(t, client.Request(ctx, mesh.Hosts()[1].ID(), request, respHandler, respErrHandler))
		select {
		case <-time.After(time.Second):
			require.FailNow(t, "timed out while waiting for message response")
		case response := <-respCh:
			require.Equal(t, request, response)
		}
	})
	t.Run("ReceiveError", func(t *testing.T) {
		require.NoError(t, client.Request(ctx, mesh.Hosts()[2].ID(), request, respHandler, respErrHandler))
		select {
		case <-time.After(time.Second):
			require.FailNow(t, "timed out while waiting for error response")
		case err := <-errCh:
			require.Equal(t, testErr, err)
		}
	})
	t.Run("DialError", func(t *testing.T) {
		require.NoError(t, client.Request(ctx, mesh.Hosts()[3].ID(), request, respHandler, respErrHandler))
		select {
		case <-time.After(time.Second):
			require.FailNow(t, "timed out while waiting for dial error")
		case err := <-errCh:
			require.Error(t, err)
		}
	})
	t.Run("NotConnected", func(t *testing.T) {
		require.ErrorIs(t, client.Request(ctx, "unknown", request, respHandler, respErrHandler), server.ErrNotConnected)
	})
}
