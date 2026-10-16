// Package proxy: reverse proxy with pooled buffers, weighted least-connections LB and passive health.
package proxy

import ("net/http";"net/http/httputil";"net/url";"sync";"sync/atomic")

type Upstream struct{ URL *url.URL; Weight int64; Active atomic.Int64; Down atomic.Bool; rp *httputil.ReverseProxy }
type Pool struct{ ups []*Upstream }
type bufPool struct{ p sync.Pool }
func (b *bufPool) Get() []byte  { return b.p.Get().([]byte) }
func (b *bufPool) Put(x []byte) { b.p.Put(x) }

func NewPool(urls []string, weights []int64) *Pool {
	bp := &bufPool{sync.Pool{New: func() any { return make([]byte, 32<<10) }}}
	p := &Pool{}
	for i, s := range urls {
		u, _ := url.Parse(s); w := int64(1); if i < len(weights) { w = weights[i] }
		up := &Upstream{URL: u, Weight: w}
		up.rp = httputil.NewSingleHostReverseProxy(u); up.rp.BufferPool = bp
		up.rp.ErrorHandler = func(rw http.ResponseWriter, _ *http.Request, _ error) { up.Down.Store(true); http.Error(rw, "upstream error", 502) }
		p.ups = append(p.ups, up)
	}
	return p
}

// Pick: lowest (active/weight) among healthy upstreams.
func (p *Pool) Pick() *Upstream {
	var best *Upstream
	for _, u := range p.ups {
		if u.Down.Load() { continue }
		if best == nil || u.Active.Load()*best.Weight < best.Active.Load()*u.Weight { best = u }
	}
	if best == nil && len(p.ups) > 0 { best = p.ups[0]; best.Down.Store(false) } // probe back
	return best
}

func (u *Upstream) Serve(w http.ResponseWriter, r *http.Request) { u.Active.Add(1); defer u.Active.Add(-1); u.rp.ServeHTTP(w, r) }
