package nats

import (
	"fmt"
	"regexp"
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
	ChannelWork    = "work"
	DoneMessageId  = "done"
	HopsMessageId  = "hops"
	MetadataKey    = "hops"
	SourceEventId  = "event"
)

var (
	NotifyStreamSubjects  = []string{fmt.Sprintf("%s.>", ChannelNotify)}
	RequestStreamSubjects = []string{fmt.Sprintf("%s.>", ChannelRequest)}
	WorkStreamSubjects    = []string{fmt.Sprintf("%s.>", ChannelWork)}
	nonAlphaNumRegex      = regexp.MustCompile(`[^a-zA-Z0-9\-_]+`)
)

type (
	HopsMsg struct {
		Action           string
		AppName          string
		Channel          string
		ConsumerSequence uint64
		Data             map[string]any
		Done             bool // Message is a pipeline 'done' message
		Event            string
		HandlerName      string
		MessageId        string
		NumPending       uint64
		SequenceId       string
		Source           string
		StreamSequence   uint64
		Subject          string
		Timestamp        time.Time
		msg              jetstream.Msg
	}

	// HopsResultMeta is metadata included in the top level of a result message
	HopsResultMeta struct {
		Error      string    `json:"error,omitempty"`
		FinishedAt time.Time `json:"finished_at"`
		StartedAt  time.Time `json:"started_at"`
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

func Parse(msg jetstream.Msg) (*HopsMsg, error) {
	message := &HopsMsg{msg: msg}

	if err := message.parseTokens(); err != nil {
		return nil, err
	}

	if err := message.parseMetadata(); err != nil {
		return nil, err
	}

	if err := message.parseData(); err != nil {
		return nil, err
	}

	return message, nil
}

func (m *HopsMsg) Msg() jetstream.Msg {
	return m.msg
}

func (m *HopsMsg) ResponseSubject() string {
	// TODO: Will need to handle work response subjects as they have an extra component (flow_name.worker_name)
	// Though, will workers ever have responses?
	tokens := []string{
		ChannelNotify,
		m.SequenceId,
		m.MessageId,
	}

	return strings.Join(tokens, ".")
}

func (m *HopsMsg) parseData() error {
	msgData := map[string]any{}
	if err := json.Unmarshal(m.msg.Data(), &msgData); err != nil {
		return fmt.Errorf("unable to unmarshal: %w", err)
	}

	metadata, ok := msgData[MetadataKey]
	if !ok {
		return fmt.Errorf("missing required metadata object: %s", MetadataKey)
	}

	metadataMap, ok := metadata.(map[string]any)
	if !ok {
		return fmt.Errorf("incorrect metadata object structure")
	}

	source, ok := metadataMap["source"].(string)
	if !ok || source == "" {
		return fmt.Errorf("missing required metadata value 'source'")
	}

	event, ok := metadataMap["event"].(string)
	if !ok || event == "" {
		return fmt.Errorf("missing required metadata value 'event'")
	}

	action, _ := metadataMap["action"].(string)

	m.Action = action
	m.Data = msgData
	m.Event = event
	m.Source = source

	return nil
}

func (m *HopsMsg) parseMetadata() error {
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

// parseTokens parses tokens from a message subject into the Msg struct
//
// Example hops subjects are:
// `notify.sequence_id.event`
// `notify.sequence_id.message_id`
// `request.sequence_id.message_id.app.handler`
func (m *HopsMsg) parseTokens() error {
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

// SanitiseToken is a helper function to ensure source event tokens are suitable
// for consumption by hops/inclusion in hops subjects
func SanitiseToken(token string) string {
	token = nonAlphaNumRegex.ReplaceAllLiteralString(token, "")
	return strings.ToLower(token)
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

func SourceEventSubjectAccount(accountId string, sequenceId string) string {
	tokens := []string{
		ChannelNotify,
		accountId,
		sequenceId,
		"event",
	}

	return strings.Join(tokens, ".")
}

// WorkerRequestFilterSubject returns the filter subject for the worker consumer
//
// Note: This has a name clash with the new concept of worker - which is user-defined functions.
// Worker in this context refers to an integration that handles outbound requests.
// This clash will be resolved prior to v1
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

func WorkSubject(sequenceId string, workerName string) string {
	tokens := []string{
		ChannelWork,
		sequenceId,
		workerName,
	}

	return strings.Join(tokens, ".")
}
