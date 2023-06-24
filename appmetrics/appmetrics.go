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

// TODO(bkeyes): package and function docs

const (
	MetricTag       = "metric"
	MetricSampleTag = "metric-sample"
)

const (
	GaugeFunctionPrefix = "Compute"
)

const (
	DefaultReservoirSize = 1028
	DefaultExpDecayAlpha = 0.015
)

var (
	counterType                = reflect.TypeOf((*metrics.Counter)(nil)).Elem()
	gaugeType                  = reflect.TypeOf((*metrics.Gauge)(nil)).Elem()
	functionalGaugeType        = reflect.TypeOf(&metrics.FunctionalGauge{})
	gaugeFloat64Type           = reflect.TypeOf((*metrics.GaugeFloat64)(nil)).Elem()
	functionalGaugeFloat64Type = reflect.TypeOf(&metrics.FunctionalGaugeFloat64{})
	histogramType              = reflect.TypeOf((*metrics.Histogram)(nil)).Elem()
	meterType                  = reflect.TypeOf((*metrics.Meter)(nil)).Elem()
	timerType                  = reflect.TypeOf((*metrics.Timer)(nil)).Elem()

	taggedCounterType      = reflect.TypeOf((*TaggedCounter)(nil)).Elem()
	taggedGaugeType        = reflect.TypeOf((*TaggedGauge)(nil)).Elem()
	taggedGaugeFloat64Type = reflect.TypeOf((*TaggedGaugeFloat64)(nil)).Elem()
	taggedHistogramType    = reflect.TypeOf((*TaggedHistogram)(nil)).Elem()
	taggedMeterType        = reflect.TypeOf((*TaggedMeter)(nil)).Elem()
	taggedTimerType        = reflect.TypeOf((*TaggedTimer)(nil)).Elem()
)

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
			if isMetricType(f.Type) {
				fields = append(fields, f)
			} else {
				return nil, fmt.Errorf("field %s: metric tag appears on non-metric type %s", f.Name, f.Type)
			}
		}
	}
	return fields, nil
}

func isMetricType(typ reflect.Type) bool {
	switch typ {
	case counterType, gaugeType, functionalGaugeType, gaugeFloat64Type, functionalGaugeFloat64Type, histogramType, meterType, timerType:
		return true
	case taggedCounterType, taggedGaugeType, taggedGaugeFloat64Type, taggedHistogramType, taggedMeterType, taggedTimerType:
		return true
	}
	return false
}

func createField(v reflect.Value, f reflect.StructField, metricName string) error {
	var value any
	switch f.Type {
	case counterType, taggedCounterType:
		newMetric := metrics.NewCounter
		if f.Type == taggedCounterType {
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

	case gaugeType, taggedGaugeType:
		newMetric := metrics.NewGauge
		if f.Type == taggedGaugeType {
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

	case gaugeFloat64Type, taggedGaugeFloat64Type:
		newMetric := metrics.NewGaugeFloat64
		if f.Type == taggedGaugeFloat64Type {
			value = &taggedMetric[metrics.GaugeFloat64]{name: metricName, newMetric: newMetric}
		} else {
			value = newMetric()
		}

	case histogramType, taggedHistogramType:
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
		if f.Type == taggedHistogramType {
			value = &taggedMetric[metrics.Histogram]{name: metricName, newMetric: newMetric}
		} else {
			value = newMetric()
		}

	case meterType, taggedMeterType:
		newMetric := metrics.NewMeter
		if f.Type == taggedMeterType {
			value = &taggedMetric[metrics.Meter]{name: metricName, newMetric: newMetric}
		} else {
			value = newMetric()
		}

	case timerType, taggedTimerType:
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
		if f.Type == taggedTimerType {
			value = &taggedMetric[metrics.Timer]{name: metricName, newMetric: newMetric}
		} else {
			value = newMetric()
		}
	}

	v.FieldByIndex(f.Index).Set(reflect.ValueOf(value))
	return nil
}

func getGaugeFunction[N int64 | float64, F func() N](v reflect.Value, fieldName string) (F, error) {
	name := GaugeFunctionPrefix + fieldName

	m := v.Addr().MethodByName(name)
	if !m.IsValid() {
		// A method does not exist, look for a field with the name instead
		m = v.FieldByName(name)
		if !m.IsValid() {
			return nil, fmt.Errorf("%s: method or field does not exist", name)
		}
		if m.Type().Kind() != reflect.Func {
			return nil, fmt.Errorf("%s: field must be a function", name)
		}
	}

	if m.Type().NumIn() != 0 {
		return nil, fmt.Errorf("%s: function must take no parameters", name)
	}
	if m.Type().NumOut() != 1 {
		return nil, fmt.Errorf("%s: function must return a single value", name)
	}
	if m.Type().Out(0) != reflect.TypeOf(N(0)) {
		return nil, fmt.Errorf("%s: function must return a value of type %T", name, N(0))
	}
	return m.Interface().(F), nil
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
