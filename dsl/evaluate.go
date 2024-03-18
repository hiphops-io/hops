package dsl

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/manterfield/fast-ctyjson/ctyjson"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hiphops-io/hops/dsl/funcs"
)

const HopsMetaKey = "hops"

type EvaluationCtx struct {
	evalCtx     *hcl.EvalContext
	automations *Automations
}

func NewEvaluationCtx(automations *Automations, variables map[string]cty.Value) *EvaluationCtx {
	if variables == nil {
		variables = make(map[string]cty.Value)
	}

	evalCtx := &hcl.EvalContext{
		Functions: funcs.StatelessFunctions,
		Variables: variables,
	}

	e := &EvaluationCtx{
		evalCtx:     evalCtx,
		automations: automations,
	}

	return e
}

func (e *EvaluationCtx) EvalContext() *hcl.EvalContext {
	return e.evalCtx
}

func (e *EvaluationCtx) BlockScopedEvalContext(block *hcl.Block, slug string) *hcl.EvalContext {
	hopsFilePath := block.DefRange.Filename
	hopsDir := filepath.Dir(hopsFilePath)

	scopedEvalCtx := e.evalCtx.NewChild()
	scopedEvalCtx.Functions = funcs.StatefulFunctions(e.automations.Files, hopsDir)

	if slug == "" {
		scopedEvalCtx.Variables = make(map[string]cty.Value)
		return scopedEvalCtx
	}
	// Hoist nested variables to the top level of the child eval context
	// We still have access to all other variables in the parent context.
	if val, ok := e.evalCtx.Variables[slug]; ok {
		scopedEvalCtx.Variables = val.AsValueMap()
	} else {
		scopedEvalCtx.Variables = make(map[string]cty.Value)
	}

	return scopedEvalCtx
}

func EvaluateGenericExpression[T any](expr hcl.Expression, evalCtx *hcl.EvalContext) (T, bool, hcl.Diagnostics) {
	var result T

	exprVal, d := expr.Value(evalCtx)
	if d.HasErrors() {
		return result, false, d
	}

	if exprVal.IsNull() {
		return result, false, hcl.Diagnostics{}
	}

	err := gocty.FromCtyValue(exprVal, &result)
	if err != nil {
		return result, false, hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid value for type",
				Detail:   fmt.Sprintf("Unable to evaluate value: %s", err.Error()),
				Subject:  expr.Range().Ptr(),
			},
		}
	}

	return result, true, hcl.Diagnostics{}
}

func EvaluateBoolExpression(expr hcl.Expression, unsetVal bool, evalCtx *hcl.EvalContext) (bool, hcl.Diagnostics) {
	// Ifs that aren't set should be considered to match/be true
	if expr == nil {
		return unsetVal, hcl.Diagnostics{}
	}

	v, d := expr.Value(evalCtx)
	if d.HasErrors() {
		return false, hcl.Diagnostics{}
	}

	if v.IsNull() {
		return unsetVal, hcl.Diagnostics{}
	}

	var value bool
	err := gocty.FromCtyValue(v, &value)
	if err != nil {
		return false, hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Must be true, false or unset",
				Detail:   fmt.Sprintf("Value must evaluate to either true or false if set: %s", err.Error()),
				Subject:  expr.Range().Ptr(),
			},
		}
	}

	return value, d
}

func EvaluateInputsExpression(expr hcl.Expression, evalCtx *hcl.EvalContext) ([]byte, hcl.Diagnostics) {
	val, d := expr.Value(evalCtx)
	if d.HasErrors() || val.IsNull() {
		return nil, d
	}

	jsonVal := ctyjson.SimpleJSONValue{Value: val}

	inputs, err := jsonVal.MarshalJSON()
	if err != nil {
		inputs, d = nil, hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unable to evaluate inputs",
				Detail:   fmt.Sprintf("Unable to convert inputs into valid JSON: %s", err.Error()),
				Subject:  expr.Range().Ptr(),
			},
		}
	}

	return inputs, d
}
