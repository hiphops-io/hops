// Package runner contains the logic for the hiphops runner/orchestrator
package runner

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/v2"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/robfig/cron"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/nats"
)

type Runner struct {
	automations       *dsl.Automations
	automationsLoader *dsl.AutomationsLoader
	consumer          jetstream.Consumer
	cron              *cron.Cron
	hopsLock          sync.RWMutex
	logger            zerolog.Logger
	natsClient        *nats.Client
	schedules         []*Schedule
}

func NewRunner(natsClient *nats.Client, automationsLoader *dsl.AutomationsLoader, consumer jetstream.Consumer, logger zerolog.Logger) (*Runner, error) {
	r := &Runner{
		automationsLoader: automationsLoader,
		consumer:          consumer,
		logger:            logger,
		natsClient:        natsClient,
	}

	err := r.Reload(context.Background())
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Runner) Reload(ctx context.Context) error {
	automations, err := r.automationsLoader.Get()
	if err != nil {
		return err
	}

	// TODO: Check we actually need to store the automations in here (with locking etc)
	// rather than just use automationsLoader copy directly
	r.hopsLock.Lock()
	defer r.hopsLock.Unlock()

	r.automations = automations

	err = r.prepareHopsSchedules()
	if err != nil {
		return fmt.Errorf("Unable to create schedules %w", err)
	}

	r.setCron()

	return nil
}

func (r *Runner) Run(ctx context.Context) error {
	defer func() {
		if r.cron != nil {
			r.cron.Stop()
		}
	}()

	return r.natsClient.Consume(ctx, r.consumer, r.MessageHandler)
}

func (r *Runner) MessageHandler(
	ctx context.Context,
	msgData []byte,
	msgMeta *nats.MsgMeta,
	ackDeadline time.Duration,
) error {
	logger := r.logger.With().Str("sequence_id", msgMeta.SequenceId).Logger()

	ons, d := r.automations.EventOns(msgData)
	if d.HasErrors() {
		r.logDiagnostics(d, logger)
		return fmt.Errorf("%w: %s", nats.ErrEventFatal, d.Error())
	}

	if len(ons) == 0 {
		return nil
	}

	var mergedErrors error
	// NOTE: We could improve performance by dispatching each function in a go routine
	for i := range ons {
		on := ons[i]
		onLogger := logger.With().Str("on", on.Slug).Logger()

		err := r.executeFunction(on, msgMeta.SequenceId, onLogger)
		if err != nil {
			mergedErrors = multierror.Append(mergedErrors, err)
		}
	}

	return mergedErrors
}

func (r *Runner) executeFunction(on *dsl.On, sequenceId string, logger zerolog.Logger) error {
	// TODO: Implement
	logger.Debug().Msgf("Running function for on: %s, sequence: %s", on.Name, sequenceId)
	return nil
}

// prepareHopsSchedules parses the schedule blocks in a hops config and inits
// the cron schedules ready for running
//
// This function will not run the schedules, just prepare them
// This function should only ever be called within a lock on r.hopsLock
func (r *Runner) prepareHopsSchedules() error {
	schedules := []*Schedule{}
	for _, scheduleConf := range r.automations.GetSchedules() {
		schedule, err := NewSchedule(scheduleConf, r.natsClient, r.logger)
		if err != nil {
			return err
		}

		schedules = append(schedules, schedule)
	}

	r.schedules = schedules

	return nil
}

// TODO: Rename setCron. Name is meaningless.
func (r *Runner) setCron() {
	if r.cron != nil {
		r.cron.Stop()
	}

	r.cron = cron.New()

	for _, schedule := range r.schedules {
		r.cron.Schedule(schedule.CronSchedule, schedule)
	}
	r.cron.Start()
}

func (r *Runner) logDiagnostics(diags hcl.Diagnostics, logger zerolog.Logger) {
	for _, diag := range diags {
		errLog := logger.Error()

		var manifest *dsl.Manifest

		if diag.Subject != nil {
			automationDir := filepath.Dir(diag.Subject.Filename)
			manifest = r.automations.Manifests[automationDir]
		}

		if manifest != nil {
			errLog = errLog.Str("automation", manifest.Name)
		}

		errLog = errLog.Interface("diagnostic", diag)

		errLog.Msg(diag.Summary)
	}
}
