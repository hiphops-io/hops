package dsl

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

const MaxLabelLength = 50

var labelRegex = regexp.MustCompile(`^[a-z\d][a-z\d]*(?:_[a-z\d]+)*$`)

func ValidateLabels(labels ...string) error {
	for _, label := range labels {
		if len(label) > MaxLabelLength {
			return fmt.Errorf(`Label "%s" is too long. (Max 50 characters)`, label)
		}
		if len(label) == 0 {
			return fmt.Errorf(`Label "%s" is empty.`, label)
		}

		if !labelRegex.MatchString(label) {
			return fmt.Errorf(`Invalid label: "%s" Labels must be lowercase alphanumeric separated by underscores (Regex is %s)`, label, labelRegex.String())
		}
	}

	return nil
}

// ValidateHops - do not use. Alpha state
func ValidateHops(filePath string) (hcl.Diagnostics, error) {
	files, err := readHops(filePath)
	if err != nil {
		return nil, err
	}

	diags := hcl.Diagnostics{}
	parser := hclparse.NewParser()
	bodies := []hcl.Body{}

	for _, file := range files {
		if file.Type != HopsFile {
			continue
		}

		hclFile, diag := parser.ParseHCL(file.Content, file.File)
		diags = diags.Extend(diag)
		bodies = append(bodies, hclFile.Body)
	}

	body := hcl.MergeBodies(bodies)
	_, bodyDiags := body.Content(HopSchema)
	diags = diags.Extend(bodyDiags)

	return diags, nil
}
