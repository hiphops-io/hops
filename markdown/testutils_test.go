package markdown

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupPopulatedTestDir(t *testing.T, source map[string][]byte) string {
	sourceDir := t.TempDir()
	for relPath, content := range source {
		// Ensure the file's dir exists (in case of nested dir structures)
		path := filepath.Join(sourceDir, relPath)
		fileDir := filepath.Dir(path)

		err := os.MkdirAll(fileDir, os.ModePerm)
		require.NoError(t, err, "Test setup: Unable to create test dir")

		err = os.WriteFile(path, content, os.ModePerm)
		require.NoError(t, err, "Test setup: Unable to write test dir file")
	}

	return sourceDir
}
