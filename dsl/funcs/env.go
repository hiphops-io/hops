package funcs

import (
	"os"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// EnvFunc is a cty.Function that returns an env var or the default value if it doesn't exist
var EnvFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "envVarName",
			Type: cty.String,
		},
		{
			Name: "defaultValue",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		envVarName := args[0]
		defaultValue := args[1]
		return Env(envVarName, defaultValue)
	},
})

func Env(envVarName, defaultValue cty.Value) (cty.Value, error) {
	val, ok := os.LookupEnv(envVarName.AsString())
	if !ok {
		val = defaultValue.AsString()
	}

	ctyVal := cty.StringVal(val)

	return ctyVal, nil
}
