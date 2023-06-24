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
	"strings"

	"github.com/rcrowley/go-metrics"
)

// TaggedCounter is a metrics.Counter with dynamic tags.
//
// Tag returns an instance of a counter that reports with the given tags. Tags
// may be either plain values or keys and values separated by a colon.
type TaggedCounter interface {
	Tag(tags ...string) metrics.Counter
}

// TaggedGauge is a metrics.Gauge with dynamic tags.
//
// Tag returns an instance of a gauge that reports with the given tags. Tags
// may be either plain values or keys and values separated by a colon.
type TaggedGauge interface {
	Tag(tags ...string) metrics.Gauge
}

// TaggedGaugeFloat64 is a metrics.GaugeFloat64 with dynamic tags.
//
// Tag returns an instance of a gauge that reports with the given tags. Tags
// may be either plain values or keys and values separated by a colon.
type TaggedGaugeFloat64 interface {
	Tag(tags ...string) metrics.GaugeFloat64
}

// TaggedHistogram is a metrics.Histogram with dynamic tags.
//
// Tag returns an instance of a histogram that reports with the given tags.
// Tags may be either plain values or keys and values separated by a colon.
type TaggedHistogram interface {
	Tag(tags ...string) metrics.Histogram
}

// TaggedMeter is a metrics.Meter with dynamic tags.
//
// Tag returns an instance of a meter that reports with the given tags. Tags
// may be either plain values or keys and values separated by a colon.
type TaggedMeter interface {
	Tag(tags ...string) metrics.Meter
}

// TaggedTimer is a metrics.Timer with dynamic tags.
//
// Tag returns an instance of a timer that reports with the given tags. Tags
// may be either plain values or keys and values separated by a colon.
type TaggedTimer interface {
	Tag(tags ...string) metrics.Timer
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
