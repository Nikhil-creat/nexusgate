// Package auth: RS256 JWT verification against a cached JWKS endpoint with automatic key refresh.
package auth

import ("crypto";"crypto/rsa";"crypto/sha256";"encoding/base64";"encoding/json";"math/big";"net/http";"strings";"sync";"time")

type Verifier struct { URL string; mu sync.RWMutex; keys map[string]*rsa.PublicKey; at time.Time }

func (v *Verifier) refresh() error {
	resp, err := http.Get(v.URL); if err != nil { return err }
	defer resp.Body.Close()
	var set struct{ Keys []struct{ Kid, N, E string `json:"-"` } }
	var raw struct{ Keys []struct{ Kid string `json:"kid"`; N string `json:"n"`; E string `json:"e"` } `json:"keys"` }
	_ = set
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil { return err }
	m := map[string]*rsa.PublicKey{}
	for _, k := range raw.Keys {
		n, _ := base64.RawURLEncoding.DecodeString(k.N); e, _ := base64.RawURLEncoding.DecodeString(k.E)
		m[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	v.mu.Lock(); v.keys, v.at = m, time.Now(); v.mu.Unlock(); return nil
}

func (v *Verifier) key(kid string) *rsa.PublicKey {
	v.mu.RLock(); k, stale := v.keys[kid], time.Since(v.at) > 10*time.Minute; v.mu.RUnlock()
	if k == nil || stale { if v.refresh() == nil { v.mu.RLock(); k = v.keys[kid]; v.mu.RUnlock() } }
	return k
}

func (v *Verifier) Verify(tok string) bool {
	p := strings.Split(tok, "."); if len(p) != 3 { return false }
	hb, _ := base64.RawURLEncoding.DecodeString(p[0]); cb, _ := base64.RawURLEncoding.DecodeString(p[1]); sig, _ := base64.RawURLEncoding.DecodeString(p[2])
	var h struct{ Alg, Kid string `json:"-"` }
	var hdr struct{ Alg string `json:"alg"`; Kid string `json:"kid"` }
	var cl struct{ Exp int64 `json:"exp"` }
	_ = h
	if json.Unmarshal(hb, &hdr) != nil || json.Unmarshal(cb, &cl) != nil || hdr.Alg != "RS256" || cl.Exp < time.Now().Unix() { return false }
	k := v.key(hdr.Kid); if k == nil { return false }
	sum := sha256.Sum256([]byte(p[0] + "." + p[1]))
	return rsa.VerifyPKCS1v15(k, crypto.SHA256, sum[:], sig) == nil
}

// Middleware protects everything except /healthz and /metrics.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/metrics" { next.ServeHTTP(w, r); return }
		if !v.Verify(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")) { http.Error(w, "unauthorized", 401); return }
		next.ServeHTTP(w, r)
	})
}
