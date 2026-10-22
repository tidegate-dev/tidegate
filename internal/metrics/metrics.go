// Package metrics records OpenTelemetry metrics and serves them in the
// Prometheus text format.
package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Setup sets the global meter provider to one that records the Go runtime
// metrics. It returns a handler that serves the metrics and a function that
// shuts the provider down.
func Setup() (http.Handler, func(context.Context) error, error) {
	// The default registry has the Prometheus client's own Go collector,
	// which would report the runtime metrics a second time under other names.
	registry := prometheus.NewRegistry()

	exporter, err := otelprometheus.New(otelprometheus.WithRegisterer(registry))
	if err != nil {
		return nil, nil, fmt.Errorf("creating the Prometheus exporter: %w", err)
	}

	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(provider)

	err = runtime.Start(runtime.WithMeterProvider(provider))
	if err != nil {
		return nil, nil, fmt.Errorf("recording the Go runtime metrics: %w", err)
	}

	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{}), provider.Shutdown, nil
}
