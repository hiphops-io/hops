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
	}

	for _, hopsFile := range hopsFiles {
		eventFile := "./testdata/raw_change_event.json"
		eventData, err := os.ReadFile(eventFile)
		require.NoError(t, err)

		a, d, err := NewAutomationsFromDir(hopsFile)
		require.NoError(t, err)
		require.Falsef(t, d.HasErrors(), "Hops decoding should have no diagnostic errors, got: %s", d.Error())

		ons, d := a.EventOns(eventData)
		require.Falsef(t, d.HasErrors(), "On evaluation should have no diagnostic errors, got: %s", d.Error())

		// Test we parsed the correct number of matching on blocks.
		require.Len(t, ons, 3)

		// Test the first on block had the proper values
		assert.Equal(t, "change_merged", ons[0].Label)
		assert.Equal(t, "change_merged-a_sensor", ons[0].Slug)
		assert.Equal(t, "valid.worker", ons[0].Worker)

		// Test the second on block had the proper values
		assert.Equal(t, "change", ons[1].Label)
		assert.Equal(t, "valid.worker", ons[1].Worker)
		// assert.Len(t, ons[1].Calls, 0)

		// Test the index named on block had the proper values
		assert.Equal(t, "do_change", ons[2].Name)
		assert.Equal(t, "change-do_change", ons[2].Slug)
	}
}
