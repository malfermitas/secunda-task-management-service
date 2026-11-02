// Package logging provides request-scoped logger helpers shared across layers.
package logging

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type contextKey struct{}

// WithLogger stores a request-scoped logger in the context.
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext returns the request-scoped logger or the global zap logger.
func FromContext(ctx context.Context) *zap.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(contextKey{}).(*zap.Logger); ok && logger != nil {
			return logger
		}
	}
	return zap.L()
}

// WithTrace enriches a logger with the active OpenTelemetry trace identifiers.
func WithTrace(ctx context.Context, logger *zap.Logger) *zap.Logger {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return logger
	}
	return logger.With(
		zap.String("trace_id", spanContext.TraceID().String()),
		zap.String("span_id", spanContext.SpanID().String()),
	)
}
