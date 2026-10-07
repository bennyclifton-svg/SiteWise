package procurement

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"sitewise/internal/knowledge"
)

type Package struct {
	ID                string          `json:"id"`
	ProjectID         string          `json:"project_id"`
	Kind              string          `json:"kind"`
	WorksScope        string          `json:"works_scope,omitempty"`
	DisciplineID      string          `json:"discipline_id,omitempty"`
	Title             string          `json:"title"`
	Novation          bool            `json:"novation"`
	LifecycleStatus   string          `json:"lifecycle_status"`
	Origin            string          `json:"origin"`
	ReviewStatus      string          `json:"review_status"`
	Meaning           string          `json:"meaning"`
	Provenance        json.RawMessage `json:"provenance"`
	SourceProposalKey string          `json:"source_proposal_key,omitempty"`
	RetiredAt         *time.Time      `json:"retired_at,omitempty"`
	Version           int64           `json:"version"`
	Stages            []Stage         `json:"stages"`
}

type Stage struct {
	ID            string     `json:"id"`
	ProjectID     string     `json:"project_id"`
	PackageID     string     `json:"package_id"`
	StageID       string     `json:"stage_id"`
	Label         string     `json:"label"`
	Ordinal       int        `json:"ordinal"`
	NovationPhase string     `json:"novation_phase"`
	Origin        string     `json:"origin"`
	RetiredAt     *time.Time `json:"retired_at,omitempty"`
	Version       int64      `json:"version"`
}

func ValidatePackage(p Package) error {
	if p.Kind != "services" && p.Kind != "works" && p.Kind != "supply" {
		return fmt.Errorf("unknown package kind")
	}
	if p.Kind == "works" {
		if p.WorksScope != "head_contract" && p.WorksScope != "trade" {
			return fmt.Errorf("works scope is required")
		}
	} else if p.WorksScope != "" {
		return fmt.Errorf("works scope belongs only to works packages")
	}
	if p.Novation && p.Kind != "services" {
		return fmt.Errorf("only services packages can be novated")
	}
	if !bounded(p.Title, 200) {
		return fmt.Errorf("package title must contain 1–200 characters")
	}
	if p.DisciplineID != "" && !bounded(p.DisciplineID, 200) {
		return fmt.Errorf("invalid discipline identifier")
	}
	switch p.LifecycleStatus {
	case "proposed", "planned", "procuring", "appointed", "closed", "cancelled":
	default:
		return fmt.Errorf("unknown lifecycle status")
	}
	return nil
}

func ValidateStage(s Stage, p Package, cat *knowledge.Catalog) error {
	if cat == nil {
		return fmt.Errorf("stage catalogue unavailable")
	}
	if _, ok := cat.DeliveryStage(s.StageID); !ok {
		return fmt.Errorf("unknown stage")
	}
	if !bounded(s.Label, 200) || s.Ordinal < 0 {
		return fmt.Errorf("invalid stage label or ordinal")
	}
	if s.NovationPhase != "none" && s.NovationPhase != "pre" && s.NovationPhase != "post" {
		return fmt.Errorf("unknown novation phase")
	}
	if s.NovationPhase != "none" && (!p.Novation || p.Kind != "services") {
		return fmt.Errorf("pre/post stages require a novated services package")
	}
	return nil
}

// Defaults use the catalogue's editable design/construction substages. Their
// order is data order; novation labels never imply an appointment or transfer.
func DefaultStages(p Package, cat *knowledge.Catalog) []Stage {
	out := []Stage{}
	if cat == nil || p.Kind != "services" {
		return out
	}
	for _, s := range cat.DeliveryStages() {
		if s.ParentID == "" {
			continue
		}
		phase := "none"
		if p.Novation {
			phase = s.Novation
		}
		out = append(out, Stage{StageID: s.ID, Label: s.Label, Ordinal: len(out), NovationPhase: phase, Origin: "calculation", Version: 1})
	}
	return out
}

func bounded(s string, n int) bool {
	return utf8.ValidString(s) && strings.TrimSpace(s) != "" && utf8.RuneCountInString(s) <= n
}
