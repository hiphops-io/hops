package funcs

import (
	"fmt"
	"io"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/valyala/fasttemplate"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// VersionTemplateFunc generates a calver or pet version according to a template
var VersionTemplateFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "template",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		templateVal := args[0]
		template := templateVal.AsString()

		version, err := TemplateVersion(template)

		return cty.StringVal(version), err
	},
})

func TemplateVersion(template string) (string, error) {
	t, err := fasttemplate.NewTemplate(template, "[", "]")
	if err != nil {
		return "", fmt.Errorf("Invalid version template: %w", err)
	}

	version := t.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		// Technically these two prep functions are only needed
		// if they're included in the template, but they're so cheap to run we
		// may as well keep the logic super simple.
		petname.NonDeterministicMode()
		now := time.Now()

		switch tag {
		case "pet":
			pet := petname.Name()
			return w.Write([]byte(pet))
		case "adj":
			adj := petname.Adjective()
			return w.Write([]byte(adj))
		case "adv":
			adverb := petname.Adverb()
			return w.Write([]byte(adverb))
		case "calver":
			val := now.Format("2006.01.02")
			return w.Write([]byte(val))
		case "yyyy":
			val := now.Format("2006")
			return w.Write([]byte(val))
		case "yy":
			val := now.Format("06")
			return w.Write([]byte(val))
		case "mm":
			val := now.Format("01")
			return w.Write([]byte(val))
		case "m":
			val := now.Format("1")
			return w.Write([]byte(val))
		case "dd":
			val := now.Format("02")
			return w.Write([]byte(val))
		case "d":
			val := now.Format("2")
			return w.Write([]byte(val))

		default:
			// Unknown template variables are returned as-is
			val := fmt.Sprintf("[%s]", tag)
			return w.Write([]byte(val))
		}
	})

	return version, nil
}
