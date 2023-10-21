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
	undist "github.com/hiphops-io/hops/undistribute"
)

const (
	consoleShortDesc = "Start the hops console locally"
	consoleLongDesc  = `Start the hops console to interact with the UI.
		
This does *not* start the hops workflow server.
The console provides credential helpers (allowing users to manually sign-in and authenticate)
in addition to info on stored secrets.`
)

// consoleCmd starts the hops console and required APIs
func consoleCmd(ctx context.Context) *cobra.Command {
	consoleCmd := &cobra.Command{
		Use:   "console",
		Short: consoleShortDesc,
		Long:  consoleLongDesc,
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

			_, lease, err := setupServer(
				ctx,
				appdirs,
				keyFile.NatsUrl(),
				keyFile.AccountId,
				viper.GetString("hops"),
				logger,
			)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to setup server")
				return err
			}
			defer lease.Close()

			if err := console(
				appdirs,
				viper.GetString("address"),
				viper.GetString("hops"),
				lease,
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

func console(appdirs setup.AppDirs, address string, hopsFilePath string, lease *undist.Lease, logger zerolog.Logger) error {
	return httpserver.Serve(appdirs, address, hopsFilePath, lease, logger)
}
