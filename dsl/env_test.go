package dsl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

func TestEnvFunc(t *testing.T) {
	type testCase struct {
		name         string
		envVarName   string
		defaultValue string
		withEnv      map[string]string
		expected     string
	}

	tests := []testCase{
		{
			name:         "Existing env with no default",
			envVarName:   "HIPHOPS_TEST",
			defaultValue: "",
			withEnv: map[string]string{
				"HIPHOPS_TEST": "ONE",
			},
			expected: "ONE",
		},
		{
			name:         "Existing env with default",
			envVarName:   "HIPHOPS_TEST",
			defaultValue: "TWO",
			withEnv: map[string]string{
				"HIPHOPS_TEST": "ONE",
			},
			expected: "ONE",
		},
		{
			name:         "Missing env with default",
			envVarName:   "HIPHOPS_TEST",
			defaultValue: "TWO",
			withEnv: map[string]string{
				"HIPHOPS_OTHER_VALUE": "Hello there",
			},
			expected: "TWO",
		},
		{
			name:         "Missing env with no default",
			envVarName:   "HIPHOPS_TEST",
			defaultValue: "",
			withEnv:      map[string]string{},
			expected:     "",
		},
	}

	for _, tc := range tests {
		testName := fmt.Sprintf(`%s env("%s", "%s")`, tc.name, tc.envVarName, tc.defaultValue)
		t.Run(testName, func(t *testing.T) {
			// Create the env vars
			for k, v := range tc.withEnv {
				t.Setenv(k, v)
			}

			nameVal := cty.StringVal(tc.envVarName)
			defaultVal := cty.StringVal(tc.defaultValue)
			expectedVal := cty.StringVal(tc.expected)

			got, err := Env(nameVal, defaultVal)

			assert.NoError(t, err, "Env function should not throw an error")
			assert.Equal(t, expectedVal, got, "Env function should return correct value")
		})
	}
}
