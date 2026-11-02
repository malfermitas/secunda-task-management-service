// Package observability configures OpenTelemetry tracing and Prometheus metrics.
package observability

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"secunda-task-management-service/internal/infrastructure/config"
)

const metricsNamespace = "secunda"

// Provider owns the OpenTelemetry providers and resource shared by the app.
type Provider struct {
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Metrics        *Metrics
	registry       *prometheus.Registry
	metricsEnabled bool
	shutdown       []func(context.Context) error
}

// New configures tracing and metrics. Observability failures never prevent the
// application from starting: disabled signals use no-op providers.
func New(
	ctx context.Context,
	app config.App,
	metricsCfg config.Metrics,
	tracingCfg config.Tracing,
	logger *zap.Logger,
) (*Provider, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(app.Name),
			semconv.ServiceVersion(app.Version),
			semconv.DeploymentEnvironmentName(app.Env),
		),
	)
	if err != nil {
		return nil, err
	}

	provider := &Provider{
		TracerProvider: tracenoop.NewTracerProvider(),
		MeterProvider:  metricnoop.NewMeterProvider(),
		registry:       prometheus.NewRegistry(),
		metricsEnabled: metricsCfg.Enabled,
	}

	if tracingCfg.Enabled {
		tracerProvider, shutdown, err := newTracerProvider(ctx, res, tracingCfg)
		if err != nil {
			return nil, err
		}
		provider.TracerProvider = tracerProvider
		provider.shutdown = append(provider.shutdown, shutdown)
	}

	provider.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	if metricsCfg.Enabled {
		meterProvider, shutdown, err := newMeterProvider(provider.registry, res)
		if err != nil {
			return nil, err
		}
		provider.MeterProvider = meterProvider
		provider.shutdown = append(provider.shutdown, shutdown)
	}

	otel.SetTracerProvider(provider.TracerProvider)
	otel.SetMeterProvider(provider.MeterProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Warn("opentelemetry error", zap.Error(err))
	}))

	metrics, err := NewMetrics(provider.MeterProvider)
	if err != nil {
		return nil, err
	}
	provider.Metrics = metrics

	return provider, nil
}

// MetricsHandler returns the Prometheus scrape handler, or nil when disabled.
func (p *Provider) MetricsHandler() http.Handler {
	if p == nil || !p.metricsEnabled {
		return nil
	}
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// Shutdown flushes telemetry providers in reverse startup order.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}

	var errs []error
	for i := len(p.shutdown) - 1; i >= 0; i-- {
		if err := p.shutdown[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func newTracerProvider(
	ctx context.Context,
	res *resource.Resource,
	cfg config.Tracing,
) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithTimeout(5 * time.Second),
	}
	if cfg.Insecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		return nil, nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(cfg.SampleRatio),
		)),
	)
	return provider, provider.Shutdown, nil
}

func newMeterProvider(
	registry prometheus.Registerer,
	res *resource.Resource,
) (*sdkmetric.MeterProvider, func(context.Context) error, error) {
	exporter, err := promexporter.New(
		promexporter.WithRegisterer(registry),
		promexporter.WithNamespace(metricsNamespace),
	)
	if err != nil {
		return nil, nil, err
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
		sdkmetric.WithResource(res),
	)
	return provider, provider.Shutdown, nil
}
