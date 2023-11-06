package dsl

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskParse(t *testing.T) {
	type testCase struct {
		name  string
		hops  string
		cmds  []TaskAST
		valid bool
	}

	tests := []testCase{
		// Test that a simple valid task is parsed correctly
		{
			name: "Simple valid task",
			hops: `task foo {}`,
			cmds: []TaskAST{
				{Name: "foo", DisplayName: "Foo"},
			},
			valid: true,
		},

		// Test that display name can be overridden
		{
			name: "Task with explicit display name",
			hops: `
			task foo {
				display_name = "Run Foo Task"
			}
			`,
			cmds: []TaskAST{
				{Name: "foo", DisplayName: "Run Foo Task"},
			},
			valid: true,
		},

		// Test that a task with basic validation errors (extra labels) throws an error
		{
			name:  "Simple invalid task",
			hops:  `task foo bar {}`,
			valid: false,
		},

		// Test metadata fields are parsed
		{
			name: "Simple valid task with metadata",
			hops: `task run_foo {
				summary = "Run a foo"
				description = "Run you a foo for great good!"
				emoji = "🤖"
			}`,
			cmds: []TaskAST{
				{
					Name:        "run_foo",
					DisplayName: "Run Foo",
					Summary:     "Run a foo",
					Description: "Run you a foo for great good!",
					Emoji:       "🤖",
				},
			},
			valid: true,
		},

		// Test that duplicate tasks throw an error
		{
			name: "Duplicate tasks throw errors",
			hops: `
		task foo {}
		task foo {}`,
			cmds:  []TaskAST{},
			valid: false,
		},

		// Test that a task parses correctly when other resources exist in the config
		{
			name: "Other hops resources",
			hops: `
		on push {
			if = event.repo == "foo"
		}

		task foo {}`,
			cmds: []TaskAST{
				{Name: "foo", DisplayName: "Foo"},
			},
			valid: true,
		},

		// Test that a param without options is parsed correctly
		{
			name: "Task with simple param",
			hops: `
		task foo {
			param a {}
		}`,
			cmds: []TaskAST{
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
			valid: true,
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
			cmds: []TaskAST{
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
			valid: true,
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
			cmds: []TaskAST{
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
			valid: true,
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
			cmds: []TaskAST{
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
			valid: true,
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
			cmds:  []TaskAST{},
			valid: false,
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
			cmds: []TaskAST{
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
			valid: true,
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
			cmds:  []TaskAST{},
			valid: false,
		},

		// Test that duplicate param names throw an error
		{
			name: "Task with duplicate param names",
			hops: `
		task foo {
			param a {}
			param a {}
		}`,
			cmds:  []TaskAST{},
			valid: false,
		},
		// Test no tasks doesn't throw an error
		{
			name:  "No tasks",
			hops:  `on push {}`,
			cmds:  []TaskAST{},
			valid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			hopsHcl, _ := createTmpHopsFile(tc.hops, t)

			hop, err := ParseHopsTasks(ctx, hopsHcl)

			// Ditch early if we're expecting invalid parsing
			if !tc.valid {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.ElementsMatch(t, tc.cmds, hop.Tasks)
			assert.ElementsMatch(t, tc.cmds, hop.ListTasks())
		})
	}
}

func createTmpHopsFile(content string, t *testing.T) (hcl.Body, string) {
	dir := t.TempDir()
	f, err := os.CreateTemp(dir, "*")
	require.NoError(t, err)

	f.WriteString(content)

	hops, err := ReadHopsFilePath(f.Name())
	require.NoError(t, err)

	return hops.Body, hops.Hash
}
