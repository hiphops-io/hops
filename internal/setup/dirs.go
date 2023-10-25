package setup

import (
	"os"
	"path/filepath"
)

type AppDirs struct {
	RootDir      string
	WorkspaceDir string
}

func NewAppDirs(rootDir string) (AppDirs, error) {
	fm := os.FileMode(0744)

	appdirs := AppDirs{
		RootDir: rootDir,
	}

	err := ensureSubDir(&appdirs.WorkspaceDir, rootDir, "workspace", fm)
	if err != nil {
		return appdirs, err
	}

	return appdirs, nil
}

func ensureSubDir(update *string, rootDir string, subDir string, fm os.FileMode) error {
	path := filepath.Join(rootDir, subDir)

	err := os.MkdirAll(path, fm)
	if err != nil {
		return err
	}

	*update = path

	return nil
}
