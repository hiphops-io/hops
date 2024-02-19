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

	bc, d := block.Body.Content(DoneSchema)
	if d.HasErrors() {
		return done, errors.New(d.Error())
	}

	errored, err := decodeDoneAttr(bc.Attributes[ErroredAttr], evalctx, logger)
	if err != nil {
		return nil, err
	}
	done.Errored = errored

	completed, err := decodeDoneAttr(bc.Attributes[CompletedAttr], evalctx, logger)
	if err != nil {
		return nil, err
	}
	done.Completed = completed

	if completed || errored {
		return done, nil
	}

	return nil, err
}

func decodeDoneAttr(attr *hcl.Attribute, evalctx *hcl.EvalContext, logger zerolog.Logger) (bool, error) {
	if attr == nil {
		return false, nil
	}

	val, d := attr.Expr.Value(evalctx)
	if d.HasErrors() {
		logger.Debug().Msgf("Evaluation skipped on 'done.%s', defaulting to null: %s", attr.Name, d.Error())
		return false, nil
	}

	return val.True(), nil
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
