package markdown

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

	// Attempt to wipe the dir if we had an error
	if err != nil {
		os.RemoveAll(build)
	}
	return err
}

func (s *StaticBuilder) BuildFile(path, build string) error {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".md":
		if err := s.BuildMarkdown(path, build); err != nil {
			return err
		}
	default:
		if err := s.BuildPlain(path, build); err != nil {
			return err
		}
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

// BuildPlain takes a source file path and copies into the equivalent location
// in the build target dir
func (s *StaticBuilder) BuildPlain(source, build string) error {
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	path := buildPath(source, build, "")
	writer, err := os.Create(path)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = writer.Write(content)
	return err
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
