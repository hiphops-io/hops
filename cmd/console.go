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

	"github.com/hiphops-io/hops/internal/httpserver"
	"github.com/hiphops-io/hops/internal/setup"
	"github.com/hiphops-io/hops/nats"
)

const (
	consoleShortDesc = "Start the hops console locally"
	consoleLongDesc  = `Start the hops console to interact with the UI.
		
This does *not* start the hops orchestration server.
The console provides access to the hops UI and the backend APIs needed to serve it`
)

// consoleCmd starts the hops console and required APIs
func consoleCmd(ctx context.Context) *cobra.Command {
	consoleCmd := &cobra.Command{
		Use:   "console",
		Short: consoleShortDesc,
		Long:  consoleLongDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := cmdLogger()

			keyFile, err := setup.NewKeyFile(viper.GetString("keyfile"))
			if err != nil {
				logger.Error().Err(err).Msg("Failed to load keyfile")
				return err
			}

			natsClient, err := nats.NewClient(ctx, keyFile.NatsUrl(), keyFile.AccountId)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to start NATS client")
				return err
			}
			defer natsClient.Close()

			if err := console(
				viper.GetString("address"),
				viper.GetString("hops"), // TODO: Replace this with pre-loaded HclFiles (as in start cmd)
				natsClient,
				logger,
			); err != nil {
				logger.Error().Err(err).Msg("Console failed to start")
				return err
			}

			return nil
		},
	}

	return consoleCmd
}

func console(address string, hopsFilePath string, natsClient httpserver.NatsClient, logger zerolog.Logger) error {
	return httpserver.Serve(address, hopsFilePath, natsClient, logger)
}
