// Package mcp: agent-to-agent guard. Enforces loop and token budgets from trace-propagated headers.
package mcp

import ("net/http";"strconv";"sync")

type Guard struct { mu sync.Mutex; depth map[string]int; spent map[string]int; Budget int }

func NewGuard(budget int) *Guard { return &Guard{depth: map[string]int{}, spent: map[string]int{}, Budget: budget} }

// Check rejects recursion beyond X-Max-Loops or token spend beyond the budget.
func (g *Guard) Check(r *http.Request, tokens int) (int, string) {
	id := r.Header.Get("X-Agent-ID"); if id == "" { return 0, "" }
	max, _ := strconv.Atoi(r.Header.Get("X-Max-Loops")); if max == 0 { max = 8 }
	g.mu.Lock(); defer g.mu.Unlock()
	g.depth[id]++; g.spent[id] += tokens
	if g.depth[id] > max { return http.StatusLoopDetected, "agent loop limit reached" }
	if g.spent[id] > g.Budget { return http.StatusPaymentRequired, "agent token budget exhausted" }
	return 0, ""
}
