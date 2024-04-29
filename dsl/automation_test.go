package dsl

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

const sampleManifest = `
---
version: v0.1
name: Blank template
description: This is an empty template so you can start from scratch! The world is your oyster.
emoji: 📋
`

func TestAutomationLoading(t *testing.T) {
	type testCase struct {
		name          string
		files         []*AutomationFile
		filePaths     []string
		manifestPaths []string
		numHopsOns    int
	}

	tests := []testCase{
		{
			name: "Single hops file",
			files: []*AutomationFile{
				{"one/main.hops", []byte(`on foo {}`)},
			},
			filePaths:  []string{"one/main.hops"},
			numHopsOns: 1,
		},
		{
			name: "Single automation, multiple hops files",
			files: []*AutomationFile{
				{"one/main.hops", []byte(`on foo {}`)},
				{"one/other.hops", []byte(`on foo {}`)},
			},
			filePaths:  []string{"one/main.hops", "one/other.hops"},
			numHopsOns: 2,
		},
		{
			name: "Multiple automations with only hops",
			files: []*AutomationFile{
				{"one/main.hops", []byte(`on foo {}`)},
				{"two/main.hops", []byte(`on foo {}`)},
			},
			filePaths:  []string{"one/main.hops", "two/main.hops"},
			numHopsOns: 2,
		},
		{
			name: "Automations with other files",
			files: []*AutomationFile{
				{"one/main.hops", []byte(`on foo {}`)},
				{"one/userList.json", []byte(`["lizzie@example.com", "dave@example.com"]`)},
				{"two/pipelines.hops", []byte(`on foo {}`)},
				{"two/notes.txt", []byte(`This automation contains nice useful pipelines`)},
			},
			filePaths:  []string{"one/main.hops", "one/userList.json", "two/pipelines.hops", "two/notes.txt"},
			numHopsOns: 2,
		},
		{
			name: "Automations with manifests",
			files: []*AutomationFile{
				{"one/main.hops", []byte(`on foo {}`)},
				{"one/manifest.yaml", []byte(sampleManifest)},
				{
					"two/hippity.hops",
					[]byte(`
						on foo {}

						task bar {}
					`),
				},
				{"two/manifest.yml", []byte(sampleManifest)},
			},
			filePaths:     []string{"one/main.hops", "one/manifest.yaml", "two/hippity.hops", "two/manifest.yml"},
			manifestPaths: []string{"one", "two"},
			numHopsOns:    2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			automations, _ := NewAutomations(tc.files)

			require.NotNil(t, automations.Hops)
			assert.Len(t, automations.Hops.Ons, tc.numHopsOns)

			gotManifestPaths := maps.Keys(automations.Manifests)
			assert.ElementsMatch(t, tc.manifestPaths, gotManifestPaths, "Automations should load any manifests")

			gotFilePaths := maps.Keys(automations.Files)
			assert.ElementsMatch(t, tc.filePaths, gotFilePaths, "Automations should load any other files")
		})
	}
}

func TestAutomationHopsDecoding(t *testing.T) {
	type testCase struct {
		name  string
		files []*AutomationFile
		// We use a JSON string as it allows us to ignore all the internal plumbing
		// fields within an AST for equality comparison
		expectedHops string
	}

	tests := []testCase{
		{
			name: "Multiple hops files",
			files: []*AutomationFile{
				{
					"one/main.hops",
					[]byte(`on foo {}`),
				},
				{
					"two/main.hops",
					[]byte(`
						on foo {}
						on bar {}
					`),
				},
			},
			expectedHops: `{
				"ons": [
					{"label": "foo", "handler": "handle"},
					{"label": "foo", "handler": "handle"},
					{"label": "bar", "handler": "handle"}
				]
			}`,
		},
		{
			name: "On blocks",
			files: []*AutomationFile{
				{
					"one/main.hops",
					[]byte(`
						on event_action {
							name = "pipeline"
							if = true != false
						}
					`),
				},
			},
			expectedHops: `{
				"ons": [
					{
						"label": "event_action",
						"name": "pipeline",
						"handler": "handle"
					}
				]
			}`,
		},
		{
			name: "Task and param blocks",
			files: []*AutomationFile{
				{
					"one/main.hops",
					[]byte(`
						task atask {}
						task some_task {
							display_name = "Some Task!"
							description = "A useful task"
							summary = "Useful"
							emoji = "🦄"

							param p_one {}

							param p_two {
								required = true
								type = "text"
								default = "a"
								display_name = "Param Two"
							}
						}
					`),
				},
				{
					"two/main.hops",
					[]byte(`
						task other_task {
							display_name = upper("other task")
						}
					`),
				},
				{
					"two/name.txt",
					[]byte(`Hey`),
				},
			},
			expectedHops: `{
				"tasks": [
					{
						"name": "atask",
						"display_name": "Atask",
						"filepath": "one/main.hops"
					},
					{
						"name": "some_task",
						"display_name": "Some Task!",
						"description": "A useful task",
						"summary": "Useful",
						"emoji": "🦄",
						"filepath": "one/main.hops",
						"params": [
							{
								"name": "p_one",
								"type": "string",
								"required": false,
								"display_name": "P One",
								"flag": "--p_one"
							},
							{
								"name": "p_two",
								"type": "text",
								"required": true,
								"default": "a",
								"display_name": "Param Two",
								"flag": "--p_two"
							}
						]
					},
					{
						"name": "other_task",
						"display_name": "OTHER TASK",
						"filepath": "two/main.hops"
					}
				]
			}`,
		},
		{
			name: "Schedules",
			files: []*AutomationFile{
				{
					"one/main.hops",
					[]byte(`
						schedule hourly {
							cron = "@hourly"
						}

						schedule midnight {
							cron = "0 0 * * *"
							inputs = {
								foo = upper("foo")
							}
						}
					`),
				},
			},
			expectedHops: `{
				"schedules": [
					{
						"name": "hourly",
						"cron": "@hourly"
					},
					{
						"name": "midnight",
						"cron": "0 0 * * *",
						"inputs": "eyJmb28iOiJGT08ifQ=="
					}
				]
			}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := NewAutomations(tc.files)
			hopsJSON, err := json.Marshal(a.Hops)
			require.NoError(t, err, "Test setup: Hops should correcly marshal to JSON")

			assert.JSONEq(t, tc.expectedHops, string(hopsJSON), "Decoded hops should match input")
		})
	}
}

func TestAutomationIndexing(t *testing.T) {
	type testCase struct {
		name          string
		files         []*AutomationFile
		expectedIndex map[string]int
	}

	tests := []testCase{
		{
			name: "Multiple hops files",
			files: []*AutomationFile{
				{
					"one/main.hops",
					[]byte(`
						on foo {}
						on event_action {}
						on foo_action {}
					`),
				},
				{
					"two/main.hops",
					[]byte(`
						on foo {}
						on bar {}
					`),
				},
			},
			expectedIndex: map[string]int{
				"foo":          2,
				"foo_action":   1,
				"bar":          1,
				"event_action": 1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := NewAutomations(tc.files)
			for event, count := range tc.expectedIndex {
				require.Contains(t, a.Hops.eventIndex, event)
				assert.Lenf(
					t, a.Hops.eventIndex[event], count,
					"Event index should contain %d matching ons, got %d",
					count, len(a.Hops.eventIndex[event]),
				)
			}
		})
	}
}
