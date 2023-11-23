package dsl

import (
	"context"
	"errors"

	"github.com/hashicorp/hcl/v2"
	"github.com/manterfield/fast-ctyjson/ctyjson"
	"github.com/rs/zerolog"
)

func DecodeDoneBlock(ctx context.Context, hop *HopAST, on *OnAST, block *hcl.Block, evalctx *hcl.EvalContext, logger zerolog.Logger) (*DoneAST, error) {
	done := &DoneAST{}

	bc, d := block.Body.Content(doneSchema)
	if d.HasErrors() {
		return done, errors.New(d.Error())
	}

	errorVal, err := decodeErrorAttr(bc.Attributes[ErrorAttr], evalctx, logger)
	if err != nil {
		return nil, err
	}
	if errorVal != nil {
		done.Error = errors.New(*errorVal)
	}

	resultVal, err := decodeResultAttr(bc.Attributes[ResultAttr], evalctx, logger)
	if err != nil {
		return nil, err
	}

	if resultVal != nil {
		done.Result = resultVal
	}

	if resultVal != nil || errorVal != nil {
		return done, nil
	}

	return nil, err
}

func decodeErrorAttr(attr *hcl.Attribute, evalctx *hcl.EvalContext, logger zerolog.Logger) (*string, error) {
	if attr == nil {
		return nil, nil
	}

	val, d := attr.Expr.Value(evalctx)
	if d.HasErrors() {
		logger.Debug().Msgf("Evaluation skipped on 'done.%s', defaulting to null: %s", attr.Name, d.Error())
		return nil, nil
	}

	// As a syntax convenience, we interpret false values as null in done.error
	if val.False() {
		return nil, nil
	}

	valStr := val.AsString()
	return &valStr, nil
}

func decodeResultAttr(attr *hcl.Attribute, evalctx *hcl.EvalContext, logger zerolog.Logger) ([]byte, error) {
	if attr == nil {
		return nil, nil
	}

	val, d := attr.Expr.Value(evalctx)
	if d.HasErrors() {
		logger.Debug().Msgf("Evaluation skipped on 'done.%s', defaulting to null: %s", attr.Name, d.Error())
		return nil, nil
	}

	jsonVal := ctyjson.SimpleJSONValue{Value: val}
	valBytes, err := jsonVal.MarshalJSON()

	if err != nil {
		return nil, err
	}

	return valBytes, err
}
