package procurement

import (
	"fmt"
	"slices"
	"unicode/utf8"

	"sitewise/internal/knowledge"
)

// ScopeContent contains editable wording and assignments. Identity, source
// snapshots and review provenance are resolved by the store, never by a client.
type ScopeContent struct {
	ItemKind      string   `json:"item_kind"`
	WorkItemID    string   `json:"work_item_id,omitempty"`
	Role          string   `json:"role,omitempty"`
	ClauseID      string   `json:"clause_id,omitempty"`
	ClauseVersion int      `json:"clause_version,omitempty"`
	UserText      string   `json:"user_text,omitempty"`
	StageID       string   `json:"stage_id,omitempty"`
	Inclusion     string   `json:"inclusion"`
	Deliverable   string   `json:"deliverable,omitempty"`
	InterfaceIDs  []string `json:"interface_ids"`
}

// ValidateScope checks catalogue and package semantics. The store must also
// check that work and stage IDs are live and belong to this project/package.
func ValidateScope(s ScopeContent, p Package, cat *knowledge.Catalog) error {
	if cat == nil {
		return fmt.Errorf("scope catalogue unavailable")
	}
	if s.ItemKind != "responsibility" && s.ItemKind != "obligation" {
		return fmt.Errorf("unknown scope item kind")
	}
	if s.ItemKind == "responsibility" && (s.WorkItemID == "" || s.Role == "") {
		return fmt.Errorf("responsibility requires a work item and role")
	}
	if s.Role != "" && !slices.Contains([]string{"design", "document", "supply", "install", "test", "certify", "inspect", "maintain_operation", "protect"}, s.Role) {
		return fmt.Errorf("unknown responsibility role")
	}
	if p.Kind == "supply" && s.Role != "" && s.Role != "supply" {
		return fmt.Errorf("supply packages can only hold the supply role")
	}
	if s.Inclusion != "included" && s.Inclusion != "excluded" {
		return fmt.Errorf("unknown scope inclusion")
	}
	if (s.ClauseID == "") == (s.UserText == "") {
		return fmt.Errorf("choose exactly one catalogue clause or user text")
	}
	if s.ClauseID != "" {
		if _, ok := cat.Clause(s.ClauseID, s.ClauseVersion); !ok {
			return fmt.Errorf("unknown clause or version")
		}
	} else if s.ClauseVersion != 0 || !bounded(s.UserText, 10000) {
		return fmt.Errorf("invalid user wording or unexpected clause version")
	}
	if !utf8.ValidString(s.Deliverable) || utf8.RuneCountInString(s.Deliverable) > 2000 {
		return fmt.Errorf("deliverable exceeds 2000 characters or is invalid text")
	}
	if len(s.InterfaceIDs) > 100 {
		return fmt.Errorf("too many interfaces")
	}
	seen := map[string]bool{}
	for _, id := range s.InterfaceIDs {
		if seen[id] {
			return fmt.Errorf("duplicate interface")
		}
		seen[id] = true
		if !slices.ContainsFunc(cat.Interfaces, func(i knowledge.Interface) bool { return i.ID == id }) {
			return fmt.Errorf("unknown interface")
		}
	}
	return nil
}

// ClauseProvisional is computed on read; accepting a draft for planning does
// not approve its standard wording. Missing versions also fail closed.
func ClauseProvisional(s ScopeContent, cat *knowledge.Catalog) bool {
	return s.ClauseID != "" && (cat == nil || !cat.ApprovedClause(s.ClauseID, s.ClauseVersion))
}
