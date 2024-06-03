package nats

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"golang.org/x/sync/errgroup"
)

var (
	ErrEventFatal          = errors.New("unrecoverable error handling event")
	nameReplacer           = strings.NewReplacer("*", "all", ".", "dot", ">", "children")
	wrongSequenceErrString = fmt.Sprintf("err_code=%d", jetstream.JSErrCodeStreamWrongLastSequence)
)

type (
	Client struct {
		JetStream jetstream.JetStream
		NatsConn  *nats.Conn
	}

	// MessageHandler is a callback function provided to Client.Consume to handle messages
	MessageHandler func(ctx context.Context, msgData []byte, msgMeta *MsgMeta, ackDeadline time.Duration) error
)

func NewClient(natsUrl string) (*Client, error) {
	conn, err := Connect(natsUrl)
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(conn)
	if err != nil {
		defer conn.Drain()
		return nil, err
	}

	c := &Client{
		JetStream: js,
		NatsConn:  conn,
	}

	return c, nil
}

func (c *Client) CheckConnection() bool {
	return c.NatsConn.IsConnected()
}

// Close closes the nats connection
//
// Note: An error will only be returned if closing doesn't complete within 1m.
// It is possible for the client to close after erroring, so it should
// never be re-used after calling Close.
func (c *Client) Close() error {
	if c.NatsConn.IsClosed() {
		return nil
	}

	closedChan := make(chan struct{})
	c.NatsConn.Opts.ClosedCB = func(*nats.Conn) {
		closedChan <- struct{}{}
	}

	c.NatsConn.Drain()

	select {
	case <-closedChan:
		return nil
	case <-time.After(1 * time.Minute):
		return errors.New("client connection did not close within timeout")
	}
}

// Consume pulls messages from the stream via the consumer, pre-processes and calls the given handler
//
// TODO: Investigate if this is the best place to handle automatically extending the ack deadline.
// If so then do that and potentially remove deadline from handler signature. If not then remove this note
func (c *Client) Consume(
	ctx context.Context,
	consumer jetstream.Consumer,
	handler MessageHandler,
) error {
	g, ctx := errgroup.WithContext(ctx)
	// Note: In future we should make the active goroutine limit configurable
	g.SetLimit(20)

	deadline := consumer.CachedInfo().Config.AckWait

	callback := func(msg jetstream.Msg) {
		msgMeta, err := Parse(msg)
		if err != nil {
			// There's no way to recover an unparseable event
			msg.TermWithReason(fmt.Sprintf("unable to parse event: %s", err.Error()))
			return
		}

		g.Go(func() error {
			if err := handler(ctx, msg.Data(), msgMeta, deadline); err != nil {
				if errors.Is(err, ErrEventFatal) {
					msg.TermWithReason(err.Error())
					return nil
				}

				msg.NakWithDelay(3 * time.Second)
				return nil
			}

			DoubleAck(ctx, msg)

			return nil
		})
	}

	consumerCtx, err := consumer.Consume(callback)
	if err != nil {
		return err
	}
	defer consumerCtx.Stop()

	<-ctx.Done()

	return nil
}

func (c *Client) Publish(ctx context.Context, data []byte, subject string) (*jetstream.PubAck, bool, error) {
	sent := true

	puback, err := c.JetStream.Publish(ctx, subject, data, jetstream.WithExpectLastSequencePerSubject(0))
	if err != nil {
		sent = false

		if strings.Contains(err.Error(), wrongSequenceErrString) {
			// Wrong last sequence error is expected in normal operation. It is how we
			// ensure idempotency at message creation level.
			err = nil
		}
	}

	return puback, sent, err
}

// PublishResult publishes a result message for a given request message
//
// TODO: Might be okay to delete this
func (c *Client) PublishResult(
	ctx context.Context,
	request jetstream.Msg,
	result interface{},
	err error,
	subjectTokens ...string,
) (error, bool) {
	// Note: We can use request.Metadata() Timestamp to decide when the original request was made
	// paired with time.Now() we can calculate latency and add it to result messages.

	return nil, false
}

// ReplayConsumer returns a consumer for replaying events
func (c *Client) ReplayConsumer(ctx context.Context, sequenceId string) (jetstream.Consumer, error) {
	// Create a new, random replay sequence ID
	replaySequenceId := fmt.Sprintf("replay-%s", uuid.NewString()[:20])

	stream, err := c.JetStream.Stream(ctx, ChannelNotify)
	if err != nil {
		return nil, err
	}

	// Get the source message to be replayed from the stream
	rawMsg, err := stream.GetLastMsgForSubject(ctx, SourceEventSubject(sequenceId))
	if err != nil || rawMsg == nil {
		return nil, fmt.Errorf("Failed to fetch source event for '%s': %w", sequenceId, err)
	}

	// Create ephemeral consumer filtered by replayed sequence ID
	consumerCfg := jetstream.ConsumerConfig{
		Name:          replaySequenceId,
		Description:   fmt.Sprintf("Replaying event: '%s'", sequenceId),
		FilterSubject: ReplayFilterSubject(replaySequenceId),
		DeliverPolicy: jetstream.DeliverAllPolicy,
	}
	consumer, err := c.JetStream.CreateConsumer(ctx, ChannelNotify, consumerCfg)
	if err != nil {
		return nil, err
	}

	// Publish the source message with replayed sequence ID so it's picked up by
	// ephemeral consumer
	c.Publish(ctx, rawMsg.Data, SourceEventSubject(replaySequenceId))

	return consumer, nil
}

// RunnerConsumer returns a consumer for the `notify` stream
func (c *Client) RunnerConsumer(ctx context.Context) (jetstream.Consumer, error) {
	cfg := jetstream.ConsumerConfig{
		Name:          ChannelNotify,
		Durable:       ChannelNotify,
		FilterSubject: NotifyFilterSubject(),
		DeliverPolicy: jetstream.DeliverNewPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       time.Minute * 1,
		MaxDeliver:    5,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
	}

	return c.JetStream.CreateOrUpdateConsumer(ctx, ChannelNotify, cfg)
}

// WorkerConsumer returns a consumer for the `request` stream
//
// The consumer will filter requests by the given appName
// If durable is true, then the consumer created will be a durable one, otherwise it will be ephemeral.
func (c *Client) WorkerConsumer(ctx context.Context, appName string, durable bool) (jetstream.Consumer, error) {
	name := fmt.Sprintf("%s-%s", ChannelRequest, appName)
	name = nameReplacer.Replace(name)

	cfg := jetstream.ConsumerConfig{
		Name:          name,
		FilterSubject: WorkerRequestFilterSubject(appName, "*"),
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       1 * time.Minute,
		MaxDeliver:    120, // Two hours of redelivery attempts
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
	}

	if durable {
		cfg.Durable = name
	}

	return c.JetStream.CreateOrUpdateConsumer(ctx, ChannelRequest, cfg)
}

// Connect establishes a NATS connection, retrying on failed connect attempts
func Connect(natsUrl string) (*nats.Conn, error) {
	nc, err := nats.Connect(
		natsUrl,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(5),
		nats.ReconnectWait(5*time.Second),
	)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

// DoubleAck is a convenience wrapper around NATS acking with a timeout
func DoubleAck(ctx context.Context, msg jetstream.Msg) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return msg.DoubleAck(ctx)
}
