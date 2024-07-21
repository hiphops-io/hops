package httpserver

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/hiphops-io/hops/nats"
)

type (
	HTTPServer struct {
		address    string
		natsClient *nats.Client
		server     *echo.Echo
	}
)

func NewHTTPServer(addr string, natsClient *nats.Client) *HTTPServer {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		// TODO: The host names etc here will require user config, given this will be self hosted
		AllowOrigins:     []string{"http://localhost:*", "http://0.0.0.0:*", "https://*.hiphops.io"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{echo.HeaderAccept, echo.HeaderAuthorization, echo.HeaderContentType, echo.HeaderXCSRFToken},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	e.Use(echo.WrapMiddleware(nats.HealthcheckMiddleware(natsClient, "/health")))

	return &HTTPServer{
		address:    addr,
		natsClient: natsClient,
		server:     e,
	}
}

func (h *HTTPServer) Serve() error {
	return h.server.Start(h.address)
}

func (h *HTTPServer) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}
