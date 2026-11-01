package mysql

import (
	"context"
	"fmt"
	"net"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"secunda-task-management-service/internal/infrastructure/config"
)

// Open creates and verifies a MySQL connection pool.
func Open(ctx context.Context, cfg config.DB) (*sqlx.DB, error) {
	driverCfg := mysqlDriver.NewConfig()
	driverCfg.User = cfg.User
	driverCfg.Passwd = cfg.Password
	driverCfg.Net = "tcp"
	driverCfg.Addr = net.JoinHostPort(cfg.Host, cfg.Port)
	driverCfg.DBName = cfg.Name
	driverCfg.ParseTime = true
	driverCfg.Loc = time.UTC
	driverCfg.Timeout = 5 * time.Second
	driverCfg.Params = map[string]string{
		"charset":   "utf8mb4",
		"time_zone": "'+00:00'",
	}

	db, err := sqlx.Open("mysql", driverCfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return db, nil
}
