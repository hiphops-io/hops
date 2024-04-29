package dsl

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/hashicorp/hcl/v2"
	"github.com/robfig/cron"
)

const (
	TagValidateCron            = "standard_cron"
	TagValidateLabel           = "block_label"
	ValidateLabelMaxLen        = 50
	InvalidRequired     string = "Required"
	InvalidNotString    string = "Should be a string"
	InvalidNotText      string = "Should be text"
	InvalidNotNumber    string = "Should be a number"
	InvalidNotBool      string = "Should be a boolean"
)

var (
	labelRegex = regexp.MustCompile(`^[a-z\d][a-z\d]*(?:_[a-z\d]+)*$`)
	valid      = NewHopsValidator()
)

type (
	// DiagnosticResult mirrors hcl.Diagnostic + json tags to control marshalling
	// We keep uppercase field names as this matches the runtime logged diagnostics
	DiagnosticResult struct {
		Severity    hcl.DiagnosticSeverity `json:"Severity"`
		Summary     string                 `json:"Summary"`
		Detail      string                 `json:"Detail,omitempty"`
		Subject     *hcl.Range             `json:"Subject,omitempty"`
		Context     *hcl.Range             `json:"Context,omitempty"`
		Expression  hcl.Expression         `json:"-"`
		EvalContext *hcl.EvalContext       `json:"-"`
		Extra       interface{}            `json:"Extra,omitempty"`
	}

	HopsValidator struct {
		validate     *validator.Validate
		slugRegister map[string]bool
	}

	ValidationResult struct {
		Diagnostics map[string][]DiagnosticResult `json:"diagnostics"`
		FileCount   int                           `json:"file_count"`
		IsValid     bool                          `json:"is_valid"`
		NumIssues   int                           `json:"num_issues"`
		ReadError   string                        `json:"read_error,omitempty"`
	}
)

func NewHopsValidator() *HopsValidator {
	h := &HopsValidator{
		slugRegister: make(map[string]bool),
	}

	validate := validator.New()
	validate.RegisterValidation(TagValidateLabel, ValidateLabel)
	validate.RegisterValidation(TagValidateCron, ValidateCron)
	validate.RegisterTagNameFunc(HCLTagName)

	h.validate = validate

	return h
}

func (h *HopsValidator) SlugRegister(slugRegister []slugRange) hcl.Diagnostics {
	d := hcl.Diagnostics{}
	seen := map[string]bool{}

	// Ensure we don't have multiple blocks with the same 'name'
	// (which may come from either a label or attribute - the block decides)
	for _, nr := range slugRegister {
		typedName := fmt.Sprintf("%s-%s", nr.blockID, nr.name)

		if seen[typedName] {
			var detail string
			switch nr.blockID {
			case BlockIDOn:
				detail = "If 'name' is set, it must be unique for all 'on' config blocks across all .hops files in all automations"
			case BlockIDTask:
				detail = "Tasks must have unique names across all .hops files in all automations."
			case BlockIDParam:
				detail = "Parameters must have unique names within a task's config block."
			case BlockIDSchedule:
				detail = "Schedules must have unique names across all .hops files in all automations."
			default:
				detail = "Names for a config block must be unique for that type of config block."
			}

			d = d.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate names found",
				Detail:   detail,
				Subject:  nr.hclRange,
			})
		}

		seen[typedName] = true
	}

	return d
}

func (h *HopsValidator) BlockStruct(ast hclReader) hcl.Diagnostics {
	d := hcl.Diagnostics{}

	err := h.validate.Struct(ast)
	validationErrors, ok := err.(validator.ValidationErrors)
	if ok {
		d = d.Extend(BlockErrsToDiagnostics(ast, validationErrors))
	}

	return d
}

// BlockErrsToDiagnostics converts validation errors into hcl.Diagnostics
//
// Note that this _only_ works for hcl attributes and labels, not block fields.
// Labels must have a name starting with `label` e.g. `hcl:"label,label" or hcl:"label_1,label"`
func BlockErrsToDiagnostics(ast hclReader, errs validator.ValidationErrors) hcl.Diagnostics {
	block := ast.Block()
	d := hcl.Diagnostics{}

	for _, v := range errs {
		fieldName := v.Field()
		switch {
		case strings.HasPrefix(fieldName, "label"):
			d = d.Append(
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Invalid label for `%s` block", block.Type),
					Detail:   prettyMsg(v),
					Subject:  &block.LabelRanges[0],
					Context:  &block.DefRange,
				},
			)
		default:
			attributes, diags := block.Body.JustAttributes()
			if diags.HasErrors() {
				d.Extend(diags)
				continue
			}

			var subject hcl.Range
			attr, ok := attributes[fieldName]
			if !ok {
				subject = block.Body.MissingItemRange()
			} else {
				subject = attr.Range
			}

			d = d.Append(
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Invalid `%s` for `%s` block", fieldName, block.Type),
					Detail:   prettyMsg(v),
					Subject:  &subject,
				},
			)
		}
	}

	return d
}

func HCLTagName(fl reflect.StructField) string {
	hclTag, found := fl.Tag.Lookup("hcl")
	if !found {
		return fl.Name
	}

	name, _, found := strings.Cut(hclTag, ",")
	if !found {
		return fl.Name
	}

	return name
}

func ValidateCron(fl validator.FieldLevel) bool {
	cronExpr := fl.Field().String()
	_, err := cron.ParseStandard(cronExpr)
	return err == nil
}

func ValidateLabel(fl validator.FieldLevel) bool {
	label := fl.Field().String()
	if len(label) == 0 {
		return false
	}

	if len(label) > ValidateLabelMaxLen {
		return false
	}

	return labelRegex.MatchString(label)
}

// ValidateDir validates a given automation dir and prints the result to the console
//
// This is intended for end users to validate their automations are syntactically correct
func ValidateDir(automationDir string, pretty bool) error {
	a, d, err := NewAutomationsFromDir(automationDir)

	diagResults := map[string][]DiagnosticResult{}

	for _, diag := range d {
		var key string

		if diag.Subject == nil {
			key = ""
		} else {
			key = diag.Subject.Filename
		}

		fileDiags := diagResults[key]
		diagResults[key] = append(fileDiags, DiagnosticResult(*diag))
	}

	vr := ValidationResult{
		Diagnostics: diagResults,
		FileCount:   len(a.Files),
		NumIssues:   len(d),
		IsValid:     !d.HasErrors() && err == nil,
	}

	if err != nil {
		vr.ReadError = err.Error()
	}

	if err == nil && vr.FileCount == 0 {
		vr.ReadError = "No automation directories found (or they're all empty)"
		vr.IsValid = false
	}

	var output []byte

	if !pretty {
		output, err = json.Marshal(vr)
	} else {
		output, err = json.MarshalIndentWithOption(
			vr, "", "  ",
			json.Colorize(json.DefaultColorScheme),
		)
	}

	if err != nil {
		return fmt.Errorf("Failed to compile validation results: %w", err)
	}

	fmt.Println(string(output))

	return nil
}

// ValidateTaskInput validates a struct of param inputs against a task
//
// Returns a map of parameter names with an array of validation error messages
// if any. Map will be empty (but not nil) if all input is valid.
func ValidateTaskInput(t *TaskAST, input map[string]any) map[string][]string {
	invalidErrs := map[string][]string{}

	for _, param := range t.Params {
		paramInput, ok := input[param.Name]
		paramErrs := []string{}

		if !ok && param.Required {
			invalidErrs[param.Name] = append(paramErrs, InvalidRequired)
			continue
		}
		// The only validation we can do on a missing param is checking required,
		// so let's ditch this joint.
		if !ok {
			continue
		}

		switch param.Type {
		case "string":
			if _, ok := paramInput.(string); !ok {
				invalidErrs[param.Name] = append(paramErrs, InvalidNotString)
			}
		case "text":
			if _, ok := paramInput.(string); !ok {
				invalidErrs[param.Name] = append(paramErrs, InvalidNotText)
			}
		case "number":
			if _, ok := paramInput.(int); ok {
				continue
			}
			if _, ok := paramInput.(float64); ok {
				continue
			}
			invalidErrs[param.Name] = append(paramErrs, InvalidNotNumber)
		case "bool":
			if _, ok := paramInput.(bool); !ok {
				invalidErrs[param.Name] = append(paramErrs, InvalidNotBool)
			}
		}

		if len(paramErrs) > 0 {
			invalidErrs[param.Name] = paramErrs
		}
	}

	return invalidErrs
}

func prettyMsg(fe validator.FieldError) string {
	var msg string
	switch fe.Tag() {
	case TagValidateCron:
		msg = fmt.Sprintf("Invalid cron format: %v. Consult syntax section of https://docs.hiphops.io for more information", fe.Value())

	case TagValidateLabel:
		msg = fmt.Sprintf("Labels must be lowercase alphanumeric separated by underscores, max %d characters", ValidateLabelMaxLen)

	case "oneof":
		msg = fmt.Sprintf("This value is not allowed, must be one of '%s'", fe.Param())

	default:
		msg = fe.Error()
	}

	return msg
}
