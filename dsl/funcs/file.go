package funcs

import (
	"fmt"
	"path"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// FileFunc is a stateful cty function that returns the contents of a file
// relative to the current .hops file. The data of the file is determined
// at startup time.
//
// It is stateful because it requires the HopsFiles struct and the hopsDirectory
// to be passed in.
func FileFunc(files map[string][]byte, hopsDirectory string) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "filename",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		// Closure over hops and hopsDirectory
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			filenameVal := args[0]
			filename := filenameVal.AsString()

			file, err := File(hopsDirectory, filename, files)

			return cty.StringVal(file), err
		},
	})
}

// File returns the content of a file from the HopsFiles struct.
//
// Default file path is the directory that is passed in.
func File(directory, filename string, files map[string][]byte) (string, error) {
	if filename == "" {
		return "", nil
	}

	filePath := path.Join(directory, filename)

	if content, ok := files[filePath]; ok {
		return string(content), nil
	}

	return "", fmt.Errorf("File %s not found", filePath)
}
