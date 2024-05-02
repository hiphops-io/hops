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
				{"one/main.hops", []byte(`on foo ping_server {}`)},
			},
			filePaths:  []string{"one/main.hops"},
			numHopsOns: 1,
		},
		{
			name: "Single automation, multiple hops files",
			files: []*AutomationFile{
				{"one/main.hops", []byte(`on foo approve_request {}`)},
				{"one/other.hops", []byte(`on foo reject_request {}`)},
			},
			filePaths:  []string{"one/main.hops", "one/other.hops"},
			numHopsOns: 2,
		},
		{
			name: "Multiple automations with only hops",
			files: []*AutomationFile{
				{"one/main.hops", []byte(`on foo draw_cat_picture {}`)},
				{"two/main.hops", []byte(`on foo run_report {}`)},
			},
			filePaths:  []string{"one/main.hops", "two/main.hops"},
			numHopsOns: 2,
		},
		{
			name: "Automations with other files",
			files: []*AutomationFile{
				{"one/main.hops", []byte(`on foo run_tests {}`)},
				{"one/userList.json", []byte(`["lizzie@example.com", "dave@example.com"]`)},
				{"two/pipelines.hops", []byte(`on foo approve_small_change {}`)},
				{"two/notes.txt", []byte(`This automation contains nice useful pipelines`)},
			},
			filePaths:  []string{"one/main.hops", "one/userList.json", "two/pipelines.hops", "two/notes.txt"},
			numHopsOns: 2,
		},
		{
			name: "Automations with manifests",
			files: []*AutomationFile{
				{"one/main.hops", []byte(`on foo run_tests {}`)},
				{"one/manifest.yaml", []byte(sampleManifest)},
				{
					"two/hippity.hops",
					[]byte(`
						on foo send_result_msg {}

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
					[]byte(`on foo handle {}`),
				},
				{
					"two/main.hops",
					[]byte(`
						on foo reject_bad {}
						on bar handle_good {}
					`),
				},
			},
			expectedHops: `{
				"ons": [
					{"label": "foo", "name": "handle", "handler": "handle"},
					{"label": "foo", "name": "reject_bad", "handler": "handle"},
					{"label": "bar", "name": "handle_good", "handler": "handle"}
				]
			}`,
		},
		{
			name: "On blocks",
			files: []*AutomationFile{
				{
					"one/main.hops",
					[]byte(`
						on event_action pipeline {
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
						on foo handle_foo {}
						on event_action handle_event {}
						on foo_action perform_foo_action {}
					`),
				},
				{
					"two/main.hops",
					[]byte(`
						on foo check_success {}
						on bar reject_bad_change {}
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
