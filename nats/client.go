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

	// Interest topic which is used by default
	DefaultInterestTopic = "default"

	// Number of events returned max
	GetEventHistoryEventLimit = 100

	// limits for GetEventHistory
	defaultBatchSize = 160
	maxWaitTime      = time.Second
)

var nameReplacer = strings.NewReplacer("*", "all", ".", "dot", ">", "children")

type (
	Client struct {
		Consumers     map[string]jetstream.Consumer
		JetStream     jetstream.JetStream
		NatsConn      *nats.Conn
		SysObjStore   nats.ObjectStore
		accountId     string
		interestTopic string
		logger        Logger
		stream        jetstream.Stream
		streamName    string
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
		SequenceCallback(context.Context, string, MessageBundle) (bool, error)
	}
)

// NewClient returns a new hiphops specific NATS client
//
// By default it is configured as a runner consumer (listening for incoming source events)
// Passing *any* ClientOpts will override this default.
func NewClient(natsUrl string, accountId string, interestTopic string, logger Logger, clientOpts ...ClientOpt) (*Client, error) {
	ctx := context.Background()

	natsClient := &Client{
		Consumers:     map[string]jetstream.Consumer{},
		accountId:     accountId,
		interestTopic: interestTopic,
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

	// We initialise the stream after applying clientopts, as they may alter
	// the stream we need to create.
	err = natsClient.initStream(ctx)
	if err != nil {
		defer natsClient.Close()
		return nil, err
	}

	logger.Debugf("Interest topic is: %s", natsClient.interestTopic)

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

		if hopsMsg.MessageId == HopsMessageId || hopsMsg.Done {
			err := DoubleAck(ctx, msg)
			if err != nil {
				c.logger.Errf(err, "Unable to acknowledge message: %s", msg.Subject())
			}

			return
		}

		msgBundle, err := c.FetchMessageBundle(ctx, hopsMsg)
		if err != nil {
			msg.NakWithDelay(3 * time.Second)
			c.logger.Errf(err, "Unable to fetch message bundle")
			return
		}

		handled, err := handler.SequenceCallback(ctx, hopsMsg.SequenceId, msgBundle)
		if err != nil {
			c.logger.Errf(err, "Failed to process message")
			msg.NakWithDelay(3 * time.Second)
			return
		}

		// Immediately clean up source events we have no handler for
		// we check for two events in the bundle as it should have a hops
		// assignment message too.
		if !handled && len(msgBundle) <= 2 {
			go c.DeleteMsgSequence(ctx, hopsMsg)
			return
		}

		DoubleAck(ctx, msg)
	}

	return c.Consume(ctx, fromConsumer, wrappedCB)
}

// DeleteMsgSequence deletes a given message and the entire sequence it is part of
//
// Main use case is preventing build up of source events that do not relate to any
// configured automation
func (c *Client) DeleteMsgSequence(ctx context.Context, msgMeta *MsgMeta) error {
	err := c.stream.Purge(ctx, jetstream.WithPurgeSubject(msgMeta.SequenceFilter()))
	if err != nil {
		c.logger.Errf(err, "Unable to delete sequence %s", msgMeta.SequenceId)
	}

	return err
}

// FetchMessageBundle pulls all historic messages for a sequenceId from the stream, converting them to a message bundle
//
// The returned message bundle will contain all previous messages in addition to the newly received message
func (c *Client) FetchMessageBundle(ctx context.Context, incomingMsg *MsgMeta) (MessageBundle, error) {
	filter := incomingMsg.SequenceFilter()

	// TODO: Create a deadline for the context
	consumerConf := jetstream.OrderedConsumerConfig{
		FilterSubjects:    []string{filter},
		DeliverPolicy:     jetstream.DeliverAllPolicy,
		InactiveThreshold: time.Millisecond * 500,
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
		if msg.StreamSequence > incomingMsg.StreamSequence {
			return nil, fmt.Errorf("Unable to find original message with NATS sequence of: %d", incomingMsg.StreamSequence)
		}

		// Add to the message bundle
		msgBundle[msg.MessageId] = m.Data()

		// If we're at the newMsg, we can stop
		if msg.StreamSequence == incomingMsg.StreamSequence {
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
	rawEvents := []jetstream.Msg{}
	events := []*MsgMeta{}
	var eventId string

	if sourceOnly {
		eventId = SourceEventId
	} else {
		eventId = AllEventId
	}

	consumerConf := jetstream.OrderedConsumerConfig{
		FilterSubjects:    []string{EventLogFilterSubject(c.accountId, c.interestTopic, eventId)},
		DeliverPolicy:     jetstream.DeliverByStartTimePolicy,
		InactiveThreshold: time.Millisecond * 500,
		OptStartTime:      &start,
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
			// count down so we don't have to timeout on the last fetch
			numPending--

			// Append to the events
			rawEvents = append(rawEvents, rawM)
		}

		// Keep only the most recent (last) 100 items
		if len(rawEvents) > 100 {
			rawEvents = rawEvents[len(rawEvents)-100:]
		}

		// If we've got all the events, we can stop
		if numPending == 0 {
			break
		}
	}

	c.logger.Debugf("Events received %d", len(rawEvents))

	// Parse the events in reverse order (most recent first)
	for i := len(rawEvents) - 1; i >= 0; i-- {
		rawM := rawEvents[i]
		m, err := Parse(rawM)
		if err != nil {
			c.logger.Errf(err, "Unable to parse message")
			return nil, err
		}

		events = append(events, m)
	}

	return events, nil
}

func (c *Client) GetMsg(ctx context.Context, subjTokens ...string) (*jetstream.RawStreamMsg, error) {
	subject := c.buildSubject(subjTokens...)

	return c.stream.GetLastMsgForSubject(ctx, subject)
}

func (c *Client) GetSysObject(key string) ([]byte, error) {
	return c.SysObjStore.GetBytes(key)
}

func (c *Client) Publish(ctx context.Context, data []byte, subjTokens ...string) (*jetstream.PubAck, bool, error) {
	sent := true
	subject := ""
	isFullSubject := len(subjTokens) == 1 && strings.Contains(subjTokens[0], ".")

	// If we have individual subject tokens, construct into string and prefix with accountId and interestTopic
	if !isFullSubject {
		subject = c.buildSubject(subjTokens...)
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

// Deprecated: PublishResult is a convenience wrapper that json encodes a ResultMsg and publishes it
//
// In most cases you should use PublishResultWithAck instead, deferring acking of the original messaging
// until after we've sent a result.
// This method will be removed in future.
func (c *Client) PublishResult(ctx context.Context, startedAt time.Time, result interface{}, err error, subjTokens ...string) (error, bool) {
	resultMsg, ok := result.(ResultMsg)
	if !ok {
		resultMsg = NewResultMsg(startedAt, result, err)
	}

	resultBytes, err := json.Marshal(resultMsg)
	if err != nil {
		return err, false
	}

	_, sent, err := c.Publish(ctx, resultBytes, subjTokens...)
	return err, sent
}

func (c *Client) PublishResultWithAck(ctx context.Context, msg jetstream.Msg, startedAt time.Time, result interface{}, err error, subjTokens ...string) (bool, error) {
	err, sent := c.PublishResult(ctx, startedAt, result, err, subjTokens...)

	if err == nil && sent {
		err = DoubleAck(ctx, msg)
	}

	return sent, err
}

func (c *Client) PutSysObject(name string, data []byte) (*nats.ObjectInfo, error) {
	return c.SysObjStore.PutBytes(name, data)
}

func (c *Client) buildSubject(subjTokens ...string) string {
	tokens := append([]string{c.accountId, c.interestTopic}, subjTokens...)
	return strings.Join(tokens, ".")
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

func (c *Client) initStream(ctx context.Context) error {
	stream, err := c.JetStream.Stream(ctx, c.streamName)
	if err != nil {
		return err
	}

	c.stream = stream

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
		sourceMsgSubject := SourceEventSubject(c.accountId, c.interestTopic, sequenceId)
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
			FilterSubject: ReplayFilterSubject(c.accountId, c.interestTopic, replaySequenceId),
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

		consumerName := fmt.Sprintf("%s-%s-%s", c.accountId, c.interestTopic, ChannelNotify)
		consumerName = nameReplacer.Replace(consumerName)

		consumer, err := c.JetStream.Consumer(ctx, c.streamName, consumerName)
		if err != nil {
			return err
		}

		c.Consumers[name] = consumer
		return nil
	}
}

// WithLocalRunner initialises a runner with a randomised interest topic and ephemeral consumer
func WithLocalRunner(name string) ClientOpt {
	return func(c *Client) error {
		ctx := context.Background()

		c.interestTopic = fmt.Sprintf("local-%s", uuid.NewString()[:7])

		consumerName := fmt.Sprintf("%s-%s-%s", c.accountId, c.interestTopic, ChannelNotify)
		consumerName = nameReplacer.Replace(consumerName)

		cfg := jetstream.ConsumerConfig{
			Name:          c.interestTopic,
			FilterSubject: NotifyFilterSubject(c.accountId, c.interestTopic),
			DeliverPolicy: jetstream.DeliverAllPolicy,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       time.Minute * 1,
			MaxDeliver:    5,
			ReplayPolicy:  jetstream.ReplayInstantPolicy,
		}
		consumer, err := c.JetStream.CreateOrUpdateConsumer(ctx, c.streamName, cfg)
		if err != nil {
			return err
		}

		c.Consumers[name] = consumer
		return nil
	}
}

// WithStreamName overrides the stream name to be used (which defaults to accountId otherwise)
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

		name := fmt.Sprintf("%s-%s-%s-%s", c.accountId, c.interestTopic, ChannelRequest, appName)
		name = nameReplacer.Replace(name)

		// Create or update the consumer, since these are created dynamically
		consumerCfg := jetstream.ConsumerConfig{
			Name:          name,
			Durable:       name,
			FilterSubject: WorkerRequestFilterSubject(c.accountId, c.interestTopic, appName, "*"),
			AckWait:       1 * time.Minute,
			MaxDeliver:    120, // Two hours of redelivery attempts
		}
		consumer, err := c.JetStream.CreateOrUpdateConsumer(ctx, c.streamName, consumerCfg)
		if err != nil {
			return err
		}

		c.Consumers[appName] = consumer
		return nil
	}
}
