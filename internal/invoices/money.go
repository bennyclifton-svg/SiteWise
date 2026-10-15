// Package invoices reads invoices into the cost plan's ledger. Code harvests
// candidates, settles what rules can, computes every amount and checks the
// ledger; Jev only picks among candidates in one fan-out per invoice.
//
//	https://docs.typesafe.ai/patterns/fan-out
//	https://docs.typesafe.ai/concepts/how-to-build-with-system-one
package invoices

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// ErrInvalid is input a caller must fix.
var ErrInvalid = errors.New("invalid invoice input")

// maxCents bounds an amount at a trillion dollars, far above any project.
const maxCents = int64(100_000_000_000_000)

var plainAmount = regexp.MustCompile(`^-?\d{1,15}(?:\.\d{1,2})?$`)

// ParseCents reads a typed amount: digits with optional cents, a leading
// minus, and optional thousands separators, spaces or a dollar sign. Blank
// is unknown (ok false, no error).
func ParseCents(s string) (cents int64, ok bool, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false, nil
	}
	neg := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		neg, s = true, s[1:len(s)-1]
	}
	s = strings.NewReplacer(",", "", " ", "", "$", "", "A$", "", "AUD", "").Replace(s)
	if strings.HasPrefix(s, "-") {
		neg, s = !neg, s[1:]
	}
	if !plainAmount.MatchString(s) {
		return 0, false, ErrInvalid
	}
	whole, frac, _ := strings.Cut(s, ".")
	for len(frac) < 2 {
		frac += "0"
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, false, ErrInvalid
	}
	f, _ := strconv.ParseInt(frac, 10, 64)
	cents = w*100 + f
	if cents > maxCents {
		return 0, false, ErrInvalid
	}
	if neg {
		cents = -cents
	}
	return cents, true, nil
}

// FormatCents is the exact decimal string, e.g. 1100000 → "11000.00".
func FormatCents(c int64) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	return sign + strconv.FormatInt(c/100, 10) + "." + pad2(c%100)
}

// DisplayCents groups thousands for a reader, e.g. "11,000.00".
func DisplayCents(c int64) string {
	s := FormatCents(c)
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign, s = "−", s[1:]
	}
	whole, frac, _ := strings.Cut(s, ".")
	var b strings.Builder
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return sign + b.String() + "." + frac
}

func pad2(n int64) string {
	if n < 10 {
		return "0" + strconv.FormatInt(n, 10)
	}
	return strconv.FormatInt(n, 10)
}

// GSTOf is 10 % of an ex-GST amount, rounded half away from zero to the cent.
func GSTOf(ex int64) int64 {
	if ex < 0 {
		return -GSTOf(-ex)
	}
	return (ex + 5) / 10
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
