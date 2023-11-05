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
	startShortDesc = "Start hops"
	startLongDesc  = `Start the hops orchestration server, worker, & console in one instance.

Orchestration server, console, and worker can be started independently with subcommands:

hops start console

hops start server

hops start worker
	`
)

// startCmd starts the hops orchestration server, listening for and processing new events
func startCmd(ctx context.Context) *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: workerShortDesc,
		Long:  serverLongDesc,
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

			natsClient, err := nats.NewClient(ctx, keyFile.NatsUrl(), keyFile.AccountId, &zlog)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to start NATS client")
				return err
			}
			defer natsClient.Close()

			natsWorkerClient, err := nats.NewWorkerClient(ctx, keyFile.NatsUrl(), keyFile.AccountId, "k8s", &zlog)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to start NATS worker client")
				return err
			}
			defer natsWorkerClient.Close()

			errs := make(chan error, 1)

			go func() {
				errs <- console(
					viper.GetString("address"),
					viper.GetString("hops"), // TODO: Replace with hops HclFiles loaded above
					natsClient,
					logger,
				)
			}()

			go func() {
				errs <- server(
					ctx,
					hops,
					natsClient,
					logger,
				)
			}()

			go func() {
				errs <- worker(
					ctx,
					natsWorkerClient,
					viper.GetString("kubeconfig"),
					keyFile.AccountId,
					viper.GetBool("port-forward"),
					logger,
				)
			}()

			if err := <-errs; err != nil {
				logger.Error().Err(err).Msg("Start failed")
				return err
			}

			return nil
		},
	}

	startCmd.PersistentFlags().StringP("address", "a", "127.0.0.1:8916", "address to listen on")
	viper.BindPFlag("address", startCmd.PersistentFlags().Lookup("address"))

	return startCmd
}
