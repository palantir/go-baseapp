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
	"strings"

	"github.com/rcrowley/go-metrics"
)

var (
	strSliceType = reflect.TypeOf([]string(nil))
)

// Tagged is a metric with dynamic tags. The type M must be one of the
// supported metric types exported by rcrowley/go-metrics.
//
// The Tag method returns an instance of the metric that reports with the given
// tags. Tags may be either plain values or key and values separated by a
// colon.
type Tagged[M any] interface {
	Tag(tags ...string) M
}

type taggedMetric[M any] struct {
	r         metrics.Registry
	name      string
	newMetric func() M
}

func (m *taggedMetric[M]) Tag(tags ...string) M {
	name := m.name
	if len(tags) > 0 {
		name += "[" + strings.Join(tags, ",") + "]"
	}
	if m.r == nil {
		return m.newMetric()
	}
	return m.r.GetOrRegister(name, m.newMetric).(M)
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
