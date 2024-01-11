package dsl

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

const (
	HopsExt   = ".hops"
	HopsFile  = "hops"
	OtherFile = "other"
)

type (
	HopsFiles struct {
		Hash        string
		BodyContent *hcl.BodyContent
		Files       []FileContent // Sorted by file name `File`
	}

	FileContent struct {
		File    string `json:"file"`
		Content []byte `json:"content"`
		Type    string `json:"type"`
	}
)

// LookupFile searches for a file in the HopsFiles struct and returns a
// reference to the file and true if found, or nil and false if not found.
func (h *HopsFiles) LookupFile(filePath string) (*FileContent, bool) {
	// Binary search since filePaths are sorted
	i := sort.Search(len(h.Files), func(i int) bool {
		return h.Files[i].File >= filePath
	})
	if i < len(h.Files) && h.Files[i].File == filePath {
		return &h.Files[i], true
	}

	return nil, false
}

// ReadHopsFilePath loads and pre-parses the content of .hops files from all
// .hops files in the first child sub directories.
//
// It returns a merged hcl.Body and a sha hash of the contents as well as
// a slice of FileContent structs containing the file name, content and type.
func ReadHopsFilePath(filePath string) (*HopsFiles, error) {
	files, err := readHops(filePath)
	if err != nil {
		return nil, err
	}

	content, hash, err := ReadHopsFileContents(files)
	if err != nil {
		return nil, err
	}

	hopsFiles := &HopsFiles{
		Hash:        hash,
		BodyContent: content,
		Files:       files,
	}

	return hopsFiles, nil
}

func ReadHopsFileContents(hopsFileContent []FileContent) (*hcl.BodyContent, string, error) {
	hopsBodies := []hcl.Body{}
	parser := hclparse.NewParser()
	sha256Hash := sha256.New()

	// parse the hops files
	for _, file := range hopsFileContent {
		// Add all file contents to the hash
		sha256Hash.Write(file.Content)

		// Do not parse non-hops files
		if file.Type != HopsFile {
			continue
		}

		hopsFile, diags := parser.ParseHCL(file.Content, file.File)

		if diags != nil && diags.HasErrors() {
			return nil, "", errors.New(diags.Error())
		}
		hopsBodies = append(hopsBodies, hopsFile.Body)
	}

	body := hcl.MergeBodies(hopsBodies)
	content, diags := body.Content(HopSchema)
	if diags.HasErrors() {
		return nil, "", errors.New(diags.Error())
	}

	if len(content.Blocks) == 0 {
		return nil, "", errors.New("Ensure --hops is set to a valid dir containing automations. A valid automation must include at least one non-empty *.hops file")
	}

	filesSha := sha256Hash.Sum(nil)
	filesShaHex := hex.EncodeToString(filesSha)

	return content, filesShaHex, nil
}

// getHopsDirFilePaths returns a slice of all the file paths of files
// in the first child subdirectories of the root directory.
//
// Excludes dirs with '..' prefix as these cause problems with kubernetes.
func getHopsDirFilePaths(root string) ([]string, error) {
	var filePaths []string // list of file paths to be returned at the end (hops and other)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from the root
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Exclude directories whose name starts with '..'
			// This is because kubernetes configMaps create a set of symlinked
			// directories for the mapped files and we don't want to pick those
			// up. Those directories are named '..<various names>'
			// Example:
			// /my-config-map-dir
			// |-- my-key -> ..data/my-key
			// |-- ..data -> ..2023_10_19_12_34_56.789012345
			// |-- ..2023_10_19_12_34_56.789012345
			// |   |-- my-key
			if strings.HasPrefix(d.Name(), "..") {
				return filepath.SkipDir
			}

			// Skip any second children of the root (i.e. root/sub, yes, root/sub/sub, no)
			if strings.Count(relativePath, string(filepath.Separator)) > 1 {
				return filepath.SkipDir
			}

			return nil
		}

		// Symlinks to dirs are not seen as dirs by filepath.WalkDir, so we need to
		// check and exclude them as well
		// TODO walk symlinks if top level directory is a symlink
		if strings.HasPrefix(d.Name(), "..") {
			return nil
		}
		// Files in root (i.e root/a.hops), and anything other than first
		// child directory of the root (i.e. root/sub/sub/a.hops) are skipped
		if strings.Count(relativePath, string(filepath.Separator)) != 1 {
			return nil
		}

		// Add file to list (both .hops and other files)
		filePaths = append(filePaths, path)

		return nil
	})
	// File walking is over, check for errors
	if err != nil {
		return nil, err
	}

	// Sort the file paths to ensure consistent order
	sort.Strings(filePaths)

	return filePaths, nil
}

// readHops retrieves the content of all .hops and other files
//
// reads from first child subdirectories of dirPath (excluding dirs with '..'
// prefix) and returns them as a slice of fileContents
func readHops(dirPath string) ([]FileContent, error) {
	filePaths, err := getHopsDirFilePaths(dirPath)
	if err != nil {
		return nil, err
	}

	files := []FileContent{}

	// Read and store filename and content of each file
	for _, filePath := range filePaths {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		relativePath, err := filepath.Rel(dirPath, filePath)
		if err != nil {
			return nil, err
		}

		fileType := OtherFile
		if filepath.Ext(relativePath) == HopsExt {
			fileType = HopsFile
		}

		files = append(files, FileContent{
			File:    relativePath,
			Content: content,
			Type:    fileType,
		})
	}

	return files, nil
}
