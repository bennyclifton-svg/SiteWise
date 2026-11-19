// Package mail holds the mail lists SiteWise ships.
package mail

import _ "embed"

// Domains is domains.json: free-mail and shared sending domains.
//
//go:embed domains.json
var Domains []byte
