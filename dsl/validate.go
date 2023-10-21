package dsl

import (
	"fmt"
	"regexp"
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
