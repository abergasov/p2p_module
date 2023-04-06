package fetcher_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"p2p_module/pkg/config"
	"p2p_module/pkg/fetcher"
	"p2p_module/pkg/http/router"
	"p2p_module/pkg/p2p"
	"p2p_module/pkg/test_helpers"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type testBootNode struct {
	bootNodeAddress string
	peersList       []string
}

type eventHandler struct {
	msgLog   []string // here will contain all the messages that this node served
	msgLogMU sync.Mutex
}

func (e *eventHandler) HandleEvent() string {
	e.msgLogMU.Lock()
	defer e.msgLogMU.Unlock()
	msg := fmt.Sprintf("stat %s - %s", time.Now().Format("2006-01-02 15:04:05"), uuid.NewString())
	e.msgLog = append(e.msgLog, msg)
	return msg
}

func (e *eventHandler) GetLog() []string {
	e.msgLogMU.Lock()
	defer e.msgLogMU.Unlock()
	return e.msgLog
}

type testFetcherNode struct {
	fetcherNode *fetcher.Fetcher
	p2pAddress  string
	evt         *eventHandler
}

func spawnBootNode(t *testing.T) *testBootNode {
	bootNodePort := getFreePort(t)
	tb := &testBootNode{
		bootNodeAddress: "http://localhost:" + strconv.Itoa(bootNodePort),
		peersList:       make([]string, 0),
	}
	e := echo.New()
	e.HideBanner = true
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, tb.peersList)
	})
	go func() {
		_ = e.Start(":" + strconv.Itoa(bootNodePort))
	}()
	t.Cleanup(func() {
		require.NoError(t, e.Close())
	})
	return tb
}

// spawnFetcherNode generate new p2p node with http routing
// after this method we will have a new node with http Server and p2p host with extra http methods to get state of fetcher
func spawnFetcherNode(t *testing.T, bootNodePath string, nodeID int) *testFetcherNode {
	logger := log.WithField("node", nodeID)
	nodeP2PAddress := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", getFreePort(t))

	httpPort := getFreePort(t)
	httpNodeAddress := fmt.Sprintf("http://127.0.0.1:%d", httpPort)

	host, err := p2p.CreateHost(logger, httpNodeAddress, config.P2P{
		GracePeersShutdown: 30,
		DataDir:            generateTmpFolder(t),
		BootNode:           bootNodePath,
		LowPeers:           15,
		HighPeers:          10,
		Listen:             nodeP2PAddress,
	})
	require.NoError(t, err)
	fetch, err := fetcher.NewFetcher(logger, host)
	require.NoError(t, err)
	t.Cleanup(fetch.Stop)

	// register http Server with fetcher methods
	httpServer := test_helpers.CreateServer(&config.Config{
		Http: config.Http{
			Port:                         strconv.Itoa(httpPort),
			MaxConcurrentStreams:         12,
			MaxUploadBufferPerStream:     110,
			MaxReadFrameSize:             110,
			MaxUploadBufferPerConnection: 110,
		},
	})

	router.CreateStateRouter(httpServer, fetch).RegisterRoutes()
	t.Cleanup(httpServer.Stop)

	eh := &eventHandler{
		msgLog:   make([]string, 0, 100),
		msgLogMU: sync.Mutex{},
	}

	httpServer.Echo.Add(http.MethodGet, "/stat", func(c echo.Context) error {
		return c.String(http.StatusOK, eh.HandleEvent())
	})

	httpServer.Echo.Add(http.MethodGet, "/log", func(c echo.Context) error {
		return c.JSON(http.StatusOK, eh.GetLog())
	})
	go httpServer.Start()
	return &testFetcherNode{
		fetcherNode: fetch,
		p2pAddress:  fmt.Sprintf("%s/p2p/%s", nodeP2PAddress, host.GetHostID()),
		evt:         eh,
	}
}

// getFreePort generate a free tcp port for testing
func getFreePort(t *testing.T) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	require.NoError(t, err)

	l, err := net.ListenTCP("tcp", addr)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, l.Close())
	}()
	return l.Addr().(*net.TCPAddr).Port
}

// generateTmpFolder generate a temporary file for testing
// will contain p2p key and addressBook file
func generateTmpFolder(t *testing.T) string {
	tmpFolder, err := os.MkdirTemp(os.TempDir(), "cere_p2p_helper_")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpFolder))
	})
	return tmpFolder
}
