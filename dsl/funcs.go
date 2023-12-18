package dsl

import (
	"github.com/hashicorp/hcl/v2/ext/tryfunc"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

// TODO: Add encode/decode b64
var defaultFunctions = map[string]function.Function{
	"abs":             stdlib.AbsoluteFunc,
	"alltrue":         AllTrueFunc,
	"anytrue":         AnyTrueFunc,
	"versiontmpl":     VersionTemplateFunc,
	"can":             tryfunc.CanFunc,
	"ceil":            stdlib.CeilFunc,
	"chomp":           stdlib.ChompFunc,
	"coalesce":        stdlib.CoalesceFunc,
	"compact":         stdlib.CompactFunc,
	"concat":          stdlib.ConcatFunc,
	"csv":             stdlib.CSVDecodeFunc,
	"env":             EnvFunc,
	"flatten":         stdlib.FlattenFunc,
	"floor":           stdlib.FloorFunc,
	"format":          stdlib.FormatFunc,
	"formatdate":      stdlib.FormatDateFunc,
	"glob":            GlobFunc,
	"indent":          stdlib.IndentFunc,
	"index":           stdlib.IndexFunc,
	"int":             stdlib.IntFunc,
	"join":            stdlib.JoinFunc,
	"jsondecode":      stdlib.JSONDecodeFunc,
	"jsonencode":      stdlib.JSONEncodeFunc,
	"keys":            stdlib.KeysFunc,
	"length":          stdlib.LengthFunc,
	"lookup":          stdlib.LookupFunc,
	"lower":           stdlib.LowerFunc,
	"max":             stdlib.MaxFunc,
	"merge":           stdlib.MergeFunc,
	"min":             stdlib.MinFunc,
	"range":           stdlib.RangeFunc,
	"regex":           stdlib.RegexAllFunc,
	"regexreplace":    stdlib.RegexReplaceFunc,
	"replace":         stdlib.ReplaceFunc,
	"reverse":         stdlib.ReverseFunc,
	"setintersection": stdlib.SetIntersectionFunc,
	"setproduct":      stdlib.SetProductFunc,
	"setunion":        stdlib.SetUnionFunc,
	"slice":           stdlib.SliceFunc,
	"sort":            stdlib.SortFunc,
	"split":           stdlib.SplitFunc,
	"strlen":          stdlib.StrlenFunc,
	"substr":          stdlib.SubstrFunc,
	"timeadd":         stdlib.TimeAddFunc,
	"title":           stdlib.TitleFunc,
	"tobool":          stdlib.MakeToFunc(cty.Bool),
	"tolist":          stdlib.MakeToFunc(cty.List(cty.DynamicPseudoType)),
	"tomap":           stdlib.MakeToFunc(cty.Map(cty.DynamicPseudoType)),
	"tonumber":        stdlib.MakeToFunc(cty.Number),
	"toset":           stdlib.MakeToFunc(cty.Set(cty.DynamicPseudoType)),
	"tostring":        stdlib.MakeToFunc(cty.String),
	"trim":            stdlib.TrimFunc,
	"trimprefix":      stdlib.TrimPrefixFunc,
	"trimspace":       stdlib.TrimSpaceFunc,
	"trimsuffix":      stdlib.TrimSuffixFunc,
	"try":             tryfunc.TryFunc,
	"upper":           stdlib.UpperFunc,
	"values":          stdlib.ValuesFunc,
	"xglob":           ExclusiveGlobFunc,
	"zipmap":          stdlib.ZipmapFunc,
}

// DefaultFunctions returns a map of all the default functions including
// stateful functions
//
// This is necessary to avoid race conditions when using the same function
// map in multiple goroutines.
func DefaultFunctions(hops *HopsFiles, hopsDirectory string) map[string]function.Function {
	functionsCopy := make(map[string]function.Function, len(defaultFunctions))
	for k, v := range defaultFunctions {
		functionsCopy[k] = v
	}

	// OK to overwrite because this will happen every time
	functionsCopy["file"] = FileFunc(hops, hopsDirectory)

	return functionsCopy
}
