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
	"os"
	"path"

	"github.com/hiphops-io/hops/logs"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	envPrefix = "HOPS"
)

const (
	rootShortDesc = "Hops CLI for building automation pipelines"
	rootLongDesc  = `Hops is a tool for creating advanced local and remote automation pipelines.

Create sophisticated CI/CD flows, power-up your GitOps, or create awesome dev tooling.`
)

func Execute() error {
	rootCmd := rootCmd()
	cobra.OnInitialize(func() {
		initConfig()
	})
	ctx := context.Background()

	startCmd := startCmd(ctx)
	startCmd.AddCommand(serverCmd(ctx), consoleCmd(ctx), workerCmd(ctx))

	configCmd := configCmd()
	configCmd.AddCommand(addkeyCmd())

	rootCmd.AddCommand(startCmd, replayCmd(ctx), configCmd)
	return rootCmd.Execute()
}

func rootCmd() cobra.Command {
	rootCmd := cobra.Command{
		Use:   "hops",
		Short: rootShortDesc,
		Long:  rootLongDesc,
	}

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path (default is $HOME/.hops/config.yaml)")
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	rootCmd.PersistentFlags().StringP("rootdir", "r", "", "path to the hops root dir (default is $HOME/.hops/)")
	viper.BindPFlag("rootdir", rootCmd.PersistentFlags().Lookup("rootdir"))

	rootCmd.PersistentFlags().StringP("hops", "H", "", "Path to root hops config or dir of .hops configs (default is $HOPS_ROOTDIR/)")
	viper.BindPFlag("hops", rootCmd.PersistentFlags().Lookup("hops"))

	rootCmd.PersistentFlags().String("keyfile", "", "Path to the hiphops key (default is $HOPS_ROOTDIR/hiphops.key)")
	viper.BindPFlag("keyfile", rootCmd.PersistentFlags().Lookup("keyfile"))

	rootCmd.PersistentFlags().StringP("kubeconfig", "k", "", "Path to the kubeconfig file for automating k8s (default is $HOME/.kube/config)")
	viper.BindPFlag("nats", rootCmd.PersistentFlags().Lookup("nats"))

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "sets log level to debug + pretty format")
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	rootCmd.PersistentFlags().Bool("port-forward", false, "whether to auto port-forward, necessary when running outside of a k8s cluster and orchestrating pods")
	viper.BindPFlag("port-forward", rootCmd.PersistentFlags().Lookup("port-forward"))

	return rootCmd
}

func cmdLogger() zerolog.Logger {
	logger := logs.InitLogger(viper.GetBool("debug"))
	logger.Debug().Msgf("Config path: %s", viper.ConfigFileUsed())
	return logger
}

// initConfig reads in a config file and ENV variables if set.
func initConfig() error {
	homeDir, err := homedir.Dir()
	cobra.CheckErr(err)
	hopsDir := path.Join(homeDir, ".hops")

	if cfgFile == "" {
		viper.SetConfigFile(path.Join(hopsDir, "config.yaml"))
	}

	viper.SetDefault("rootdir", hopsDir)
	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	err = viper.ReadInConfig()
	if err != nil && cfgFile != "" {
		return fmt.Errorf("Config file not found: %s", viper.GetString("config"))
	}

	// Ensure the root dir exists
	if hopsRootDir := viper.GetString("rootdir"); hopsRootDir != "" {
		err := os.MkdirAll(hopsRootDir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// file arg's default is dependent on the value of "rootdir", so we set it here
	if hopsFilePath := viper.GetString("hops"); hopsFilePath == "" {
		hopsRootDir := viper.GetString("rootdir")
		viper.SetDefault("hops", hopsRootDir)
	}

	// keyfile arg's default is also dependent on the value of "rootdir"
	if keyfile := viper.GetString("keyfile"); keyfile == "" {
		hopsRootDir := viper.GetString("rootdir")
		viper.SetDefault("keyfile", path.Join(hopsRootDir, "hiphops.key"))
	}

	return nil
}
