package nats

import (
	"context"
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
)

type Client struct {
	NatsConn  *nats.Conn
	JetStream jetstream.JetStream
	Consumer  jetstream.Consumer
	accountId string
}

// TODO: NewClient/NewReplayClient/NewWorkerClient could become a single function
// with the use of variadic opt functions that init the appropriate consumer(s)
// This would reduce duplication.

// NewClient returns a new hiphops NATS client with an account prefixed consumer on the `notify` subject
func NewClient(ctx context.Context, natsUrl string, accountId string) (*Client, error) {
	natsClient := &Client{
		accountId: accountId,
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

	err = natsClient.initConsumer(ctx, accountId)
	if err != nil {
		defer natsClient.Close()
		return nil, err
	}

	return natsClient, err
}

// NewReplayClient returns a new hiphops NATS client with an ephemeral consumer containing a replayed source event
func NewReplayClient(ctx context.Context, natsUrl string, accountId string, sequenceId string) (*Client, error) {
	natsClient := &Client{
		accountId: accountId,
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

	err = natsClient.initReplayConsumer(ctx, accountId, sequenceId)
	if err != nil {
		defer natsClient.Close()
		return nil, err
	}

	return natsClient, err
}

// NewWorkerClient returns a new hiphops NATS client with a consumer on the account.request subject
// for each app worker
//
// appName is the name of the app the worker is handling messages for,
// e.g. our github app's worker would use appName='github'
func NewWorkerClient(ctx context.Context, natsUrl string, accountId string, appName string) (*Client, error) {
	natsClient := &Client{
		accountId: accountId,
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

	err = natsClient.initWorkerConsumer(ctx, accountId, appName)
	if err != nil {
		defer natsClient.Close()
		return nil, err
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

// Consume consumes messages from the HopsNats.Consumer
//
// This will block the calling goroutine until the context is cancelled
// and can be ran as a long-lived service
func (c *Client) Consume(ctx context.Context, callback jetstream.MessageHandler) error {
	consumer, err := c.Consumer.Consume(callback)
	if err != nil {
		return err
	}
	defer consumer.Stop()

	// Run until context cancelled
	<-ctx.Done()

	return nil
}

// MessageBundle is a map of messageIDs and the data that message contained
//
// MessageBundle is designed to be passed to a runner to ensure it has the aggregate state
// of a hiphops sequence of messages.
type MessageBundle map[string][]byte

// SequenceHandler is a function that receives the sequenceId and message bundle for a sequence of messages
// type SequenceHandler func(context.Context, string, MessageBundle) error
type SequenceHandler interface {
	SequenceCallback(context.Context, string, MessageBundle) error
}

// ConsumeSequences is a wrapper around consume that presents the aggregate state of a sequence to the callback
// instead of individual messages.
func (c *Client) ConsumeSequences(ctx context.Context, handler SequenceHandler) error {
	wrappedCB := func(msg jetstream.Msg) {
		hopsMsg, err := Parse(msg)
		if err != nil {
			// If parsing is failing, there's no point retrying the message
			msg.Term()
		}

		msgBundle, err := c.FetchMessageBundle(ctx, hopsMsg)
		if err != nil {
			msg.NakWithDelay(3 * time.Second)
		}

		handler.SequenceCallback(ctx, hopsMsg.SequenceId, msgBundle)
	}

	return c.Consume(ctx, wrappedCB)
}

// FetchMessageBundle pulls all historic messages for a sequenceId from the stream, converting them to a message bundle
//
// The returned message bundle will contain all previous messages in addition to the newly received message
func (c *Client) FetchMessageBundle(ctx context.Context, newMsg *MsgFmt) (MessageBundle, error) {
	filter := newMsg.SequenceFilter()

	// TODO: Create a deadline for the context

	consumerConf := jetstream.OrderedConsumerConfig{
		FilterSubjects: []string{filter},
		DeliverPolicy:  jetstream.DeliverAllPolicy,
	}
	cons, err := c.JetStream.OrderedConsumer(ctx, "hops-account", consumerConf)
	if err != nil {
		return nil, fmt.Errorf("Unable to create ordered consumer: %w", err)
	}

	msgBundle := MessageBundle{}

	for {
		// Get the next message in the sequence
		m, err := cons.Next()
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

func (c *Client) Publish(ctx context.Context, data []byte, subjTokens ...string) (*jetstream.PubAck, error) {
	// Prefix subject with accountID
	tokens := append([]string{c.accountId}, subjTokens...)
	subject := strings.Join(tokens, ".")
	return c.JetStream.Publish(ctx, subject, data)
}

func (c *Client) initConsumer(ctx context.Context, accountId string) error {
	consumerName := fmt.Sprintf("%s-%s", accountId, ChannelNotify)
	consumer, err := c.JetStream.Consumer(ctx, accountId, consumerName)
	if err != nil {
		return err
	}

	c.Consumer = consumer
	return nil
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

func (c *Client) initReplayConsumer(ctx context.Context, accountId string, sequenceId string) error {
	// Get the source message from the stream
	stream, err := c.JetStream.Stream(ctx, accountId)
	if err != nil {
		return err
	}

	// Get the source message to be replayed from the stream
	sourceMsgSubject := SourceEventSubject(accountId, sequenceId)
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
		FilterSubject: ReplayFilterSubject(accountId, replaySequenceId),
		DeliverPolicy: jetstream.DeliverAllPolicy,
	}
	consumer, err := c.JetStream.CreateConsumer(ctx, accountId, consumerCfg)
	if err != nil {
		return err
	}

	// Publish the source message with replayed sequence ID so it's picked up by
	// ephemeral consumer
	c.Publish(ctx, rawMsg.Data, ChannelNotify, replaySequenceId, "event")

	// Set the consumer on the client
	c.Consumer = consumer
	return nil
}

func (c *Client) initWorkerConsumer(ctx context.Context, accountId string, appName string) error {
	name := fmt.Sprintf("%s-%s-%s", accountId, ChannelRequest, appName)
	// Create or update the consumer, since these are created dynamically
	consumerCfg := jetstream.ConsumerConfig{
		Name:          name,
		Durable:       name,
		FilterSubject: WorkerRequestSubject(accountId, appName, "*"),
	}
	consumer, err := c.JetStream.CreateOrUpdateConsumer(ctx, accountId, consumerCfg)
	if err != nil {
		return err
	}

	c.Consumer = consumer
	return nil
}
