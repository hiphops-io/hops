package dsl

import (
	"crypto/sha1"
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

type (
	HopsFiles struct {
		Hash        string
		BodyContent *hcl.BodyContent
		Files       []FileContent
	}

	FileContent struct {
		File    string `json:"file"`
		Content []byte `json:"content"`
	}
)

// ReadHopsFilePath loads and pre-parses the content of .hops files either from a
// single file or from all .hops files in a directory.
// It returns a merged hcl.Body and a sha hash of the contents
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
	sha1Hash := sha1.New()

	// parse the hops files
	for _, file := range hopsFileContent {
		hopsFile, diags := parser.ParseHCL(file.Content, file.File)

		if diags != nil && diags.HasErrors() {
			return nil, "", errors.New(diags.Error())
		}
		hopsBodies = append(hopsBodies, hopsFile.Body)

		sha1Hash.Write(file.Content)
	}

	body := hcl.MergeBodies(hopsBodies)
	content, diags := body.Content(HopSchema)
	if diags.HasErrors() {
		return nil, "", errors.New(diags.Error())
	}

	if len(content.Blocks) == 0 {
		return nil, "", errors.New("At least one resource must be defined in your hops config(s)")
	}

	filesSha := sha1Hash.Sum(nil)
	filesShaHex := hex.EncodeToString(filesSha)

	return content, filesShaHex, nil
}

func readHops(hopsPath string) ([]FileContent, error) {
	info, err := os.Stat(hopsPath)
	if err != nil {
		return nil, err
	}

	// read in the hops files and prepare for parsing
	if info.IsDir() {
		return readHopsDir(hopsPath)
	}

	content, err := os.ReadFile(hopsPath)
	if err != nil {
		return nil, err
	}

	files := []FileContent{{
		File:    hopsPath,
		Content: content,
	}}

	return files, nil
}

// readHopsDir retrieves the content of all .hops files in a directory,
// including sub directories, and returns then as a slice of fileContents
func readHopsDir(dirPath string) ([]FileContent, error) {
	filePaths := []string{}

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
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
		if d.IsDir() && strings.HasPrefix(d.Name(), "..") {
			return filepath.SkipDir
		}
		if !d.IsDir() && filepath.Ext(path) == ".hops" {
			filePaths = append(filePaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort the file paths to ensure consistent order
	sort.Strings(filePaths)

	files := []FileContent{}

	// Read and store filename and content of each file
	for _, filePath := range filePaths {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		files = append(files, FileContent{
			File:    filePath,
			Content: content,
		})
	}

	return files, nil
}
