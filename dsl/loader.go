package dsl

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/slok/reload"
)

type (
	// DirNotifier watches a path and its subdirectories for changes
	// notifying when one occurs
	DirNotifier struct {
		watcher *fsnotify.Watcher
	}

	AutomationsLoader struct {
		automations *Automations
		mu          sync.RWMutex
		path        string
	}
)

func NewAutomationsLoader(path string, tolerant bool) (*AutomationsLoader, error) {
	al := &AutomationsLoader{path: path}
	err := al.Reload(context.Background(), tolerant)
	if err != nil {
		return al, err
	}

	return al, nil
}

func (al *AutomationsLoader) Reload(ctx context.Context, tolerant bool) error {
	a, d, err := NewAutomationsFromDir(al.path)
	if err != nil && !tolerant {
		return fmt.Errorf("Failed to read hops files: %w", err)
	}
	if d.HasErrors() && !tolerant {
		return fmt.Errorf("Failed to decode hops files: %s", d.Error())
	}

	// Only reached when in tolerant mode or there's no errors
	failedLoad := err != nil || d.HasErrors()
	if failedLoad && al.automations != nil && al.automations.Hash != "empty" {
		// If an automation is already loaded, then don't replace with the broken one
		return nil
	}

	if failedLoad {
		a = &Automations{
			Files:     map[string][]byte{},
			Hash:      "empty",
			Hops:      &HopsAST{},
			Manifests: map[string]*Manifest{},
		}
	}

	al.mu.Lock()
	al.automations = a
	al.mu.Unlock()

	return nil
}

// Get returns the automations for the given hash key or the local automations
// if hash key is the empty string
func (al *AutomationsLoader) Get() (*Automations, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return al.automations, nil
}

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
