package runner

import (
	"time"

	"github.com/robfig/cron"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/dsl"
)

type Schedule struct {
	Config       dsl.ScheduleAST
	CronSchedule cron.Schedule
	logger       zerolog.Logger
	natsClient   NatsClient
}

func NewSchedule(config dsl.ScheduleAST, natsClient NatsClient, logger zerolog.Logger) (*Schedule, error) {
	cronSchedule, err := cron.ParseStandard(config.Cron)
	if err != nil {
		return nil, err
	}

	schedule := &Schedule{
		Config:       config,
		CronSchedule: cronSchedule,
		logger:       logger,
		natsClient:   natsClient,
	}

	return schedule, nil
}

func (s *Schedule) Run() {
	s.logger.Info().Msg("Running job")
	now := time.Now().UTC()
	triggerTime := now.Format(time.RFC822Z)
	// Create hops metadata with schedule as event, name as action
	// Inputs should be payload of event
	// Sequence should be??

	// Publish on account account_id.notify.sequence_id.event
}
