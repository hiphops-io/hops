package funcs

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	type testCase struct {
		name        string
		template    string
		expectMatch string
		valid       bool
	}

	tests := []testCase{
		{
			name:        "Valid calver",
			template:    "[calver]",
			expectMatch: `^[0-9]{4}\.[0-9]{2}\.[0-9]{2}$`,
			valid:       true,
		},
		{
			name:        "Valid yyyy",
			template:    "[yyyy]",
			expectMatch: "^[0-9]{4}$",
			valid:       true,
		},
		{
			name:        "Valid yy",
			template:    "[yy]",
			expectMatch: "^[0-9]{2}$",
			valid:       true,
		},
		{
			name:        "Valid mm",
			template:    "[mm]",
			expectMatch: "^[0-9]{2}$",
			valid:       true,
		},
		{
			name:        "Valid m",
			template:    "[m]",
			expectMatch: "^[0-9]{1,2}$",
			valid:       true,
		},
		{
			name:        "Valid dd",
			template:    "[dd]",
			expectMatch: "^[0-9]{2}$",
			valid:       true,
		},
		{
			name:        "Valid d",
			template:    "[d]",
			expectMatch: "^[0-9]{1,2}$",
			valid:       true,
		},
		{
			name:        "Valid petname",
			template:    "[pet]",
			expectMatch: "^[a-z]+$",
			valid:       true,
		},
		{
			name:        "Valid adjective",
			template:    "[adj]",
			expectMatch: "^[a-z]+$",
			valid:       true,
		},
		{
			name:        "Valid adverb",
			template:    "[adv]",
			expectMatch: "^[a-z]+$",
			valid:       true,
		},
		{
			name:        "Unknown template var",
			template:    "[flib]",
			expectMatch: `^\[flib\]$`,
			valid:       true,
		},
		{
			name:        "Invalid version template",
			template:    "My version [",
			expectMatch: "",
			valid:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, err := TemplateVersion(tc.template)
			if !tc.valid {
				assert.Error(t, err, "Invalid version template should return error")
				return
			}

			require.NoError(t, err, "Valid version template should not return error")

			matched, err := regexp.MatchString(tc.expectMatch, v)
			require.NoError(t, err, "Test setup: Invalid regex for testing results")
			assert.Truef(t, matched, "%s does not match expected regex %s", v, tc.expectMatch)
		})
	}
}
