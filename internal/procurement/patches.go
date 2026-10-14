package procurement

type PackagePatch struct {
	RemoveCostLines bool    `json:"remove_cost_lines,omitempty"`
	CostPlanID      string  `json:"cost_plan_id,omitempty"`
	CostPlanVersion int64   `json:"cost_plan_version,omitempty"`
	Retired         *bool   `json:"retired,omitempty"`
	Version         int64   `json:"version"`
	Kind            *string `json:"kind,omitempty"`
	WorksScope      *string `json:"works_scope,omitempty"`
	DisciplineID    *string `json:"discipline_id,omitempty"`
	Title           *string `json:"title,omitempty"`
	Novation        *string `json:"novation,omitempty"`
	EngagedBy       *string `json:"engaged_by,omitempty"`
	LifecycleStatus *string `json:"lifecycle_status,omitempty"`
}

func (p PackagePatch) Apply(to *Package) {
	if p.Kind != nil {
		to.Kind = *p.Kind
	}
	if p.WorksScope != nil {
		to.WorksScope = *p.WorksScope
	}
	if p.DisciplineID != nil {
		to.DisciplineID = *p.DisciplineID
	}
	if p.Title != nil {
		to.Title = *p.Title
	}
	if p.Novation != nil {
		to.Novation = *p.Novation
	}
	if p.EngagedBy != nil {
		to.EngagedBy = *p.EngagedBy
	}
	if p.LifecycleStatus != nil {
		to.LifecycleStatus = *p.LifecycleStatus
	}
}

type StagePatch struct {
	Retired       *bool   `json:"retired,omitempty"`
	Version       int64   `json:"version"`
	StageID       *string `json:"stage_id,omitempty"`
	Label         *string `json:"label,omitempty"`
	Ordinal       *int    `json:"ordinal,omitempty"`
	NovationPhase *string `json:"novation_phase,omitempty"`
}

func (p StagePatch) Apply(to *Stage) {
	if p.StageID != nil {
		to.StageID = *p.StageID
	}
	if p.Label != nil {
		to.Label = *p.Label
	}
	if p.Ordinal != nil {
		to.Ordinal = *p.Ordinal
	}
	if p.NovationPhase != nil {
		to.NovationPhase = *p.NovationPhase
	}
}
