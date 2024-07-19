package staticgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/antchfx/htmlquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	type testCase struct {
		name string
		// Source files as path (relative to source dir): content as bytes
		source map[string][]byte
		// Expected output files as path (relative to build dir): content as bytes
		expected map[string][]byte
	}

	tests := []testCase{
		{
			"Markdown to HTML",
			map[string][]byte{
				"hello.md":    []byte("# Hello"),
				"sub/page.MD": []byte("A page"),
			},
			map[string][]byte{
				"hello.html":    []byte("<h1>Hello</h1>"),
				"sub/page.html": []byte("<p>A page</p>"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buildDir := t.TempDir()
			sourceDir := setupSourceDir(t, tc.source)

			builder, err := NewStaticBuilder()
			require.NoError(t, err, "Test setup error")

			err = builder.Build(sourceDir, buildDir)
			require.NoError(t, err, "Build should complete without error")

			for relPath, expectedContent := range tc.expected {
				path := filepath.Join(buildDir, relPath)
				content, err := os.ReadFile(path)
				if assert.NoError(t, err, "Should be able to read output file") {
					assert.Contains(
						t,
						string(content),
						string(expectedContent),
						"Built file should contain rendered content",
					)
				}
			}
		})
	}
}

func TestBuildMarkdown(t *testing.T) {
	type testCase struct {
		name string
		// Content of input markdown as bytes
		source []byte
		// XPath queries that should match against the output document
		queries map[string]int
	}

	tests := []testCase{
		{
			"Simple",
			[]byte("# Hello"),
			map[string]int{
				"/html/head/title": 1,
				"/html/body":       1,
				"//h1":             1,
			},
		},

		{
			"Fromtmatter",
			[]byte(
				`---
title: Atitle
description: This page
---
# Hello
`),
			map[string]int{
				`/html/head/meta[@name = "description" and @content = "This page"]`: 1,
				`/html/head/title[text() = "Atitle"]`:                               1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buildDir := t.TempDir()
			sourceFile := map[string][]byte{
				"test.md": tc.source,
			}
			sourceDir := setupSourceDir(t, sourceFile)
			source := filepath.Join(sourceDir, "test.md")

			builder, err := NewStaticBuilder()
			require.NoError(t, err, "Test setup error")

			err = builder.BuildMarkdown(source, buildDir)
			require.NoError(t, err, "BuildMarkdown should complete without error")

			for q, matches := range tc.queries {
				path := filepath.Join(buildDir, "test.html")
				doc, err := htmlquery.LoadDoc(path)
				if !assert.NoError(t, err, "HTML doc should be well formatted and present") {
					continue
				}

				results, err := htmlquery.QueryAll(doc, q)
				require.NoError(t, err, "Invalid test: Xpath queries should all be valid")

				assert.Equalf(
					t,
					matches,
					len(results),
					"Query '%s' should return %d results in output doc. Found %d",
					q, matches, len(results),
				)
			}
		})
	}
}

func setupSourceDir(t *testing.T, source map[string][]byte) string {
	sourceDir := t.TempDir()
	for relPath, content := range source {
		// Ensure the file's dir exists (in case of nested dir structures)
		path := filepath.Join(sourceDir, relPath)
		fileDir := filepath.Dir(path)

		err := os.MkdirAll(fileDir, os.ModePerm)
		require.NoError(t, err, "Test setup: Unable to create source file dir")

		err = os.WriteFile(path, content, os.ModePerm)
		require.NoError(t, err, "Test setup: Unable to write source file")
	}

	return sourceDir
}
