package undistribute

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type TestMsg struct {
	subject string
	data    []byte
}

func (t TestMsg) Subject() string {
	return t.subject
}

func (t TestMsg) Data() []byte {
	return t.data
}

func TestUpFetchState(t *testing.T) {
	stateDir := t.TempDir()
	eventData := []byte("Some data")

	// Test upfetching against an empty state
	msgId := uuid.NewString()
	msg := TestMsg{
		subject: fmt.Sprintf("some_lease.foo.%s", msgId),
		data:    eventData,
	}
	sequenceId, state, err := UpFetchState(stateDir, msg)
	if assert.NoError(t, err) {
		assert.Equal(t, eventData, state[msgId], "Returned data should match message data")
		assert.Len(t, state, 1, "State should contain one message")
		assert.Equal(t, "foo", sequenceId, "Sequence ID should match")
	}

	// Test upfetching against an existing state
	msgId2 := uuid.NewString()
	msg2 := TestMsg{
		subject: fmt.Sprintf("some_lease.foo.%s", msgId2),
		data:    eventData,
	}
	sequenceId, state, err = UpFetchState(stateDir, msg2)
	if assert.NoError(t, err) {
		assert.Equal(t, eventData, state[msgId2], "Returned data should match message data")
		assert.Len(t, state, 2, "State should contain all messages")
		assert.Equal(t, "foo", sequenceId, "Sequence ID should match")
	}

	// Test multiple existing state entries
	msgId3 := uuid.NewString()
	msg3 := TestMsg{
		subject: fmt.Sprintf("some_lease.foo.%s", msgId3),
		data:    eventData,
	}
	_, state, err = UpFetchState(stateDir, msg3)
	if assert.NoError(t, err) {
		assert.Equal(t, eventData, state[msgId3], "Returned data should match message data")
		assert.Len(t, state, 3, "State should contain all messages")
	}

	// Test writing to an existing ID
	updatedEventData := []byte("Some other data")
	msgUpdate := TestMsg{
		subject: fmt.Sprintf("some_lease.foo.%s", msgId),
		data:    updatedEventData,
	}
	_, state, err = UpFetchState(stateDir, msgUpdate)
	if assert.NoError(t, err) {
		assert.Equal(t, updatedEventData, state[msgId], "Returned data should match message data")
		assert.Len(t, state, 3, "State should contain all messages")
	}

	// Test state isn't mixed across different sequence IDs
	seqTwoMsgId := uuid.NewString()
	seqTwoMsgData := []byte("Hello there")
	seqTwoMsg := TestMsg{
		subject: fmt.Sprintf("some_lease.bar.%s", seqTwoMsgId),
		data:    seqTwoMsgData,
	}
	_, state, err = UpFetchState(stateDir, seqTwoMsg)
	if assert.NoError(t, err) {
		assert.Equal(t, seqTwoMsgData, state[seqTwoMsgId], "Returned data should match message data")
		assert.Len(t, state, 1, "State should contain all messages")
	}
}
