package nats

import (
	"net/url"
	"testing"

	"github.com/hiphops-io/hops/logs"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalNatsConnection(t *testing.T) {
	localNats := setupLocalNatsServer(t)
	defer localNats.Close()

	nc, err := localNats.Connect("")

	if assert.NotNil(t, nc) {
		defer nc.Drain()
	}
	require.NoError(t, err, "Should connect to local NATS without errors")
	require.True(t, nc.IsConnected(), "Connection to local NATS should be active")
}

func TestNewLocalNats(t *testing.T) {
	localNats := setupLocalNatsServer(t)
	defer localNats.Close()

	require.True(t, localNats.NatsServer.Running(), "Local NATS server should be running")
}

func TestLocalNatsConnect(t *testing.T) {
	localNats := setupLocalNatsServer(t)
	defer localNats.Close()

	nc, err := localNats.Connect("")

	if assert.NotNil(t, nc) {
		defer nc.Drain()
	}
	require.NoError(t, err, "Local NATS client should connect without errors")
	assert.True(t, nc.IsConnected(), "Local NATS client connection should be active")
}

func TestLocalNatsAuthUrl(t *testing.T) {
	localNats := setupLocalNatsServer(t)
	defer localNats.Close()

	authUrl, err := localNats.AuthUrl("")
	require.NoError(t, err, "Local nats should return authenticated URL without error")

	nc, err := nats.Connect(authUrl)
	if assert.NotNil(t, nc) {
		defer nc.Drain()
	}
	if assert.NoError(t, err, "Should connect to NATS with auth URL without error") {
		assert.True(t, nc.IsConnected(), "Auth URL should connect to NATS")
	}

	parsedUrl, err := url.Parse(authUrl)
	if assert.NoError(t, err, "Authed URL should be valid and parsable") {
		assert.NotNil(t, parsedUrl.User, "Authed URL should contain user info")
	}
}

func TestLocalNatsClose(t *testing.T) {
	t.Skip("Not implemented: Ensure calling close shuts down the server")
}

// setupLocalNatsServer is a test helper to create a local NATS server with a silent logger
func setupLocalNatsServer(t *testing.T) *LocalServer {
	natsDir := t.TempDir()
	// Create no-op logger
	logger := logs.NoOpLogger()
	natsLogger := logs.NewNatsZeroLogger(logger)

	localNats, err := NewLocalServer("./testdata/hub-nats.conf", natsDir, false, &natsLogger)
	require.NoError(t, err, "Test setup: Embedded NATS server should start without errors")

	return localNats
}
