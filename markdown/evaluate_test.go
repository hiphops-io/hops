package markdown

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func TestMatchFlows(t *testing.T) {
	type testCase struct {
		name        string
		source      map[string][]byte
		event       []byte
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
			event:       setupTestEvent(t, "github", "pull_request", "closed", nil),
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
			event:       setupTestEvent(t, "github", "pull_request", "closed", nil),
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
			event: setupTestEvent(t, "github", "pull_request", "closed", map[string]any{"data": "hello"}),
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
			event:       setupTestEvent(t, "github", "pull_request", "closed", map[string]any{"data": "hello"}),
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
			event: setupTestEvent(t, "github", "pull_request", "closed", nil),
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
			event:       setupTestEvent(t, "github", "pull_request", "closed", nil),
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
			event:       setupTestEvent(t, "github", "pull_request", "closed", nil),
			expectError: true,
		},

		{
			name: "Invalid hops metadata",
			source: map[string][]byte{
				"flow/one.md": []byte(`---
on: "pull_request"
---
A flow
`),
			},
			event:       setupTestEvent(t, "github", "", "closed", nil),
			expectError: true,
		},

		{
			name: "Invalid event",
			source: map[string][]byte{
				"flow/one.md": []byte(`---
on: "pull_request"
---
A flow
`),
			},
			event:       []byte("not json"),
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flowsDir := setupPopulatedTestDir(t, tc.source)

			flowReader := NewFlowReader(flowsDir)
			flowIdx, err := flowReader.ReadAll()
			require.NoError(t, err, "Test setup: Failed to read flows")

			matchedFlows, err := MatchFlows(flowIdx, tc.event)
			if tc.expectError {
				assert.Error(t, err, "Invalid flows should return an error")
				return
			} else {
				assert.NoError(t, err, "Valid flow & event should not return erro")
			}

			flowIDs := []string{}
			for _, f := range matchedFlows {
				flowIDs = append(flowIDs, f.ID)
			}

			assert.ElementsMatch(t, tc.expectedIDs, flowIDs, "Flows that match the event should be returned")
		})
	}
}

func setupTestEvent(t *testing.T, source, event, action string, data map[string]any) []byte {
	payload := map[string]any{
		"hops": map[string]string{
			"source": source,
			"event":  event,
			"action": action,
		},
	}

	if data != nil {
		maps.Copy(payload, data)
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err, "Test setup: Test event should marshall without error")

	return payloadBytes
}
