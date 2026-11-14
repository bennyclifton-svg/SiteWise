package tender

import (
	"strconv"
	"strings"
)

// A dollar: the tolerance when two sums are compared.
const tol = 100

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// scanAmount reads one amount starting at s[i], which must be a digit. It
// accepts thousands commas ("1,279,781", "11, 950"), and, when loose, a
// space before a group of three digits ("45 500.00"). Decimals follow a dot
// or a dash ("9,580-00"). It returns the cents and the index after it.
func scanAmount(s string, i int, loose bool) (cents int64, end int, ok bool) {
	var digits []byte
	j := i
scan:
	for j < len(s) {
		c := s[j]
		switch {
		case isDigit(c):
			digits = append(digits, c)
			j++
		case c == ',' && j+1 < len(s) && isDigit(s[j+1]):
			j++
		case c == ',' && j+2 < len(s) && s[j+1] == ' ' && isDigit(s[j+2]):
			j += 2
		case loose && c == ' ' && threeDigitsThenStop(s, j+1):
			j++
		default:
			break scan
		}
	}
	if len(digits) == 0 || len(digits) > 13 {
		return 0, j, false
	}
	n, err := strconv.ParseInt(string(digits), 10, 64)
	if err != nil {
		return 0, j, false
	}
	cents = n * 100
	if j < len(s) && (s[j] == '.' || s[j] == '-') {
		k := j + 1
		for k < len(s) && isDigit(s[k]) {
			k++
		}
		if d := k - (j + 1); d == 1 || d == 2 {
			frac, _ := strconv.ParseInt(s[j+1:k], 10, 64)
			if d == 1 {
				frac *= 10
			}
			cents += frac
			j = k
		}
	}
	return cents, j, true
}

func threeDigitsThenStop(s string, i int) bool {
	if i+3 > len(s) {
		return false
	}
	for k := i; k < i+3; k++ {
		if !isDigit(s[k]) {
			return false
		}
	}
	return i+3 == len(s) || !isDigit(s[i+3])
}

// parsePrinted reads an amount as the model copied it. rate is true when
// words after the figure make it a rate ("per tonne", "/hr", "each").
func parsePrinted(s string) (cents int64, ok, rate bool) {
	s = strings.TrimSpace(s)
	i := strings.IndexFunc(s, func(r rune) bool { return r >= '0' && r <= '9' })
	if i < 0 {
		return 0, false, false
	}
	c, end, ok := scanAmount(s, i, true)
	if !ok {
		return 0, false, false
	}
	if strings.ContainsAny(s[:i], "-(") {
		c = -c
	}
	rest := strings.ToLower(strings.TrimLeft(s[end:], " )"))
	for _, w := range []string{"/", "per ", "p/", "each", "ea.", "ea ", "an hour", "hourly"} {
		if strings.HasPrefix(rest, w) {
			rate = true
		}
	}
	return c, true, rate
}

// pageFigures is every amount that appears in a page's text, in cents,
// without sign. It is what R3 checks a returned amount against.
func pageFigures(text string) map[int64]bool {
	out := map[int64]bool{}
	for i := 0; i < len(text); i++ {
		if !isDigit(text[i]) || (i > 0 && isDigit(text[i-1])) {
			continue
		}
		if c, _, ok := scanAmount(text, i, false); ok {
			out[c] = true
		}
		if c, _, ok := scanAmount(text, i, true); ok {
			out[c] = true
		}
	}
	return out
}

// exGST takes one eleventh off an amount that includes GST, to the cent.
func exGST(inc int64) int64 {
	if inc < 0 {
		return -exGST(-inc)
	}
	return (inc*10 + 5) / 11
}

// dollars writes cents for a flag, e.g. "$83,697.00" or "-$120.50".
func dollars(c int64) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	whole := strconv.FormatInt(c/100, 10)
	var b strings.Builder
	for i, d := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(d)
	}
	frac := strconv.FormatInt(c%100, 10)
	if len(frac) < 2 {
		frac = "0" + frac
	}
	return sign + "$" + b.String() + "." + frac
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func near(a, b int64) bool { return abs64(a-b) <= tol }
