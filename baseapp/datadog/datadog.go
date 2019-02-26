// Copyright 2018 Palantir Technologies, Inc.
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

// Package datadog defines configuration and functions for emitting metrics to
// Datadog using the DogStatd protocol.
//
// It supports a special format for metric names to add metric-specific tags:
//
//   metricName[tag1,tag2:value2,...]
//
// Global tags for all metrics can be set in the configuration.
package datadog

import (
	"context"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/pkg/errors"
	"github.com/rcrowley/go-metrics"

	"github.com/palantir/go-baseapp/baseapp"
)

const (
	DefaultAddress  = "127.0.0.1:8125"
	DefaultInterval = 10 * time.Second
)

type Config struct {
	Address  string        `yaml:"address" json:"address"`
	Interval time.Duration `yaml:"interval" json:"interval"`
	Tags     []string      `yaml:"tags" json:"tags"`
}

// StartEmitter starts a goroutine that emits metrics from the server's
// registry to the configured DogStatsd endpoint.
func StartEmitter(s *baseapp.Server, c Config) error {
	if c.Address == "" {
		c.Address = DefaultAddress
	}
	if c.Interval == 0 {
		c.Interval = DefaultInterval
	}

	client, err := statsd.New(c.Address)
	if err != nil {
		return errors.Wrap(err, "datadog: failed to create client")
	}
	client.Tags = append(client.Tags, c.Tags...)

	emitter := NewEmitter(client, s.Registry())
	go emitter.Emit(context.Background(), c.Interval)

	return nil
}

type Emitter struct {
	client   *statsd.Client
	registry metrics.Registry
}

func NewEmitter(client *statsd.Client, registry metrics.Registry) *Emitter {
	return &Emitter{
		registry: registry,
		client:   client,
	}
}

func (e *Emitter) Emit(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			e.EmitOnce()
		case <-ctx.Done():
			return
		}
	}
}

func (e *Emitter) EmitOnce() {
	e.registry.Each(func(name string, metric interface{}) {
		name, tags := tagsFromName(name)

		switch m := metric.(type) {
		case metrics.Counter:
			e.client.Count(name, m.Count(), tags, 1)

		case metrics.Gauge:
			e.client.Gauge(name, float64(m.Value()), tags, 1)

		case metrics.GaugeFloat64:
			e.client.Gauge(name, m.Value(), tags, 1)

		case metrics.Histogram:
			ms := m.Snapshot()
			e.client.Gauge(name+".avg", ms.Mean(), tags, 1)
			e.client.Gauge(name+".count", float64(ms.Count()), tags, 1)
			e.client.Gauge(name+".max", float64(ms.Max()), tags, 1)
			e.client.Gauge(name+".median", ms.Percentile(0.5), tags, 1)
			e.client.Gauge(name+".min", float64(ms.Min()), tags, 1)
			e.client.Gauge(name+".sum", float64(ms.Sum()), tags, 1)
			e.client.Gauge(name+".95percentile", ms.Percentile(0.95), tags, 1)

		case metrics.Meter:
			ms := m.Snapshot()
			e.client.Gauge(name+".avg", ms.RateMean(), tags, 1)
			e.client.Gauge(name+".count", float64(ms.Count()), tags, 1)
			e.client.Gauge(name+".rate1", ms.Rate1(), tags, 1)
			e.client.Gauge(name+".rate5", ms.Rate5(), tags, 1)
			e.client.Gauge(name+".rate15", ms.Rate15(), tags, 1)

		case metrics.Timer:
			ms := m.Snapshot()
			e.client.Gauge(name+".avg", ms.Mean(), tags, 1)
			e.client.Gauge(name+".count", float64(ms.Count()), tags, 1)
			e.client.Gauge(name+".max", float64(ms.Max()), tags, 1)
			e.client.Gauge(name+".median", ms.Percentile(0.5), tags, 1)
			e.client.Gauge(name+".min", float64(ms.Min()), tags, 1)
			e.client.Gauge(name+".sum", float64(ms.Sum()), tags, 1)
			e.client.Gauge(name+".95percentile", ms.Percentile(0.95), tags, 1)
		}
	})
}

func tagsFromName(name string) (string, []string) {
	start := strings.IndexRune(name, '[')
	if start < 0 || name[len(name)-1] != ']' {
		return name, nil
	}
	return name[:start], strings.Split(name[start+1:len(name)-1], ",")
}
