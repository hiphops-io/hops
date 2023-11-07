package nats

import (
	"context"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// DoubleAck is a convenience wrapper around NATS acking with a timeout
func DoubleAck(ctx context.Context, msg jetstream.Msg) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return msg.DoubleAck(ctx)
}
