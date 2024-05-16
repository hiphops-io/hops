package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-getter"
)

type InitCmd struct {
	Dir      string `arg:"positional" default:"." help:"path to Hiphops dir - defaults to current directory"`
	Template string `arg:"-t,--template" help:"URL or filepath of the project template" default:"github.com/hiphops-io/hops/deploy/project_template/"`
}

func (i *InitCmd) Run() error {
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("hiphops-tmpl-%d", time.Now().Unix()))
	defer func() {
		os.RemoveAll(tmpDir)
	}()

	if err := getter.Get(tmpDir, i.Template, getWithPWD); err != nil {
		return fmt.Errorf("unable to fetch project template: %w", err)
	}

	if err := copyTemplateDir(tmpDir, i.Dir); err != nil {
		return fmt.Errorf("unable to render template into dir: %w", err)
	}

	return nil
}

func copyTemplateDir(src, dest string) error {
	srcPath, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}

	return filepath.WalkDir(srcPath, func(path string, d fs.DirEntry, _ error) error {
		relativePath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relativePath)

		switch {
		case filepath.Ext(destPath) == ".tmpl":
			// In future we may want to render .tmpl files with go templating.
			// For now we just use it as a way to commit files that would otherwise
			// cause side effects (e.g. .gitignore .dockerignore)
			destPath = strings.TrimSuffix(destPath, ".tmpl")
			return copyFile(path, destPath)
		case d.IsDir():
			err := os.Mkdir(destPath, 0766)

			if os.IsExist(err) {
				return nil
			}

			return err
		default:
			return copyFile(path, destPath)
		}
	})
}

func copyFile(src, dest string) error {
	// Do not write files to destinations that are already populated
	w, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}

		return err
	}
	defer w.Close()

	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	_, err = io.Copy(w, r)
	return err
}

func getWithPWD(client *getter.Client) error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get current working directory: %w", err)
	}

	client.Pwd = pwd

	return nil
}
