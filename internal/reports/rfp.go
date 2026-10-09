package reports

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const ReviewSectionID = "pre_issue_review"

var briefKeyLabels = map[string]string{
	"gfa sqm":       "Gross floor area (sq m)",
	"total gfa sqm": "Total gross floor area (sq m)",
	"site sqm":      "Site area (sq m)",
	"car parks":     "Car parks",
}

var briefValueLabels = map[string]string{
	"da":               "Development application",
	"cdc":              "Complying development certificate",
	"design_construct": "Design and construct",
}

// shapeRFP keeps an RFP to what a consultant needs to price services. Open
// review prompts, unknown facts and allocation gaps move to an internal
// section the project manager clears before issue; nothing is dropped.
func shapeRFP(sections []Section, s Snapshot) []Section {
	review := Section{ID: ReviewSectionID, Title: "Before issue — resolve or accept", Internal: true, Blocks: []Block{}}
	note := func(id, label, text string, from Block) {
		from.ID, from.Label, from.Text, from.Provisional = id, label, text, true
		review.Blocks = append(review.Blocks, from)
	}
	type group struct {
		first Block
		count int
	}
	proposals, gaps := map[string]*group{}, map[string]*group{}
	var proposalOrder, gapOrder, unknownBrief []string
	var unknownBriefBlock Block
	packageWorks := map[string]bool{}
	scoped := false
	for _, row := range s.Scope {
		if row.PackageID == s.Package.ID && row.RetiredAt == nil && row.WorkItemID != "" {
			scoped = true
		}
	}
	for _, w := range s.Works {
		packageWorks[w.ID] = !scoped || workInPackage(w, s.Works, s.Scope, s.Package.ID, reportPackages(s))
	}
	var included, excluded []string
	var worksBlock Block
	hasServices := false
	for i := range sections {
		kept := sections[i].Blocks[:0]
		for _, b := range sections[i].Blocks {
			switch {
			case strings.HasPrefix(b.ID, "proposal:"):
				if g, ok := proposals[b.Text]; ok {
					g.count++
				} else {
					proposals[b.Text] = &group{b, 1}
					proposalOrder = append(proposalOrder, b.Text)
				}
				continue
			case strings.HasPrefix(b.ID, "gap:"):
				key := b.Label + "|" + b.Text
				if g, ok := gaps[key]; ok {
					g.count++
				} else {
					gaps[key] = &group{b, 1}
					gapOrder = append(gapOrder, key)
				}
				continue
			case b.ID == "reading_status" || strings.HasPrefix(b.ID, "unknown:") || b.ID == "cost:unavailable":
				note(b.ID, b.Label, b.Text, b)
				continue
			case strings.HasPrefix(b.ID, "brief:") && b.Provisional && b.Text == "Not established":
				unknownBrief = append(unknownBrief, briefLabel(b.Label))
				unknownBriefBlock = b
				continue
			case strings.HasPrefix(b.ID, "brief:"):
				b.Label, b.Text = briefLabel(b.Label), briefValue(b.Text)
			case strings.HasPrefix(b.ID, "work:") && sections[i].ID == "brief":
				id := strings.TrimPrefix(b.ID, "work:")
				isExcluded := strings.Contains(b.Label, "— excluded")
				if !isExcluded && !packageWorks[id] {
					continue
				}
				title, _, _ := strings.Cut(b.Text, ". Action:")
				if loc := workLocation(b.Text); loc != "" && loc != "Whole project" && loc != "Location not established" {
					title += " (" + loc + ")"
				}
				if isExcluded {
					excluded = append(excluded, title)
				} else {
					included = append(included, title)
				}
				worksBlock = b
				continue
			case strings.HasPrefix(b.ID, "scope:") && sections[i].ID == "services":
				hasServices = true
			}
			kept = append(kept, b)
		}
		sections[i].Blocks = kept
	}
	for i := range sections {
		if sections[i].ID == "brief" {
			sections[i].Blocks = dropRepeatedTotals(sections[i].Blocks)
		}
	}
	if len(included)+len(excluded) > 0 {
		calc := func(id, label string, items []string) Block {
			b := worksBlock
			b.ID, b.Label, b.Text, b.Origin = id, label, strings.Join(items, ", ")+".", "calculation"
			return b
		}
		for i := range sections {
			if sections[i].ID != "brief" {
				continue
			}
			if len(included) > 0 {
				sections[i].Blocks = append(sections[i].Blocks, calc("work:summary", "Building elements in scope", included))
			}
			if len(excluded) > 0 {
				sections[i].Blocks = append(sections[i].Blocks, calc("work:excluded_summary", "Excluded from the works", excluded))
			}
		}
	}
	if !hasServices {
		note("review:services", "Services", "No services or deliverables are recorded for this package. Add the package scope before issue so the consultant can price each stage.", Block{Origin: "calculation", ReviewStatus: "proposed", Meaning: "stated", Basis: json.RawMessage(`{"method":"package_scope_check"}`)})
	}
	if len(unknownBrief) > 0 {
		note("review:brief_unknown", "Facts not established", strings.Join(unknownBrief, ", ")+". Confirm these or leave them out of the brief.", unknownBriefBlock)
	}
	for _, key := range proposalOrder {
		g := proposals[key]
		note(g.first.ID, "Open review", withCount(g.first.Text, g.count), g.first)
	}
	for _, key := range gapOrder {
		g := gaps[key]
		note(g.first.ID, strings.ReplaceAll(g.first.Label, "_", " "), withCount("No package accepted for the "+g.first.Text+" role", g.count), g.first)
	}
	sort.SliceStable(review.Blocks, func(i, j int) bool {
		return review.Blocks[i].ID == "review:services" && review.Blocks[j].ID != "review:services"
	})
	return append(sections, review)
}

// The profile can state a fact and its whole-project total; when they agree,
// the total adds a line without adding information.
func dropRepeatedTotals(blocks []Block) []Block {
	values := map[string]string{}
	for _, b := range blocks {
		values[b.Label] = b.Text
	}
	kept := blocks[:0]
	for _, b := range blocks {
		if rest, ok := strings.CutPrefix(b.Label, "Total "); ok && strings.HasPrefix(b.ID, "brief:") {
			if v, ok := values[sentence(rest)]; ok && v == b.Text {
				continue
			}
		}
		kept = append(kept, b)
	}
	return kept
}

func withCount(text string, n int) string {
	if n == 1 {
		return text + " (1 work item)"
	}
	return fmt.Sprintf("%s (%d work items)", text, n)
}

func workLocation(text string) string {
	_, rest, ok := strings.Cut(text, ". Location: ")
	if !ok {
		return ""
	}
	loc, _, _ := strings.Cut(rest, ". ")
	return loc
}

func briefLabel(label string) string {
	label = strings.TrimPrefix(label, "Whole project — ")
	for _, prefix := range []string{"cond.", "scale."} {
		if strings.HasPrefix(label, prefix) {
			key := strings.TrimPrefix(label, prefix)
			if known, ok := briefKeyLabels[key]; ok {
				return known
			}
			return sentence(key)
		}
	}
	return label
}

func briefValue(text string) string {
	if known, ok := briefValueLabels[text]; ok {
		return known
	}
	if !strings.Contains(text, " ") && text == strings.ToLower(text) {
		return sentence(strings.ReplaceAll(text, "_", " "))
	}
	return text
}

func sentence(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
