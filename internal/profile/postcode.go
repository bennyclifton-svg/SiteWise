package profile

import (
	"regexp"
	"strconv"
	"strings"
)

// Australian postcode ranges by state, from Australia Post's published
// allocation (draft, verified: false). Delivery-area ranges only matter
// here; PO-box ranges are included so a postal address still resolves.
var postcodeStates = []struct {
	lo, hi int
	state  string
}{
	{200, 299, "ACT"}, {800, 999, "NT"},
	{1000, 2599, "NSW"}, {2600, 2618, "ACT"}, {2619, 2899, "NSW"}, {2900, 2920, "ACT"}, {2921, 2999, "NSW"},
	{3000, 3999, "VIC"}, {8000, 8999, "VIC"},
	{4000, 4999, "QLD"}, {9000, 9999, "QLD"},
	{5000, 5999, "SA"}, {6000, 6999, "WA"}, {7000, 7999, "TAS"},
}

var addressStateRe = regexp.MustCompile(`\b(NSW|VIC|QLD|SA|WA|TAS|ACT|NT)\b\.?,?\s*(\d{4})?\s*$`)
var trailingPostcodeRe = regexp.MustCompile(`\b(\d{4})\s*$`)

// stateOfAddress is the state an Australian address names, else the state
// its trailing postcode is allocated to, and the postcode when there is one.
func stateOfAddress(address string) (state, postcode string) {
	a := strings.TrimSpace(strings.TrimRight(address, ". "))
	if m := addressStateRe.FindStringSubmatch(a); m != nil {
		return m[1], m[2]
	}
	if m := trailingPostcodeRe.FindStringSubmatch(a); m != nil {
		n, _ := strconv.Atoi(m[1])
		for _, r := range postcodeStates {
			if n >= r.lo && n <= r.hi {
				return r.state, m[1]
			}
		}
	}
	return "", ""
}

// withAddressState suggests the state from the site address when no
// document states it, and flags a stated state the address contradicts. A
// consultant's office address in a letterhead can carry a different state;
// the site address is the project's.
func withAddressState(rows []Row, whole string) []Row {
	address := ""
	for _, r := range rows {
		if r.Key == "site.address" && strings.TrimSpace(r.Value) != "" && r.Band != bandBlank {
			address = r.Value
		}
	}
	state, postcode := stateOfAddress(address)
	if state == "" {
		return rows
	}
	where := "the site address"
	if postcode != "" {
		where += " (postcode " + postcode + ")"
	}
	for i := range rows {
		r := &rows[i]
		if r.Key != "det.state" || r.PartID != whole {
			continue
		}
		switch {
		case strings.TrimSpace(r.Value) == "":
			r.Value, r.Band, r.Note = state, bandSuggest, "From "+where+"."
		case !strings.EqualFold(r.Value, state) && r.Band != bandUser:
			r.Note = strings.TrimSpace(r.Note + " " + "The site address is in " + state + ", from " + where + "; check this.")
		}
		return rows
	}
	return append(rows, Row{PartID: whole, Key: "det.state", Value: state, Band: bandSuggest, Note: "From " + where + "."})
}
