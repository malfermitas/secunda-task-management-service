package mysql

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/XSAM/otelsql"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"secunda-task-management-service/internal/infrastructure/config"
)

// DB wraps the SQL pool and its OpenTelemetry stats registration.
type DB struct {
	*sqlx.DB
	stats metric.Registration
}

// Open creates and verifies an instrumented MySQL connection pool.
func Open(
	ctx context.Context,
	cfg config.DB,
	logger *zap.Logger,
	tracerProvider trace.TracerProvider,
	meterProvider metric.MeterProvider,
) (*DB, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	driverCfg := mysqlDriver.NewConfig()
	driverCfg.User = cfg.User
	driverCfg.Passwd = cfg.Password
	driverCfg.Net = "tcp"
	driverCfg.Addr = net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	driverCfg.DBName = cfg.Name
	driverCfg.ParseTime = true
	driverCfg.Loc = time.UTC
	driverCfg.Timeout = cfg.ConnectTimeout
	driverCfg.Params = map[string]string{
		"charset":   "utf8mb4",
		"time_zone": "'+00:00'",
	}

	sqlDB, err := otelsql.Open(
		"mysql",
		driverCfg.FormatDSN(),
		otelsql.WithTracerProvider(tracerProvider),
		otelsql.WithMeterProvider(meterProvider),
		otelsql.WithAttributes(semconv.DBSystemNameMySQL),
		otelsql.WithSpanOptions(otelsql.SpanOptions{DisableQuery: true}),
	)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	db := &DB{DB: sqlx.NewDb(sqlDB, "mysql")}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	registration, err := otelsql.RegisterDBStatsMetrics(
		sqlDB,
		otelsql.WithMeterProvider(meterProvider),
		otelsql.WithAttributes(semconv.DBSystemNameMySQL),
	)
	if err != nil {
		logger.Warn("register mysql pool metrics failed", zap.Error(err))
	} else {
		db.stats = registration
	}

	return db, nil
}

// Close unregisters pool metrics and closes the SQL pool.
func (db *DB) Close() error {
	if db == nil {
		return nil
	}
	if db.stats != nil {
		_ = db.stats.Unregister()
	}
	return db.DB.Close()
}
