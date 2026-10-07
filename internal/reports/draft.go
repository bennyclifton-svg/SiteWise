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
type Table struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

type Section struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Essential bool    `json:"essential"`
	Blocks    []Block `json:"blocks"`
}

type Edit struct {
	TargetID          string    `json:"target_id"`
	Text              string    `json:"text"`
	BaseContentSHA256 string    `json:"base_content_sha256"`
	UserID            string    `json:"user_id"`
	Version           int64     `json:"version"`
	UpdatedAt         time.Time `json:"updated_at"`
}
