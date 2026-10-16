# NexusGate: AI & API Gateway (Go 1.22+)
Built for Nikhil Chary Sriramoju.

## Layout
- `cmd/gateway`, `internal/*`: the Go gateway (proxy, mcp, rag, cnn_filter, ratelimit, breaker, telemetry)
- `scripts/redis/token_bucket.lua`: atomic token bucket
- `Dockerfile`, `docker-compose.yml`, `prometheus.yml`, `loadtest/load_test.js` (k6)
- `docs/`: the GitHub Pages showcase with live demos and the Groq chatbot

## Run the gateway
`docker compose up --build`, then `curl localhost:8080/healthz` and `localhost:8080/metrics`.
Load test: `k6 run loadtest/load_test.js`

## Publish on GitHub Pages
1. Create a public repo named `nexusgate` at github.com/new.
2. Extract this zip, open a terminal inside the folder, then run:
   `git init && git add . && git commit -m "NexusGate" && git branch -M main`
   `git remote add origin https://github.com/Nikhil-creat/nexusgate.git && git push -u origin main`
3. Repo Settings > Pages > Source: "GitHub Actions" (the included workflow deploys `docs/`). Alternative: "Deploy from a branch", `main`, `/docs`.
4. After about a minute: https://nikhil-creat.github.io/nexusgate/
5. Open the site and paste your free Groq key (console.groq.com/keys) into the chatbot.

## Honest status
Implemented (new: RS256 JWT/JWKS auth via JWKS_URL): LB with pooled buffers, breaker, token bucket (local plus Lua script), in-memory semantic cache, heuristic threat filter, A2A guard, Prometheus counters.
Still to wire for production: Redis client for `ratelimit.Remote`, Redis Search/HNSW in `rag`, ONNX model in `cnn_filter` (CGO), mTLS, OTel exporters. The p99 targets need your own benchmarking.
