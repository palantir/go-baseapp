package appmetrics

import (
	"fmt"
	"reflect"

	"github.com/rcrowley/go-metrics"
)

const (
	GaugeFunctionPrefix = "Compute"
)

// FunctionalGauge is a [metrics.Gauge] that computes its value by calling a
// function.
//
// A FunctionalGauge cannot be used as a [Tagged] metric.
type FunctionalGauge interface {
	Snapshot() metrics.Gauge
	Value() int64
}

// FunctionalGaugeFloat64 is a [metrics.GaugeFloat64] that computes its value
// by calling a function.
//
// A FunctionalGaugeFloat64 cannot be used as a [Tagged] metric.
type FunctionalGaugeFloat64 interface {
	Snapshot() metrics.GaugeFloat64
	Value() float64
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
