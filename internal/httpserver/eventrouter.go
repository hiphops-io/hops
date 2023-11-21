package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
	"github.com/hiphops-io/hops/nats"
	"github.com/rs/zerolog"
)

type (
	EventsClient interface {
		GetEventHistory(ctx context.Context, start time.Time) ([]nats.Event, error)
		GetEventHistoryDefault(ctx context.Context) ([]nats.Event, error)
	}
	eventController struct {
		logger       zerolog.Logger
		eventsClient EventsClient
	}

	eventResponse struct {
		Errors     map[string][]string `json:"errors"`
		Message    string              `json:"message"`
		SequenceID string              `json:"sequence_id"`
		statusCode int
	}
)

func EventRouter(eventsClient EventsClient, logger zerolog.Logger) chi.Router {
	r := chi.NewRouter()
	controller := &eventController{
		logger:       logger,
		eventsClient: eventsClient,
	}
	r.Get("/", controller.listEvents)

	return r
}

func (c *eventController) listEvents(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	events, err := c.eventsClient.GetEventHistoryDefault(ctx)
	if err != nil {
		c.logger.Error().Err(err).Msg("Error getting event history")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
