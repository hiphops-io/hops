package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	result := slugify("Hello World")
	assert.Equal(t, "hello-world", result)

	result = slugify("on", "Hello World")
	assert.Equal(t, "on-hello-world", result)

	result = slugify("on", "change.opened", "Hello World")
	assert.Equal(t, "on-change-opened-hello-world", result)

	result = slugify("change_opened", "hello_world")
	assert.Equal(t, "change_opened-hello_world", result)
}
