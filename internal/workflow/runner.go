package workflow

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/undistribute"
	undist "github.com/hiphops-io/hops/undistribute"
)

type Runner struct {
	logger      zerolog.Logger
	hopsFiles   dsl.HclFiles
	secretstore SecretStoreReader
	lease       LeasePublisher
}

type LeasePublisher interface {
	Publish(context.Context, undist.Channel, string, string, []byte) (*jetstream.PubAck, error)
}

type SecretStoreReader interface {
	InterpolateSecrets(data []byte) ([]byte, error)
}

func NewRunner(lease LeasePublisher, hopsFiles dsl.HclFiles, logger zerolog.Logger) (*Runner, error) {
	runner := &Runner{
		logger:    logger,
		hopsFiles: hopsFiles,
		lease:     lease,
	}

	return runner, nil
}

func (r *Runner) Run(
	ctx context.Context,
	sequenceId string,
	eventBundle map[string][]byte,
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
		err := r.dispatchTasks(ctx, sensor, sequenceId, logger)
		if err != nil {
			sensorErrors = multierror.Append(sensorErrors, err)
		}
	}

	return sensorErrors
}

func (r *Runner) dispatchTasks(ctx context.Context, sensor *dsl.OnAST, sequenceId string, logger zerolog.Logger) error {
	var wg sync.WaitGroup
	var errs error

	logger = logger.With().Str("sensor", sensor.Slug).Logger()
	logger.Info().Msg("Running sensor tasks")

	numTasks := len(sensor.Calls)
	errorchan := make(chan error, numTasks)

	for _, task := range sensor.Calls {
		task := task
		wg.Add(1)
		go r.dispatchTask(ctx, &wg, task, sequenceId, errorchan, logger)
	}

	wg.Wait()
	close(errorchan)

	for err := range errorchan {
		errs = errors.Join(errs, err)
	}

	return errs
}

func (r *Runner) dispatchTask(ctx context.Context, wg *sync.WaitGroup, task dsl.CallAST, sequenceId string, errorchan chan<- error, logger zerolog.Logger) {
	defer wg.Done()

	_, err := r.lease.Publish(ctx, undistribute.Request, sequenceId, task.Slug, task.Inputs)

	// At the time of writing, the go client does not contain an error matching
	// the 'maximum massages per subject exceeded' error. We match on the code here instead
	if err != nil && !strings.Contains(err.Error(), "err_code=10077") {
		errorchan <- err
		return
	}

	if err == nil {
		logger.Info().Msgf("Dispatched task: %s", task.Slug)
	}

	errorchan <- nil
}
