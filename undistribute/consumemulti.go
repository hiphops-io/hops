package undistribute

import "github.com/nats-io/nats.go/jetstream"

// MultiConsumeContext wraps multiple consumer contexts such that they can be stopped together
type MultiConsumeContext struct {
	consumeContexts []jetstream.ConsumeContext
}

func (m *MultiConsumeContext) Stop() {
	for _, conCtx := range m.consumeContexts {
		defer conCtx.Stop()
	}
}

// ConsumeMulti consumes multiple consumers with a single callback, returning a MultiConsumeContext
//
// Use MultiConsumeContext.Stop() to stop all consumers after use.
// Failure to call Stop() will result in leaky goroutines.
func ConsumeMulti(callback jetstream.MessageHandler, consumers ...jetstream.Consumer) (*MultiConsumeContext, error) {
	consumeContexts := make([]jetstream.ConsumeContext, len(consumers))

	for i, consumer := range consumers {
		i, consumer := i, consumer

		cns, err := consumer.Consume(callback)
		if err != nil {
			return nil, err
		}

		consumeContexts[i] = cns
	}

	multiCtx := &MultiConsumeContext{
		consumeContexts: consumeContexts,
	}

	return multiCtx, nil
}
