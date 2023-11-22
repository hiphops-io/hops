package dsl

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gosimple/slug"
	"github.com/hashicorp/hcl/v2"
	"github.com/manterfield/fast-ctyjson/ctyjson"
	"github.com/rs/zerolog"
	"github.com/zclconf/go-cty/cty/gocty"
)

const hopsMetadataKey = "hops"

func ParseHops(ctx context.Context, hopsContent *hcl.BodyContent, eventBundle map[string][]byte, logger zerolog.Logger) (*HopAST, error) {
	hop := &HopAST{
		SlugRegister: make(map[string]bool),
	}

	ctxVariables, err := eventBundleToCty(eventBundle, "-")
	if err != nil {
		return nil, err
	}

	evalctx := &hcl.EvalContext{
		Functions: DefaultFunctions,
		Variables: ctxVariables,
	}

	err = DecodeHopsBody(ctx, hop, hopsContent, evalctx, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to decode hops file")

		logger.Debug().Msg("Parse failed on pipeline, dumping state:")
		for k, v := range eventBundle {
			logger.Debug().RawJSON(k, v).Msgf("%s message content", k)
		}

		return hop, err
	}

	return hop, nil
}

func DecodeHopsBody(ctx context.Context, hop *HopAST, hopsContent *hcl.BodyContent, evalctx *hcl.EvalContext, logger zerolog.Logger) error {
	onBlocks := hopsContent.Blocks.OfType(OnID)
	for idx, onBlock := range onBlocks {
		err := DecodeOnBlock(ctx, hop, onBlock, idx, evalctx, logger)
		if err != nil {
			return err
		}
	}

	return nil
}

func DecodeOnBlock(ctx context.Context, hop *HopAST, block *hcl.Block, idx int, evalctx *hcl.EvalContext, logger zerolog.Logger) error {
	on := &OnAST{}

	bc, d := block.Body.Content(OnSchema)
	if d.HasErrors() {
		return errors.New(d.Error())
	}

	on.EventType = block.Labels[0]
	name, err := DecodeNameAttr(bc.Attributes[NameAttr])
	if err != nil {
		return err
	}
	// If no name is given, append stringified index of the block
	if name == "" {
		name = fmt.Sprintf("%s%d", on.EventType, idx)
	}

	on.Name = name
	on.Slug = slugify(on.Name)

	err = ValidateLabels(on.EventType, on.Name)
	if err != nil {
		return err
	}

	if hop.SlugRegister[on.Slug] {
		return fmt.Errorf("Duplicate 'on' block found: %s", on.Slug)
	} else {
		hop.SlugRegister[on.Slug] = true
	}

	// TODO: This should be done once outside of the on block and passed in as an argument
	eventType, eventAction, err := parseEventVar(evalctx.Variables)
	if err != nil {
		return err
	}

	blockEventType, blockAction, hasAction := strings.Cut(on.EventType, "_")
	if blockEventType != eventType {
		logger.Debug().Msgf("%s does not match event type %s", on.Slug, eventType)
		return nil
	}
	if hasAction && blockAction != eventAction {
		logger.Debug().Msgf("%s does not match event action %s", on.Slug, eventAction)
		return nil
	}

	evalctx = scopedEvalContext(evalctx, on.EventType, on.Name)

	ifClause := bc.Attributes[IfAttr]
	val, err := DecodeConditionalAttr(ifClause, true, evalctx)
	if err != nil {
		return err
	}

	// If condition is not met. Omit the block and stop parsing.
	if !val {
		logger.Debug().Msgf("%s 'if' not met", on.Slug)
		return nil
	}

	on.IfClause = val

	logger.Info().Msgf("%s matches event", on.Slug)

	// Evaluate done blocks first, as we don't want to dispatch further calls
	// after a pipeline is marked as done
	doneBlocks := bc.Blocks.OfType(DoneID)
	for _, doneBlock := range doneBlocks {
		done, err := DecodeDoneBlock(ctx, hop, on, doneBlock, evalctx, logger)
		if err != nil {
			return err
		}
		// If any done block is not nil, then finish parsing
		if done != nil {
			on.Done = done
			hop.Ons = append(hop.Ons, *on)
			return nil
		}
	}

	callBlocks := bc.Blocks.OfType(CallID)
	for idx, callBlock := range callBlocks {
		err := DecodeCallBlock(ctx, hop, on, callBlock, idx, evalctx, logger)
		if err != nil {
			return err
		}
	}

	hop.Ons = append(hop.Ons, *on)
	return nil
}

func DecodeCallBlock(ctx context.Context, hop *HopAST, on *OnAST, block *hcl.Block, idx int, evalctx *hcl.EvalContext, logger zerolog.Logger) error {
	call := &CallAST{}

	bc, d := block.Body.Content(callSchema)
	if d.HasErrors() {
		return errors.New(d.Error())
	}

	call.TaskType = block.Labels[0]
	name, err := DecodeNameAttr(bc.Attributes[NameAttr])
	if err != nil {
		return err
	}
	if name == "" {
		name = fmt.Sprintf("%s%d", call.TaskType, idx)
	}

	call.Name = name
	call.Slug = slugify(on.Slug, call.Name)

	err = ValidateLabels(call.TaskType, call.Name)
	if err != nil {
		return err
	}

	if hop.SlugRegister[call.Slug] {
		return fmt.Errorf("Duplicate call block found: %s", call.Slug)
	} else {
		hop.SlugRegister[call.Slug] = true
	}

	ifClause := bc.Attributes[IfAttr]
	val, err := DecodeConditionalAttr(ifClause, true, evalctx)
	if err != nil {
		logger.Debug().Msgf(
			"%s 'if' not ready for evaluation, defaulting to false: %s",
			call.Slug,
			err.Error(),
		)
	}

	if !val {
		logger.Debug().Msgf("%s 'if' not met", call.Slug)
		return nil
	}

	call.IfClause = val

	logger.Info().Msgf("%s matches event", call.Slug)

	inputs := bc.Attributes["inputs"]
	if inputs != nil {
		val, d := inputs.Expr.Value(evalctx)
		if d.HasErrors() {
			return errors.New(d.Error())
		}

		jsonVal := ctyjson.SimpleJSONValue{Value: val}
		inputs, err := jsonVal.MarshalJSON()

		if err != nil {
			return err
		}

		call.Inputs = inputs
	}

	on.Calls = append(on.Calls, *call)
	return nil
}

func DecodeNameAttr(attr *hcl.Attribute) (string, error) {
	if attr == nil {
		// Not an error, as the attribute is not required
		return "", nil
	}

	val, diag := attr.Expr.Value(nil)
	if diag.HasErrors() {
		return "", errors.New(diag.Error())
	}

	var value string

	err := gocty.FromCtyValue(val, &value)
	if err != nil {
		return "", fmt.Errorf("%s %w", attr.NameRange, err)
	}

	return value, nil
}

func DecodeConditionalAttr(attr *hcl.Attribute, defaultValue bool, ctx *hcl.EvalContext) (bool, error) {
	if attr == nil {
		return defaultValue, nil
	}

	v, diag := attr.Expr.Value(ctx)
	if diag.HasErrors() {
		return false, errors.New(diag.Error())
	}

	var value bool

	err := gocty.FromCtyValue(v, &value)
	if err != nil {
		return false, fmt.Errorf("%s %w", attr.NameRange, err)
	}

	return value, nil
}

func slugify(parts ...string) string {
	joined := strings.Join(parts, "-")
	return slug.Make(joined)
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
