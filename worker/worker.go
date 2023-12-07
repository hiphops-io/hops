package worker

import (
	"context"
	"time"

	"github.com/hiphops-io/hops/nats"
	"github.com/nats-io/nats.go/jetstream"
)

type (
	App interface {
		AppName() string
		Handlers() map[string]Handler
	}

	// TODO: Update function to return a pointer to a ResultMsg
	Handler func(context.Context, jetstream.Msg) error

	Worker struct {
		app        App
		logger     Logger
		natsClient *nats.Client
		handlers   map[string]Handler
	}
)

func NewWorker(natsClient *nats.Client, app App, logger Logger) *Worker {
	w := &Worker{
		app:        app,
		logger:     logger,
		natsClient: natsClient,
	}

	w.handlers = app.Handlers()

	return w
}

func (w *Worker) Run(ctx context.Context) error {
	consumerName := w.app.AppName()
	// Get the ack deadline
	ackDeadline := w.natsClient.Consumers[consumerName].CachedInfo().Config.AckWait

	callback := func(msg jetstream.Msg) {
		startedAt := time.Now()

		subject := msg.Subject()
		w.logger.Infof("Received request %s", subject)

		parsedMsg, err := nats.Parse(msg)
		if err != nil {
			w.logger.Errf(err, "Unable to handle request message: %s", subject)
			msg.Nak()
			return
		}

		// Get the handler function if it exists. Terminate if not as there's nothing
		// to be done.
		handler, ok := w.handlers[parsedMsg.HandlerName]
		if !ok {
			w.logger.Warnf("Unknown handler call '%s' in msg '%s'", parsedMsg.HandlerName, subject)
			msg.Term()
			return
		}

		// Attempt to run the task's handler, immediately respond with failure if not
		var replyErr error
		err = w.runHandler(ctx, msg, handler, ackDeadline)
		if err != nil {
			w.logger.Errf(err, "Failed to handle request %s", subject)
			err, _ := w.natsClient.PublishResult(ctx, startedAt, nil, err, parsedMsg.ResponseSubject())
			replyErr = err
		}

		if replyErr != nil {
			w.logger.Errf(err, "Unable to send reply to request message: %s", subject)
			msg.Nak()
			return
		}

		err = nats.DoubleAck(ctx, msg)
		if err != nil {
			w.logger.Errf(err, "Unable to acknowledge request message: %s", subject)
			msg.NakWithDelay(3 * time.Second)
		}

		w.logger.Debugf("Request message acknowledged (will not be re-sent) %s", subject)
	}

	w.logger.Infof("Listening for requests")

	// Blocks until cancelled or errors
	return w.natsClient.Consume(ctx, consumerName, callback)
}

// runHandler runs a WorkHandler function whilst automatically extending the ack deadline until completion
func (w *Worker) runHandler(ctx context.Context, msg jetstream.Msg, handler Handler, deadline time.Duration) error {
	doneChan := make(chan bool)
	errChan := make(chan error)

	// We'll extend the deadline when there's a third of the duration left
	ticker := time.NewTicker(deadline - (deadline / 3))
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

		case <-doneChan:
			return nil

		case err := <-errChan:
			return err
		}
	}
}
