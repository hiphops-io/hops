package ctyutils

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

type (
	capsulatedType1 struct {
		Field1 string
	}

	capsulatedType2 struct {
		Field2 int
	}

	// testCase for convertCtyValueToInterface
	testCaseConvertCtyValueToInterface struct {
		name     string
		input    cty.Value
		expected interface{}
	}
)

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
			name:     "Empty Object Conversion",
			input:    cty.EmptyObjectVal,
			expected: map[string]interface{}{}, // an empty map
		},
		{
			name: "Map Conversion",
			input: cty.MapVal(map[string]cty.Value{
				"key1": cty.StringVal("value1"),
				"key2": cty.StringVal("value2"),
			}),
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:     "Empty Map Conversion",
			input:    cty.MapValEmpty(cty.String), // An empty map with string values
			expected: map[string]interface{}{},    // Expect an empty Go map
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
			name:     "Empty List Conversion",
			input:    cty.ListValEmpty(cty.String), // An empty list of strings
			expected: []interface{}{},              // Expect an empty slice
		},
		{
			name: "Set Conversion",
			input: cty.SetVal([]cty.Value{
				cty.StringVal("item1"),
				cty.StringVal("item2"),
			}),
			expected: []interface{}{"item1", "item2"}, // Elements expected in the set
		},
		{
			name:     "Empty Set Conversion",
			input:    cty.SetValEmpty(cty.String), // An empty set of strings
			expected: []interface{}{},             // Expect an empty slice
		},
		{
			name:     "Tuple to Array Conversion",
			input:    cty.TupleVal([]cty.Value{cty.StringVal("tupleItem1"), cty.NumberIntVal(789), cty.True}),
			expected: []interface{}{"tupleItem1", 789.0, true},
		},
		{
			name:     "Empty Tuple Conversion",
			input:    cty.EmptyTupleVal,
			expected: []interface{}{}, // an empty slice
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
		{
			name:     "Unknown Value Conversion",
			input:    cty.UnknownVal(cty.String),
			expected: nil, // or whatever representation you expect for unknown values
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ConvertCtyValueToInterface(tc.input)

			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestConvertCtyValueToInterface_CapsuleTypes(t *testing.T) {
	capsulatedType1Capsule := cty.Capsule("capsulatedType1", reflect.TypeOf(capsulatedType1{}))
	capsulatedType2Capsule := cty.Capsule("capsulatedType2", reflect.TypeOf(capsulatedType2{}))

	customValue1 := capsulatedType1{Field1: "Test1"}
	customValue2 := capsulatedType2{Field2: 42}

	capsuleVal1 := cty.CapsuleVal(capsulatedType1Capsule, &customValue1)
	capsuleVal2 := cty.CapsuleVal(capsulatedType2Capsule, &customValue2)

	t.Run("capsule Type 1", func(t *testing.T) {
		result, err := ConvertCtyValueToInterface(capsuleVal1)
		if assert.NoError(t, err) {
			assert.IsType(t, &capsulatedType1{}, result)
			assert.Equal(t, &customValue1, result)
		}
	})

	t.Run("capsule Type 2", func(t *testing.T) {
		result, err := ConvertCtyValueToInterface(capsuleVal2)
		if assert.NoError(t, err) {
			assert.IsType(t, &capsulatedType2{}, result)
			assert.Equal(t, &customValue2, result)
		}
	})
}
