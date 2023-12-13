package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

const (
	appUsage = "Hops CLI for creating and running automations"

	appDescription = `Hops is a tool for creating, sharing and running automations.
	
Create awesome DevEx, automate common ops tasks, or allow cross-functional teams to self-serve.`

	commonFlagCategory = "Global"

	configFlagName = "config"
)

func Run() error {
	app, err := initCliApp()
	if err != nil {
		return err
	}
	return app.Run(os.Args)
}

func initCliApp() (*cli.App, error) {
	commonFlags, err := initCommonFlags()
	if err != nil {
		return nil, err
	}

	app := &cli.App{
		Name:        "hops",
		Usage:       appUsage,
		Description: appDescription,
		Commands: []*cli.Command{
			initStartCommand(commonFlags),
			initConfigCommand(commonFlags),
		},
	}

	return app, nil
}

func initCommonFlags() ([]cli.Flag, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return nil, fmt.Errorf("Unable to determine home directory: '%w'", err)
	}
	defaultRootDir := path.Join(homeDir, ".hops")
	defaultConfigPath := path.Join(defaultRootDir, "config.yaml")
	defaultKeyFilePath := path.Join(defaultRootDir, "hiphops.key")

	commonFlags := []cli.Flag{
		&cli.StringFlag{
			Name:     configFlagName,
			Aliases:  []string{"c"},
			Usage:    "Config file path for configuring the hops server/instance",
			Value:    defaultConfigPath,
			Category: commonFlagCategory,
		},
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:     "hops",
				Aliases:  []string{"H"},
				Usage:    "Path to dir containing subdirectories of *.hops configs (and additional files)",
				Value:    defaultRootDir,
				Category: commonFlagCategory,
			},
		),
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:     "keyfile",
				Usage:    "Path to the hiphops key",
				Value:    defaultKeyFilePath,
				Category: commonFlagCategory,
			},
		),
		altsrc.NewBoolFlag(
			&cli.BoolFlag{
				Name:     "debug",
				Aliases:  []string{"d"},
				Usage:    "Sets log level to debug + pretty format",
				Category: commonFlagCategory,
			},
		),
	}

	return commonFlags, nil
}
