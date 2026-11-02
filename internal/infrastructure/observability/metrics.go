package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const meterName = "secunda-task-management-service"

// Metrics owns the application metric instruments. HTTP server metrics are
// emitted by otelgin using the OpenTelemetry HTTP semantic conventions:
//
//	secunda_http_server_request_duration_seconds
//	secunda_http_server_request_body_size_bytes
//	secunda_http_server_response_body_size_bytes
//
// Dashboard: histogram_quantile(0.99, sum(rate(secunda_http_server_request_duration_seconds_bucket[5m])) by (le, http_route))
// Alert:     sum(rate(secunda_http_server_request_duration_seconds_count{http_response_status_code=~"5.."}[5m]))
//
//	/ sum(rate(secunda_http_server_request_duration_seconds_count[5m])) > 0.01
type Metrics struct {
	// Dashboard: sum(rate(secunda_cache_requests_total[5m])) by (cache_result)
	// Alert:     sum(rate(secunda_cache_requests_total{cache_result="error"}[5m])) > 0.5
	cacheRequests metric.Int64Counter

	// Dashboard: sum(rate(secunda_cache_operations_total[5m])) by (cache_operation, cache_result)
	cacheOperations metric.Int64Counter

	// Dashboard: sum(rate(secunda_cache_invalidations_total[5m]))
	cacheInvalidations metric.Int64Counter
}

// NewMetrics creates the application metric instruments.
func NewMetrics(provider metric.MeterProvider) (*Metrics, error) {
	meter := provider.Meter(meterName)

	cacheRequests, err := meter.Int64Counter(
		"cache.requests",
		metric.WithDescription("Task cache requests by result."),
	)
	if err != nil {
		return nil, err
	}

	cacheOperations, err := meter.Int64Counter(
		"cache.operations",
		metric.WithDescription("Task cache operations by operation and result."),
	)
	if err != nil {
		return nil, err
	}

	cacheInvalidations, err := meter.Int64Counter(
		"cache.invalidations",
		metric.WithDescription("Task cache generation invalidations."),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		cacheRequests:      cacheRequests,
		cacheOperations:    cacheOperations,
		cacheInvalidations: cacheInvalidations,
	}, nil
}

// CacheRequest records a cache lookup result.
func (m *Metrics) CacheRequest(ctx context.Context, result string) {
	if m == nil {
		return
	}
	m.cacheRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("cache.result", result),
	))
}

// CacheOperation records a cache operation result.
func (m *Metrics) CacheOperation(ctx context.Context, operation, result string) {
	if m == nil {
		return
	}
	m.cacheOperations.Add(ctx, 1, metric.WithAttributes(
		attribute.String("cache.operation", operation),
		attribute.String("cache.result", result),
	))
}

// CacheInvalidation records a team cache invalidation.
func (m *Metrics) CacheInvalidation(ctx context.Context) {
	if m == nil {
		return
	}
	m.cacheInvalidations.Add(ctx, 1)
}
