package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"secunda-task-management-service/internal/infrastructure/config"
	taskapp "secunda-task-management-service/internal/task/app"
	taskdomain "secunda-task-management-service/internal/task/domain"
)

const keyPrefix = "tasks:v1"

// Metrics is the cache instrumentation port.
type Metrics interface {
	CacheRequest(ctx context.Context, result string)
	CacheOperation(ctx context.Context, operation, result string)
	CacheInvalidation(ctx context.Context)
}

// Dependencies contains optional cache instrumentation.
type Dependencies struct {
	Logger         *zap.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Metrics        Metrics
}

// Cache is the Redis adapter for task list caching.
type Cache struct {
	rdb     *redis.Client
	ttl     time.Duration
	log     *zap.Logger
	metrics Metrics
}

var _ taskapp.Cache = (*Cache)(nil)

// NewCache creates a Redis-backed task cache. The cache is fail-open: an
// unreachable Redis only disables caching and never prevents startup.
func NewCache(cfg config.Redis, ttl time.Duration, deps Dependencies) *Cache {
	logger := deps.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	if deps.TracerProvider != nil {
		if err := redisotel.InstrumentTracing(
			rdb,
			redisotel.WithTracerProvider(deps.TracerProvider),
		); err != nil {
			logger.Warn("instrument redis tracing failed", zap.Error(err))
		}
	}
	if deps.MeterProvider != nil {
		if err := redisotel.InstrumentMetrics(
			rdb,
			redisotel.WithMeterProvider(deps.MeterProvider),
		); err != nil {
			logger.Warn("instrument redis metrics failed", zap.Error(err))
		}
	}

	cache := &Cache{rdb: rdb, ttl: ttl, log: logger, metrics: deps.Metrics}

	timeout := cfg.DialTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		logger.Warn("redis unavailable; task cache disabled", zap.Error(err), zap.String("addr", cfg.Addr))
	}

	return cache
}

// Close releases the Redis client.
func (c *Cache) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

// GetTasks returns a cached page and the page key observed for this lookup.
// Any Redis or decoding error is a miss; the key may still be returned so the
// caller can replace a corrupt value.
func (c *Cache) GetTasks(ctx context.Context, teamID int64, filterKey string) (*taskapp.TaskList, taskapp.CacheKey, bool) {
	if c == nil || c.rdb == nil {
		return nil, "", false
	}

	key, err := c.key(ctx, teamID, filterKey)
	if err != nil {
		c.warn("build cache key failed", teamID, err)
		c.observeRequest(ctx, "error")
		c.observeOperation(ctx, "get", "error")
		return nil, "", false
	}

	data, err := c.rdb.Get(ctx, string(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		c.observeRequest(ctx, "miss")
		c.observeOperation(ctx, "get", "success")
		c.setCacheHit(ctx, false)
		return nil, key, false
	}
	if err != nil {
		c.warn("read task cache failed", teamID, err)
		c.observeRequest(ctx, "error")
		c.observeOperation(ctx, "get", "error")
		return nil, "", false
	}

	var list taskapp.TaskList
	if err := json.Unmarshal(data, &list); err != nil {
		c.warn("decode task cache failed", teamID, err)
		c.observeRequest(ctx, "error")
		c.observeOperation(ctx, "get", "error")
		return nil, key, false
	}
	if list.Items == nil {
		list.Items = []taskdomain.Task{}
	}

	c.observeRequest(ctx, "hit")
	c.observeOperation(ctx, "get", "success")
	c.setCacheHit(ctx, true)
	return &list, key, true
}

// SetTasks stores a page under the key returned by GetTasks. Any Redis or
// encoding error is ignored.
func (c *Cache) SetTasks(ctx context.Context, teamID int64, key taskapp.CacheKey, list *taskapp.TaskList) {
	if c == nil {
		return
	}
	if c.rdb == nil || key == "" || list == nil {
		c.observeOperation(ctx, "set", "skipped")
		return
	}

	data, err := json.Marshal(list)
	if err != nil {
		c.warn("encode task cache failed", teamID, err)
		c.observeOperation(ctx, "set", "error")
		return
	}

	if err := c.rdb.Set(ctx, string(key), data, c.ttl).Err(); err != nil {
		c.warn("write task cache failed", teamID, err)
		c.observeOperation(ctx, "set", "error")
		return
	}
	c.observeOperation(ctx, "set", "success")
}

// InvalidateTeam advances the team's cache generation, making every page
// cached before the successful database write unreachable.
func (c *Cache) InvalidateTeam(ctx context.Context, teamID int64) {
	if c == nil || c.rdb == nil {
		return
	}

	if err := c.rdb.Incr(ctx, generationKey(teamID)).Err(); err != nil {
		c.warn("invalidate task cache failed", teamID, err)
		c.observeOperation(ctx, "invalidate", "error")
		return
	}
	c.observeOperation(ctx, "invalidate", "success")
	if c.metrics != nil {
		c.metrics.CacheInvalidation(ctx)
	}
}

func (c *Cache) key(ctx context.Context, teamID int64, filterKey string) (taskapp.CacheKey, error) {
	generation, err := c.rdb.Get(ctx, generationKey(teamID)).Int64()
	if errors.Is(err, redis.Nil) {
		generation = 0
	} else if err != nil {
		return "", err
	}

	return taskapp.CacheKey(fmt.Sprintf("%s:team:%d:gen:%d:%s", keyPrefix, teamID, generation, filterKey)), nil
}

func generationKey(teamID int64) string {
	return fmt.Sprintf("%s:team:%d:gen", keyPrefix, teamID)
}

func (c *Cache) observeRequest(ctx context.Context, result string) {
	if c != nil && c.metrics != nil {
		c.metrics.CacheRequest(ctx, result)
	}
}

func (c *Cache) observeOperation(ctx context.Context, operation, result string) {
	if c != nil && c.metrics != nil {
		c.metrics.CacheOperation(ctx, operation, result)
	}
}

func (c *Cache) setCacheHit(ctx context.Context, hit bool) {
	trace.SpanFromContext(ctx).SetAttributes(attribute.Bool("cache.hit", hit))
}

func (c *Cache) warn(message string, teamID int64, err error) {
	if c == nil || c.log == nil {
		return
	}
	c.log.Warn(message, zap.Error(err), zap.Int64("team_id", teamID))
}
