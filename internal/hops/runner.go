package hops

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/v2"
	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/nats"
)

type Runner struct {
	automations       *dsl.Automations
	automationsLoader *AutomationsLoader
	cache             *cache.Cache
	cron              *cron.Cron
	hopsLock          sync.RWMutex
	logger            zerolog.Logger
	natsClient        *nats.Client
	schedules         []*Schedule
}

func NewRunner(natsClient *nats.Client, automationsLoader *AutomationsLoader, logger zerolog.Logger) (*Runner, error) {
	r := &Runner{
		logger:            logger,
		natsClient:        natsClient,
		automationsLoader: automationsLoader,
		cache:             cache.New(5*time.Minute, 10*time.Minute),
	}

	err := r.Reload(context.Background())
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Runner) Reload(ctx context.Context) error {
	automations, err := r.automationsLoader.Get("")
	if err != nil {
		return err
	}

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

func (r *Runner) Run(ctx context.Context, fromConsumer string) error {
	defer func() {
		if r.cron != nil {
			r.cron.Stop()
		}
	}()

	return r.natsClient.ConsumeSequences(ctx, fromConsumer, r)
}

func (r *Runner) SequenceCallback(
	ctx context.Context,
	sequenceId string,
	msgBundle nats.MessageBundle,
) (bool, error) {
	logger := r.logger.With().Str("sequence_id", sequenceId).Logger()

	automations, err := r.automationsLoader.GetForSequence(ctx, sequenceId, msgBundle)
	if err != nil {
		return false, fmt.Errorf("Unable to fetch automations config for sequence: %w", err)
	}

	ons, d := automations.EventOns(msgBundle)
	if d.HasErrors() {
		r.logDiagnostics(d, logger)
		return false, fmt.Errorf("%w: %s", nats.ErrEventFatal, d.Error())
	}

	if len(ons) == 0 {
		return false, nil
	}

	logger.Debug().Msg("Successfully evaluated automations")

	var mergedErrors error
	// NOTE: We could potentially get a speed boost by dispatching/handling each
	// on block concurrently
	for i := range ons {
		on := ons[i]
		onLogger := logger.With().Str("on", on.Slug).Logger()

		if on.Done != nil {
			err := r.dispatchDone(ctx, on.Slug, on.Done, sequenceId, onLogger)
			if err != nil {
				onLogger.Error().Err(err).Msg("Unable to send pipeline 'done' message")
			}

			continue
		}

		err = r.dispatchCalls(ctx, on, sequenceId, logger)
		if err != nil {
			mergedErrors = multierror.Append(mergedErrors, err)
		}
	}

	return true, mergedErrors
}

func (r *Runner) dispatchDone(ctx context.Context, onSlug string, done *dsl.Done, sequenceId string, logger zerolog.Logger) error {
	var err error
	if done.Errored {
		err = errors.New("Pipeline errored")
	}

	err, sent := r.natsClient.PublishResult(
		ctx,
		time.Now(),
		done.Completed,
		err,
		nats.ChannelNotify,
		sequenceId,
		onSlug,
		nats.DoneMessageId,
	)

	if err != nil {
		return err
	}

	if sent {
		logger.Info().Msg("Pipeline is done")
	}

	return nil
}

func (r *Runner) dispatchCalls(ctx context.Context, on *dsl.On, sequenceId string, logger zerolog.Logger) error {
	var wg sync.WaitGroup
	var errs error

	logger = logger.With().Str("on", on.Slug).Logger()
	logger.Info().Msg("Running on calls")

	numTasks := len(on.Calls)
	errorchan := make(chan error, numTasks)

	for _, call := range on.Calls {
		call := call
		wg.Add(1)
		go r.dispatchCall(ctx, &wg, call, sequenceId, errorchan, logger)
	}

	wg.Wait()
	close(errorchan)

	for err := range errorchan {
		errs = errors.Join(errs, err)
	}

	return errs
}

func (r *Runner) dispatchCall(ctx context.Context, wg *sync.WaitGroup, call *dsl.Call, sequenceId string, errorchan chan<- error, logger zerolog.Logger) {
	defer wg.Done()

	app, handler, found := strings.Cut(call.Label, "_")
	if !found {
		errorchan <- fmt.Errorf("Unable to parse app/handler from call %s", call.Name)
		return
	}

	_, _, err := r.natsClient.Publish(ctx, call.Inputs, nats.ChannelRequest, sequenceId, call.Slug, app, handler)
	if err != nil {
		errorchan <- err
		return
	}

	logger.Info().Msgf("Dispatched call: %s", call.Slug)

	errorchan <- nil
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
