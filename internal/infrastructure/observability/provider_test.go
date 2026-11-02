package observability

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"secunda-task-management-service/internal/infrastructure/config"
)

func TestProviderDisabledSignals(t *testing.T) {
	provider, err := New(
		context.Background(),
		config.App{Name: "test-service", Env: "test", Version: "1.0.0"},
		config.Metrics{Enabled: false},
		config.Tracing{Enabled: false},
		zap.NewNop(),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if provider.MetricsHandler() != nil {
		t.Fatal("MetricsHandler() != nil, want nil when metrics are disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := provider.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestProviderMetricsEnabled(t *testing.T) {
	provider, err := New(
		context.Background(),
		config.App{Name: "test-service", Env: "test", Version: "1.0.0"},
		config.Metrics{Enabled: true, Addr: ":9090", Path: "/metrics"},
		config.Tracing{Enabled: false},
		zap.NewNop(),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if provider.MetricsHandler() == nil {
		t.Fatal("MetricsHandler() = nil, want handler")
	}
	if provider.Metrics == nil {
		t.Fatal("Metrics = nil, want instruments")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := provider.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
