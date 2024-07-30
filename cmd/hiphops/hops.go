package main

import (
	"context"

	"github.com/oklog/run"
	"github.com/rs/zerolog"
	"github.com/slok/reload"

	"github.com/hiphops-io/hops/config"
	"github.com/hiphops-io/hops/internal/httpserver"
	"github.com/hiphops-io/hops/internal/runner"
	"github.com/hiphops-io/hops/logs"
	"github.com/hiphops-io/hops/markdown"
	"github.com/hiphops-io/hops/nats"
)

type (
	HopsServer struct {
		logger     zerolog.Logger
		natsClient *nats.Client
		runGroup   run.Group
	}

	Reloader func(ctx context.Context) error
)

func Start(cfg *config.Config) error {
	// TODO: Ensure errors are gathered and logged at the top level
	ctx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	h := &HopsServer{
		logger: logs.InitLogger(cfg.Dev),
	}

	close, err := h.startNATS(cfg)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to start NATS client")
		return err
	}
	defer close()

	runnerReload, err := h.initRunner(ctx, cfg)
	if err != nil {
		return err
	}

	h.startHTTPServer(ctx)

	if cfg.Dev {
		if err := h.startReloader(ctx, cfg, runnerReload); err != nil {
			h.logger.Error().Err(err).Msg("Failed to watch reloader")
			return err
		}
	}

	return h.runGroup.Run()
}

func (h *HopsServer) startReloader(ctx context.Context, cfg *config.Config, reloaders ...Reloader) error {
	reloadManager := reload.NewManager()

	for _, r := range reloaders {
		reloadManager.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
			err := r(ctx)
			if err != nil {
				h.logger.Warn().Msgf("Unable to reload: %s", err.Error())
				return nil
			}

			h.logger.Info().Msg("Flows reloaded")
			return nil
		}))
	}

	notifer, err := NewDirNotifier(cfg.FlowsPath(), h.logger)
	if err != nil {
		return err
	}

	notifer.NotifyReload(ctx, &reloadManager, &h.runGroup)

	reloadManager.On(notifer.Notifier(ctx))

	{
		ctx, cancel := context.WithCancel(ctx)
		h.runGroup.Add(
			func() error {
				return reloadManager.Run(ctx)
			},
			func(_ error) {
				h.logger.Info().Msg("Auto-reloading cancelled")
				cancel()
			},
		)
	}

	return nil
}

func (h *HopsServer) startHTTPServer(ctx context.Context) {
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
	logger := h.logger.Level(zerolog.InfoLevel)
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

	natsClient, err := nats.NewClient(server.URL(), "")
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

func (h *HopsServer) initRunner(ctx context.Context, cfg *config.Config) (Reloader, error) {
	flowReader := markdown.NewFlowReader(cfg.FlowsPath())

	consumer, err := h.natsClient.RunnerConsumer(ctx)
	if err != nil {
		return nil, err
	}

	runner, err := runner.NewRunner(h.natsClient, flowReader, consumer, h.logger)
	if err != nil {
		return nil, err
	}

	// if cfg.Dev {
	// 	h.reloadManager.Add(10, reload.ReloaderFunc(func(ctx context.Context, id string) error {
	// 		return runner.Reload(ctx)
	// 	}))
	// }

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

	return runner.Load, nil
}
