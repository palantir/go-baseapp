// Copyright 2023 Palantir Technologies, Inc.
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

package appmetrics

import (
	"reflect"
	"sort"
	"strings"

	"github.com/rcrowley/go-metrics"
)

var (
	strSliceType = reflect.TypeOf([]string(nil))
)

// Tagged is a metric with dynamic tags. The type M must be one of the
// supported metric types. Tags are strings that can either be plain values or
// key-value pairs where the key and value are separated by a colon.
//
// While Tagged metrics can be used directly, it's helpful to wrap them in a
// function that accepts the expected tag values using the correct types. For
// example:
//
//	struct M {
//		Responses Tagged[metrics.Counter] `metric:"responses"`
//	}
//
//	func (m *M) ResponsesByTypeAndStatus(typ string, status int) metrics.Counter {
//		return m.Responses.Tag("type:" + type, "status:" + strconv.Itoa(stats))
//	}
//
// Tags are added as a suffix to the base metric name: the tags are joined by
// commas, then surrounded by square brackets. Using the previous example, the
// full metric names might be:
//
//   - "responses[type:api,status:200]"
//   - "responses[type:file,status:404]"
//
// Note that each unique combination of tags produces a separate metric in the
// registry. For this reason avoid tags that can take many values, like IDs.
type Tagged[M any] interface {
	// Tag returns an instance of the metric that reports with the given tags.
	// Tags may be either plain values or key-value pairs separated by a colon.
	// Tag trims whitespace from each tag and ignores any empty tags.
	Tag(tags ...string) M
}

type taggedMetric[M any] struct {
	r         metrics.Registry
	name      string
	newMetric func() M
}

func (m *taggedMetric[M]) Tag(tags ...string) M {
	if m.r == nil {
		return m.newMetric()
	}

	var name strings.Builder
	name.WriteString(m.name)

	if tags := cleanAndSortTags(tags); len(tags) > 0 {
		name.WriteString("[")
		for i, t := range tags {
			if i > 0 {
				name.WriteString(",")
			}
			name.WriteString(t)
		}
		name.WriteString("]")
	}

	return m.r.GetOrRegister(name.String(), m.newMetric).(M)
}

func (m *taggedMetric[M]) register(r metrics.Registry) {
	m.r = r

	// Add the bare metric immediately so emitters can find it in the registry
	r.GetOrRegister(m.name, m.newMetric)
}

// isTagged determines if typ is a Tagged instantiation and returns the
// parameter type. As of Go 1.20, the reflect package does not support direct
// access to type parameters.
func isTagged(typ reflect.Type) (bool, reflect.Type) {
	if typ.Kind() != reflect.Interface {
		return false, nil
	}

	m, ok := typ.MethodByName("Tag")
	if !ok {
		return false, nil
	}

	mt := m.Type
	if !mt.IsVariadic() || mt.NumIn() != 1 || mt.In(0) != strSliceType {
		return false, nil
	}
	if mt.NumOut() != 1 {
		return false, nil
	}
	return true, mt.Out(0)
}

func cleanAndSortTags(tags []string) []string {
	cleanTags := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" {
			cleanTags = append(cleanTags, t)
		}
	}
	sort.Strings(cleanTags)
	return cleanTags
}
