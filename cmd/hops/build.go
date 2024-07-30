package main

import (
	"path/filepath"

	"github.com/hiphops-io/hops/markdown"
)

type BuildCmd struct {
	Dir          string `arg:"positional" default:"." help:"path to Hiphops dir [default: .]"`
	SkipFrontend bool   `arg:"--no-fe" help:"skip building the frontend"`
}

func (b *BuildCmd) Run() error {
	if !b.SkipFrontend {
		return buildSite(b.Dir)
	}

	return nil
}

func buildSite(rootDir string) error {
	sourceDir := filepath.Join(rootDir, "pages")
	buildDir := filepath.Join(rootDir, ".hiphops", "site")

	builder, err := markdown.NewStaticBuilder()
	if err != nil {
		return err
	}

	return builder.Build(sourceDir, buildDir)
}
