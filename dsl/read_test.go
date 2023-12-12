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
		expectedFilePaths []string
		expectedFileTypes []string
		expectError       bool
	}{
		{
			name:              "No files",
			files:             map[string]string{},
			expectedContent:   "",
			expectedFilePaths: []string{},
			expectedFileTypes: []string{},
			expectError:       false,
		},
		{
			name: "No root dir",
			files: map[string]string{
				"a.hops": "content of a",
			},
			expectedContent:   "",
			expectedFilePaths: []string{},
			expectedFileTypes: []string{},
			expectError:       false,
		},
		{
			name: "Single file",
			files: map[string]string{
				"hops1/a.hops": "content of a",
			},
			expectedContent:   "content of a\n",
			expectedFilePaths: []string{"hops1/a.hops"},
			expectedFileTypes: []string{HopsFile},
			expectError:       false,
		},
		{
			name: "Multiple files",
			files: map[string]string{
				"hopsa/a.hops": "content of a",
				"hopsb/b.hops": "content of b",
			},
			expectedContent:   "content of a\ncontent of b\n",
			expectedFilePaths: []string{"hopsa/a.hops", "hopsb/b.hops"},
			expectedFileTypes: []string{HopsFile, HopsFile},
			expectError:       false,
		},
		{
			name: "Multiple files and ignore root dir",
			files: map[string]string{
				"hopsa/a.hops": "content of a",
				"hopsb/b.hops": "content of b",
				"c.hops":       "content of a",
			},
			expectedContent:   "content of a\ncontent of b\n",
			expectedFilePaths: []string{"hopsa/a.hops", "hopsb/b.hops"},
			expectedFileTypes: []string{HopsFile, HopsFile},
			expectError:       false,
		},
		{
			name: "Files sorted by filename/directory",
			files: map[string]string{
				"hopsb/b.hops": "content of b",
				"hopsa/a.hops": "content of a",
			},
			expectedContent:   "content of a\ncontent of b\n",
			expectedFilePaths: []string{"hopsa/a.hops", "hopsb/b.hops"},
			expectedFileTypes: []string{HopsFile, HopsFile},
			expectError:       false,
		},
		{
			name: "Only dot hops files",
			files: map[string]string{
				"hopsa/a.hops": "content of a",
				"hopsa/b.txt":  "content of b",
				"hopsc/c.hops": "content of c",
			},
			expectedContent:   "content of a\ncontent of c\n",
			expectedFilePaths: []string{"hopsa/a.hops", "hopsc/c.hops"},
			expectedFileTypes: []string{HopsFile, HopsFile},
			expectError:       false,
		},
		{
			name: "Subdirectories searched but not root",
			files: map[string]string{
				"subdir/b.hops":          "content of b",
				"subdir/evenmore/b.hops": "content of b",
				"a.hops":                 "content of a",
			},
			expectedContent:   "content of b\n",
			expectedFilePaths: []string{"subdir/b.hops"},
			expectedFileTypes: []string{HopsFile},
			expectError:       false,
		},
		{
			name: "Multiple files subdirs sorted by name including subdirs",
			files: map[string]string{
				"sub2/b.hops": "content of b",
				"sub1/c.hops": "content of c",
				"sub3/d.hops": "content of d",
			},
			expectedContent:   "content of c\ncontent of b\ncontent of d\n",
			expectedFilePaths: []string{"sub1/c.hops", "sub2/b.hops", "sub3/d.hops"},
			expectedFileTypes: []string{HopsFile, HopsFile, HopsFile},
			expectError:       false,
		},
		{
			name: "Dont pick up leading double dot files/dirs",
			files: map[string]string{
				"..a.hops":          "content of a",
				"..sub3/b.hops":     "content of b",
				"sub4/..sub/c.hops": "content of c",
				"subd/d.hops":       "content of d",
			},
			expectedContent:   "content of a\ncontent of d\n",
			expectedFilePaths: []string{"subd/d.hops"},
			expectedFileTypes: []string{HopsFile},
			expectError:       false,
		},
		{
			name: "Only one hops file per subdir",
			files: map[string]string{
				"sub1/a.hops": "content of a",
				"sub1/b.hops": "content of b",
			},
			expectedContent:   "",
			expectedFilePaths: []string{},
			expectedFileTypes: []string{},
			expectError:       true,
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
				createFile(t, tmpDir, filename, content)
			}

			// Run the function
			resultFileContent, err := readHops(tmpDir)

			// Check for an unexpected error
			if !tt.expectError {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				return
			}

			assert.Equal(
				t,
				len(tt.expectedFilePaths),
				len(resultFileContent),
				"Files returned don't match. Test: "+tt.name+"; Files returned: %#v",
				extractFileFields(resultFileContent),
			)
			if len(tt.expectedFilePaths) == len(resultFileContent) {
				for i, fileContent := range resultFileContent {
					assert.Equal(t, tt.expectedFilePaths[i], fileContent.File)
					assert.Equal(t, tt.expectedFileTypes[i], fileContent.Type)
				}
			}
		})
	}
}

// createFile creates a file in the given temp directory with the given content
// including any required subdirectories
func createFile(t *testing.T, tmpDir string, filename string, content string) {
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

func extractFileFields(fileContents []FileContent) []string {
	var files []string
	for _, fc := range fileContents {
		files = append(files, fc.File)
	}
	return files
}
