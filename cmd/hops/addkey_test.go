package main

// import (
// 	"os"
// 	"path/filepath"
// 	"syscall"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// )

// func TestAddKeyfile(t *testing.T) {
// 	// Get the current umask
// 	originalUmask := syscall.Umask(0)
// 	syscall.Umask(originalUmask)

// 	// Define the expected permissions, accounting for the umask
// 	expectedPerms := os.FileMode(0666) &^ os.FileMode(originalUmask)

// 	// Test successful overwriting of an existing file
// 	t.Run("overwrite existing file", func(t *testing.T) {
// 		tempDir := t.TempDir()
// 		filePath := filepath.Join(tempDir, "testfile.txt")

// 		initialContent := []byte("initial content")
// 		err := addOrUpdateKeyfile(filePath, initialContent)
// 		if err != nil {
// 			t.Fatal("Failed to write file with initial content:", err)
// 		}

// 		newContent := []byte("new content")
// 		err = addOrUpdateKeyfile(filePath, newContent)
// 		if err != nil {
// 			t.Fatal("Failed to overwrite existing file with new content:", err)
// 		}

// 		contents, err := os.ReadFile(filePath)
// 		if err != nil {
// 			t.Fatal("Failed to read file contents:", err)
// 		}
// 		assert.Equal(t, newContent, contents)

// 		fileInfo, err := os.Stat(filePath)
// 		if err != nil {
// 			t.Fatal("Failed to get file info:", err)
// 		}
// 		assert.Equal(t, fileInfo.Mode().Perm(), expectedPerms, "File permissions should be as expected considering umask")
// 	})

// 	// Test creation and writing to a new file if it doesn't exist
// 	// Kind of covered in previous test, but good to have a specific test
// 	t.Run("create and write new file", func(t *testing.T) {
// 		tempDir := t.TempDir()
// 		filePath := filepath.Join(tempDir, "newfile.txt")

// 		content := []byte("new file content")
// 		err := addOrUpdateKeyfile(filePath, content)
// 		if err != nil {
// 			t.Fatal("Failed to write file with new content:", err)
// 		}

// 		contents, err := os.ReadFile(filePath)
// 		if err != nil {
// 			t.Fatal("Failed to read file contents:", err)
// 		}
// 		assert.Equal(t, content, contents)

// 		fileInfo, err := os.Stat(filePath)
// 		if err != nil {
// 			t.Fatal("Failed to get file info:", err)
// 		}
// 		assert.Equal(t, fileInfo.Mode().Perm(), expectedPerms, "File permissions should be 0660")
// 	})

// 	// Test creation of directory and file if they don't exist
// 	t.Run("create directory and file", func(t *testing.T) {
// 		tempDir := t.TempDir()
// 		newDir := filepath.Join(tempDir, "newdir")
// 		filePath := filepath.Join(newDir, "newfile.txt")

// 		content := []byte("content for new file in new directory")
// 		err := addOrUpdateKeyfile(filePath, content)
// 		if err != nil {
// 			t.Fatal("Failed to create directory and file:", err)
// 		}

// 		// Check if directory was created
// 		_, err = os.Stat(newDir)
// 		if err != nil {
// 			t.Fatal("Failed to stat the new directory:", err)
// 		}

// 		// Check if file was created in the new directory
// 		contents, err := os.ReadFile(filePath)
// 		if err != nil {
// 			t.Fatal("Failed to read file contents:", err)
// 		}
// 		assert.Equal(t, content, contents)

// 		fileInfo, err := os.Stat(filePath)
// 		if err != nil {
// 			t.Fatal("Failed to get file info:", err)
// 		}
// 		assert.Equal(t, fileInfo.Mode().Perm(), expectedPerms, "File permissions should be as expected considering umask")
// 	})

// 	// Test invalid input (e.g., directory path)
// 	t.Run("invalid input - directory path", func(t *testing.T) {
// 		tempDir := t.TempDir()
// 		err := addOrUpdateKeyfile(tempDir, []byte("content"))
// 		assert.Error(t, err)
// 	})
// }
