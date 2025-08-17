// internal/store/redis_store.go
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"os"
	"strconv"
	"time"
)

type RedisStore struct {
	Rdb                 *redis.Client
	IdemTTL             time.Duration
	LastPositionTTL     time.Duration
	GEOKey              string
	StreamChannelPrefix string
}

func NewRedisStore() (*RedisStore, error) {
	addr := env("REDIS_ADDR", "localhost:6379")
	pwd := env("REDIS_PASSWORD", "")
	db, _ := strconv.Atoi(env("REDIS_DB", "0"))

	rdb := redis.NewClient(&redis.Options{Addr: addr, Password: pwd, DB: db})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	idemTTL := time.Duration(envInt("IDEMPOTENCY_TTL_SEC", 3600)) * time.Second
	lastTTL := time.Duration(envInt("LAST_POSITION_TTL_SEC", 172800)) * time.Second

	return &RedisStore{
		Rdb:                 rdb,
		IdemTTL:             idemTTL,
		LastPositionTTL:     lastTTL,
		GEOKey:              "drivers:last",
		StreamChannelPrefix: "locations",
	}, nil
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

// Idempotency: returns true if this (driver,seq) is NEW and we should process.
func (s *RedisStore) CheckIdempotency(ctx context.Context, driver string, seq int64) (bool, error) {
	key := fmt.Sprintf("idem:%s:%d", driver, seq)
	ok, err := s.Rdb.SetNX(ctx, key, 1, s.IdemTTL).Result()
	return ok, err
}

// Update last position in GEO set and set per-driver TTL meta.
func (s *RedisStore) UpdateLastPosition(ctx context.Context, driver string, lat, lng float64) error {
	if err := s.Rdb.GeoAdd(ctx, s.GEOKey, &redis.GeoLocation{
		Name:      driver,
		Longitude: lng,
		Latitude:  lat,
	}).Err(); err != nil {
		return err
	}
	// opcional: TTL global do conjunto não resolve; mas podemos setar uma chave heartbeat
	return s.Rdb.Set(ctx, "driver:heartbeat:"+driver, time.Now().UTC().Format(time.RFC3339), s.LastPositionTTL).Err()
}

// Publish fan-out: order and driver channels
func (s *RedisStore) PublishPoint(ctx context.Context, orderID, driverID string, payload any) error {
	data, _ := json.Marshal(payload)
	// sempre publica canal do driver
	if err := s.Rdb.Publish(ctx, s.StreamChannelPrefix+":driver:"+driverID, data).Err(); err != nil {
		return err
	}
	// se houver pedido, publica também
	if orderID != "" {
		if err := s.Rdb.Publish(ctx, s.StreamChannelPrefix+":order:"+orderID, data).Err(); err != nil {
			return err
		}
	}
	return nil
}
