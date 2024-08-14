package markdown

import (
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hiphops-io/hops/expression/ctyconv"
	"github.com/hiphops-io/hops/expression/funcs"
	"github.com/hiphops-io/hops/nats"
)

const MetadataKey = "hops"

var ErrCommandNotFound = errors.New("command not found")

func MatchCommandFlows(flowIdx map[string]*Flow, hopsMsg *nats.HopsMsg, evalCtx *hcl.EvalContext) (*Flow, error) {
	if evalCtx == nil {
		eval, err := EventEvalContext(hopsMsg)
		if err != nil {
			return nil, err
		}

		evalCtx = eval
	}

	flow, ok := flowIdx[hopsMsg.Action]
	if !ok {
		return nil, ErrCommandNotFound
	}

	matches, err := flow.IfValue(evalCtx)
	if err != nil {
		return nil, err
	}

	if !matches {
		return nil, nil
	}

	return flow, nil
}

func MatchFlows(flowIdx map[string][]*Flow, hopsMsg *nats.HopsMsg, evalCtx *hcl.EvalContext) ([]*Flow, error) {
	if evalCtx == nil {
		eval, err := EventEvalContext(hopsMsg)
		if err != nil {
			return nil, err
		}

		evalCtx = eval
	}

	lookups := expandEventLookups(hopsMsg.Source, hopsMsg.Event, hopsMsg.Action)

	flows := []*Flow{}
	for _, l := range lookups {
		flows = append(flows, flowIdx[l]...)
	}

	// Omit flows with a non-matching 'if' condition
	matchedFlows := []*Flow{}
	for _, f := range flows {
		matches, err := f.IfValue(evalCtx)
		if err != nil {
			return nil, err
		}

		if matches {
			matchedFlows = append(matchedFlows, f)
		}
	}

	return matchedFlows, nil
}

func EventEvalContext(hopsMsg *nats.HopsMsg) (*hcl.EvalContext, error) {
	eventVal, err := ctyconv.InterfaceToCtyVal(hopsMsg.Data)
	if err != nil {
		return nil, err
	}

	return &hcl.EvalContext{
		Functions: funcs.DefaultFunctions,
		Variables: map[string]cty.Value{
			"event": eventVal,
		},
	}, nil
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
