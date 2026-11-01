package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"secunda-task-management-service/internal/infrastructure/config"
	taskapp "secunda-task-management-service/internal/task/app"
	taskdomain "secunda-task-management-service/internal/task/domain"
)

const keyPrefix = "tasks:v1"

// Cache is the Redis adapter for task list caching.
type Cache struct {
	rdb *redis.Client
	ttl time.Duration
	log *slog.Logger
}

var _ taskapp.Cache = (*Cache)(nil)

// NewCache creates a Redis-backed task cache. The cache is fail-open: an
// unreachable Redis only disables caching and never prevents startup.
func NewCache(cfg config.Redis, ttl time.Duration, log *slog.Logger) *Cache {
	if log == nil {
		log = slog.Default()
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

	cache := &Cache{rdb: rdb, ttl: ttl, log: log}

	timeout := cfg.DialTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		log.Warn("redis unavailable; task cache disabled", "error", err, "addr", cfg.Addr)
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
		return nil, "", false
	}

	data, err := c.rdb.Get(ctx, string(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, key, false
	}
	if err != nil {
		c.warn("read task cache failed", teamID, err)
		return nil, "", false
	}

	var list taskapp.TaskList
	if err := json.Unmarshal(data, &list); err != nil {
		c.warn("decode task cache failed", teamID, err)
		return nil, key, false
	}
	if list.Items == nil {
		list.Items = []taskdomain.Task{}
	}

	return &list, key, true
}

// SetTasks stores a page under the key returned by GetTasks. Any Redis or
// encoding error is ignored.
func (c *Cache) SetTasks(ctx context.Context, teamID int64, key taskapp.CacheKey, list *taskapp.TaskList) {
	if c == nil || c.rdb == nil || key == "" || list == nil {
		return
	}

	data, err := json.Marshal(list)
	if err != nil {
		c.warn("encode task cache failed", teamID, err)
		return
	}

	if err := c.rdb.Set(ctx, string(key), data, c.ttl).Err(); err != nil {
		c.warn("write task cache failed", teamID, err)
	}
}

// InvalidateTeam advances the team's cache generation, making every page
// cached before the successful database write unreachable.
func (c *Cache) InvalidateTeam(ctx context.Context, teamID int64) {
	if c == nil || c.rdb == nil {
		return
	}

	if err := c.rdb.Incr(ctx, generationKey(teamID)).Err(); err != nil {
		c.warn("invalidate task cache failed", teamID, err)
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

func (c *Cache) warn(message string, teamID int64, err error) {
	if c == nil || c.log == nil {
		return
	}
	c.log.Warn(message, "error", err, "team_id", teamID)
}
