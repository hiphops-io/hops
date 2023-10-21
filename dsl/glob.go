// Implements the glob functions for use in HCL expressions.
//
// Glob match syntax is:
// pattern:
//
//	{ term }
//
// term:
//
//	 '**'				matches any sequence of characters (including across /).
//								must be a path component by itself. (e.g. "a/**/b" matches "a/x/b", "a/x/y/b", "a/b", but not "ab").
//		'*'         matches any sequence of non-/ characters
//		'?'         matches any single non-/ character
//		'[' [ '^' ] { character-range } ']'
//		            character class (must be non-empty)
//		c           matches character c (c != '*', '?', '\\', '[')
//		'\\' c      matches character c
//
// character-range:
//
//	c           matches character c (c != '\\', '-', ']')
//	'\\' c      matches character c
//	lo '-' hi   matches character c for lo <= c <= hi
package dsl

import (
	"github.com/bmatcuk/doublestar/v4"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// GlobFunc is a cty.Function that matches a string or list of strings against
// a glob pattern or list of glob patterns. Returns true if _any_ values match
// at least one pattern.
var GlobFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "patterns",
			Type: cty.DynamicPseudoType,
		},
		{
			Name: "values",
			Type: cty.DynamicPseudoType,
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		values := args[0]
		patterns := args[1]
		return Glob(values, patterns)
	},
})

func Glob(values, patterns cty.Value) (cty.Value, error) {
	if patterns.IsNull() || values.IsNull() {
		return cty.False, nil
	}

	// Ensure both values and patterns are tuples, so we can have one code path
	if !patterns.Type().IsTupleType() {
		patterns = cty.TupleVal([]cty.Value{patterns})
	}
	if !values.Type().IsTupleType() {
		values = cty.TupleVal([]cty.Value{values})
	}

	return listMatcher(patterns, values)
}

// A cty.Function that matches a string or list of strings against
// a glob pattern or list of glob patterns. Returns true if _all_ values match
// at least one pattern.
var ExclusiveGlobFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "patterns",
			Type: cty.DynamicPseudoType,
		},
		{
			Name: "values",
			Type: cty.DynamicPseudoType,
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		values := args[0]
		patterns := args[1]
		return XGlob(values, patterns)
	},
})

func XGlob(values, patterns cty.Value) (cty.Value, error) {
	if patterns.IsNull() || values.IsNull() {
		return cty.False, nil
	}

	// Coerce to tuple if not already to simplify logic below
	if !patterns.Type().IsTupleType() {
		patterns = cty.TupleVal([]cty.Value{patterns})
	}
	if !values.Type().IsTupleType() {
		values = cty.TupleVal([]cty.Value{values})
	}

	return exclusiveListMatcher(patterns, values)
}

// Note: listMatcher and exclusiveListMatcher are almost identical, but I'm not
// sure how to DRY them up without sacrificing readability. Suggestions welcome.
func listMatcher(patterns, values cty.Value) (cty.Value, error) {
	for it := patterns.ElementIterator(); it.Next(); {
		_, pattern := it.Element()
		for it := values.ElementIterator(); it.Next(); {
			_, value := it.Element()
			matches, err := matcherSingle(pattern, value)
			if err != nil {
				return cty.False, err
			}
			if matches.True() {
				return cty.True, nil
			}
		}
	}
	return cty.False, nil
}

func exclusiveListMatcher(patterns, values cty.Value) (cty.Value, error) {
	for it := patterns.ElementIterator(); it.Next(); {
		_, pattern := it.Element()
		for it := values.ElementIterator(); it.Next(); {
			_, value := it.Element()
			matches, err := matcherSingle(pattern, value)
			if err != nil {
				return cty.False, err
			}
			if matches.False() {
				return cty.False, nil
			}
		}
	}
	return cty.True, nil
}

func matcherSingle(pattern, value cty.Value) (cty.Value, error) {
	patternStr := pattern.AsString()
	valueStr := value.AsString()

	if patternStr == "" || valueStr == "" {
		return cty.False, nil
	}

	matches, err := doublestar.Match(patternStr, valueStr)
	if err != nil {
		return cty.False, err
	}

	if matches {
		return cty.True, nil
	}

	return cty.False, nil
}
