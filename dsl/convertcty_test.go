package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

// testCase for convertCtyValueToInterface
type testCaseConvertCtyValueToInterface struct {
	name     string
	input    cty.Value
	expected interface{}
}

// TestConvertCtyValueToInterface tests convertCtyValueToInterface and
// full functionality of conversion from cty.Value to interface{}.
func TestConvertCtyValueToInterface(t *testing.T) {
	testCases := []testCaseConvertCtyValueToInterface{
		{
			name:     "String Conversion",
			input:    cty.StringVal("hello world"),
			expected: "hello world",
		},
		{
			name:     "Number Conversion",
			input:    cty.NumberIntVal(42),
			expected: 42.0,
		},
		{
			name:     "Bool Conversion",
			input:    cty.True,
			expected: true,
		},
		{
			name:     "Null Conversion",
			input:    cty.NullVal(cty.String),
			expected: nil,
		},
		{
			name: "Object to Map Conversion",
			input: cty.ObjectVal(map[string]cty.Value{
				"key1": cty.StringVal("value1"),
				"key2": cty.NumberIntVal(123),
			}),
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": 123.0,
			},
		},
		{
			name: "List to Array Conversion",
			input: cty.ListVal([]cty.Value{
				cty.StringVal("item1"),
				cty.StringVal("item2"),
			}),
			expected: []interface{}{"item1", "item2"},
		},
		{
			name:     "Tuple to Array Conversion",
			input:    cty.TupleVal([]cty.Value{cty.StringVal("tupleItem1"), cty.NumberIntVal(789), cty.True}),
			expected: []interface{}{"tupleItem1", 789.0, true},
		},
		{
			name: "Nested Object Conversion",
			input: cty.ObjectVal(map[string]cty.Value{
				"outerKey1": cty.StringVal("outerValue1"),
				"outerKey2": cty.ObjectVal(map[string]cty.Value{
					"innerKey1": cty.StringVal("innerValue1"),
					"innerKey2": cty.NumberIntVal(456),
				}),
				"outerKey3": cty.TupleVal([]cty.Value{
					cty.StringVal("tupleItem1"),
					cty.NumberIntVal(789),
				}),
			}),
			expected: map[string]interface{}{
				"outerKey1": "outerValue1",
				"outerKey2": map[string]interface{}{
					"innerKey1": "innerValue1",
					"innerKey2": 456.0,
				},
				"outerKey3": []interface{}{"tupleItem1", 789.0},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := convertCtyValueToInterface(tc.input)

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}
