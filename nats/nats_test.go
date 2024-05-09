package nats

import (
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiphops-io/hops/logs"
)

func TestNewNatsServer(t *testing.T) {
	natsServer := setupNatsServer(t)
	defer natsServer.Close()

	require.True(t, natsServer.Server.Running(), "NATS server should be running")
}

func TestNatsServerConnect(t *testing.T) {
	natsServer := setupNatsServer(t)
	defer natsServer.Close()

	nc, err := natsServer.Connect()

	if assert.NotNil(t, nc) {
		defer nc.Drain()
	}
	require.NoError(t, err, "Local NATS client should connect without errors")
	assert.True(t, nc.IsConnected(), "Local NATS client connection should be active")
}

func TestNatsServerClose(t *testing.T) {
	t.Skip("Not implemented: Ensure calling close shuts down the server")
}

// setupNatsServer is a test helper to create a local NATS server with a silent logger
func setupNatsServer(t *testing.T) *NatsServer {
	// Create no-op logger
	logger := logs.NoOpLogger()
	natsLogger := logs.NewNatsZeroLogger(logger)

	natsServer, err := NewNatsServer("./testdata/embedded-nats.conf", false, &natsLogger, func(opts *server.Options) {
		opts.StoreDir = t.TempDir()
	})
	require.NoError(t, err, "Test setup: Embedded NATS server should start without errors")

	return natsServer
}
