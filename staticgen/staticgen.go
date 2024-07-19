// Package staticgen is a static site generator for hiphops
package staticgen

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.abhg.dev/goldmark/frontmatter"
)

//go:embed templates
var templates embed.FS

type (
	PageData struct {
		Content     template.HTML
		Description string `yaml:"description"`
		Title       string `yaml:"title"`
	}

	StaticBuilder struct {
		md           *Markdown
		pageTemplate *template.Template
	}
)

func NewStaticBuilder() (*StaticBuilder, error) {
	pageTemplate, err := template.ParseFS(templates, "templates/page.html")
	if err != nil {
		return nil, err
	}

	s := &StaticBuilder{
		md:           NewMarkdownHTML(),
		pageTemplate: pageTemplate,
	}

	return s, nil
}

func (s *StaticBuilder) Build(source, build string) error {
	if err := os.RemoveAll(build); err != nil {
		return err
	}
	// TODO: On error, delete all contents in build dir if any

	err := filepath.WalkDir(source, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		buildPath := filepath.Join(build, relPath)

		if de.IsDir() {
			return os.MkdirAll(buildPath, os.ModePerm)
		}

		return s.BuildFile(path, filepath.Dir(buildPath))
	})
	// Walk the source directory
	// - creating an equivalent structure in the build dir
	// - render markdown files into their html output files
	// - append generated headers/full html page structure
	// - - use frontmatter to help populate
	// - generate an index.html
	// - include default css file(s)
	// What about dynamic paths? Do they even exist?
	return err
}

func (s *StaticBuilder) BuildFile(path, build string) error {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".md":
		if err := s.BuildMarkdown(path, build); err != nil {
			return err
		}
	case ".css":
		// TODO: Copy CSS and add to header/create cache buster filename
	default:
		// Just straight copy
	}

	return nil
}

func (s *StaticBuilder) BuildMarkdown(source, build string) error {
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	var mdOutput bytes.Buffer
	pCtx, err := s.md.Convert(content, &mdOutput)
	if err != nil {
		return err
	}

	path := buildPath(source, build, ".html")
	writer, err := os.Create(path)
	if err != nil {
		return err
	}
	defer writer.Close()

	pageData := PageData{}

	if fm := frontmatter.Get(pCtx); fm != nil {
		if err := fm.Decode(&pageData); err != nil {
			return err
		}
	}

	pageData.Content = template.HTML(mdOutput.String())

	return s.pageTemplate.Execute(writer, pageData)
}

// buildPath converts a source file path into a path in the build dir, optionally
// altering the extension.
func buildPath(source, build, ext string) string {
	fileName := strings.ToLower(filepath.Base(source))
	buildPath := filepath.Join(build, fileName)

	if ext != "" {
		buildPath = strings.TrimSuffix(buildPath, filepath.Ext(buildPath)) + ext
	}

	return buildPath
}
