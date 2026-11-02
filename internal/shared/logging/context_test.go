package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestWithLoggerAndFromContext(t *testing.T) {
	var buffer bytes.Buffer
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buffer),
		zapcore.DebugLevel,
	))

	ctx := WithLogger(context.Background(), logger)
	FromContext(ctx).Info("hello")

	if buffer.Len() == 0 {
		t.Fatal("FromContext() logger wrote no output")
	}
}

func TestWithTraceAddsSpanIdentifiers(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	ctx, span := provider.Tracer("test").Start(context.Background(), "request")
	defer span.End()

	var buffer bytes.Buffer
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buffer),
		zapcore.DebugLevel,
	))
	WithTrace(ctx, logger).Info("hello")

	var entry map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
		t.Fatalf("decode log: %v", err)
	}
	spanContext := span.SpanContext()
	if entry["trace_id"] != spanContext.TraceID().String() {
		t.Errorf("trace_id = %v, want %s", entry["trace_id"], spanContext.TraceID())
	}
	if entry["span_id"] != spanContext.SpanID().String() {
		t.Errorf("span_id = %v, want %s", entry["span_id"], spanContext.SpanID())
	}
}

func TestFromContextFallsBackToGlobalLogger(t *testing.T) {
	if logger := FromContext(context.Background()); logger == nil {
		t.Fatal("FromContext() = nil, want global logger")
	}
}
