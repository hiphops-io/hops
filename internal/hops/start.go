package hops

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/oklog/run"
	"github.com/rs/zerolog"
	"github.com/slok/reload"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/internal/httpapp"
	"github.com/hiphops-io/hops/internal/k8sapp"
	"github.com/hiphops-io/hops/internal/runner"
	"github.com/hiphops-io/hops/logs"
	"github.com/hiphops-io/hops/nats"
	"github.com/hiphops-io/hops/worker"
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
		Watch       bool
		Console
		HTTPApp
		K8sApp
		Runner
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
)

func (h *HopsServer) Start(ctx context.Context) error {
	ctx, rootCancel := context.WithCancel(ctx)
	defer rootCancel()

	if !(h.Runner.Serve || h.Console.Serve || h.K8sApp.Serve || h.HTTPApp.Serve) {
		return errors.New("No serve functions are enabled. Nothing to do.")
	}

	// TODO: Should include this in oklog/run too
	natsClient, err := h.startNATSClient()
	if natsClient != nil {
		defer natsClient.Close()
	}
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to start NATS client")
		return err
	}

	// errs := make(chan error, 1)

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

	if h.Console.Serve {
		console, err := NewConsole(h.Address, hopsLoader, natsClient, h.Logger)
		if err != nil {
			return err
		}

		reloadManager.Add(10, reload.ReloaderFunc(func(ctx context.Context, id string) error {
			return console.Reload(ctx)
		}))

		runGroup.Add(
			func() error {
				return console.Serve()
			},
			func(_ error) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err := console.Shutdown(ctx)
				if err != nil {
					h.Logger.Error().Err(err).Msg("Unable to shut down http server")
				}
			},
		)

		// err = console.Serve(h.Address)
		// if err != nil {
		// 	return err
		// }
	}

	{
		ctx, cancel := context.WithCancel(ctx)
		runGroup.Add(
			func() error {
				return reloadManager.Run(ctx)
			},
			func(_ error) {
				// TODO: Check if error and log appropriately if so
				// TODO: Could totally avoid reloader if watch not true
				h.Logger.Info().Msg("Auto-reloading cancelled")
				cancel()
			},
		)
	}

	{
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		err = watcher.Add(h.HopsPath)
		if err != nil {
			return fmt.Errorf("Unable to add file watcher for %s: %w", h.HopsPath, err)
		}

		// Add subdirectories
		// It's invalid for hops path to be anything other than a directory,
		// so we assume that and error otherwise
		entries, err := os.ReadDir(h.HopsPath)
		if err != nil {
			h.Logger.Warn().Err(err).Msgf("Unable to read subdirectories for %s", h.HopsPath)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			err = watcher.Add(filepath.Join(h.HopsPath, entry.Name()))
			if err != nil {
				h.Logger.Warn().Err(err).Msgf("Unable to add file watcher for %s", entry.Name())
			}
		}

		// Add file watcher based reload notifier.
		reloadManager.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
			select {
			case event := <-watcher.Events:
				switch {
				case event.Op&fsnotify.Create == fsnotify.Create:
					// File created, is it a dir?
					fileInfo, _ := os.Stat(event.Name)
					// We ignore the error from above as normal use would cause this to
					// return an error (e.g. when saving files via vim)

					if fileInfo.IsDir() {
						watcher.Add(event.Name)
					}
				}

				return "file-watch", nil
			case err := <-watcher.Errors:
				return "", err
			}
		}))

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
				watcher.Close()
				cancel()
			},
		)
	}

	return runGroup.Run()

	// if h.Runner.Serve {
	// 	go func() {
	// 		err := startRunner(
	// 			ctx,
	// 			hops,
	// 			natsClient,
	// 			nats.DefaultConsumerName,
	// 			h.Logger,
	// 		)
	// 		if err != nil {
	// 			errs <- nil
	// 		}
	// 	}()
	// }

	// if h.HTTPApp.Serve {
	// 	go func() {
	// 		err := startHTTPApp(
	// 			ctx,
	// 			natsClient,
	// 			h.Logger,
	// 		)
	// 		if err != nil {
	// 			errs <- err
	// 		}
	// 	}()
	// }

	// if h.K8sApp.Serve {
	// 	go func() {
	// 		err := startK8sApp(
	// 			ctx,
	// 			natsClient,
	// 			h.K8sApp.KubeConfig,
	// 			h.K8sApp.PortForward,
	// 			h.Logger,
	// 		)
	// 		if err != nil {
	// 			errs <- err
	// 		}
	// 	}()
	// }

	// if err := <-errs; err != nil {
	// 	h.Logger.Error().Err(err).Msg("Start failed")
	// 	return err
	// }

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
		return nil, err
	}

	return natsClient, nil
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
