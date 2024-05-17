package main

import (
	"context"

	"github.com/oklog/run"
	"github.com/rs/zerolog"
	"github.com/slok/reload"

	"github.com/hiphops-io/hops/config"
	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/internal/httpserver"
	"github.com/hiphops-io/hops/internal/runner"
	"github.com/hiphops-io/hops/logs"
	"github.com/hiphops-io/hops/nats"
)

type HopsServer struct {
	logger        zerolog.Logger
	natsClient    *nats.Client
	reloadManager reload.Manager
	runGroup      run.Group
}

func Start(cfg *config.Config) error {
	// TODO: Ensure errors are gathered and logged at the top level
	ctx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	h := &HopsServer{
		logger: logs.InitLogger(cfg.Dev),
	}

	if cfg.Dev {
		h.reloadManager = reload.NewManager()
	}

	close, err := h.startNATS(cfg)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to start NATS client")
		return err
	}
	defer close()

	automationsLoader, err := h.startAutomationsLoader(ctx, cfg)
	if err != nil {
		h.logger.Error().Err(err).Msg("Start failed")
		return err
	}

	if err := h.startRunner(ctx, cfg, automationsLoader); err != nil {
		return err
	}

	h.startHTTPServer(ctx, h.natsClient)

	return h.runGroup.Run()
}

func (h *HopsServer) startAutomationsLoader(ctx context.Context, cfg *config.Config) (*dsl.AutomationsLoader, error) {
	automationsLoader, err := dsl.NewAutomationsLoader(cfg.FlowsPath(), cfg.Dev)
	if !cfg.Dev {
		return automationsLoader, err
	}

	h.reloadManager.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		err := automationsLoader.Reload(ctx, true)
		if err != nil {
			h.logger.Warn().Msgf("Hops files could not be reloaded: %s", err.Error())
			return nil
		}

		h.logger.Info().Msg("Hops files reloaded")
		return nil
	}))

	{
		dirNotifier, err := dsl.NewDirNotifier(cfg.FlowsPath())
		if err != nil {
			return nil, err
		}

		// Add file watcher based reload notifier.
		h.reloadManager.On(dirNotifier.Notifier(ctx))

		ctx, cancel := context.WithCancel(ctx)
		h.runGroup.Add(
			func() error {
				// Block forever until the watcher stops.
				h.logger.Info().Msgf("Watching %s for changes", cfg.FlowsPath())
				<-ctx.Done()
				return nil
			},
			func(_ error) {
				h.logger.Info().Msg("Stopping hops file watcher")
				dirNotifier.Close()
				cancel()
			},
		)
	}

	{
		ctx, cancel := context.WithCancel(ctx)
		h.runGroup.Add(
			func() error {
				return h.reloadManager.Run(ctx)
			},
			func(_ error) {
				h.logger.Info().Msg("Auto-reloading cancelled")
				cancel()
			},
		)
	}

	return automationsLoader, nil
}

func (h *HopsServer) startHTTPServer(ctx context.Context, natsClient *nats.Client) {
	server := httpserver.NewHTTPServer(":8080", h.natsClient)

	h.runGroup.Add(
		func() error {
			return server.Serve()
		},
		func(_ error) {
			server.Shutdown(ctx)
		},
	)
}

func (h *HopsServer) startNATS(cfg *config.Config) (func(), error) {
	logger := h.logger.Level(zerolog.WarnLevel)
	zlog := logs.NewNatsZeroLogger(logger)

	server, err := nats.NewNatsServer(
		cfg.NATSConfigPath(),
		cfg.Dev,
		&zlog,
		nats.WithDataDirOpt(cfg.Runner.DataDir),
	)
	if err != nil {
		return nil, err
	}

	natsClient, err := nats.NewClient(server.URL())
	if err != nil {
		defer server.Close()
		h.logger.Error().Err(err).Msg("Failed to start NATS client")
		return nil, err
	}

	close := func() {
		defer server.Close()
		defer natsClient.Close()
	}

	h.natsClient = natsClient

	return close, nil
}

func (h *HopsServer) startRunner(ctx context.Context, cfg *config.Config, automationsLoader *dsl.AutomationsLoader) error {
	if !cfg.Runner.Serve {
		return nil
	}

	consumer, err := h.natsClient.RunnerConsumer(ctx)
	if err != nil {
		return err
	}

	runner, err := runner.NewRunner(h.natsClient, automationsLoader, consumer, h.logger)
	if err != nil {
		return err
	}

	if cfg.Dev {
		h.reloadManager.Add(10, reload.ReloaderFunc(func(ctx context.Context, id string) error {
			return runner.Reload(ctx)
		}))
	}

	ctx, cancel := context.WithCancel(ctx)
	h.runGroup.Add(
		func() error {
			h.logger.Info().Msg("Hops is ready")
			return runner.Run(ctx)
		},
		func(_ error) {
			cancel()
		},
	)

	return nil
}
