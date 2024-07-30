package markdown

import (
	"github.com/go-playground/validator/v10"
	"github.com/robfig/cron"
)

var flowValidator = NewFlowValidator()

const (
	TagValidateCron    = "standard_cron"
	TagValidateCommand = "command"
)

type FlowValidator struct {
	validate *validator.Validate
}

func NewFlowValidator() *FlowValidator {
	fv := &FlowValidator{}

	validate := validator.New()
	validate.RegisterValidation(TagValidateCron, ValidateCron)
	validate.RegisterValidation(TagValidateCommand, ValidateCommand)

	fv.validate = validate

	return fv
}

func ValidateCron(fl validator.FieldLevel) bool {
	cronExpr := fl.Field().String()
	_, err := cron.ParseStandard(cronExpr)
	return err == nil
}

func ValidateCommand(fl validator.FieldLevel) bool {
	command, ok := fl.Field().Interface().(Command)
	if !ok {
		return false
	}

	for _, p := range command {
		ok := ValidateParamValueType(p.Type, p.Default)
		if !ok {
			return false
		}
	}

	return true
}

func ValidateParamValueType(paramType string, value any) bool {
	switch paramType {
	case "string":
		if _, ok := value.(string); !ok {
			return false
		}
	case "text":
		if _, ok := value.(string); !ok {
			return false
		}
	case "number":
		if _, ok := value.(int); ok {
			break
		}
		if _, ok := value.(float64); ok {
			break
		}

		return false
	case "bool":
		if _, ok := value.(bool); !ok {
			return false
		}
	default:
		return false
	}

	return true
}
