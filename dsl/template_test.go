package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRunTemplate tests runTemplate with focus on special cases of
// maintaining indentation, and enabling autoescaping of inputs for safe HTML
// templating
func TestRunTemplate(t *testing.T) {
	templateContent := `my code test {
  We are calling {{ accountId }}:{{ password }}
    -> And we got back {{ password }}:{{ accountId }}

	{{ html_danger }}
}`

	// Test for text mode (no autoescaping)
	t.Run("Test Indentation and No HTML Escaping", func(t *testing.T) {
		variables := map[string]any{
			"accountId":   "thisisabigaccountid",
			"password":    "averybigsecret",
			"html_danger": "<script>alert('xss');</script>",
		}

		expectedOutput := `my code test {
  We are calling thisisabigaccountid:averybigsecret
    -> And we got back averybigsecret:thisisabigaccountid

	<script>alert('xss');</script>
}`

		output, err := runTemplate(templateContent, variables)

		if assert.NoError(t, err) {
			assert.Equal(t, expectedOutput, output)
		}
	})

	// Test for HTML mode (with autoescaping)
	t.Run("Test Indentation and HTML Escaping", func(t *testing.T) {
		variables := map[string]any{
			"autoescape":  true,
			"accountId":   "thisisabigaccountid",
			"password":    "averybigsecret",
			"html_danger": "<script>alert('xss');</script>",
		}

		expectedOutput := `my code test {
  We are calling thisisabigaccountid:averybigsecret
    -> And we got back averybigsecret:thisisabigaccountid

	&lt;script&gt;alert(&#39;xss&#39;);&lt;/script&gt;
}`

		output, err := runTemplate(templateContent, variables)
		if assert.NoError(t, err) {
			assert.Equal(t, expectedOutput, output)
		}
	})
}
