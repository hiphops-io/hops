package dsl

import (
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
)

var (
	ErrorAttr  = "error"
	ResultAttr = "result"
	IfAttr     = "if"
	NameAttr   = "name"

	HopSchema = &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{},
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type:       OnID,
				LabelNames: []string{"eventType"},
			},
			{
				Type:       TaskID,
				LabelNames: []string{"Name"},
			},
		},
	}

	OnID     = "on"
	OnSchema = &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type:       CallID,
				LabelNames: []string{"taskType"},
			},
			{
				Type: DoneID,
			},
		},
		Attributes: []hcl.AttributeSchema{
			{Name: "name", Required: false},
			{Name: IfAttr, Required: false},
		},
	}

	CallID     = "call"
	callSchema = &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{},
		Attributes: []hcl.AttributeSchema{
			{Name: "name", Required: false},
			{Name: IfAttr, Required: false},
			{Name: "inputs", Required: false},
		},
	}

	DoneID     = "done"
	doneSchema = &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{},
		Attributes: []hcl.AttributeSchema{
			{Name: ErrorAttr, Required: false},
			{Name: ResultAttr, Required: false},
		},
	}

	TaskID     = "task"
	taskSchema = &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type:       ParamID,
				LabelNames: []string{"Name"},
			},
		},
		Attributes: []hcl.AttributeSchema{
			{Name: "display_name", Required: false},
			{Name: "summary", Required: false},
			{Name: "description", Required: false},
			{Name: "emoji", Required: false},
		},
	}

	ParamID = "param" // Schema defined via tags on the struct
)

type HopAST struct {
	Ons          []OnAST
	Tasks        []TaskAST
	SlugRegister map[string]bool
	StartedAt    time.Time
}

func (h *HopAST) ListTasks() []TaskAST {
	return h.Tasks
}

func (h *HopAST) GetTask(taskName string) (TaskAST, error) {
	// TODO: This currently searches all tasks rather than map lookup. Improve in future
	for _, task := range h.Tasks {
		if task.Name == taskName {
			return task, nil
		}
	}

	return TaskAST{}, fmt.Errorf("Task '%s' not found", taskName)
}

type OnAST struct {
	Slug      string
	EventType string
	Name      string
	Calls     []CallAST
	Done      *DoneAST
	ConditionalAST
}

type CallAST struct {
	Slug     string
	TaskType string
	Name     string
	Inputs   []byte
	ConditionalAST
}

type DoneAST struct {
	Error  error
	Result []byte
}

type ConditionalAST struct {
	IfClause bool
}

type TaskAST struct {
	Name        string     `json:"name"`
	DisplayName string     `json:"display_name"`
	Summary     string     `json:"summary"`
	Description string     `json:"description"`
	Emoji       string     `json:"emoji"`
	Params      []ParamAST `json:"params"`
}

const (
	InvalidRequired  string = "Required"
	InvalidNotString string = "Should be a string"
	InvalidNotText   string = "Should be text"
	InvalidNotNumber string = "Should be a number"
	InvalidNotBool   string = "Should be a boolean"
)

// ValidateInput validates a struct of param inputs against a task
//
// Returns a map of parameter names with an array of validation error messages
// if any. Map will be empty (but not nil) if all input is valid.
func (c *TaskAST) ValidateInput(input map[string]any) map[string][]string {
	invalidErrs := map[string][]string{}

	for _, param := range c.Params {
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

type ParamAST struct {
	// We use HCL tags to auto-decode as params need very little custom decoding logic
	Name        string `hcl:"name,label" json:"name"`
	DisplayName string `hcl:"display_name,optional" json:"display_name"`
	Type        string `hcl:"type,optional" json:"type"`
	Default     any    `hcl:"default,optional" json:"default"`
	Help        string `hcl:"help,optional" json:"help"`
	Flag        string `hcl:"flag,optional" json:"flag"`
	ShortFlag   string `hcl:"shortflag,optional" json:"shortflag"`
	Required    bool   `hcl:"required,optional" json:"required"`
}
