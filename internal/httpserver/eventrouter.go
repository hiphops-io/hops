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
		GetEventHistory(ctx context.Context, start time.Time) ([]*nats.MsgMeta, error)
	}
	eventController struct {
		logger       zerolog.Logger
		eventsClient EventsClient
	}

	// Event is arbitrary json struct of event
	Event map[string](interface{})

	// EventLog is a list of events with search start and search end timestamps
	EventLog struct {
		StartTimestamp time.Time   `json:"start_timestamp"`
		EndTimestamp   time.Time   `json:"end_timestamp"`
		EventItems     []EventItem `json:"event_items"`
	}

	// EventItem includes metadata for /events api endpoint
	EventItem struct {
		Event      Event     `json:"event"`
		SequenceId string    `json:"sequence_id"`
		Timestamp  time.Time `json:"timestamp"`
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

	// default lookback
	start := time.Now().Add(nats.DefaultEventLookback)

	msgs, err := c.eventsClient.GetEventHistory(ctx, start)
	if err != nil {
		c.logger.Error().Err(err).Msg("Error getting event history")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	eventLog, err := eventLogFromMsgMetas(msgs, start)
	if err != nil {
		c.logger.Error().Err(err).Msg("Error reading event history")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eventLog)
}

func eventLogFromMsgMetas(msgs []*nats.MsgMeta, start time.Time) (*EventLog, error) {
	events := []EventItem{}

	n := len(msgs)

	// max 100 messages
	if n > 100 {
		n = 100
	}

	for _, m := range msgs[:n] {
		event := make(Event)
		err := json.Unmarshal([]byte(m.Msg().Data()), &event)
		if err != nil {
			return nil, err
		}
		eventItem := EventItem{
			Event:      event,
			SequenceId: m.SequenceId,
			Timestamp:  m.Timestamp,
		}
		events = append(events, eventItem)
	}

	eventLog := EventLog{
		EventItems:     events,
		StartTimestamp: start,
		EndTimestamp:   time.Now(),
	}

	return &eventLog, nil
}
