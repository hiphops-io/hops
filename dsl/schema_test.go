package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandInputValidation(t *testing.T) {
	command := TaskAST{
		Params: []ParamAST{
			{
				Name:     "a_req_string",
				Type:     "string",
				Required: true,
			},
			{
				Name: "a_string",
				Type: "string",
			},
			{
				Name: "a_text",
				Type: "text",
			},
			{
				Name: "a_number",
				Type: "number",
			},
			{
				// Two numbers so we can test floats and ints in one input
				Name: "a_nother_number",
				Type: "number",
			},
			{
				Name: "a_bool",
				Type: "bool",
			},
		},
	}

	type testCase struct {
		name           string
		input          map[string]any
		expectedErrors map[string][]string
	}

	tests := []testCase{
		{
			name: "Full valid input returns an empty map",
			input: map[string]any{
				"a_req_string":    "Hello",
				"a_string":        "Goodbye",
				"a_text":          "Hello, Goodbye!",
				"a_number":        42,
				"a_nother_number": 3.14159,
				"a_bool":          true,
			},
			expectedErrors: map[string][]string{},
		},
		{
			name: "Missing required field returns error",
			input: map[string]any{
				"a_string": "Hello",
			},
			expectedErrors: map[string][]string{
				"a_req_string": {
					InvalidRequired,
				},
			},
		},
		{
			name: "Incorrect types return errors",
			input: map[string]any{
				"a_string": true,
				"a_text":   123,
				"a_number": "Hello",
				"a_bool":   123,
			},
			expectedErrors: map[string][]string{
				"a_req_string": {
					InvalidRequired,
				},
				"a_string": {
					InvalidNotString,
				},
				"a_text": {
					InvalidNotText,
				},
				"a_number": {
					InvalidNotNumber,
				},
				"a_bool": {
					InvalidNotBool,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			validationErrors := command.ValidateInput(tc.input)
			assert.Equal(
				t,
				tc.expectedErrors,
				validationErrors,
				"Command validation result should match",
			)
		})
	}
}
