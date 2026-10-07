// Package works holds deterministic work-item rules. It never calls a model.
package works

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"sitewise/internal/knowledge"
)

type TargetValue struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
	Unit  string          `json:"unit,omitempty"`
}
type ClauseRef struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}
type Target struct {
	Values     []TargetValue `json:"values,omitempty"`
	ClauseRefs []ClauseRef   `json:"clause_refs,omitempty"`
	Text       string        `json:"text,omitempty"`
}
type Provenance struct {
	LastEditedBy string          `json:"last_edited_by,omitempty"`
	LastEditedAt *time.Time      `json:"last_edited_at,omitempty"`
	Sources      json.RawMessage `json:"sources,omitempty"`
	Actor        string          `json:"actor,omitempty"`
	Rationale    string          `json:"rationale,omitempty"`
	// Band preserves the existing scope picker's evidence/suggestion mark.
	Band string `json:"band,omitempty"`
}
type Item struct {
	ID                    string     `json:"id"`
	ProjectID             string     `json:"project_id"`
	SiteID                string     `json:"site_id"`
	PartID                string     `json:"part_id"`
	SystemID              string     `json:"system_id"`
	Action                string     `json:"action"`
	Inclusion             string     `json:"inclusion"`
	ParentID              string     `json:"parent_id,omitempty"`
	IsGroup               bool       `json:"is_group"`
	Title                 string     `json:"title"`
	ExistingConditionNote string     `json:"existing_condition_note,omitempty"`
	Target                Target     `json:"target"`
	Quantity              *string    `json:"quantity,omitempty"`
	Unit                  *string    `json:"unit,omitempty"`
	Origin                string     `json:"origin"`
	ReviewStatus          string     `json:"review_status"`
	Meaning               string     `json:"meaning"`
	Provenance            Provenance `json:"provenance"`
	UserTouched           bool       `json:"user_touched"`
	CoarseKey             string     `json:"coarse_key,omitempty"`
	SourceProposalKey     string     `json:"source_proposal_key,omitempty"`
	RetiredAt             *time.Time `json:"retired_at,omitempty"`
	Version               int64      `json:"version"`
	Deprecated            bool       `json:"deprecated,omitempty"`
	ExistingCondition     string     `json:"existing_condition,omitempty"`
}

// CoarseID uses the RFC DNS namespace and an unambiguous project|part|system
// name. IDs survive retirement, reappearance, retries and catalogue renames.
func CoarseID(project, part, system string) string {
	namespace := []byte{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}
	h := sha1.New()
	h.Write(namespace)
	h.Write([]byte(project + "|" + part + "|" + system))
	id := h.Sum(nil)[:16]
	id[6] = (id[6] & 0x0f) | 0x50
	id[8] = (id[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])
}

// A part override wins. Without a stated work type, propose investigation;
// silently choosing a physical intervention would invent scope.
func DefaultAction(cat *knowledge.Catalog, partType, projectType string) string {
	kind := projectType
	if partType != "" {
		kind = partType
	}
	if cat != nil {
		if action := cat.DefaultAction(kind); action != "" {
			return action
		}
	}
	return LegacyDefaultAction(kind)
}

// LegacyDefaultAction freezes the migration-015 mapping, independent of future
// catalogue edits. Runtime defaults use the loaded catalogue above.
func LegacyDefaultAction(kind string) string {
	switch kind {
	case "new", "extend":
		return "new"
	case "refurb":
		return "alter"
	case "remediation":
		return "repair"
	default:
		return "investigate"
	}
}

func ValidAction(action string) bool {
	switch action {
	case "new", "replace", "upgrade", "alter", "repair", "remove", "retain", "investigate":
		return true
	}
	return false
}

var quantityRE = regexp.MustCompile(`^[0-9]{1,18}(\.[0-9]{1,9})?$`)

// Validate is shared by API and store boundaries. IDs are checked against the
// tenant/project in the store transaction, never trusted from the request.
func Validate(item Item, cat *knowledge.Catalog) error {
	if cat == nil {
		return fmt.Errorf("work catalogue unavailable")
	}
	sys, ok := cat.System(item.SystemID)
	if !ok || sys.Status == "deprecated" {
		return fmt.Errorf("unknown or deprecated system")
	}
	if !ValidAction(item.Action) {
		return fmt.Errorf("unknown action")
	}
	if item.ExistingCondition != "" {
		valid := false
		for _, condition := range cat.ExistingConditions() {
			valid = valid || condition == item.ExistingCondition
		}
		if !valid {
			return fmt.Errorf("unknown existing condition")
		}
	}
	if utf8.RuneCountInString(item.Title) > 200 || utf8.RuneCountInString(item.ExistingConditionNote) > 120 {
		return fmt.Errorf("title or existing condition note too long")
	}
	if (item.Quantity == nil) != (item.Unit == nil) {
		return fmt.Errorf("quantity and unit must be supplied together")
	}
	if item.Quantity != nil && (!quantityRE.MatchString(*item.Quantity) || strings.TrimSpace(*item.Unit) == "" || utf8.RuneCountInString(*item.Unit) > 32) {
		return fmt.Errorf("invalid quantity or unit")
	}
	if len(item.Target.Values) > 50 || len(item.Target.ClauseRefs) > 50 || utf8.RuneCountInString(item.Target.Text) > 1000 {
		return fmt.Errorf("target too large")
	}
	for _, v := range item.Target.Values {
		var scalar any
		if strings.TrimSpace(v.Key) == "" || len(v.Key) > 100 || len(v.Unit) > 32 || len(v.Value) > 500 || json.Unmarshal(v.Value, &scalar) != nil {
			return fmt.Errorf("invalid target value")
		}
		switch scalar.(type) {
		case string, float64, bool:
		default:
			return fmt.Errorf("target value must be a scalar")
		}
		if err := validateTargetValue(cat, v); err != nil {
			return err
		}
	}
	for _, ref := range item.Target.ClauseRefs {
		if strings.TrimSpace(ref.ID) == "" || len(ref.ID) > 200 || ref.Version < 1 {
			return fmt.Errorf("invalid clause reference")
		}
		if !cat.ApprovedClause(ref.ID, ref.Version) {
			return fmt.Errorf("clause reference is not an approved catalogue version")
		}
	}
	return nil
}
