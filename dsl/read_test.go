package dsl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hiphops-io/hops/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcatenateHopsFiles(t *testing.T) {
	tests := []struct {
		name              string
		files             map[string]string // filename -> content
		expectedFilePaths []string
		expectedFileTypes []string
		expectError       bool
	}{
		{
			name:              "No files",
			files:             map[string]string{},
			expectedFilePaths: []string{},
			expectedFileTypes: []string{},
			expectError:       false,
		},
		{
			name: "No root dir",
			files: map[string]string{
				"a.hops": "content of a",
			},
			expectedFilePaths: []string{},
			expectedFileTypes: []string{},
			expectError:       false,
		},
		{
			name: "Single file",
			files: map[string]string{
				"hops1/a.hops": "content of a",
			},
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
			expectedFilePaths: []string{"hopsa/a.hops", "hopsb/b.hops"},
			expectedFileTypes: []string{HopsFile, HopsFile},
			expectError:       false,
		},
		{
			// DO NOT CHANGE without making sure the `file` built in function still works.
			// The `file` function depends on these files being sorted alphabetically.
			name: "Files sorted by filename/directory",
			files: map[string]string{
				"hopsb/b.hops": "content of b",
				"hopsa/a.hops": "content of a",
			},
			expectedFilePaths: []string{"hopsa/a.hops", "hopsb/b.hops"},
			expectedFileTypes: []string{HopsFile, HopsFile},
			expectError:       false,
		},
		{
			name: "Return all files, but typed as HopsFile or OtherFile",
			files: map[string]string{
				"hopsa/a.hops": "content of a",
				"hopsa/b.txt":  "content of b",
				"hopsc/c.hops": "content of c",
			},
			expectedFilePaths: []string{"hopsa/a.hops", "hopsa/b.txt", "hopsc/c.hops"},
			expectedFileTypes: []string{HopsFile, OtherFile, HopsFile},
			expectError:       false,
		},
		{
			name: "First subdirectories searched but not root and not second level",
			files: map[string]string{
				"subdir/b.hops":          "content of b",
				"subdir/evenmore/b.hops": "content of b",
				"a.hops":                 "content of a",
			},
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
			expectedFilePaths: []string{"subd/d.hops"},
			expectedFileTypes: []string{HopsFile},
			expectError:       false,
		},
		{
			name: "Multiple hops files per subdir",
			files: map[string]string{
				"sub1/a.hops": "content of a",
				"sub1/b.hops": "content of b",
			},
			expectedFilePaths: []string{"sub1/a.hops", "sub1/b.hops"},
			expectedFileTypes: []string{HopsFile, HopsFile},
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

func TestGetOtherFiles(t *testing.T) {
	logger := logs.NoOpLogger()
	hopsFiles, err := ReadHopsFilePath("./testdata/valid", logger)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(hopsFiles.Files))
	assert.Equal(t, "valid/additional.txt", hopsFiles.Files[0].File)
	assert.Equal(t, OtherFile, hopsFiles.Files[0].Type)

	assert.Equal(t, "valid/valid.hops", hopsFiles.Files[1].File)
	assert.Equal(t, HopsFile, hopsFiles.Files[1].Type)
}

// Exclude directories, symlinks and files whose name starts with '..'
// This is because kubernetes configMaps create a set of symlinked
// directories for the mapped files and we don't want to pick those
// up. Those directories are named '..<various names>'
// Example:
// /automations
// |-- main.hops -> ..data/main.hops
// |-- ..data -> ..2024_01_10_10_32_09.1478597074
// |-- ..2024_01_10_10_32_09.1478597074
// |   |-- main.hops
func TestKubernetesHopsStructure(t *testing.T) {
	// Create a temporary directory to mimic ~/hops-conf/automations
	baseDir, err := os.MkdirTemp("", "hops-conf")
	if err != nil {
		t.Fatalf("Failed to create base directory: %s", err)
	}
	defer os.RemoveAll(baseDir)

	automationsDir := filepath.Join(baseDir, "automations")
	err = os.MkdirAll(automationsDir, 0777)
	if err != nil {
		t.Fatalf("Failed to create automations directory: %s", err)
	}

	// Creating the dated directory
	datedDir := filepath.Join(automationsDir, "..2024_01_10_10_32_09.1478597074")
	err = os.Mkdir(datedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create dated directory: %s", err)
	}

	// Create main.hops file in datedDir
	mainHopsFile := filepath.Join(datedDir, "main.hops")
	file, err := os.Create(mainHopsFile)
	if err != nil {
		t.Fatalf("Failed to create main.hops file: %s", err)
	}
	file.Close()

	// Creating the symbolic link ..data
	dataLink := filepath.Join(automationsDir, "..data")
	err = os.Symlink(datedDir, dataLink)
	if err != nil {
		t.Fatalf("Failed to create data symlink: %s", err)
	}

	// Creating the symbolic link main.hops
	mainHopsLink := filepath.Join(automationsDir, "main.hops")
	targetMainHops := filepath.Join(dataLink, "main.hops")
	err = os.Symlink(targetMainHops, mainHopsLink)
	if err != nil {
		t.Fatalf("Failed to create main.hops symlink: %s", err)
	}

	// Run the function
	resultFileContent, err := readHops(baseDir)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resultFileContent))
	assert.Equal(t, "automations/main.hops", resultFileContent[0].File)
}

func TestReadValidationOn(t *testing.T) {
	type testCase struct {
		name  string
		hops  string
		valid bool
	}

	tests := []testCase{
		{
			name:  "Simple valid config",
			hops:  `on foo {}`,
			valid: true,
		},
		{
			name: "Expansive valid config",
			hops: `
			on anevent_action {}

			on anevent {
				name = "pipeline"
			}

			on anevent {
				name = "pipeline"

				if = true != false
			}

			on anevent {
				call app_handler {
					name = "first_call"

					if = true

					inputs = {
						foo = "bar"
					}
				}

				done {
					errored = first_call.errored
					completed = first_call.completed
				}
			}
			`,
			valid: true,
		},
		{
			name: "Unknown root attribute",
			hops: `
			on foo {}
			an_unknown_attr = "value"
			`,
			valid: false,
		},
		{
			name:  "Too many labels",
			hops:  `on foo bar {}`,
			valid: false,
		},
		// TODO: The below tests do not currently work, as we only read with a basic
		// schema containing top level attributes/blocks and their expected labels.
		// Leaving here as they describe the behaviour we want to achieve in future.
		// {
		// 	name: "Unknown attribute",
		// 	hops: `on foo {
		// 		an_unknown_attr = "value"
		// 	}`,
		// 	valid: false,
		// },
		// {
		// 	name: "Unknown block",
		// 	hops: `on foo {
		// 		an_unknown_block {
		// 			val = "val"
		// 		}
		// 	}`,
		// 	valid: false,
		// },
		// {
		// 	name: "Inputs defined as block",
		// 	hops: `on foo {
		// 		inputs {
		// 			value = "hey"
		// 		}
		// 	}`,
		// 	valid: false,
		// },
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := createTmpHopsFile(t, tc.hops)
			if !tc.valid {
				assert.Error(t, err)
				return
			}
			if tc.valid {
				assert.NoError(t, err)
			}
		})
	}
}

func extractFileFields(fileContents []FileContent) []string {
	var files []string
	for _, fc := range fileContents {
		files = append(files, fc.File)
	}
	return files
}
