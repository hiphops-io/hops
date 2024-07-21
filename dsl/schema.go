// Package dsl defines the schema, parsing, validation and evaluation logic for .hops files and automations
package dsl

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gosimple/slug"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	BlockIDOn       = "on"
	BlockIDTask     = "task"
	BlockIDParam    = "param"
	BlockIDSchedule = "schedule"
)

var (
	SchemaHops, _     = gohcl.ImpliedBodySchema(HopsAST{})
	SchemaOn, _       = gohcl.ImpliedBodySchema(OnAST{})
	SchemaParam, _    = gohcl.ImpliedBodySchema(ParamAST{})
	SchemaSchedule, _ = gohcl.ImpliedBodySchema(ScheduleAST{})
	SchemaTask, _     = gohcl.ImpliedBodySchema(TaskAST{})
)

type (
	HopsAST struct {
		Ons           []*OnAST       `json:"ons,omitempty" hcl:"on,block"`
		Schedules     []*ScheduleAST `json:"schedules,omitempty" hcl:"schedule,block"`
		Tasks         []*TaskAST     `json:"tasks,omitempty" hcl:"task,block"`
		Body          hcl.Body       `json:"-" hcl:",body"`
		eventIndex    map[string][]*OnAST
		slugRegister  []slugRange
		evaluationCtx *EvaluationCtx
	}

	OnAST struct {
		Label  string         `json:"label" hcl:"label,label" validate:"block_label"`
		Name   string         `json:"name" hcl:"name,label" validate:"block_label"`
		Worker string         `json:"worker,omitempty" hcl:"worker,optional" validate:"required"`
		IfExpr hcl.Expression `json:"-" hcl:"if,optional"`
		Slug   string         `json:"-"`
		hclStore
	}

	Manifest struct {
		Description  string         `json:"description" validate:"omitempty,gte=1"`
		Emoji        string         `json:"emoji"`
		Name         string         `json:"name" validate:"required,gte=1"`
		RequiredApps []string       `json:"required_apps"`
		Steps        []ManifestStep `json:"steps"`
		Tags         []string       `json:"tags"`
		Version      string         `json:"version" validate:"required,gte=1"`
	}

	ManifestStep struct {
		Title        string `json:"title" validate:"required,gte=1"`
		Instructions string `json:"instructions"`
		Emoji        string `json:"emoji"`
	}

	ParamAST struct {
		Default     any            `json:"default,omitempty"`
		DefaultExpr hcl.Expression `json:"-" hcl:"default,optional"`
		DisplayName string         `json:"display_name,omitempty" hcl:"display_name,optional"`
		Flag        string         `json:"flag,omitempty" hcl:"flag,optional"`
		Help        string         `json:"help,omitempty" hcl:"help,optional"`
		Name        string         `json:"name" hcl:"label,label" validate:"block_label"`
		Required    bool           `json:"required" hcl:"required,optional"`
		ShortFlag   string         `json:"shortflag,omitempty" hcl:"shortflag,optional"`
		Type        string         `json:"type" hcl:"type,optional" validate:"omitempty,oneof=string text number bool"`
		hclStore
	}

	ScheduleAST struct {
		Cron       string         `json:"cron" hcl:"cron,attr" validate:"standard_cron"`
		Inputs     []byte         `json:"inputs,omitempty"`
		InputsExpr hcl.Expression `json:"-" hcl:"inputs,optional"`
		Name       string         `json:"name" hcl:"label,label" validate:"block_label"`
		hclStore
	}

	TaskAST struct {
		Description string      `json:"description,omitempty" hcl:"description,optional"`
		DisplayName string      `json:"display_name,omitempty" hcl:"display_name,optional"`
		Emoji       string      `json:"emoji,omitempty" hcl:"emoji,optional"`
		Name        string      `json:"name" hcl:"label,label" validate:"block_label"`
		Params      []*ParamAST `json:"params,omitempty" hcl:"param,block"`
		Summary     string      `json:"summary,omitempty" hcl:"summary,optional"`
		FilePath    string      `json:"filepath"`
		hclStore
	}

	hclReader interface {
		Block() *hcl.Block
	}

	hclStore struct {
		block *hcl.Block
	}

	slugRange struct {
		name     string
		blockID  string
		hclRange *hcl.Range
	}
)

// DecodeToHopsAST takes a parsed hcl.File and partially decodes it into an AST
//
// Any diagnostic errors will be gathered and returned rather than causing early
// exit. This means the returned HopsAST may be partially populated.
// This function will not evaluate runtime expressions (e.g. 'if' statements).
func DecodeToHopsAST(body hcl.Body, evaluationCtx *EvaluationCtx) (*HopsAST, hcl.Diagnostics) {
	h := &HopsAST{
		eventIndex:    map[string][]*OnAST{},
		slugRegister:  []slugRange{},
		evaluationCtx: evaluationCtx,
	}
	var d hcl.Diagnostics
	// We ignore diagnostics from this first pass, as we'll gather them on a per
	// schema item basis. DecodeBody is used for simplicity here
	gohcl.DecodeBody(body, evaluationCtx.evalCtx, h)

	if h.Body != nil {
		d = h.DecodeHopsAST()
	}

	return h, d
}

func (h *HopsAST) DecodeHopsAST() hcl.Diagnostics {
	content, d := h.Body.Content(SchemaHops)
	if content == nil {
		return d
	}

	ons := content.Blocks.OfType(BlockIDOn)
	for i, on := range h.Ons {
		on.block = ons[i]
		d = d.Extend(h.DecodeOnAST(on, i))
	}

	tasks := content.Blocks.OfType(BlockIDTask)
	for i, task := range h.Tasks {
		task := task
		task.block = tasks[i]
		d = d.Extend(h.DecodeTaskAST(task))
	}

	schedules := content.Blocks.OfType(BlockIDSchedule)
	for i, schedule := range h.Schedules {
		schedule.block = schedules[i]
		d = d.Extend(h.DecodeScheduleAST(schedule))
	}

	d = d.Extend(valid.SlugRegister(h.slugRegister))

	return d
}

func (h *HopsAST) DecodeOnAST(on *OnAST, idx int) hcl.Diagnostics {
	content, d := on.block.Body.Content(SchemaOn)
	if content == nil {
		return d
	}

	h.indexOn(on)

	on.Slug = slugify(on.Label, on.Name)
	h.slugRegister = append(h.slugRegister, slugRange{on.Slug, BlockIDOn, &on.block.LabelRanges[1]})

	// Convert relative worker names to absolute
	if !strings.Contains(on.Worker, ".") {
		dir, _ := filepath.Split(on.block.DefRange.Filename)
		on.Worker = fmt.Sprintf("%s.%s", filepath.Base(dir), on.Worker)
	}

	d = d.Extend(valid.BlockStruct(on))

	return d
}

func (h *HopsAST) DecodeTaskAST(task *TaskAST) hcl.Diagnostics {
	content, d := task.block.Body.Content(SchemaTask)
	if content == nil {
		return d
	}

	if task.DisplayName == "" {
		task.DisplayName = titleCase(task.Name)
	}

	task.FilePath = task.block.DefRange.Filename

	d = d.Extend(valid.BlockStruct(task))

	h.slugRegister = append(h.slugRegister, slugRange{task.Name, BlockIDTask, &task.block.LabelRanges[0]})

	params := content.Blocks.OfType(BlockIDParam)
	for i, param := range task.Params {
		param.block = params[i]
		d = d.Extend(h.DecodeParamAST(param, task.Name))
	}

	return d
}

func (h *HopsAST) DecodeScheduleAST(schedule *ScheduleAST) hcl.Diagnostics {
	content, d := schedule.block.Body.Content(SchemaSchedule)
	if content == nil {
		return d
	}

	h.slugRegister = append(h.slugRegister, slugRange{schedule.Name, BlockIDSchedule, &schedule.block.LabelRanges[0]})

	d = d.Extend(valid.BlockStruct(schedule))

	if schedule.InputsExpr == nil || d.HasErrors() {
		return d
	}

	// Evaluate the inputs block as an expression
	blockEval := h.evaluationCtx.BlockScopedEvalContext(schedule.block, schedule.Name)
	inputs, diags := EvaluateInputsExpression(schedule.InputsExpr, blockEval)
	if !diags.HasErrors() {
		schedule.Inputs = inputs
	}

	d = d.Extend(diags)

	return d
}

func (h *HopsAST) DecodeParamAST(param *ParamAST, namePrefix string) hcl.Diagnostics {
	content, d := param.block.Body.Content(SchemaParam)
	if content == nil {
		return d
	}

	if param.Type == "" {
		param.Type = "string"
	}

	if param.DisplayName == "" {
		param.DisplayName = titleCase(param.Name)
	}

	if param.Flag == "" {
		param.Flag = fmt.Sprintf("--%s", param.Name)
	}

	name := fmt.Sprintf("%s-%s", namePrefix, param.Name)
	h.slugRegister = append(h.slugRegister, slugRange{name, BlockIDParam, &param.block.LabelRanges[0]})

	d = d.Extend(valid.BlockStruct(param))

	if param.DefaultExpr == nil || d.HasErrors() {
		return d
	}

	// Now evaluate the dynamically typed 'default' attribute
	evalCtx := h.evaluationCtx.BlockScopedEvalContext(param.block, name)
	var (
		defaultSet  bool
		defaultDiag hcl.Diagnostics
		defaultVal  any
	)

	switch param.Type {
	case "string", "text":
		defaultVal, defaultSet, defaultDiag = EvaluateGenericExpression[string](param.DefaultExpr, evalCtx)
	case "number":
		defaultVal, defaultSet, defaultDiag = EvaluateGenericExpression[float64](param.DefaultExpr, evalCtx)
	case "bool":
		defaultVal, defaultSet, defaultDiag = EvaluateGenericExpression[bool](param.DefaultExpr, evalCtx)
	default:
		defaultDiag = hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid default for param",
				Detail:   fmt.Sprintf("Unable to evaluate param %s, unknown param type %s", param.Name, param.Type),
				Subject:  param.DefaultExpr.Range().Ptr(),
			},
		}
		defaultVal, defaultSet = nil, false
	}

	d = d.Extend(defaultDiag)

	if !defaultSet {
		param.Default = nil
	} else {
		param.Default = defaultVal
	}

	return d
}

func (h *HopsAST) indexOn(on *OnAST) {
	eventOns, ok := h.eventIndex[on.Label]
	if !ok {
		h.eventIndex[on.Label] = []*OnAST{on}
		return
	}

	h.eventIndex[on.Label] = append(eventOns, on)
}

func (h *hclStore) Block() *hcl.Block {
	return h.block
}

func slugify(parts ...string) string {
	joined := strings.Join(parts, "-")
	return slug.Make(joined)
}

func titleCase(label string) string {
	caser := cases.Title(language.BritishEnglish)
	label = strings.ReplaceAll(label, "_", " ")
	return caser.String(label)
}
