package staticgen

import (
	"io"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
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
