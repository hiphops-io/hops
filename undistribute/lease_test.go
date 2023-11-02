package undistribute

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiphops-io/hops/logs"
)

func TestLease(t *testing.T) {
	// Set up test specific config
	ctx := context.Background()
	stateDir := t.TempDir()
	receivedEventBundle := make(chan map[string][]byte)
	sequenceId := "SEQUENCE_ID"

	testLease, err := NewTestLeaseServer(ctx, t)
	require.NoError(t, err)
	defer testLease.Cleanup()

	// We use consumeMultiple directly as `lease.Consume` runs forever
	consumer, err := ConsumeMulti(func(m jetstream.Msg) {
		_, eventBundle, err := UpFetchState(stateDir, m)
		if err != nil {
			m.Nak()
		}

		receivedEventBundle <- eventBundle
		m.DoubleAck(ctx)
	}, testLease.Consumer)
	require.NoError(t, err)
	defer consumer.Stop()

	// Publish an event
	_, err = testLease.Lease.Publish(
		ctx,
		Notify,
		sequenceId,
		"MSG_ID",
		[]byte("event data"),
	)
	if assert.NoError(t, err) {
		received := <-receivedEventBundle
		assert.NotNil(t, received, "Event bundle should not be nil")
		assert.Equal(t, []byte("event data"), received["MSG_ID"])
	}

	// Now separately test that the hub received the message
	msg, err := testLease.HubStreamMgr.ReadMessage(1)
	if assert.NoError(t, err) {
		assert.Equal(t, []byte("event data"), msg.Data)
	}

	// Test multiple events against the same sequence
	_, err = testLease.Lease.Publish(
		ctx,
		Notify,
		sequenceId,
		"OTHER_MSG_ID",
		[]byte("event data"),
	)
	if assert.NoError(t, err) {
		received := <-receivedEventBundle
		assert.NotNil(t, received, "Event bundle should not be nil")
		assert.Equal(t, []byte("event data"), received["OTHER_MSG_ID"])
	}

	// Test that the new message was published too
	msg, err = testLease.HubStreamMgr.ReadMessage(2)
	if assert.NoError(t, err) {
		assert.Contains(t, msg.Subject, "OTHER_MSG_ID")
	}

	// Test that duplicate events throw an error
	_, err = testLease.Lease.Publish(
		ctx,
		Notify,
		sequenceId,
		"MSG_ID",
		[]byte("Updated event data"),
	)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "maximum messages per subject exceeded")
}

func TestLeaseConsumerCreation(t *testing.T) {
	// Set up test specific config
	ctx := context.Background()
	testLease, err := NewTestLeaseServer(ctx, t)
	require.NoError(t, err)
	defer testLease.Cleanup()

	assert.NotNil(t, testLease.Lease.consumers, "Consumers should not be nil")
	assert.Equal(t, len(testLease.Lease.consumers), 2, "Two consumers should be created")

	// Preserve values for comparison against recreated lease
	leaseId := testLease.Lease.Config().LeaseSubject
	consumerSubjects := getConsumerSubjects(testLease.Lease)

	expectedSubjects := []string{
		fmt.Sprintf(
			"%s.%s.notify.>",
			testLease.StreamName,
			testLease.Lease.Config().LeaseSubject,
		),
		fmt.Sprintf("%s.any.notify.>", testLease.StreamName),
	}
	assert.ElementsMatch(
		t,
		consumerSubjects,
		expectedSubjects,
		"Consumers should have correct filter subjects",
	)

	// Populate the stream with existing data so we can ensure it's still there after creating a new lease
	existingValue := []byte("Existing value")
	existingMsgId := "SEQUENCE_ID.EXISTING_MSG_ID"
	_, err = testLease.Lease.Publish(
		ctx,
		Notify,
		"SEQUENCE_ID",
		"EXISTING_MSG_ID",
		existingValue,
	)
	require.NoError(t, err, "Test setup: Error publishing existing value")

	// Recreate a lease and ensure everything is preserved
	lease2, err := NewLease(ctx, testLease.Lease.Config())
	require.NoError(t, err)
	defer lease2.Close()

	recreatedConsumerSubjects := getConsumerSubjects(lease2)

	assert.Equal(
		t,
		leaseId,
		lease2.Config().LeaseSubject,
		"Recreated lease should have same ID as original",
	)
	assert.ElementsMatch(
		t,
		consumerSubjects,
		recreatedConsumerSubjects,
		"Recreated lease should have unchanged filter subjects",
	)

	// Check that the existing message is still there
	existingSubj := fmt.Sprintf(
		"%s.%s.notify.%s",
		testLease.StreamName,
		testLease.Lease.Config().LeaseSubject,
		existingMsgId,
	)
	msg, err := lease2.stream.GetLastMsgForSubject(ctx, existingSubj)
	require.NoError(t, err)
	assert.Equal(t, existingValue, msg.Data, "Stream should retain existing values")
}

// getConsumerSubjects is a test helper to get all subjects from a lease's consumers
func getConsumerSubjects(lease *Lease) []string {
	consumerSubjects := []string{}
	for _, consumer := range lease.consumers {
		consumerSubjects = append(consumerSubjects, consumer.CachedInfo().Config.FilterSubject)
	}

	return consumerSubjects
}

type TestLeaseServer struct {
	StreamName string
	Lease      *Lease
	Consumer   jetstream.Consumer
	Cleanup    func()
	NatsHub    *server.Server
	// LeafHub      *server.Server
	HubConn      *nats.Conn
	HubStreamMgr *jsm.Stream
}

// createTestLease is a test helper to bootstrap a lease
func NewTestLeaseServer(ctx context.Context, t *testing.T) (TestLeaseServer, error) {
	hub, natsCleanup := createNatsServers(t)

	hubConn, err := nats.Connect(hub.ClientURL())
	if err != nil {
		defer natsCleanup()
		return TestLeaseServer{}, err
	}

	streamMgr, err := createTestStream(ctx, hubConn)
	if err != nil {
		defer natsCleanup()
		return TestLeaseServer{}, err
	}

	lease, streamName, consumer, err := createTestLease(ctx, t, hub.ClientURL())
	if err != nil {
		defer natsCleanup()
		return TestLeaseServer{}, err
	}

	cleanup := func() {
		defer natsCleanup()
		defer lease.js.DeleteStream(ctx, streamName)
		defer lease.Close()
	}

	testLeaseServer := TestLeaseServer{
		StreamName:   streamName,
		Lease:        lease,
		Consumer:     consumer,
		Cleanup:      cleanup,
		NatsHub:      hub,
		HubConn:      hubConn,
		HubStreamMgr: streamMgr,
	}

	return testLeaseServer, nil
}

// createTestLease is a test helper to bootstrap a lease
func createTestLease(ctx context.Context, t *testing.T, natsUrl string) (*Lease, string, jetstream.Consumer, error) {
	leaseDir := t.TempDir()
	streamName := "hops-account"

	// Create a new lease
	leaseConf := LeaseConfig{
		StreamName: streamName,
		RootDir:    leaseDir,
		NatsUrl:    natsUrl,
	}
	lease, err := NewLease(ctx, leaseConf)
	if err != nil {
		return nil, "", nil, err
	}

	testConsumerConf := lease.config.LeaseConsumerConfig()
	testConsumerConf.FilterSubject = fmt.Sprintf("%s.%s.>", lease.config.StreamName, lease.config.LeaseSubject)
	testConsumer, err := lease.stream.CreateOrUpdateConsumer(ctx, testConsumerConf)

	return lease, streamName, testConsumer, err
}

// createNatsServers is a test helper function to set up NATS servers
//
// It will create a local NATS server to act as the hiphops.io hub
func createNatsServers(t *testing.T) (*server.Server, func()) {
	logger := logs.NoOpLogger()
	hubLogger := logs.NewNatsZeroLogger(logger, "nats")

	// Hub server
	hubOpts, err := server.ProcessConfigFile("./testdata/hub-nats.conf")
	require.NoError(t, err, "Test setup: Hub server config should be valid")

	hubStoreDir := t.TempDir()
	hubOpts.StoreDir = hubStoreDir

	hub, err := NewNatsServerFromOpts(hubOpts, false, &hubLogger)
	require.NoError(t, err, "Test setup: Hub server should start")

	cleanup := func() {
		hub.Shutdown()
		hub.WaitForShutdown()
	}

	return hub, cleanup
}

func createTestStream(ctx context.Context, nc *nats.Conn) (*jsm.Stream, error) {
	streamName := "hops-account"

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}

	streamCfg := jetstream.StreamConfig{
		Name:                 streamName,
		Subjects:             []string{"hops-account.>"},
		MaxMsgSize:           1024 * 1024 * 8, // 8MB
		Discard:              jetstream.DiscardNew,
		DiscardNewPerSubject: true,
		MaxMsgsPerSubject:    1,
	}
	_, err = js.CreateStream(ctx, streamCfg)
	if err != nil {
		return nil, err
	}

	// Create the manager for the stream
	mgr, err := jsm.New(nc, jsm.WithTimeout(10*time.Second))
	if err != nil {
		return nil, err
	}

	return mgr.LoadStream(streamName)
}

// 	---
// apiVersion: jetstream.nats.io/v1beta2
// kind: Stream
// metadata:
//   name: hiphops-0395b0b2-0dcd-4dfb-89f8-65a36d32d9f3-notify
//   namespace: nats
// spec:
//   account: HIPHOPS
//   name: HIPHOPS-0395b0b2-0dcd-4dfb-89f8-65a36d32d9f3-notify
//   subjects: ["0395b0b2-0dcd-4dfb-89f8-65a36d32d9f3.*.notify.>"]
//   maxMsgSize: 8388608
//   maxConsumers: -1
//   discard: "new"
//   discardPerSubject: true
//   maxMsgsPerSubject: 1
//   storage: "file"
// Attempt stream creation

// ---
// apiVersion: jetstream.nats.io/v1beta2
// kind: Stream
// metadata:
//   name: hiphops-all-project-requests
//   namespace: nats
// spec:
//   account: HIPHOPS
//   name: HIPHOPS-all-project-requests
//   subjects: ["*.*.request.>"]
//   maxMsgSize: 8388608
//   maxConsumers: -1
//   discard: "new"
//   discardPerSubject: true
//   maxMsgsPerSubject: 1
//   storage: "file"
