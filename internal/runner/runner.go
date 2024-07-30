// Package runner contains the logic for the hiphops runner/orchestrator
package runner

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/robfig/cron"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/markdown"
	"github.com/hiphops-io/hops/nats"
)

type Runner struct {
	flows      markdown.FlowIndex
	flowReader *markdown.FlowReader
	consumer   jetstream.Consumer
	cron       *cron.Cron
	flowMtx    sync.RWMutex
	logger     zerolog.Logger
	natsClient *nats.Client
	schedules  []*Schedule
}

func NewRunner(natsClient *nats.Client, flowReader *markdown.FlowReader, consumer jetstream.Consumer, logger zerolog.Logger) (*Runner, error) {
	r := &Runner{
		flowReader: flowReader,
		consumer:   consumer,
		logger:     logger,
		natsClient: natsClient,
	}

	err := r.Load(context.Background())
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Runner) Load(ctx context.Context) error {
	flows, err := r.flowReader.ReadAll()
	if err != nil {
		return err
	}

	r.flowMtx.Lock()
	r.flows = flows
	r.flowMtx.Unlock()

	err = r.prepareHopsSchedules()
	if err != nil {
		return fmt.Errorf("Unable to create schedules %w", err)
	}

	r.startCron()

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
	logger.Debug().Msgf("Received event '%s'", msgMeta.Subject)

	r.flowMtx.RLock()
	matchedFlows, err := markdown.MatchFlows(r.flows, msgData)
	r.flowMtx.RUnlock()
	if err != nil {
		return fmt.Errorf("%w: %w", nats.ErrEventFatal, err)
	}

	if len(matchedFlows) == 0 {
		return nil
	}

	// Now we've collected the matching flows, dispatch their work.

	var errs error
	var wg sync.WaitGroup
	errChan := make(chan error, len(matchedFlows))

	for _, flow := range matchedFlows {
		flow := flow
		wg.Add(1)
		onLogger := logger.With().Str("on", flow.ID).Logger()

		go r.dispatchWork(ctx, &wg, flow, msgData, msgMeta, errChan, onLogger)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		errs = errors.Join(errs, err)
	}

	return errs
}

func (r *Runner) dispatchWork(ctx context.Context, wg *sync.WaitGroup, flow *markdown.Flow, data []byte, meta *nats.MsgMeta, errChan chan<- error, logger zerolog.Logger) {
	defer wg.Done()

	subject := nats.WorkSubject(meta.SequenceId, flow.Worker)
	if _, _, err := r.natsClient.Publish(ctx, data, subject); err != nil {
		errChan <- err
		return
	}

	logger.Info().Msgf("Dispatched work: %s", flow.ID)

	errChan <- nil
}

// prepareHopsSchedules parses the schedule blocks in a hops config and inits
// the cron schedules ready for running
//
// This function will not run the schedules, just prepare them
// This function should only ever be called within a lock on r.hopsLock
func (r *Runner) prepareHopsSchedules() error {
	schedules := []*Schedule{}
	for _, flow := range r.flowReader.ScheduledFlows() {
		schedule, err := NewSchedule(flow, r.natsClient, r.logger)
		if err != nil {
			return err
		}

		schedules = append(schedules, schedule)
	}

	r.schedules = schedules

	return nil
}

func (r *Runner) startCron() {
	if r.cron != nil {
		r.cron.Stop()
	}

	r.cron = cron.New()

	for _, schedule := range r.schedules {
		r.cron.Schedule(schedule.CronSchedule, schedule)
	}
	r.cron.Start()
}
