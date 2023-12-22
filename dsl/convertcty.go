package dsl

import (
	"fmt"
	"reflect"

	"github.com/zclconf/go-cty/cty"
)

// convertCtyValueToInterface converts a cty.Value to an interface{}.
//
// Calls itself recursively to convert nested values.
// Does not cover all possible cty types, such as unknown, capsule, empty object,
// and empty tuple.
func convertCtyValueToInterface(val cty.Value) (interface{}, error) {
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
			convertedVal, err := convertCtyValueToInterface(value)
			if err != nil {
				return nil, err
			}
			resultMap[key] = convertedVal
		}
		return resultMap, nil

	case valType.IsListType(), valType.IsSetType(), valType.IsTupleType():
		resultList := []interface{}{}
		for _, item := range val.AsValueSlice() {
			convertedItem, err := convertCtyValueToInterface(item)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, convertedItem)
		}
		return resultList, nil

	case valType.IsCapsuleType():
		encapsulatedValue := val.EncapsulatedValue()

		return reflect.ValueOf(encapsulatedValue).Interface(), nil

	default:
		return nil, fmt.Errorf("Unsupported cty type: %s", val.Type().FriendlyName())
	}
}
