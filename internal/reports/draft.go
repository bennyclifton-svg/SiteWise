// Package reports assembles saved records deterministically. It does not call
// Jev or invent report prose, appointments or verification.
package reports

import (
	"encoding/json"
	"time"
)

type Block struct {
	Table         *Table          `json:"table,omitempty"`
	GeneratedText *string         `json:"generated_text,omitempty"`
	ID            string          `json:"id"`
	Label         string          `json:"label"`
	Text          string          `json:"text"`
	Origin        string          `json:"origin"`
	ReviewStatus  string          `json:"review_status"`
	Meaning       string          `json:"meaning"`
	Basis         json.RawMessage `json:"basis"`
	Provisional   bool            `json:"provisional"`
	ContentSHA256 string          `json:"content_sha256"`
	Edited        bool            `json:"edited"`
	Conflict      bool            `json:"conflict"`
	SourceMissing bool            `json:"source_missing"`
	EditUserID    string          `json:"edit_user_id,omitempty"`
	EditUpdatedAt time.Time       `json:"edit_updated_at,omitzero"`
	CitationIDs   []string        `json:"citation_ids,omitempty"`
}

// Table is saved source data, displayed separately from editable prose.
// Display "document" marks a table that is the document's own content (an
// RFP schedule) rather than a comparison of saved sources.
type Table struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
	Display string     `json:"display,omitempty"`
	// ID names a project report table the PM can edit row by row; Meta
	// says what each row is (RowFacts) and Hidden how many rows the PM
	// hid. All omitted elsewhere, so other tables hash as before.
	ID        string     `json:"id,omitempty"`
	Meta      []RowFacts `json:"meta,omitempty"`
	Hidden    int        `json:"hidden,omitempty"`
	HiddenIDs []string   `json:"hidden_ids,omitempty"`
	// HiddenKeys are the hidden rows' keys, in HiddenIDs' order.
	HiddenKeys []string `json:"hidden_keys,omitempty"`
}

type Section struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Essential bool   `json:"essential"`
	// Internal sections are review lists for the project manager; never issued.
	Internal bool    `json:"internal,omitempty"`
	Blocks   []Block `json:"blocks"`
}

type Edit struct {
	TargetID          string    `json:"target_id"`
	Text              string    `json:"text"`
	BaseContentSHA256 string    `json:"base_content_sha256"`
	UserID            string    `json:"user_id"`
	Version           int64     `json:"version"`
	UpdatedAt         time.Time `json:"updated_at"`
}
