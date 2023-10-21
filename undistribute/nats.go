package undistribute

import (
	"context"
	"errors"
	"path"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go/jetstream"
)

// DoubleAck is a convenience wrapper around jetstream's msg.DoubleAck
func DoubleAck(ctx context.Context, msg jetstream.Msg) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return msg.DoubleAck(ctx)
}

// NewNatsServer starts an in-process nats server from a config file
func NewNatsServer(natsConfigPath string, rootDir string, debug bool, logger server.Logger) (*server.Server, *server.Options, error) {
	opts, err := server.ProcessConfigFile(natsConfigPath)
	if err != nil {
		return nil, nil, err
	}

	if opts.StoreDir == "" {
		opts.StoreDir = path.Join(rootDir, "jetstream-data")
	}

	server, err := NewNatsServerFromOpts(opts, debug, logger)

	return server, opts, err
}

func NewNatsServerFromOpts(opts *server.Options, debug bool, logger server.Logger) (*server.Server, error) {
	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, err
	}

	ns.SetLoggerV2(logger, debug, debug, debug)

	go ns.Start()

	if !ns.ReadyForConnections(5 * time.Second) {
		return nil, errors.New("NATS server didn't become ready")
	}

	return ns, nil
}
