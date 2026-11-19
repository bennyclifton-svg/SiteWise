package httpapi

import "sort"

// RoutePatterns is every pattern the server serves, so the guest sweep can
// walk the same table (docs/plans/2026-11-17-guest-access.md, Task 7).
func RoutePatterns() []string {
	var out []string
	for pattern := range apiRoutes() {
		out = append(out, pattern)
	}
	sort.Strings(out)
	return out
}

// SettleStaged is the webhook's settling of one request's staged blob, for
// the two-request staging test (review LOW-5).
var SettleStaged = settleStaged
