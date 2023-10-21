package dsl

import (
	"fmt"
	"strings"

	"github.com/goccy/go-json"
	"github.com/manterfield/fast-ctyjson/ctyjson"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// TODO: This method effectively parses the JSON string twice. Once via unmarshal
// called directly, then again via ctyjson.ImpliedType which runs a decoder.
// It is likely worth the time to write a decoder that directly takes the unmarshalled
// json and manually maps to cty values.
// This function is quite expensive as an overall portion of the runtime.
// Taking around 200-250µs for a single change event (around 20-25% of hops parsing time).
func AnyJSONToCtyValue(jsonStr []byte) (cty.Value, error) {
	var data interface{}

	err := json.Unmarshal(jsonStr, &data)
	if err != nil {
		return cty.NilVal, err
	}

	inputType, err := ctyjson.ImpliedType(jsonStr)
	if err != nil {
		return cty.NilVal, err
	}

	event, err := gocty.ToCtyValue(data, inputType)
	if err != nil {
		return cty.NilVal, err
	}

	return event, nil
}

func eventBundleToCty(eventBundle map[string][]byte, pathDelim string) (map[string]cty.Value, error) {
	ctxVariables := make(map[string]cty.Value)
	for k, v := range eventBundle {
		ctyVal, err := AnyJSONToCtyValue(v)
		if err != nil {
			return nil, err
		}

		path := strings.Split(k, pathDelim)
		ctxVariables = nestedPathToCty(ctxVariables, path, ctyVal)
	}

	return ctxVariables, nil
}

func nestedPathToCty(ctxVal map[string]cty.Value, path []string, eventVal cty.Value) map[string]cty.Value {
	if ctxVal == nil {
		ctxVal = make(map[string]cty.Value)
	}

	if len(path) == 1 {
		ctxVal[path[0]] = eventVal
		return ctxVal
	}

	val, ok := ctxVal[path[0]]
	if !ok {
		val = cty.EmptyObjectVal
	}

	ctxVal[path[0]] = cty.ObjectVal(nestedPathToCty(val.AsValueMap(), path[1:], eventVal))

	return ctxVal
}

func parseEventVar(evalVars map[string]cty.Value) (string, string, error) {
	event, ok := evalVars["event"]
	if !ok {
		return "", "", fmt.Errorf("Source event not found")
	}

	metadata, ok := event.AsValueMap()[hopsMetadataKey]
	if !ok {
		return "", "", fmt.Errorf("Source event does not contain required metadata")
	}
	metadataMap := metadata.AsValueMap()

	eventType, ok := metadataMap["event"]
	if !ok {
		return "", "", fmt.Errorf("Source event does not contain required metadata. Missing 'event' key")
	}

	action, ok := metadataMap["action"]
	if !ok {
		return eventType.AsString(), "", nil
	}

	return eventType.AsString(), action.AsString(), nil
}
