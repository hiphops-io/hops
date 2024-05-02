package hops

// import (
// 	"context"
// 	"time"

// 	"github.com/goccy/go-json"
// 	"github.com/robfig/cron"
// 	"github.com/rs/zerolog"

// 	"github.com/hiphops-io/hops/dsl"
// 	"github.com/hiphops-io/hops/nats"
// )

// type Schedule struct {
// 	Config       *dsl.ScheduleAST
// 	CronSchedule cron.Schedule
// 	logger       zerolog.Logger
// 	natsClient   *nats.Client
// }

// func NewSchedule(config *dsl.ScheduleAST, natsClient *nats.Client, logger zerolog.Logger) (*Schedule, error) {
// 	cronSchedule, err := cron.ParseStandard(config.Cron)
// 	if err != nil {
// 		return nil, err
// 	}

// 	schedule := &Schedule{
// 		Config:       config,
// 		CronSchedule: cronSchedule,
// 		logger:       logger,
// 		natsClient:   natsClient,
// 	}

// 	return schedule, nil
// }

// func (s *Schedule) Run() {
// 	s.logger.Info().Msgf("Triggering schedule %s", s.Config.Name)
// 	ctx := context.Background()

// 	now := time.Now().UTC()
// 	// Timestamp without seconds to create 'buckets' for idempotency
// 	triggerTime := now.Format(time.RFC822Z)

// 	// Create the payload and add inputs + trigger time
// 	schedulePayload := map[string]any{}

// 	if s.Config.Inputs != nil {
// 		err := json.Unmarshal(s.Config.Inputs, &schedulePayload)
// 		if err != nil {
// 			s.logger.Error().Err(err).Msgf("Unable to parse inputs for schedule: %s", s.Config.Name)
// 			return
// 		}
// 	}

// 	schedulePayload["trigger_time"] = triggerTime

// 	// Construct the source event
// 	sourceEvent, sequenceID, err := dsl.CreateSourceEvent(schedulePayload, "hiphops", "schedule", s.Config.Name)
// 	if err != nil {
// 		s.logger.Error().Err(err).Msgf("Unable to create source event for schedule: %s", s.Config.Name)
// 		return
// 	}

// 	// Dispatch the source event
// 	_, _, err = s.natsClient.Publish(ctx, sourceEvent, nats.ChannelNotify, sequenceID, "event")
// 	if err != nil {
// 		s.logger.Error().Err(err).Msgf("Unable to dispatch source event for schedule: %s", s.Config.Name)
// 	}
// }
