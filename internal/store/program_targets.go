package store

import (
	"fmt"
	"strings"
	"time"

	"sitewise/internal/profile"
	engine "sitewise/internal/program"
)

// ProgramTarget is an owner target or a builder's estimate beside what the
// program computes (Walsh review, 2026-10-27). It never moves a date: the
// program stays computed in Go from its activities.
type ProgramTarget struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Stated is the target as the document or the user wrote it.
	Stated string `json:"stated"`
	// Programmed is the program's date or duration in words, or "".
	Programmed string `json:"programmed"`
	// Status is met, later, longer, shorter (than a builder's estimate),
	// within or unknown.
	Status  string `json:"status"`
	Message string `json:"message"`
}

// programTargetKeys are the profile details the Program compares.
var programTargetKeys = []string{"proj.target_da_lodgement", "proj.target_construction_start", "proj.max_construction_months", "proj.preliminary_construction_months"}

// ownerTargets compares each stated target with the computed program.
func ownerTargets(st ProgramState, stated map[string]string) []ProgramTarget {
	times := map[string]engine.Times{}
	for i, t := range st.Computed.Activities {
		id := t.ID
		if id == "" && i < len(st.Activities) {
			id = st.Activities[i].ID
		}
		times[id] = t
	}
	// Construction starts with the Delivery stage. Delivery also holds the
	// defects period, so time on site ends at practical completion when the
	// program has it.
	var lodge, construction, completion *engine.Times
	for _, a := range st.Activities {
		key := string(a.SourceKey)
		t, ok := times[a.ID]
		switch {
		case !ok:
		case strings.HasPrefix(key, "fragment:fr.da-lodgement:") && strings.HasSuffix(key, ":lodge") && lodge == nil:
			lodge = &t
		case a.Kind == "stage" && string(a.CatalogueStageID) == "delivery" && construction == nil:
			construction = &t
		case strings.HasPrefix(key, "fragment:fr.practical-completion:") && completion == nil:
			completion = &t
		}
	}
	onSite := construction
	if construction != nil && completion != nil && !completion.Unknown && completion.Finish != "" {
		span := *construction
		span.Finish = completion.Finish
		onSite = &span
	}
	out := []ProgramTarget{}
	if v := strings.TrimSpace(stated["proj.target_da_lodgement"]); v != "" {
		out = append(out, dateTarget("proj.target_da_lodgement", "DA lodgement", v, lodge, func(t engine.Times) string { return t.Finish }))
	}
	if v := strings.TrimSpace(stated["proj.target_construction_start"]); v != "" {
		out = append(out, dateTarget("proj.target_construction_start", "Construction start", v, construction, func(t engine.Times) string { return t.Start }))
	}
	months, known := constructionMonths(onSite)
	for _, k := range []struct{ key, label string }{{"proj.max_construction_months", "Maximum time on site"}, {"proj.preliminary_construction_months", "Builder's estimated time on site"}} {
		v := strings.TrimSpace(stated[k.key])
		if v == "" {
			continue
		}
		t := ProgramTarget{Key: k.key, Label: k.label, Stated: v, Status: "unknown"}
		low, high, ok := profile.MonthRange(v)
		switch {
		case !ok:
			t.Message = fmt.Sprintf("%s: %s. Not a month count the program can compare.", k.label, v)
		case !known:
			t.Message = fmt.Sprintf("%s: %s. The program's construction stage has no computed dates yet.", k.label, v)
		default:
			t.Programmed = fmt.Sprintf("about %.1f months", months)
			switch {
			case months > float64(high):
				t.Status = "longer"
			case k.key == "proj.preliminary_construction_months" && months < float64(low):
				// A program much shorter than the builder's own estimate
				// usually means durations are missing or assumed.
				t.Status = "shorter"
			default:
				t.Status = "within"
			}
			t.Message = fmt.Sprintf("%s: %s. The program's construction stage runs %s.", k.label, profile.MonthsText(low, high), t.Programmed)
		}
		out = append(out, t)
	}
	return out
}

func dateTarget(key, label, stated string, at *engine.Times, pick func(engine.Times) string) ProgramTarget {
	t := ProgramTarget{Key: key, Label: label, Stated: stated, Status: "unknown"}
	by, ok := profile.LatestDay(stated)
	switch {
	case !ok:
		t.Message = fmt.Sprintf("Owner target for %s: %s. Not a date the program can compare.", strings.ToLower(label), stated)
	case at == nil || at.Unknown || pick(*at) == "":
		t.Message = fmt.Sprintf("Owner target for %s: %s (by %s). The program has no computed date for it yet.", strings.ToLower(label), stated, wordsDate(by))
	default:
		date := pick(*at)
		t.Programmed = wordsDate(date)
		t.Status = "met"
		if date > by {
			t.Status = "later"
		}
		t.Message = fmt.Sprintf("Owner target for %s: %s (by %s). The program has it on %s.", strings.ToLower(label), stated, wordsDate(by), t.Programmed)
	}
	return t
}

func constructionMonths(t *engine.Times) (float64, bool) {
	if t == nil || t.Unknown || t.Start == "" || t.Finish == "" {
		return 0, false
	}
	a, err1 := time.Parse("2006-01-02", t.Start)
	b, err2 := time.Parse("2006-01-02", t.Finish)
	if err1 != nil || err2 != nil || b.Before(a) {
		return 0, false
	}
	return b.Sub(a).Hours() / 24 / 30.44, true
}

func wordsDate(iso string) string {
	if d, err := time.Parse("2006-01-02", iso); err == nil {
		return d.Format("2 January 2006")
	}
	return iso
}
