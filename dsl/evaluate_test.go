package dsl

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEvaluateValid is a port of an older test case, kept to ensure equivalent
// parsing with new parser logic.
//
// This should be retired and converted into proper table based testing with
// inline configs as in the other schema/automation/evaluation test suites.
func TestEvaluateValid(t *testing.T) {
	// test that split hops files have identical result as single hops file
	hopsFiles := []string{
		"./testdata/valid",
		"./testdata/multi-hops",
	}

	for _, hopsFile := range hopsFiles {
		eventFile := "./testdata/raw_change_event.json"
		eventData, err := os.ReadFile(eventFile)
		require.NoError(t, err)

		eventBundle := map[string][]byte{
			"event": eventData,
		}

		a, d, err := NewAutomationsFromDir(hopsFile)
		require.NoError(t, err)
		require.Falsef(t, d.HasErrors(), "Hops decoding should have no diagnostic errors, got: %s", d.Error())

		ons, d := a.EventOns(eventBundle)
		require.Falsef(t, d.HasErrors(), "On evaluation should have no diagnostic errors, got: %s", d.Error())

		// Test we parsed the correct number of matching on blocks.
		require.Len(t, ons, 3)

		// Test the first on block had the proper values
		assert.Equal(t, "change_merged", ons[0].Label)
		assert.Equal(t, `a_sensor`, ons[0].Slug)

		// Test the second on block had the proper values
		assert.Equal(t, "change", ons[1].Label)
		assert.Len(t, ons[1].Calls, 0)

		// Test the index named on block had the proper values
		assert.Equal(t, "", ons[2].Name)
		assert.Equal(t, "change2", ons[2].Slug)

		// Now dig into calls in the first on block, checking how they were parsed
		if assert.Len(t, ons[0].Calls, 2) {
			call := ons[0].Calls[0]
			assert.Equal(t, `a_sensor-first_task`, call.Slug)
			assert.JSONEq(t, `{"a":"b", "from_env": "", "source": "GITHUB_COM"}`, string(call.Inputs))

			call = ons[0].Calls[1]
			assert.Equal(t, `a_sensor-index_id_call2`, call.Slug)
		}

		// Ensure the done block is empty
		assert.Nil(t, ons[0].Done)
	}
}

func TestValidParseResponseStep(t *testing.T) {
	hopsFile := "./testdata/valid"
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

	a, d, err := NewAutomationsFromDir(hopsFile)
	require.NoError(t, err)
	require.Falsef(t, d.HasErrors(), "Hops decoding should have no diagnostic errors, got: %s", d.Error())

	ons, d := a.EventOns(eventBundle)
	require.Falsef(t, d.HasErrors(), "On evaluation should have no diagnostic errors, got: %s", d.Error())

	// Test we parsed the correct number of matching on blocks.
	require.Len(t, ons, 3)

	// Test the first on block had the correct number of calls
	require.Len(t, ons[0].Calls, 3)

	// Ensure the slugs align with what we want
	assert.Equal(t, ons[0].Calls[0].Slug, "a_sensor-first_task")

	// Ensure the done block is empty
	assert.Nil(t, ons[0].Done)
}

func TestValidParseDone(t *testing.T) {
	hopsFile := "./testdata/valid"
	eventFile := "./testdata/raw_change_event.json"
	responseFile := "./testdata/task_response.json"

	eventData, err := os.ReadFile(eventFile)
	require.NoError(t, err)

	responseData, err := os.ReadFile(responseFile)
	require.NoError(t, err)

	eventBundle := map[string][]byte{
		"event":               eventData,
		"a_sensor-first_task": responseData,
		"a_sensor-depends":    responseData,
	}

	a, d, err := NewAutomationsFromDir(hopsFile)
	require.NoError(t, err)
	require.Falsef(t, d.HasErrors(), "Hops decoding should have no diagnostic errors, got: %s", d.Error())

	ons, d := a.EventOns(eventBundle)
	require.Falsef(t, d.HasErrors(), "On evaluation should have no diagnostic errors, got: %s", d.Error())

	// Test we parsed the correct number of matching on blocks.
	require.Len(t, ons, 3)

	// Test the first on block had the correct number of calls
	require.Len(t, ons[0].Calls, 0)

	// Ensure the done block is generated and check the values
	require.NotNil(t, ons[0].Done)

	done := ons[0].Done
	assert.True(t, done.Completed)
	assert.False(t, done.Errored)
}

func TestInvalidParse(t *testing.T) {
	hopsFile := "./testdata/invalid"
	eventFile := "./testdata/raw_change_event.json"

	eventData, err := os.ReadFile(eventFile)
	require.NoError(t, err)

	eventBundle := map[string][]byte{
		"event": eventData,
	}

	a, d, err := NewAutomationsFromDir(hopsFile)
	require.NoError(t, err)
	require.True(t, d.HasErrors(), "Invalid hops decoding should have diagnostic errors")

	ons, d := a.EventOns(eventBundle)
	assert.True(t, d.HasErrors(), "Invalid hops should have diagnostic errors")
	assert.Len(t, ons, 1, "One on block should be returned")
}
