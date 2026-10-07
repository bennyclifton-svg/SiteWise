package reports

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Reference carries printable detail and the original snapshot. Reading it
// never requires a live document, current catalogue or signed-in app session.
type Reference struct {
	ID       string         `json:"citation_id"`
	Label    string         `json:"label"`
	AnchorID string         `json:"anchor_id"`
	Basis    ReferenceBasis `json:"basis"`
}

type ReferenceBasis struct {
	Text               string          `json:"text"`
	Source             json.RawMessage `json:"source"`
	MaterialAssumption bool            `json:"material_assumption"`
	ProtectedEdit      bool            `json:"protected_edit"`
	UserID             string          `json:"user_id,omitempty"`
	UpdatedAt          time.Time       `json:"updated_at,omitzero"`
}

// Until a narrower materiality rule is approved, list every assumption used
// in the report. Acceptance for planning never promotes it to evidence.
func MaterialAssumptions(refs []Reference) []Reference {
	out := []Reference{}
	for _, ref := range refs {
		if ref.Basis.MaterialAssumption {
			out = append(out, ref)
		}
	}
	return out
}

// Cite adds a primary citation for every block and a separate user citation
// for protected wording. An edit does not reclassify its original evidence.
func Cite(sections []Section) ([]Section, []Reference, error) {
	out := make([]Section, len(sections))
	refs := []Reference{}
	counts := map[string]int{}
	seen := map[string]bool{}
	for i, section := range sections {
		out[i] = section
		out[i].Blocks = append([]Block(nil), section.Blocks...)
		for j := range out[i].Blocks {
			b := &out[i].Blocks[j]
			if b.ID == "" || seen[b.ID] {
				return nil, nil, fmt.Errorf("duplicate or empty citation anchor")
			}
			seen[b.ID] = true
			var source map[string]any
			dec := json.NewDecoder(strings.NewReader(string(b.Basis)))
			dec.UseNumber()
			if err := dec.Decode(&source); err != nil || source == nil {
				return nil, nil, fmt.Errorf("citation %s requires an object basis", b.ID)
			}
			label := "C"
			switch b.Origin {
			case "document":
				label = "E"
			case "user":
				label = "U"
			case "assumption":
				label = "A"
			case "calculation", "":
			default:
				return nil, nil, fmt.Errorf("unknown report origin %q", b.Origin)
			}
			b.CitationIDs = nil
			add := func(label string, basis ReferenceBasis) {
				counts[label]++
				id := fmt.Sprintf("%s%d", label, counts[label])
				b.CitationIDs = append(b.CitationIDs, id)
				refs = append(refs, Reference{ID: id, Label: label, AnchorID: b.ID, Basis: basis})
			}
			if !b.SourceMissing {
				text := referenceText(*b, label, source)
				add(label, ReferenceBasis{Text: text, Source: append(json.RawMessage(nil), b.Basis...), MaterialAssumption: label == "A"})
			}
			if b.Edited {
				text := "User wording by " + b.EditUserID + ": " + b.Text
				if !b.EditUpdatedAt.IsZero() {
					text += ". Recorded: " + b.EditUpdatedAt.UTC().Format(time.RFC3339Nano)
				}
				if b.SourceMissing {
					text += ". Original source no longer present; review required."
				}
				if b.Conflict {
					text += " Source changed; review required."
				}
				add("U", ReferenceBasis{Text: text, Source: json.RawMessage(`{}`), ProtectedEdit: true, UserID: b.EditUserID, UpdatedAt: b.EditUpdatedAt})
			}
			if len(b.CitationIDs) == 0 {
				return nil, nil, fmt.Errorf("uncited report value %s", b.ID)
			}
		}
	}
	return out, refs, nil
}

func referenceText(b Block, label string, source map[string]any) string {
	names := map[string]string{"E": "Evidence", "U": "User input", "C": "Calculation", "A": "Assumption"}
	parts := []string{names[label] + ": " + b.Label, "Review: " + known(b.ReviewStatus), "Meaning: " + known(b.Meaning)}
	if b.Provisional {
		parts = append(parts, "Provisional")
	}
	if label == "A" {
		parts = append(parts, "Material assumption requiring review")
	}
	if label == "U" || label == "A" {
		parts = append(parts, "Author: "+sourceDetail(source, "last_edited_by", "user_id", "actor"), "Recorded: "+sourceDetail(source, "last_edited_at", "updated_at", "at"))
	}
	if label == "A" {
		parts = append(parts, "Rationale: "+sourceDetail(source, "rationale", "note", "Note"))
	}
	if method, ok := source["method"].(string); ok {
		parts = append(parts, "Method: "+strings.ReplaceAll(method, "_", " "))
	}
	if label == "C" {
		parts = append(parts, calculationDetails(source)...)
	}
	docs := []string{}
	collectDocumentReferences(source, &docs)
	sort.Strings(docs)
	last := ""
	for _, doc := range docs {
		if doc != last {
			parts = append(parts, doc)
			last = doc
		}
	}
	if label == "E" && len(docs) == 0 {
		parts = append(parts, "Document details not recorded")
	}
	return strings.Join(parts, ". ")
}

// Prefer an explicitly attached audit record, then the saved value itself.
// Do not borrow an unrelated nested document's author or timestamp.
func sourceDetail(source map[string]any, keys ...string) string {
	objects := []map[string]any{}
	if audit, ok := source["audit"].(map[string]any); ok {
		objects = append(objects, audit)
	}
	objects = append(objects, source)
	if provenance, ok := source["provenance"].(map[string]any); ok {
		objects = append(objects, provenance)
	}
	for _, object := range objects {
		for _, key := range keys {
			if value, ok := object[key].(string); ok && value != "" {
				return value
			}
		}
	}
	return "not recorded"
}

func known(value string) string {
	if value == "" {
		return "not recorded"
	}
	return strings.ReplaceAll(value, "_", " ")
}

// Traverse saved provenance, including profile Sources and nested work inputs.
// Only objects with document identity are printed as evidence documents.
func collectDocumentReferences(value any, out *[]string) {
	switch v := value.(type) {
	case []any:
		for _, child := range v {
			collectDocumentReferences(child, out)
		}
	case map[string]any:
		if v["document_id"] != nil || v["filename"] != nil {
			field := func(key string) string {
				if x, ok := v[key]; ok && x != nil && fmt.Sprint(x) != "" && fmt.Sprint(x) != "0" {
					return fmt.Sprint(x)
				}
				return "not recorded"
			}
			*out = append(*out, fmt.Sprintf("File: %s; document: %s; revision: %s; page: %s; location: %s; section: %s", field("filename"), field("document_number"), field("revision"), field("page"), field("location"), field("section")))
			if excerpt, ok := v["excerpt"].(string); ok && excerpt != "" {
				(*out)[len(*out)-1] += "; excerpt: " + excerpt
			}
		}
		for _, child := range v {
			collectDocumentReferences(child, out)
		}
	}
}
