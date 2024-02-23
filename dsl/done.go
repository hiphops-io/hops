package dsl

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/manterfield/fast-ctyjson/ctyjson"
	"github.com/rs/zerolog"
)

func DecodeDoneBlock(ctx context.Context, hop *HopAST, on *OnAST, block *hcl.Block, evalctx *hcl.EvalContext, logger zerolog.Logger) (*DoneAST, hcl.Diagnostics) {
	done := &DoneAST{}

	bc, diag := block.Body.Content(DoneSchema)
	if diag.HasErrors() {
		return done, diag
	}

	done.Errored = decodeDoneAttr(bc.Attributes[ErroredAttr], evalctx, logger)
	done.Completed = decodeDoneAttr(bc.Attributes[CompletedAttr], evalctx, logger)

	if done.Completed || done.Errored {
		return done, nil
	}

	return nil, nil
}

func decodeDoneAttr(attr *hcl.Attribute, evalctx *hcl.EvalContext, logger zerolog.Logger) bool {
	if attr == nil {
		return false
	}

	val, d := attr.Expr.Value(evalctx)
	if d.HasErrors() {
		logger.Debug().Msgf("Evaluation skipped on 'done.%s', defaulting to null: %s", attr.Name, d.Error())
		return false
	}

	return val.True()
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
