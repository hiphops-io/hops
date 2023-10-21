package workflow

import (
	"context"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/internal/setup"
	undist "github.com/hiphops-io/hops/undistribute"
)

func InitLeasedRunner(ctx context.Context, leaseConf undist.LeaseConfig, appdirs setup.AppDirs, hopses dsl.HclFiles, logger zerolog.Logger) (*Runner, *undist.Lease, error) {
	lease, err := undist.NewLease(ctx, leaseConf)
	if err != nil {
		return nil, nil, err
	}

	runner, err := NewRunner(lease, hopses, logger)

	return runner, lease, err
}

// CreateRunnerCallback is a helper method to return a consume callback that runs workflow.Runner.Run
//
// TODO: Properly handle the two error states rather than just Nak'ing them
func CreateRunnerCallback(runner *Runner, stateDir string, logger zerolog.Logger) func(m jetstream.Msg) {
	nakDelay := 5 * time.Second

	callback := func(m jetstream.Msg) {
		ctx := context.Background()

		sequenceId, eventBundle, err := undist.UpFetchState(stateDir, m)
		if err != nil {
			m.NakWithDelay(nakDelay)
			return
		}

		err = runner.Run(ctx, sequenceId, eventBundle)
		if err != nil {
			logger.Error().Err(err).Msgf("Failed to run workflow for: %s", m.Subject())
			m.NakWithDelay(nakDelay)
			return
		}

		err = undist.DoubleAck(ctx, m)
		if err != nil {
			logger.Error().Err(err).Msgf("Failed to acknowledge workflow event for: %s", m.Subject())
			return
		}
	}

	return callback
}
