package dsl

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// FileFunc does stuff TODO
func FileFunc(hops *HopsFiles) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "filename",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			filenameVal := args[0]
			filename := filenameVal.AsString()

			file, err := File(filename, hops)

			return cty.StringVal(file), err
		},
	})
}

func File(filename string, hops *HopsFiles) (string, error) {
	if filename == "" {
		return "", nil
	}

	return fmt.Sprintf("%s/%#v", filename, hops), nil
}
