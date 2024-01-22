package dsl

import (
	"context"
	"errors"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/rs/zerolog"
)

func DecodeDoneBlock(ctx context.Context, hop *HopAST, on *OnAST, block *hcl.Block, evalctx *hcl.EvalContext, logger zerolog.Logger) (*DoneAST, error) {
	done := &DoneAST{}

	// TODO: Need to make this tolerant of missing variables on errored and completed
	diag := gohcl.DecodeBody(block.Body, evalctx, done)
	if diag.HasErrors() {
		return done, errors.New(diag.Error())
	}

	if done.Errored != false || done.Completed != false {
		return done, nil
	}

	return nil, nil
}

// currently unused - will be re-used to allow appending of error data to results
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
