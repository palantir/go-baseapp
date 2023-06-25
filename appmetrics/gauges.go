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
	isField := false

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
		isField = true
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

	if isField {
		// If the function is a field, return a wrapper that calls the current
		// field value at the time of the the call. This is because the field
		// value is nil when we discover the function as part of New()
		return func() N { return m.Call(nil)[0].Interface().(N) }, nil
	}
	return m.Interface().(F), nil
}
