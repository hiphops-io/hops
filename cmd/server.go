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

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/internal/runner"
	"github.com/hiphops-io/hops/logs"
	"github.com/hiphops-io/hops/nats"
)

const (
	serverShortDesc = "Start the hops orchestration server & listen for events"
	serverLongDesc  = `Start an instance of the hops orchestration server to process events and run workflows.
	
Hops can run locally only, or connect with a cluster and share workloads.`
)

// serverCmd starts the hops orchestrator server, listening for and processing new events
func serverCmd(ctx context.Context) *cobra.Command {
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: workerShortDesc,
		Long:  serverLongDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := cmdLogger()
			zlog := logs.NewNatsZeroLogger(logger)

			keyFile, err := nats.NewKeyFile(viper.GetString("keyfile"))
			if err != nil {
				logger.Error().Err(err).Msg("Failed to load keyfile")
				return err
			}

			hops, err := dsl.ReadHopsFilePath(viper.GetString("hops"))
			if err != nil {
				logger.Error().Err(err).Msg("Failed to read hops files")
				return fmt.Errorf("Failed to read hops file: %w", err)
			}

			natsClient, err := nats.NewClient(keyFile.NatsUrl(), keyFile.AccountId, &zlog)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to start NATS client")
				return err
			}
			defer natsClient.Close()

			if err := server(
				ctx,
				hops,
				natsClient,
				nats.DefaultConsumerName,
				logger,
			); err != nil {
				logger.Error().Err(err).Msg("Server start failed")
				return err
			}

			return nil
		},
	}

	return serverCmd
}

func server(ctx context.Context, hops *dsl.HopsFiles, natsClient *nats.Client, fromConsumer string, logger zerolog.Logger) error {
	logger.Info().Msg("Listening for events")

	runner, err := runner.NewRunner(natsClient, hops, logger)
	if err != nil {
		return err
	}

	return runner.Run(ctx, fromConsumer)
}
