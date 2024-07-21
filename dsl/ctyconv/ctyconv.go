// Package ctyconv provides helpers for converting betweem dynamic JSON/go structs and cty values
package ctyconv

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/goccy/go-json"
	"github.com/zclconf/go-cty/cty"
)

// JSONToCtyValue converts an aribitrary json byte slice and converts to a corresponding cty.Value
func JSONToCtyValue(jsonStr []byte) (cty.Value, error) {
	var data interface{}

	err := json.Unmarshal(jsonStr, &data)
	if err != nil {
		return cty.NilVal, err
	}

	event, err := InterfaceToCtyVal(data)
	if err != nil {
		return cty.NilVal, err
	}

	return event, nil
}

func InterfaceToCtyVal(i interface{}) (cty.Value, error) {
	switch i := i.(type) {
	case string:
		return cty.StringVal(i), nil

	case bool:
		return cty.BoolVal(i), nil

	case float64:
		bigFloat := new(big.Float).SetFloat64(i)
		return cty.NumberVal(bigFloat), nil

	case nil:
		return cty.NullVal(cty.DynamicPseudoType), nil

	case []interface{}:
		vals := []cty.Value{}
		for _, v := range i {
			v, err := InterfaceToCtyVal(v)
			if err != nil {
				return cty.NilVal, err
			}

			vals = append(vals, v)
		}
		return cty.TupleVal(vals), nil

	case map[string]interface{}:
		mapVals := map[string]cty.Value{}
		for k, v := range i {
			val, err := InterfaceToCtyVal(v)
			if err != nil {
				return cty.NilVal, err
			}

			mapVals[k] = val
		}
		return cty.ObjectVal(mapVals), nil

	default:
		return cty.NilVal, fmt.Errorf("Unknown type: %v", i)
	}
}

func CtyToEventAction(val cty.Value, metadataKey string) (event string, action string, err error) {
	metadata, ok := val.AsValueMap()[metadataKey]
	if !ok {
		return "", "", fmt.Errorf("event is missing required metadata")
	}
	metadataMap := metadata.AsValueMap()

	eventVal, ok := metadataMap["event"]
	if !ok {
		return "", "", fmt.Errorf("event is missing required metadata. Missing 'event' key")
	}

	actionVal, ok := metadataMap["action"]
	if !ok {
		return eventVal.AsString(), "", nil
	}

	return eventVal.AsString(), actionVal.AsString(), nil
}

// CtyValueToInterface converts a cty.Value to an interface{}.
//
// Calls itself recursively to convert nested values.
// Does not cover all possible cty types, such as unknown, capsule, empty object,
// and empty tuple.
func CtyValueToInterface(val cty.Value) (interface{}, error) {
	if val.IsNull() || !val.IsKnown() {
		return nil, nil
	}

	valType := val.Type()
	switch {

	case valType.Equals(cty.String):
		return val.AsString(), nil

	case valType.Equals(cty.Number):
		num, _ := val.AsBigFloat().Float64()
		return num, nil

	case valType.Equals(cty.Bool):
		return val.True(), nil

	case valType.IsMapType(), valType.IsObjectType():
		resultMap := make(map[string]interface{})
		for key, value := range val.AsValueMap() {
			convertedVal, err := CtyValueToInterface(value)
			if err != nil {
				return nil, err
			}
			resultMap[key] = convertedVal
		}
		return resultMap, nil

	case valType.IsListType(), valType.IsSetType(), valType.IsTupleType():
		resultList := []interface{}{}
		for _, item := range val.AsValueSlice() {
			convertedItem, err := CtyValueToInterface(item)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, convertedItem)
		}
		return resultList, nil

	case valType.IsCapsuleType():
		return reflect.ValueOf(val.EncapsulatedValue()).Interface(), nil

	default:
		return nil, fmt.Errorf("Unsupported cty type: %s", val.Type().FriendlyName())
	}
}
