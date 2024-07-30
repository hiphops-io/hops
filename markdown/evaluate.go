package markdown

import (
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hiphops-io/hops/expression/ctyconv"
	"github.com/hiphops-io/hops/expression/funcs"
)

const MetadataKey = "hops"

func MatchFlows(flowIdx FlowIndex, eventData []byte) ([]*Flow, error) {
	eventVal, err := ctyconv.JSONToCtyValue(eventData)
	if err != nil {
		return nil, err
	}

	source, event, action, err := ctyconv.CtyToEventAction(eventVal, MetadataKey)
	if err != nil {
		return nil, err
	}

	lookups := expandEventLookups(source, event, action)

	flows := []*Flow{}
	for _, l := range lookups {
		flows = append(flows, flowIdx[l]...)
	}

	evalCtx := &hcl.EvalContext{
		Functions: funcs.DefaultFunctions,
		Variables: map[string]cty.Value{
			"event": eventVal,
		},
	}

	// Omit flows with a non-matching 'if' condition
	matchedFlows := []*Flow{}
	for _, f := range flows {
		if f.If == "" {
			matchedFlows = append(matchedFlows, f)
			continue
		}

		ifVal, diags := f.ifExpression.Value(evalCtx)
		if diags.HasErrors() {
			return nil, errors.Join(diags.Errs()...)
		}

		var matches bool
		err := gocty.FromCtyValue(ifVal, &matches)
		if err != nil {
			return nil, fmt.Errorf("'if' expression must evaluate to true or false: %w", err)
		}

		if matches {
			matchedFlows = append(matchedFlows, f)
		}
	}

	return matchedFlows, nil
}

func expandEventLookups(source, event, action string) []string {
	lookups := []string{
		fmt.Sprintf("*.%s.*", event),
		fmt.Sprintf("%s.%s.*", source, event),
	}

	if action == "" {
		return lookups
	}

	lookups = append(
		lookups,
		fmt.Sprintf("%s.%s.%s", source, event, action),
		fmt.Sprintf("*.%s.%s", event, action),
	)

	return lookups
}
