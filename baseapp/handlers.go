package baseapp

import (
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/instrumentation/othttp"
	"net/http"
)

// TelemetryHandler wraps `othttp.WithRouteTag` to use the route tag value as the name of the span. By default the span
// name will be the name of the service defined in the exporters.
func TelemetryHandler(route string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		span.SetName(route)

		handler := othttp.WithRouteTag(route, h)
		handler.ServeHTTP(w, r)
	})
}