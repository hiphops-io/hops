package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/v2"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/nats"
)

type (
	NatsClient interface {
		ConsumeSequences(context.Context, nats.SequenceHandler) error
		GetSysObject(key string) ([]byte, error)
		Publish(context.Context, []byte, ...string) (*jetstream.PubAck, error, bool)
		PutSysObject(string, []byte) (*natsgo.ObjectInfo, error)
	}

	Runner struct {
		cache      *cache.Cache
		hops       hcl.Body
		hopsHash   string
		logger     zerolog.Logger
		natsClient NatsClient
	}
)

func NewRunner(natsClient NatsClient, hops *dsl.HopsFiles, logger zerolog.Logger) (*Runner, error) {
	runner := &Runner{
		hops:       hops.Body,
		hopsHash:   hops.Hash,
		logger:     logger,
		natsClient: natsClient,
	}

	runner.initCache()

	err := runner.initHopsBackup(hops)
	if err != nil {
		return nil, fmt.Errorf("Unable to store hops filesL %w", err)
	}

	return runner, nil
}

func (r *Runner) Run(ctx context.Context) error {
	return r.natsClient.ConsumeSequences(ctx, r)
}

func (r *Runner) SequenceCallback(
	ctx context.Context,
	sequenceId string,
	msgBundle nats.MessageBundle,
) error {
	logger := r.logger.With().Str("sequence_id", sequenceId).Logger()

	hops, err := r.sequenceHops(msgBundle)
	if err != nil {
		return err
	}

	hop, err := dsl.ParseHops(ctx, hops, msgBundle, logger)
	if err != nil {
		return err
	}

	r.logger.Debug().Msg("Successfully parsed hops file")

	// TODO: Run all sensors concurrently via goroutines
	var sensorErrors error
	for i := range hop.Ons {
		sensor := &hop.Ons[i]
		err := r.dispatchCalls(ctx, sensor, sequenceId, logger)
		if err != nil {
			sensorErrors = multierror.Append(sensorErrors, err)
		}
	}

	return sensorErrors
}

func (r *Runner) dispatchCalls(ctx context.Context, sensor *dsl.OnAST, sequenceId string, logger zerolog.Logger) error {
	var wg sync.WaitGroup
	var errs error

	logger = logger.With().Str("on", sensor.Slug).Logger()
	logger.Info().Msg("Running on calls")

	numTasks := len(sensor.Calls)
	errorchan := make(chan error, numTasks)

	for _, call := range sensor.Calls {
		call := call
		wg.Add(1)
		go r.dispatchCall(ctx, &wg, call, sequenceId, errorchan, logger)
	}

	wg.Wait()
	close(errorchan)

	for err := range errorchan {
		errs = errors.Join(errs, err)
	}

	return errs
}

func (r *Runner) dispatchCall(ctx context.Context, wg *sync.WaitGroup, call dsl.CallAST, sequenceId string, errorchan chan<- error, logger zerolog.Logger) {
	defer wg.Done()

	app, handler, found := strings.Cut(call.TaskType, "_")
	if !found {
		errorchan <- fmt.Errorf("Unable to parse app/handler from call %s", call.Name)
		return
	}

	_, err, _ := r.natsClient.Publish(ctx, call.Inputs, nats.ChannelRequest, sequenceId, call.Slug, app, handler)

	// At the time of writing, the go client does not contain an error matching
	// the 'maximum massages per subject exceeded' error.
	// We match on the code here instead - @manterfield
	if err != nil && !strings.Contains(err.Error(), "err_code=10077") {
		errorchan <- err
		return
	}

	if err == nil {
		logger.Info().Msgf("Dispatched call: %s", call.Slug)
	}

	errorchan <- nil
}

func (r *Runner) getHopsBody(key string) (hcl.Body, error) {
	if storedBody, found := r.cache.Get(key); found {
		body := storedBody.(hcl.Body)
		return body, nil
	}

	// No locally cached copy, fetch from object store
	body, err := r.getHopsBodyFromStore(key)
	if err != nil {
		return nil, err
	}

	// Store in cache
	r.cache.Set(key, body, cache.DefaultExpiration)

	return body, nil
}

func (r *Runner) getHopsBodyFromStore(key string) (hcl.Body, error) {
	hopsFileB, err := r.natsClient.GetSysObject(key)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve hops config '%s': %w", key, err)
	}

	hopsFiles := []dsl.FileContent{}
	err = json.Unmarshal(hopsFileB, &hopsFiles)
	if err != nil {
		return nil, fmt.Errorf("Unable to decode retrieved hops config '%s': %w", key, err)
	}

	body, hash, err := dsl.ReadHopsFileContents(hopsFiles)
	if err != nil {
		return nil, fmt.Errorf("Unable to read retrieved hops config '%s': '%w'", key, err)
	}
	// Validate the integrity. Hash should be identical to hash found in key
	if !strings.Contains(key, hash) {
		return nil, fmt.Errorf("Invalid hash for stored hops config, hash was '%s' but key was '%s'", hash, key)
	}

	return body, nil
}

func (r *Runner) initCache() {
	r.cache = cache.New(2*time.Hour, 4*time.Hour)
}

func (r *Runner) initHopsBackup(hops *dsl.HopsFiles) error {
	key := fmt.Sprintf("hopsconf-%s", hops.Hash)

	hopsFileB, err := json.Marshal(hops.Files)
	if err != nil {
		return err
	}

	// Store in object store
	_, err = r.natsClient.PutSysObject(key, hopsFileB)
	if err != nil {
		return err
	}

	_, err = r.getHopsBody(key)
	if err != nil {
		fmt.Println("Failed to get hops body", err)
	}

	// Store hcl.Body in cache
	r.cache.Set(key, hops.Body, cache.NoExpiration)

	return nil
}

func (r *Runner) sequenceHops(msgBundle nats.MessageBundle) (hcl.Body, error) {
	_, ok := msgBundle["hops"]
	if !ok {
		return r.hops, nil
	}

	return r.hops, nil
}
