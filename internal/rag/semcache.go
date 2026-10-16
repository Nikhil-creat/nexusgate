// Package rag: semantic cache (cosine >= 0.96 short-circuits the LLM). Swap the slice for Redis Search/HNSW in prod.
package rag

import ("math";"sync")

type entry struct{ vec []float32; resp []byte }
type SemCache struct { mu sync.RWMutex; items []entry; Threshold float32 }

func NewSemCache() *SemCache { return &SemCache{Threshold: 0.96} }

func cosine(a, b []float32) float32 {
	var d, x, y float64
	for i := range a { d += float64(a[i] * b[i]); x += float64(a[i] * a[i]); y += float64(b[i] * b[i]) }
	if x == 0 || y == 0 { return 0 }
	return float32(d / (math.Sqrt(x) * math.Sqrt(y)))
}

func (c *SemCache) Get(v []float32) ([]byte, bool) {
	c.mu.RLock(); defer c.mu.RUnlock()
	for _, e := range c.items { if cosine(e.vec, v) >= c.Threshold { return e.resp, true } }
	return nil, false
}
func (c *SemCache) Put(v []float32, r []byte) { c.mu.Lock(); c.items = append(c.items, entry{v, r}); c.mu.Unlock() }
