package cmd

import (
	"os"
	"path"

	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

const (
	appUsage = "Hops CLI for creating and running automations"

	appDescription = `Hops is a tool for creating, sharing and running automations.`

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

// expandHomePath is a shared util function to expand flags that are paths with ~ in them
func expandHomePath(flagName string) func(c *cli.Context, val string) error {
	return func(c *cli.Context, val string) error {
		flagValue := c.String(flagName)
		if flagValue == "" {
			return nil
		}

		expandedPath, err := homedir.Expand(flagValue)
		if err != nil {
			return err
		}

		c.Set(flagName, expandedPath)
		return nil
	}
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
			initValidateCommand(commonFlags),
		},
	}

	return app, nil
}

func initCommonFlags() ([]cli.Flag, error) {
	defaultRootDir, err := homedir.Expand("~/.hops")
	if err != nil {
		return nil, err
	}
	defaultConfigPath := path.Join(defaultRootDir, "config.yaml")
	defaultKeyFilePath := path.Join(defaultRootDir, "hiphops.key")

	commonFlags := []cli.Flag{
		&cli.StringFlag{
			Name:     configFlagName,
			Aliases:  []string{"c"},
			Usage:    "Config file path for configuring the hops server/instance",
			Value:    defaultConfigPath,
			Category: commonFlagCategory,
			Action:   expandHomePath(configFlagName),
		},
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:     "hops",
				Aliases:  []string{"H"},
				Usage:    "Path to dir containing hiphops automations",
				Value:    defaultRootDir,
				Category: commonFlagCategory,
				Action:   expandHomePath("hops"),
			},
		),
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:     "keyfile",
				Usage:    "Path to the hiphops key",
				Value:    defaultKeyFilePath,
				Category: commonFlagCategory,
				Action:   expandHomePath("keyfile"),
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

// optionalYamlSrc is a shared util function to _optionally_ load config from yaml file
// silently continuing if the file is not found
func optionalYamlSrc(flags []cli.Flag) func(*cli.Context) error {
	return func(c *cli.Context) error {
		configFilePath, err := homedir.Expand(c.String(configFlagName))
		if err != nil {
			return err
		}

		// Succeed if no config file
		if _, err := os.Stat(configFilePath); err == nil {
			inputSource, err := altsrc.NewYamlSourceFromFile(configFilePath)
			if err != nil {
				return err
			}
			return altsrc.ApplyInputSourceValues(c, inputSource, flags)
		} else if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
}
