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
	"reflect"
	"strconv"
	"strings"

	"github.com/rcrowley/go-metrics"
)

const (
	MetricTag       = "metric"
	MetricSampleTag = "metric-sample"
)

// DefaultReservoirSize and DefaultExpDecayAlpha are the values used for
// histogram sampling when the "metric-sample" tag is not present. They create
// an exponentially-decaying sample with the same behavior as UNIX load
// averages.
const (
	DefaultReservoirSize = 1028
	DefaultExpDecayAlpha = 0.015
)

var (
	counterType                = reflect.TypeOf((*metrics.Counter)(nil)).Elem()
	gaugeType                  = reflect.TypeOf((*metrics.Gauge)(nil)).Elem()
	functionalGaugeType        = reflect.TypeOf((*FunctionalGauge)(nil)).Elem()
	gaugeFloat64Type           = reflect.TypeOf((*metrics.GaugeFloat64)(nil)).Elem()
	functionalGaugeFloat64Type = reflect.TypeOf((*FunctionalGaugeFloat64)(nil)).Elem()
	histogramType              = reflect.TypeOf((*metrics.Histogram)(nil)).Elem()
	meterType                  = reflect.TypeOf((*metrics.Meter)(nil)).Elem()
	timerType                  = reflect.TypeOf((*metrics.Timer)(nil)).Elem()
)

// New creates a new metrics struct. The type M must be a struct and should
// have one or more fields that have the "metric" tag. New allocates a new
// struct and populates all of the tagged metric fields.
//
// The metric tag gives the name of the associated metric in the registry.
// Every field with this tag must be one of the supported metric interface
// types:
//
//   - [metrics.Counter]
//   - [metrics.Gauge]
//   - [metrics.GaugeFloat64]
//   - [metrics.Histogram]
//   - [metrics.Meter]
//   - [metrics.Timer]
//   - [Tagged]
//
// For example, this struct defines two metrics, a counter and a gauge:
//
//	type M struct {
//		APIResponses metrics.Counter `metric:"api.responses"`
//		Workers	     metrics.Gauge   `metric:"workers"`
//	}
//
// New panics if any aspect of the struct definition is invalid. New does not
// register the metrics with a registry. Call [Register] with the new struct to
// make the metrics available to emitters and other clients.
//
// By default, each metric registers as the static name given in the "metric"
// tag. You can define metrics with dynamic names by using the [Tagged]
// interface; see that type for more details.
//
// If the metric is a histogram or a timer, the field may also set the
// "metric-sample" tag. This tag defines the sample type for the metric's
// histogram. The tag value is a comma-separated list of the sample type and
// the sample's parameters. The supported types are:
//
//   - "uniform": optionally accepts an integer for the reservoir size
//   - "expdecay": optionally accepts an integer for the reservoir size and a
//     float for the alpha value; you must set both or neither value
//
// For example:
//
//	type M struct {
//		DownloadSize    metrics.Histogram `metric:"download.size" metric-sample:"uniform,100"`
//		DownloadLatency metrics.Time      `metric:"download.latency" metric-sample:"expdecay,1028,0.015"`
//	}
//
// See [rcrowley/go-metrics] for an explanation of the differences between
// sample types.
//
// If the tag is not set, the histogram uses an exponentially decaying sample
// with DefaultReservoirSize and DefaultExpDecayAlpha. These values are also
// used when the reservoir size and alpha are not specified.
//
// Metric fields can also be one of the functional metric interface types:
//
//   - [FunctionalGauge]
//   - [FunctionalGaugeFloat64]
//
// A functional metrics execute a function each time a client requests its
// value. Each functional metric must have a corresponding exported method or
// function field that is the field name with the "Compute" prefix.
// For example:
//
//	type M struct {
//		QueueLength FunctionalGauge		   `metric:"queue_length"`
//		Temperature FunctionalGaugeFloat64 `metric:"temperature"`
//
//		ComputeQueueLength func() int64
//	}
//
//	func (m *M) ComputeTemperature() float64 {
//		return getCurrentTemperature()
//	}
//
// New panics if a functional metric is missing its compute function or if the
// function has the wrong type. At this time, functional metrics do not support
// tagging.
//
// [rcrowley/go-metrics]: https://pkg.go.dev/github.com/rcrowley/go-metrics
func New[M any]() *M {
	var m M

	typ := reflect.TypeOf(m)
	if typ.Kind() != reflect.Struct {
		panic("appmetrics.New: type is not a struct")
	}

	fields, err := getMetricFields(typ)
	if err != nil {
		panic("appmetrics.New: " + err.Error())
	}

	v := reflect.ValueOf(&m).Elem()
	for _, f := range fields {
		if err := createField(v, f, f.Tag.Get(MetricTag)); err != nil {
			panic(fmt.Sprintf("appmetrics.New: field %s: %v", f.Name, err))
		}
	}
	return &m
}

// Register registers all of the metrics in the struct m with the registry. See
// New for an explanation of how this package identifies metric fields.
// Register panics if the struct contains invalid metric definitions.
//
// Register skips any metric with a name that already exist in the registry,
// even if the existing metric has a different type.
func Register[M any](r metrics.Registry, m *M) {
	v := reflect.ValueOf(m).Elem()
	if v.Type().Kind() != reflect.Struct {
		panic("appmetrics.Register: type is not a struct pointer")
	}

	fields, err := getMetricFields(v.Type())
	if err != nil {
		panic("appmetrics.Register: " + err.Error())
	}

	for _, f := range fields {
		name := f.Tag.Get(MetricTag)
		metric := v.FieldByIndex(f.Index).Interface()

		if m, ok := metric.(interface{ register(metrics.Registry) }); ok {
			m.register(r)
		} else {
			_ = r.Register(name, metric)
		}
	}
}

// Unregister unregisters all of the metrics in the struct m from the registry.
// See New for an explanation of how this package identifies metric fields.
// Unregister panics if the struct contains invalid metric definitions.
//
// Unregistering is generally not required, but is necessary to free meter and
// timer metrics if they are otherwise unreferenced.
func Unregister[M any](r metrics.Registry, m *M) {
	v := reflect.ValueOf(m).Elem()
	if v.Type().Kind() != reflect.Struct {
		panic("appmetrics.Unregister: type is not a struct pointer")
	}

	fields, err := getMetricFields(v.Type())
	if err != nil {
		panic("appmetrics.Unregister: " + err.Error())
	}

	for _, f := range fields {
		r.Unregister(f.Tag.Get(MetricTag))
	}
}

// MetricNames returns the names of the metrics in the struct m. See New for an
// explanation of how this package identifies metric fields. MetricNames panics
// if the struct contains invalid metric definitions.
func MetricNames[M any](m *M) []string {
	v := reflect.ValueOf(m).Elem()
	if v.Type().Kind() != reflect.Struct {
		panic("appmetrics.MetricNames: type is not a struct pointer")
	}

	fields, err := getMetricFields(v.Type())
	if err != nil {
		panic("appmetrics.MetricNames: " + err.Error())
	}

	var names []string
	for _, f := range fields {
		names = append(names, f.Tag.Get(MetricTag))
	}
	return names
}

func getMetricFields(typ reflect.Type) ([]reflect.StructField, error) {
	var fields []reflect.StructField
	for _, f := range reflect.VisibleFields(typ) {
		if metric := f.Tag.Get(MetricTag); metric != "" {
			if isMetric(f.Type) {
				fields = append(fields, f)
			} else {
				return nil, fmt.Errorf("field %s: metric tag appears on non-metric type %s", f.Name, f.Type)
			}
		}
	}
	return fields, nil
}

func isMetric(typ reflect.Type) bool {
	tagged, taggedType := isTagged(typ)
	if tagged {
		typ = taggedType
	}
	switch typ {
	case counterType, gaugeType, gaugeFloat64Type, histogramType, meterType, timerType:
		return true
	case functionalGaugeType, functionalGaugeFloat64Type:
		// Functional gauges cannot be tagged because there's currently no way
		// to pass the tags in to the function. Without this, every tag will
		// report the same value, making the tags redundant.
		return !tagged
	}
	return false
}

func createField(v reflect.Value, f reflect.StructField, metricName string) error {
	metricType := f.Type

	tagged, taggedType := isTagged(metricType)
	if tagged {
		metricType = taggedType
	}

	var value any
	switch metricType {
	case counterType:
		newMetric := metrics.NewCounter
		if tagged {
			value = &taggedMetric[metrics.Counter]{name: metricName, newMetric: newMetric}
		} else {
			value = newMetric()
		}

	case functionalGaugeType:
		fn, err := getGaugeFunction[int64](v, f.Name)
		if err != nil {
			return err
		}
		value = metrics.NewFunctionalGauge(fn)

	case gaugeType:
		newMetric := metrics.NewGauge
		if tagged {
			value = &taggedMetric[metrics.Gauge]{name: metricName, newMetric: newMetric}
		} else {
			value = newMetric()
		}

	case functionalGaugeFloat64Type:
		fn, err := getGaugeFunction[float64](v, f.Name)
		if err != nil {
			return err
		}
		value = metrics.NewFunctionalGaugeFloat64(fn)

	case gaugeFloat64Type:
		newMetric := metrics.NewGaugeFloat64
		if tagged {
			value = &taggedMetric[metrics.GaugeFloat64]{name: metricName, newMetric: newMetric}
		} else {
			value = newMetric()
		}

	case histogramType:
		newMetric := func() metrics.Histogram {
			return metrics.NewHistogram(
				metrics.NewExpDecaySample(DefaultReservoirSize, DefaultExpDecayAlpha),
			)
		}
		if sample := f.Tag.Get(MetricSampleTag); sample != "" {
			s, err := parseSample(sample)
			if err != nil {
				return err
			}
			newMetric = func() metrics.Histogram {
				return metrics.NewHistogram(s())
			}
		}
		if tagged {
			value = &taggedMetric[metrics.Histogram]{name: metricName, newMetric: newMetric}
		} else {
			value = newMetric()
		}

	case meterType:
		newMetric := metrics.NewMeter
		if tagged {
			value = &taggedMetric[metrics.Meter]{name: metricName, newMetric: newMetric}
		} else {
			value = newMetric()
		}

	case timerType:
		newMetric := metrics.NewTimer
		if sample := f.Tag.Get(MetricSampleTag); sample != "" {
			s, err := parseSample(sample)
			if err != nil {
				return err
			}
			newMetric = func() metrics.Timer {
				return metrics.NewCustomTimer(metrics.NewHistogram(s()), metrics.NewMeter())
			}
		}
		if tagged {
			value = &taggedMetric[metrics.Timer]{name: metricName, newMetric: newMetric}
		} else {
			value = newMetric()
		}
	}

	v.FieldByIndex(f.Index).Set(reflect.ValueOf(value))
	return nil
}

func parseSample(s string) (func() metrics.Sample, error) {
	parts := strings.Split(strings.ToLower(s), ",")
	switch parts[0] {
	case "uniform":
		return parseUniformSample(parts)
	case "expdecay":
		return parseExpDecaySample(parts)
	default:
		return nil, fmt.Errorf("invalid sample type")
	}
}

func parseUniformSample(parts []string) (func() metrics.Sample, error) {
	var fn func() metrics.Sample
	switch len(parts) {
	case 1:
		fn = func() metrics.Sample {
			return metrics.NewUniformSample(DefaultReservoirSize)
		}
	case 2:
		rs, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid uniform sample: reservoir: %w", err)
		}
		fn = func() metrics.Sample {
			return metrics.NewUniformSample(rs)
		}
	default:
		return nil, fmt.Errorf("invalid uniform sample")
	}
	return fn, nil
}

func parseExpDecaySample(parts []string) (func() metrics.Sample, error) {
	var fn func() metrics.Sample
	switch len(parts) {
	case 1:
		fn = func() metrics.Sample {
			return metrics.NewExpDecaySample(DefaultReservoirSize, DefaultExpDecayAlpha)
		}
	case 3:
		rs, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid expdecay sample: reservoir: %w", err)
		}
		alpha, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid expdecay sample: alpha: %w", err)
		}
		fn = func() metrics.Sample {
			return metrics.NewExpDecaySample(rs, alpha)
		}
	default:
		return nil, fmt.Errorf("invalid expdecay sample")
	}
	return fn, nil
}
