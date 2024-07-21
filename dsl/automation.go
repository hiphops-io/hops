package dsl

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"

	"github.com/hiphops-io/hops/dsl/ctyconv"
)

const (
	FileTypeHops FileType = iota
	FileTypeManifest
	FileTypeOther
)

var ManifestNames = []string{"manifest.yaml", "manifest.yml"}

type (
	Automations struct {
		Files     map[string][]byte
		Hash      string
		Hops      *HopsAST
		Manifests map[string]*Manifest
	}

	AutomationFile struct {
		Path    string
		Content []byte
	}

	FileType int

	Hops struct {
		Ons []*OnAST
		*HopsAST
	}

	On struct {
		*OnAST
	}
)

func NewAutomations(files []*AutomationFile) (*Automations, hcl.Diagnostics) {
	a := &Automations{
		Files:     map[string][]byte{},
		Manifests: map[string]*Manifest{},
	}

	d := hcl.Diagnostics{}

	if len(files) == 0 {
		return a, d
	}

	sha := sha256.New()
	bodies := []hcl.Body{}
	parser := hclparse.NewParser()

	for _, f := range files {
		sha.Write(f.Content)

		switch automationFileType(f.Path) {
		case FileTypeHops:
			hclFile, diags := parser.ParseHCL(f.Content, f.Path)
			d = d.Extend(diags)
			bodies = append(bodies, hclFile.Body)

		case FileTypeManifest:
			manifest, err := NewManifest(f.Content)
			if err != nil {
				d = d.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid automation manifest",
					Detail:   fmt.Sprintf("Failed to parse manifest: %s", err.Error()),
					Subject: &hcl.Range{
						Filename: f.Path,
						Start:    hcl.InitialPos,
						End:      hcl.InitialPos,
					},
				})
			}

			a.Manifests[filepath.Dir(f.Path)] = manifest
		}

		a.Files[f.Path] = f.Content
	}

	// We use a basic evaluation context to decode the AST initially
	// (without any event variables since we don't have it yet)
	evaluationCtx := NewEvaluationCtx(a, nil)
	body := hcl.MergeBodies(bodies)
	hops, diags := DecodeToHopsAST(body, evaluationCtx)
	d = d.Extend(diags)

	a.Hops = hops
	a.Hash = hex.EncodeToString(sha.Sum(nil))
	return a, d
}

func NewAutomationsFromContent(contents map[string][]byte) (*Automations, hcl.Diagnostics) {
	files := []*AutomationFile{}

	// Read and store filename and content of each file
	for path, content := range contents {
		files = append(files, &AutomationFile{
			Path:    path,
			Content: content,
		})
	}

	return NewAutomations(files)
}

func NewAutomationsFromDir(dirPath string) (*Automations, hcl.Diagnostics, error) {
	files, err := readAutomationDir(dirPath)
	if err != nil {
		return nil, nil, err
	}

	a, d := NewAutomations(files)

	return a, d, nil
}

func (a *Automations) EventOns(eventData []byte) ([]*On, hcl.Diagnostics) {
	d := hcl.Diagnostics{}
	ons := []*On{}

	if a.Hops == nil {
		return nil, d
	}

	eventVal, err := ctyconv.JSONToCtyValue(eventData)
	if err != nil {
		d = d.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unable to parse source event",
			Detail:   fmt.Sprintf("An error occurred parsing the event: %s", err.Error()),
		})
		return nil, d
	}

	ctxVars := map[string]cty.Value{"event": eventVal}
	evaluationCtx := NewEvaluationCtx(a, ctxVars)

	// Get the event name/action from the bundle
	event, action, err := ctyconv.CtyToEventAction(eventVal, HopsMetaKey)
	if err != nil {
		return nil, hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Unable to parse source event: %s", err.Error()),
			},
		}
	}

	// Fetch the on blocks that listen for this event/action from the index
	eventActionKey := fmt.Sprintf("%s_%s", event, action)
	// Concat over append since we can't afford to muck up the eventIndex slices
	eventOns := slices.Concat(a.Hops.eventIndex[eventActionKey], a.Hops.eventIndex[event])

	for _, onAST := range eventOns {
		blockEval := evaluationCtx.BlockScopedEvalContext(onAST.block, onAST.Slug)

		ifVal, diags := EvaluateBoolExpression(onAST.IfExpr, true, blockEval)
		if diags.HasErrors() {
			d = d.Extend(diags)
			continue
		}

		if !ifVal {
			continue
		}

		on := &On{
			OnAST: onAST,
		}

		ons = append(ons, on)
	}

	return ons, d
}

func (a *Automations) GetSchedules() []*ScheduleAST {
	return a.Hops.Schedules
}

func (a *Automations) GetTask(label string) (*TaskAST, error) {
	if a.Hops == nil {
		return &TaskAST{}, fmt.Errorf("No hops files are loaded")
	}

	// NOTE: This currently searches all tasks rather than map lookup. Improve in future
	for _, task := range a.Hops.Tasks {
		if task.Name == label {
			return task, nil
		}
	}

	return &TaskAST{}, fmt.Errorf("Task '%s' not found", label)
}

func (a *Automations) GetTasks() []*TaskAST {
	if a.Hops == nil {
		return []*TaskAST{}
	}
	return a.Hops.Tasks
}

func (a *Automations) GetTasksInPath(path string) []*TaskAST {
	if a.Hops == nil {
		return []*TaskAST{}
	}

	pathTasks := []*TaskAST{}

	for _, task := range a.Hops.Tasks {
		if !strings.HasPrefix(task.FilePath, path) {
			continue
		}

		pathTasks = append(pathTasks, task)
	}

	return pathTasks
}

func NewManifest(content []byte) (*Manifest, error) {
	m := &Manifest{
		Emoji:        "⚪️",
		RequiredApps: []string{},
	}

	err := yaml.Unmarshal(content, m)
	if err != nil {
		return nil, err
	}

	err = valid.validate.Struct(m)

	return m, err
}

// readAutomationDir loads the content of automations from a directory
func readAutomationDir(dirPath string) ([]*AutomationFile, error) {
	filePaths, err := automationDirFilePaths(dirPath)
	if err != nil {
		return nil, err
	}

	files := []*AutomationFile{}

	// Read and store filename and content of each file
	for _, filePath := range filePaths {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		relativePath, err := filepath.Rel(dirPath, filePath)
		if err != nil {
			return nil, err
		}

		files = append(files, &AutomationFile{
			Path:    relativePath,
			Content: content,
		})
	}

	if len(files) == 0 {
		return nil, errors.New("No flows have been defined")
	}

	return files, nil
}

// automationDirFilePaths returns a slice of all the file paths of files
// in the first child subdirectories of the root directory.
//
// Excludes dirs with '..' prefix as these cause problems with kubernetes.
func automationDirFilePaths(root string) ([]string, error) {
	var filePaths []string // list of file paths to be returned at the end (hops and other)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from the root
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Exclude directories whose name starts with '..'
			// Kubernetes config maps create a set of symlinked
			// directories starting with '..' and we don't want to pick those up.
			if strings.HasPrefix(d.Name(), "..") {
				return filepath.SkipDir
			}

			// Only look one sub dir deep
			if strings.Count(relativePath, string(filepath.Separator)) > 1 {
				return filepath.SkipDir
			}

			return nil
		}

		// Symlinks to dirs are not seen as dirs by filepath.WalkDir,
		// so we need to check and exclude them as well
		if strings.HasPrefix(d.Name(), "..") {
			return nil
		}

		// Files in root (i.e root/a.hops), and anything other than first
		// child directory of the root (i.e. root/sub/sub/a.hops) are skipped
		if strings.Count(relativePath, string(filepath.Separator)) != 1 {
			return nil
		}

		// Add file to list
		filePaths = append(filePaths, path)

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort the file paths to ensure consistent order
	sort.Strings(filePaths)

	return filePaths, nil
}

func automationFileType(path string) FileType {
	path = strings.ToLower(path)

	if filepath.Ext(path) == ".hops" {
		return FileTypeHops
	}

	if slices.Contains(ManifestNames, filepath.Base(path)) {
		return FileTypeManifest
	}

	return FileTypeOther
}
