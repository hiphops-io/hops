package nats

import (
	"fmt"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	AllEventId     = ">"
	ChannelNotify  = "notify"
	ChannelRequest = "request"
	DoneMessageId  = "done"
	HopsMessageId  = "hops"
	SourceEventId  = "event"
)

var (
	NotifyStreamSubjects  = []string{fmt.Sprintf("%s.>", ChannelNotify)}
	RequestStreamSubjects = []string{fmt.Sprintf("%s.>", ChannelRequest)}
)

type (
	// HopsResultMeta is metadata included in the top level of a result message
	HopsResultMeta struct {
		Error      string    `json:"error,omitempty"`
		FinishedAt time.Time `json:"finished_at"`
		StartedAt  time.Time `json:"started_at"`
	}

	MsgMeta struct {
		AppName          string
		Channel          string
		ConsumerSequence uint64
		Done             bool // Message is a pipeline 'done' message
		HandlerName      string
		MessageId        string
		NumPending       uint64
		SequenceId       string
		StreamSequence   uint64
		Subject          string
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

	SourceMeta struct {
		Source string `json:"source"`
		Event  string `json:"event"`
		Action string `json:"action"`
		Unique string `json:"unique,omitempty"`
	}
)

func CreateSourceEvent(rawEvent map[string]any, source string, event string, action string, unique string) ([]byte, string, error) {
	rawEvent["hops"] = SourceMeta{
		Source: source,
		Event:  event,
		Action: action,
		// unique is used when we want identical input to be regarded as a different message.
		// Any random string will do the job of changing the hash result.
		Unique: unique,
	}

	sourceBytes, err := json.Marshal(rawEvent)
	if err != nil {
		return nil, "", err
	}

	// We don't really care about the UUID namespace, so we just use an existing one
	sourceUUID := uuid.NewSHA1(uuid.NameSpaceDNS, sourceBytes)
	hash := sourceUUID.String()

	return sourceBytes, hash, nil
}

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
		ChannelNotify,
		m.SequenceId,
		m.MessageId,
	}

	return strings.Join(tokens, ".")
}

func (m *MsgMeta) initMetadata() error {
	meta, err := m.msg.Metadata()
	if err != nil {
		return err
	}

	m.StreamSequence = meta.Sequence.Stream
	m.Subject = m.msg.Subject()
	m.ConsumerSequence = meta.Sequence.Consumer
	m.Timestamp = meta.Timestamp
	m.NumPending = meta.NumPending

	return nil
}

// initTokens parses tokens from a message subject into the Msg struct
//
// Example hops subjects are:
// `notify.sequence_id.event`
// `notify.sequence_id.message_id`
// `request.sequence_id.message_id.app.handler`
func (m *MsgMeta) initTokens() error {
	subjectTokens := strings.Split(m.msg.Subject(), ".")
	if len(subjectTokens) < 3 {
		return fmt.Errorf("Invalid message subject (too few tokens): %s", m.msg.Subject())
	}

	m.Channel = subjectTokens[0]
	m.SequenceId = subjectTokens[1]
	m.MessageId = subjectTokens[2]

	// TODO: Check if we still need the concept of a 'done' message
	if len(subjectTokens) == 4 {
		m.Done = subjectTokens[3] == DoneMessageId
	}

	switch m.Channel {
	case ChannelNotify:
		return nil
	case ChannelRequest:
		if len(subjectTokens) < 5 {
			return fmt.Errorf("Invalid request message subject (too few tokens): %s", m.msg.Subject())
		}

		m.AppName = subjectTokens[3]
		m.HandlerName = subjectTokens[4]

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

// NotifyFilterSubject returns the filter subject to get notify messages
func NotifyFilterSubject() string {
	tokens := []string{
		ChannelNotify,
		">",
	}

	return strings.Join(tokens, ".")
}

func ReplayFilterSubject(sequenceId string) string {
	tokens := []string{
		ChannelNotify,
		sequenceId,
		">",
	}

	return strings.Join(tokens, ".")
}

// RequestFilterSubject returns the filter subject to get request messages
func RequestFilterSubject() string {
	tokens := []string{
		ChannelRequest,
		">",
	}

	return strings.Join(tokens, ".")
}

func RequestSubject(sequenceId string, messageId string, appName string, handler string) string {
	tokens := []string{
		ChannelRequest,
		messageId,
		sequenceId,
		appName,
		handler,
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

func SourceEventSubject(sequenceId string) string {
	tokens := []string{
		ChannelNotify,
		sequenceId,
		"event",
	}

	return strings.Join(tokens, ".")
}

// WorkerRequestFilterSubject returns the filter subject for the worker consumer
func WorkerRequestFilterSubject(appName string, handler string) string {
	tokens := []string{
		ChannelRequest,
		"*",
		"*",
		appName,
		handler,
	}

	return strings.Join(tokens, ".")
}
