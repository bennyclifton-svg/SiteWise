package procurement

import "strings"

// Discipline is one entry of the filing discipline list
// (data/intake/disciplines.json), passed in by the caller so this package
// stays free of the intake loader.
type Discipline struct {
	ID      string
	Label   string
	Aliases []string
}

// DisciplinePackage is the package a Procurement row makes for a discipline.
// Consultants are services (RFP); trades are works (RFT), and Main Works is
// the head contract. Suppliers have no RFP/RFT, so they are refused.
func DisciplinePackage(id, label string) (Package, bool) {
	p := Package{DisciplineID: id, Title: label, LifecycleStatus: "planned"}
	switch {
	case strings.HasPrefix(id, "consultant."):
		p.Kind = "services"
	case id == "trade.main_works":
		p.Kind, p.WorksScope = "works", "head_contract"
	case strings.HasPrefix(id, "trade."):
		p.Kind, p.WorksScope = "works", "trade"
	default:
		return Package{}, false
	}
	return p, true
}

// ProposalDiscipline names the discipline a proposal stands for, so an
// accepted suggestion gets the discipline RFP without a confirm step.
// Consultant proposal keys end in the roster name; an unknown name yields "".
func ProposalDiscipline(key string, ds []Discipline) string {
	if key == WorksProposalKey {
		return "trade.main_works"
	}
	name, ok := strings.CutPrefix(key, "default:services:")
	if !ok {
		return ""
	}
	for _, d := range ds {
		if strings.EqualFold(d.Label, name) {
			return d.ID
		}
		for _, a := range d.Aliases {
			if strings.EqualFold(a, name) {
				return d.ID
			}
		}
	}
	return ""
}
