package funcs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

// Test glob and xglob work as expected with various combinations of strings and tuples
func TestGlobTypes(t *testing.T) {
	tests := []struct {
		values        cty.Value
		patterns      cty.Value
		expectedGlob  cty.Value
		expectedXGlob cty.Value
	}{
		{
			values:        cty.StringVal("foo"),
			patterns:      cty.StringVal("foo"),
			expectedGlob:  cty.True,
			expectedXGlob: cty.True,
		},
		{
			values: cty.TupleVal([]cty.Value{
				cty.StringVal("foo"),
			}),
			patterns:      cty.StringVal("foo"),
			expectedGlob:  cty.True,
			expectedXGlob: cty.True,
		},
		{
			values: cty.TupleVal([]cty.Value{
				cty.StringVal("foo"),
			}),
			patterns: cty.TupleVal([]cty.Value{
				cty.StringVal("foo"),
			}),
			expectedGlob:  cty.True,
			expectedXGlob: cty.True,
		},
		{
			values:        cty.StringVal("foo"),
			patterns:      cty.StringVal("bar"),
			expectedGlob:  cty.False,
			expectedXGlob: cty.False,
		},
		{
			values: cty.StringVal("foo"),
			patterns: cty.TupleVal([]cty.Value{
				cty.StringVal("bar"),
			}),
			expectedGlob:  cty.False,
			expectedXGlob: cty.False,
		},
		{
			values: cty.TupleVal([]cty.Value{
				cty.StringVal("foo"),
			}),
			patterns: cty.TupleVal([]cty.Value{
				cty.StringVal("bar"),
			}),
			expectedGlob:  cty.False,
			expectedXGlob: cty.False,
		},
	}

	for _, test := range tests {
		testName := fmt.Sprintf("glob(%#v,%#v)", test.values, test.patterns)
		t.Run(testName, func(t *testing.T) {
			got, err := Glob(test.values, test.patterns)

			if assert.NoError(t, err) {
				assert.Equal(t, test.expectedGlob, got)
			}
		})

		testName = fmt.Sprintf("xglob(%#v,%#v)", test.values, test.patterns)
		t.Run(testName, func(t *testing.T) {
			got, err := XGlob(test.values, test.patterns)

			if assert.NoError(t, err) {
				assert.Equal(t, test.expectedXGlob, got)
			}
		})
	}
}

// Test glob pattern matching works as advertised
func TestGlobPatterns(t *testing.T) {
	tests := []struct {
		values        cty.Value
		patterns      cty.Value
		expectedGlob  cty.Value
		expectedXGlob cty.Value
	}{
		{
			values:        cty.StringVal("foo"),
			patterns:      cty.StringVal("*oo"),
			expectedGlob:  cty.True,
			expectedXGlob: cty.True,
		},
		{
			values:        cty.StringVal("foo/bar/buzz.txt"),
			patterns:      cty.StringVal("foo/**/*.txt"),
			expectedGlob:  cty.True,
			expectedXGlob: cty.True,
		},
		{
			values:        cty.StringVal("notfoo/bar/buzz.txt"),
			patterns:      cty.StringVal("foo/**/*.txt"),
			expectedGlob:  cty.False,
			expectedXGlob: cty.False,
		},
		{
			values:        cty.StringVal("foo/bar/buzz.txt"),
			patterns:      cty.StringVal("**/*.txt"),
			expectedGlob:  cty.True,
			expectedXGlob: cty.True,
		},
		{
			values:        cty.StringVal("hello world"),
			patterns:      cty.StringVal("h[e-o]llo world"),
			expectedGlob:  cty.True,
			expectedXGlob: cty.True,
		},
		{
			values: cty.TupleVal([]cty.Value{
				cty.StringVal("foo"),
				cty.StringVal("bar"),
			}),
			patterns: cty.TupleVal([]cty.Value{
				cty.StringVal("bar"),
			}),
			expectedGlob:  cty.True,
			expectedXGlob: cty.False,
		},
		{
			values: cty.TupleVal([]cty.Value{
				cty.StringVal("foo"),
				cty.StringVal("boo"),
			}),
			patterns: cty.TupleVal([]cty.Value{
				cty.StringVal("?oo"),
			}),
			expectedGlob:  cty.True,
			expectedXGlob: cty.True,
		},
		{
			values: cty.TupleVal([]cty.Value{
				cty.StringVal("dir/docs/readme.md"),
				cty.StringVal("dir/contributing.md"),
			}),
			patterns: cty.TupleVal([]cty.Value{
				cty.StringVal("**/*.md"),
			}),
			expectedGlob:  cty.True,
			expectedXGlob: cty.True,
		},
		{
			values: cty.TupleVal([]cty.Value{
				cty.StringVal("dir/docs/readme.md"),
				cty.StringVal("dir/src/config.json"),
			}),
			patterns: cty.TupleVal([]cty.Value{
				cty.StringVal("**/*.md"),
			}),
			expectedGlob:  cty.True,
			expectedXGlob: cty.False,
		},
	}

	for _, test := range tests {
		testName := fmt.Sprintf("glob(%#v,%#v)", test.values, test.patterns)
		t.Run(testName, func(t *testing.T) {
			got, err := Glob(test.values, test.patterns)

			if assert.NoError(t, err) {
				assert.Equal(t, test.expectedGlob, got)
			}
		})

		testName = fmt.Sprintf("xglob(%#v,%#v)", test.values, test.patterns)
		t.Run(testName, func(t *testing.T) {
			got, err := XGlob(test.values, test.patterns)

			if assert.NoError(t, err) {
				assert.Equal(t, test.expectedXGlob, got)
			}
		})
	}
}
