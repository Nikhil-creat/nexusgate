// Package telemetry: lock-free counters exposed in Prometheus text format. Add OTel exporters in main.
package telemetry

import ("fmt";"net/http";"sync/atomic")

var Requests, Errors, CacheHits, Blocked, DurMicros atomic.Int64

func Handler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(w, "gateway_requests_total %d\ngateway_errors_total %d\ngateway_cache_hits_total %d\ngateway_cnn_blocked_total %d\ngateway_duration_micros_total %d\n",
		Requests.Load(), Errors.Load(), CacheHits.Load(), Blocked.Load(), DurMicros.Load())
}
