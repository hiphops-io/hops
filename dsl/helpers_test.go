// This file contains test helpers/utils for testing the DSL.
package dsl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hiphops-io/hops/logs"
)

// createTmpHopsFile creates a temporary hops file in a subdirectory
// with the given content and returns the parsed HCL body content
func createTmpHopsFile(t *testing.T, content string) (*HopsFiles, error) {
	logger := logs.NoOpLogger()

	// temporary directory
	tmpDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFilename := filepath.Join(tmpDir, "hopsdir/hops.hops")
	if err := os.MkdirAll(filepath.Dir(tmpFilename), 0755); err != nil {
		t.Fatalf("Failed to create subdirectory for file %s: %s", tmpFilename, err)
	}
	err = os.WriteFile(tmpFilename, []byte(content), 0666)
	if err != nil {
		t.Fatalf("Failed to write to temp file %s: %s", tmpFilename, err)
	}

	hops, err := ReadHopsFilePath(tmpDir, logger)
	if err != nil {
		return nil, err
	}

	return hops, nil
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
