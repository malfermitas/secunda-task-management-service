package observability

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestMetricsRecordsCacheSignals(t *testing.T) {
	registry := prometheus.NewRegistry()
	exporter, err := promexporter.New(
		promexporter.WithRegisterer(registry),
		promexporter.WithNamespace(metricsNamespace),
	)
	if err != nil {
		t.Fatalf("create prometheus exporter: %v", err)
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	metrics, err := NewMetrics(provider)
	if err != nil {
		t.Fatalf("NewMetrics() error = %v", err)
	}

	ctx := context.Background()
	metrics.CacheRequest(ctx, "hit")
	metrics.CacheOperation(ctx, "get", "success")
	metrics.CacheInvalidation(ctx)

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	names := make(map[string]bool, len(families))
	for _, family := range families {
		names[family.GetName()] = true
	}

	for _, prefix := range []string{
		"secunda_cache_requests",
		"secunda_cache_operations",
		"secunda_cache_invalidations",
	} {
		if !hasMetricPrefix(names, prefix) {
			t.Errorf("metric with prefix %q not found in %v", prefix, names)
		}
	}
}

func hasMetricPrefix(names map[string]bool, prefix string) bool {
	for name := range names {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
