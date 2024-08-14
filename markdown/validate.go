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

	uniqueNames := map[string]bool{}

	for _, p := range command {
		name, ok := ValidateParam(p)
		if !ok {
			return false
		}

		if seen := uniqueNames[name]; seen {
			return false
		}

		uniqueNames[name] = true
	}

	return true
}

func ValidateParam(p ParamItem) (string, bool) {
	if len(p) != 1 {
		return "", false
	}

	name, param := p.Param()

	switch param.Type {
	case "string":
		if param.Default == nil {
			break
		}

		if _, ok := param.Default.(string); !ok {
			return "", false
		}
	case "text":
		if param.Default == nil {
			break
		}

		if _, ok := param.Default.(string); !ok {
			return "", false
		}
	case "number":
		if param.Default == nil {
			break
		}

		if _, ok := param.Default.(int); ok {
			break
		}
		if _, ok := param.Default.(float64); ok {
			break
		}

		return "", false
	case "bool":
		if param.Default == nil {
			break
		}

		if _, ok := param.Default.(bool); !ok {
			return "", false
		}
	default:
		return "", false
	}

	return name, true
}
