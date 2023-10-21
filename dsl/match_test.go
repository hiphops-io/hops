package dsl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

func TestAnyMatch(t *testing.T) {
	tests := []struct {
		values   []cty.Value
		expected cty.Value
	}{
		{
			values: []cty.Value{
				cty.True,
			},
			expected: cty.True,
		},
		{
			values: []cty.Value{
				cty.True,
				cty.False,
			},
			expected: cty.True,
		},
		{
			values: []cty.Value{
				cty.False,
			},
			expected: cty.False,
		},
		{
			values: []cty.Value{
				cty.False,
				cty.False,
			},
			expected: cty.False,
		},
	}

	for _, test := range tests {
		testName := fmt.Sprintf("anymatch(%#v)", test.values)
		t.Run(testName, func(t *testing.T) {
			got, err := AnyMatch(test.values)

			if assert.NoError(t, err) {
				assert.Equal(t, test.expected, got)
			}
		})
	}
}

func TestAllMatch(t *testing.T) {
	tests := []struct {
		values   []cty.Value
		expected cty.Value
	}{
		{
			values: []cty.Value{
				cty.True,
			},
			expected: cty.True,
		},
		{
			values: []cty.Value{
				cty.True,
				cty.False,
			},
			expected: cty.False,
		},
		{
			values: []cty.Value{
				cty.False,
			},
			expected: cty.False,
		},
		{
			values: []cty.Value{
				cty.False,
				cty.False,
			},
			expected: cty.False,
		},
	}

	for _, test := range tests {
		testName := fmt.Sprintf("anymatch(%#v)", test.values)
		t.Run(testName, func(t *testing.T) {
			got, err := AllMatch(test.values)

			if assert.NoError(t, err) {
				assert.Equal(t, test.expected, got)
			}
		})
	}
}
