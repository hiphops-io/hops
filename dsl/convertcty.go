package dsl

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// convertCtyValueToInterface converts a cty.Value to an interface{}.
//
// Calls itself recursively to convert nested values.
// Does not cover all possible cty types, such as unknown, capsule, empty object,
// and empty tuple.
func convertCtyValueToInterface(val cty.Value) (interface{}, error) {
	if val.IsNull() {
		return nil, nil
	}

	switch {

	case val.Type().Equals(cty.String):
		return val.AsString(), nil

	case val.Type().Equals(cty.Number):
		num, _ := val.AsBigFloat().Float64()
		return num, nil

	case val.Type().Equals(cty.Bool):
		return val.True(), nil

	case val.Type().IsMapType():
		resultMap := make(map[string]interface{})
		for key, value := range val.AsValueMap() {
			convertedVal, err := convertCtyValueToInterface(value)
			if err != nil {
				return nil, err
			}
			resultMap[key] = convertedVal
		}
		return resultMap, nil

	case val.Type().IsListType() || val.Type().IsSetType():
		var resultList []interface{}
		for _, item := range val.AsValueSlice() {
			convertedItem, err := convertCtyValueToInterface(item)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, convertedItem)
		}
		return resultList, nil

	case val.Type().IsObjectType():
		objValMap := val.AsValueMap()
		resultMap := make(map[string]interface{})
		for key, value := range objValMap {
			convertedVal, err := convertCtyValueToInterface(value)
			if err != nil {
				return nil, err
			}
			resultMap[key] = convertedVal
		}
		return resultMap, nil

	case val.Type().IsTupleType():
		var resultList []interface{}
		for _, item := range val.AsValueSlice() {
			convertedItem, err := convertCtyValueToInterface(item)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, convertedItem)
		}
		return resultList, nil

	default:
		return nil, fmt.Errorf("unsupported cty type: %s", val.Type().FriendlyName())
	}
}
