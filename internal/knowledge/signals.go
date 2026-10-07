package knowledge

import "sort"

// SignalQuestions joins the evidence fan-out; it never launches another call.
func (c *Catalog) SignalQuestions(labels []string) []Question {
	out := []Question{}
	for _, s := range c.works.signals {
		if s.Status != "deprecated" && c.runsOn(labels, s.RunsOn) {
			out = append(out, s.Question())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
