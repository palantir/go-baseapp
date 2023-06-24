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
	ActiveWorkers metrics.Gauge `metric:"active_workers" metric-func:"GetWorkers"`

	workers int64
}

func (m *FunctionalMetrics) GetWorkers() int64 {
	m.workers++
	return m.workers
}

type SampleMetrics struct {
	LatencyA metrics.Histogram `metric:"latency.a" metric-sample:"uniform,100"`
	LatencyB metrics.Histogram `metric:"latency.b" metric-sample:"expdecay,20,0.1"`
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
}
