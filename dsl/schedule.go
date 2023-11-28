package dsl

import (
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/manterfield/fast-ctyjson/ctyjson"
	"github.com/robfig/cron"
	"github.com/rs/zerolog"
)

func DecodeSchedules(hop *HopAST, hopsContent *hcl.BodyContent, evalctx *hcl.EvalContext) error {
	scheduleBlocks := hopsContent.Blocks.OfType(ScheduleID)
	for _, scheduleBlock := range scheduleBlocks {
		err := DecodeScheduleBlock(scheduleBlock, hop, evalctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func DecodeScheduleBlock(block *hcl.Block, hop *HopAST, evalctx *hcl.EvalContext) error {
	schedule := ScheduleAST{}

	diag := gohcl.DecodeBody(block.Body, evalctx, &schedule)
	if diag.HasErrors() {
		return errors.New(diag.Error())
	}

	schedule.Name = block.Labels[0]
	slug := slugify(schedule.Name)

	if hop.SlugRegister[slug] {
		return fmt.Errorf("Duplicate schedule name found: %s", slug)
	} else {
		hop.SlugRegister[slug] = true
	}

	_, err := cron.ParseStandard(schedule.Cron)
	if err != nil {
		return fmt.Errorf("Invalid cron for schedule '%s' %s: %w", schedule.Name, block.TypeRange.String(), err)
	}

	inputAttr, found := schedule.Remain["inputs"]
	if found && inputAttr != nil {
		val, d := inputAttr.Expr.Value(evalctx)
		if d.HasErrors() {
			return errors.New(d.Error())
		}

		jsonVal := ctyjson.SimpleJSONValue{Value: val}
		inputs, err := jsonVal.MarshalJSON()

		if err != nil {
			return err
		}

		schedule.Inputs = inputs
	}

	hop.Schedules = append(hop.Schedules, schedule)

	return nil
}

func ParseHopsSchedules(hopsContent *hcl.BodyContent, logger zerolog.Logger) (*HopAST, error) {
	hop := &HopAST{
		SlugRegister: make(map[string]bool),
	}

	evalctx := &hcl.EvalContext{
		Functions: DefaultFunctions,
	}

	err := DecodeSchedules(hop, hopsContent, evalctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to decode hops schedules")
		return hop, err
	}

	return hop, nil
}
