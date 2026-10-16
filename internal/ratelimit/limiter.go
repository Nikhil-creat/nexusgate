// Package ratelimit: token bucket. Redis (Lua) is primary; local bucket is the fail-open fallback.
package ratelimit

import ("context";"sync";"time")

type Remote interface { Take(ctx context.Context, key string, rate, burst, cost float64) (ok bool, remaining int, err error) }

type Limiter struct {
	Remote Remote; FailOpen bool; Rate, Burst float64
	mu sync.Mutex; local map[string]*bucket
}
type bucket struct{ tokens float64; last time.Time }

func New(r Remote, rate, burst float64, failOpen bool) *Limiter {
	return &Limiter{Remote: r, Rate: rate, Burst: burst, FailOpen: failOpen, local: map[string]*bucket{}}
}

// Allow keys by tenant|ip|apikey. Returns allowed + remaining for X-RateLimit-* headers.
func (l *Limiter) Allow(ctx context.Context, key string) (bool, int) {
	if l.Remote != nil {
		c, cancel := context.WithTimeout(ctx, 3*time.Millisecond); defer cancel()
		if ok, rem, err := l.Remote.Take(c, key, l.Rate, l.Burst, 1); err == nil { return ok, rem }
		if !l.FailOpen { return false, 0 }
	}
	l.mu.Lock(); defer l.mu.Unlock()
	b := l.local[key]; now := time.Now()
	if b == nil { b = &bucket{l.Burst, now}; l.local[key] = b }
	b.tokens = min(l.Burst, b.tokens+now.Sub(b.last).Seconds()*l.Rate); b.last = now
	if b.tokens < 1 { return false, 0 }
	b.tokens--; return true, int(b.tokens)
}
