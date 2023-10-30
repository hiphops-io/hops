package undistribute

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiphops-io/hops/logs"
)

func TestNewNatsServer(t *testing.T) {
	rootDir := t.TempDir()
	testSubj := fmt.Sprintf("hops-account.notify.%s", uuid.NewString())
	expectedMsg := []byte("Hello!")
	expectedReply := []byte("Reply")
	// Create no-op logger
	logger := logs.NoOpLogger()
	natsLogger := logs.NewNatsZeroLogger(logger)

	ns, _, err := NewNatsServer("./testdata/leaf-nats.conf", rootDir, false, &natsLogger)
	require.NoError(t, err, "Embedded NATS server should start without errors")
	assert.True(t, ns.Running(), "Embedded NATS server should be running")

	natsurl := ns.ClientURL()
	assert.NotEmpty(t, natsurl, "Embedded NATS URL should not be empty")

	nc, err := nats.Connect(natsurl)
	require.NoError(t, err, "Should connect to embedded NATS without errors")
	require.True(t, nc.IsConnected(), "Connection to NATS should be active")

	// Create a subscription for the test subject
	sub, err := nc.SubscribeSync(testSubj)
	require.NoError(t, err, "Subscription should be created successfully")

	// Reply to the test message when it is received
	go func() {
		msg, err := sub.NextMsg(1 * time.Second)
		require.NoError(t, err, "Next message should be received successfully")

		assert.Equal(t, string(expectedMsg), string(msg.Data), "Message should match expected message")
		// Respond back with the same info
		msg.Respond(expectedReply)
	}()

	// Send the test message
	msg, err := nc.Request(testSubj, expectedMsg, 200*time.Millisecond)
	require.NoError(t, err, "Request should receive a reply without error")

	// Verify the reply
	assert.Equal(t, string(expectedReply), string(msg.Data), "Reply should match expected reply")
}
