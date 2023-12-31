package cmd

import (
	"context"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	"github.com/hiphops-io/hops/internal/hops"
	"github.com/hiphops-io/hops/logs"
)

const (
	k8sCategory   = "K8s App"
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
	before := optionalYamlSrc(startFlags)

	return &cli.Command{
		Name:        "start",
		Usage:       "Start Hiphops",
		Description: startDescription,
		Before:      before,
		Flags:       startFlags,
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			logger := logs.InitLogger(c.Bool("debug"))

			hopsServer := &hops.HopsServer{
				HTTPServerConf: hops.HTTPServerConf{
					Address: c.String("address"),
					Serve:   c.Bool("serve-console"),
				},
				HopsPath: c.String("hops"),
				HTTPAppConf: hops.HTTPAppConf{
					Serve: c.Bool("serve-httpapp"),
				},
				K8sAppConf: hops.K8sAppConf{
					KubeConfig:  c.String("kubeconfig"),
					PortForward: c.Bool("portforward"),
					Serve:       c.Bool("serve-k8sapp"),
				},
				KeyFilePath: c.String("keyfile"),
				Logger:      logger,
				ReplayEvent: c.String("replay-event"),
				RunnerConf: hops.RunnerConf{
					Serve: c.Bool("serve-runner"),
				},
				Watch: c.Bool("watch"),
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
				Category: k8sCategory,
				Value:    "",
				Action:   expandHomePath("kubeconfig"),
			},
		),
		altsrc.NewBoolFlag(
			&cli.BoolFlag{
				Name:     "portforward",
				Aliases:  []string{"k8s.portforward"},
				Usage:    "Whether to auto port-forward, necessary when running outside of a k8s cluster and orchestrating pods",
				Category: k8sCategory,
			},
		),
		altsrc.NewBoolFlag(
			&cli.BoolFlag{
				Name:  "watch",
				Usage: "Auto reload on change to the given hops directory",
			},
		),
	}

	return append(startFlags, commonFlags...)
}
