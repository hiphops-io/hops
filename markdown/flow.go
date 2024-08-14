package markdown

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/yuin/goldmark/ast"
	"github.com/zclconf/go-cty/cty/gocty"
	"go.abhg.dev/goldmark/frontmatter"
)

type (
	Command []ParamItem

	Flow struct {
		If       string  `yaml:"if"`
		Command  Command `yaml:"command" validate:"required_without_all=On Schedule,omitempty,command"`
		On       string  `yaml:"on" validate:"required_without_all=Command Schedule"`
		Schedule string  `yaml:"schedule" validate:"required_without_all=On Command,omitempty,standard_cron"`
		Worker   string  `yaml:"worker"`
		// Calculated fields
		ID           string
		dirName      string
		fileName     string
		ifExpression hcl.Expression
		markdown     ast.Node
		path         string
	}

	FlowIndex struct {
		Commands  map[string]*Flow
		Schedules []*Flow
		Sensors   map[string][]*Flow
	}

	FlowReader struct {
		basePath   string
		index      FlowIndex
		indexMutex sync.RWMutex
		md         *Markdown
	}

	ParamItem map[string]Param

	Param struct {
		Type     string `yaml:"type" validate:"oneof=string text number bool"`
		Default  any    `yaml:"default"`
		Required bool   `yaml:"required"`
	}
)

func NewFlowIndex() FlowIndex {
	return FlowIndex{
		Sensors:   map[string][]*Flow{},
		Commands:  map[string]*Flow{},
		Schedules: []*Flow{},
	}
}

func NewFlowReader(basePath string) *FlowReader {
	return &FlowReader{
		basePath: basePath,
		index:    NewFlowIndex(),
		md:       NewMarkdownHTML(),
	}
}

// IndexedCommands returns all indexed flows that are triggered by commands
func (fr *FlowReader) IndexedCommands() map[string]*Flow {
	fr.indexMutex.RLock()
	defer fr.indexMutex.RUnlock()
	return fr.index.Commands
}

// IndexedSchedules returns all indexed flows that are triggered by a schedule
func (fr *FlowReader) IndexedSchedules() []*Flow {
	fr.indexMutex.RLock()
	defer fr.indexMutex.RUnlock()
	return fr.index.Schedules
}

// IndexedSensors returns all indexed flows that are triggered by events
func (fr *FlowReader) IndexedSensors() map[string][]*Flow {
	fr.indexMutex.RLock()
	defer fr.indexMutex.RUnlock()
	return fr.index.Sensors
}

func (fr *FlowReader) ReadAll() error {
	fr.indexMutex.Lock()
	defer fr.indexMutex.Unlock()

	fr.index = NewFlowIndex()

	baseDepth := strings.Count(fr.basePath, string(os.PathSeparator))

	err := filepath.WalkDir(fr.basePath, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Only go one sub-directory deep
		if de.IsDir() && strings.Count(path, string(os.PathSeparator)) > baseDepth+1 {
			return fs.SkipDir
		}

		if de.IsDir() {
			return nil
		}

		if strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}

		flow, err := fr.ReadFlow(path)
		if err != nil {
			return err
		}

		if err := fr.indexFlow(flow); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to read flow files: %w", err)
	}

	return nil
}

func (fr *FlowReader) ReadFlow(path string) (*Flow, error) {
	ast, pCtx, err := fr.md.ParseFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to load flow: %w", err)
	}

	f := &Flow{path: path}
	fm := frontmatter.Get(pCtx)
	if fm == nil {
		return nil, fmt.Errorf("flow does not contain any fields")
	}

	if err := fm.Decode(f); err != nil {
		return nil, fmt.Errorf("unable to decode flow: %w", err)
	}

	flowDirName := filepath.Base(filepath.Dir(path))
	fileName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	f.dirName = flowDirName
	f.fileName = fileName
	f.markdown = ast
	f.ID = fmt.Sprintf("%s.%s", flowDirName, fileName)

	if f.Worker == "" {
		f.Worker = f.ID
	}

	// Convert relative worker names to absolute ones
	if !strings.Contains(f.Worker, ".") {
		f.Worker = fmt.Sprintf("%s.%s", flowDirName, f.Worker)
	}

	if f.If != "" {
		expr, diags := hclsyntax.ParseExpression([]byte(f.If), path, hcl.InitialPos)
		if diags.HasErrors() {
			return nil, errors.Join(diags.Errs()...)
		}

		f.ifExpression = expr
	}

	if err := flowValidator.validate.Struct(f); err != nil {
		return nil, err
	}

	return f, nil
}

func (fr *FlowReader) indexFlow(flow *Flow) error {
	// Create index for sensor
	// Convert the `on` statement from shorthand syntax to full
	var on string
	splitOn := strings.Split(flow.On, ".")
	switch len(splitOn) {
	case 1:
		on = fmt.Sprintf("*.%s.*", flow.On)
	case 2:
		on = fmt.Sprintf("*.%s", flow.On)
	case 3:
		on = flow.On
	default:
		return fmt.Errorf("invalid 'on' field, must be defined and have no more than three parts")
	}
	fr.indexSensor(on, flow)

	// Create index for command
	if flow.Command != nil {
		fr.index.Commands[flow.ActionName()] = flow
		fr.indexSensor(fmt.Sprintf("*.command.%s", flow.ActionName()), flow)
	}

	// Create index for schedule
	if flow.Schedule != "" {
		fr.index.Schedules = append(fr.index.Schedules, flow)
		fr.indexSensor(fmt.Sprintf("hiphops.schedule.%s", flow.ActionName()), flow)
	}

	return nil
}

func (fr *FlowReader) indexSensor(index string, flow *Flow) {
	indexFlows, ok := fr.index.Sensors[index]
	if !ok {
		fr.index.Sensors[index] = []*Flow{flow}
		return
	}

	fr.index.Sensors[index] = append(indexFlows, flow)
}

// ActionName returns the name of the action for events generated by this flow
//
// Note: schedules and commands both generate events
func (f *Flow) ActionName() string {
	if strings.ToLower(f.fileName) == "index" {
		return f.dirName
	}

	return strings.ReplaceAll(f.ID, ".", "-")
}

func (f *Flow) IfValue(evalCtx *hcl.EvalContext) (bool, error) {
	if f.If == "" {
		return true, nil
	}

	ifVal, diags := f.ifExpression.Value(evalCtx)
	if diags.HasErrors() {
		return false, errors.Join(diags.Errs()...)
	}

	var matches bool
	err := gocty.FromCtyValue(ifVal, &matches)
	if err != nil {
		return false, fmt.Errorf("'if' expression must evaluate to true or false: %w", err)
	}

	return matches, nil
}

func (pi *ParamItem) Param() (string, Param) {
	for name, p := range *pi {
		if p.Type == "" {
			p.Type = "string"
		}
		// We only allow a single param per ParamItem, so return immediately
		return name, p
	}

	return "", Param{}
}
