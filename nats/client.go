package nats

import (
	"context"
	"fmt"
	"strings"
	"time"

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

func NewClient(ctx context.Context, natsUrl string, accountId string) (*Client, error) {
	hopsNats := &Client{
		accountId: accountId,
	}
	err := hopsNats.initNatsConnection(natsUrl)
	if err != nil {
		return nil, err
	}

	err = hopsNats.initJetStream()
	if err != nil {
		defer hopsNats.Close()
		return nil, err
	}

	err = hopsNats.initConsumer(ctx, accountId)
	if err != nil {
		defer hopsNats.Close()
		return nil, err
	}

	return hopsNats, err
}

func (h *Client) Close() {
	h.NatsConn.Drain()
}

// Consume consumes messages from the HopsNats.Consumer
//
// This will block the calling goroutine until the context is cancelled
// and can be ran as a long-lived service
func (h *Client) Consume(ctx context.Context, callback jetstream.MessageHandler) error {
	consumer, err := h.Consumer.Consume(callback)
	if err != nil {
		return err
	}
	defer consumer.Stop()

	// Run until context cancelled
	<-ctx.Done()

	return nil
}

func (h *Client) Publish(ctx context.Context, data []byte, subjTokens ...string) (*jetstream.PubAck, error) {
	// Prefix subject with accountID
	tokens := append([]string{h.accountId}, subjTokens...)
	subject := strings.Join(tokens, ".")
	return h.JetStream.Publish(ctx, subject, data)
}

func (h *Client) initConsumer(ctx context.Context, accountId string) error {
	consumerName := fmt.Sprintf("%s-%s", accountId, ChannelNotify)
	consumer, err := h.JetStream.Consumer(ctx, accountId, consumerName)
	if err != nil {
		return err
	}

	h.Consumer = consumer
	return nil
}

func (h *Client) initJetStream() error {
	js, err := jetstream.New(h.NatsConn)
	if err != nil {
		return err
	}

	h.JetStream = js
	return nil
}

func (h *Client) initNatsConnection(natsUrl string) error {
	nc, err := nats.Connect(
		natsUrl,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(5),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return err
	}

	h.NatsConn = nc
	return nil
}

// TODO:
// Logic to fetch state of pipeline
// Logic to fetch/store hopses
// Logic to publish messages
