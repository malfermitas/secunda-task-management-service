package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestOTelGinEmitsNamespacedHTTPMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := prometheus.NewRegistry()
	exporter, err := promexporter.New(
		promexporter.WithRegisterer(registry),
		promexporter.WithNamespace(metricsNamespace),
	)
	if err != nil {
		t.Fatalf("create prometheus exporter: %v", err)
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	previous := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() { otel.SetMeterProvider(previous) })

	engine := gin.New()
	engine.Use(otelgin.Middleware("test-service"))
	engine.GET("/tasks/:id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	engine.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/tasks/42", nil),
	)

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	names := make(map[string]bool, len(families))
	for _, family := range families {
		names[family.GetName()] = true
	}

	if !hasMetricPrefix(names, "secunda_http_server_request_duration_seconds") {
		t.Fatalf("HTTP duration metric not found in %v", names)
	}
	for name := range names {
		if strings.Contains(name, "/tasks/42") {
			t.Errorf("metric name contains raw path: %s", name)
		}
	}
}
