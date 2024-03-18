package ctyutils

import (
	"fmt"
	"reflect"

	"github.com/zclconf/go-cty/cty"
)

// ConvertCtyValueToInterface converts a cty.Value to an interface{}.
//
// Calls itself recursively to convert nested values.
// Does not cover all possible cty types, such as unknown, capsule, empty object,
// and empty tuple.
func ConvertCtyValueToInterface(val cty.Value) (interface{}, error) {
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
			convertedVal, err := ConvertCtyValueToInterface(value)
			if err != nil {
				return nil, err
			}
			resultMap[key] = convertedVal
		}
		return resultMap, nil

	case valType.IsListType(), valType.IsSetType(), valType.IsTupleType():
		resultList := []interface{}{}
		for _, item := range val.AsValueSlice() {
			convertedItem, err := ConvertCtyValueToInterface(item)
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
