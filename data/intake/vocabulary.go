// Package intake exposes the shipped filing vocabulary for catalogue validation.
package intake

import _ "embed"

//go:embed disciplines.json
var Disciplines []byte
