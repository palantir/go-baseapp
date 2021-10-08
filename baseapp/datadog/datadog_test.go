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
	"strings"
	"testing"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		name, tags := tagsFromName("multiple[tag2:value,tag1]")
		assert.Equal(t, "multiple", name)
		assert.Equal(t, []string{"tag1", "tag2:value"}, tags)
	})

	t.Run("invalidFormat", func(t *testing.T) {
		name, tags := tagsFromName("invalid[tag1")
		assert.Equal(t, "invalid[tag1", name)
		assert.Empty(t, tags)
	})
}

func TestEmitCounts(t *testing.T) {
	initialize := func() (*Emitter, *MemoryWriter, *statsd.Client, metrics.Registry) {
		w := &MemoryWriter{}
		c, err := statsd.NewWithWriter(w)
		require.NoError(t, err)

		r := metrics.NewRegistry()
		return NewEmitter(c, r), w, c, r
	}

	t.Run("single", func(t *testing.T) {
		e, w, client, r := initialize()
		c := metrics.NewRegisteredCounter("counter", r)

		c.Inc(1)
		e.EmitOnce()

		// Close the client to ensure flushing of results. Required for testing.
		// See official tests https://github.com/DataDog/datadog-go/blob/ebec44a4d1f34e521da53ddb5fd8190aa379fd99/statsd/statsd_test.go#L87
		require.NoError(t, client.Close())

		assert.Equal(t, int64(1), c.Count())
		assert.Equal(t, []string{"counter:1|c"}, w.Messages)
	})

	t.Run("difference", func(t *testing.T) {
		e, w, client, r := initialize()
		c := metrics.NewRegisteredCounter("counter", r)

		c.Inc(1)
		e.EmitOnce()
		c.Inc(2)
		e.EmitOnce()

		// Close the client to ensure flushing of results. Required for testing.
		// See official tests https://github.com/DataDog/datadog-go/blob/ebec44a4d1f34e521da53ddb5fd8190aa379fd99/statsd/statsd_test.go#L87
		require.NoError(t, client.Close())

		assert.Equal(t, int64(3), c.Count())
		assert.Equal(t, []string{"counter:1|c", "counter:2|c"}, w.Messages)

	})
}

type MemoryWriter struct {
	Messages []string
}

func (mw *MemoryWriter) Write(b []byte) (int, error) {
	for _, m := range strings.Split(string(b), "\n") {
		if m != "" {
			mw.Messages = append(mw.Messages, m)
		}
	}
	return len(b), nil
}

func (mw *MemoryWriter) Close() error { return nil }

func (mw *MemoryWriter) SetWriteTimeout(t time.Duration) error { return nil }
