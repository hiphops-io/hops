package cmd

import (
	"context"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	"github.com/hiphops-io/hops/internal/hops"
	"github.com/hiphops-io/hops/logs"
)

const (
	serveCategory = "Serve"

	startDescription = `Start Hiphops

Basic usage: 
	hops start

Start individual components e.g. the runner:
	hops start --serve-runner

Or any combination:
	hops start --serve-console --serve-runner
`
)

func initStartCommand(commonFlags []cli.Flag) *cli.Command {
	startFlags := initStartFlags(commonFlags)
	before := func(ctx *cli.Context) error {
		// Handle '~' exapansion for config file
		configFilePath, err := homedir.Expand(ctx.String(configFlagName))
		if err != nil {
			return err
		}

		// Don't fail if config file doesn't exist
		if _, err := os.Stat(configFilePath); err == nil {
			inputSource, err := altsrc.NewYamlSourceFromFile(configFilePath)
			if err != nil {
				return err
			}
			return altsrc.ApplyInputSourceValues(ctx, inputSource, startFlags)
		} else if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}

	return &cli.Command{
		Name:        "start",
		Usage:       "Start Hiphops",
		Description: startDescription,
		Before:      before,
		Flags:       startFlags,
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			logger := logs.InitLogger(c.Bool("debug"))

			// Handle '~' exapansion for config files
			hopsPath, err := homedir.Expand(c.String("hops"))
			if err != nil {
				return err
			}
			kubeconfigPath, err := homedir.Expand(c.String("kubeconfig"))
			if err != nil {
				return err
			}
			keyfilePath, err := homedir.Expand(c.String("keyfile"))
			if err != nil {
				return err
			}

			hopsServer := &hops.HopsServer{
				Console: hops.Console{
					Address: c.String("address"),
					Serve:   c.Bool("serve-console"),
				},
				HopsPath: hopsPath,
				HTTPApp: hops.HTTPApp{
					Serve: c.Bool("serve-httpapp"),
				},
				K8sApp: hops.K8sApp{
					KubeConfig:  kubeconfigPath,
					PortForward: c.Bool("portforward"),
					Serve:       c.Bool("serve-k8sapp"),
				},
				KeyFilePath: keyfilePath,
				Logger:      logger,
				ReplayEvent: c.String("replay-event"),
				Runner: hops.Runner{
					Serve: c.Bool("serve-runner"),
				},
			}

			return hopsServer.Start(ctx)
		},
	}
}

func initStartFlags(commonFlags []cli.Flag) []cli.Flag {
	startFlags := []cli.Flag{
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:    "address",
				Aliases: []string{"a", "console.address"},
				Usage:   "Address to serve console/API on",
				Value:   "127.0.0.1:8916",
			},
		),
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:  "replay-event",
				Usage: "Replay a specific source event against current hops configs. Takes a source event ID",
			},
		),
		altsrc.NewBoolFlag(
			&cli.BoolFlag{
				Name:     "serve-console",
				Aliases:  []string{"console.serve"},
				Usage:    "Whether to start the console",
				Category: serveCategory,
				Value:    true,
			},
		),
		altsrc.NewBoolFlag(
			&cli.BoolFlag{
				Name:     "serve-runner",
				Aliases:  []string{"runner.serve"},
				Usage:    "Whether to start the workflow runner",
				Category: serveCategory,
				Value:    true,
			},
		),
		altsrc.NewBoolFlag(
			&cli.BoolFlag{
				Name:     "serve-httpapp",
				Aliases:  []string{"http.serve"},
				Usage:    "Whether to start the http app",
				Category: serveCategory,
				Value:    true,
			},
		),
		altsrc.NewBoolFlag(
			&cli.BoolFlag{
				Name:     "serve-k8sapp",
				Aliases:  []string{"k8s.serve"},
				Usage:    "Whether to start the k8s app",
				Category: serveCategory,
			},
		),
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:     "kubeconfig",
				Aliases:  []string{"k", "k8s.kubeconfig"},
				Usage:    "Path to the kubeconfig file for automating k8s (default will use kubernetes standard search locations)",
				Category: "Kubernetes App",
				Value:    "",
			},
		),
		altsrc.NewBoolFlag(
			&cli.BoolFlag{
				Name:     "portforward",
				Aliases:  []string{"k8s.portforward"},
				Usage:    "Whether to auto port-forward, necessary when running outside of a k8s cluster and orchestrating pods",
				Category: "Kubernetes App",
			},
		),
	}

	return append(startFlags, commonFlags...)
}
