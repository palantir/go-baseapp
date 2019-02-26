// Copyright 2019 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
