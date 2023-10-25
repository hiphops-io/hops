package dsl

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidParse(t *testing.T) {
	logger := initTestLogger()
	ctx := context.Background()

	// test that split hops files have identical result as single hops file
	hopsFiles := []string{
		"./testdata/valid.hops",
		"./testdata/multi-hops",
	}

	for _, hopsFile := range hopsFiles {
		eventFile := "./testdata/raw_change_event.json"

		eventData, err := os.ReadFile(eventFile)
		require.NoError(t, err)

		eventBundle := map[string][]byte{
			"event": eventData,
		}

		hclFile, _, err := ReadHopsFiles(hopsFile)
		assert.NoError(t, err)

		hop, err := ParseHops(ctx, hclFile, eventBundle, logger)
		assert.NoError(t, err)

		// Test we parsed the correct number of matching on blocks.
		require.Len(t, hop.Ons, 3)

		// Test the first on block had the proper values
		assert.Equal(t, "change_merged", hop.Ons[0].EventType)
		assert.Equal(t, `a_sensor`, hop.Ons[0].Slug)

		// Test the second on block had the proper values
		assert.Equal(t, "change", hop.Ons[1].EventType)
		assert.Len(t, hop.Ons[1].Calls, 0)

		// Test the index named on block had the proper values
		assert.Equal(t, "change2", hop.Ons[2].Name)
		assert.Equal(t, "change2", hop.Ons[2].Slug)

		// Now dig into calls in the first on block, checking how they were parsed
		require.Len(t, hop.Ons[0].Calls, 2)

		call := hop.Ons[0].Calls[0]
		assert.Equal(t, `a_sensor-first_task`, call.Slug)
		assert.JSONEq(t, `{"a":"b", "from_env": ""}`, string(call.Inputs))

		call = hop.Ons[0].Calls[1]
		assert.Equal(t, `a_sensor-index_id_call2`, call.Slug)
	}
}

// This has duplication with the above test.
// Ideally we'll move them both to a single table based test, but there's a bit
// of work there due to the nature of the test reaching into deep data structures to check values
func TestValidParseResponseStep(t *testing.T) {
	logger := initTestLogger()
	ctx := context.Background()

	hopsFile := "./testdata/valid.hops"
	eventFile := "./testdata/raw_change_event.json"
	responseFile := "./testdata/task_response.json"

	eventData, err := os.ReadFile(eventFile)
	require.NoError(t, err)

	responseData, err := os.ReadFile(responseFile)
	require.NoError(t, err)

	eventBundle := map[string][]byte{
		"event":               eventData,
		"a_sensor-first_task": responseData,
	}

	hclFile, _, err := ReadHopsFiles(hopsFile)
	assert.NoError(t, err)

	hop, err := ParseHops(ctx, hclFile, eventBundle, logger)
	assert.NoError(t, err)

	// Test we parsed the correct number of matching on blocks.
	require.Len(t, hop.Ons, 3)

	// Test the first on block had the correct number of calls
	require.Len(t, hop.Ons[0].Calls, 3)

	// Ensure the slugs align with what we want
	assert.Equal(t, hop.Ons[0].Calls[0].Slug, "a_sensor-first_task")
}

func TestInvalidParse(t *testing.T) {
	hopsFile := "./testdata/invalid.hops"
	eventFile := "./testdata/raw_change_event.json"
	ctx := context.Background()
	logger := initTestLogger()

	eventData, err := os.ReadFile(eventFile)
	require.NoError(t, err)

	eventBundle := map[string][]byte{
		"event": eventData,
	}

	hclFile, _, err := ReadHopsFiles(hopsFile)
	assert.NoError(t, err)

	hop, err := ParseHops(ctx, hclFile, eventBundle, logger)
	assert.Error(t, err)
	assert.Nil(t, hop.Ons)
}

func TestSlugify(t *testing.T) {
	result := slugify("Hello World")
	assert.Equal(t, "hello-world", result)

	result = slugify("on", "Hello World")
	assert.Equal(t, "on-hello-world", result)

	result = slugify("on", "change.opened", "Hello World")
	assert.Equal(t, "on-change-opened-hello-world", result)

	result = slugify("change_opened", "hello_world")
	assert.Equal(t, "change_opened-hello_world", result)
}

func TestConcatenateHopsFiles(t *testing.T) {
	tests := []struct {
		name            string
		files           map[string]string // filename -> content
		expectedContent string
		expectedRows    int
		expectError     bool
	}{
		{
			name:            "NoFiles",
			files:           map[string]string{},
			expectedContent: "",
			expectedRows:    0,
			expectError:     false,
		},
		{
			name: "SingleFile",
			files: map[string]string{
				"a.hops": "content of a",
			},
			expectedContent: "content of a\n",
			expectedRows:    1,
			expectError:     false,
		},
		{
			name: "MultipleFiles",
			files: map[string]string{
				"a.hops": "content of a",
				"b.hops": "content of b",
			},
			expectedContent: "content of a\ncontent of b\n",
			expectedRows:    2,
			expectError:     false,
		},
		{
			name: "FilesSortedByFilename",
			files: map[string]string{
				"b.hops": "content of b",
				"a.hops": "content of a",
			},
			expectedContent: "content of a\ncontent of b\n",
			expectedRows:    2,
			expectError:     false,
		},
		{
			name: "OnlyDotHopsFiles",
			files: map[string]string{
				"a.hops": "content of a",
				"b.txt":  "content of b",
				"c.hops": "content of c",
			},
			expectedContent: "content of a\ncontent of c\n",
			expectedRows:    2,
			expectError:     false,
		},
		{
			name: "SubdirectoriesSearched",
			files: map[string]string{
				"subdir/b.hops": "content of b",
				"a.hops":        "content of a",
			},
			expectedContent: "content of a\ncontent of b\n",
			expectedRows:    2,
			expectError:     false,
		},
		{
			name: "MultipleFilesSubdirsSortedByNameIncludingSubdirs",
			files: map[string]string{
				"a.hops":      "content of a",
				"sub2/b.hops": "content of b",
				"sub1/c.hops": "content of c",
				"sub3/d.hops": "content of d",
			},
			expectedContent: "content of a\ncontent of c\ncontent of b\ncontent of d\n",
			expectedRows:    4,
			expectError:     false,
		},
		{
			name: "DontPickUpSubdirsFromKubernetesConfigMapsAKAWithLeadingDoubleDot",
			files: map[string]string{
				"..a.hops":          "content of a",
				"..sub3/b.hops":     "content of b",
				"sub4/..sub/c.hops": "content of c",
				"d.hops":            "content of d",
			},
			expectedContent: "content of a\ncontent of d\n",
			expectedRows:    2,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// temporary directory
			tmpDir, err := os.MkdirTemp("", "testdir")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %s", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create files in temp dir
			for filename, content := range tt.files {
				tmpFilename := filepath.Join(tmpDir, filename)

				// Create subdirs
				if err := os.MkdirAll(filepath.Dir(tmpFilename), 0755); err != nil {
					t.Fatalf("Failed to create subdirectory for file %s: %s", tmpFilename, err)
				}
				err := os.WriteFile(tmpFilename, []byte(content), 0666)
				if err != nil {
					t.Fatalf("Failed to write to temp file %s: %s", tmpFilename, err)
				}
			}

			// Run the function
			resultFileContent, resultContent, err := concatenateHopsFiles(tmpDir)

			// Check for an unexpected error
			if !tt.expectError {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			// Compare the result with the expected content
			assert.Equal(t, tt.expectedContent, string(resultContent))

			// Compare the result with the expected file content
			assert.Equal(t, tt.expectedRows, len(resultFileContent))
		})
	}
}

func initTestLogger() zerolog.Logger {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	return log.Logger
}
