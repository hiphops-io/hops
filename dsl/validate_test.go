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
			hops: `on foo {}`,
		},
		{
			name: "Full valid config",
			hops: `
			on anevent_action {
				name = "anevent"

				call app_handler {
					name = "first_call"
			
					if = lower("FOO") == "foo"
			
					inputs = {
						foo = "bar"
					}
				}
			
				call app_handler {}
			
				call app_handler {
					if = first_call.completed
				}
			
				done {
					errored = first_call.errored
					completed = first_call.completed
				}
			}

			on bar {}

			on bar {}
			
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
			on foo {}
			bar = ""`,
			numDiags: 1,
		},
		{
			name: "Unknown root block",
			hops: `
			on foo {}
			bar {}`,
			numDiags: 1,
		},
		{
			name:     "On too many labels",
			hops:     `on foo bar {}`,
			numDiags: 1,
		},
		{
			name: "On unknown attribute",
			hops: `on foo {
				an_unknown_attr = "value"
			}`,
			numDiags: 1,
		},
		{
			name: "On unknown block",
			hops: `on foo {
				an_unknown_block {
					val = "val"
				}
			}`,
			numDiags: 1,
		},
		{
			name: "Call inputs defined as block",
			hops: `
			on foo {
				call app_handler {
					inputs {
						value = "hey"
					}
				}
			}`,
			numDiags: 1,
		},
		// Check lots of different ways labels can be invalid across blocks
		{
			name: "Invalid labels",
			hops: `
			on FOO {}
			on _foo {}
			on foo_ {}
			on fo-o {}
			on foo__bar {}
			on areallylonglabelisnotallowed_the_max_is_fifty_chars {}

			on bar {
				call FOO {}
				call _foo {}
				call foo_ {}
				call fo-o {}
				call foo__bar {}
				call areallylonglabelisnotallowed_the_max_is_fifty_chars {}
			}
			
			task FOO {}

			task foo {
				param FOO {}
			}

			schedule FOO {
				cron = "@daily"
			}
			`,
			numDiags: 15,
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
			on app_event {
				name = "same"
			}

			schedule same {
				cron = "@daily"
			}

			task same {}
			`,
		},
		{
			name: "Duplicate on names",
			hops: `
			on app_event {
				name = "same"
			}

			on app_event {
				name = "same"
			}
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
			name: "Shared call names different on",
			hops: `
			on foo {
				call app_handler {
					name = "same"
				}
			}

			on foo {
				call app_handler {
					name = "same"
				}
			}
			`,
		},
		{
			name: "Duplicate call names",
			hops: `
			on foo {
				call app_handler {
					name = "same"
				}

				call otherapp_handler {
					name = "same"
				}
			}
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
				{"one/main.hops", []byte(`on foo {}`)},
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
						on foo {
							call app_handler {
								inputs {
									inputs_should_not = "be_a_block"
								}
							}
						}
					`),
				},
				{"two/invalid.hops", []byte(`on foo extra_label {}`)},
			},
			numDiags: 2,
		},
		{
			name: "Duplicate names in single hops file",
			files: []*AutomationFile{
				{
					"one/invalid.hops",
					[]byte(`
						on foo {
							name = "the_same"
						}

						on foo {
							name = "the_same"
						}
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
					[]byte(`
						on foo {
							name = "the_same"
						}
					`),
				},
				{
					"one/copy.hops",
					[]byte(`
						on foo {
							name = "the_same"
						}
					`),
				},
			},
			numDiags: 1,
		},
		{
			name: "Duplicate names in multiple automations",
			files: []*AutomationFile{
				{
					"one/main.hops",
					[]byte(`
						on foo {
							name = "the_same"
						}
					`),
				},
				{
					"two/main.hops",
					[]byte(`
						on foo {
							name = "the_same"
						}
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
