package hops

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hiphops-io/hops/dsl"
	"github.com/rs/zerolog"
	"github.com/slok/reload"
)

type (
	// DirNotifer watches a path and its subdirectories for changes
	// notifying when one occurs
	DirNotifier struct {
		notifier reload.Notifier
		watcher  *fsnotify.Watcher
	}

	HopsFileLoader struct {
		hopsFiles dsl.HopsFiles
		logger    zerolog.Logger
		mu        sync.RWMutex
		path      string
	}
)

func NewDirNotifier(path string) (*DirNotifier, error) {
	d := &DirNotifier{}

	err := d.initWatcher(path)
	if err != nil {
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
				break
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

func NewHopsFileLoader(path string, tolerant bool, logger zerolog.Logger) (*HopsFileLoader, error) {
	h := &HopsFileLoader{path: path, logger: logger}
	err := h.Reload(context.Background(), tolerant)
	if err != nil {
		return h, err
	}

	return h, nil
}

func (h *HopsFileLoader) Reload(ctx context.Context, tolerant bool) error {
	hops, err := dsl.ReadHopsFilePath(h.path, h.logger)
	if err != nil && !tolerant {
		return fmt.Errorf("Failed to read hops files: %w", err)
	}
	if err != nil && h.hopsFiles.Hash != "" {
		// If hopsFiles already set, then just don't update it with the broken one
		return nil
	}

	if err != nil {
		hops = &dsl.HopsFiles{
			Hash:  "empty",
			Files: []dsl.FileContent{},
		}
	}

	h.mu.Lock()
	h.hopsFiles = *hops
	h.mu.Unlock()

	return nil
}

func (h *HopsFileLoader) Get() (*dsl.HopsFiles, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return &h.hopsFiles, nil
}
