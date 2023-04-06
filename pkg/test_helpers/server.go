package test_helpers

import (
	"net/http"
	"p2p_module/pkg/config"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	Echo   *echo.Echo
	config *config.Config
}

func CreateServer(config *config.Config) *Server {
	newEcho := echo.New()

	newEcho.HideBanner = true
	newEcho.HidePort = true

	corsConfig := middleware.CORSConfig{
		Skipper:      middleware.DefaultSkipper,
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}

	newEcho.Use(
		middleware.CORSWithConfig(corsConfig),
		middleware.BodyLimit("101MB"),
	)

	p := prometheus.NewPrometheus("echo", nil)
	p.Use(newEcho)

	return &Server{
		Echo:   newEcho,
		config: config,
	}
}

func (s *Server) Stop() {
	if err := s.Echo.Close(); err != nil {
		log.WithError(err).Warning("Can't stop HTTP Server!")
	}
}

func (s *Server) Start() {
	port := s.config.Http.Port
	if port == "" {
		log.Fatal("Port is not set! (config.Http.Port)")
	}

	s.Echo.Logger.Fatal(s.startHttp2Server(port))
}

func (s *Server) startHttp2Server(port string) error {
	http2Server := &http2.Server{
		MaxConcurrentStreams:         s.config.Http.MaxConcurrentStreams,
		MaxReadFrameSize:             s.config.Http.MaxReadFrameSize,
		MaxUploadBufferPerConnection: s.config.Http.MaxUploadBufferPerConnection,
		MaxUploadBufferPerStream:     s.config.Http.MaxUploadBufferPerStream,
	}
	httpServer := http.Server{
		Addr:    ":" + port,
		Handler: h2c.NewHandler(s.Echo, http2Server),
	}

	return httpServer.ListenAndServe()
}

func (s *Server) RegisterRoute(method string, path string, handler echo.HandlerFunc) {
	(*s.Echo).Add(method, path, handler)
}

func (s *Server) RegisterGroupMiddleware(prefix string, m ...echo.MiddlewareFunc) *echo.Group {
	return (*s.Echo).Group(prefix, m...)
}
