package profile

import (
	"sitewise/internal/works"
	"strings"
)

// workContext keeps D-07's project type separate from each part override.
// Only applied, stated values can decide whether presence implies works.
type workContext struct {
	whole      string
	types      map[string]string
	actions    map[[2]string]string
	actionRows map[[2]string]Row
	// contractor marks systems an applied provider reading says the
	// contractor supplies "as part of their works or price".
	contractor map[[2]string]bool
}

func workContextFor(rows []Row, whole string) workContext {
	c := workContext{whole: whole, types: map[string]string{}, actions: map[[2]string]string{}, actionRows: map[[2]string]Row{}, contractor: map[[2]string]bool{}}
	for _, r := range rows {
		if !appliedWorkRow(r) {
			continue
		}
		if r.Key == "hdr.work_type" {
			c.types[r.PartID] = r.Value
		}
		if strings.HasPrefix(r.Key, "sys.") && strings.HasSuffix(r.Key, ".action") && (works.ValidAction(r.Value) || r.Value == "several") {
			c.actions[[2]string{r.PartID, strings.TrimSuffix(r.Key, ".action")}] = r.Value
			c.actionRows[[2]string{r.PartID, strings.TrimSuffix(r.Key, ".action")}] = r
		}
		if strings.HasPrefix(r.Key, "sys.") && strings.HasSuffix(r.Key, ".provider") && r.Value == "contractor" {
			c.contractor[[2]string{r.PartID, strings.TrimSuffix(r.Key, ".provider")}] = true
		}
	}
	return c
}

func appliedWorkRow(r Row) bool {
	return (r.Band == bandUser || r.Band == bandAmber || r.Band == bandGreen) &&
		r.Value != "" && r.ValueState != StateUnknown && r.ValueState != StateCleared &&
		r.ReviewStatus != ReviewSupersede && r.Origin != OriginAssumption &&
		(r.Meaning == "" || r.Meaning == MeaningStated)
}

func (c workContext) workType(part string) string {
	if v := c.types[part]; v != "" {
		return v
	}
	return c.types[c.whole]
}

func (c workContext) action(part, system string) string {
	return c.actions[[2]string{part, "sys." + system}]
}

func (c workContext) severalActions(r Row) bool {
	return strings.HasPrefix(r.Key, "sys.") && strings.HasSuffix(r.Key, ".presence") &&
		c.actions[[2]string{r.PartID, strings.TrimSuffix(r.Key, ".presence")}] == "several"
}

// existingPresence is D-27: the document describes what already exists, but
// supplies no applied works action. Preserve the original presence answer;
// only its scope consequence changes, so factual answer keys stay intact.
// A system the contractor provides is works, not existing building: the
// provider question defines contractor as part of the works or price.
func (c workContext) existingPresence(r Row) bool {
	if r.Band == bandUser || !appliedWorkRow(r) || r.Value != valIncluded ||
		!strings.HasPrefix(r.Key, "sys.") || !strings.HasSuffix(r.Key, ".presence") {
		return false
	}
	switch c.workType(r.PartID) {
	case "refurb", "remediation", "advisory":
		system := [2]string{r.PartID, strings.TrimSuffix(r.Key, ".presence")}
		if c.contractor[system] {
			return false
		}
		action := c.actions[system]
		return action == "" || action == "not_stated"
	}
	return false
}

func withExistingSystems(rows []Row, whole string) []Row {
	c := workContextFor(rows, whole)
	present := map[[2]string]bool{}
	for _, r := range rows {
		present[[2]string{r.PartID, r.Key}] = true
	}
	for _, r := range rows {
		if !c.existingPresence(r) {
			continue
		}
		r.Key = strings.TrimSuffix(r.Key, ".presence") + ".existing"
		if present[[2]string{r.PartID, r.Key}] {
			continue
		} // a site user value is final
		r.Value = "present"
		rows = append(rows, r) // retain the complete document provenance
	}
	return rows
}
