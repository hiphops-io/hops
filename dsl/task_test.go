package dsl

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskParse(t *testing.T) {
	type testCase struct {
		name       string
		hops       string
		tasks      []TaskAST
		validParse bool
		validRead  bool
	}

	tests := []testCase{
		// Test that a simple valid task is parsed correctly
		{
			name: "Simple valid task",
			hops: `task foo {}`,
			tasks: []TaskAST{
				{Name: "foo", DisplayName: "Foo"},
			},
			validParse: true,
			validRead:  true,
		},

		// Test that display name can be overridden
		{
			name: "Task with explicit display name",
			hops: `
			task foo {
				display_name = "Run Foo Task"
			}
			`,
			tasks: []TaskAST{
				{Name: "foo", DisplayName: "Run Foo Task"},
			},
			validParse: true,
			validRead:  true,
		},

		// Test that a task with basic validation errors (extra labels) throws an error
		{
			name:       "Simple invalid task",
			hops:       `task foo bar {}`,
			validParse: false,
			validRead:  false,
		},

		// Test that tasks with param name same as task name does not throw error
		{
			name: "Param name matches task name",
			hops: `
				task foo {
					param foo {}
				}
			`,
			tasks: []TaskAST{
				{
					Name:        "foo",
					DisplayName: "Foo",
					Params: []ParamAST{
						{
							Name:        "foo",
							DisplayName: "Foo",
							Type:        "string",
							Flag:        "--foo",
						},
					},
				},
			},
			validParse: true,
			validRead:  true,
		},

		// Test that tasks with same named params do not throw an error
		{
			name: "Shared param names",
			hops: `
				task first {
					param bar {}
				}

				task second {
					param bar {}
				}
			`,
			tasks: []TaskAST{
				{
					Name:        "first",
					DisplayName: "First",
					Params: []ParamAST{
						{
							Name:        "bar",
							DisplayName: "Bar",
							Type:        "string",
							Flag:        "--bar",
						},
					},
				},
				{
					Name:        "second",
					DisplayName: "Second",
					Params: []ParamAST{
						{
							Name:        "bar",
							DisplayName: "Bar",
							Type:        "string",
							Flag:        "--bar",
						},
					},
				},
			},
			validParse: true,
			validRead:  true,
		},

		// Test metadata fields are parsed
		{
			name: "Simple valid task with metadata",
			hops: `task run_foo {
				summary = "Run a foo"
				description = "Run you a foo for great good!"
				emoji = "🤖"
			}`,
			tasks: []TaskAST{
				{
					Name:        "run_foo",
					DisplayName: "Run Foo",
					Summary:     "Run a foo",
					Description: "Run you a foo for great good!",
					Emoji:       "🤖",
				},
			},
			validParse: true,
			validRead:  true,
		},

		// Test that duplicate tasks throw an error
		{
			name: "Duplicate tasks throw errors",
			hops: `
		task foo {}
		task foo {}`,
			tasks:      []TaskAST{},
			validParse: false,
			validRead:  true,
		},

		// Test that a task parses correctly when other resources exist in the config
		{
			name: "Other hops resources",
			hops: `
		on push {
			if = event.repo == "foo"
		}

		task foo {}`,
			tasks: []TaskAST{
				{Name: "foo", DisplayName: "Foo"},
			},
			validParse: true,
			validRead:  true,
		},

		// Test that a param without options is parsed correctly
		{
			name: "Task with simple param",
			hops: `
		task foo {
			param a {}
		}`,
			tasks: []TaskAST{
				{
					Name:        "foo",
					DisplayName: "Foo",
					Params: []ParamAST{
						{
							Name:        "a",
							DisplayName: "A",
							Type:        reflect.String.String(),
							Default:     nil,
							Help:        "",
							Flag:        "--a",
							Required:    false,
						},
					},
				},
			},
			validParse: true,
			validRead:  true,
		},

		// Test that a string param with simple options is parsed correctly
		{
			name: "Task with param config options",
			hops: `
task foo {
	param a {
		default = "avalue"
		help = "Helpful help"
	}
}`,
			tasks: []TaskAST{
				{
					Name:        "foo",
					DisplayName: "Foo",
					Params: []ParamAST{
						{
							Name:        "a",
							DisplayName: "A",
							Type:        reflect.String.String(),
							Default:     "avalue",
							Help:        "Helpful help",
							Flag:        "--a",
							Required:    false,
						},
					},
				},
			},
			validParse: true,
			validRead:  true,
		},

		// Test that non-string param types are parsed correctly
		{
			name: "Task with typed params",
			hops: `
		task foo {
			param a {
				default = 1
				type = "number"
			}

			param b {
				display_name = "B Param"
				default = true
				type = "bool"
			}
		}`,
			tasks: []TaskAST{
				{
					Name:        "foo",
					DisplayName: "Foo",
					Params: []ParamAST{
						{
							Name:        "a",
							DisplayName: "A",
							Type:        "number",
							Default:     float64(1),
							Help:        "",
							Flag:        "--a",
							Required:    false,
						},
						{
							Name:        "b",
							DisplayName: "B Param",
							Type:        "bool",
							Default:     true,
							Help:        "",
							Flag:        "--b",
							Required:    false,
						},
					},
				},
			},
			validParse: true,
			validRead:  true,
		},

		// Test that int params default to nil, rather than 0
		{
			name: "Task with number and no default",
			hops: `
		task foo {
			param a {
				type = "number"
			}
		}`,
			tasks: []TaskAST{
				{
					Name:        "foo",
					DisplayName: "Foo",
					Params: []ParamAST{
						{
							Name:        "a",
							DisplayName: "A",
							Type:        "number",
							Default:     nil,
							Help:        "",
							Flag:        "--a",
							Required:    false,
						},
					},
				},
			},
			validParse: true,
			validRead:  true,
		},

		// Test that incorrect type/default pairs throw an error
		{
			name: "Task param with mismatched type and default",
			hops: `
		task foo {
			param a {
				type = "int"
				default = true
			}
		}`,
			tasks:      []TaskAST{},
			validParse: false,
			validRead:  true,
		},

		// Test that default values from expressions are set correctly
		{
			name: "Task param with default from expression",
			hops: `
		task foo {
			param a {
				type = "bool"
				default = "foo" == "foo"
			}
		}`,
			tasks: []TaskAST{
				{
					Name:        "foo",
					DisplayName: "Foo",
					Params: []ParamAST{
						{
							Name:        "a",
							DisplayName: "A",
							Type:        "bool",
							Default:     true,
							Help:        "",
							Flag:        "--a",
							Required:    false,
						},
					},
				},
			},
			validParse: true,
			validRead:  true,
		},

		// Test that unknown types throw errors
		{
			name: "Task param with unknown type",
			hops: `
		task foo {
			param a {
				type = "nosuchtype"
				default = "whatever"
			}
		}`,
			tasks:      []TaskAST{},
			validParse: false,
			validRead:  true,
		},

		// Test that duplicate param names throw an error
		{
			name: "Task with duplicate param names",
			hops: `
		task foo {
			param a {}
			param a {}
		}`,
			tasks:      []TaskAST{},
			validParse: false,
			validRead:  true,
		},
		// Test no tasks doesn't throw an error
		{
			name:       "No tasks",
			hops:       `on push {}`,
			tasks:      []TaskAST{},
			validParse: true,
			validRead:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Ditch early if we're expecting invalid parsing
			hops, err := createTmpHopsFile(tc.hops, t)
			if !tc.validRead {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			hop, err := ParseHopsTasks(ctx, hops)
			if !tc.validParse {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.ElementsMatch(t, tc.tasks, hop.Tasks)
			assert.ElementsMatch(t, tc.tasks, hop.ListTasks())
		})
	}
}

// createTmpHopsFile creates a temporary hops file in a subdirectory
// with the given content and returns the parsed HCL body content
func createTmpHopsFile(content string, t *testing.T) (*HopsFiles, error) {
	// temporary directory
	tmpDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFilename := filepath.Join(tmpDir, "hopsdir/hops.hops")
	if err := os.MkdirAll(filepath.Dir(tmpFilename), 0755); err != nil {
		t.Fatalf("Failed to create subdirectory for file %s: %s", tmpFilename, err)
	}
	err = os.WriteFile(tmpFilename, []byte(content), 0666)
	if err != nil {
		t.Fatalf("Failed to write to temp file %s: %s", tmpFilename, err)
	}

	hops, err := ReadHopsFilePath(tmpDir)
	if err != nil {
		return nil, err
	}

	return hops, nil
}
