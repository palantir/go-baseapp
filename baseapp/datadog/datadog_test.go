package datadog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTagsFromName(t *testing.T) {
	t.Run("noTags", func(t *testing.T) {
		name, tags := tagsFromName("notags")
		assert.Equal(t, "notags", name)
		assert.Empty(t, tags)
	})

	t.Run("singleTag", func(t *testing.T) {
		name, tags := tagsFromName("single[tag1]")
		assert.Equal(t, "single", name)
		assert.Equal(t, []string{"tag1"}, tags)
	})

	t.Run("multipleTags", func(t *testing.T) {
		name, tags := tagsFromName("multiple[tag1,tag2:value]")
		assert.Equal(t, "multiple", name)
		assert.Equal(t, []string{"tag1", "tag2:value"}, tags)
	})

	t.Run("invalidFormat", func(t *testing.T) {
		name, tags := tagsFromName("invalid[tag1")
		assert.Equal(t, "invalid[tag1", name)
		assert.Empty(t, tags)
	})
}
