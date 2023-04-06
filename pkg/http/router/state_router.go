package router

import (
	"encoding/json"
	"net/http"
	"p2p_module/pkg/fetcher"
	"p2p_module/pkg/test_helpers"

	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	StateBasePathV1 = "/api/v1/state"
)

type StateRouterV1 struct {
	server *test_helpers.Server
	fetch  *fetcher.Fetcher
}

func CreateStateRouter(server *test_helpers.Server, fetch *fetcher.Fetcher) *StateRouterV1 {
	return &StateRouterV1{
		server: server,
		fetch:  fetch,
	}
}

func (p *StateRouterV1) RegisterRoutes() {
	pieceGroup := p.server.RegisterGroupMiddleware(StateBasePathV1)
	pieceGroup.Add(http.MethodGet, "/bootnode", p.getKnownPeers)
	pieceGroup.Add(http.MethodPost, "", p.handleEchoMeasure)
}

func (p *StateRouterV1) getKnownPeers(c echo.Context) error {
	return c.JSON(http.StatusOK, p.fetch.ServeKnownPeers())
}

func (p *StateRouterV1) handleEchoMeasure(c echo.Context) error {
	var payload struct {
		ID  peer.ID `json:"id"`
		MID string  `json:"mid"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&payload); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}
	return c.JSON(http.StatusOK, p.fetch.HandleEchoMeasure(payload.ID, payload.MID))
}
