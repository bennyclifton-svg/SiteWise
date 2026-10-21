package proposals

import (
	"fmt"
	"regexp"
	"strings"

	"sitewise/internal/invoices"
)

// BuilderReview is a read-only view of literal quote evidence. Detail amounts
// are deliberately not additive: a breakdown can contain both parent and child
// figures. Accepting a comparison price remains a user's cell correction.
type BuilderReview struct {
	ExGST    *string          `json:"ex_gst"`
	IncGST   *string          `json:"inc_gst"`
	Basis    string           `json:"basis"`
	Sources  []string         `json:"sources"`
	Issues   []string         `json:"issues"`
	Sections []BuilderSection `json:"sections"`
}

type BuilderSection struct {
	Key     string          `json:"key"`
	Label   string          `json:"label"`
	Details []BuilderDetail `json:"details"`
	Summary string          `json:"summary"`
}

type BuilderDetail struct {
	Text         string `json:"text"`
	Relationship string `json:"relationship"`
	// Stored passages may span pages. This is the section's start, not a
	// fabricated precise page reference for each line.
	SectionPage int `json:"section_page"`
}

var builderSections = []struct {
	key, label string
	words      *regexp.Regexp
}{
	{"site", "Site, demolition and groundworks", regexp.MustCompile(`(?i)demoli|\bdemo\b|earthwork|excavat|footing|concrete|\bslab\b|asbestos|rubbish|site setout`)},
	{"structure", "Structure and roof", regexp.MustCompile(`(?i)fram|\broof|truss|structur|timber|brick|carpentr|\beaves|gutter`)},
	{"envelope", "Windows, doors and cladding", regexp.MustCompile(`(?i)window|\bdoor|cladding|glazing|axon|wall wrap`)},
	{"services", "Electrical, plumbing and air conditioning", regexp.MustCompile(`(?i)electric|plumb|air.?con|ducted|daikin|switchboard|meter box|\bgpo\b|power point|downlight|sewer|stormwater|\bgas\b|\bhwh\b`)},
	{"interiors", "Interiors, joinery and finishes", regexp.MustCompile(`(?i)joinery|kitchen|laundry|bath|vanity|tiles?|tiling|waterproof|paint|plaster|gyprock|insulat|carpet|floor cover|floor,|skirting|wardrobe|bench.?top|appliance|\bpc\b|p/c`)},
	{"external", "External works", regexp.MustCompile(`(?i)shed|storage|carport|landscap|driveway|fenc|garden|outdoor`)},
	{"terms", "Conditions and exclusions", regexp.MustCompile(`(?i)exclud|exclusion|not includ|no allowance|supplied by owner|week|valid|subject to|insurance|guarantee|unforeseen|uforseen|delay`)},
	{"other", "Other quote detail", nil},
}

var builderSubtotal = regexp.MustCompile(`(?i)^\s*(?:sub\s*-?\s*total|exc(?:l)?\.?\s*gst)\s*:?\s*\$`)
var builderTotal = regexp.MustCompile(`(?i)^\s*(?:(?:grand\s+)?total(?:\s+price)?(?:\s+(?:incl?\.?|including|excl?\.?|excluding)\s+gst)?|incl?\.?\s*gst)\s*:?\s*\$`)
var builderMargin = regexp.MustCompile(`(?i)^\s*plus\s+(\d{1,2})%\s+builder.?\s*margin\s*:?\s*\$`)
var builderUnitPrice = regexp.MustCompile(`(?i)\bper\b|/\s*(?:m|hr|hour|item)|\bm[²2]\b|\bx\s*\d|\beach\b`)

func decimalMoney(v int64) string { return fmt.Sprintf("%d.%02d", v/100, v%100) }

// ReviewBuilderQuote recognises only explicit total labels and checked margin
// arithmetic. Conflicting totals or missing tax wording stay unresolved. It
// performs no model calls and never treats the largest amount as the total.
func ReviewBuilderQuote(passages []Passage) BuilderReview {
	r := BuilderReview{Sources: []string{}, Issues: []string{}, Sections: []BuilderSection{}}
	for _, s := range builderSections {
		r.Sections = append(r.Sections, BuilderSection{Key: s.key, Label: s.label, Details: []BuilderDetail{}})
	}
	var ex, inc, sub, tax *int64
	var summarySubtotal *int64
	conflict, marginFound := false, false
	set := func(dst **int64, n int64) {
		if *dst != nil && **dst != n {
			conflict = true
		}
		v := n
		*dst = &v
	}
	seen := map[string]bool{}
	summary := false
	for _, p := range passages {
		section := len(builderSections) - 1
		allowanceTable := false
		for _, line := range strings.Split(p.Body, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || seen[line] {
				continue
			}
			seen[line] = true
			lower := strings.ToLower(line)
			label := strings.NewReplacer("(", "", ")", "").Replace(line)
			if strings.Contains(lower, "quote summary") || strings.Contains(lower, "quotation summary") {
				summary = true
			}
			if strings.Contains(lower, "terms & conditions") {
				summary = false
			}
			matches := moneyRe.FindAllString(line, -1)
			if len(matches) == 0 {
				allowanceTable = strings.HasPrefix(lower, "allowances")
			}
			if strings.Contains(lower, "option") || builderTotal.MatchString(label) || builderSubtotal.MatchString(line) {
				allowanceTable = false
			}
			rel := "Scope wording; inclusion not confirmed"
			if len(matches) > 0 {
				rel = "Quoted amount; inclusion unresolved"
			}
			if allowanceTable || strings.Contains(lower, "allowance") || strings.Contains(lower, "pro sum") || strings.Contains(lower, "provisional") {
				rel = "Allowance; inclusion unresolved"
			}
			if strings.Contains(lower, "option") {
				rel = "Option; inclusion unresolved"
			}
			if strings.Contains(lower, "stage") && len(matches) > 0 {
				rel = "Stage / component amounts; do not add together"
			}
			if strings.Contains(lower, "not included") || strings.Contains(lower, "no allowance") || strings.Contains(lower, "exclusion") {
				rel = "Exclusion wording"
			}
			if strings.Contains(lower, "supplied by owner") {
				rel = "Owner supply wording"
			}
			if len(matches) == 1 {
				n, ok, err := invoices.ParseCents(matches[0])
				if ok && err == nil && n >= 0 {
					switch {
					case summary && strings.HasPrefix(lower, "gst:"):
						set(&tax, n)
					case builderSubtotal.MatchString(line):
						set(&sub, n)
						rel = "Subtotal; do not add to total"
						if summary {
							set(&summarySubtotal, n)
							r.Sources = append(r.Sources, line)
						}
						if exGSTRe.MatchString(line) || strings.HasPrefix(lower, "exc gst") {
							set(&ex, n)
							r.Sources = append(r.Sources, line)
						}
					case builderMargin.MatchString(line):
						var percent int64
						fmt.Sscan(builderMargin.FindStringSubmatch(line)[1], &percent)
						if sub != nil && ex != nil && *sub <= 1_000_000_000_000 && (*sub*(100+percent)+50)/100 == n {
							ex = &n
							marginFound = true
							r.Sources = append(r.Sources, line)
						} else {
							conflict = true
							r.Issues = append(r.Issues, "Builder margin could not be reconciled with an explicit ex-GST subtotal.")
						}
						rel = "Total after builder margin; not GST"
					case builderTotal.MatchString(label):
						if incGSTRe.MatchString(line) {
							set(&inc, n)
							r.Sources = append(r.Sources, line)
						} else if exGSTRe.MatchString(line) {
							set(&ex, n)
							r.Sources = append(r.Sources, line)
						} else {
							r.Issues = append(r.Issues, "A total has no explicit GST basis; confirm it from the quote.")
						}
						rel = "Quoted total; not an extra"
					}
				}
			}
			// Literal term lookup is navigation only, not a scope judgement.
			for i, s := range builderSections {
				if s.words != nil && s.words.MatchString(line) {
					section = i
					break
				}
			}
			if builderTotal.MatchString(label) || builderSubtotal.MatchString(line) || builderMargin.MatchString(line) {
				section = len(builderSections) - 1
			}
			details := r.Sections[section].Details
			if len(matches) == 0 && len(details) > 0 && details[len(details)-1].Relationship == rel && details[len(details)-1].SectionPage == p.Page && len(details[len(details)-1].Text)+len(line) < 600 {
				details[len(details)-1].Text += "\n" + line
			} else {
				r.Sections[section].Details = append(details, BuilderDetail{line, rel, p.Page})
			}
		}
	}
	// A bare subtotal is not automatically ex GST. A summary must reconcile
	// its printed subtotal, GST and inclusive total before using that basis.
	if summarySubtotal != nil && tax != nil && inc != nil && *inc >= *tax && *summarySubtotal == *inc-*tax {
		set(&ex, *summarySubtotal)
	}
	if conflict {
		ex, inc = nil, nil
		r.Issues = append(r.Issues, "Conflicting printed totals: review the source before setting a comparison price.")
	}
	if ex == nil && inc != nil {
		n := exGST(*inc)
		ex = &n
		r.Basis = "Calculated ex GST from the printed inclusive total (÷ 1.1)."
	} else if ex != nil {
		r.Basis = "Printed ex-GST subtotal / total."
	}
	if marginFound {
		r.Basis = "Printed ex-GST total including builder margin. Margin is separate from GST."
	}
	if ex != nil {
		s := decimalMoney(*ex)
		r.ExGST = &s
	}
	if inc != nil {
		s := decimalMoney(*inc)
		r.IncGST = &s
	}
	if ex != nil && inc != nil && exGST(*inc) != *ex {
		diff := exGST(*inc) - *ex
		if diff < 0 {
			diff = -diff
		}
		r.Issues = append(r.Issues, "GST / rounding difference $"+decimalMoney(diff)+"; printed subtotal retained.")
	}
	if ex == nil {
		r.Issues = append(r.Issues, "Quoted total unresolved. Review the PDF or enter a confirmed comparison amount.")
	}
	for i := range r.Sections {
		r.Sections[i].Summary = builderSectionSummary(r.Sections[i])
	}
	return r
}

// Section inclusion requires wording naming the whole section. A mention of a
// component, or the absence of a separate price, is not evidence of inclusion.
func builderSectionSummary(s BuilderSection) string {
	status := "Unclear"
	var prices []string
	for _, d := range s.Details {
		for _, line := range strings.Split(d.Text, "\n") {
			lower := strings.ToLower(strings.TrimSpace(line))
			label := strings.ToLower(s.Label)
			for _, state := range []string{"included", "excluded", "not included"} {
				if lower == label+": "+state || lower == label+" — "+state {
					next := "Included"
					if state != "included" {
						next = "Excluded"
					}
					if status != "Unclear" && status != next {
						return "Conflicting scope"
					}
					status = next
				}
			}
			amounts := moneyRe.FindAllString(line, -1)
			if len(amounts) != 1 || builderUnitPrice.MatchString(line) || (!exGSTRe.MatchString(line) && !incGSTRe.MatchString(line)) || strings.Contains(d.Relationship, "total") {
				continue
			}
			n, ok, err := invoices.ParseCents(amounts[0])
			if !ok || err != nil || n < 0 {
				continue
			}
			if incGSTRe.MatchString(line) {
				n = exGST(n)
			}
			kind := "Component"
			if strings.HasPrefix(d.Relationship, "Allowance") {
				kind = "Allowance"
			}
			if strings.HasPrefix(d.Relationship, "Option") {
				kind = "Option"
			}
			prices = append(prices, kind+" $"+decimalMoney(n))
		}
	}
	if len(prices) == 1 {
		return prices[0]
	}
	if len(prices) > 1 {
		return prices[0] + " · +" + fmt.Sprint(len(prices)-1) + " in excerpts"
	}
	return status
}
