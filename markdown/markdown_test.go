package markdown

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownToHTML(t *testing.T) {
	type testCase struct {
		name     string
		markdown []byte
		expected []byte
	}

	tests := []testCase{
		{
			"Blank file",
			[]byte(""),
			[]byte(nil),
		},

		{
			"Common markdown",
			[]byte(
				`# Heading

## Heading two

This is a paragraph

- A
- List

1. Ordered
2. List

`),
			[]byte(
				`<h1>Heading</h1>
<h2>Heading two</h2>
<p>This is a paragraph</p>
<ul>
<li>A</li>
<li>List</li>
</ul>
<ol>
<li>Ordered</li>
<li>List</li>
</ol>
`),
		},

		{
			"Emojis",
			[]byte(`:blush:`),
			[]byte("<p>&#x1f60a;</p>\n"),
		},

		{
			"Front matter",
			[]byte(`---
title: A page
---
# Heading
`),
			[]byte("<h1>Heading</h1>\n"),
		},

		{
			"Unsafe",
			[]byte(`<script>console.log('foo')</script>`),
			[]byte("<script>console.log('foo')</script>"),
		},
	}

	md := NewMarkdownHTML()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var b bytes.Buffer

			_, err := md.Convert(tc.markdown, &b)
			require.NoError(t, err, "Markdown should convert without error")

			// Compare as strings to make failure output more readable
			assert.Equal(t, string(tc.expected), b.String())
		})
	}
}
