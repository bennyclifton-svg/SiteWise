package reports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ContentHash covers the generated value and its source basis. A source change
// under unchanged wording still deserves review, while prior edit flags do not
// alter the generated content fingerprint.
func ContentHash(b Block) (string, error) {
	// JSONB may reorder object keys on persistence. Compare the source's value,
	// not the serializer's whitespace or key order.
	if len(b.Basis) > 0 {
		if !json.Valid(b.Basis) {
			return "", fmt.Errorf("invalid report source basis")
		}
		var basis any
		decoder := json.NewDecoder(strings.NewReader(string(b.Basis)))
		decoder.UseNumber()
		if err := decoder.Decode(&basis); err != nil {
			return "", err
		}
		var err error
		b.Basis, err = json.Marshal(basis)
		if err != nil {
			return "", err
		}
	}
	b.ContentSHA256 = ""
	b.Edited = false
	b.Conflict = false
	b.SourceMissing = false
	b.EditUserID = ""
	b.EditUpdatedAt = time.Time{}
	b.CitationIDs = nil
	b.GeneratedText = nil
	raw, err := json.Marshal(b)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func ApplyEdits(generated []Section, edits []Edit) ([]Section, error) {
	byTarget := map[string]Edit{}
	for _, edit := range edits {
		if _, ok := byTarget[edit.TargetID]; ok {
			return nil, fmt.Errorf("duplicate protected edit")
		}
		byTarget[edit.TargetID] = edit
	}
	out := make([]Section, len(generated))
	seen := map[string]bool{}
	for i, section := range generated {
		out[i] = section
		out[i].Blocks = make([]Block, len(section.Blocks))
		for j, block := range section.Blocks {
			if block.ID == "" || seen[block.ID] {
				return nil, fmt.Errorf("duplicate or empty report target")
			}
			seen[block.ID] = true
			hash, err := ContentHash(block)
			if err != nil {
				return nil, err
			}
			block.ContentSHA256 = hash
			if edit, ok := byTarget[block.ID]; ok {
				original := block.Text
				block.GeneratedText = &original
				block.Text = edit.Text
				block.Edited = true
				block.EditUserID = edit.UserID
				block.EditUpdatedAt = edit.UpdatedAt
				block.Conflict = hash != edit.BaseContentSHA256
				delete(byTarget, block.ID)
			}
			out[i].Blocks[j] = block
		}
	}
	// Deleting a source cannot delete someone's protected wording. Keep it as a
	// visible conflict until the author explicitly resolves that edit.
	if len(byTarget) > 0 {
		missing := Section{ID: "protected_edits", Title: "Edits requiring review", Essential: true, Blocks: []Block{}}
		ids := []string{}
		for id := range byTarget {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			edit := byTarget[id]
			missing.Blocks = append(missing.Blocks, Block{ID: id, Label: "Source no longer present", Text: edit.Text, Origin: "user", ReviewStatus: "accepted_for_planning", Meaning: "stated", Basis: json.RawMessage(`{}`), Edited: true, Conflict: true, SourceMissing: true, EditUserID: edit.UserID, EditUpdatedAt: edit.UpdatedAt})
		}
		out = append(out, missing)
	}
	return out, nil
}

// ReapplyEdits puts a draft's generated wording back under its current
// protected edits: after an undo changed the edits, each block shows the
// edit now saved for it, or its generated text when it has none.
func ReapplyEdits(sections []Section, edits []Edit) ([]Section, error) {
	generated := make([]Section, 0, len(sections))
	for _, s := range sections {
		if s.ID == "protected_edits" {
			continue // ApplyEdits lists orphaned edits again from edits
		}
		out := s
		out.Blocks = make([]Block, len(s.Blocks))
		for i, b := range s.Blocks {
			if b.GeneratedText != nil {
				b.Text = *b.GeneratedText
			}
			b.GeneratedText, b.Edited, b.Conflict, b.SourceMissing = nil, false, false, false
			b.EditUserID, b.EditUpdatedAt = "", time.Time{}
			out.Blocks[i] = b
		}
		generated = append(generated, out)
	}
	return ApplyEdits(generated, edits)
}
