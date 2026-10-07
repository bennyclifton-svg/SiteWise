package reports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// IssueSnapshot is the complete input to reproduction. Neither rendering nor
// download reads current source documents, knowledge or project records.
type IssueSnapshot struct {
	SchemaVersion   int         `json:"schema_version"`
	ReportID        string      `json:"report_id"`
	VersionID       string      `json:"version_id"`
	Kind            string      `json:"kind"`
	Title           string      `json:"title"`
	ReportingDate   string      `json:"reporting_date"`
	SourceRevisions SourceState `json:"source_revisions"`
	Sections        []Section   `json:"sections"`
	References      []Reference `json:"references"`
	BudgetDisclosed bool        `json:"budget_disclosed"`
	StaleReason     string      `json:"stale_reason,omitempty"`
	RendererVersion string      `json:"renderer_version"`
}

func CanonicalSnapshot(s IssueSnapshot) ([]byte, string, error) {
	if s.SchemaVersion != 1 || s.ReportID == "" || s.VersionID == "" || s.Title == "" || s.RendererVersion == "" {
		return nil, "", fmt.Errorf("incomplete issue snapshot")
	}
	for _, section := range s.Sections {
		for _, b := range section.Blocks {
			if b.Conflict || b.SourceMissing {
				return nil, "", fmt.Errorf("resolve report conflicts before issuing")
			}
		}
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, "", err
	}
	// JSONB can reorder nested RawMessage keys. Canonicalize the entire value,
	// preserving number tokens, so retrieval produces the same integrity hash.
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err = decoder.Decode(&value); err != nil {
		return nil, "", err
	}
	b, err := json.Marshal(value)
	if err != nil {
		return nil, "", err
	}
	h := sha256.Sum256(b)
	return b, hex.EncodeToString(h[:]), nil
}
