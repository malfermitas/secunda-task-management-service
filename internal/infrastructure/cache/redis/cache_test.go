package redis

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"secunda-task-management-service/internal/infrastructure/config"
	"secunda-task-management-service/internal/shared"
	taskapp "secunda-task-management-service/internal/task/app"
	taskdomain "secunda-task-management-service/internal/task/domain"
)

func TestCacheSetAndGet(t *testing.T) {
	mr := miniredis.RunT(t)
	cache := newTestCache(t, mr, 5*time.Minute)
	ctx := context.Background()

	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	list := &taskapp.TaskList{
		Items: []taskdomain.Task{{
			ID:          10,
			TeamID:      7,
			Title:       "write cache tests",
			Description: shared.Some("cover Optional fields"),
			Status:      taskdomain.StatusTodo,
			CreatedBy:   2,
			AssigneeID:  shared.Some(int64(3)),
			CreatedAt:   now,
			UpdatedAt:   now,
			ClosedAt:    shared.None[time.Time](),
			Version:     4,
		}},
		Total:  shared.Some(int64(1)),
		Limit:  20,
		Offset: 0,
	}
	filterKey := "status:todo:assignee:3:limit:20:offset:0"
	key := missKey(t, cache, 7, filterKey)

	cache.SetTasks(ctx, 7, key, list)

	got, _, ok := cache.GetTasks(ctx, 7, filterKey)
	if !ok {
		t.Fatal("GetTasks() miss, want hit")
	}
	if !reflect.DeepEqual(got, list) {
		t.Fatalf("GetTasks() = %#v, want %#v", got, list)
	}

	if ttl := mr.TTL(string(key)); ttl <= 0 || ttl > 5*time.Minute {
		t.Fatalf("page TTL = %v, want (0, 5m]", ttl)
	}
}

func TestCacheKeyIsolationAndInvalidation(t *testing.T) {
	mr := miniredis.RunT(t)
	cache := newTestCache(t, mr, 5*time.Minute)
	ctx := context.Background()

	list := &taskapp.TaskList{Items: []taskdomain.Task{}, Total: shared.Some(int64(0))}
	todoKey := "status:todo:assignee:all:limit:20:offset:0"
	doneKey := "status:done:assignee:all:limit:20:offset:0"
	key := missKey(t, cache, 7, todoKey)

	cache.SetTasks(ctx, 7, key, list)

	if _, _, ok := cache.GetTasks(ctx, 7, todoKey); !ok {
		t.Fatal("same team/filter miss, want hit")
	}
	if _, _, ok := cache.GetTasks(ctx, 7, doneKey); ok {
		t.Fatal("different filter hit, want miss")
	}
	if _, _, ok := cache.GetTasks(ctx, 8, todoKey); ok {
		t.Fatal("different team hit, want miss")
	}

	cache.InvalidateTeam(ctx, 7)

	generation, err := mr.Get(generationKey(7))
	if err != nil {
		t.Fatalf("read generation: %v", err)
	}
	if generation != "1" {
		t.Fatalf("generation = %q, want 1", generation)
	}
	if _, _, ok := cache.GetTasks(ctx, 7, todoKey); ok {
		t.Fatal("cache hit after invalidation, want miss")
	}

	newKey := missKey(t, cache, 7, todoKey)
	cache.SetTasks(ctx, 7, newKey, list)
	if _, _, ok := cache.GetTasks(ctx, 7, todoKey); !ok {
		t.Fatal("cache miss after repopulation, want hit")
	}
}

func TestCacheSetKeepsInvalidatedGenerationUnreachable(t *testing.T) {
	mr := miniredis.RunT(t)
	cache := newTestCache(t, mr, 5*time.Minute)
	ctx := context.Background()

	list := &taskapp.TaskList{Total: shared.Some(int64(0))}
	filterKey := "status:all:assignee:all:limit:20:offset:0"
	staleKey := missKey(t, cache, 7, filterKey)

	cache.InvalidateTeam(ctx, 7)
	cache.SetTasks(ctx, 7, staleKey, list)

	if _, _, ok := cache.GetTasks(ctx, 7, filterKey); ok {
		t.Fatal("GetTasks() hit on a stale generation, want miss")
	}
}

func TestCacheTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	cache := newTestCache(t, mr, time.Minute)
	ctx := context.Background()

	filterKey := "status:all:assignee:all:limit:20:offset:0"
	key := missKey(t, cache, 7, filterKey)
	cache.SetTasks(ctx, 7, key, &taskapp.TaskList{Total: shared.Some(int64(0))})

	if _, _, ok := cache.GetTasks(ctx, 7, filterKey); !ok {
		t.Fatal("GetTasks() miss before TTL, want hit")
	}

	mr.FastForward(2 * time.Minute)

	if _, _, ok := cache.GetTasks(ctx, 7, filterKey); ok {
		t.Fatal("GetTasks() hit after TTL, want miss")
	}
}

func TestCacheFailsOpen(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()

	cache := NewCache(
		config.Redis{
			Addr:         addr,
			PoolSize:     1,
			DialTimeout:  50 * time.Millisecond,
			ReadTimeout:  50 * time.Millisecond,
			WriteTimeout: 50 * time.Millisecond,
		},
		time.Minute,
		discardLogger(),
	)
	defer cache.Close()

	ctx := context.Background()
	if _, _, ok := cache.GetTasks(ctx, 7, "filter"); ok {
		t.Fatal("GetTasks() hit with unavailable Redis, want miss")
	}

	cache.SetTasks(ctx, 7, "key", &taskapp.TaskList{})
	cache.InvalidateTeam(ctx, 7)
}

func TestZeroCacheFailsOpen(t *testing.T) {
	var cache Cache
	ctx := context.Background()

	if _, _, ok := cache.GetTasks(ctx, 7, "filter"); ok {
		t.Fatal("GetTasks() hit on zero cache, want miss")
	}
	cache.SetTasks(ctx, 7, "key", &taskapp.TaskList{})
	cache.InvalidateTeam(ctx, 7)
	if err := cache.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func missKey(t *testing.T, cache *Cache, teamID int64, filterKey string) taskapp.CacheKey {
	t.Helper()

	_, key, ok := cache.GetTasks(context.Background(), teamID, filterKey)
	if ok {
		t.Fatal("GetTasks() hit, want miss")
	}
	if key == "" {
		t.Fatal("GetTasks() returned empty key on miss")
	}
	return key
}

func newTestCache(t *testing.T, mr *miniredis.Miniredis, ttl time.Duration) *Cache {
	t.Helper()

	cache := NewCache(
		config.Redis{
			Addr:         mr.Addr(),
			PoolSize:     2,
			DialTimeout:  time.Second,
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
		},
		ttl,
		discardLogger(),
	)
	t.Cleanup(func() {
		_ = cache.Close()
	})
	return cache
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
