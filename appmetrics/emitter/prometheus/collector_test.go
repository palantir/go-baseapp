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

package prometheus

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/rcrowley/go-metrics"
)

func TestCollector(t *testing.T) {
	t.Run("metricTypes", func(t *testing.T) {
		r := metrics.NewRegistry()
		c := NewCollector(r)

		metrics.NewRegisteredCounter("counter", r)
		metrics.NewRegisteredGauge("gauge", r)
		metrics.NewRegisteredGaugeFloat64("gauge_float64", r)
		metrics.NewRegisteredHistogram("histogram", r, metrics.NewUniformSample(64))
		metrics.NewRegisteredMeter("meter", r)
		metrics.NewRegisteredTimer("timer", r)

		expected := `
# HELP counter metrics.Counter
# TYPE counter untyped
counter 0
# HELP gauge metrics.Gauge
# TYPE gauge gauge
gauge 0
# HELP gauge_float64 metrics.GaugeFloat64
# TYPE gauge_float64 gauge
gauge_float64 0
# HELP histogram metrics.Histogram
# TYPE histogram summary
histogram{quantile="0.5"} 0
histogram{quantile="0.95"} 0
histogram_sum 0
histogram_count 0
# HELP histogram_max metrics.Histogram
# TYPE histogram_max untyped
histogram_max 0
# HELP histogram_min metrics.Histogram
# TYPE histogram_min untyped
histogram_min 0
# HELP meter_count metrics.Meter
# TYPE meter_count untyped
meter_count 0
# HELP timer_max_seconds metrics.Timer
# TYPE timer_max_seconds untyped
timer_max_seconds 0
# HELP timer_min_seconds metrics.Timer
# TYPE timer_min_seconds untyped
timer_min_seconds 0
# HELP timer_seconds metrics.Timer
# TYPE timer_seconds summary
timer_seconds{quantile="0.5"} 0
timer_seconds{quantile="0.95"} 0
timer_seconds_sum 0
timer_seconds_count 0
`

		if err := testutil.CollectAndCompare(c, strings.NewReader(expected)); err != nil {
			t.Error(err)
		}
	})

	t.Run("labels", func(t *testing.T) {
		r := metrics.NewRegistry()
		c := NewCollector(r, WithLabels(map[string]string{
			"test": "labels",
		}))

		counterA := metrics.NewRegisteredCounter("counter[subsystem:a,role:server]", r)
		counterB := metrics.NewRegisteredCounter("counter[subsystem:b,role:server]", r)
		counterC := metrics.NewRegisteredCounter("unlabeled_counter", r)

		counterA.Inc(1)
		counterB.Inc(2)
		counterC.Inc(3)

		expected := `
# HELP counter metrics.Counter
# TYPE counter untyped
counter{role="server",subsystem="a",test="labels"} 1
counter{role="server",subsystem="b",test="labels"} 2
# HELP unlabeled_counter metrics.Counter
# TYPE unlabeled_counter untyped
unlabeled_counter{test="labels"} 3
`

		if err := testutil.CollectAndCompare(c, strings.NewReader(expected)); err != nil {
			t.Error(err)
		}
	})

	t.Run("sanitize", func(t *testing.T) {
		r := metrics.NewRegistry()
		c := NewCollector(r)

		metrics.NewRegisteredCounter("> invalid metric names! are ~~fun~~ ☃️", r)

		expected := `
# HELP invalid_metric_names_are_fun_ metrics.Counter
# TYPE invalid_metric_names_are_fun_ untyped
invalid_metric_names_are_fun_ 0
`

		if err := testutil.CollectAndCompare(c, strings.NewReader(expected)); err != nil {
			t.Error(err)
		}
	})

	t.Run("histogramQuantiles", func(t *testing.T) {
		r := metrics.NewRegistry()
		c := NewCollector(r, WithHistogramQuantiles([]float64{0.25, 0.5, 0.75}))

		hist := metrics.NewRegisteredHistogram("histogram", r, metrics.NewUniformSample(1024))
		for _, v := range []int64{0, 2, 4, 6, 8} {
			for i := 0; i < 10; i++ {
				hist.Update(v)
			}
		}

		expected := `
# HELP histogram metrics.Histogram
# TYPE histogram summary
histogram{quantile="0.25"} 2
histogram{quantile="0.5"} 4
histogram{quantile="0.75"} 6
histogram_sum 200
histogram_count 50
# HELP histogram_max metrics.Histogram
# TYPE histogram_max untyped
histogram_max 8
# HELP histogram_min metrics.Histogram
# TYPE histogram_min untyped
histogram_min 0
`

		if err := testutil.CollectAndCompare(c, strings.NewReader(expected)); err != nil {
			t.Error(err)
		}
	})

	t.Run("timerQuantiles", func(t *testing.T) {
		r := metrics.NewRegistry()
		c := NewCollector(r, WithTimerQuantiles([]float64{0.25, 0.5, 0.75}))

		timer := metrics.NewCustomTimer(metrics.NewHistogram(metrics.NewUniformSample(1024)), metrics.NewMeter())
		for _, v := range []int64{0, 2, 4, 6, 8} {
			for i := 0; i < 10; i++ {
				timer.Update(time.Duration(v) * time.Millisecond)
			}
		}

		_ = r.Register("timer", timer)

		expected := `
# HELP timer_max_seconds metrics.Timer
# TYPE timer_max_seconds untyped
timer_max_seconds 0.008
# HELP timer_min_seconds metrics.Timer
# TYPE timer_min_seconds untyped
timer_min_seconds 0
# HELP timer_seconds metrics.Timer
# TYPE timer_seconds summary
timer_seconds{quantile="0.25"} 0.002
timer_seconds{quantile="0.5"} 0.004
timer_seconds{quantile="0.75"} 0.006
timer_seconds_sum 0.2
timer_seconds_count 50
`

		if err := testutil.CollectAndCompare(c, strings.NewReader(expected)); err != nil {
			t.Error(err)
		}
	})
}
