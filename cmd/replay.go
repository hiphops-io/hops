/*
Copyright © 2023 Tom Manterfield <tom@hiphops.io>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/internal/orchestrator"
	"github.com/hiphops-io/hops/internal/setup"
	undist "github.com/hiphops-io/hops/undistribute"
)

const (
	replayShortDesc = "Replay a sequence of events against your local hops"
	replayLongDesc  = `Useful for creating and debugging workflows,
use replay to pull a sequence of events and replay them against this instance.
Each replay will use a seperate, segregated lease. You can replay repeatedly whilst avoiding task de-duplication
Example use:
	
hops replay -e SEQUENCE_ID`
)

// replayCmd replays a sequence of events against the local hops instance
func replayCmd(ctx context.Context) *cobra.Command {
	replayCmd := &cobra.Command{
		Use:   "replay",
		Short: replayShortDesc,
		Long:  replayLongDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := cmdLogger()

			appdirs, err := setup.NewAppDirs(viper.GetString("rootdir"))
			if err != nil {
				logger.Error().Err(err).Msg("Failed to create app dirs")
				return err
			}

			keyFile, err := setup.NewKeyFile(viper.GetString("keyfile"))
			if err != nil {
				logger.Error().Err(err).Msg("Failed to load keyfile")
				return err
			}

			replayRunner, lease, err := setupReplay(
				ctx,
				appdirs,
				keyFile,
				viper.GetString("hops"),
				viper.GetString("event"),
				viper.GetBool("debug"),
				logger,
			)
			defer lease.Close()

			if err := replay(
				ctx,
				replayRunner,
				lease,
				logger,
			); err != nil {
				logger.Error().Err(err).Msg("Replay failed")
				return err
			}

			return nil
		},
	}

	replayCmd.Flags().StringP("event", "e", "", "[REQUIRED] The event sequence ID to replay - usually the hash of the source event")
	viper.BindPFlag("event", replayCmd.Flags().Lookup("event"))
	replayCmd.MarkFlagRequired("event")

	return replayCmd
}

func setupReplay(ctx context.Context, appdirs setup.AppDirs, keyFile setup.KeyFile, hopsFilePath string, eventSequence string, debug bool, logger zerolog.Logger) (*orchestrator.Runner, *undist.Lease, error) {
	hops, _, err := dsl.ReadHopsFiles(hopsFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to read hops file: %w", err)
	}

	replayId := uuid.NewString()

	leaseConf := undist.LeaseConfig{
		NatsUrl:             keyFile.NatsUrl(),
		StreamName:          keyFile.AccountId,
		LeaseDurability:     undist.Ephemeral,
		SourceSubject:       "*",
		SourceFilter:        fmt.Sprintf("%s.>", eventSequence),
		SourceDurability:    undist.Ephemeral,
		SourceDeliverPolicy: undist.DeliverAllPolicy,
		SourceConsumerName:  replayId,
		RootDir:             appdirs.WorkspaceDir,
		Seed:                []byte(replayId),
	}

	runner, lease, err := orchestrator.InitLeasedRunner(ctx, leaseConf, appdirs, hops, logger)

	return runner, lease, err
}

func replay(ctx context.Context, runner *orchestrator.Runner, lease *undist.Lease, logger zerolog.Logger) error {
	logger.Info().Msgf("Replaying events - Replay ID: %s", lease.Config().SourceConsumerName)

	callback := orchestrator.CreateRunnerCallback(runner, lease.Dir(), logger)
	err := lease.Consume(ctx, callback)
	if err != nil {
		return err
	}

	return nil
}
