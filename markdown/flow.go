package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty/gocty"
	"go.abhg.dev/goldmark/frontmatter"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type (
	Command []ParamItem

	Flow struct {
		If       string  `yaml:"if"`
		Command  Command `yaml:"command" validate:"required_without_all=On Schedule,omitempty,command"`
		On       string  `yaml:"on" validate:"required_without_all=Command Schedule"`
		Schedule string  `yaml:"schedule" validate:"required_without_all=On Command,omitempty,standard_cron"`
		Worker   string  `yaml:"worker"`
		// Computed fields
		ID           string
		dirName      string
		fileName     string
		ifExpression hcl.Expression
		markdown     []byte
		md           *Markdown
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
		md:       NewMarkdown(),
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
	content, pCtx, err := fr.md.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to load flow: %w", err)
	}

	flowDirName := filepath.Base(filepath.Dir(path))
	fileName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	f := &Flow{
		ID:       fmt.Sprintf("%s.%s", flowDirName, fileName),
		path:     path,
		markdown: content,
		md:       fr.md,
		dirName:  flowDirName,
		fileName: fileName,
	}

	fm := frontmatter.Get(pCtx)
	if fm == nil {
		return nil, fmt.Errorf("flow does not contain any fields")
	}

	if err := fm.Decode(f); err != nil {
		return nil, fmt.Errorf("unable to decode flow: %w", err)
	}

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

func (f *Flow) DisplayName() string {
	if strings.ToLower(f.fileName) == "index" {
		return titleCase(f.dirName)
	}

	return titleCase(f.ID)
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

func (f *Flow) Markdown() (string, error) {
	var b bytes.Buffer
	if _, err := f.md.Markdown(f.markdown, &b); err != nil {
		return "", err
	}

	return b.String(), nil
}

func (pi *ParamItem) DisplayName() string {
	for name := range *pi {
		return titleCase(name)
	}

	return ""
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

var titleCaseReplacer = strings.NewReplacer("_", " ", ".", " ")

func titleCase(label string) string {
	caser := cases.Title(language.BritishEnglish)
	label = titleCaseReplacer.Replace(label)
	return caser.String(label)
}
