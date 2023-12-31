package hops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/hiphops-io/hops/dsl"
	"github.com/slok/reload"
)

type (
	// DirNotifer watches a path and its subdirectories for changes
	// notifying when one occurs
	DirNotifier struct {
		watcher  *fsnotify.Watcher
		notifier reload.Notifier
	}

	HopsFileLoader struct {
		path      string
		hopsFiles dsl.HopsFiles
		mu        sync.RWMutex
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

func (d *DirNotifier) Notifier() reload.Notifier {
	return reload.NotifierFunc(func(ctx context.Context) (string, error) {
		select {
		case event := <-d.watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				// File created, is it a dir?
				fileInfo, _ := os.Stat(event.Name)
				// We ignore the error from above as normal use would cause this to
				// return an error (e.g. when saving files via vim)

				if fileInfo.IsDir() {
					d.watcher.Add(event.Name)
				}
			}

			return "file-watch", nil
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

func NewHopsFileLoader(path string) (*HopsFileLoader, error) {
	h := &HopsFileLoader{path: path}
	err := h.Reload(context.Background())
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (h *HopsFileLoader) Reload(ctx context.Context) error {
	hops, err := dsl.ReadHopsFilePath(h.path)
	if err != nil {
		return fmt.Errorf("Failed to read hops files: %w", err)
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
