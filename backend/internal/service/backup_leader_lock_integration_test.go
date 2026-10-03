//go:build integration

package service

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type integrationLeaderLockCache struct {
	client *redis.Client
}

func (c integrationLeaderLockCache) TryAcquireLeaderLock(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	return c.client.SetNX(ctx, "leader:lock:"+key, owner, ttl).Result()
}

func (c integrationLeaderLockCache) ReleaseLeaderLock(ctx context.Context, key, owner string) error {
	value, err := c.client.Get(ctx, "leader:lock:"+key).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if value == owner {
		return c.client.Del(ctx, "leader:lock:"+key).Err()
	}
	return nil
}

func TestBackupLeaderLockIntegration_RedisTwoInstances(t *testing.T) {
	addr := os.Getenv("ANLAPI_C47_REDIS_ADDR")
	if addr == "" {
		t.Skip("ANLAPI_C47_REDIS_ADDR is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()
	require.NoError(t, client.Ping(ctx).Err())
	cache := integrationLeaderLockCache{client: client}
	lockKey := "leader:lock:" + backupRecordsLeaderLockKey
	require.NoError(t, client.Del(ctx, lockKey).Err())
	t.Cleanup(func() { _ = client.Del(context.Background(), lockKey).Err() })

	first := NewBackupService(nil, nil, nil, nil, nil)
	second := NewBackupService(nil, nil, nil, nil, nil)
	first.SetLeaderLock(cache, nil)
	second.SetLeaderLock(cache, nil)

	releaseFirst, acquired, err := first.tryAcquireRecordsLock(ctx)
	require.NoError(t, err)
	require.True(t, acquired)
	defer releaseFirst()

	_, acquired, err = second.tryAcquireRecordsLock(ctx)
	require.NoError(t, err)
	require.False(t, acquired)

	releaseFirst()
	releaseFirst = nil
	releaseSecond, acquired, err := second.tryAcquireRecordsLock(ctx)
	require.NoError(t, err)
	require.True(t, acquired)
	releaseSecond()
}

func TestBackupLeaderLockIntegration_PostgresAdvisoryFallback(t *testing.T) {
	dsn := os.Getenv("ANLAPI_C47_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("ANLAPI_C47_POSTGRES_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.PingContext(ctx))

	first := NewBackupService(nil, nil, nil, nil, nil)
	second := NewBackupService(nil, nil, nil, nil, nil)
	first.SetLeaderLock(nil, db)
	second.SetLeaderLock(nil, db)

	releaseFirst, acquired, err := first.tryAcquireRecordsLock(ctx)
	require.NoError(t, err)
	require.True(t, acquired)
	defer releaseFirst()

	_, acquired, err = second.tryAcquireRecordsLock(ctx)
	require.NoError(t, err)
	require.False(t, acquired)

	releaseFirst()
	releaseFirst = nil
	releaseSecond, acquired, err := second.tryAcquireRecordsLock(ctx)
	require.NoError(t, err)
	require.True(t, acquired)
	releaseSecond()
}
