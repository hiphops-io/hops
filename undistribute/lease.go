package undistribute

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Lease struct {
	config    *LeaseConfig
	nc        *nats.Conn
	js        jetstream.JetStream
	stream    jetstream.Stream
	consumers []jetstream.Consumer
	leaseDir  string
}

// NewLease creates a new undistribute lease, with all associated directories and NATS resources
//
// You may provide a sparse leaseConf, missing values will be given sensible defaults.
// Note: You usually _don't_ want to provide LeaseSubject. This will be idempotently generated
func NewLease(ctx context.Context, leaseConf LeaseConfig) (*Lease, error) {
	leaseConf.MergeLeaseConfig(DefaultLeaseConfig)
	lease := &Lease{
		config: &leaseConf,
	}

	nc, err := nats.Connect(
		leaseConf.NatsUrl,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(5),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, err
	}
	lease.nc = nc

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Drain()
		return nil, err
	}
	lease.js = js

	err = lease.initLeaseId()
	if err != nil {
		nc.Drain()
		return nil, err
	}

	// Ensure the stream exists
	err = lease.initStream(ctx)
	if err != nil {
		return nil, err
	}

	err = lease.initConsumers(ctx)
	if err != nil {
		nc.Drain()
		return nil, err
	}

	return lease, nil
}

// Consume consumes messages from all lease consumers (any and notify)
//
// This will block the calling goroutine until the context is cancelled
// and is mostly aimed at running as a persistent long-lived service
func (l *Lease) Consume(ctx context.Context, callback jetstream.MessageHandler) error {
	consumer, err := ConsumeMulti(callback, l.consumers...)
	if err != nil {
		return err
	}
	defer consumer.Stop()

	// Run until context cancelled
	<-ctx.Done()

	return nil
}

type Channel string

const (
	Notify  Channel = "notify"
	Request Channel = "request"
)

// Publish publishes a message on the lease subject
func (l *Lease) Publish(ctx context.Context, channel Channel, sequenceId string, msgId string, data []byte) (*jetstream.PubAck, error) {
	msgSubject := l.config.LeaseMsgSubject(string(channel), sequenceId, msgId)
	return l.js.Publish(ctx, msgSubject, data)
}

// PublishSource publishes a message on the source subject
func (l *Lease) PublishSource(ctx context.Context, channel Channel, sequenceId string, msgId string, data []byte) (*jetstream.PubAck, error) {
	msgSubject := l.config.SourceMsgSubject(string(channel), sequenceId, msgId)
	return l.js.Publish(ctx, msgSubject, data)
}

func (l *Lease) Dir() string {
	return l.leaseDir
}

func (l *Lease) Close() {
	defer l.nc.Drain()
}

func (l *Lease) Config() LeaseConfig {
	return *l.config
}

func (l *Lease) NatsConnection() *nats.Conn {
	return l.nc
}

// Init functions and helpers //

// initConsumers creates/updates the lease and source consumer for a lease
func (l *Lease) initConsumers(ctx context.Context) error {
	leaseConsConf := l.config.LeaseConsumerConfig()
	leaseConsumer, err := l.stream.CreateOrUpdateConsumer(ctx, leaseConsConf)
	if err != nil {
		return err
	}

	sourceConsConf := l.config.SourceConsumerConfig()
	sourceConsumer, err := l.stream.CreateOrUpdateConsumer(ctx, sourceConsConf)
	if err != nil {
		return err
	}

	l.consumers = []jetstream.Consumer{
		leaseConsumer,
		sourceConsumer,
	}

	return nil
}

// initStream gets the stream for the lease
func (l *Lease) initStream(ctx context.Context) error {
	stream, err := l.js.Stream(ctx, l.config.StreamName)
	if err != nil {
		return err
	}

	l.stream = stream
	return nil
}

// initLeaseId generates a new lease ID in the given directory
//
// A lease is composed of a base lease ID (auto generated and stored on disk) and
// optional subleases. Any number of sublease arguments can be given
// and they will be combined to create the lease ID.
// leaseConf.LeaseSubject, leaseConf.RootDir, leaseConf.Seed
func (l *Lease) initLeaseId() error {
	if l.config.LeaseSubject == "" {
		leaseId, err := createLeaseId(l.config.RootDir, l.config.Seed)

		if err != nil {
			return err
		}
		l.config.LeaseSubject = leaseId
	}

	leaseDir, err := createLeaseDir(l.config.RootDir, l.config.LeaseSubject)
	if err != nil {
		return err
	}

	l.leaseDir = leaseDir
	return nil
}

func createLeaseId(rootLeaseDir string, seeds ...[]byte) (string, error) {
	baseId, err := getOrCreateBaseId(rootLeaseDir)
	if err != nil {
		return "", err
	}
	sublease := bytes.Join(seeds, []byte{})
	leaseId := uuid.NewSHA1(baseId, sublease).String()

	return leaseId, nil
}

func createLeaseDir(rootLeaseDir string, leaseId string) (string, error) {
	leaseDir := path.Join(rootLeaseDir, leaseId)
	err := os.Mkdir(leaseDir, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return "", err
	}

	return leaseDir, nil
}

func getOrCreateBaseId(leaseDir string) (uuid.UUID, error) {
	rootId, err := getBaseId(leaseDir)
	if err == nil {
		return createBaseId(leaseDir)
	}

	return rootId, nil
}

func getBaseId(dir string) (uuid.UUID, error) {
	baseId, err := os.ReadFile(path.Join(dir, "baseid"))
	if err != nil {
		return uuid.Nil, err
	}

	return uuid.ParseBytes(baseId)
}

func createBaseId(dir string) (uuid.UUID, error) {
	baseId := uuid.New()
	baseIdBytes, err := baseId.MarshalBinary()
	if err != nil {
		return uuid.Nil, err
	}

	err = os.WriteFile(path.Join(dir, "baseid"), baseIdBytes, 0644)
	if err != nil {
		return uuid.Nil, err
	}

	return baseId, nil
}
