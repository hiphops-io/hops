package nats

import (
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const HopsMessageId = "hops"
const DoneMessageId = "done"

type (
	// HopsResultMeta is metadata included in the top level of a result message
	HopsResultMeta struct {
		Error      error     `json:"error,omitempty"`
		FinishedAt time.Time `json:"finished_at"`
		StartedAt  time.Time `json:"started_at"`
	}

	MsgMeta struct {
		AccountId        string
		AppName          string
		Channel          string
		ConsumerSequence uint64
		Done             bool
		HandlerName      string
		MessageId        string
		SequenceId       string
		StreamSequence   uint64
		msg              jetstream.Msg
	}

	// ResultMsg is the schema for handler call result messages
	ResultMsg struct {
		Body      string         `json:"body"`
		Completed bool           `json:"completed"`
		Done      bool           `json:"done"`
		Errored   bool           `json:"errored"`
		Hops      HopsResultMeta `json:"hops"`
		JSON      interface{}    `json:"json,omitempty"`
	}
)

func Parse(msg jetstream.Msg) (*MsgMeta, error) {
	message := &MsgMeta{msg: msg}

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

func (m *MsgMeta) ResponseSubject() string {
	tokens := []string{
		m.AccountId,
		ChannelNotify,
		m.SequenceId,
		m.MessageId,
	}

	return strings.Join(tokens, ".")
}

func (m *MsgMeta) SequenceFilter() string {
	tokens := []string{
		m.AccountId,
		ChannelNotify,
		m.SequenceId,
		">",
	}

	return strings.Join(tokens, ".")
}

func (m *MsgMeta) initMetadata() error {
	meta, err := m.msg.Metadata()
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
func (m *MsgMeta) initTokens() error {
	subjectTokens := strings.Split(m.msg.Subject(), ".")
	if len(subjectTokens) < 4 {
		return fmt.Errorf("Invalid message subject (too few tokens): %s", m.msg.Subject())
	}

	m.AccountId = subjectTokens[0]
	m.Channel = subjectTokens[1]
	m.SequenceId = subjectTokens[2]
	m.MessageId = subjectTokens[3]

	if len(subjectTokens) == 5 {
		m.Done = subjectTokens[4] == DoneMessageId
	}

	if m.Channel == ChannelNotify {
		return nil
	}

	switch m.Channel {
	case ChannelNotify:
		return nil
	case ChannelRequest:
		if len(subjectTokens) < 6 {
			return fmt.Errorf("Invalid request message subject (too few tokens): %s", m.msg.Subject())
		}

		m.AppName = subjectTokens[4]
		m.HandlerName = subjectTokens[5]

		return nil
	default:
		return fmt.Errorf("Invalid message subject (unknown channel %s): %s", m.Channel, m.msg.Subject())
	}
}

func NewResultMsg(startedAt time.Time, result interface{}, err error) ResultMsg {
	var resultJson interface{}
	resultStr, ok := result.(string)
	if !ok {
		resultJson = result
	}

	resultMsg := ResultMsg{
		Body:      resultStr,
		Completed: err == nil,
		Done:      true,
		Errored:   err != nil,
		Hops: HopsResultMeta{
			StartedAt:  startedAt,
			FinishedAt: time.Now(),
			Error:      err,
		},
		JSON: resultJson,
	}

	return resultMsg
}

func ReplayFilterSubject(accountId string, sequenceId string) string {
	tokens := []string{
		accountId,
		"*",
		sequenceId,
		">",
	}

	return strings.Join(tokens, ".")
}

func SequenceHopsKeyTokens(sequenceId string) []string {
	return []string{
		ChannelNotify,
		sequenceId,
		HopsMessageId,
	}
}

func SourceEventSubject(accountId string, sequenceId string) string {
	tokens := []string{
		accountId,
		ChannelNotify,
		sequenceId,
		"event",
	}
	return strings.Join(tokens, ".")
}

func WorkerRequestSubject(accountId string, appName string, handler string) string {
	tokens := []string{
		accountId,
		ChannelRequest,
		"*",
		"*",
		appName,
		handler,
	}

	return strings.Join(tokens, ".")
}
