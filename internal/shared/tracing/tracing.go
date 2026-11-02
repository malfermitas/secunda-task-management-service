// Package tracing provides small helpers for application-level spans.
package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "secunda-task-management-service"

// Start begins an application span and applies optional attributes.
func Start(
	ctx context.Context,
	name string,
	attributes ...attribute.KeyValue,
) (context.Context, trace.Span) {
	ctx, span := otel.Tracer(instrumentationName).Start(ctx, name)
	if len(attributes) > 0 {
		span.SetAttributes(attributes...)
	}
	return ctx, span
}

// RecordError marks a span as failed when err is non-nil.
func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
