// internal/mw/ratelimit.go
package mw

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	Rdb   *redis.Client
	RPS   int
	Burst int
}

func NewRateLimiter(rdb *redis.Client, rps, burst int) *RateLimiter {
	return &RateLimiter{Rdb: rdb, RPS: rps, Burst: burst}
}

// token bucket simples em Redis: INCR + EXPIRE por janela de 1s + burst
func (rl *RateLimiter) Allow(ctx context.Context, driverID string) bool {
	now := time.Now().Unix()
	key := "rl:" + driverID + ":" + strconv.FormatInt(now, 10)
	cnt, err := rl.Rdb.Incr(ctx, key).Result()
	if err != nil {
		return true // falha: seja permissivo
	}
	_ = rl.Rdb.Expire(ctx, key, 2*time.Second).Err()
	return int(cnt) <= rl.RPS+rl.Burst
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		driver := r.Context().Value(CtxDriverKey{}).(string)
		if !rl.Allow(r.Context(), driver) {
			http.Error(w, "rate limit", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type CtxDriverKey struct{}
