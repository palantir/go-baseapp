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
	"fmt"

	"github.com/rcrowley/go-metrics"
)

type Metrics struct {
	Counter       metrics.Counter         `metric:"counter"`
	TaggedCounter Tagged[metrics.Counter] `metric:"counter.tagged"`

	Gauge           metrics.Gauge   `metric:"gauge"`
	FunctionalGauge FunctionalGauge `metric:"gauge.functional"`

	calls int64
}

func (m *Metrics) ComputeFunctionalGauge() int64 {
	m.calls++
	return m.calls
}

func Example() {
	// Initialize a Metrics struct
	m := New[Metrics]()

	// Register it with a registry
	Register(metrics.DefaultRegistry, m)

	// Report metrics
	m.Counter.Inc(5)
	m.TaggedCounter.Tag("foo").Inc(1)
	m.TaggedCounter.Tag("bar").Inc(1)
	m.Gauge.Update(42)

	// Read metrics from the registry
	metrics.Each(func(name string, m any) {
		switch m := m.(type) {
		case metrics.Counter:
			fmt.Printf("%s: %d\n", name, m.Count())
		case metrics.Gauge:
			fmt.Printf("%s: %d\n", name, m.Value())
		}
	})

	// Unordered output: counter: 5
	// counter.tagged: 0
	// counter.tagged[foo]: 1
	// counter.tagged[bar]: 1
	// gauge: 42
	// gauge.functional: 1
}
