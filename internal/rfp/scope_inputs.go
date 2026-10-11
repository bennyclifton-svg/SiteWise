package rfp

import (
	"strings"

	"sitewise/internal/knowledge"
)

// Inputs remain facts, not scope overrides. Assign each its first affected
// service or stage; shared appointment parameters have a scope-basis home.
func assignScopeInputs(c *Controls, d knowledge.RFPDiscipline, sk knowledge.RFPSkeleton) {
	uses := func(key, module string, whens ...knowledge.RFPWhen) bool {
		for _, m := range sk.Modules {
			if m.Key == module {
				whens = append(whens, m.When)
			}
		}
		for _, when := range whens {
			for _, clause := range when {
				for name, values := range clause {
					if name == key {
						return true
					}
					if name == "flag" && strings.HasPrefix(key, "flag.") && contains(values, strings.TrimPrefix(key, "flag.")) {
						return true
					}
				}
			}
		}
		return false
	}
	for i := range c.Groups {
		g := &c.Groups[i]
		if g.ID != "appointment" {
			continue
		}
		for j := range g.Controls {
			input := &g.Controls[j]
			input.Target = "scope_basis"
			key := strings.TrimPrefix(input.Key, "fact:")
			for _, p := range d.Positions {
				whens := []knowledge.RFPWhen{p.When, p.Suggest}
				for _, cell := range []*knowledge.RFPCell{p.Value, p.Simple, p.Complex} {
					if cell != nil {
						whens = append(whens, cell.When)
					}
				}
				if uses(key, p.Module, whens...) {
					input.Target = "position:" + p.Key
					break
				}
			}
			if input.Target != "scope_basis" {
				continue
			}
			for _, stage := range d.Stages {
				if uses(key, stage.Module, stage.When, stage.Suggest, stage.IncludeWhen) {
					input.Target = "stage:" + stage.Key
					break
				}
			}
		}
		g.Controls = append(g.Controls, Control{Key: "scale", Label: "Service level", Kind: "choice", Value: c.Scale.Value, Source: c.Scale.Source, Set: c.Scale.Set, Options: []Choice{{"simple", "Simple"}, {"complex", "Complex"}}, Target: "scope_basis", Help: d.Scale.Text})
	}
}
