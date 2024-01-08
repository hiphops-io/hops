package nats

import (
	"context"
	"testing"

	"github.com/hiphops-io/hops/logs"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	ctx := context.Background()
	hopsNats, cleanup := setupClient(ctx, t)
	defer cleanup()

	if assert.NotNil(t, hopsNats) {
		defer hopsNats.Close()
	}

	if assert.NotNil(t, hopsNats.NatsConn) {
		assert.True(t, hopsNats.NatsConn.IsConnected(), "HopsNats should be connected to NATS server")
	}

	assert.NotNil(t, hopsNats.JetStream, "HopsNats should initialise JetStream")
	assert.NotNil(t, hopsNats.Consumers[DefaultConsumerName], "HopsNats should initialise the Consumer")
}

func TestClientConsume(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	hopsNats, cleanup := setupClient(ctx, t)
	defer cleanup()

	type testMsg struct {
		subject string
		data    []byte
	}

	receivedChan := make(chan testMsg)

	go func() {
		hopsNats.Consume(ctx, DefaultConsumerName, func(m jetstream.Msg) {
			m.DoubleAck(ctx) // Ack before logging to avoid race condition in tests
			receivedChan <- testMsg{
				subject: m.Subject(),
				data:    m.Data(),
			}
		})
	}()

	_, _, err := hopsNats.Publish(ctx, []byte("Hello world"), ChannelNotify, "SEQ_ID", "MSG_ID")
	if assert.NoError(t, err, "Message should be published without errror") {
		receivedMsg := <-receivedChan
		assert.Contains(t, receivedMsg.subject, "SEQ_ID.MSG_ID")
		assert.Equal(t, []byte("Hello world"), receivedMsg.data)
	}
}

type testSequenceHandler struct {
	receivedChan chan MessageBundle
}

func (t *testSequenceHandler) SequenceCallback(ctx context.Context, sequenceId string, msgBundle MessageBundle) error {
	t.receivedChan <- msgBundle
	return nil
}

func TestClientConsumeSequences(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	hopsNats, cleanup := setupClient(ctx, t)
	defer cleanup()

	receivedChan := make(chan MessageBundle)
	expectedBundleOne := MessageBundle{
		"event": []byte("One"),
	}
	expectedBundleTwo := MessageBundle{
		"event":     []byte("One"),
		"event-two": []byte("Two"),
	}
	expectedBundleThree := MessageBundle{
		"event":       []byte("One"),
		"event-two":   []byte("Two"),
		"event-three": []byte("Three"),
	}

	sqncHandler := &testSequenceHandler{receivedChan: receivedChan}

	go func() {
		hopsNats.ConsumeSequences(ctx, DefaultConsumerName, sqncHandler)
	}()

	_, _, err := hopsNats.Publish(ctx, []byte("One"), ChannelNotify, "SEQ_ID", "event")
	if assert.NoError(t, err, "Message should be published without error") {
		receivedMsgBundle := <-receivedChan
		assert.Equal(t, receivedMsgBundle, expectedBundleOne)
	}

	_, _, err = hopsNats.Publish(ctx, []byte("Two"), ChannelNotify, "SEQ_ID", "event-two")
	if assert.NoError(t, err, "Second message in sequence should be published without error") {
		receivedMsgBundle := <-receivedChan
		assert.Equal(t, receivedMsgBundle, expectedBundleTwo)
	}

	_, _, err = hopsNats.Publish(ctx, []byte("Three"), ChannelNotify, "SEQ_ID", "event-three")
	if assert.NoError(t, err, "Third message in sequence should be published without error") {
		receivedMsgBundle := <-receivedChan
		assert.Equal(t, receivedMsgBundle, expectedBundleThree)
	}
}

// setupClient is a test helper to create an instance of HopsNats with a local NATS server
func setupClient(ctx context.Context, t *testing.T) (*Client, func()) {
	localNats := setupLocalNatsServer(t)

	logger := logs.NoOpLogger()
	natsLogger := logs.NewNatsZeroLogger(logger)

	authUrl, err := localNats.AuthUrl("")
	require.NoError(t, err, "Test setup: Should have valid auth URL for NATS")

	user, err := localNats.User("")
	require.NoError(t, err, "Test setup: Should have valid NATS user")

	hopsNats, err := NewClient(authUrl, user.Account.Name, &natsLogger)
	require.NoError(t, err, "Test setup: HopsNats should initialise without error")

	cleanup := func() {
		hopsNats.Close()
		localNats.Close()
	}

	return hopsNats, cleanup
}
