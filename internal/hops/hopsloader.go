package hops

import (
	"context"
	"fmt"
	"sync"

	"github.com/hiphops-io/hops/dsl"
)

type HopsFileLoader struct {
	path      string
	hopsFiles dsl.HopsFiles
	mu        sync.RWMutex
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
