package dsl

import (
	"path"
	"sort"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// FileFunc does stuff TODO
func FileFunc(hops *HopsFiles, hopsDirectory string) function.Function {
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

			file, err := File(hopsDirectory, filename, hops)

			return cty.StringVal(file), err
		},
	})
}

func File(directory, filename string, hops *HopsFiles) (string, error) {
	if filename == "" {
		return "", nil
	}

	filePath := path.Join(directory, filename)

	// Binary search since filePaths are sorted
	i := sort.Search(len(hops.Files), func(i int) bool {
		return hops.Files[i].File >= filePath
	})
	if i < len(hops.Files) && hops.Files[i].File == filePath {
		return string(hops.Files[i].Content), nil
	}

	return "", nil
}
