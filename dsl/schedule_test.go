package dsl

import (
	"testing"

	"github.com/hiphops-io/hops/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduleParse(t *testing.T) {
	type testCase struct {
		name       string
		hops       string
		schedules  []ScheduleAST
		validParse bool
		validRead  bool
	}

	tests := []testCase{
		// Test that a simple valid schedule is parsed correctly
		{
			name: "Simple valid schedule",
			hops: `schedule foo {
				cron = "@hourly"
			}`,
			schedules: []ScheduleAST{
				{Name: "foo", Cron: "@hourly"},
			},
			validParse: true,
			validRead:  true,
		},

		// Test that a schedule with missing cron fails validations
		{
			name:       "Simple invalid schedule",
			hops:       `schedule foo {}`,
			validParse: false,
			validRead:  true,
		},

		// Test no schedules doesn't throw an error
		{
			name:       "No schedules",
			hops:       `on push {}`,
			schedules:  []ScheduleAST{},
			validParse: true,
			validRead:  true,
		},

		// Test object input is correctly parsed
		{
			name: "Object input",
			hops: `schedule foo {
				cron = "@hourly"
				inputs = {
					a = "a_val"
					b = 2
				}
			}`,
			schedules: []ScheduleAST{
				{Name: "foo", Cron: "@hourly", Inputs: []byte(`{"a":"a_val","b":2}`)},
			},
			validParse: true,
			validRead:  true,
		},

		// Empty cron should throw error
		{
			name: "Empty cron",
			hops: `schedule foo {
				cron = ""
			}`,
			schedules:  []ScheduleAST{},
			validParse: false,
			validRead:  true,
		},

		// Invalid cron should throw error
		{
			name: "Invalid cron",
			hops: `schedule foo {
					cron = "not a * cron * though"
				}`,
			schedules:  []ScheduleAST{},
			validParse: false,
			validRead:  true,
		},
	}

	logger := logs.NoOpLogger()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Ditch early if we're expecting invalid parsing
			hops, err := createTmpHopsFile(tc.hops, t)
			if !tc.validRead {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			hop, err := ParseHopsSchedules(hops, logger)
			if !tc.validParse {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			for i, schedule := range hop.Schedules {
				assert.Equal(t, tc.schedules[i].Name, schedule.Name)
				assert.Equal(t, tc.schedules[i].Cron, schedule.Cron)
				if tc.schedules[i].Inputs != nil {
					assert.JSONEq(t, string(tc.schedules[i].Inputs), string(schedule.Inputs))
				}
			}
		})
	}
}
