// Package markdown handles markdown processing for Hiphops
package markdown

import (
	"io"
	"os"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"
)

type Markdown struct {
	md goldmark.Markdown
}

func NewMarkdownHTML() *Markdown {
	md := goldmark.New(
		goldmark.WithExtensions(
			emoji.Emoji,
			extension.GFM,
			&frontmatter.Extender{
				Formats: []frontmatter.Format{frontmatter.YAML},
			},
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	return &Markdown{md: md}
}

func (m *Markdown) Convert(source []byte, w io.Writer) (parser.Context, error) {
	ctx := parser.NewContext()
	if err := m.md.Convert(source, w, parser.WithContext(ctx)); err != nil {
		return nil, err
	}

	return ctx, nil
}

func (m *Markdown) ParseFile(path string) (ast.Node, parser.Context, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	reader := text.NewReader(content)
	pCtx := parser.NewContext()
	ast := m.md.Parser().Parse(reader, parser.WithContext(pCtx))

	return ast, pCtx, nil
}
