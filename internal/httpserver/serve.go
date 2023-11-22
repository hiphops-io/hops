package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/hashicorp/hcl/v2"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/logs"
	"github.com/hiphops-io/hops/nats"
)

type NatsClient interface {
	Publish(context.Context, []byte, ...string) (*jetstream.PubAck, bool, error)
	CheckConnection() bool
	GetEventHistory(context.Context, time.Time) (*nats.EventLog, error)
	GetEventHistoryDefault(context.Context) (*nats.EventLog, error)
}

func Serve(addr string, hopsContent *hcl.BodyContent, natsClient NatsClient, logger zerolog.Logger) error {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RedirectSlashes)
	r.Use(logs.AccessLogMiddleware(logger))
	r.Use(Healthcheck(natsClient, "/health"))
	// TODO: Make CORS configurable and lock down by default. As-is it would be
	// insecure for production/deployed use.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Serve the single page app for the console from the UI dir
	r.Mount("/console", ConsoleRouter(logger))

	// Serve the tasks API
	taskHops, err := parseTasks(hopsContent)
	if err != nil {
		return err
	}

	// Serve the tasks API
	r.Mount("/tasks", TaskRouter(taskHops, natsClient, logger))

	// Serve the events API
	r.Mount("/events", EventRouter(natsClient, logger))

	logger.Info().Msgf("Console available on http://%s/console", addr)
	return http.ListenAndServe(addr, r)
}

func parseTasks(hopsContent *hcl.BodyContent) (*dsl.HopAST, error) {
	ctx := context.Background()
	taskHops, err := dsl.ParseHopsTasks(ctx, hopsContent)
	if err != nil {
		return nil, err
	}

	return taskHops, nil
}
