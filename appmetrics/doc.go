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

// Package appmetrics creates and registers metrics structs.
//
// Applications that report metrics often want to define those metrics in a
// common type or package so they are easy to use in other parts of the
// application. The appmetrics package provides a way to easily define,
// initialize, and register these shared metric structs.
//
// A metrics struct contains one or more fields of a supported metric interface
// type that have the "metric" tag giving the metric's name in a registry. The
// metric types and the registry are defined by the [go-metrics] package.
//
// For global metrics, this struct is often exported as a field:
//
//	// in the app's "metrics" package
//	type Metrics struct {
//		Errors        metrics.Counter `metric:"errors"`
//		ActiveWorkers metrics.Gauge   `metric:"active_workers"`
//	}
//
//	var M *Metrics
//
//	func init() {
//		M = appmetrics.New()
//		appmetrics.Register(metrics.DefaultRegistry, M)
//	}
//
//	// in a different package
//	metrics.M.Errors.Inc(1)
//	metrics.M.ActiveWorkers.Update(len(workers))
//
// [go-metrics]: https://pkg.go.dev/github.com/rcrowley/go-metrics
package appmetrics
