package hops

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/internal/httpapp"
	"github.com/hiphops-io/hops/internal/httpserver"
	"github.com/hiphops-io/hops/internal/k8sapp"
	"github.com/hiphops-io/hops/internal/plugin"
	"github.com/hiphops-io/hops/internal/runner"
	"github.com/hiphops-io/hops/logs"
	"github.com/hiphops-io/hops/nats"
	"github.com/hiphops-io/hops/worker"
)

// TODO: I'm not super happy with this string based toggle approach as it has
// ended up being overly complex for such a simple capability.
// This was formed due to limitations in the CLI library, but we should probably refactor a bit
// and stop those limitations leaking into the rest of the code.
const (
	ServeConsoleOpt = "console"
	ServeK8sAppOpt  = "k8sapp"
	ServeRunnerOpt  = "runner"
)

type (
	Console struct {
		Address string
		Serve   bool
	}

	HopsServer struct {
		HopsPath    string
		KeyFilePath string
		Logger      zerolog.Logger
		ReplayEvent string
		Console
		HTTPApp
		K8sApp
		Runner
		Plugin
	}

	HTTPApp struct {
		Serve bool
	}

	K8sApp struct {
		KubeConfig  string
		Serve       bool
		PortForward bool
	}

	Runner struct {
		Serve bool
	}

	Plugin struct {
		Serve bool
	}
)

func (h *HopsServer) Start(ctx context.Context) error {
	if !(h.Runner.Serve || h.Console.Serve || h.K8sApp.Serve) {
		return errors.New("No serve functions are enabled. Nothing to do.")
	}

	zlog := logs.NewNatsZeroLogger(h.Logger)

	hops, err := dsl.ReadHopsFilePath(h.HopsPath)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to read hops files")
		return fmt.Errorf("Failed to read hops file: %w", err)
	}

	keyFile, err := nats.NewKeyFile(h.KeyFilePath)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to load keyfile")
		return err
	}

	runnerConsumer := nats.DefaultConsumerName
	clientOpts := []nats.ClientOpt{}
	if h.ReplayEvent != "" {
		runnerConsumer := "replay"
		clientOpts = append(clientOpts, nats.WithReplay(runnerConsumer, h.ReplayEvent))
		h.Logger.Info().Msgf("Replaying source event: %s", h.ReplayEvent)
	} else if h.Runner.Serve {
		clientOpts = append(clientOpts, nats.WithRunner(nats.DefaultConsumerName))
	}

	if h.HTTPApp.Serve {
		clientOpts = append(clientOpts, nats.WithWorker("http"))
	}

	if h.K8sApp.Serve {
		clientOpts = append(clientOpts, nats.WithWorker("k8s"))
	}

	natsClient, err := nats.NewClient(
		keyFile.NatsUrl(),
		keyFile.AccountId,
		&zlog,
		clientOpts...,
	)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to start NATS client")
		return err
	}
	defer natsClient.Close()

	errs := make(chan error, 1)

	if h.Console.Serve {
		go func() {
			err := startConsole(
				h.Console.Address,
				hops.BodyContent,
				natsClient,
				h.Logger,
			)
			if err != nil {
				errs <- err
			}
		}()
	}

	if h.Runner.Serve {
		go func() {
			err := startRunner(
				ctx,
				hops,
				natsClient,
				runnerConsumer,
				h.Logger,
			)
			if err != nil {
				errs <- nil
			}
		}()
	}

	if h.HTTPApp.Serve {
		go func() {
			err := startHTTPApp(
				ctx,
				natsClient,
				h.Logger,
			)
			if err != nil {
				errs <- err
			}
		}()
	}

	if h.K8sApp.Serve {
		go func() {
			err := startK8sApp(
				ctx,
				natsClient,
				h.K8sApp.KubeConfig,
				h.K8sApp.PortForward,
				h.Logger,
			)
			if err != nil {
				errs <- err
			}
		}()
	}

	if h.Plugin.Serve {
		go func() {
			err := startPlugin(
				ctx,
				h.Logger,
			)
			if err != nil {
				errs <- err
			}
		}()
	}

	if err := <-errs; err != nil {
		h.Logger.Error().Err(err).Msg("Start failed")
		return err
	}

	return nil
}

func startConsole(address string, hopsContent *hcl.BodyContent, natsClient httpserver.NatsClient, logger zerolog.Logger) error {
	return httpserver.Serve(address, hopsContent, natsClient, logger)
}

func startRunner(ctx context.Context, hops *dsl.HopsFiles, natsClient *nats.Client, fromConsumer string, logger zerolog.Logger) error {
	logger.Info().Msg("Listening for events")

	runner, err := runner.NewRunner(natsClient, hops, logger)
	if err != nil {
		return err
	}

	return runner.Run(ctx, fromConsumer)
}

func startHTTPApp(ctx context.Context, natsClient *nats.Client, logger zerolog.Logger) error {
	logger = logger.With().Str("from", "httpapp").Logger()

	httpApp, err := httpapp.NewHTTPHandler(ctx, natsClient, logger)
	if err != nil {
		return err
	}

	zlogger := logs.NewNatsZeroLogger(logger)
	worker := worker.NewWorker(natsClient, httpApp, &zlogger)

	// Blocks until complete or errored
	return worker.Run(ctx)
}

func startK8sApp(ctx context.Context, natsClient *nats.Client, kubeConfPath string, requiresPortForward bool, logger zerolog.Logger) error {
	logger = logger.With().Str("from", "k8sapp").Logger()

	k8s, err := k8sapp.NewK8sHandler(ctx, natsClient, kubeConfPath, requiresPortForward, logger)
	if err != nil {
		return err
	}

	// Due to automated config loading, this worker may naturally decide to not work.
	// This will be logged by the worker. We just need to move on.
	if k8s == nil {
		return nil
	}

	zlogger := logs.NewNatsZeroLogger(logger)
	worker := worker.NewWorker(natsClient, k8s, &zlogger)

	// Blocks until complete or errored
	return worker.Run(ctx)
}

func startPlugin(ctx context.Context, logger zerolog.Logger) error {
	logger.Info().Msg("Listening for plugin events")

	return plugin.Run(ctx)
}
