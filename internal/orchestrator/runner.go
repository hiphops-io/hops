package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/nats"
)

type NatsClient interface {
	Publish(context.Context, []byte, ...string) (*jetstream.PubAck, error)
	ConsumeSequences(context.Context, nats.SequenceHandler) error
}

type Runner struct {
	logger     zerolog.Logger
	hopsFiles  dsl.HclFiles
	natsClient NatsClient
}

func NewRunner(natsClient NatsClient, hopsFiles dsl.HclFiles, logger zerolog.Logger) (*Runner, error) {
	runner := &Runner{
		logger:     logger,
		hopsFiles:  hopsFiles,
		natsClient: natsClient,
	}

	return runner, nil
}

func (r *Runner) Run(ctx context.Context) {
	r.natsClient.ConsumeSequences(ctx, r)
}

func (r *Runner) SequenceCallback(
	ctx context.Context,
	sequenceId string,
	eventBundle nats.MessageBundle,
) error {
	logger := r.logger.With().Str("sequence_id", sequenceId).Logger()

	hop, err := dsl.ParseHops(ctx, r.hopsFiles, eventBundle, logger)
	if err != nil {
		return err
	}

	r.logger.Debug().Msg("Successfully parsed hops file")

	// TODO: Run all sensors concurrently via goroutines
	var sensorErrors error
	for i := range hop.Ons {
		sensor := &hop.Ons[i]
		err := r.dispatchCalls(ctx, sensor, sequenceId, logger)
		if err != nil {
			sensorErrors = multierror.Append(sensorErrors, err)
		}
	}

	return sensorErrors
}

func (r *Runner) dispatchCalls(ctx context.Context, sensor *dsl.OnAST, sequenceId string, logger zerolog.Logger) error {
	var wg sync.WaitGroup
	var errs error

	logger = logger.With().Str("sensor", sensor.Slug).Logger()
	logger.Info().Msg("Running sensor calls")

	numTasks := len(sensor.Calls)
	errorchan := make(chan error, numTasks)

	for _, call := range sensor.Calls {
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

func (r *Runner) dispatchCall(ctx context.Context, wg *sync.WaitGroup, call dsl.CallAST, sequenceId string, errorchan chan<- error, logger zerolog.Logger) {
	defer wg.Done()

	app, handler, found := strings.Cut(call.TaskType, "_")
	if !found {
		errorchan <- fmt.Errorf("Unable to parse app/handler from call %s", call.Name)
		return
	}

	_, err := r.natsClient.Publish(ctx, call.Inputs, nats.ChannelRequest, sequenceId, call.Slug, app, handler)

	// At the time of writing, the go client does not contain an error matching
	// the 'maximum massages per subject exceeded' error.
	// We match on the code here instead - @manterfield
	if err != nil && !strings.Contains(err.Error(), "err_code=10077") {
		errorchan <- err
		return
	}

	if err == nil {
		logger.Info().Msgf("Dispatched call: %s", call.Slug)
	}

	errorchan <- nil
}
