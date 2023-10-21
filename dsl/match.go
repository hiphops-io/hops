package dsl

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

/*
allmatch() takes a variable number of bool arguments and returns true if all of them are true.
*/
var AllMatchFunc = function.New(&function.Spec{
	VarParam: &function.Parameter{
		Name: "clauses",
		Type: cty.Bool,
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return AllMatch(args)
	},
})

func AllMatch(values []cty.Value) (cty.Value, error) {
	if len(values) == 0 {
		return cty.False, nil
	}

	for _, clause := range values {
		if !clause.True() {
			return cty.False, nil
		}
	}

	return cty.True, nil
}

/*
anymatch() takes a variable number of bool arguments and returns true if any of them are true.
*/
var AnyMatchFunc = function.New(&function.Spec{
	VarParam: &function.Parameter{
		Name: "clauses",
		Type: cty.Bool,
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		if len(args) == 0 {
			return cty.False, nil
		}

		for _, clause := range args {
			if clause.True() {
				return cty.True, nil
			}
		}

		return cty.False, nil
	},
})

func AnyMatch(values []cty.Value) (cty.Value, error) {
	if len(values) == 0 {
		return cty.False, nil
	}

	for _, clause := range values {
		if clause.True() {
			return cty.True, nil
		}
	}

	return cty.False, nil
}
