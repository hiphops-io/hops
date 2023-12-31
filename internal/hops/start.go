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
		HopsPath    string
		KeyFilePath string
		Logger      zerolog.Logger
		ReplayEvent string
		Watch       bool
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
		Serve       bool
		PortForward bool
	}

	RunnerConf struct {
		Serve bool
	}
)

func (h *HopsServer) Start(ctx context.Context) error {
	ctx, rootCancel := context.WithCancel(ctx)
	defer rootCancel()

	if !(h.RunnerConf.Serve || h.HTTPServerConf.Serve || h.K8sAppConf.Serve || h.HTTPAppConf.Serve) {
		return errors.New("No serve functions are enabled. Nothing to do.")
	}

	// TODO: Should include this in runGroup too
	natsClient, err := h.startNATSClient()
	if natsClient != nil {
		defer natsClient.Close()
	}
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to start NATS client")
		return err
	}

	var (
		runGroup      run.Group
		reloadManager = reload.NewManager()
	)

	hopsLoader, err := NewHopsFileLoader(h.HopsPath)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to read hops files")
		return fmt.Errorf("Failed to read hops files: %w", err)
	}

	reloadManager.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
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
		reloadManager.On(dirNotifier.Notifier())

		ctx, cancel := context.WithCancel(ctx)
		runGroup.Add(
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

	if h.HTTPServerConf.Serve {
		httpServer, err := NewHTTPServer(h.Address, hopsLoader, natsClient, h.Logger)
		if err != nil {
			return err
		}

		reloadManager.Add(10, reload.ReloaderFunc(func(ctx context.Context, id string) error {
			return httpServer.Reload(ctx)
		}))

		runGroup.Add(
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
	}

	if h.RunnerConf.Serve {
		runner, err := NewRunner(natsClient, hopsLoader, h.Logger)
		if err != nil {
			return err
		}

		reloadManager.Add(10, reload.ReloaderFunc(func(ctx context.Context, id string) error {
			return runner.Reload(ctx)
		}))

		ctx, cancel := context.WithCancel(ctx)
		runGroup.Add(
			func() error {
				return runner.Run(ctx, nats.DefaultConsumerName)
			},
			func(_ error) {
				cancel()
			},
		)
	}

	if h.HTTPAppConf.Serve {
		ctx, cancel := context.WithCancel(ctx)
		runGroup.Add(
			func() error {
				return startHTTPApp(ctx, natsClient, h.Logger)
			},
			func(_ error) {
				cancel()
			},
		)
	}

	if h.K8sAppConf.Serve {
		ctx, cancel := context.WithCancel(ctx)
		runGroup.Add(
			func() error {
				return startK8sApp(
					ctx,
					natsClient,
					h.K8sAppConf.KubeConfig,
					h.K8sAppConf.PortForward,
					h.Logger,
				)
			},
			func(_ error) {
				cancel()
			},
		)
	}

	{
		ctx, cancel := context.WithCancel(ctx)
		runGroup.Add(
			func() error {
				return reloadManager.Run(ctx)
			},
			func(_ error) {
				// TODO: Check if error and log appropriately if so
				// TODO: Avoid all reloader logic if watch not true
				h.Logger.Info().Msg("Auto-reloading cancelled")
				cancel()
			},
		)
	}

	return runGroup.Run()
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
		&zlog,
		clientOpts...,
	)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to start NATS client")
		return nil, err
	}

	return natsClient, nil
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
