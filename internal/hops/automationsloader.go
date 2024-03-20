package hops

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/goccy/go-json"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
	"github.com/slok/reload"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/nats"
)

const AutomationsHashKey = "hops"

type (
	// DirNotifier watches a path and its subdirectories for changes
	// notifying when one occurs
	DirNotifier struct {
		notifier reload.Notifier
		watcher  *fsnotify.Watcher
	}

	AutomationsLoader struct {
		automations *dsl.Automations
		cache       *cache.Cache
		logger      zerolog.Logger
		mu          sync.RWMutex
		natsClient  *nats.Client
		path        string
	}
)

func NewAutomationsLoader(path string, tolerant bool, natsClient *nats.Client, logger zerolog.Logger) (*AutomationsLoader, error) {
	al := &AutomationsLoader{
		cache:      cache.New(5*time.Minute, 10*time.Minute),
		path:       path,
		natsClient: natsClient,
		logger:     logger,
	}
	err := al.Reload(context.Background(), tolerant)
	if err != nil {
		return al, err
	}

	return al, nil
}

func (al *AutomationsLoader) Reload(ctx context.Context, tolerant bool) error {
	a, d, err := dsl.NewAutomationsFromDir(al.path)
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
		a = &dsl.Automations{
			Files:     map[string][]byte{},
			Hash:      "empty",
			Hops:      &dsl.HopsAST{},
			Manifests: map[string]*dsl.Manifest{},
		}
	}

	al.mu.Lock()
	al.automations = a
	al.mu.Unlock()

	err = al.Save()
	if err != nil {
		return fmt.Errorf("Unable to save automations to store: %w", err)
	}

	return nil
}

// Get returns the automations for the given hash key or the local automations
// if hash key is the empty string
func (al *AutomationsLoader) Get(hash string) (*dsl.Automations, error) {
	if hash == "" {
		al.mu.RLock()
		defer al.mu.RUnlock()

		return al.automations, nil
	}

	a := al.GetFromCache(hash)
	if a != nil {
		return a, nil
	}

	return al.GetFromStore(hash)
}

// GetFromCache returns Automations by hash key from the local cache
func (al *AutomationsLoader) GetFromCache(hash string) *dsl.Automations {
	if cachedContent, found := al.cache.Get(hash); found {
		return cachedContent.(*dsl.Automations)
	}

	return nil
}

// GetFromStore returns the Automations by hash key from object storage
func (al *AutomationsLoader) GetFromStore(hash string) (*dsl.Automations, error) {
	automationFilesB, err := al.natsClient.GetSysObject(hash)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve automations from store with key '%s': %w", hash, err)
	}

	automationFiles := map[string][]byte{}
	err = json.Unmarshal(automationFilesB, &automationFiles)
	if err != nil {
		return nil, fmt.Errorf("Unable to read automations config fetched from store with key '%s': %w", hash, err)
	}

	a, d := dsl.NewAutomationsFromContent(automationFiles)
	if d.HasErrors() {
		// This should only happen if the stored automation is written in an older version of hops dsl
		// that has breaking changes vs the current code.
		return nil, fmt.Errorf("Unable to parse retrieved automations config with key '%s': %w", hash, err)
	}

	// Validate the integrity of the retrieved automations.
	// Re-calculated hash should be identical to the hash key it was stored under
	if !strings.Contains(hash, a.Hash) {
		return nil, fmt.Errorf("Invalid hash for retrieved automations config. Hash was '%s', but we expected '%s'.", a.Hash, hash)
	}

	return a, nil
}

// GetForSequence returns the Automations assigned to an event sequence.
// If no Automations are assigned to a sequence, it first assigns and then returns the local Automations
//
// This ensures that sequences are handled by a consistent configuration until
// completion, even if multiple hops instances handle the work.
func (al *AutomationsLoader) GetForSequence(ctx context.Context, sequenceID string, msgBundle nats.MessageBundle) (*dsl.Automations, error) {
	hash, err := al.GetOrSetHashForSequence(ctx, sequenceID, msgBundle)
	if err != nil {
		return nil, err
	}

	a, err := al.Get(hash)
	if err != nil {
		return nil, err
	}

	// If our hash isn't the same as our local, then we need to make sure to populate
	// our local cache with this automations config to avoid store lookups in future.
	al.mu.RLock()
	localHash := al.automations.Hash
	al.mu.RUnlock()

	if localHash != hash {
		al.SaveInCache(a, cache.DefaultExpiration)
	}

	return a, nil
}

// GetOrSetHashForSequence returns the Automations hash assigned to an event sequence.
// If no hash are assigned to a sequence, it first assigns and then returns the local Automations
//
// This ensures that sequences are handled by a consistent configuration until
// completion, even if multiple hops instances handle the work.
func (al *AutomationsLoader) GetOrSetHashForSequence(ctx context.Context, sequenceID string, msgBundle nats.MessageBundle) (string, error) {
	// First check the message bundle if an automations config is already assigned.
	// This is the most common case.
	automationHashB, ok := msgBundle[AutomationsHashKey]
	if ok {
		return automationsKeyFromBytes(automationHashB)
	}

	// If no automations config is assigned, then we'll attempt to assign our own.
	tokens := nats.SequenceHopsKeyTokens(sequenceID)

	al.mu.RLock()
	hash := al.automations.Hash
	al.mu.RUnlock()

	jsonHash := fmt.Sprintf("\"%s\"", hash)
	_, sent, err := al.natsClient.Publish(ctx, []byte(jsonHash), tokens...)
	if err != nil {
		return "", fmt.Errorf("Unable to assign hops config to pipeline: %w", err)
	}

	// If the message was successfully sent, it means we assigned first and can continue
	if sent {
		return hash, nil
	}

	// If we failed to assign our automation to the sequence, then another instance won
	// the race. Let's grab that from the source.
	msg, err := al.natsClient.GetMsg(ctx, tokens...)
	if err != nil {
		return "", fmt.Errorf("Unable to fetch assigned automations config for pipeline: %w", err)
	}

	return automationsKeyFromBytes(msg.Data)
}

func (al *AutomationsLoader) Save() error {
	al.mu.RLock()
	defer al.mu.RUnlock()

	err := al.SaveInStore(al.automations)
	if err != nil {
		return err
	}

	al.SaveInCache(al.automations, cache.NoExpiration)

	return nil
}

func (al *AutomationsLoader) SaveInCache(a *dsl.Automations, ttl time.Duration) {
	al.cache.Set(a.Hash, a, ttl)
}

func (al *AutomationsLoader) SaveInStore(a *dsl.Automations) error {
	automationFiles, err := json.Marshal(a.Files)
	if err != nil {
		return err
	}

	_, err = al.natsClient.PutSysObject(a.Hash, automationFiles)
	return err
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

func automationsKeyFromBytes(keyB []byte) (string, error) {
	key := ""
	err := json.Unmarshal(keyB, &key)
	if err != nil {
		err = fmt.Errorf("Unable to decode automations key %w", err)
	}
	return key, err
}
