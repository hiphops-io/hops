package markdown

import (
	"testing"

	"github.com/hiphops-io/hops/nats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func TestMatchFlows(t *testing.T) {
	type testCase struct {
		name        string
		source      map[string][]byte
		event       *nats.HopsMsg
		expectedIDs []string
		expectError bool
	}

	tests := []testCase{
		{
			name: "Absolute event without conditional",
			source: map[string][]byte{
				"flow/one.md": []byte(`---
on: "github.pull_request.closed"
---
A flow
`),
			},
			event:       setupTestMsg("github", "pull_request", "closed", nil),
			expectedIDs: []string{"flow.one"},
		},

		{
			name: "Shorthand event without conditional",
			source: map[string][]byte{
				"flow/one.md": []byte(`---
on: "pull_request"
---
A flow
`),
			},
			event:       setupTestMsg("github", "pull_request", "closed", nil),
			expectedIDs: []string{"flow.one"},
		},

		{
			name: "Shorthand event with conditional",
			source: map[string][]byte{
				"flow/one.md": []byte(`---
on: "pull_request"
if: event.data != "hello"
---
A flow
`),
			},
			event: setupTestMsg("github", "pull_request", "closed", map[string]any{"data": "hello"}),
		},

		{
			name: "Two flows on event with conditional",
			source: map[string][]byte{
				"flow/one.md": []byte(`---
on: "pull_request"
if: event.data != "hello"
---
A flow
`),
				"flow/two.md": []byte(`---
on: "pull_request"
if: event.data == "hello"
---
A flow
`),
			},
			event:       setupTestMsg("github", "pull_request", "closed", map[string]any{"data": "hello"}),
			expectedIDs: []string{"flow.two"},
		},

		{
			name: "Valid conditional with non-existent key",
			source: map[string][]byte{
				"flow/one.md": []byte(`---
on: "pull_request"
if: can(event.no_such_key)
---
A flow
`),
			},
			event: setupTestMsg("github", "pull_request", "closed", nil),
		},

		{
			name: "Invalid conditional with non-existent function",
			source: map[string][]byte{
				"flow/one.md": []byte(`---
on: "pull_request"
if: not_a_func(event.no_such_key)
---
A flow
`),
			},
			event:       setupTestMsg("github", "pull_request", "closed", nil),
			expectError: true,
		},

		{
			name: "Invalid conditional with non-existent key",
			source: map[string][]byte{
				"flow/one.md": []byte(`---
on: "pull_request"
if: event.no_such_key != "hello"
---
A flow
`),
			},
			event:       setupTestMsg("github", "pull_request", "closed", nil),
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flowsDir := setupPopulatedTestDir(t, tc.source)

			flowReader := NewFlowReader(flowsDir)
			err := flowReader.ReadAll()
			require.NoError(t, err, "Test setup: Failed to read flows")

			matchedFlows, err := MatchFlows(flowReader.IndexedSensors(), tc.event, nil)
			if tc.expectError {
				assert.Error(t, err, "Invalid flows should return an error")
				return
			} else {
				assert.NoError(t, err, "Valid flow & event should not return error")
			}

			flowIDs := []string{}
			for _, f := range matchedFlows {
				flowIDs = append(flowIDs, f.ID)
			}

			assert.ElementsMatch(t, tc.expectedIDs, flowIDs, "Flows that match the event should be returned")
		})
	}
}

func setupTestMsg(source, event, action string, data map[string]any) *nats.HopsMsg {
	payload := map[string]any{
		"hops": map[string]any{
			"source": source,
			"event":  event,
			"action": action,
		},
	}

	if data != nil {
		maps.Copy(payload, data)
	}

	return &nats.HopsMsg{
		Source: source,
		Event:  event,
		Action: action,
		Data:   payload,
	}
}
