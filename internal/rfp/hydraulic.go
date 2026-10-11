package rfp

func (r *resolver) hydraulicSourceFacts() {
	old := r.fact("flag.hyd_sources")
	for _, key := range []string{"flag.hyd_rainwater", "flag.hyd_alternative_sources"} {
		if r.fact(key).State != Unknown {
			continue
		}
		// The old umbrella answer can establish absence, but a positive
		// answer cannot identify which source was intended.
		if old.State == Known && contains(old.Values, "no") {
			r.out.Facts[key] = Fact{State: Known, Values: []string{"no"}, Source: "derived", Detail: "No alternative sources recorded in the previous combined answer"}
		} else if old.State == Conflict {
			r.out.Facts[key] = old
		} else if old.State == Known && contains(old.Values, "yes") {
			r.check(Check{ID: "legacy:" + key, Label: "Separate the recorded water sources", Text: "The previous combined answer says an alternative source applies. Confirm rainwater reuse and other sources separately.", Blocking: true, Control: "fact:" + key})
		}
	}
}

func hydraulicStageService(key string) bool {
	return key == "existing" || key == "concept" || key == "continuation" || key == "construction" || key == "witness" || key == "defects" || key == "advice"
}
