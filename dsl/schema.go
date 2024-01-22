package dsl

import (
	"fmt"
	"strings"
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
			{
				Type:       ScheduleID,
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

	DoneID     = "done"     // Schema defined via tags on the struct
	ParamID    = "param"    // Schema defined via tags on the struct
	ScheduleID = "schedule" // Schema defined via tags on the struct

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
)

type HopAST struct {
	Ons          []OnAST
	Schedules    []ScheduleAST
	SlugRegister map[string]bool
	StartedAt    time.Time
	Tasks        []TaskAST
}

func (h *HopAST) ListSchedules() []ScheduleAST {
	return h.Schedules
}

func (h *HopAST) ListTasks() []TaskAST {
	return h.Tasks
}

func (h *HopAST) ListFileTasks(path string) []TaskAST {
	fileTasks := []TaskAST{}

	for _, task := range h.Tasks {
		if !strings.HasPrefix(task.FilePath, path) {
			continue
		}

		fileTasks = append(fileTasks, task)
	}
	return fileTasks
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
	Errored   bool `hcl:"errored,optional" json:"errored"`
	Completed bool `hcl:"completed,optional" json:"completed"`
	Done      bool
}

type ConditionalAST struct {
	IfClause bool
}

type TaskAST struct {
	Description string     `json:"description"`
	DisplayName string     `json:"display_name"`
	Emoji       string     `json:"emoji"`
	FilePath    string     `json:"file_path"`
	Name        string     `json:"name"`
	Params      []ParamAST `json:"params"`
	Summary     string     `json:"summary"`
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

type ScheduleAST struct {
	Name   string         `hcl:"name,label" json:"name"`
	Cron   string         `hcl:"cron,attr" json:"cron"`
	Inputs []byte         `json:"inputs"` // Inputs is decoded explicitly from remain
	Remain hcl.Attributes `hcl:",remain"`
}
