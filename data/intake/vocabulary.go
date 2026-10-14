// Package intake exposes the shipped filing vocabulary for catalogue validation.
package intake

import _ "embed"

//go:embed disciplines.json
var Disciplines []byte

// Kinds is the document kind vocabulary filing chooses from.
//
//go:embed kinds.json
var Kinds []byte
