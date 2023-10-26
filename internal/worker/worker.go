package worker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"k8s.io/client-go/rest"

	undist "github.com/hiphops-io/hops/undistribute"
)

type TaskResponse struct {
	Status     string      `json:"status"`
	StartedAt  time.Time   `json:"started_at"`
	FinishedAt time.Time   `json:"finished_at"`
	Result     interface{} `json:"result,omitempty"`
	Error      error       `json:"error,omitempty"`
	subject    string
}

type handlerFunc func(context.Context, jetstream.Msg) error

type responseCallback func(context.Context, *TaskResponse) (error, bool)

type Worker struct {
	kubeConf        *rest.Config
	nc              *nats.Conn
	js              jetstream.JetStream
	consumer        jetstream.Consumer
	logger          zerolog.Logger
	handlers        map[string]handlerFunc
	handlerDeadline time.Duration
}

// NewWorker creates a new worker to handle task requests from NATS
//
// Ensure you cleanup after creating a worker:
// defer worker.Close()
func NewWorker(ctx context.Context, natsUrl string, streamName string, kubeConfPath string, requiresPortForward bool, logger zerolog.Logger) (*Worker, error) {
	// TODO: Handlers should really bootstrap themselves and be passed into the worker as args.
	// Given we've only got one at the moment, we set up here for simplicity

	workerLogger := logger.With().Str("from", "worker").Logger()

	worker := &Worker{
		logger:   workerLogger,
		handlers: map[string]handlerFunc{},
	}

	err := worker.initNats(ctx, natsUrl, streamName)
	if err != nil {
		return nil, err
	}

	err = worker.initK8sHandler(ctx, kubeConfPath, requiresPortForward)
	if err != nil {
		return nil, err
	}

	return worker, nil
}

// initNats initialises the nats connection, streams, and consumers required by a worker.
func (w *Worker) initNats(ctx context.Context, natsUrl string, streamName string) error {
	leaseConf := undist.LeaseConfig{
		NatsUrl:    natsUrl,
		StreamName: streamName,
	}
	leaseConf.MergeLeaseConfig(undist.DefaultLeaseConfig)

	nc, err := nats.Connect(
		leaseConf.NatsUrl,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(5),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return err
	}
	w.nc = nc

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Drain()
		return err
	}
	w.js = js

	stream, err := js.Stream(ctx, leaseConf.StreamName)
	if err != nil {
		return err
	}

	// Hardcoded to the k8s handler. Will need to take config from handlers in future
	requestConsConf := leaseConf.RequestConsumerConfig("k8s", "*")
	requestConsConf.MaxDeliver = 3
	requestConsumer, err := stream.CreateOrUpdateConsumer(ctx, requestConsConf)
	if err != nil {
		return err
	}

	w.consumer = requestConsumer

	w.handlerDeadline = requestConsumer.CachedInfo().Config.AckWait

	return nil
}

// initK8sHandler creates a new K8sHandler and registers it's task handler funcs
func (w *Worker) initK8sHandler(ctx context.Context, kubeConfPath string, requiresPortForward bool) error {
	// As we create more handler types, this should be moved out of the worker
	// and shift responsibility for setup onto the handlers themselves.
	k8sHandler, err := NewK8sHandler(ctx, kubeConfPath, w.sendResponse, requiresPortForward, w.logger)
	if err != nil {
		return err
	}
	w.handlers["k8s_run"] = k8sHandler.RunPod

	return nil
}

func (w *Worker) Run(ctx context.Context) error {
	callback := func(msg jetstream.Msg) {
		subject := msg.Subject()
		w.logger.Info().Msgf("Received request %s", subject)

		responseSubject, err := ParseResponseSubject(msg)
		if err != nil {
			w.logger.Error().Err(err).Msgf("Unable to handle request message: %s", subject)
			msg.Nak()
			return
		}

		response := &TaskResponse{
			StartedAt: time.Now(),
			subject:   responseSubject,
		}

		// Check if relevant, skip if not
		handler, err := w.getHandler(subject)
		if err != nil {
			w.logger.Error().Err(err).Msgf("Unable to handle request message: %s", subject)
			msg.Nak()
			return
		}
		if handler == nil {
			msg.Nak()
			return
		}

		// Attempt to run the task's handler, immediately respond with failure if not
		var replyErr error
		err = w.runHandler(ctx, msg, handler)
		if err != nil {
			w.logger.Error().Err(err).Msgf("Failed to handle request %s", subject)
			response.Status = "FAILURE"
			response.Error = err
			response.FinishedAt = time.Now()
			err, _ := w.sendResponse(ctx, response)
			replyErr = err
		}

		if replyErr != nil {
			w.logger.Error().Err(err).Msgf("Unable to send reply to request message: %s", subject)
			msg.Nak()
			return
		}

		err = undist.DoubleAck(ctx, msg)
		if err != nil {
			w.logger.Error().Err(err).Msgf("Unable to acknowledge request message: %s", subject)
		}

		w.logger.Debug().Msgf("Request message acknowledged (will not be re-sent) %s", subject)
	}

	consumerCtx, err := w.consumer.Consume(callback)
	if err != nil {
		return err
	}
	defer consumerCtx.Stop()

	w.logger.Info().Msg("Listening for requests")

	// Allow it to run until context is cancelled
	<-ctx.Done()

	return nil
}

func (w *Worker) runHandler(ctx context.Context, msg jetstream.Msg, handler handlerFunc) error {
	doneChan := make(chan bool)
	errChan := make(chan error)

	ticker := time.NewTicker(w.handlerDeadline - (w.handlerDeadline / 3))
	defer ticker.Stop()

	go func() {
		err := handler(ctx, msg)
		if err != nil {
			errChan <- err
			return
		}

		doneChan <- true
	}()

	// Immediately extend redelivery window so we can start from a known duration
	msg.InProgress()

	for {
		select {
		// Periodically extend the ack deadline whilst we work
		case <-ticker.C:
			err := msg.InProgress()
			if err != nil {
				return err
			}

		// Exit when done
		case <-doneChan:
			return nil

		// Or return the error in case of failure
		case err := <-errChan:
			return err
		}
	}
}

// sendResponse takes a task response and replies to the original request msg on the appropriate subject
//
// If sending a reply fails, the _original_ message will be NAK'd and retried
// as this is a system failure rather than a handler failure.
// Deciding the logic in the case of a handler failure is the responsibility of end users
// and these errors should not be hidden from them.
func (w *Worker) sendResponse(ctx context.Context, response *TaskResponse) (error, bool) {
	sent := false

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return err, sent
	}

	_, err = w.js.Publish(ctx, response.subject, responseBytes)
	// It's janky catching the error by string, but NATS doesn't expose this one as a type
	// and anything else is effectively doing the same thing with extra steps.
	if err != nil && strings.Contains(err.Error(), "maximum messages per subject exceeded") {
		err = nil
		sent = false
	}

	return err, sent
}

// getHandler returns the handler function that matches the request subject
func (w *Worker) getHandler(subject string) (handlerFunc, error) {
	appName, handlerName, err := ParseAppHandler(subject)
	if err != nil {
		return nil, err
	}

	handlerKey := fmt.Sprintf("%s_%s", appName, handlerName)

	handler, ok := w.handlers[handlerKey]
	if !ok {
		handler = nil
	}

	return handler, nil
}

// Close should be called with defer worker.Close() after creation of the worker
func (w *Worker) Close() {
	if w.nc != nil {
		defer w.nc.Drain()
	}
}
