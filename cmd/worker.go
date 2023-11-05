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

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/hiphops-io/hops/internal/k8sapp"
	"github.com/hiphops-io/hops/internal/setup"
	"github.com/hiphops-io/hops/logs"
	"github.com/hiphops-io/hops/nats"
	work "github.com/hiphops-io/hops/worker"
)

const (
	workerShortDesc = "Start the hiphops worker & listen for events"
	workerLongDesc  = `Start an instance of a hiphops worker to process events and execute tasks`
)

// workerCmd starts the hops worker, listening for hops tasks and processing them
func workerCmd(ctx context.Context) *cobra.Command {
	workerCmd := &cobra.Command{
		Use:   "worker",
		Short: workerShortDesc,
		Long:  workerLongDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := cmdLogger()
			zlog := logs.NewNatsZeroLogger(logger)

			keyFile, err := setup.NewKeyFile(viper.GetString("keyfile"))
			if err != nil {
				logger.Error().Err(err).Msg("Failed to load keyfile")
				return err
			}

			natsClient, err := nats.NewWorkerClient(ctx, keyFile.NatsUrl(), keyFile.AccountId, "k8s", &zlog)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to start NATS client")
				return err
			}
			defer natsClient.Close()

			if err := worker(
				ctx,
				natsClient,
				viper.GetString("kubeconfig"),
				keyFile.AccountId,
				viper.GetBool("port-forward"),
				logger,
			); err != nil {
				logger.Error().Err(err).Msg("Worker start failed")
				return err
			}

			return nil
		},
	}

	return workerCmd
}

func worker(ctx context.Context, natsClient *nats.Client, kubeConfPath string, accountId string, requiresPortForward bool, logger zerolog.Logger) error {
	logger = logger.With().Str("from", "worker").Logger()

	k8s, err := k8sapp.NewK8sHandler(ctx, natsClient, kubeConfPath, requiresPortForward, logger)
	if err != nil {
		return err
	}

	zlogger := logs.NewNatsZeroLogger(logger)
	worker := work.NewWorker(natsClient, k8s, &zlogger)

	// Blocks until complete or errored
	return worker.Run(ctx)
}
