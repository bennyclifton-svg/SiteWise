package program

import (
	"fmt"
	"strings"
)

// The ask-first review (plan 7.4): after a program exists, a change made
// elsewhere never edits it. The review job regenerates in memory and Diff
// lists each difference with the ops Adopt would apply. Matching is by
// source_key only, never by label, so a renamed row is a rename, not a
// removal and an addition.

// Change kinds (program_changes.kind).
const (
	ChangeAdd      = "add"
	ChangeRemove   = "remove"
	ChangeRename   = "rename"
	ChangeDuration = "duration"
	ChangeLink     = "link"
)

// Default choices.
const (
	ChoiceAdopt = "adopt"
	ChoiceKeep  = "keep"
)

// Change sources (program_changes.source), from the source key's prefix.
const (
	SourceProfile     = "profile"
	SourceWorkItems   = "work_items"
	SourceProcurement = "procurement"
	SourceLibrary     = "library"
)

// EditedSince is appended to the summary of a change that defaults to Keep
// because the user has touched the row.
const EditedSince = "You've edited this since."

// LiveRow is a live (not retired) program row as the store holds it.
type LiveRow struct {
	ID               string
	SourceKey        string // "" for a user-made row
	ParentID         string
	Kind             string
	Label            string
	CatalogueStageID string
	PackageID        string
	Duration         *int
	Low, High        *int
	DurationOrigin   string
	Origin           string // library, user or document
	ReviewStatus     string // proposed or accepted
	// GenLabel is the label generation last gave the row, set only while
	// the row carries a label the user typed. A rename back to it is the
	// user's own edit undone, never a difference (plan section 2 rule 2).
	GenLabel string
}

// Edited is true once the user has created, edited or accepted the row.
// Editing a library row sets origin user and review_status accepted (8.1).
func (r LiveRow) Edited() bool {
	return r.Origin != OriginLibrary || r.ReviewStatus != ReviewProposed
}

// LiveLink is a live program link.
type LiveLink struct {
	ID        string
	PredID    string
	SuccID    string
	Type      string
	Lag       int
	LagUnit   string
	Origin    string
	SourceKey string // "" for a user-drawn link
	// GenShape is the type and lag generation last gave the link, set only
	// while the link carries a type or lag the user chose.
	GenShape *LinkShape
}

// LinkShape is a link's type and lag.
type LinkShape struct {
	Type    string
	Lag     int
	LagUnit string
}

// UserRetired names, by source key, the generated rows and links the user
// deleted. A deletion is a user decision (plan 7.4, 13.1), so the review
// never offers the row or link back; a row the review itself removed is not
// listed and comes back when its input does.
type UserRetired struct {
	Rows  map[string]bool
	Links map[string]bool
}

// Op is one op of the batch Adopt applies, in the vocabulary of plan 8.1.
// An `add` made by the review carries the generated row's library fields
// (source_key, catalogue_stage_id, package and work item ids, systems, the
// duration range and its origin and source, origin "library") beside the
// 8.1 fields, so the adopted row is the generated row and not a user row.
// Ids that are not yet stored are temp ids equal to the row's source key;
// a link op whose temp id is not added in the same batch is dropped by the
// store, since its row was kept out.
type Op map[string]any

// Change is one review item (a program_changes row).
type Change struct {
	Kind             string `json:"kind"`
	Source           string `json:"source"`
	SourceKey        string `json:"source_key"`
	TargetActivityID string `json:"target_activity_id,omitempty"`
	Summary          string `json:"summary"`
	Ops              []Op   `json:"ops"`
	DefaultChoice    string `json:"default_choice"`
	Edited           bool   `json:"edited"`
	// PackageID is set on the removal of a package row, so an edited one
	// can feed the package_removed check (RemovedPackageRows).
	PackageID string `json:"package_id,omitempty"`
}

// SourceOf names what changed for a source key: packages are Procurement,
// work items and hold points are the Profile's work items, skeleton stages
// are the Profile; anything else is the library.
func SourceOf(key string) string {
	switch {
	case strings.HasPrefix(key, "package:"):
		return SourceProcurement
	case strings.HasPrefix(key, "fragment:"), strings.HasPrefix(key, "investigate:"), strings.HasPrefix(key, "hold:"):
		return SourceWorkItems
	case strings.HasPrefix(key, "skeleton:"):
		return SourceProfile
	}
	return SourceLibrary
}

// Diff compares the live program with a regenerated plan. It writes
// nothing. Changes come in a fixed order: additions, renames and duration
// changes in plan order, removals in live order, then link changes (plan
// order, then live order).
func Diff(live []LiveRow, liveLinks []LiveLink, plan Plan, gone UserRetired) []Change {
	byKey := map[string]LiveRow{}
	byID := map[string]LiveRow{}
	for _, r := range live {
		byID[r.ID] = r
		if r.SourceKey != "" {
			byKey[r.SourceKey] = r
		}
	}
	planKeys := map[string]Row{}
	for _, r := range plan.Rows {
		planKeys[r.SourceKey] = r
	}
	// ref names a plan row in an op: its stored id, or its temp id.
	ref := func(key string) string {
		if r, ok := byKey[key]; ok {
			return r.ID
		}
		return key
	}
	var changes []Change
	added := map[string]int{} // plan key → index of its add change
	// held: plan rows not offered because the user deleted them, or deleted
	// the stage they sit in.
	held := map[string]bool{}

	for i, r := range plan.Rows {
		lr, ok := byKey[r.SourceKey]
		if !ok && (gone.Rows[r.SourceKey] || held[r.ParentKey]) {
			held[r.SourceKey] = true
			continue
		}
		if !ok {
			op := addOp(r, plan, i, ref, held)
			added[r.SourceKey] = len(changes)
			changes = append(changes, Change{Kind: ChangeAdd, Source: SourceOf(r.SourceKey), SourceKey: r.SourceKey,
				Summary: fmt.Sprintf("Add %s.", quote(r.Label)), Ops: []Op{op}, DefaultChoice: ChoiceAdopt})
			continue
		}
		if lr.Label != r.Label && !(lr.GenLabel != "" && lr.GenLabel == r.Label) {
			c := Change{Kind: ChangeRename, Source: SourceOf(r.SourceKey), SourceKey: r.SourceKey, TargetActivityID: lr.ID,
				Summary: fmt.Sprintf("Rename %s to %s.", quote(lr.Label), quote(r.Label)),
				Ops:     []Op{{"op": "update", "id": lr.ID, "label": r.Label}}}
			changes = append(changes, withDefault(c, lr))
		}
		if !sameDuration(lr, r) {
			c := Change{Kind: ChangeDuration, Source: SourceOf(r.SourceKey), SourceKey: r.SourceKey, TargetActivityID: lr.ID,
				Summary: fmt.Sprintf("%s: duration %s to %s.", lr.Label, days(lr.Duration), days(r.Duration)),
				Ops: []Op{{"op": "update", "id": lr.ID, "duration_days": r.Duration, "duration_low": r.Low,
					"duration_high": r.High, "duration_origin": r.DurationOrigin, "duration_source": r.DurationSource}}}
			changes = append(changes, withDefault(c, lr))
		}
	}

	for _, lr := range live {
		if lr.SourceKey == "" {
			continue // a row the user made is never proposed for removal
		}
		if _, ok := planKeys[lr.SourceKey]; ok {
			continue
		}
		if parent, ok := byID[lr.ParentID]; ok && parent.SourceKey != "" {
			if _, kept := planKeys[parent.SourceKey]; !kept {
				continue // the stage's removal takes its activities with it
			}
		}
		// Deleting a stage takes its activities, so an edited activity
		// inside it makes the stage's removal default to Keep as well.
		edited := lr.Edited()
		var editedLabel string
		if edited {
			editedLabel = lr.Label
		}
		if lr.Kind == KindStage {
			for _, child := range live {
				if child.ParentID == lr.ID && child.Edited() && !edited {
					edited, editedLabel = true, child.Label
				}
			}
		}
		c := Change{Kind: ChangeRemove, Source: SourceOf(lr.SourceKey), SourceKey: lr.SourceKey, TargetActivityID: lr.ID,
			Summary: removeSummary(lr, editedLabel), Ops: []Op{{"op": "delete", "ids": []string{lr.ID}}},
			DefaultChoice: ChoiceAdopt}
		if edited {
			c.DefaultChoice, c.Edited = ChoiceKeep, true
		}
		// Packages no longer generate rows (owner direction 25 October 2026),
		// so an old package row going is a fold into the stage's own lines,
		// not a package removed in Procurement: PackageID stays empty and the
		// package_removed check does not name it.
		changes = append(changes, c)
	}

	liveLinkByKey := map[string]LiveLink{}
	for _, l := range liveLinks {
		if l.SourceKey != "" {
			liveLinkByKey[l.SourceKey] = l
		}
	}
	planLinkKeys := map[string]bool{}
	for _, l := range plan.Links {
		planLinkKeys[l.SourceKey] = true
		ll, ok := liveLinkByKey[l.SourceKey]
		if !ok && (gone.Links[l.SourceKey] || held[l.PredKey] || held[l.SuccKey]) {
			continue
		}
		if !ok {
			op := Op{"op": "link_add", "pred_id": ref(l.PredKey), "succ_id": ref(l.SuccKey), "type": l.Type,
				"lag_days": l.Lag, "lag_unit": l.LagUnit, "origin": l.Origin, "source_key": l.SourceKey}
			if l.InterfaceID != "" {
				op["interface_id"] = l.InterfaceID
			}
			// A link to a row being added travels with that row's add.
			if i, ok := added[l.SuccKey]; ok {
				changes[i].Ops = append(changes[i].Ops, op)
				continue
			}
			if i, ok := added[l.PredKey]; ok {
				changes[i].Ops = append(changes[i].Ops, op)
				continue
			}
			changes = append(changes, Change{Kind: ChangeLink, Source: SourceOf(l.SuccKey), SourceKey: l.SourceKey,
				Summary: fmt.Sprintf("Link %s to %s.", quote(labelOf(l.PredKey, byKey, planKeys)), quote(labelOf(l.SuccKey, byKey, planKeys))),
				Ops:     []Op{op}, DefaultChoice: ChoiceAdopt})
			continue
		}
		if g := ll.GenShape; g != nil && g.Type == l.Type && g.Lag == l.Lag && unitOr(g.LagUnit) == unitOr(l.LagUnit) {
			continue // the user's own type or lag, and generation's has not moved
		}
		if ll.Type != l.Type || ll.Lag != l.Lag || unitOr(ll.LagUnit) != unitOr(l.LagUnit) {
			c := Change{Kind: ChangeLink, Source: SourceOf(l.SuccKey), SourceKey: l.SourceKey,
				Summary: fmt.Sprintf("Link %s to %s becomes %s%s.", quote(labelOf(l.PredKey, byKey, planKeys)),
					quote(labelOf(l.SuccKey, byKey, planKeys)), l.Type, lagText(l.Lag)),
				Ops:           []Op{{"op": "link_update", "id": ll.ID, "type": l.Type, "lag_days": l.Lag, "lag_unit": l.LagUnit}},
				DefaultChoice: ChoiceAdopt}
			if ll.Origin == OriginUser || ll.Origin == OriginDocument {
				c.DefaultChoice, c.Edited = ChoiceKeep, true
				c.Summary += " " + EditedSince
			}
			changes = append(changes, c)
		}
	}
	for _, ll := range liveLinks {
		if ll.SourceKey == "" || planLinkKeys[ll.SourceKey] {
			continue
		}
		p, okp := byID[ll.PredID]
		s, oks := byID[ll.SuccID]
		if !okp || !oks || removedRow(p, planKeys) || removedRow(s, planKeys) {
			continue // a removed row retires its links with it
		}
		c := Change{Kind: ChangeLink, Source: SourceOf(s.SourceKey), SourceKey: ll.SourceKey,
			Summary:       fmt.Sprintf("Remove the link from %s to %s.", quote(p.Label), quote(s.Label)),
			Ops:           []Op{{"op": "link_delete", "ids": []string{ll.ID}}},
			DefaultChoice: ChoiceAdopt}
		if ll.Origin == OriginUser || ll.Origin == OriginDocument {
			c.DefaultChoice, c.Edited = ChoiceKeep, true
			c.Summary += " " + EditedSince
		}
		changes = append(changes, c)
	}
	return changes
}

// RemovedPackageRows lists rows of removed packages the user had edited:
// they stay, and the package_removed check names them (plan 6.6).
func RemovedPackageRows(changes []Change) []string {
	var ids []string
	for _, c := range changes {
		if c.Kind == ChangeRemove && c.PackageID != "" && c.Edited {
			ids = append(ids, c.TargetActivityID)
		}
	}
	return ids
}

// withDefault: Adopt only for a row still proposed and unedited; Keep for
// anything the user created, edited or accepted.
func withDefault(c Change, lr LiveRow) Change {
	c.DefaultChoice = ChoiceAdopt
	if lr.Edited() {
		c.DefaultChoice, c.Edited = ChoiceKeep, true
		c.Summary += " " + EditedSince
	}
	return c
}

// removeSummary says why a generated row would go, e.g. "Remove “Basement
// excavation”: its work item is no longer in scope; you edited “Basement
// excavation”." edited names the edited row, "" when none.
func removeSummary(lr LiveRow, edited string) string {
	var why string
	switch {
	case strings.HasPrefix(lr.SourceKey, "package:"):
		why = "consultants now follow the stage's own activities"
	case strings.HasPrefix(lr.SourceKey, "fragment:"), strings.HasPrefix(lr.SourceKey, "investigate:"):
		why = "its work item is no longer in scope"
	case strings.HasPrefix(lr.SourceKey, "hold:"):
		why = "the hold point no longer applies"
	case strings.HasPrefix(lr.SourceKey, "skeleton:"):
		why = "the program template no longer has this stage"
	default:
		why = "no longer generated"
	}
	if edited != "" {
		return fmt.Sprintf("Remove %s: %s; you edited %s.", quote(lr.Label), why, quote(edited))
	}
	return fmt.Sprintf("Remove %s: %s.", quote(lr.Label), why)
}

func removedRow(r LiveRow, plan map[string]Row) bool {
	if r.SourceKey == "" {
		return false
	}
	_, ok := plan[r.SourceKey]
	return !ok
}

func addOp(r Row, plan Plan, i int, ref func(string) string, held map[string]bool) Op {
	op := Op{"op": "add", "temp_id": r.SourceKey, "kind": r.Kind, "label": r.Label, "duration_days": r.Duration,
		"duration_low": r.Low, "duration_high": r.High, "duration_origin": r.DurationOrigin,
		"duration_source": r.DurationSource, "source_key": r.SourceKey, "origin": OriginLibrary, "systems": r.Systems}
	if r.ParentKey == "" {
		op["parent_id"] = nil
	} else {
		op["parent_id"] = ref(r.ParentKey)
	}
	// after_id is the plan row before this one in the same parent, so the
	// added row lands where generation put it.
	for j := i - 1; j >= 0; j-- {
		if plan.Rows[j].ParentKey == r.ParentKey && !held[plan.Rows[j].SourceKey] {
			op["after_id"] = ref(plan.Rows[j].SourceKey)
			break
		}
	}
	for k, v := range map[string]string{"catalogue_stage_id": r.CatalogueStageID, "package_id": r.PackageID,
		"package_stage_id": r.PackageStageID, "work_item_id": r.WorkItemID, "fragment_id": r.FragmentID, "interface_id": r.InterfaceID} {
		if v != "" {
			op[k] = v
		}
	}
	return op
}

func sameDuration(lr LiveRow, r Row) bool {
	return eqInt(lr.Duration, r.Duration) && eqInt(lr.Low, r.Low) && eqInt(lr.High, r.High)
}

func eqInt(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func days(d *int) string {
	switch {
	case d == nil:
		return "unknown"
	case *d == 1:
		return "1 day"
	}
	return fmt.Sprintf("%d days", *d)
}

func lagText(lag int) string {
	switch {
	case lag > 0:
		return fmt.Sprintf("+%dd", lag)
	case lag < 0:
		return fmt.Sprintf("%dd", lag)
	}
	return ""
}

func unitOr(u string) string {
	if u == "" {
		return LagWorking
	}
	return u
}

func labelOf(key string, live map[string]LiveRow, plan map[string]Row) string {
	if r, ok := live[key]; ok {
		return r.Label
	}
	return plan[key].Label
}

func quote(s string) string { return "“" + s + "”" }
