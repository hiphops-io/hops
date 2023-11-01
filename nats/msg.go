package nats

import (
	"fmt"
	"strings"

	"github.com/nats-io/nats.go/jetstream"
)

type Msg struct {
	Msg              jetstream.Msg
	AccountId        string
	AppName          string
	Channel          string
	ConsumerSequence uint64
	HandlerName      string
	MessageId        string
	SequenceId       string
	StreamSequence   uint64
}

func Parse(msg jetstream.Msg) (*Msg, error) {
	message := &Msg{Msg: msg}

	err := message.initTokens()
	if err != nil {
		return nil, err
	}

	err = message.initMetadata()
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (m *Msg) SequenceFilter() string {
	tokens := []string{
		m.AccountId,
		ChannelNotify,
		m.SequenceId,
		">",
	}

	return strings.Join(tokens, ".")
}

func (m *Msg) initMetadata() error {
	meta, err := m.Msg.Metadata()
	if err != nil {
		return err
	}

	m.StreamSequence = meta.Sequence.Stream
	m.ConsumerSequence = meta.Sequence.Consumer

	return nil
}

// initTokens parses tokens from a message subject into the Msg struct
//
// Example hops subjects are:
// `account_id.notify.sequence_id.event`
// `account_id.notify.sequence_id.hops`
// `account_id.notify.sequence_id.message_id`
// `account_id.request.sequence_id.message_id.app.handler`
func (m *Msg) initTokens() error {
	subjectTokens := strings.Split(m.Msg.Subject(), ".")
	if len(subjectTokens) < 4 {
		return fmt.Errorf("Invalid message subject (too few tokens): %s", m.Msg.Subject())
	}

	m.AccountId = subjectTokens[0]
	m.Channel = subjectTokens[1]
	m.SequenceId = subjectTokens[2]
	m.MessageId = subjectTokens[3]

	if m.Channel == ChannelNotify {
		return nil
	}

	switch m.Channel {
	case ChannelNotify:
		return nil
	case ChannelRequest:
		if len(subjectTokens) < 6 {
			return fmt.Errorf("Invalid request message subject (too few tokens): %s", m.Msg.Subject())
		}

		m.AppName = subjectTokens[4]
		m.HandlerName = subjectTokens[5]

		return nil
	default:
		return fmt.Errorf("Invalid message subject (unknown channel %s): %s", m.Channel, m.Msg.Subject())
	}
}
