package dsl

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/gosimple/slug"
	"github.com/hashicorp/hcl/v2"
	"github.com/manterfield/fast-ctyjson/ctyjson"
	"github.com/rs/zerolog"
	"github.com/zclconf/go-cty/cty/gocty"
)

const hopsMetadataKey = "hops"

func ParseHops(ctx context.Context, hops *HopsFiles, eventBundle map[string][]byte, logger zerolog.Logger) (*HopAST, error) {
	hop := &HopAST{
		SlugRegister: make(map[string]bool),
	}

	ctxVariables, err := eventBundleToCty(eventBundle, "-")
	if err != nil {
		return nil, err
	}

	evalctx := &hcl.EvalContext{
		Functions: StatelessFunctions,
		Variables: ctxVariables,
	}

	err = DecodeHopsBody(ctx, hop, hops, evalctx, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to parse hops configs")

		logger.Debug().Msg("Parse failed on pipeline, dumping state:")
		for k, v := range eventBundle {
			logger.Debug().RawJSON(k, v).Msgf("%s message content", k)
		}

		return hop, err
	}

	return hop, nil
}

func DecodeHopsBody(ctx context.Context, hop *HopAST, hops *HopsFiles, evalctx *hcl.EvalContext, logger zerolog.Logger) error {
	onBlocks := hops.BodyContent.Blocks.OfType(OnID)
	for idx, onBlock := range onBlocks {
		hop.Diagnostics = DecodeOnBlock(ctx, hop, hops, onBlock, idx, evalctx, logger)

		if hop.Diagnostics.HasErrors() {
			logDiagnostics(hop.Diagnostics, logger)
			return errors.Join(hop.Diagnostics.Errs()...)
		}
	}

	return nil
}

func DecodeOnBlock(ctx context.Context, hop *HopAST, hops *HopsFiles, block *hcl.Block, idx int, evalctx *hcl.EvalContext, logger zerolog.Logger) hcl.Diagnostics {
	on := &OnAST{}

	bc, d := block.Body.Content(OnSchema)
	if d.HasErrors() {
		return d
	}

	on.EventType = block.Labels[0]
	name, diag := DecodeNameAttr(bc.Attributes[NameAttr])
	if diag.HasErrors() {
		return diag
	}
	// If no name is given, append stringified index of the block
	if name == "" {
		name = fmt.Sprintf("%s%d", on.EventType, idx)
	}

	on.Name = name
	on.Slug = slugify(on.Name)

	err := ValidateLabels(on.EventType, on.Name)
	if err != nil {
		diag := hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagInvalid,
				Summary:  fmt.Sprintf("Invalid label: %s", err.Error()),
				Subject:  &block.LabelRanges[0],
			},
		}
		return diag
	}

	if hop.SlugRegister[on.Slug] {
		diag := hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagInvalid,
				Summary:  fmt.Sprintf("Duplicate named 'on' block found: %s", on.Slug),
				Detail:   "'on' blocks must have unique names across all automations",
				Subject:  &bc.Attributes[NameAttr].Range,
			},
		}
		return diag
	} else {
		hop.SlugRegister[on.Slug] = true
	}

	// TODO: This should be done once outside of the on block and passed in as an argument
	eventType, eventAction, err := parseEventVar(evalctx.Variables)
	if err != nil {
		// Diagnostic being used here is a bit ropey, since an error here won't
		// relate to an issue with user's hops syntax.
		return hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Unable to parse source event: %s", err.Error()),
				Subject:  &block.DefRange,
			},
		}
	}

	blockEventType, blockAction, hasAction := strings.Cut(on.EventType, "_")
	if (blockEventType != eventType) || (hasAction && blockAction != eventAction) {
		// This on block doesn't match the event being processed
		return nil
	}

	evalctx = blockEvalContext(evalctx, hops, block)
	evalctx = scopedEvalContext(evalctx, on.EventType, on.Name)

	ifClause := bc.Attributes[IfAttr]
	val, diag := DecodeConditionalAttr(ifClause, true, evalctx)
	if diag.HasErrors() {
		return diag
	}

	if !val {
		// If condition is not met. Omit the block and stop parsing.
		logger.Debug().Msgf("%s 'if' not met", on.Slug)
		return nil
	}

	on.If = &val

	logger.Info().Msgf("%s matches event", on.Slug)

	// Evaluate done blocks first, as we don't want to dispatch further calls
	// after a pipeline is marked as done
	doneBlocks := bc.Blocks.OfType(DoneID)
	for _, doneBlock := range doneBlocks {
		done, diag := DecodeDoneBlock(ctx, hop, on, doneBlock, evalctx, logger)
		if diag.HasErrors() {
			return diag
		}

		if done != nil {
			// If any done block is not nil, then finish parsing
			on.Done = done
			hop.Ons = append(hop.Ons, *on)
			return nil
		}
	}

	callBlocks := bc.Blocks.OfType(CallID)
	for idx, callBlock := range callBlocks {
		diag := DecodeCallBlock(ctx, hop, on, callBlock, idx, evalctx, logger)
		if diag.HasErrors() {
			return diag
		}
	}

	hop.Ons = append(hop.Ons, *on)
	return nil
}

func DecodeCallBlock(ctx context.Context, hop *HopAST, on *OnAST, block *hcl.Block, idx int, evalctx *hcl.EvalContext, logger zerolog.Logger) hcl.Diagnostics {
	call := &CallAST{}

	bc, diag := block.Body.Content(CallSchema)
	if diag.HasErrors() {
		return diag
	}

	call.ActionType = block.Labels[0]
	name, diag := DecodeNameAttr(bc.Attributes[NameAttr])
	if diag.HasErrors() {
		return diag
	}
	if name == "" {
		name = fmt.Sprintf("%s%d", call.ActionType, idx)
	}

	call.Name = name
	call.Slug = slugify(on.Slug, call.Name)

	err := ValidateLabels(call.ActionType, call.Name)
	if err != nil {
		diag := hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagInvalid,
				Summary:  fmt.Sprintf("Invalid label: %s", err.Error()),
				Subject:  &block.LabelRanges[0],
			},
		}
		return diag
	}

	if hop.SlugRegister[call.Slug] {
		diag := hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagInvalid,
				Summary:  fmt.Sprintf("Duplicate named 'call' block found: %s", call.Slug),
				Detail:   "'call' blocks must have unique names within an 'on' block",
				Subject:  &bc.Attributes[NameAttr].Range,
			},
		}
		return diag
	} else {
		hop.SlugRegister[call.Slug] = true
	}

	ifClause := bc.Attributes[IfAttr]
	val, diag := DecodeConditionalAttr(ifClause, true, evalctx)
	if diag.HasErrors() {
		logger.Debug().Msgf(
			"%s 'if' not ready for evaluation, defaulting to false: %s",
			call.Slug,
			diag.Error(),
		)
	}

	if !val {
		logger.Debug().Msgf("%s 'if' not met", call.Slug)
		return nil
	}

	call.If = &val

	logger.Info().Msgf("%s matches event", call.Slug)

	inputs := bc.Attributes["inputs"]
	if inputs != nil {
		val, diag := inputs.Expr.Value(evalctx)
		if diag.HasErrors() {
			return diag
		}

		jsonVal := ctyjson.SimpleJSONValue{Value: val}
		inputs, err := jsonVal.MarshalJSON()

		if err != nil {
			return hcl.Diagnostics{
				&hcl.Diagnostic{
					Severity: hcl.DiagInvalid,
					Summary:  fmt.Sprintf("Unable to encode inputs as JSON: %s", err.Error()),
					Subject:  &bc.Attributes["inputs"].Range,
				},
			}
		}

		call.Inputs = inputs
	}

	on.Calls = append(on.Calls, *call)
	return nil
}

func DecodeNameAttr(attr *hcl.Attribute) (string, hcl.Diagnostics) {
	if attr == nil {
		// Not an error, as the attribute is not required
		return "", nil
	}

	val, diag := attr.Expr.Value(nil)
	if diag.HasErrors() {
		return "", diag
	}

	var value string

	err := gocty.FromCtyValue(val, &value)
	if err != nil {
		return "", hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  err.Error(),
				Subject:  &attr.NameRange,
			},
		}
	}

	return value, nil
}

func DecodeConditionalAttr(attr *hcl.Attribute, defaultValue bool, ctx *hcl.EvalContext) (bool, hcl.Diagnostics) {
	if attr == nil {
		return defaultValue, nil
	}

	v, diag := attr.Expr.Value(ctx)
	if diag.HasErrors() {
		return false, diag
	}

	var value bool

	err := gocty.FromCtyValue(v, &value)
	if err != nil {
		return false, hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  err.Error(),
				Subject:  &attr.NameRange,
			},
		}
	}

	return value, nil
}

func slugify(parts ...string) string {
	joined := strings.Join(parts, "-")
	return slug.Make(joined)
}

// blockEvalContext adds directory level StatefulFunctions to the eval context
// for a block
//
// Makes sure directory level StatefulFunctions are created with
// the correct state.
func blockEvalContext(evalCtx *hcl.EvalContext, hops *HopsFiles, block *hcl.Block) *hcl.EvalContext {
	// For file() calls, get the directory prefix of the current hops file
	hopsFilename := block.DefRange.Filename
	hopsDir := path.Dir(hopsFilename)

	blockEvalCtx := evalCtx.NewChild()
	blockEvalCtx.Functions = StatefulFunctions(hops, hopsDir)
	blockEvalCtx.Variables = evalCtx.Variables // Not inherited from parent (unlike Functions, which are merged)

	return blockEvalCtx
}

func logDiagnostics(diags hcl.Diagnostics, logger zerolog.Logger) {
	for _, diag := range diags {
		logEvent := logger.Error()

		if diag.Subject.Filename != "" {
			logEvent = logEvent.Interface("range", diag.Subject)
		}

		logEvent.Str("detail", diag.Detail).Msg(diag.Summary)
	}
}

// scopedEvalContext creates eval contexts that are relative to the current scope
//
// This function effectively fakes relative/local variables by checking where
// we are in the hops code (defined by scopePath) and bringing any nested variables matching
// that path to the top level.
func scopedEvalContext(evalCtx *hcl.EvalContext, scopePath ...string) *hcl.EvalContext {
	scopedVars := evalCtx.Variables

	for _, scopeToken := range scopePath {
		if val, ok := scopedVars[scopeToken]; ok {
			scopedVars = val.AsValueMap()
		}
	}

	scopedEvalCtx := evalCtx.NewChild()
	scopedEvalCtx.Variables = scopedVars

	return scopedEvalCtx
}
