package hops

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
	"github.com/hiphops-io/hops/nats"
	"github.com/rs/zerolog"
)

type (
	EventsClient interface {
		GetEventHistory(ctx context.Context, start time.Time, sourceOnly bool) ([]*nats.MsgMeta, error)
	}
	eventController struct {
		logger       zerolog.Logger
		eventsClient EventsClient
	}

	// Event is arbitrary json struct of event
	Event interface{}

	// EventLog is a list of events with search start and search end timestamps
	EventLog struct {
		StartTimestamp time.Time   `json:"start_timestamp"`
		EndTimestamp   time.Time   `json:"end_timestamp"`
		EventItems     []EventItem `json:"event_items"`
	}

	// EventItem includes metadata for /events api endpoint
	EventItem struct {
		Event       Event     `json:"event"`
		SequenceId  string    `json:"sequence_id"`
		Timestamp   time.Time `json:"timestamp"`
		AppName     string    `json:"app_name"`
		Channel     string    `json:"channel"`
		Done        bool      `json:"done"`
		HandlerName string    `json:"handler_name"`
		MessageId   string    `json:"message_id"`
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

// listEvents returns a list of events in reverse chronological order, with a
// default lookback of 1 hour and a limit of 100 events. (const nats.DefaultEventLookback,
// const nats.GetEventHistoryEventLimit)
func (c *eventController) listEvents(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	query := r.URL.Query()

	sourceOnly := false
	if query.Get("sourceonly") == "true" {
		sourceOnly = true
	}
	// default lookback
	start := time.Now().Add(nats.DefaultEventLookback)

	msgs, err := c.eventsClient.GetEventHistory(ctx, start, sourceOnly)
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
		var event Event
		err := json.Unmarshal([]byte(m.Msg().Data()), &event)
		if err != nil {
			return nil, fmt.Errorf("Error unmarshalling event: %v", err)
		}
		eventItem := EventItem{
			Event:       event,
			SequenceId:  m.SequenceId,
			Timestamp:   m.Timestamp,
			AppName:     m.AppName,
			Channel:     m.Channel,
			Done:        m.Done,
			HandlerName: m.HandlerName,
			MessageId:   m.MessageId,
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
