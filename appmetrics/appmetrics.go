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
		_ = r.Register(name, v.FieldByIndex(f.Index).Interface())
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
	}
	return false
}

func createField(v reflect.Value, f reflect.StructField, metric string) error {
	var value any
	switch f.Type {
	case counterType:
		value = metrics.NewCounter()

	case functionalGaugeType:
		fn, err := getGaugeFunction[int64](v, f.Name)
		if err != nil {
			return err
		}
		value = metrics.NewFunctionalGauge(fn)

	case gaugeType:
		value = metrics.NewGauge()

	case functionalGaugeFloat64Type:
		fn, err := getGaugeFunction[float64](v, f.Name)
		if err != nil {
			return err
		}
		value = metrics.NewFunctionalGaugeFloat64(fn)

	case gaugeFloat64Type:
		value = metrics.NewGaugeFloat64()

	case histogramType:
		if sample := f.Tag.Get(MetricSampleTag); sample != "" {
			s, err := parseSample(sample)
			if err != nil {
				return err
			}
			value = metrics.NewHistogram(s)
		} else {
			value = metrics.NewHistogram(metrics.NewExpDecaySample(DefaultReservoirSize, DefaultExpDecayAlpha))
		}

	case meterType:
		value = metrics.NewMeter()

	case timerType:
		if sample := f.Tag.Get(MetricSampleTag); sample != "" {
			s, err := parseSample(sample)
			if err != nil {
				return err
			}
			value = metrics.NewCustomTimer(metrics.NewHistogram(s), metrics.NewMeter())
		} else {
			value = metrics.NewTimer()
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

func parseSample(s string) (metrics.Sample, error) {
	parts := strings.Split(strings.ToLower(s), ",")
	switch parts[0] {
	case "uniform":
		return parseUniformSample(parts)
	case "expdecay":
		return parseExpDecaySample(parts)
	default:
		return nil, fmt.Errorf("invalid sample type")
	}
	return metrics.NilSample{}, nil
}

func parseUniformSample(parts []string) (metrics.Sample, error) {
	switch len(parts) {
	case 1:
		return metrics.NewUniformSample(DefaultReservoirSize), nil
	case 2:
		rs, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid uniform sample: reservoir: %w", err)
		}
		return metrics.NewUniformSample(rs), nil
	}
	return nil, fmt.Errorf("invalid uniform sample")
}

func parseExpDecaySample(parts []string) (metrics.Sample, error) {
	switch len(parts) {
	case 1:
		return metrics.NewExpDecaySample(DefaultReservoirSize, DefaultExpDecayAlpha), nil
	case 3:
		rs, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid expdecay sample: reservoir: %w", err)
		}
		alpha, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid expdecay sample: alpha: %w", err)
		}
		return metrics.NewExpDecaySample(rs, alpha), nil
	}
	return nil, fmt.Errorf("invalid expdecay sample")
}
