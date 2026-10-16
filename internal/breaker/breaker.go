// Package breaker: per-route circuit breaker over a rolling 10s window (Closed/Open/Half-Open).
package breaker

import ("sync";"time")

type State int
const (Closed State = iota; Open; HalfOpen)

type Breaker struct {
	mu sync.Mutex
	st State
	openedAt time.Time
	ok, bad []time.Time
	Window, Cooldown time.Duration
	MinReqs int
	ErrRate float64
}

func New() *Breaker { return &Breaker{Window: 10 * time.Second, Cooldown: 5 * time.Second, MinReqs: 20, ErrRate: 0.5} }

func (b *Breaker) trim(now time.Time) {
	cut := now.Add(-b.Window)
	f := func(s []time.Time) []time.Time { i := 0; for i < len(s) && s[i].Before(cut) { i++ }; return s[i:] }
	b.ok, b.bad = f(b.ok), f(b.bad)
}

// Allow reports whether a request may pass.
func (b *Breaker) Allow() bool {
	b.mu.Lock(); defer b.mu.Unlock()
	if b.st == Open {
		if time.Since(b.openedAt) < b.Cooldown { return false }
		b.st = HalfOpen
	}
	return true
}

// Record feeds a result (slow responses should be reported as failures).
func (b *Breaker) Record(success bool) {
	b.mu.Lock(); defer b.mu.Unlock()
	now := time.Now(); b.trim(now)
	if success { b.ok = append(b.ok, now) } else { b.bad = append(b.bad, now) }
	if b.st == HalfOpen { if success { b.st = Closed; b.ok, b.bad = nil, nil } else { b.st, b.openedAt = Open, now }; return }
	n := len(b.ok) + len(b.bad)
	if n >= b.MinReqs && float64(len(b.bad))/float64(n) >= b.ErrRate { b.st, b.openedAt = Open, now }
}
