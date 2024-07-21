package nats

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type receivedMsg struct {
	meta *MsgMeta
	data []byte
}

func TestNewClient(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()

	assert.NotNil(t, client.JetStream, "Client should initialise JetStream")
	assert.NotNil(t, client.NatsConn, "Client should initialise a NATS connection")
	assert.True(t, client.CheckConnection(), "Client should correctly report connection status")
}

func TestClientClose(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()

	require.True(t, client.CheckConnection(), "Client should be connected")

	client.Close()

	assert.False(t, client.CheckConnection(), "Client should not be connected after calling Close()")
}

func TestClientConsume(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client, cleanup := setupClient(t)
	defer cleanup()

	consumer, err := client.RunnerConsumer(ctx)
	require.NoError(t, err, "Consumer must be created without error")

	msgData := []byte("Hello world")
	sequenceID := "SEQ_ID"
	subject := SourceEventSubject(sequenceID)

	msg, err := publishAndConsumeMessage(t, client, consumer, msgData, subject)
	require.NoError(t, err, "Publishing and consuming a message should not return an error")

	assert.Equal(t, msg.meta.Subject, fmt.Sprintf("notify.%s.event", sequenceID))
	assert.Equal(t, msgData, msg.data)
}

func TestClientPublish(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client, cleanup := setupClient(t)
	defer cleanup()

	subject := SourceEventSubject("SEQ_ID")
	_, sent, err := client.Publish(ctx, []byte("a"), subject)
	assert.NoError(t, err, "Message should be published without error")
	assert.True(t, sent, "Message should be sent")

	_, sent, err = client.Publish(ctx, []byte("a"), subject)
	assert.NoError(t, err, "Duplicate message should be ignored without error")
	assert.False(t, sent, "Duplicate message should not be sent")
}

// TODO: Test that a deleted message is still protect for idempotency by using expected sequence
func TestClientWorkerPublish(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client, cleanup := setupClient(t)
	defer cleanup()

	consumer, err := client.WorkerConsumer(ctx, "app", true)
	require.NoError(t, err, "Consumer must be created without error")

	msgData := []byte("Hello world")
	subject := RequestSubject("SEQ_ID", "MSG_ID", "app", "handler")

	_, err = publishAndConsumeMessage(t, client, consumer, msgData, subject)
	require.NoError(t, err)

	// Check pending messages in the consumer is now zero
	info, err := consumer.Info(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, info.NumAckPending, "All messages should have been acked")

	// Now we send another message on the same subject.
	// This should be rejected even though the worker consumer has caused the message to be deleted
	_, sent, err := client.Publish(ctx, []byte("a"), subject)
	assert.NoError(t, err, "Duplicate message should be ignored without error")
	assert.False(t, sent, "Duplicate message should not be sent")
}

// publishAndConsumeMessage is a helper method to send a message and consume it,
// returning the message as it was received by the handler
func publishAndConsumeMessage(t *testing.T, client *Client, consumer jetstream.Consumer, msgData []byte, subject string) (receivedMsg, error) {
	ctx := context.Background()
	msgChan := make(chan receivedMsg)

	go func() {
		client.Consume(ctx, consumer, func(ctx context.Context, msgData []byte, msgMeta *MsgMeta, ackDeadline time.Duration) error {
			// We double ack here as otherwise there's a race condition where we could
			// return the received message back before client.Consume finished everything up.
			msgMeta.msg.DoubleAck(ctx)

			msgChan <- receivedMsg{
				meta: msgMeta,
				data: msgData,
			}

			return nil
		})
	}()

	_, _, err := client.Publish(ctx, msgData, subject)
	require.NoError(t, err, "Message should be published without error")

	select {
	case msg := <-msgChan:
		return msg, nil
	case <-time.After(2 * time.Second):
		t.Error("Message not received within time limit")
	}

	return receivedMsg{}, errors.New("Unknown failure - expected message not consumed")
}

// setupClient is a test helper to create an instance of HopsNats with a local NATS server
func setupClient(t *testing.T) (*Client, func()) {
	server := setupNatsServer(t)

	client, err := NewClient(server.URL(), "")
	require.NoError(t, err, "Test setup: HopsNats should initialise without error")

	cleanup := func() {
		client.Close()
		server.Close()
	}

	return client, cleanup
}
