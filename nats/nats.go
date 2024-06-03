// Package nats provides the Hiphops specific NATS implementation and utilities
//
// This package contains:
// - An embeddable NATS server with streams configured as required for Hiphops to operate
// - Hiphops specific client, handling various consumers and idempotency checks
// - Schemas and utility classes for Hiphops' expected message and subject formats
package nats

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type (
	NatsServer struct {
		Server  *server.Server
		Options *server.Options
	}

	ServerOpt func(*server.Options)
)

// NewNatsServer starts an in-process nats server from a config file
//
// This will create the NATS server and ensure the streams required for hops to
// function exist.
// NatsServer.Close() should be called when finished with the server
func NewNatsServer(configPath string, debug bool, logger server.Logger, serverOpts ...ServerOpt) (*NatsServer, error) {
	ctx := context.Background()
	opts, err := server.ProcessConfigFile(configPath)
	if err != nil {
		return nil, err
	}

	for _, opt := range serverOpts {
		opt(opts)
	}

	n := &NatsServer{
		Options: opts,
	}

	if err := n.initServer(debug, logger); err != nil {
		return nil, err
	}

	if err = n.initStreams(ctx); err != nil {
		return nil, err
	}

	return n, nil
}

// Close shuts down the nats server, waiting until shutdown is complete
func (n *NatsServer) Close() {
	n.Server.Shutdown()
	n.Server.WaitForShutdown()
}

// Connect establishes a client connection with the nats server
func (n *NatsServer) Connect() (*nats.Conn, error) {
	url := n.Server.ClientURL()

	return nats.Connect(url)
}

func (n *NatsServer) URL() string {
	return n.Server.ClientURL()
}

func (n *NatsServer) initServer(debug bool, logger server.Logger) error {
	server, err := server.NewServer(n.Options)
	if err != nil {
		return err
	}

	server.SetLoggerV2(logger, debug, debug, debug)

	go server.Start()

	if !server.ReadyForConnections(30 * time.Second) {
		return errors.New("NATS server didn't become ready within 30 seconds")
	}

	n.Server = server
	return nil
}

func (n *NatsServer) initStreams(ctx context.Context) error {
	conn, err := n.Connect()
	if err != nil {
		return err
	}

	js, err := jetstream.New(conn)
	if err != nil {
		return err
	}

	// TODO: maxGB for all streams needs to be configurable
	if _, err := UpsertNotifyStream(ctx, js, 10); err != nil {
		return err
	}

	if _, err := UpsertRequestStream(ctx, js, 5); err != nil {
		return err
	}

	if _, err := UpsertWorkStream(ctx, js, 5); err != nil {
		return err
	}

	if _, err := UpsertWorkConsumer(ctx, js); err != nil {
		return err
	}

	return nil
}

// UpsertNotifyStream creates the stream for 'notify' (inbound) messages to hops
func UpsertNotifyStream(ctx context.Context, js jetstream.JetStream, maxGB float64) (jetstream.Stream, error) {
	maxBytes := int64(math.Floor(1024 * 1024 * 1024 * maxGB))

	cfg := jetstream.StreamConfig{
		Name:              ChannelNotify,
		Subjects:          NotifyStreamSubjects,
		Discard:           jetstream.DiscardOld,
		Retention:         jetstream.LimitsPolicy,
		MaxBytes:          maxBytes,
		MaxMsgsPerSubject: 1,
	}

	return js.CreateOrUpdateStream(ctx, cfg)
}

// UpsertRequestStream creates the stream for 'request' (outbound) messages from hops
func UpsertRequestStream(ctx context.Context, js jetstream.JetStream, maxGB float64) (jetstream.Stream, error) {
	maxBytes := int64(math.Floor(1024 * 1024 * 1024 * maxGB))

	cfg := jetstream.StreamConfig{
		Name:              ChannelRequest,
		Subjects:          RequestStreamSubjects,
		Discard:           jetstream.DiscardOld,
		Retention:         jetstream.LimitsPolicy,
		MaxAge:            time.Hour * 24 * 3,
		MaxBytes:          maxBytes,
		MaxMsgsPerSubject: 1,
	}

	return js.CreateOrUpdateStream(ctx, cfg)
}

// UpsertWorkStream creates the stream for 'work' messages from hops/to user workers
func UpsertWorkStream(ctx context.Context, js jetstream.JetStream, maxGB float64) (jetstream.Stream, error) {
	maxBytes := int64(math.Floor(1024 * 1024 * 1024 * maxGB))

	cfg := jetstream.StreamConfig{
		Name:              ChannelWork,
		Subjects:          WorkStreamSubjects,
		Discard:           jetstream.DiscardOld,
		Retention:         jetstream.LimitsPolicy,
		MaxAge:            time.Hour * 24 * 3,
		MaxBytes:          maxBytes,
		MaxMsgsPerSubject: 1,
	}

	return js.CreateOrUpdateStream(ctx, cfg)
}

// UpsertWorkConsumer creates the consumer for 'work' messages used by user-backend
func UpsertWorkConsumer(ctx context.Context, js jetstream.JetStream) (jetstream.Consumer, error) {
	cfg := jetstream.ConsumerConfig{
		Name:          ChannelWork,
		Durable:       ChannelWork,
		DeliverPolicy: jetstream.DeliverNewPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       time.Minute * 1,
		MaxDeliver:    5,
	}

	return js.CreateOrUpdateConsumer(ctx, ChannelWork, cfg)
}

func WithDataDirOpt(dataDir string) ServerOpt {
	return func(opts *server.Options) {
		if dataDir == "" {
			return
		}

		opts.StoreDir = dataDir
	}
}
