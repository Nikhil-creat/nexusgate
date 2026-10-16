// Package cnn_filter: threat scanner. Scanner is the ONNX seam; Heuristic is the dependency-free default.
package cnn_filter

import ("regexp";"time")

type Scanner interface { Scan(payload []byte) (score float64, took time.Duration) }

// For production, add an OnnxScanner using github.com/yalue/onnxruntime_go (byte-histogram -> 1D CNN -> sigmoid), built with CGO.
type Heuristic struct{}

var sigs = regexp.MustCompile(`(?i)(ignore (all )?previous|system prompt|\bunion\s+select\b|<script|/bin/(ba)?sh|\\x[0-9a-f]{2}\\x[0-9a-f]{2}|base64,)`)

func (Heuristic) Scan(p []byte) (float64, time.Duration) {
	t := time.Now(); s := 0.0
	if sigs.Match(p) { s = 0.97 }
	return s, time.Since(t)
}
