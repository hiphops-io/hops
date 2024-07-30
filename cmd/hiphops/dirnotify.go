package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/oklog/run"
	"github.com/rs/zerolog"
	"github.com/slok/reload"
)

type (
	// DirNotifier watches a path and its subdirectories for changes
	// notifying when one occurs
	DirNotifier struct {
		path    string
		logger  zerolog.Logger
		watcher *fsnotify.Watcher
	}
)

func NewDirNotifier(path string, logger zerolog.Logger) (*DirNotifier, error) {
	d := &DirNotifier{path: path, logger: logger}

	err := d.initWatcher(path)
	if err != nil {
		logger.Error().Err(err).Msg("Error watching directory for changes")
		return nil, err
	}

	return d, nil
}

func (d *DirNotifier) Close() error {
	return d.watcher.Close()
}

func (d *DirNotifier) Notifier(ctx context.Context) reload.Notifier {
	// Using a timer to debounce file change events, preventing multiple events
	// triggering hops reload for a single action
	notifyChan := make(chan string)
	t := time.AfterFunc(math.MaxInt64, func() { notifyChan <- "file-watch" })
	t.Stop()
	waitFor := 150 * time.Millisecond

	go func() {
		for {
			select {
			case event := <-d.watcher.Events:
				if event.Has(fsnotify.Chmod) {
					continue
				}

				if event.Has(fsnotify.Create) {
					// File created, is it a dir?
					// We ignore the error from os.Stat as normal use would cause this to
					// return an error (e.g., when saving files via vim).
					if fileInfo, err := os.Stat(event.Name); err == nil && fileInfo.IsDir() {
						_ = d.watcher.Add(event.Name)
					}
				}

				// This loop only needs to start/reset the timer. The reload listens
				// for the timer being activated
				t.Reset(waitFor)

			case <-ctx.Done():
				return
			}
		}
	}()

	return reload.NotifierFunc(func(ctx context.Context) (string, error) {
		select {
		case id := <-notifyChan:
			return id, nil
		case err := <-d.watcher.Errors:
			return "", err
		}
	})
}

func (d *DirNotifier) NotifyReload(ctx context.Context, reloadMngr *reload.Manager, runGroup *run.Group) {
	// Add file watcher based reload notifier.
	reloadMngr.On(d.Notifier(ctx))

	ctx, cancel := context.WithCancel(ctx)
	runGroup.Add(
		func() error {
			// Block forever until the watcher stops.
			d.logger.Info().Msgf("Watching %s for changes", d.path)
			<-ctx.Done()
			return nil
		},
		func(_ error) {
			d.logger.Info().Msg("Stopping watching dir for changes")
			d.Close()
			cancel()
		},
	)
}

func (d *DirNotifier) initWatcher(path string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = watcher.Add(path)
	if err != nil {
		return fmt.Errorf("Unable to add file watcher for %s: %w", path, err)
	}

	// Add subdirectories
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("Unable to read subdirectories for %s: %w", path, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		err = watcher.Add(filepath.Join(path, entry.Name()))
		if err != nil {
			return fmt.Errorf("Unable to add file watcher for %s, %w", entry.Name(), err)
		}
	}

	d.watcher = watcher

	return nil
}
