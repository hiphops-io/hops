package nats

import (
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const AllEventId = ">"
const HopsMessageId = "hops"
const DoneMessageId = "done"
const SourceEventId = "event"

type (
	// HopsResultMeta is metadata included in the top level of a result message
	HopsResultMeta struct {
		Error      string    `json:"error,omitempty"`
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
		InterestTopic    string
		MessageId        string
		SequenceId       string
		StreamSequence   uint64
		Timestamp        time.Time
		msg              jetstream.Msg
	}

	// ResultMsg is the schema for handler call result messages
	ResultMsg struct {
		Body       string            `json:"body"`
		Completed  bool              `json:"completed"`
		Done       bool              `json:"done"`
		Errored    bool              `json:"errored"`
		Headers    map[string]string `json:"headers,omitempty"`
		Hops       HopsResultMeta    `json:"hops"`
		JSON       interface{}       `json:"json,omitempty"`
		StatusCode int               `json:"status_code,omitempty"`
		URL        string            `json:"url,omitempty"`
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

func (m *MsgMeta) Msg() jetstream.Msg {
	return m.msg
}

func (m *MsgMeta) ResponseSubject() string {
	tokens := []string{
		m.AccountId,
		m.InterestTopic,
		ChannelNotify,
		m.SequenceId,
		m.MessageId,
	}

	return strings.Join(tokens, ".")
}

func (m *MsgMeta) SequenceFilter() string {
	tokens := []string{
		m.AccountId,
		m.InterestTopic,
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
	m.Timestamp = meta.Timestamp

	return nil
}

// initTokens parses tokens from a message subject into the Msg struct
//
// Example hops subjects are:
// `account_id.interest_topic.notify.sequence_id.event`
// `account_id.interest_topic.notify.sequence_id.hops`
// `account_id.interest_topic.notify.sequence_id.message_id`
// `account_id.interest_topic.request.sequence_id.message_id.app.handler`
func (m *MsgMeta) initTokens() error {
	subjectTokens := strings.Split(m.msg.Subject(), ".")
	if len(subjectTokens) < 5 {
		return fmt.Errorf("Invalid message subject (too few tokens): %s", m.msg.Subject())
	}

	m.AccountId = subjectTokens[0]
	m.InterestTopic = subjectTokens[1]
	m.Channel = subjectTokens[2]
	m.SequenceId = subjectTokens[3]
	m.MessageId = subjectTokens[4]

	if len(subjectTokens) == 6 {
		m.Done = subjectTokens[5] == DoneMessageId
	}

	switch m.Channel {
	case ChannelNotify:
		return nil
	case ChannelRequest:
		if len(subjectTokens) < 7 {
			return fmt.Errorf("Invalid request message subject (too few tokens): %s", m.msg.Subject())
		}

		m.AppName = subjectTokens[5]
		m.HandlerName = subjectTokens[6]

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

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	resultMsg := ResultMsg{
		Body:      resultStr,
		Completed: err == nil,
		Done:      true,
		Errored:   err != nil,
		Hops: HopsResultMeta{
			StartedAt:  startedAt,
			FinishedAt: time.Now(),
			Error:      errMsg,
		},
		JSON: resultJson,
	}

	return resultMsg
}

// EventLogSubject returns the subject used to get events for display to the
// user in the UI.
//
// accountId: The account id to filter on
// eventFilter: either AllEventId or SourceEventId
func EventLogSubject(accountId string, interestTopic string, eventFilter string) string {
	tokens := []string{
		accountId,
		interestTopic,
		"*",
		"*",
		eventFilter,
	}

	return strings.Join(tokens, ".")
}

// NotifyFilterSubject returns the filter subject to get notify messages for the account
func NotifyFilterSubject(accountId string, interestTopic string) string {
	tokens := []string{
		accountId,
		interestTopic,
		ChannelNotify,
		">",
	}

	return strings.Join(tokens, ".")
}

func ReplayFilterSubject(accountId string, interestTopic string, sequenceId string) string {
	tokens := []string{
		accountId,
		interestTopic,
		"*",
		sequenceId,
		">",
	}

	return strings.Join(tokens, ".")
}

// RequestFilterSubject returns the filter subject to get request messages for the account
func RequestFilterSubject(accountId string, interestTopic string) string {
	tokens := []string{
		accountId,
		interestTopic,
		ChannelRequest,
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

func SourceEventSubject(accountId string, interestTopic string, sequenceId string) string {
	tokens := []string{
		accountId,
		interestTopic,
		ChannelNotify,
		sequenceId,
		"event",
	}

	return strings.Join(tokens, ".")
}

// WorkerRequestFilterSubject returns the filter subject for the worker consumer
func WorkerRequestFilterSubject(accountId string, interestTopic string, appName string, handler string) string {
	tokens := []string{
		accountId,
		interestTopic,
		ChannelRequest,
		"*",
		"*",
		appName,
		handler,
	}

	return strings.Join(tokens, ".")
}
