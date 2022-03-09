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

package baseapp

import (
	"net/http"
	"time"

	"github.com/bluekeyes/hatpear"
	"github.com/rcrowley/go-metrics"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/filters"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// DefaultMiddleware returns the default middleware stack. The stack:
//
//  - Adds a logger to request contexts
//  - Adds a metrics registry to request contexts
//  - Adds a request ID to all requests and responses
//  - Extracts Telemetry headers from requests
//  - Logs and records metrics for all requests
//  - Handles errors returned by route handlers
//  - Recovers from panics in route handlers
//
// All components are exported so users can select individual middleware to
// build their own stack if desired.
func DefaultMiddleware(logger zerolog.Logger, registry metrics.Registry) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		hlog.NewHandler(logger),
		NewMetricsHandler(registry),
		hlog.RequestIDHandler("rid", "X-Request-ID"),
		NewTelemetryHandler(DefaultTelemetryHandlerOptions...),
		NewTraceIDHandler("trace_id"),
		AccessHandler(RecordRequest),
		hatpear.Catch(HandleRouteError),
		hatpear.Recover(),
	}
}

// NewMetricsHandler returns middleware that add the given metrics registry to
// the request context.
func NewMetricsHandler(registry metrics.Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(WithMetricsCtx(r.Context(), registry))
			next.ServeHTTP(w, r)
		})
	}
}

// LogRequest is an AccessCallback that logs request information.
func LogRequest(r *http.Request, status int, size int64, elapsed time.Duration) {
	hlog.FromRequest(r).Info().
		Str("method", r.Method).
		Str("path", r.URL.String()).
		Str("client_ip", r.RemoteAddr).
		Int("status", status).
		Int64("size", size).
		Dur("elapsed", elapsed).
		Str("user_agent", r.UserAgent()).
		Msg("http_request")
}

// RecordRequest is an AccessCallback that logs request information and
// records request metrics.
func RecordRequest(r *http.Request, status int, size int64, elapsed time.Duration) {
	LogRequest(r, status, size, elapsed)
	CountRequest(r, status, size, elapsed)
}

type AccessCallback func(r *http.Request, status int, size int64, duration time.Duration)

// AccessHandler returns a handler that call f after each request.
func AccessHandler(f AccessCallback) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := WrapWriter(w)
			next.ServeHTTP(wrapped, r)
			f(r, wrapped.Status(), wrapped.BytesWritten(), time.Since(start))
		})
	}
}

// NewTelemetryHandler returns middleware that adds an OpenTelemetry span to the request if a tracing provider is set.
func NewTelemetryHandler(options ...otelhttp.Option) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := ""
			if xid, ok := hlog.IDFromRequest(r); ok {
				requestID = xid.String()
			}

			allOptions := []otelhttp.Option{
				otelhttp.WithSpanOptions(
					trace.WithAttributes(
						attribute.String("request.id", requestID))),
			}
			allOptions = append(allOptions, options...)

			h := otelhttp.NewHandler(next, r.Host, allOptions...)
			h.ServeHTTP(w, r)
		})
	}
}

// NewTraceIDHandler returns middleware that adds the OpenTelemetry trace ID to the request logger.
func NewTraceIDHandler(fieldKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if fieldKey == "" {
				// If there's no field key, don't log. This is the same behaviour as the zerolog request ID handler
				next.ServeHTTP(w, r)
				return
			}

			// Ensure the Trace ID is in all logs
			ctx := r.Context()
			log := zerolog.Ctx(ctx)
			span := trace.SpanFromContext(ctx)
			log.UpdateContext(func(c zerolog.Context) zerolog.Context {
				if span.SpanContext().TraceID().IsValid() {
					return c.Str(fieldKey, span.SpanContext().TraceID().String())
				}

				return c
			})

			next.ServeHTTP(w, r)
		})
	}
}

// DefaultOTelFilters is a set of common things that are evaluated to decide if a request should be excluded
// from OpenTelemetry tracing, for example a path starting with /ping.
var DefaultOTelFilters = filters.None(
	filters.PathPrefix("/ping"),
	filters.PathPrefix("/api/ping"),
	filters.PathPrefix("/health"),
	filters.PathPrefix("/api/health"),
	filters.PathPrefix("/debug"),
	filters.PathPrefix("/api/debug"),
	filters.PathPrefix("/pprof"),
	filters.PathPrefix("/api/pprof"),
)

// WithDefaultFilter wraps DefaultOTelFilters into an otelhttp.Option.
func WithDefaultFilter() otelhttp.Option {
	return otelhttp.WithFilter(DefaultOTelFilters)
}

// WithDefaultOTelPropagators sets a standard collection of propagators, in a specific order, that covers most of the types
// of headers that could include a trace ID.
//
// Ordering here is important as it's a "last one wins" scenario. Baggage will always be decoded from the headers as
// this is separate standard. Then, in the following order the trace ID will be created from the headers:
// 1. X-Amzn-Trace-Id header
// 2. W3C Trace Context headers
// 3. B3/Zipkin headers
func WithDefaultOTelPropagators() otelhttp.Option {
	return otelhttp.WithPropagators(
		propagation.NewCompositeTextMapPropagator(
			propagation.Baggage{},
			xray.Propagator{},
			propagation.TraceContext{},
			b3.New(),
		))
}

// DefaultTelemetryHandlerOptions is a slice of standard otelhttp.Option to use when handling requests.
var DefaultTelemetryHandlerOptions = []otelhttp.Option{
	otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
	WithDefaultFilter(),
	WithDefaultOTelPropagators(),
}
