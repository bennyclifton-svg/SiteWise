package invoices

import (
	"regexp"
	"strings"
)

// abnGlued separates "ABN74 099 962 231" so the digits stand alone.
var abnGlued = regexp.MustCompile(`(?i)ABN[:#]?(\d)`)

var abnPattern = regexp.MustCompile(`\b\d{2}[ ]?\d{3}[ ]?\d{3}[ ]?\d{3}\b`)

// abnWeights are the Australian Business Register's published weights.
// https://abr.business.gov.au/Help/AbnFormat
var abnWeights = [11]int{10, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19}

// NormalizeABN keeps the eleven digits, or returns "" when there are not
// exactly eleven.
func NormalizeABN(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-':
		default:
			return ""
		}
	}
	if b.Len() != 11 {
		return ""
	}
	return b.String()
}

// ValidABN applies the checksum: subtract one from the first digit, weight
// each digit, and the sum must divide by 89.
func ValidABN(digits string) bool {
	if len(digits) != 11 {
		return false
	}
	sum := 0
	for i, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
		d := int(r - '0')
		if i == 0 {
			if d == 0 {
				return false
			}
			d--
		}
		sum += d * abnWeights[i]
	}
	return sum%89 == 0
}

// DisplayABN groups the digits as printed: "54 165 354 218".
func DisplayABN(digits string) string {
	if len(digits) != 11 {
		return digits
	}
	return digits[:2] + " " + digits[2:5] + " " + digits[5:8] + " " + digits[8:]
}
