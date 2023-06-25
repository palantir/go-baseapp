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
	"testing"

	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

type SimpleMetrics struct {
	FooCount      metrics.Counter `metric:"foo.count"`
	BarCount      metrics.Counter `metric:"bar.count"`
	ActiveWorkers metrics.Gauge   `metric:"active_workers"`
}

type FunctionalMetrics struct {
	ActiveWorkers *metrics.FunctionalGauge `metric:"active_workers"`

	workers int64
}

func (m *FunctionalMetrics) ComputeActiveWorkers() int64 {
	m.workers++
	return m.workers
}

type SampleMetrics struct {
	LatencyA metrics.Histogram `metric:"latency.a" metric-sample:"uniform,100"`
	LatencyB metrics.Histogram `metric:"latency.b" metric-sample:"expdecay,20,0.1"`
}

type TaggedMetrics struct {
	Responses Tagged[metrics.Counter] `metric:"responses"`
	QueueSize Tagged[metrics.Gauge]   `metric:"queue_size"`
}

func TestNew(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		m := New[SimpleMetrics]()
		m.FooCount.Inc(1)
		m.BarCount.Inc(1)
		m.ActiveWorkers.Update(17)
	})

	t.Run("functional", func(t *testing.T) {
		m := New[FunctionalMetrics]()
		assert.Equal(t, int64(1), m.ActiveWorkers.Value())
		assert.Equal(t, int64(2), m.ActiveWorkers.Value())
	})

	t.Run("sample", func(t *testing.T) {
		m := New[SampleMetrics]()
		m.LatencyA.Update(300)
		m.LatencyB.Update(150)

		assert.IsType(t, &metrics.UniformSample{}, m.LatencyA.Sample(), "incorrect sample type")
		assert.IsType(t, &metrics.ExpDecaySample{}, m.LatencyB.Sample(), "incorrect sample type")
	})

	t.Run("tagged", func(t *testing.T) {
		m := New[TaggedMetrics]()
		m.Responses.Tag("code:200").Inc(1)
		m.QueueSize.Tag("reindex").Update(12)
	})
}
