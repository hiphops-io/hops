package hops

import (
	"fmt"

	"github.com/goccy/go-json"
	"github.com/hashicorp/hcl/v2"

	"github.com/hiphops-io/hops/dsl"
)

type (
	// DiagnosticResult mirrors hcl.Diagnostic + json tags to control marshalling
	// We keep uppercase field names as this matches the runtime logged diagnostics
	DiagnosticResult struct {
		Severity    hcl.DiagnosticSeverity
		Summary     string
		Detail      string
		Subject     *hcl.Range
		Context     *hcl.Range
		Expression  hcl.Expression   `json:"-"`
		EvalContext *hcl.EvalContext `json:"-"`
		Extra       interface{}      `json:"Extra,omitempty"`
	}

	ValidationResult struct {
		Diagnostics []DiagnosticResult `json:"diagnostics"`
		FileCount   int                `json:"file_count"`
		IsValid     bool               `json:"is_valid"`
		NumIssues   int                `json:"num_issues"`
		ReadError   string             `json:"read_error,omitempty"`
	}
)

func ValidateDir(automationDir string, pretty bool) error {
	a, d, err := dsl.NewAutomationsFromDir(automationDir)

	diagResults := []DiagnosticResult{}

	for _, diag := range d {
		diagResults = append(diagResults, DiagnosticResult(*diag))
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
