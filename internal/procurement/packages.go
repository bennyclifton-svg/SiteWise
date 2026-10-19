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
	ID           string `json:"id"`
	ProjectID    string `json:"project_id"`
	Kind         string `json:"kind"`
	WorksScope   string `json:"works_scope,omitempty"`
	DisciplineID string `json:"discipline_id,omitempty"`
	Title        string `json:"title"`
	Novation     string `json:"novation"`
	// EngagedBy is EngagedByClient, EngagedByLead or "" (not decided).
	EngagedBy         string          `json:"engaged_by,omitempty"`
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
	// ProgramStageID is the program stage this stage points to and
	// StageRevision the program stage revision it was built from (program
	// plan 7.6). RemovedFromProgram marks a stage kept downstream after its
	// program stage was deleted, because money is attached to it.
	ProgramStageID     string     `json:"program_stage_id,omitempty"`
	StageRevision      int64      `json:"stage_revision,omitempty"`
	RemovedFromProgram *time.Time `json:"removed_from_program,omitempty"`
}

// ProgramStage is a live program stage, in program order: the project's
// stage plan that package stages point to (program plan 7.6).
type ProgramStage struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	CatalogueStageID string `json:"catalogue_stage_id"`
}

// Who engages a services package: the client directly, or the lead design
// consultant as a subconsultant (owner decision, 13 October 2026).
const (
	EngagedByClient = "client"
	EngagedByLead   = "lead_consultant"
)

// Novation is "no", "option" (the consultant prices the delivery phase and
// the client may elect to novate later) or "yes" (the client has elected).
// Design & Construct proposes an option, never a novation.
const (
	NovationNo     = "no"
	NovationOption = "option"
	NovationYes    = "yes"
)

// NovationState reads an unset novation as no.
func (p Package) NovationState() string {
	if p.Novation == "" {
		return NovationNo
	}
	return p.Novation
}

// Novated reports whether the package has stages before and after a
// possible novation: an elected one, or an option to price.
func (p Package) Novated() bool { return p.NovationState() != NovationNo }

// DefaultNovation is the state a new consultant package starts in: an option
// when the route puts design with the builder, otherwise no.
func DefaultNovation(route string) string {
	if BuilderDesigns(route) {
		return NovationOption
	}
	return NovationNo
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
	switch p.Novation {
	case "", "no", "option", "yes":
	default:
		return fmt.Errorf("novation must be no, option or yes")
	}
	if p.Novated() && p.Kind != "services" {
		return fmt.Errorf("only services packages can be novated")
	}
	switch p.EngagedBy {
	case "", EngagedByClient, EngagedByLead:
	default:
		return fmt.Errorf("engaged_by must be client or lead_consultant")
	}
	if p.EngagedBy != "" && p.Kind != "services" {
		return fmt.Errorf("only a consultant is engaged by the client or the lead consultant")
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
	if s.NovationPhase != "none" && (!p.Novated() || p.Kind != "services") {
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
		if p.Novated() {
			phase = s.Novation
		}
		out = append(out, Stage{StageID: s.ID, Label: s.Label, Ordinal: len(out), NovationPhase: phase, Origin: "calculation", Version: 1})
	}
	return out
}

// ProgramDefaultStages is DefaultStages once the project has a program
// (program plan 7.6): a services package gets the program stages whose
// catalogue stage is one of the catalogue sub-stages DefaultStages copies,
// in program order, labelled as the program labels them and pointing to
// them. Nothing is copied from the catalogue but the novation side.
func ProgramDefaultStages(p Package, cat *knowledge.Catalog, plan []ProgramStage, revision int64) []Stage {
	out := []Stage{}
	if cat == nil || p.Kind != "services" {
		return out
	}
	for _, ps := range plan {
		s, ok := cat.DeliveryStage(ps.CatalogueStageID)
		if !ok || s.ParentID == "" {
			continue
		}
		out = append(out, Stage{StageID: s.ID, Label: ps.Label, Ordinal: len(out), NovationPhase: ProgramNovationPhase(p, cat, s.ID),
			Origin: "calculation", Version: 1, ProgramStageID: ps.ID, StageRevision: revision})
	}
	return out
}

// ProgramNovationPhase is the side of a novation a stage falls on, from the
// catalogue, for a novated package; "none" otherwise or for a stage the
// catalogue gives no side (a top-level or user-made stage).
func ProgramNovationPhase(p Package, cat *knowledge.Catalog, catalogueStageID string) string {
	if cat == nil || !p.Novated() || p.Kind != "services" {
		return "none"
	}
	if s, ok := cat.DeliveryStage(catalogueStageID); ok && (s.Novation == "pre" || s.Novation == "post") {
		return s.Novation
	}
	return "none"
}

func bounded(s string, n int) bool {
	return utf8.ValidString(s) && strings.TrimSpace(s) != "" && utf8.RuneCountInString(s) <= n
}
