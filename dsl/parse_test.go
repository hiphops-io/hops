package dsl

import (
	"context"
	"os"
	"testing"

	"github.com/hiphops-io/hops/internal/hopsfile"
	"github.com/hiphops-io/hops/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidParse(t *testing.T) {
	logger := logs.NoOpLogger()
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

		hopsFiles, err := hopsfile.ReadHopsFiles(hopsFile)
		assert.NoError(t, err)

		hop, err := ParseHops(ctx, hopsFiles.Body, eventBundle, logger)
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
	logger := logs.NoOpLogger()
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

	hopsFiles, err := hopsfile.ReadHopsFiles(hopsFile)
	assert.NoError(t, err)

	hop, err := ParseHops(ctx, hopsFiles.Body, eventBundle, logger)
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
	logger := logs.NoOpLogger()

	eventData, err := os.ReadFile(eventFile)
	require.NoError(t, err)

	eventBundle := map[string][]byte{
		"event": eventData,
	}

	hopsFiles, err := hopsfile.ReadHopsFiles(hopsFile)
	assert.NoError(t, err)

	hop, err := ParseHops(ctx, hopsFiles.Body, eventBundle, logger)
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
