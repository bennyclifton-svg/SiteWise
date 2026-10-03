package intake

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// A revision belongs to one series: plain letters, a plain number, or a
// one-letter prefix plus a number (P10, C1). Those series are not one order.
// Rank is the integer written in the token, so P10 is after P2.
type parsedRevision struct {
	display    string
	series     string
	rank       int
	normalized string
}

var revPrefix = regexp.MustCompile(`(?i)^(?:rev(?:ision)?)\s+`)

// Compare reports whether a is earlier (-1), the same (0), or later (1) than b.
// comparable is false when either token is not a revision or the series differ.
func Compare(a, b string) (cmp int, comparable bool) {
	left, lok := parseRevision(a)
	right, rok := parseRevision(b)
	if !lok || !rok || left.series != right.series {
		return 0, false
	}
	switch {
	case left.rank < right.rank:
		return -1, true
	case left.rank > right.rank:
		return 1, true
	default:
		return 0, true
	}
}

// Sequence orders one revision series from earlier to later, keeping each
// caller's literal token. Mixed series return ok false and no sequence.
func Sequence(revisions []string) (ordered []string, ok bool) {
	if len(revisions) == 0 {
		return nil, false
	}
	parsed := make([]parsedRevision, len(revisions))
	for i, token := range revisions {
		item, parsedOK := parseRevision(token)
		if !parsedOK {
			return nil, false
		}
		item.display = token
		parsed[i] = item
	}
	series := parsed[0].series
	for _, item := range parsed[1:] {
		if item.series != series {
			return nil, false
		}
	}
	sort.SliceStable(parsed, func(i, j int) bool {
		return parsed[i].rank < parsed[j].rank
	})
	ordered = make([]string, len(parsed))
	for i, item := range parsed {
		ordered[i] = item.display
	}
	return ordered, true
}

func parseRevision(token string) (parsedRevision, bool) {
	cleaned := strings.TrimSpace(token)
	if hasRevPrefix(cleaned) {
		cleaned = strings.Trim(revPrefix.ReplaceAllString(cleaned, ""), " .")
	} else {
		cleaned = strings.Trim(cleaned, " .")
	}
	if cleaned == "" {
		return parsedRevision{}, false
	}
	upper := cleaned
	for i := 0; i < len(cleaned); i++ {
		if cleaned[i] >= 'a' && cleaned[i] <= 'z' {
			upper = strings.ToUpper(cleaned)
			break
		}
	}
	for i := 0; i < len(upper); i++ {
		c := upper[i]
		if (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return parsedRevision{}, false
		}
	}
	if digitsOnly(upper) {
		if len(upper) > 3 {
			return parsedRevision{}, false
		}
		n, ok := atoi(upper)
		if !ok {
			return parsedRevision{}, false
		}
		norm := upper
		if len(upper) > 1 && upper[0] == '0' {
			norm = strconv.Itoa(n)
		}
		return parsedRevision{series: "numeric", rank: n, normalized: norm}, true
	}
	if lettersOnly(upper) {
		if len(upper) > 2 {
			return parsedRevision{}, false
		}
		return parsedRevision{series: "alpha", rank: alphaRank(upper), normalized: upper}, true
	}
	if len(upper) >= 2 && upper[0] >= 'A' && upper[0] <= 'Z' && digitsOnly(upper[1:]) && len(upper[1:]) <= 3 {
		n, ok := atoi(upper[1:])
		if !ok {
			return parsedRevision{}, false
		}
		norm := upper
		if upper[1] == '0' {
			norm = upper[:1] + strconv.Itoa(n)
		}
		return parsedRevision{series: upper[:1], rank: n, normalized: norm}, true
	}
	return parsedRevision{}, false
}

func hasRevPrefix(s string) bool {
	if len(s) < 3 || (s[0] != 'R' && s[0] != 'r') || (s[1] != 'E' && s[1] != 'e') || (s[2] != 'V' && s[2] != 'v') {
		return false
	}
	return true
}

func atoi(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

var reportRevision = regexp.MustCompile(`(?i)^(?:[A-Z]{1,8}\s*)?\d{1,3}\.\d{1,3}$`)

var stagedReportRevision = regexp.MustCompile(`(?i)^(?:draft|final)\s+(v?\d{1,3}\.\d{1,3})$`)

func revisionNormalized(token string) (string, bool) {
	if stagedReportRevision.MatchString(strings.TrimSpace(token)) {
		// Draft and final issues can share a version number; preserve the stage.
		return strings.ToUpper(strings.Join(strings.Fields(token), " ")), true
	}
	if reportRevision.MatchString(strings.TrimSpace(token)) {
		// These are literal report versions, not an established ordered series.
		// parseRevision deliberately still declines automatic supersession.
		return strings.ToUpper(strings.Join(strings.Fields(token), " ")), true
	}
	parsed, ok := parseRevision(token)
	if !ok {
		return "", false
	}
	return parsed.normalized, true
}

func alphaRank(letters string) int {
	rank := 0
	for _, r := range letters {
		rank = rank*26 + int(r-'A'+1)
	}
	return rank
}

func digitsOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func lettersOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
