package dsl

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/rs/zerolog"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func ParseHopsTasks(ctx context.Context, hopsFiles HclFiles) (*HopAST, error) {
	logger := zerolog.Ctx(ctx)

	hop := &HopAST{
		SlugRegister: make(map[string]bool),
	}

	evalctx := &hcl.EvalContext{
		Functions: DefaultFunctions,
	}

	for _, hopsFile := range hopsFiles {
		err := DecodeTasks(ctx, hop, hopsFile.Body, evalctx)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to decode hops tasks")
			return hop, err
		}
	}

	return hop, nil
}

func DecodeTasks(ctx context.Context, hop *HopAST, body hcl.Body, evalctx *hcl.EvalContext) error {
	bc, d := body.Content(HopSchema)
	if d.HasErrors() {
		return d.Errs()[0]
	}

	if len(bc.Blocks) == 0 {
		return errors.New("At least one resource must be defined")
	}

	blocks := bc.Blocks.OfType(TaskID)
	for _, block := range blocks {
		err := DecodeTaskBlock(ctx, hop, block, evalctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func DecodeTaskBlock(ctx context.Context, hop *HopAST, block *hcl.Block, evalctx *hcl.EvalContext) error {
	task := TaskAST{}

	content, diag := block.Body.Content(taskSchema)
	if diag.HasErrors() {
		return errors.New(diag.Error())
	}

	task.Name = block.Labels[0]
	err := decodeTaskStrAttribute(content, evalctx, "display_name", &task.DisplayName)
	if err != nil {
		return err
	}
	if task.DisplayName == "" {
		task.DisplayName = titleCase(task.Name)
	}

	slug := slugify(task.Name)

	if hop.SlugRegister[slug] {
		return fmt.Errorf("Duplicate task name found: %s", slug)
	} else {
		hop.SlugRegister[slug] = true
	}

	err = decodeTaskStrAttribute(content, evalctx, "summary", &task.Summary)
	if err != nil {
		return err
	}

	err = decodeTaskStrAttribute(content, evalctx, "description", &task.Description)
	if err != nil {
		return err
	}

	err = decodeTaskStrAttribute(content, evalctx, "emoji", &task.Emoji)
	if err != nil {
		return err
	}

	blocks := content.Blocks.OfType(ParamID)
	for _, block := range blocks {
		err := DecodeParamBlock(block, &task, hop, evalctx)
		if err != nil {
			return err
		}
	}

	hop.Tasks = append(hop.Tasks, task)
	return nil
}

func decodeTaskStrAttribute(content *hcl.BodyContent, evalctx *hcl.EvalContext, attrName string, target *string) error {
	attr := content.Attributes[attrName]

	if attr != nil {
		val, diag := attr.Expr.Value(evalctx)
		if diag.HasErrors() {
			return errors.New(diag.Error())
		}
		*target = val.AsString()
	}

	return nil
}

func DecodeParamBlock(block *hcl.Block, task *TaskAST, hop *HopAST, evalctx *hcl.EvalContext) error {
	param := ParamAST{}

	diag := gohcl.DecodeBody(block.Body, evalctx, &param)
	if diag.HasErrors() {
		return errors.New(diag.Error())
	}

	param.Name = block.Labels[0]

	if param.DisplayName == "" {
		param.DisplayName = titleCase(param.Name)
	}

	slug := slugify(param.Name)

	if hop.SlugRegister[slug] {
		return fmt.Errorf("Duplicate param name found: %s", slug)
	} else {
		hop.SlugRegister[slug] = true
	}

	if param.Flag == "" {
		param.Flag = fmt.Sprintf("--%s", param.Name)
	}

	if param.Type == "" {
		param.Type = "string"
	}

	// I'm not happy with the repetition when converting values to the correct type
	// but ran out of time so will need to polish later. Improvements welcome! @manterfield
	switch param.Type {
	case "string", "text":
		defaultVal, ok, err := attrAsType[string](param.Default, evalctx)
		if err != nil {
			return err
		}
		if !ok {
			param.Default = nil
			break
		}

		param.Default = defaultVal

	case "number":
		defaultVal, ok, err := attrAsType[float64](param.Default, evalctx)
		if err != nil {
			return err
		}
		if !ok {
			param.Default = nil
			break
		}
		param.Default = defaultVal

	case "bool":
		defaultVal, ok, err := attrAsType[bool](param.Default, evalctx)
		if err != nil {
			return err
		}
		if !ok {
			param.Default = nil
			break
		}
		param.Default = defaultVal

	default:
		return fmt.Errorf("Unknown type for param %s", param.Name)
	}

	task.Params = append(task.Params, param)

	return nil
}

// attrAsType takes a value that may be an hcl.Attribute, and returns it as the
// specific type.
//
// The boolean return value describes whether the attribute was actually set,
// or if we're just returning the zero value for that type.
func attrAsType[T any](val any, evalctx *hcl.EvalContext) (T, bool, error) {
	var ctyVal cty.Value
	var result T

	// Assert as attribute and then get the value if possible
	valAttr, ok := val.(*hcl.Attribute)
	if ok {
		exprVal, diag := valAttr.Expr.Value(evalctx)

		if diag.HasErrors() {
			return result, false, errors.New(diag.Error())
		}
		ctyVal = exprVal
	}

	if ctyVal.IsNull() {
		return result, false, nil
	}

	err := gocty.FromCtyValue(ctyVal, &result)
	return result, true, err
}

func titleCase(label string) string {
	caser := cases.Title(language.BritishEnglish)
	label = strings.ReplaceAll(label, "_", " ")
	return caser.String(label)
}
