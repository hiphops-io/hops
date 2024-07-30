package runner

import (
	"context"
	"time"

	"github.com/robfig/cron"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/markdown"
	"github.com/hiphops-io/hops/nats"
)

type Schedule struct {
	CronSchedule cron.Schedule
	flow         *markdown.Flow
	logger       zerolog.Logger
	natsClient   *nats.Client
}

func NewSchedule(flow *markdown.Flow, natsClient *nats.Client, logger zerolog.Logger) (*Schedule, error) {
	cronSchedule, err := cron.ParseStandard(flow.Schedule)
	if err != nil {
		return nil, err
	}

	schedule := &Schedule{
		flow:         flow,
		CronSchedule: cronSchedule,
		logger:       logger,
		natsClient:   natsClient,
	}

	return schedule, nil
}

func (s *Schedule) Run() {
	s.logger.Info().Msgf("Triggering schedule %s", s.flow.ID)
	ctx := context.Background()

	now := time.Now().UTC()
	// Timestamp without seconds to create 'buckets' for idempotency
	triggerTime := now.Format(time.RFC822Z)

	// Create the payload with trigger time
	schedulePayload := map[string]any{"trigger_time": triggerTime}

	// Construct the source event
	sourceEvent, sequenceID, err := nats.CreateSourceEvent(schedulePayload, "hiphops", "schedule", s.flow.ActionName(), "")
	if err != nil {
		s.logger.Error().Err(err).Msgf("Unable to create source event for schedule: %s", s.flow.ID)
		return
	}

	// Dispatch the source event
	subject := nats.SourceEventSubject(sequenceID)
	_, _, err = s.natsClient.Publish(ctx, sourceEvent, subject)
	if err != nil {
		s.logger.Error().Err(err).Msgf("Unable to dispatch source event for schedule: %s", s.flow.ID)
	}
}
