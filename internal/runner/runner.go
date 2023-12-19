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
	"github.com/robfig/cron"
	"github.com/rs/zerolog"

	"github.com/hiphops-io/hops/dsl"
	"github.com/hiphops-io/hops/nats"
)

const (
	hopsKeyPrefix = "hopsconf-"
)

type (
	NatsClient interface {
		ConsumeSequences(context.Context, string, nats.SequenceHandler) error
		GetMsg(ctx context.Context, subjTokens ...string) (*jetstream.RawStreamMsg, error)
		GetSysObject(key string) ([]byte, error)
		Publish(context.Context, []byte, ...string) (*jetstream.PubAck, bool, error)
		PublishResult(context.Context, time.Time, interface{}, error, ...string) (error, bool)
		PutSysObject(string, []byte) (*natsgo.ObjectInfo, error)
	}

	Runner struct {
		cache       *cache.Cache
		hopsContent *hcl.BodyContent
		hopsKey     string
		logger      zerolog.Logger
		natsClient  NatsClient
		schedules   []*Schedule
	}
)

func NewRunner(natsClient NatsClient, hops *dsl.HopsFiles, logger zerolog.Logger) (*Runner, error) {
	runner := &Runner{
		hopsKey:     hops.Hash,
		hopsContent: hops.BodyContent,
		logger:      logger,
		natsClient:  natsClient,
	}

	runner.initCache()

	err := runner.initHopsBackup(hops)
	if err != nil {
		return nil, fmt.Errorf("Unable to store hops files %w", err)
	}

	err = runner.initHopsSchedules(hops)
	if err != nil {
		return nil, fmt.Errorf("Unable to start schedules %w", err)
	}

	return runner, nil
}

func (r *Runner) Run(ctx context.Context, fromConsumer string) error {
	c := cron.New()
	defer c.Stop()

	for _, schedule := range r.schedules {
		c.Schedule(schedule.CronSchedule, schedule)
	}
	c.Start()

	return r.natsClient.ConsumeSequences(ctx, fromConsumer, r)
}

func (r *Runner) SequenceCallback(
	ctx context.Context,
	sequenceId string,
	msgBundle nats.MessageBundle,
) error {
	logger := r.logger.With().Str("sequence_id", sequenceId).Logger()

	hops, err := r.sequenceHops(ctx, sequenceId, msgBundle)
	if err != nil {
		return fmt.Errorf("Unable to fetch assigned hops file for sequence: %w", err)
	}

	hop, err := dsl.ParseHops(ctx, hops, msgBundle, logger)
	if err != nil {
		return fmt.Errorf("Error parsing hops config: %w", err)
	}

	r.logger.Debug().Msg("Successfully parsed hops file")

	// TODO: Run all sensors concurrently via goroutines
	var mergedErrors error
	for i := range hop.Ons {
		sensor := &hop.Ons[i]

		done, err := r.checkIfDone(ctx, sensor, sequenceId, msgBundle, logger)
		if err != nil {
			mergedErrors = multierror.Append(mergedErrors, err)
		}
		if done {
			continue
		}

		err = r.dispatchCalls(ctx, sensor, sequenceId, logger)
		if err != nil {
			mergedErrors = multierror.Append(mergedErrors, err)
		}
	}

	return mergedErrors
}

func (r *Runner) checkIfDone(ctx context.Context, sensor *dsl.OnAST, sequenceId string, msgBundle nats.MessageBundle, logger zerolog.Logger) (bool, error) {
	if sensor.Done != nil {
		err := r.dispatchDone(ctx, sensor.Slug, sensor.Done, sequenceId, logger)
		return true, err
	}

	// If all dispatchable calls have results already, then we're done regardless
	done := true
	for _, call := range sensor.Calls {
		_, ok := msgBundle[call.Slug]
		if !ok {
			done = false
			break
		}
	}

	if done {
		done := &dsl.DoneAST{
			Result: []byte("{}"),
		}
		err := r.dispatchDone(ctx, sensor.Slug, done, sequenceId, logger)
		return true, err
	}

	return false, nil
}

func (r *Runner) dispatchDone(ctx context.Context, onSlug string, done *dsl.DoneAST, sequenceId string, logger zerolog.Logger) error {
	logger = logger.With().Str("on", onSlug).Logger()

	err, sent := r.natsClient.PublishResult(
		ctx,
		time.Now(),
		done.Result,
		done.Error,
		nats.ChannelNotify,
		sequenceId,
		onSlug,
		nats.DoneMessageId,
	)

	if err != nil {
		return err
	}

	if sent {
		logger.Info().Msg("Pipeline is done")
	}

	return nil
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

	_, _, err := r.natsClient.Publish(ctx, call.Inputs, nats.ChannelRequest, sequenceId, call.Slug, app, handler)
	if err != nil {
		errorchan <- err
		return
	}

	if err == nil {
		logger.Info().Msgf("Dispatched call: %s", call.Slug)
	}

	errorchan <- nil
}

func (r *Runner) initCache() {
	r.cache = cache.New(5*time.Minute, 10*time.Minute)
}

func (r *Runner) initHopsBackup(hops *dsl.HopsFiles) error {
	hopsFileB, err := json.Marshal(hops.Files)
	if err != nil {
		return err
	}

	// Store in object store
	_, err = r.natsClient.PutSysObject(r.hopsKey, hopsFileB)
	if err != nil {
		return err
	}

	// Pre-populate local cache (local hops cache item should never expire)
	r.logger.Debug().Msgf("Populating local cache with hops config: %s", r.hopsKey)
	r.cache.Set(r.hopsKey, hops, cache.NoExpiration)

	return nil
}

// initHopsSchedules parses the schedule blocks in a hops config and inits
// the cron schedules ready for running
//
// This function will not run the schedules, just prepare them
func (r *Runner) initHopsSchedules(hops *dsl.HopsFiles) error {
	hop, err := dsl.ParseHopsSchedules(hops, r.logger)
	if err != nil {
		return err
	}

	schedules := []*Schedule{}
	for _, scheduleConf := range hop.ListSchedules() {
		schedule, err := NewSchedule(scheduleConf, r.natsClient, r.logger)
		if err != nil {
			return err
		}

		schedules = append(schedules, schedule)
	}

	r.schedules = schedules

	return nil
}

// sequenceHops attempts to assign the local hops config to a sequence,
// returning either the newly assigned hops body or the existing one if present.
func (r *Runner) sequenceHops(ctx context.Context, sequenceId string, msgBundle nats.MessageBundle) (*dsl.HopsFiles, error) {
	key, err := r.sequenceHopsKey(ctx, sequenceId, msgBundle)
	if err != nil {
		return nil, fmt.Errorf("Unable to decide hops config for pipeline: %w", err)
	}

	// Attempt to fetch from cache
	content := r.sequenceHopsCached(key)
	if content != nil {
		r.logger.Debug().Msg("Using cached hops config")
		return content, nil
	}

	// No cached copy, fetch from object store
	r.logger.Debug().Msg("Using remote stored hops config")
	return r.sequenceHopsStored(key)
}

// sequenceHopsKey gets or sets the hops key for a sequence, returning the final key
func (r *Runner) sequenceHopsKey(ctx context.Context, sequenceId string, msgBundle nats.MessageBundle) (string, error) {
	hopsKeyB, ok := msgBundle["hops"]
	if ok {
		return hopsKeyFromBytes(hopsKeyB)
	}

	tokens := nats.SequenceHopsKeyTokens(sequenceId)

	hopsJson := fmt.Sprintf("\"%s\"", r.hopsKey)
	_, sent, err := r.natsClient.Publish(ctx, []byte(hopsJson), tokens...)
	if err != nil {
		return "", fmt.Errorf("Unable to assign hops config to pipeline: %w", err)
	}

	// If the message was successfully sent, it means we assigned first and can continue
	if sent {
		return r.hopsKey, nil
	}

	// Otherwise it means another client won the race - we need to get that hops key
	msg, err := r.natsClient.GetMsg(ctx, tokens...)
	if err != nil {
		return "", fmt.Errorf("Unable to fetch assigned hops config for pipeline: %w", err)
	}

	return hopsKeyFromBytes(msg.Data)
}

// sequenceHopsCached gets the hops config assigned to a sequence by key,
// first looking up in the cache, then falling back to object store
func (r *Runner) sequenceHopsCached(key string) *dsl.HopsFiles {
	if cachedContent, found := r.cache.Get(key); found {
		return cachedContent.(*dsl.HopsFiles)
	}

	return nil
}

// sequenceHopsFromStore gets the hops config assigned to a sequence by key,
// fetching from the object store
func (r *Runner) sequenceHopsStored(key string) (*dsl.HopsFiles, error) {
	hopsFileB, err := r.natsClient.GetSysObject(key)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve hops config '%s': %w", key, err)
	}

	hopsFilesContent := []dsl.FileContent{}
	err = json.Unmarshal(hopsFileB, &hopsFilesContent)
	if err != nil {
		return nil, fmt.Errorf("Unable to decode retrieved hops config '%s': %w", key, err)
	}

	// Update types for legacy format
	for i := range hopsFilesContent {
		if hopsFilesContent[i].Type == "" {
			hopsFilesContent[i].Type = dsl.HopsFile
		}
	}

	hopsContent, hash, err := dsl.ReadHopsFileContents(hopsFilesContent)
	if err != nil {
		return nil, fmt.Errorf("Unable to read retrieved hops config '%s': '%w'", key, err)
	}

	// Validate the integrity. Hash should be identical to hash found in key
	if !strings.Contains(key, hash) {
		return nil, fmt.Errorf("Invalid hash for stored hops config, hash was '%s' but key was '%s'", hash, key)
	}

	// Store in cache
	r.logger.Debug().Msg("Caching stored hops locally")
	hopsFiles := &dsl.HopsFiles{
		Hash:        key,
		BodyContent: hopsContent,
		Files:       hopsFilesContent,
	}
	r.cache.Set(key, hopsFiles, cache.DefaultExpiration)

	return hopsFiles, nil
}

func hopsKeyFromBytes(keyB []byte) (string, error) {
	key := ""
	err := json.Unmarshal(keyB, &key)
	if err != nil {
		err = fmt.Errorf("Unable to decode hops key %w", err)
	}
	return key, err
}
