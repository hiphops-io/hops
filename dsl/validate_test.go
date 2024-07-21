package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaValidation(t *testing.T) {
	type testCase struct {
		name     string
		hops     string
		numDiags int
	}

	tests := []testCase{
		{
			name: "Simple valid config",
			hops: `on foo handle_foo {worker = "worker"}`,
		},
		{
			name: "Full valid config",
			hops: `
			on anevent_action handle_event {
				worker = "worker"
			}

			on bar handle_bar {
				worker = "worker"
			}

			on bar handle_bar_two {
				worker = "worker"
			}

			schedule hourly {
				cron = "@hourly"
			}

			schedule daily_midnight {
				cron = "0 0 * * *"
			}

			task goodbye {}

			task say_hello {
				display_name = "Send Greeting"
				summary = "Send a greeting"
				description = "Send a greeting to someone of your choosing"
				emoji = "👋"

				param greeting {
					required=true
					type="text"
					default="Hello there"
				}

				param greetee {
					type = "string"
				}

				param a_number {
					type = "number"
				}

				param true_or_false {
					type = "bool"
				}
			}
			`,
		},
		{
			name: "Unknown root attribute",
			hops: `
			on foo my_foo {worker = "worker"}
			bar = ""`,
			numDiags: 1,
		},
		{
			name: "Unknown root block",
			hops: `
			on foo do_thing {worker = "worker"}
			bar {}`,
			numDiags: 1,
		},
		{
			name:     "On too few labels",
			hops:     `on foo {worker = "worker"}`,
			numDiags: 1,
		},
		{
			name: "On unknown attribute",
			hops: `on foo handle_foo {
				an_unknown_attr = "value"
				worker = "worker"
			}`,
			numDiags: 1,
		},
		{
			name: "On unknown block",
			hops: `on foo do_thing {
				worker = "worker"

				an_unknown_block {
					val = "val"
				}
			}`,
			numDiags: 1,
		},
		// Check lots of different ways labels can be invalid across blocks
		{
			name: "Invalid labels",
			hops: `
			on FOO bar {worker = "worker"}
			on foo BAR1 {worker = "worker"}
			on foo areallylonglabelisnotallowed_the_max_is_fifty_chars {worker = "worker"}
			on _foo bar2 {worker = "worker"}
			on foo_ bar3 {worker = "worker"}
			on fo-o bar4 {worker = "worker"}
			on foo__bar buzz {worker = "worker"}
			on areallylonglabelisnotallowed_the_max_is_fifty_chars foo {worker = "worker"}

			task FOO {}

			task foo {
				param FOO {}
			}

			schedule FOO {
				cron = "@daily"
			}
			`,
			numDiags: 11,
		},
		{
			name: "Invalid schedule cron",
			hops: `
			schedule empty {
				cron = ""
			}

			schedule gibberish {
				cron = "Ekekekeh! Ni! Ni! Ni!"
			}

			schedule wrong_cron_format {
				cron = "* * * * * *"
			}
			`,
			numDiags: 3,
		},
		{
			name: "Invalid schedule cron",
			hops: `
			schedule empty {
				cron = ""
			}

			schedule gibberish {
				cron = "ekekekeh! 12 12"
			}

			schedule wrong_cron_format {
				cron = "* * * * * *"
			}
			`,
			numDiags: 3,
		},
		{
			name: "Invalid param type",
			hops: `
			task bad_param {
				param wrong_type {
					type = "int"
				}
			}
			`,
			numDiags: 1,
		},
		{
			name: "Shared names different types",
			hops: `
			on app_event same {worker = "worker"}

			schedule same {
				cron = "@daily"
			}

			task same {}
			`,
		},
		{
			name: "Duplicate on names",
			hops: `
			on app_event same {worker = "worker"}
			on app_event same {worker = "worker"}
			`,
			numDiags: 1,
		},
		{
			name: "Duplicate schedule names",
			hops: `
			schedule dupe {
				cron = "@daily"
			}

			schedule dupe {
				cron = "@hourly"
			}
			`,
			numDiags: 1,
		},
		{
			name: "Duplicate task names",
			hops: `
			task duplicate_name {}

			task duplicate_name {}
			`,
			numDiags: 1,
		},
		{
			name: "Shared param names different tasks",
			hops: `
			task foo {
				param foo {}
			}

			task bar {
				param foo {}
			}
			`,
		},
		{
			name: "Duplicate param names",
			hops: `
			task foo {
				param foo {}
				param foo {}
			}
			`,
			numDiags: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			files := []*AutomationFile{{Path: "test.hops", Content: []byte(tc.hops)}}
			_, diags := NewAutomations(files)

			assert.Lenf(
				t, diags.Errs(), tc.numDiags,
				"Schema should return %d diagnostic errors, got %d",
				tc.numDiags, len(diags.Errs()),
			)
		})
	}
}

func TestAutomationValidation(t *testing.T) {
	type testCase struct {
		name     string
		files    []*AutomationFile
		numDiags int
	}

	tests := []testCase{
		{
			name: "Valid",
			files: []*AutomationFile{
				{"one/main.hops", []byte(`on foo ensure_foo_is_good {worker = "worker"}`)},
				{"one/other.txt", []byte(``)},
			},
			numDiags: 0,
		},
		{
			name: "Invalid hops",
			files: []*AutomationFile{
				{
					"one/invalid.hops",
					[]byte(`
						on foo bar {
							worker = "worker"
							no_such_attribute = "anything"
						}
					`),
				},
				{"two/invalid.hops", []byte(`on only_one_label {worker = "worker"}`)},
			},
			numDiags: 2,
		},
		{
			name: "Duplicate names in single hops file",
			files: []*AutomationFile{
				{
					"one/invalid.hops",
					[]byte(`
						on foo do_good_foo {worker = "worker"}
						on foo do_good_foo {worker = "worker"}
					`),
				},
			},
			numDiags: 1,
		},
		{
			name: "Duplicate names in multiple hops files in single automation",
			files: []*AutomationFile{
				{
					"one/main.hops",
					[]byte(`on foo do_good_foo {worker = "worker"}`),
				},
				{
					"one/copy.hops",
					[]byte(`on foo do_good_foo {worker = "worker"}`),
				},
			},
			numDiags: 1,
		},
		{
			name: "Duplicate names in multiple automations",
			files: []*AutomationFile{
				{
					"one/main.hops",
					[]byte(`on foo do_good_foo {worker = "worker"}`),
				},
				{
					"two/main.hops",
					[]byte(`
						on foo do_good_foo {worker = "worker"}
					`),
				},
			},
			numDiags: 1,
		},
		{
			name: "Invalid manifest",
			files: []*AutomationFile{
				{"one/manifest.yaml", []byte(`manifest_name: "Foo"`)},
			},
			numDiags: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, d := NewAutomations(tc.files)

			assert.Lenf(
				t, d.Errs(), tc.numDiags,
				"Automation should return %d diagnostic error(s), got %d",
				tc.numDiags, len(d.Errs()),
			)
		})
	}
}
