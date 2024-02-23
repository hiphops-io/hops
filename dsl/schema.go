package dsl

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var (
	ErroredAttr   = "errored"
	CompletedAttr = "completed"
	IfAttr        = "if"
	NameAttr      = "name"
	OnID          = "on"
	CallID        = "call"
	DoneID        = "done"
	TaskID        = "task"
	ParamID       = "param"
	ScheduleID    = "schedule"
)

var (
	HopSchema, _  = gohcl.ImpliedBodySchema(HopAST{})
	OnSchema, _   = gohcl.ImpliedBodySchema(OnAST{})
	CallSchema, _ = gohcl.ImpliedBodySchema(CallAST{})
	DoneSchema, _ = gohcl.ImpliedBodySchema(DoneAST{})
	TaskSchema, _ = gohcl.ImpliedBodySchema(TaskAST{})
)

type HopAST struct {
	Ons          []OnAST       `hcl:"on,block" json:"ons"`
	Schedules    []ScheduleAST `hcl:"schedule,block" json:"schedules"`
	Tasks        []TaskAST     `hcl:"task,block" json:"tasks"`
	StartedAt    time.Time     `json:"started_at"`
	SlugRegister map[string]bool
	Diagnostics  hcl.Diagnostics
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
	EventType string    `hcl:"event_type,label" json:"event_type"`
	Calls     []CallAST `hcl:"call,block" json:"calls"`
	Done      *DoneAST  `hcl:"done,block" json:"done"` // TODO: This probably needs to be swapped to a slice of Done blocks
	If        *bool     `hcl:"if" json:"if"`
	Name      string    `hcl:"name,optional" json:"name"`
	Slug      string
}

type CallAST struct {
	ActionType string `hcl:"action_type,label" json:"action_type"`
	If         *bool  `hcl:"if" json:"if"`
	Name       string `hcl:"name,optional" json:"name"`
	RawInputs  any    `hcl:"inputs,optional"`
	Inputs     []byte `json:"inputs"` // Inputs field is decoded explicitly from remain
	Slug       string
}

type DoneAST struct {
	Errored   bool `hcl:"errored,optional" json:"errored"`
	Completed bool `hcl:"completed,optional" json:"completed"`
}

type TaskAST struct {
	Name        string     `hcl:"name,label" json:"name"`
	Description string     `hcl:"description,optional" json:"description"`
	DisplayName string     `hcl:"display_name,optional" json:"display_name"`
	Emoji       string     `hcl:"emoji,optional" json:"emoji"`
	Params      []ParamAST `hcl:"param,block" json:"params"`
	Summary     string     `hcl:"summary,optional" json:"summary"`
	FilePath    string     `json:"file_path"`
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
	Inputs []byte         `json:"inputs"` // Inputs field is decoded explicitly from remain
	Remain hcl.Attributes `hcl:",remain"`
}
