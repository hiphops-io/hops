package funcs

import (
	"fmt"

	"github.com/flosch/pongo2/v6"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/hiphops-io/hops/dsl/ctyconv"
)

// TemplateFunc is a stateful cty function that evaluates a file and variables
// using the pongo2 library, which matches the Django template language. It
// returns the results as a string. It finds the file relative to the current
// .hops file. The data of the file is determined at startup time.
//
// It is stateful because it requires the HopsFiles struct and the hopsDirectory
// to be passed in.
func TemplateFunc(files map[string][]byte, hopsDirectory string) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "filename",
				Type: cty.String,
			},
			{
				Name: "variables",
				Type: cty.DynamicPseudoType,
			},
		},
		Type: function.StaticReturnType(cty.String),
		// Closure over hops and hopsDirectory
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			filenameVal := args[0]
			filename := filenameVal.AsString()
			variablesVal := args[1]

			ctyVariables, err := ctyconv.CtyValueToInterface(variablesVal)
			if err != nil {
				return cty.Value{}, err
			}

			variables, ok := ctyVariables.(map[string]interface{})
			if !ok {
				return cty.Value{}, fmt.Errorf("Variables must be key value pairs")
			}

			file, err := Template(hopsDirectory, filename, files, variables)
			return cty.StringVal(file), err
		},
	})
}

// Template returns the evaluated template content of a file from the HopsFiles
// struct and the passed in variables. Handles special case where "autoescape"
// is desired.
//
// Default file path is the directory that is passed in.
// If "autoescape" is true, then the template is wrapped in autoescape tags
// which protects against dangerous HTML inputs in variables.
func Template(directory, filename string, files map[string][]byte, variables map[string]any) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("Filename must be provided")
	}

	fileContent, err := File(directory, filename, files)
	if err != nil {
		return "", err
	}

	return runTemplate(fileContent, variables)
}

func runTemplate(fileContent string, variables map[string]any) (string, error) {
	// by default, handle all text
	pongo2.SetAutoescape(false)

	// Can't change global setting for threadsafety reasons so wrap file contents
	// in autoescape tags instead
	if variables["autoescape"] == true {
		fileContent = "{% autoescape on %}" + fileContent + "{% endautoescape %}"
	}

	tpl, err := pongo2.FromString(fileContent)
	if err != nil {
		return "", err
	}

	result, err := tpl.Execute(pongo2.Context(variables))
	if err != nil {
		return "", err
	}

	return result, nil
}
