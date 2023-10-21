package dsl

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

func TestCreateSourceEvent(t *testing.T) {
	input := map[string]any{
		"whatever": "trevor",
	}
	expectedHops := map[string]any{
		"source": "hiphops",
		"event":  "command",
		"action": "my_command",
	}

	sourceEventByte, hash, err := CreateSourceEvent(input, "hiphops", "command", "my_command")
	assert.NoError(t, err, "Source event should be created without error")

	var sourceEvent map[string]any

	err = json.Unmarshal(sourceEventByte, &sourceEvent)

	assert.NoError(t, err, "Source event should be valid JSON")
	assert.Equal(t, sourceEvent["hops"], expectedHops)
	assert.Equal(t, sourceEvent["whatever"], "trevor")
	assert.NotEmpty(t, hash, "Hash should not be empty")
}
