package main

import ("bytes";"context";"crypto/rand";"encoding/hex";"io";"log";"net/http";"os";"os/signal";"strconv";"strings";"syscall";"time"
	"github.com/Nikhil-creat/nexusgate/internal/auth"
	"github.com/Nikhil-creat/nexusgate/internal/breaker"
	"github.com/Nikhil-creat/nexusgate/internal/cnn_filter"
	"github.com/Nikhil-creat/nexusgate/internal/mcp"
	"github.com/Nikhil-creat/nexusgate/internal/proxy"
	"github.com/Nikhil-creat/nexusgate/internal/ratelimit"
	"github.com/Nikhil-creat/nexusgate/internal/telemetry"
)

func env(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
func envF(k string, d float64) float64 { if f, err := strconv.ParseFloat(os.Getenv(k), 64); err == nil { return f }; return d }

type rec struct{ http.ResponseWriter; code int }
func (r *rec) WriteHeader(c int) { r.code = c; r.ResponseWriter.WriteHeader(c) }

func main() {
	pool := proxy.NewPool(strings.Split(env("UPSTREAMS", "http://mock-backend:9000"), ","), nil)
	pool.StartHealth(5 * time.Second)
	rate, burst := envF("RATE", 1000), envF("BURST", 2000)
	br, rl := breaker.New(), ratelimit.New(nil, rate, burst, env("FAIL_MODE", "open") == "open")
	scan, guard := cnn_filter.Heuristic{}, mcp.NewGuard(int(envF("TOKEN_BUDGET", 200000)))
	origin := os.Getenv("ALLOW_ORIGIN")
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", telemetry.Handler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/", func(w0 http.ResponseWriter, r *http.Request) {
		w := &rec{w0, 200}; t := time.Now(); telemetry.Requests.Add(1)
		defer func() { telemetry.DurMicros.Add(time.Since(t).Microseconds()); if w.code >= 500 { telemetry.Errors.Add(1) } }()
		id := make([]byte, 8); rand.Read(id); rid := hex.EncodeToString(id)
		h := w.Header()
		h.Set("X-Request-ID", rid); h.Set("X-Content-Type-Options", "nosniff"); h.Set("Strict-Transport-Security", "max-age=31536000")
		if origin != "" { h.Set("Access-Control-Allow-Origin", origin); h.Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-API-Key,X-Agent-ID,X-Max-Loops") }
		if r.Method == http.MethodOptions { w.WriteHeader(204); return }
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		key := r.Header.Get("X-API-Key"); if key == "" { key = r.RemoteAddr }
		ok, rem := rl.Allow(r.Context(), key)
		h.Set("X-RateLimit-Limit", strconv.Itoa(int(rate))); h.Set("X-RateLimit-Remaining", strconv.Itoa(rem)); h.Set("X-RateLimit-Reset", "1")
		if !ok { http.Error(w, "rate limited", 429); return }
		if c, m := guard.Check(r, 500); c != 0 { http.Error(w, m, c); return }
		body, err := io.ReadAll(r.Body); if err != nil { http.Error(w, "payload too large", 413); return }
		if s, _ := scan.Scan(body); s >= 0.9 { telemetry.Blocked.Add(1); http.Error(w, "blocked by CNN filter", 403); return }
		r.Body = io.NopCloser(bytes.NewReader(body))
		if !br.Allow() { http.Error(w, "circuit open", 503); return }
		up := pool.Pick(); if up == nil { http.Error(w, "no upstream", 503); return }
		up.Serve(w, r); br.Record(w.code < 500 && time.Since(t) < 2*time.Second)
	})
	var handler http.Handler = mux
	if u := os.Getenv("JWKS_URL"); u != "" { handler = (&auth.Verifier{URL: u}).Middleware(mux) }
	srv := &http.Server{Addr: ":8080", Handler: handler, ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM); defer stop()
	go func() { log.Println("NexusGate on :8080"); if err := srv.ListenAndServe(); err != http.ErrServerClosed { log.Fatal(err) } }()
	<-ctx.Done()
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second); defer cancel()
	log.Println("draining"); srv.Shutdown(c)
}
