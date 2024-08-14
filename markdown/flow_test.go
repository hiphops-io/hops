package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlowReader(t *testing.T) {
	type testCase struct {
		name        string
		source      map[string][]byte
		expected    map[string][]*Flow
		expectError bool
	}

	tests := []testCase{
		{
			name: "Single flow",
			source: map[string][]byte{
				"first_flow/hello.md": []byte(`---
on: "github.pull_request.closed"
schedule: "* * * * *"
command:
- p: {type: text, default: Hello, required: false}
if: event.branch == "main"
---
This is a basic flow that runs on closed PRs
`),
			},
			expected: map[string][]*Flow{
				"github.pull_request.closed": {
					{
						On:       "github.pull_request.closed",
						Schedule: "* * * * *",
						Worker:   "first_flow.hello",
						ID:       "first_flow.hello",
						If:       `event.branch == "main"`,
						Command: Command{
							{"p": Param{"text", "Hello", false}},
						},
					},
				},
				"hiphops.schedule.first_flow-hello": {
					{
						On:       "github.pull_request.closed",
						Schedule: "* * * * *",
						Worker:   "first_flow.hello",
						ID:       "first_flow.hello",
						If:       `event.branch == "main"`,
						Command: Command{
							{"p": Param{"text", "Hello", false}},
						},
					},
				},
				"*.command.first_flow-hello": {
					{
						On:       "github.pull_request.closed",
						Schedule: "* * * * *",
						Worker:   "first_flow.hello",
						ID:       "first_flow.hello",
						If:       `event.branch == "main"`,
						Command: Command{
							{"p": Param{"text", "Hello", false}},
						},
					},
				},
			},
		},

		{
			name: "Multiple flows",
			source: map[string][]byte{
				"first_flow/one.md": []byte(`---
on: "pull_request"
---
Flow one
`),
				"first_flow/two.md": []byte(`---
on: pull_request
---
Flow two
`),
				"second_flow/one.md": []byte(`---
on: "github.pull_request.*"
worker: first_flow.one
---
Other flow one
`),
				"second_flow/two.md": []byte(`---
on: "pull_request.opened"
worker: one
---
Other flow two
`),
			},
			expected: map[string][]*Flow{
				"*.pull_request.*": {
					{
						On:     "pull_request",
						Worker: "first_flow.one",
						ID:     "first_flow.one",
					},
					{
						On:     "pull_request",
						Worker: "first_flow.two",
						ID:     "first_flow.two",
					},
				},
				"github.pull_request.*": {
					{
						On:     "github.pull_request.*",
						Worker: "first_flow.one",
						ID:     "second_flow.one",
					},
				},
				"*.pull_request.opened": {
					{
						On:     "pull_request.opened",
						Worker: "second_flow.one",
						ID:     "second_flow.two",
					},
				},
			},
		},

		{
			name: "Ignores nested flow",
			source: map[string][]byte{
				"first_flow/hello.md": []byte(`---
on: "github.pull_request.closed"
---
A flow
`),
				"first_flow/subdir/hello.md": []byte(`---
on: "github.pull_request.closed"
---
A flow
`),
			},
			expected: map[string][]*Flow{
				"github.pull_request.closed": {
					{
						On:     "github.pull_request.closed",
						Worker: "first_flow.hello",
						ID:     "first_flow.hello",
					},
				},
			},
		},

		{
			name: "Invalid cron",
			source: map[string][]byte{
				"first_flow/hello.md": []byte(`---
schedule: "* * * * * *"
---
Flow
`),
			},
			expectError: true,
		},

		{
			name: "Invalid on",
			source: map[string][]byte{
				"first_flow/hello.md": []byte(`---
on: source.event.action.nothing
---
Flow
`),
			},
			expectError: true,
		},

		{
			name: "Invalid expression",
			source: map[string][]byte{
				"first_flow/hello.md": []byte(`---
on: source.event.action
if: meaning..life != 42
---
Flow
`),
			},
			expectError: true,
		},

		{
			name: "No triggers",
			source: map[string][]byte{
				"first_flow/hello.md": []byte(`---
---
Flow
`),
			},
			expectError: true,
		},

		{
			name: "Unknown param type",
			source: map[string][]byte{
				"first_flow/hello.md": []byte(`---
command:
- foo: {type: "madeupstuff"}
---
Flow
`),
			},
			expectError: true,
		},

		{
			name: "Invalid bool default for text param",
			source: map[string][]byte{
				"first_flow/hello.md": []byte(`---
command:
- foo: {type: "text", default: true}
---
Flow
`),
			},
			expectError: true,
		},

		{
			name: "Invalid number default for string param",
			source: map[string][]byte{
				"first_flow/hello.md": []byte(`---
command:
- foo: {type: "string", default: 1}
---
Flow
`),
			},
			expectError: true,
		},

		{
			name: "Invalid text default for bool param",
			source: map[string][]byte{
				"first_flow/hello.md": []byte(`---
command:
  foo: {type: "bool", default: "Hello!"}
---
Flow
`),
			},
			expectError: true,
		},

		{
			name: "Invalid text default for number param",
			source: map[string][]byte{
				"first_flow/hello.md": []byte(`---
command:
- foo: {type: "number", default: "Hello!"}
---
Flow
`),
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flowsDir := setupPopulatedTestDir(t, tc.source)

			flowReader := NewFlowReader(flowsDir)
			err := flowReader.ReadAll()
			if tc.expectError {
				assert.Error(t, err, "Invalid flows should return error")
				return
			}

			flowIdx := flowReader.IndexedSensors()

			assert.NoError(t, err, "Flows should parse without error")
			for key, expectedFlows := range tc.expected {
				if !assert.Len(t, flowIdx[key], len(expectedFlows), "Index key should contain expected flows") {
					continue
				}

				for i, flow := range expectedFlows {
					assert.EqualExportedValues(t, *flow, *flowIdx[key][i])
				}
			}
		})
	}
}
