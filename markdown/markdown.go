// Package markdown handles markdown processing for Hiphops
package markdown

import (
	"io"
	"os"

	mdrender "github.com/teekennedy/goldmark-markdown"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"
)

type Markdown struct {
	md               goldmark.Markdown
	htmlRenderer     renderer.Renderer
	markdownRenderer renderer.Renderer
}

func NewMarkdown() *Markdown {
	md := goldmark.New(
		goldmark.WithExtensions(
			emoji.Emoji,
			extension.GFM,
			&frontmatter.Extender{
				Formats: []frontmatter.Format{frontmatter.YAML},
			},
		),
	)

	htmlRend := md.Renderer()
	htmlRend.AddOptions(html.WithUnsafe())

	mdRend := mdrender.NewRenderer()

	return &Markdown{
		md:               md,
		htmlRenderer:     htmlRend,
		markdownRenderer: mdRend,
	}
}

func (m *Markdown) HTML(source []byte, w io.Writer) (parser.Context, error) {
	ctx := parser.NewContext()
	m.md.SetRenderer(m.htmlRenderer)

	if err := m.md.Convert(source, w, parser.WithContext(ctx)); err != nil {
		return nil, err
	}

	return ctx, nil
}

func (m *Markdown) Markdown(source []byte, w io.Writer) (parser.Context, error) {
	ctx := parser.NewContext()
	m.md.SetRenderer(m.markdownRenderer)

	if err := m.md.Convert(source, w, parser.WithContext(ctx)); err != nil {
		return nil, err
	}

	return ctx, nil
}

func (m *Markdown) ReadFile(path string) ([]byte, parser.Context, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	reader := text.NewReader(content)
	pCtx := parser.NewContext()
	m.md.Parser().Parse(reader, parser.WithContext(pCtx))
	return content, pCtx, nil
}
