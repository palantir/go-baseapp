// Copyright 2020 Palantir Technologies, Inc.
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

package opentelemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/standard"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	DefaultAddress = "127.0.0.1:55680"
)

type Config struct {
	Address          string            `yaml:"address" json:"address"`
	Insecure         bool              `yaml:"insecure" json:"insecure"`
	ServiceName      string            `yaml:"service_name" json:"service_name"`
	ServiceNamespace string            `yaml:"service_namespace" json:"service_namespace"`
	ServiceVersion   string            `yaml:"service_version" json:"service_version"`
	ResourceTags     map[string]string `yaml:"resource_tags" json:"resource_tags"`
}

func StartTracingExporter(c Config) (*otlp.Exporter, error) {
	if c.ServiceNamespace == "" {
		return nil, fmt.Errorf("a value for 'ServiceNamespace' must be provided")
	}
	if c.ServiceName == "" {
		return nil, fmt.Errorf("a value for 'ServiceName' must be provided")
	}
	if c.ServiceVersion == "" {
		return nil, fmt.Errorf("a value for 'ServiceVersion' must be provided")
	}

	if c.Address == "" {
		c.Address = DefaultAddress
	}

	exporterOptions := []otlp.ExporterOption{
		otlp.WithAddress(c.Address),
	}
	if c.Insecure {
		exporterOptions = append(exporterOptions, otlp.WithInsecure())
	}

	exp, err := otlp.NewExporter(exporterOptions...)
	if err != nil {
		return nil, err
	}

	attributes := []kv.KeyValue{
		standard.ServiceNameKey.String(c.ServiceName),
		standard.ServiceNamespaceKey.String(c.ServiceNamespace),
		standard.ServiceVersionKey.String(c.ServiceVersion),
	}
	for key, value := range c.ResourceTags {
		attributes = append(attributes, kv.String(key, value))
	}

	traceProvider, err := sdktrace.NewProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithResource(resource.New(attributes...)),
		sdktrace.WithSyncer(exp),
	)
	if err != nil {
		return nil, err
	}

	global.SetTraceProvider(traceProvider)
	return exp, nil
}
