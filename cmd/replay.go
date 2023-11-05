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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/internal/setup"
	"github.com/hiphops-io/hops/logs"
	"github.com/hiphops-io/hops/nats"
)

const (
	replayShortDesc = "Replay a sequence of events against your local hops"
	replayLongDesc  = `Useful for creating and debugging workflows,
use replay to pull a sequence of events and replay them against this instance.
Each replay will use a seperate, segregated lease. You can replay repeatedly whilst avoiding task de-duplication
Example use:
	
hops replay -e SEQUENCE_ID`
)

// replayCmd replays a source event against the local hops instance
//
// Identical to server, except the NATS Client is configured to use an ephemeral
// consumer with the source event republished under a replay sequence ID
func replayCmd(ctx context.Context) *cobra.Command {
	replayCmd := &cobra.Command{
		Use:   "replay",
		Short: replayShortDesc,
		Long:  replayLongDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := cmdLogger()
			zlog := logs.NewNatsZeroLogger(logger)

			keyFile, err := setup.NewKeyFile(viper.GetString("keyfile"))
			if err != nil {
				logger.Error().Err(err).Msg("Failed to load keyfile")
				return err
			}

			hops, _, err := dsl.ReadHopsFiles(viper.GetString("hops"))
			if err != nil {
				logger.Error().Err(err).Msg("Failed to read hops files")
				return fmt.Errorf("Failed to read hops file: %w", err)
			}

			natsClient, err := nats.NewReplayClient(ctx, keyFile.NatsUrl(), keyFile.AccountId, viper.GetString("event"), &zlog)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to start NATS client")
				return err
			}
			defer natsClient.Close()

			logger.Info().Msgf("Starting replay of sequence: %s", viper.GetString("event"))

			if err := server(
				ctx,
				hops,
				natsClient,
				logger,
			); err != nil {
				logger.Error().Err(err).Msg("Replay failed")
				return err
			}

			return nil
		},
	}

	replayCmd.Flags().StringP("event", "e", "", "[REQUIRED] The event sequence ID to replay")
	viper.BindPFlag("event", replayCmd.Flags().Lookup("event"))
	replayCmd.MarkFlagRequired("event")

	return replayCmd
}
