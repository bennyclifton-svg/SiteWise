package reports

import (
	"errors"
	"fmt"
	"time"
)

var ErrProtectedEditNotFound = errors.New("protected edit not found")

// ResetEdit uses the saved generated wording, not today's project inputs.
// Missing sources have no generated block to restore: remove their orphan.
func ResetEdit(sections []Section, target string) ([]Section, error) {
	out := make([]Section, 0, len(sections))
	found := false
	for _, section := range sections {
		copy := section
		copy.Blocks = make([]Block, 0, len(section.Blocks))
		for _, b := range section.Blocks {
			if b.ID == target {
				if !b.Edited || found {
					return nil, fmt.Errorf("target has no unique protected edit")
				}
				found = true
				if b.SourceMissing {
					continue
				}
				if b.GeneratedText == nil {
					return nil, fmt.Errorf("refresh this draft before restoring source wording")
				}
				b.Text = *b.GeneratedText
				b.GeneratedText = nil
				b.Edited = false
				b.Conflict = false
				b.EditUserID = ""
				b.EditUpdatedAt = time.Time{}
				b.CitationIDs = nil
			}
			copy.Blocks = append(copy.Blocks, b)
		}
		if section.ID != "protected_edits" || len(copy.Blocks) > 0 {
			out = append(out, copy)
		}
	}
	if !found {
		return nil, ErrProtectedEditNotFound
	}
	return out, nil
}
