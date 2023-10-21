// This file is a WIP - The tests in this file don't current test anything of use (TODO).
package workflow

import (
	"context"
	"os"
	"testing"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"

	"github.com/hiphops-io/hops/dsl"
	undist "github.com/hiphops-io/hops/undistribute"
)

type LeaseStub struct {
	calledWith []map[string]string
}

func (l *LeaseStub) Publish(ctx context.Context, channel undist.Channel, sequenceId string, msgId string, data []byte) (*jetstream.PubAck, error) {
	// TODO: Append sequenceID and msgId to l.calledWith
	return nil, nil
}

func TestTaskDispatch(t *testing.T) {
	ctx := context.Background()
	logger := initTestLogger()
	lease := &LeaseStub{}

	hops, _, err := dsl.ReadHopsFiles("./testdata/simple.hops")
	require.NoError(t, err)

	runner, err := NewRunner(lease, hops, logger)
	require.NoError(t, err)

	eventBundle, err := initTestEventBundle()
	require.NoError(t, err)

	err = runner.Run(ctx, "1", eventBundle)
	require.NoError(t, err)

	err = runner.Run(ctx, "1", eventBundle)
	require.NoError(t, err)

	t.Skip("No actual tests implemented yet")
}

func initTestEventBundle() (map[string][]byte, error) {
	eventFile := "./testdata/source_testevent.json"

	eventData, err := os.ReadFile(eventFile)
	if err != nil {
		return nil, err
	}

	eventBundle := map[string][]byte{
		"event": eventData,
	}

	return eventBundle, nil
}

func initTestLogger() zerolog.Logger {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	return log.Logger
}
