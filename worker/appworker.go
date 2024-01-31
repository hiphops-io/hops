package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/hiphops-io/hops/nats"
	"github.com/nats-io/nats.go/jetstream"
)

type (
	AppWorker struct {
		ackWait    time.Duration
		appName    string
		handlers   Handlers
		logger     Logger
		natsClient *nats.Client
		workChan   chan requestMsg
	}

	HandlerFunc func([]byte, *nats.MsgMeta) (Executor, error)

	Executor func(context.Context) (interface{}, error)

	Handlers map[string]HandlerFunc

	requestMsg struct {
		executor        Executor
		msg             jetstream.Msg
		responseSubject string
		startedAt       time.Time
	}
)

func NewAppWorker(appName string, handlers Handlers, bufferSize int, natsClient *nats.Client, logger Logger) *AppWorker {
	a := &AppWorker{
		appName:    appName,
		handlers:   handlers,
		logger:     logger,
		natsClient: natsClient,
		workChan:   make(chan requestMsg, bufferSize),
		ackWait:    natsClient.Consumers[appName].CachedInfo().Config.AckWait,
	}

	return a
}

func (a *AppWorker) Run(ctx context.Context) {
	go a.listenForRequests(ctx)
	go a.processWork(ctx)

	<-ctx.Done()
}

func (a *AppWorker) listenForRequests(ctx context.Context) {
	callback := func(msg jetstream.Msg) {
		startedAt := time.Now()

		subject := msg.Subject()
		a.logger.Infof("Received request %s", subject)

		parsedMsg, err := nats.Parse(msg)
		if err != nil {
			a.logger.Errf(err, "Unable to handle request message: %s", subject)
			a.natsClient.PublishResultWithAck(
				ctx,
				msg,
				startedAt,
				nil,
				err,
				parsedMsg.ResponseSubject(),
			)
			return
		}

		// Get the handler function if it exists. If not, immediately fail
		handler, ok := a.handlers[parsedMsg.HandlerName]
		if !ok {
			handlerErr := fmt.Errorf("Unknown handler call '%s' in msg '%s'", parsedMsg.HandlerName, subject)
			a.logger.Errf(handlerErr, "Failed to handle request")

			a.natsClient.PublishResultWithAck(
				ctx,
				msg,
				startedAt,
				nil,
				handlerErr,
				parsedMsg.ResponseSubject(),
			)
			return
		}

		// Parse the payload with the handler
		executor, err := handler(msg.Data(), parsedMsg)
		if err != nil {
			a.logger.Errf(err, "Failed to parse request")
			a.natsClient.PublishResultWithAck(
				ctx,
				msg,
				startedAt,
				nil,
				err,
				parsedMsg.ResponseSubject(),
			)
			return
		}

		request := requestMsg{
			msg:             msg,
			startedAt:       startedAt,
			executor:        executor,
			responseSubject: parsedMsg.ResponseSubject(),
		}

		a.workChan <- request
	}

	a.logger.Infof("Starting to listen for requests")

	// Blocks until cancelled or errors
	err := a.natsClient.Consume(ctx, a.appName, callback)
	if err != nil {
		a.logger.Errf(err, "Consuming messages failed for app %s", a.appName)
	}

	close(a.workChan)
}

func (a *AppWorker) processWork(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-a.workChan:
			a.executeRequest(ctx, request)
		}
	}
}

func (a *AppWorker) executeRequest(ctx context.Context, request requestMsg) {
	// Immediately extend redelivery before commencing work
	err := request.msg.InProgress()
	if err != nil {
		// Abort as the message will either be re-sent or has already been handled
		return
	}

	// We'll extend the deadline when there's a third of the duration left
	ticker := time.NewTicker(a.ackWait - (a.ackWait / 3))
	defer ticker.Stop()

	errChan := make(chan error)
	resultChan := make(chan interface{})

	// Execute the actual request handling code
	go func() {
		result, err := request.executor(ctx)
		if err != nil {
			errChan <- err
		}

		resultChan <- result
	}()

	var result interface{}
	var responseErr error

	// Now keep the message pending until the request code fails or succeeds, handling the response
runRequest:
	for {
		select {
		case <-ticker.C:
			err := request.msg.InProgress()
			if err != nil {
				// Abort as the message will either be re-sent or has already been handled
				return
			}

		case result = <-resultChan:
			_, responseErr = a.natsClient.PublishResultWithAck(
				ctx,
				request.msg,
				request.startedAt,
				result,
				nil,
				request.responseSubject,
			)
			break runRequest

		case err = <-errChan:
			_, responseErr = a.natsClient.PublishResultWithAck(
				ctx,
				request.msg,
				request.startedAt,
				nil,
				err,
				request.responseSubject,
			)
			break runRequest
		}
	}

	if responseErr != nil {
		a.logger.Warnf("Failed to send result: %s", responseErr.Error())
	}
}
