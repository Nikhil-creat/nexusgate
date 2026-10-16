package main

import ("io";"log";"net/http";"os";"strconv";"strings";"time"
	"github.com/Nikhil-creat/nexusgate/internal/auth"
	"github.com/Nikhil-creat/nexusgate/internal/breaker"
	"github.com/Nikhil-creat/nexusgate/internal/cnn_filter"
	"github.com/Nikhil-creat/nexusgate/internal/mcp"
	"github.com/Nikhil-creat/nexusgate/internal/proxy"
	"github.com/Nikhil-creat/nexusgate/internal/ratelimit"
	"github.com/Nikhil-creat/nexusgate/internal/telemetry"
)

func env(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }

func main() {
	pool := proxy.NewPool(strings.Split(env("UPSTREAMS", "http://mock-backend:9000"), ","), nil)
	br, rl := breaker.New(), ratelimit.New(nil, 1000, 2000, true)
	scan, guard := cnn_filter.Heuristic{}, mcp.NewGuard(200000)
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", telemetry.Handler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t := time.Now(); telemetry.Requests.Add(1)
		defer func() { telemetry.DurMicros.Add(time.Since(t).Microseconds()) }()
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		key := r.Header.Get("X-API-Key"); if key == "" { key = r.RemoteAddr }
		ok, rem := rl.Allow(r.Context(), key)
		w.Header().Set("X-RateLimit-Limit", "1000"); w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(rem))
		if !ok { http.Error(w, "rate limited", 429); return }
		if c, m := guard.Check(r, 500); c != 0 { http.Error(w, m, c); return }
		body, _ := io.ReadAll(r.Body)
		if s, _ := scan.Scan(body); s >= 0.9 { telemetry.Blocked.Add(1); http.Error(w, "blocked by CNN filter", 403); return }
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		if !br.Allow() { http.Error(w, "circuit open", 503); return }
		up := pool.Pick(); if up == nil { http.Error(w, "no upstream", 503); return }
		up.Serve(w, r); br.Record(true)
	})
	log.Println("NexusGate on :8080")
	log.Fatal((&http.Server{Addr: ":8080", Handler: wrap(mux), ReadHeaderTimeout: 2 * time.Second}).ListenAndServe())
}

// wrap enables JWT/JWKS auth when JWKS_URL is set.
func wrap(h http.Handler) http.Handler {
	if u := os.Getenv("JWKS_URL"); u != "" { return (&auth.Verifier{URL: u}).Middleware(h) }
	return h
}
