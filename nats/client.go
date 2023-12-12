package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	ChannelNotify  = "notify"
	ChannelRequest = "request"

	DefaultConsumerName = "runner"
	// How far back to look for events by default
	DefaultEventLookback = -time.Hour

	// Number of events returned max
	GetEventHistoryEventLimit = 100

	// limits for GetEventHistory
	defaultBatchSize = 160
	maxWaitTime      = time.Second
)

var nameReplacer = strings.NewReplacer("*", "all", ".", "dot", ">", "children")

type (
	Client struct {
		Consumers   map[string]jetstream.Consumer
		JetStream   jetstream.JetStream
		NatsConn    *nats.Conn
		SysObjStore nats.ObjectStore
		accountId   string
		logger      Logger
		streamName  string
	}

	// ClientOpt functions configure a nats.Client via NewClient()
	ClientOpt func(*Client) error

	// MessageBundle is a map of messageIDs and the data that message contained
	//
	// MessageBundle is designed to be passed to a runner to ensure it has the aggregate state
	// of a hiphops sequence of messages.
	MessageBundle map[string][]byte

	// SequenceHandler is a function that receives the sequenceId and message bundle for a sequence of messages
	SequenceHandler interface {
		SequenceCallback(context.Context, string, MessageBundle) error
	}
)

// NewClient returns a new hiphops specific NATS client
//
// By default it is configured as a runner consumer (listening for incoming source events)
// Passing *any* ClientOpts will override this default.
func NewClient(natsUrl string, accountId string, logger Logger, clientOpts ...ClientOpt) (*Client, error) {
	ctx := context.Background()

	natsClient := &Client{
		Consumers: map[string]jetstream.Consumer{},
		accountId: accountId,
		// Override this using WithStreamName ClientOpt if required.
		streamName: nameReplacer.Replace(accountId),
		logger:     logger,
	}
	err := natsClient.initNatsConnection(natsUrl)
	if err != nil {
		return nil, err
	}

	err = natsClient.initJetStream()
	if err != nil {
		defer natsClient.Close()
		return nil, err
	}

	err = natsClient.initObjectStore(ctx, accountId)
	if err != nil {
		defer natsClient.Close()
		return nil, err
	}

	if len(clientOpts) == 0 {
		clientOpts = DefaultClientOpts()
	}

	for _, opt := range clientOpts {
		err := opt(natsClient)
		if err != nil {
			defer natsClient.Close()
			return nil, err
		}
	}

	return natsClient, err
}

func (c *Client) CheckConnection() bool {
	// TODO: Enhance this with more meaningful checks (e.g. sending a message back and forth)
	return c.NatsConn.IsConnected()
}

func (c *Client) Close() {
	c.NatsConn.Drain()
}

// Consume consumes messages from the HopsNats.Consumers[fromConsumer]
//
// This will block the calling goroutine until the context is cancelled
// and can be ran as a long-lived service
func (c *Client) Consume(ctx context.Context, fromConsumer string, callback jetstream.MessageHandler) error {
	consumer, found := c.Consumers[fromConsumer]
	if !found {
		return fmt.Errorf("Consumer '%s' not found on client", fromConsumer)
	}

	consumerCtx, err := consumer.Consume(callback)
	if err != nil {
		return err
	}
	defer consumerCtx.Stop()

	// Run until context cancelled
	<-ctx.Done()

	return nil
}

// ConsumeSequences is a wrapper around consume that presents the aggregate state of a sequence to the callback
// instead of individual messages.
func (c *Client) ConsumeSequences(ctx context.Context, fromConsumer string, handler SequenceHandler) error {
	wrappedCB := func(msg jetstream.Msg) {
		hopsMsg, err := Parse(msg)
		if err != nil {
			// If parsing is failing, there's no point retrying the message
			msg.Term()
			c.logger.Errf(err, "Unable to parse message")
			return
		}

		if hopsMsg.MessageId == HopsMessageId {
			c.logger.Debugf("Skipping 'hops assignment' message")

			err := DoubleAck(ctx, msg)
			if err != nil {
				c.logger.Errf(err, "Unable to ack 'hops assignment' message")
			}

			return
		}

		if hopsMsg.Done {
			// TODO: Actually finalise the pipeline here
			c.logger.Debugf("Skipping 'pipeline done' message")

			err := DoubleAck(ctx, msg)
			if err != nil {
				c.logger.Errf(err, "Unable to ack 'pipeline done' message")
			}

			return
		}

		msgBundle, err := c.FetchMessageBundle(ctx, hopsMsg)
		if err != nil {
			msg.NakWithDelay(3 * time.Second)
			c.logger.Errf(err, "Unable to fetch message bundle")
			return
		}

		err = handler.SequenceCallback(ctx, hopsMsg.SequenceId, msgBundle)
		if err != nil {
			c.logger.Errf(err, "Failed to process message")
			msg.NakWithDelay(3 * time.Second)
			return
		}

		DoubleAck(ctx, msg)
	}

	return c.Consume(ctx, fromConsumer, wrappedCB)
}

// FetchMessageBundle pulls all historic messages for a sequenceId from the stream, converting them to a message bundle
//
// The returned message bundle will contain all previous messages in addition to the newly received message
func (c *Client) FetchMessageBundle(ctx context.Context, newMsg *MsgMeta) (MessageBundle, error) {
	filter := newMsg.SequenceFilter()

	// TODO: Create a deadline for the context
	consumerConf := jetstream.OrderedConsumerConfig{
		FilterSubjects: []string{filter},
		DeliverPolicy:  jetstream.DeliverAllPolicy,
	}
	cons, err := c.JetStream.OrderedConsumer(ctx, c.streamName, consumerConf)
	if err != nil {
		return nil, fmt.Errorf("Unable to create ordered consumer: %w", err)
	}

	msgBundle := MessageBundle{}

	msgCtx, err := cons.Messages()
	if msgCtx != nil {
		defer msgCtx.Stop()
	}
	if err != nil {
		return nil, fmt.Errorf("Unable to read back messages: %w", err)
	}

	for {
		// Get the next message in the sequence
		m, err := msgCtx.Next()
		if err != nil {
			return nil, err
		}

		// Parse the important bits for easy handling
		msg, err := Parse(m)
		if err != nil {
			return nil, err
		}

		// Ensure we've not surpassed the nats message sequence we're reading up to
		if msg.StreamSequence > newMsg.StreamSequence {
			return nil, fmt.Errorf("Unable to find original message with NATS sequence of: %d", newMsg.StreamSequence)
		}

		// Add to the message bundle
		msgBundle[msg.MessageId] = m.Data()

		// If we're at the newMsg, we can stop
		if msg.StreamSequence == newMsg.StreamSequence {
			break
		}
	}

	return msgBundle, nil
}

// GetEventHistory pulls historic events, most recent first, from now back to start time.
//
// Times out if events take longer than a second to be received.
// Only returns the first 100 events. (const GetEventHistoryEventLimit)
// If sourceOnly is true, only returns source events (i.e. not pipeline events)
func (c *Client) GetEventHistory(ctx context.Context, start time.Time, sourceOnly bool) ([]*MsgMeta, error) {
	events := []*MsgMeta{}
	var eventId string

	if sourceOnly {
		eventId = SourceEventId
	} else {
		eventId = AllEventId
	}

	consumerConf := jetstream.OrderedConsumerConfig{
		FilterSubjects: []string{EventLogSubject(c.accountId, eventId)},
		DeliverPolicy:  jetstream.DeliverByStartTimePolicy,
		OptStartTime:   &start,
	}
	cons, err := c.JetStream.OrderedConsumer(ctx, c.streamName, consumerConf)
	if err != nil {
		return nil, fmt.Errorf("Unable to create ordered consumer: %w", err)
	}

	info, err := cons.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("Unable to get consumer info: %w", err)
	}

	numPending := int(info.NumPending)
	if numPending == 0 {
		return events, nil
	}

	for {
		// Don't call more than is in the stream (otherwise have to wait for timeout)
		var batchSize int
		if numPending > defaultBatchSize {
			batchSize = defaultBatchSize
		} else {
			batchSize = numPending
		}

		msgs, err := cons.Fetch(batchSize, jetstream.FetchMaxWait(maxWaitTime))
		if err != nil {
			return nil, fmt.Errorf("Unable to fetch messages: %w", err)
		}

		for rawM := range msgs.Messages() {
			m, err := Parse(rawM)
			if err != nil {
				c.logger.Errf(err, "Unable to parse message")
				return nil, err
			}

			// count down so we don't have to timeout on the last fetch
			numPending--

			// Prepend to the events
			events = append([]*MsgMeta{m}, events...)
		}

		// Keep only the first 100 items
		if len(events) > 100 {
			events = events[:100]
		}

		// If we've got all the events, we can stop
		if numPending == 0 {
			break
		}
	}

	c.logger.Debugf("Events received %d", len(events))

	return events, nil
}

func (c *Client) GetMsg(ctx context.Context, subjTokens ...string) (*jetstream.RawStreamMsg, error) {
	stream, err := c.JetStream.Stream(ctx, c.streamName)
	if err != nil {
		return nil, err
	}

	tokens := append([]string{c.accountId}, subjTokens...)
	subject := strings.Join(tokens, ".")

	return stream.GetLastMsgForSubject(ctx, subject)
}

func (c *Client) GetSysObject(key string) ([]byte, error) {
	return c.SysObjStore.GetBytes(key)
}

func (c *Client) Publish(ctx context.Context, data []byte, subjTokens ...string) (*jetstream.PubAck, bool, error) {
	sent := true
	subject := ""
	isFullSubject := len(subjTokens) == 1 && strings.Contains(subjTokens[0], ".")

	// If we have individual subject tokens, construct into string and prefix with accountId
	if !isFullSubject {
		tokens := append([]string{c.accountId}, subjTokens...)
		subject = strings.Join(tokens, ".")
	} else {
		subject = subjTokens[0]
	}

	puback, err := c.JetStream.Publish(ctx, subject, data)
	if err != nil && strings.Contains(err.Error(), "maximum messages per subject exceeded") {
		err = nil
		sent = false
		c.logger.Debugf("Skipping duplicate message %s", subject)
	} else if err == nil {
		c.logger.Debugf("Message sent %s", subject)
	}

	return puback, sent, err
}

// PublishResult is a convenience wrapper that json encodes a ResultMsg and publishes it
func (c *Client) PublishResult(ctx context.Context, startedAt time.Time, result interface{}, err error, subjTokens ...string) (error, bool) {
	resultMsg := NewResultMsg(startedAt, result, err)
	resultBytes, err := json.Marshal(resultMsg)
	if err != nil {
		return err, false
	}

	_, sent, err := c.Publish(ctx, resultBytes, subjTokens...)
	return err, sent
}

func (c *Client) PutSysObject(name string, data []byte) (*nats.ObjectInfo, error) {
	return c.SysObjStore.PutBytes(name, data)
}

func (c *Client) initJetStream() error {
	js, err := jetstream.New(c.NatsConn)
	if err != nil {
		return err
	}

	c.JetStream = js
	return nil
}

func (c *Client) initNatsConnection(natsUrl string) error {
	nc, err := nats.Connect(
		natsUrl,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(5),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return err
	}

	c.NatsConn = nc
	return nil
}

// initObjectStore initialises required system object store in NATS for a client
func (c *Client) initObjectStore(ctx context.Context, accountId string) error {
	js, err := c.NatsConn.JetStream()
	if err != nil {
		return err
	}

	sysObjConf := nats.ObjectStoreConfig{
		Bucket: "system",
	}
	sysObj, err := js.CreateObjectStore(&sysObjConf)
	if err != nil {
		return err
	}

	// Set the system object store on the client
	c.SysObjStore = sysObj
	return nil
}

// ClientOpts - passed through to NewClient() to configure the client setup

// DefaultClientOpts configures the hiphops nats.Client as a RunnerClient
func DefaultClientOpts() []ClientOpt {
	return []ClientOpt{
		WithRunner(DefaultConsumerName),
	}
}

// WithReplay initialises the client with a consumer for replaying a sequence
func WithReplay(name string, sequenceId string) ClientOpt {
	return func(c *Client) error {
		ctx := context.Background() // TODO: Move all context creation in ClientOpts to argument rather than in function

		// Get the source message from the stream
		stream, err := c.JetStream.Stream(ctx, c.streamName)
		if err != nil {
			return err
		}

		// Get the source message to be replayed from the stream
		sourceMsgSubject := SourceEventSubject(c.accountId, sequenceId)
		rawMsg, err := stream.GetLastMsgForSubject(ctx, sourceMsgSubject)
		if err != nil {
			return fmt.Errorf("Failed to fetch source event: %w", err)
		}
		if rawMsg == nil {
			return fmt.Errorf("No source event found for subject '%s'", sourceMsgSubject)
		}

		// Create a new, random replay sequence ID
		replaySequenceId := fmt.Sprintf("replay-%s", uuid.NewString()[:20])

		// Create ephemeral consumer filtered by replayed sequence ID
		consumerCfg := jetstream.ConsumerConfig{
			Name:          replaySequenceId,
			Description:   fmt.Sprintf("Replay request for sequence: '%s'", sequenceId),
			FilterSubject: ReplayFilterSubject(c.accountId, replaySequenceId),
			DeliverPolicy: jetstream.DeliverAllPolicy,
		}
		consumer, err := c.JetStream.CreateConsumer(ctx, c.streamName, consumerCfg)
		if err != nil {
			return err
		}

		// Publish the source message with replayed sequence ID so it's picked up by
		// ephemeral consumer
		c.Publish(ctx, rawMsg.Data, ChannelNotify, replaySequenceId, "event")

		// Set the consumer on the client
		c.Consumers[name] = consumer
		return nil
	}
}

// WithRunner initialises the client with a consumer for running pipelines
func WithRunner(name string) ClientOpt {
	return func(c *Client) error {
		ctx := context.Background()

		consumerName := fmt.Sprintf("%s-%s", c.accountId, ChannelNotify)
		consumerName = nameReplacer.Replace(consumerName)

		consumer, err := c.JetStream.Consumer(ctx, c.streamName, consumerName)
		if err != nil {
			return err
		}

		c.Consumers[name] = consumer
		return nil
	}
}

// WithStreamName overrides the stream name to be used (which default to accountId otherwise)
//
// Should be given before any ClientOpts that use the stream,
// as otherwise they will be initialised with the default stream name
func WithStreamName(name string) ClientOpt {
	return func(c *Client) error {
		c.streamName = name
		return nil
	}
}

// WithWorker initialises the client with a consumer to receive call requests for a worker
func WithWorker(appName string) ClientOpt {
	return func(c *Client) error {
		ctx := context.Background()

		name := fmt.Sprintf("%s-%s-%s", c.accountId, ChannelRequest, appName)
		name = nameReplacer.Replace(name)

		// Create or update the consumer, since these are created dynamically
		consumerCfg := jetstream.ConsumerConfig{
			Name:          name,
			Durable:       name,
			FilterSubject: WorkerRequestSubject(c.accountId, appName, "*"),
			AckWait:       1 * time.Minute,
		}
		consumer, err := c.JetStream.CreateOrUpdateConsumer(ctx, c.streamName, consumerCfg)
		if err != nil {
			return err
		}

		c.Consumers[appName] = consumer
		return nil
	}
}
