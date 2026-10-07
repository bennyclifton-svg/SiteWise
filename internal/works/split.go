package works

import (
	"fmt"
	"sitewise/internal/knowledge"
	"strings"
)

type SplitChild struct {
	LayoutChange string `json:"layout_change,omitempty"`
	PartID       string `json:"part_id,omitempty"`
	SystemID     string `json:"system_id,omitempty"`
	Action       string `json:"action,omitempty"`
	Title        string `json:"title"`
}
type SplitRequest struct {
	Version  int64        `json:"version"`
	Children []SplitChild `json:"children"`
}

// SplitChildren keeps evidence lineage, but planning acceptance is not verification.
// Quantities stay on the parent until explicitly allocated to avoid double counting.
func SplitChildren(parent Item, request SplitRequest, actor string, cat *knowledge.Catalog) ([]Item, error) {
	if request.Version < 1 || len(request.Children) < 2 || len(request.Children) > 100 {
		return nil, fmt.Errorf("split requires version and 2 to 100 children")
	}
	out := make([]Item, 0, len(request.Children))
	for _, child := range request.Children {
		item := parent
		item.ID = ""
		item.ParentID = parent.ID
		item.IsGroup = false
		item.CoarseKey = ""
		item.SourceProposalKey = ""
		item.RetiredAt = nil
		item.Version = 1
		item.LayoutChange = child.LayoutChange
		if item.LayoutChange == "" {
			item.LayoutChange = "unknown"
		}
		item.Quantity = nil
		item.Unit = nil
		item.Title = strings.TrimSpace(child.Title)
		if child.PartID != "" {
			item.PartID = child.PartID
		}
		if child.SystemID != "" {
			item.SystemID = child.SystemID
		}
		if child.Action != "" {
			item.Action = child.Action
		}
		item.Origin = "user"
		item.ReviewStatus = "accepted_for_planning"
		item.Meaning = "stated"
		item.UserTouched = true
		item.Provenance.LastEditedBy = actor
		if item.Title == "" {
			return nil, fmt.Errorf("each child needs a title")
		}
		if err := Validate(item, cat); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}
