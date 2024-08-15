// Package runner contains the logic for the hiphops runner/orchestrator
package runner

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/robfig/cron"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/markdown"
	"github.com/hiphops-io/hops/nats"
)

type Runner struct {
	flowReader *markdown.FlowReader
	consumer   jetstream.Consumer
	cron       *cron.Cron
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
	if err := r.flowReader.ReadAll(); err != nil {
		return err
	}

	if err := r.prepareHopsSchedules(); err != nil {
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
	hopsMsg *nats.HopsMsg,
	ackDeadline time.Duration,
) error {
	logger := r.logger.With().Str("sequence_id", hopsMsg.SequenceId).Logger()
	logger.Debug().Msgf("Received event '%s'", hopsMsg.Subject)

	switch hopsMsg.Event {
	case "command_request":
		return r.handleCommandRequest(hopsMsg)
	case "command":
		return r.handleCommand(ctx, hopsMsg, logger)
	default:
		return r.handleSourceEvent(ctx, hopsMsg, logger)
	}
}

func (r *Runner) dispatchFlow(ctx context.Context, wg *sync.WaitGroup, flow *markdown.Flow, hopsMsg *nats.HopsMsg, errChan chan<- error, logger zerolog.Logger) {
	defer wg.Done()

	dataB, err := json.Marshal(hopsMsg.Data)
	if err != nil {
		errChan <- err
		return
	}

	subject := nats.WorkSubject(hopsMsg.SequenceId, flow.Worker)
	if _, _, err := r.natsClient.Publish(ctx, dataB, subject); err != nil {
		errChan <- err
		return
	}

	logger.Info().Msgf("Dispatched flow: %s", flow.ID)

	errChan <- nil
}

func (r *Runner) dispatchFlows(ctx context.Context, flows []*markdown.Flow, hopsMsg *nats.HopsMsg, logger zerolog.Logger) error {
	if len(flows) == 0 {
		return nil
	}

	var err error
	var wg sync.WaitGroup
	errChan := make(chan error, len(flows))

	for _, flow := range flows {
		flow := flow
		wg.Add(1)
		flowLogger := logger.With().Str("flow", flow.ID).Logger()
		go r.dispatchFlow(ctx, &wg, flow, hopsMsg, errChan, flowLogger)
	}

	wg.Wait()
	close(errChan)

	for e := range errChan {
		err = errors.Join(err, e)
	}

	if err != nil {
		logger.Error().Err(err).Msg("Unable to dispatch flows")
	}

	return err
}

func (r *Runner) handleCommandRequest(hopsMsg *nats.HopsMsg) error {
	flow, err := markdown.MatchCommandFlows(r.flowReader.IndexedCommands(), hopsMsg, nil)

	switch hopsMsg.Source {
	case "slack":
		return SlackCommandRequest(flow, hopsMsg, err, SlackAccessTokenFunc(r.natsClient))
	default:
		if err != nil {
			return err
		}

		return fmt.Errorf("unsupported command request source '%s'", hopsMsg.Source)
	}
}

func (r *Runner) handleCommand(ctx context.Context, hopsMsg *nats.HopsMsg, logger zerolog.Logger) error {
	switch hopsMsg.Source {
	case "slack":
		if err := SlackBlocksToCommandEvent(hopsMsg); err != nil {
			return fmt.Errorf("unable to process slack command: %w", err)
		}
	default:
		return fmt.Errorf("unsupported command source '%s'", hopsMsg.Source)
	}

	// Get the flow for this command and trigger it
	cmd, ok := r.flowReader.IndexedCommands()[hopsMsg.Action]
	if !ok {
		return fmt.Errorf("unknown command received '%s'", hopsMsg.Action)
	}

	return r.dispatchFlows(ctx, []*markdown.Flow{cmd}, hopsMsg, logger)
}

func (r *Runner) handleSourceEvent(ctx context.Context, hopsMsg *nats.HopsMsg, logger zerolog.Logger) error {
	matchedFlows, err := markdown.MatchFlows(r.flowReader.IndexedSensors(), hopsMsg, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", nats.ErrEventFatal, err)
	}

	return r.dispatchFlows(ctx, matchedFlows, hopsMsg, logger)
}

// prepareHopsSchedules parses the schedule blocks in a hops config and inits
// the cron schedules ready for running
//
// This function will not run the schedules, just prepare them
func (r *Runner) prepareHopsSchedules() error {
	schedules := []*Schedule{}
	scheduledFlows := r.flowReader.IndexedSchedules()

	for _, flow := range scheduledFlows {
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
