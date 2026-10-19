package program

// The program's stages are the project's stage plan (plan 7.6). Package
// stage sets and RFP drafts record the stage_revision they were built from;
// when the program's revision is newer, the document shows the differences
// and offers Update or Keep. Differences are keyed by the stable program
// stage id, so a rename is never mistaken for a delete and an add. All of
// this is code; no Jev.

// Stage difference kinds.
const (
	StageRenamed = "renamed"
	StageAdded   = "added"
	StageRemoved = "removed"
)

// RecordedStage is a stage as a document recorded it.
type RecordedStage struct {
	ProgramStageID string
	Label          string
}

// ProgramStage is a live program stage, in program order.
type ProgramStage struct {
	ID               string
	Label            string
	CatalogueStageID string
}

// StageChange is one difference, keyed by program stage id.
type StageChange struct {
	Kind           string `json:"kind"`
	ProgramStageID string `json:"program_stage_id"`
	OldLabel       string `json:"old_label,omitempty"`
	NewLabel       string `json:"new_label,omitempty"`
}

// StagePlanDiff is what a document is shown when its revision is behind.
// Reordered is true when the stages both sides have are in a different
// order; it changes no row on its own, but an Update rewrites row order.
type StagePlanDiff struct {
	Changes   []StageChange `json:"changes"`
	Reordered bool          `json:"reordered"`
}

// NeedsOffer is true when the program has moved past the revision the
// document was built from and past the one it last kept. Keep records the
// current revision as acknowledged, so the offer does not repeat until the
// stages change again.
func NeedsOffer(builtFrom, acknowledged, current int64) bool {
	return current > builtFrom && current > acknowledged
}

// DiffStages compares a document's recorded stages with the program's
// current ones: renamed and removed in the document's order, then added in
// program order.
func DiffStages(recorded []RecordedStage, current []ProgramStage) StagePlanDiff {
	now := map[string]ProgramStage{}
	for _, s := range current {
		now[s.ID] = s
	}
	was := map[string]bool{}
	d := StagePlanDiff{Changes: []StageChange{}}
	var keptOrder []string
	for _, r := range recorded {
		was[r.ProgramStageID] = true
		s, ok := now[r.ProgramStageID]
		switch {
		case !ok:
			d.Changes = append(d.Changes, StageChange{Kind: StageRemoved, ProgramStageID: r.ProgramStageID, OldLabel: r.Label})
		case s.Label != r.Label:
			d.Changes = append(d.Changes, StageChange{Kind: StageRenamed, ProgramStageID: r.ProgramStageID, OldLabel: r.Label, NewLabel: s.Label})
			keptOrder = append(keptOrder, r.ProgramStageID)
		default:
			keptOrder = append(keptOrder, r.ProgramStageID)
		}
	}
	k := 0
	for _, s := range current {
		if !was[s.ID] {
			d.Changes = append(d.Changes, StageChange{Kind: StageAdded, ProgramStageID: s.ID, NewLabel: s.Label})
			continue
		}
		if k < len(keptOrder) && keptOrder[k] != s.ID {
			d.Reordered = true
		}
		k++
	}
	return d
}

// Downstream documents (plan 7.6 table).
const (
	DocPackageStages   = "package_stages"
	DocDraftRFP        = "draft_rfp"
	DocIssuedRFP       = "issued_rfp"
	DocProposal        = "proposal"
	DocAwardedContract = "awarded_contract"
	DocFrozenBaseline  = "frozen_baseline"
)

// What Update does to one difference in one document.
const (
	// CopyLabel copies the program stage's new label across.
	CopyLabel = "copy_label"
	// OfferUnticked offers the added stage as an unticked box.
	OfferUnticked = "offer_unticked"
	// Retire retires the downstream stage row.
	Retire = "retire"
	// MarkRemovedKeep keeps the downstream stage, marked "removed from
	// program": it carries invoices or awarded money.
	MarkRemovedKeep = "mark_removed_keep"
	// RewriteRow rewrites the draft's stage row (rename, add or remove).
	RewriteRow = "rewrite_row"
	// NewDraftRevision creates a new draft revision; the issued one stays.
	NewDraftRevision = "new_draft_revision"
	// AddUnpricedLine adds a new unpriced line; contract values stay.
	AddUnpricedLine = "add_unpriced_line"
	// FlagMismatch changes nothing and flags the stage as not matching.
	FlagMismatch = "flag_mismatch"
	// NoChange leaves the document as it is.
	NoChange = "none"
)

// RemovedFromProgram is the mark a kept downstream stage carries.
const RemovedFromProgram = "removed from program"

// UpdateAction applies the 7.6 table to one difference. moneyAttached is
// true when the downstream stage has invoices or awarded money; such a stage
// is never removed downstream, whatever the document.
func UpdateAction(doc string, c StageChange, moneyAttached bool) string {
	if c.Kind == StageRemoved && moneyAttached {
		switch doc {
		case DocFrozenBaseline, DocProposal, DocIssuedRFP:
		default:
			return MarkRemovedKeep
		}
	}
	switch doc {
	case DocPackageStages:
		switch c.Kind {
		case StageRenamed:
			return CopyLabel
		case StageAdded:
			return OfferUnticked
		case StageRemoved:
			return Retire
		}
	case DocDraftRFP:
		return RewriteRow
	case DocIssuedRFP:
		return NewDraftRevision
	case DocProposal:
		// Proposals received never change. They match stages by id, so a
		// rename still matches; an added or removed stage does not.
		if c.Kind == StageRenamed {
			return NoChange
		}
		return FlagMismatch
	case DocAwardedContract:
		switch c.Kind {
		case StageRenamed:
			return CopyLabel
		case StageAdded:
			return AddUnpricedLine
		case StageRemoved:
			// An awarded stage carries contract money by definition.
			return MarkRemovedKeep
		}
	case DocFrozenBaseline:
		return NoChange
	}
	return NoChange
}
