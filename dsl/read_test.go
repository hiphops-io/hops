package dsl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcatenateHopsFiles(t *testing.T) {
	tests := []struct {
		name              string
		files             map[string]string // filename -> content
		expectedContent   string
		expectedRows      int
		expectedFilePaths []string
		expectError       bool
	}{
		{
			name:              "No files",
			files:             map[string]string{},
			expectedContent:   "",
			expectedRows:      0,
			expectedFilePaths: []string{},
			expectError:       false,
		},
		{
			name: "Single file",
			files: map[string]string{
				"a.hops": "content of a",
			},
			expectedContent:   "content of a\n",
			expectedRows:      1,
			expectedFilePaths: []string{"a.hops"},
			expectError:       false,
		},
		{
			name: "Multiple files",
			files: map[string]string{
				"a.hops": "content of a",
				"b.hops": "content of b",
			},
			expectedContent:   "content of a\ncontent of b\n",
			expectedRows:      2,
			expectedFilePaths: []string{"a.hops", "b.hops"},
			expectError:       false,
		},
		{
			name: "Files sorted by filename",
			files: map[string]string{
				"b.hops": "content of b",
				"a.hops": "content of a",
			},
			expectedContent:   "content of a\ncontent of b\n",
			expectedRows:      2,
			expectedFilePaths: []string{"a.hops", "b.hops"},
			expectError:       false,
		},
		{
			name: "Only dot hops files",
			files: map[string]string{
				"a.hops": "content of a",
				"b.txt":  "content of b",
				"c.hops": "content of c",
			},
			expectedContent:   "content of a\ncontent of c\n",
			expectedRows:      2,
			expectedFilePaths: []string{"a.hops", "c.hops"},
			expectError:       false,
		},
		{
			name: "Subdirectories searched",
			files: map[string]string{
				"subdir/b.hops": "content of b",
				"a.hops":        "content of a",
			},
			expectedContent:   "content of a\ncontent of b\n",
			expectedRows:      2,
			expectedFilePaths: []string{"a.hops", "subdir/b.hops"},
			expectError:       false,
		},
		{
			name: "Multiple files subdirs sorted by name including subdirs",
			files: map[string]string{
				"a.hops":      "content of a",
				"sub2/b.hops": "content of b",
				"sub1/c.hops": "content of c",
				"sub3/d.hops": "content of d",
			},
			expectedContent:   "content of a\ncontent of c\ncontent of b\ncontent of d\n",
			expectedRows:      4,
			expectedFilePaths: []string{"a.hops", "sub1/c.hops", "sub2/b.hops", "sub3/d.hops"},
			expectError:       false,
		},
		{
			name: "Dont pick up leading double dot files/dirs",
			files: map[string]string{
				"..a.hops":          "content of a",
				"..sub3/b.hops":     "content of b",
				"sub4/..sub/c.hops": "content of c",
				"d.hops":            "content of d",
			},
			expectedContent:   "content of a\ncontent of d\n",
			expectedRows:      2,
			expectedFilePaths: []string{"..a.hops", "d.hops"},
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// temporary directory
			tmpDir, err := os.MkdirTemp("", "testdir")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %s", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create files in temp dir
			for filename, content := range tt.files {
				tmpFilename := filepath.Join(tmpDir, filename)

				// Create subdirs
				if err := os.MkdirAll(filepath.Dir(tmpFilename), 0755); err != nil {
					t.Fatalf("Failed to create subdirectory for file %s: %s", tmpFilename, err)
				}
				err := os.WriteFile(tmpFilename, []byte(content), 0666)
				if err != nil {
					t.Fatalf("Failed to write to temp file %s: %s", tmpFilename, err)
				}
			}

			// Run the function
			resultFileContent, err := readHopsDir(tmpDir)

			// Check for an unexpected error
			if !tt.expectError {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			// Compare the result with the expected file content
			assert.Equal(t, tt.expectedRows, len(resultFileContent))
			for i, fileContent := range resultFileContent {
				assert.Equal(t, tt.expectedFilePaths[i], fileContent.File)
			}
		})
	}
}
