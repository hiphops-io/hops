package dsl

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnyJSONToCtyValue(t *testing.T) {
	eventFile := "./testdata/raw_change_event.json"

	eventJson, err := os.ReadFile(eventFile)
	assert.NoError(t, err)

	eventCty, err := AnyJSONToCtyValue(eventJson)
	assert.NoError(t, err)

	assert.Equal(t, "0395b0b2-0dcd-4dfb-89f8-65a36d32d9f3", eventCty.GetAttr("project_id").AsString())
	assert.Equal(t, "change", eventCty.GetAttr("hops").GetAttr("event").AsString())
}

func TestEventBundleToCty(t *testing.T) {
	eventFile := "./testdata/raw_change_event.json"
	aaEvent := []byte(`{"path": "a.a"}`)
	abEvent := []byte(`{"path": "a.b"}`)
	acaEvent := []byte(`{"path": "a.c.a"}`)
	acbEvent := []byte(`{"path": "a.c.b"}`)
	deeplyNestedEvent := []byte(`{"path": "b.c.d.e.f.g"}`)

	eventJson, err := os.ReadFile(eventFile)
	assert.NoError(t, err)

	eventBundle := map[string][]byte{
		"event":       eventJson,         // Ensure non-nested paths are set properly
		"a.a":         aaEvent,           // Ensure nested paths are set properly
		"a.b":         abEvent,           // ... and that their siblings are preserved too
		"a.c.a":       acaEvent,          // Then the same for deeply nested paths
		"a.c.b":       acbEvent,          // ... and their siblings
		"b.c.d.e.f.g": deeplyNestedEvent, // and a really deep path
	}

	bundleCty, err := eventBundleToCty(eventBundle, ".")
	require.NoError(t, err)

	eventCty, ok := bundleCty["event"]
	if assert.True(t, ok, "Key 'event' should be present on cty bundle") {
		if assert.True(t, eventCty.Type().HasAttribute("project_id"), "Path 'event.project_id' should be present") {
			assert.Equal(t, "0395b0b2-0dcd-4dfb-89f8-65a36d32d9f3", eventCty.GetAttr("project_id").AsString())
		}

		if assert.True(t, eventCty.Type().HasAttribute("hops"), "Path 'event.hops' should be present") {
			assert.Equal(t, "change", eventCty.GetAttr("hops").GetAttr("event").AsString())
		}
	}

	aVal, ok := bundleCty["a"]
	require.True(t, ok, "Key 'a' should be present on cty bundle")

	// Test value at a.a
	if assert.True(t, aVal.Type().HasAttribute("a"), "Path 'a.a' should be present") {
		aaVal := aVal.GetAttr("a")

		if assert.True(t, aaVal.Type().HasAttribute("path"), "Path 'a.a' should should have expected attributes") {
			assert.Equal(t, "a.a", aaVal.GetAttr("path").AsString())
		}
	}

	// Test value at a.b
	if assert.True(t, aVal.Type().HasAttribute("b"), "Path 'a.b' should be present") {
		abVal := aVal.GetAttr("b")

		if assert.True(t, abVal.Type().HasAttribute("path"), "Path 'a.b' should have expected attributes") {
			assert.Equal(t, "a.b", abVal.GetAttr("path").AsString())
		}
	}

	// Test values at a.c.*
	require.True(t, aVal.Type().HasAttribute("c"), "Path 'a.c' should be present")
	acVal := aVal.GetAttr("c")

	// Test value at a.c.a
	if assert.True(t, acVal.Type().HasAttribute("a"), "Path 'a.c.a' should be present") {
		acaVal := acVal.GetAttr("a")

		if assert.True(t, acaVal.Type().HasAttribute("path"), "Path 'a.c.a' should have expected attributes") {
			assert.Equal(t, "a.c.a", acaVal.AsValueMap()["path"].AsString())
		}
	}

	// Test value at a.c.b
	if assert.True(t, acVal.Type().HasAttribute("b"), "Path 'a.c.b' should be present") {
		acbVal := acVal.GetAttr("b")

		if assert.True(t, acbVal.Type().HasAttribute("path"), "Path 'a.c.b' should have expected attributes") {
			assert.Equal(t, "a.c.b", acbVal.AsValueMap()["path"].AsString())
		}
	}

	deepVal := bundleCty["b"].GetAttr("c").GetAttr("d").GetAttr("e").GetAttr("f").GetAttr("g").GetAttr("path").AsString()
	assert.Equal(t, "b.c.d.e.f.g", deepVal)
}
