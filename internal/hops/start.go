package hops

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oklog/run"
	"github.com/rs/zerolog"
	"github.com/slok/reload"

	"github.com/hiphops-io/hops/internal/httpapp"
	"github.com/hiphops-io/hops/internal/k8sapp"
	"github.com/hiphops-io/hops/logs"
	"github.com/hiphops-io/hops/nats"
	"github.com/hiphops-io/hops/worker"
)

type (
	HTTPServerConf struct {
		Address string
		Serve   bool
	}

	HopsServer struct {
		HopsPath      string
		KeyFilePath   string
		Logger        zerolog.Logger
		ReplayEvent   string
		Watch         bool
		reloadManager reload.Manager
		runGroup      run.Group

		HTTPServerConf
		HTTPAppConf
		K8sAppConf
		RunnerConf
	}

	HTTPAppConf struct {
		Serve bool
	}

	K8sAppConf struct {
		KubeConfig  string
		PortForward bool
		Serve       bool
	}

	RunnerConf struct {
		Serve bool
	}
)

func (h *HopsServer) Start(ctx context.Context) error {
	ctx, rootCancel := context.WithCancel(ctx)
	defer rootCancel()

	if !(h.RunnerConf.Serve || h.HTTPServerConf.Serve || h.K8sAppConf.Serve || h.HTTPAppConf.Serve) {
		return errors.New("No components are enabled. Nothing to do.")
	}

	if h.Watch {
		h.reloadManager = reload.NewManager()
	}

	natsClient, err := h.startNATSClient()
	if natsClient != nil {
		defer natsClient.Close()
	}
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to start NATS client")
		return err
	}

	hopsLoader, err := NewHopsFileLoader(h.HopsPath)
	if err != nil {
		err := fmt.Errorf("Failed to read hops files: %w", err)
		h.Logger.Error().Err(err).Msg("Start failed")
		return err
	}

	err = h.startHTTPServer(hopsLoader, natsClient)
	if err != nil {
		return err
	}

	err = h.startRunner(ctx, hopsLoader, natsClient)
	if err != nil {
		return err
	}

	err = h.startHTTPApp(ctx, natsClient)
	if err != nil {
		return err
	}

	err = h.startK8sApp(ctx, natsClient)
	if err != nil {
		return err
	}

	err = h.startReloader(ctx, hopsLoader)
	if err != nil {
		return err
	}

	return h.runGroup.Run()
}

func (h *HopsServer) startHTTPApp(ctx context.Context, natsClient *nats.Client) error {
	if !h.HTTPAppConf.Serve {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	start := func() error {
		logger := h.Logger.With().Str("from", "httpapp").Logger()

		httpApp, err := httpapp.NewHTTPHandler(ctx, natsClient, logger)
		if err != nil {
			return err
		}

		zlogger := logs.NewNatsZeroLogger(logger)
		worker := worker.NewWorker(natsClient, httpApp, &zlogger)

		// Blocks until complete or errored
		return worker.Run(ctx)
	}

	h.runGroup.Add(
		func() error {
			return start()
		},
		func(_ error) {
			cancel()
		},
	)

	return nil
}

func (h *HopsServer) startHTTPServer(hopsLoader *HopsFileLoader, natsClient *nats.Client) error {
	if !h.HTTPServerConf.Serve {
		return nil
	}

	httpServer, err := NewHTTPServer(h.Address, hopsLoader, natsClient, h.Logger)
	if err != nil {
		return err
	}

	if h.Watch {
		h.reloadManager.Add(10, reload.ReloaderFunc(func(ctx context.Context, id string) error {
			return httpServer.Reload(ctx)
		}))
	}

	h.runGroup.Add(
		func() error {
			return httpServer.Serve()
		},
		func(_ error) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := httpServer.Shutdown(ctx)
			if err != nil {
				h.Logger.Error().Err(err).Msg("Unable to shut down http server")
			}
		},
	)

	return nil
}

func (h *HopsServer) startK8sApp(ctx context.Context, natsClient *nats.Client) error {
	if !h.K8sAppConf.Serve {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)

	start := func() error {
		logger := h.Logger.With().Str("from", "k8sapp").Logger()

		k8s, err := k8sapp.NewK8sHandler(ctx, natsClient, h.K8sAppConf.KubeConfig, h.K8sAppConf.PortForward, logger)
		if err != nil {
			return err
		}

		// Due to automated config loading, this worker may naturally decide to not work.
		// This will interrupt all other components
		if k8s == nil {
			logger.Warn().Msg("Unable to load kubeconfig, exiting")
			return nil
		}

		zlogger := logs.NewNatsZeroLogger(logger)
		worker := worker.NewWorker(natsClient, k8s, &zlogger)

		// Blocks until complete or errored
		return worker.Run(ctx)
	}

	h.runGroup.Add(
		func() error {
			return start()
		},
		func(_ error) {
			cancel()
		},
	)

	return nil
}

func (h *HopsServer) startNATSClient() (*nats.Client, error) {
	zlog := logs.NewNatsZeroLogger(h.Logger)

	keyFile, err := nats.NewKeyFile(h.KeyFilePath)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to load keyfile")
		return nil, err
	}

	clientOpts := []nats.ClientOpt{}
	if h.ReplayEvent != "" {
		clientOpts = append(clientOpts, nats.WithReplay(nats.DefaultConsumerName, h.ReplayEvent))
		h.Logger.Info().Msgf("Replaying source event: %s", h.ReplayEvent)
	} else if h.RunnerConf.Serve {
		clientOpts = append(clientOpts, nats.WithRunner(nats.DefaultConsumerName))
	}

	if h.HTTPAppConf.Serve {
		clientOpts = append(clientOpts, nats.WithWorker("http"))
	}

	if h.K8sAppConf.Serve {
		clientOpts = append(clientOpts, nats.WithWorker("k8s"))
	}

	natsClient, err := nats.NewClient(
		keyFile.NatsUrl(),
		keyFile.AccountId,
		nats.DefaultInterestTopic,
		&zlog,
		clientOpts...,
	)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to start NATS client")
		return nil, err
	}

	return natsClient, nil
}

func (h *HopsServer) startReloader(ctx context.Context, hopsLoader *HopsFileLoader) error {
	if !h.Watch {
		return nil
	}

	h.reloadManager.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		err := hopsLoader.Reload(ctx)
		if err != nil {
			h.Logger.Warn().Msgf("Hops files could not be reloaded: %s", err.Error())
			return nil
		}

		h.Logger.Info().Msg("Hops files reloaded")
		return nil
	}))

	{
		dirNotifier, err := NewDirNotifier(h.HopsPath)
		if err != nil {
			return err
		}

		// Add file watcher based reload notifier.
		h.reloadManager.On(dirNotifier.Notifier())

		ctx, cancel := context.WithCancel(ctx)
		h.runGroup.Add(
			func() error {
				// Block forever until the watcher stops.
				h.Logger.Info().Msgf("Watching %s for changes", h.HopsPath)
				<-ctx.Done()
				return nil
			},
			func(_ error) {
				h.Logger.Info().Msg("Stopping hops file watcher")
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
				// TODO: Avoid all reloader logic if watch not true
				h.Logger.Info().Msg("Auto-reloading cancelled")
				cancel()
			},
		)
	}

	return nil
}

func (h *HopsServer) startRunner(ctx context.Context, hopsLoader *HopsFileLoader, natsClient *nats.Client) error {
	if !h.RunnerConf.Serve {
		return nil
	}

	runner, err := NewRunner(natsClient, hopsLoader, h.Logger)
	if err != nil {
		return err
	}

	if h.Watch {
		h.reloadManager.Add(10, reload.ReloaderFunc(func(ctx context.Context, id string) error {
			return runner.Reload(ctx)
		}))
	}

	ctx, cancel := context.WithCancel(ctx)
	h.runGroup.Add(
		func() error {
			return runner.Run(ctx, nats.DefaultConsumerName)
		},
		func(_ error) {
			cancel()
		},
	)

	return nil
}
