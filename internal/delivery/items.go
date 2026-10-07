// Package delivery validates explicit project delivery records. Uploads and
// evidence extraction never infer progress or approval outcomes.
package delivery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

type Content struct {
	Kind         string          `json:"kind"`
	Title        string          `json:"title"`
	OwnerText    string          `json:"owner_text"`
	OwnerUserID  string          `json:"owner_user_id,omitempty"`
	BaselineDate *string         `json:"baseline_date"`
	TargetDate   *string         `json:"target_date"`
	ForecastDate *string         `json:"forecast_date"`
	ActualDate   *string         `json:"actual_date"`
	Status       string          `json:"status"`
	AsOf         *string         `json:"as_of"`
	PackageID    string          `json:"package_id,omitempty"`
	WorkItemID   string          `json:"work_item_id,omitempty"`
	StageID      string          `json:"stage_id,omitempty"`
	Details      json.RawMessage `json:"details"`
}

type Item struct {
	Content
	ID                string          `json:"id"`
	ProjectID         string          `json:"project_id"`
	Origin            string          `json:"origin"`
	ReviewStatus      string          `json:"review_status"`
	Meaning           string          `json:"meaning"`
	Provenance        json.RawMessage `json:"provenance"`
	SourceProposalKey string          `json:"source_proposal_key,omitempty"`
	RetiredAt         *time.Time      `json:"retired_at,omitempty"`
	Version           int64           `json:"version"`
}

func Statuses(kind string) []string {
	switch kind {
	case "activity", "action":
		return []string{"not_started", "in_progress", "blocked", "complete", "cancelled"}
	case "milestone":
		return []string{"planned", "achieved", "cancelled"}
	case "risk":
		return []string{"open", "mitigating", "closed"}
	case "issue":
		return []string{"open", "in_progress", "resolved", "closed"}
	case "decision":
		return []string{"open", "decided", "superseded"}
	case "approval":
		return []string{"not_submitted", "submitted", "approved", "rejected", "withdrawn"}
	default:
		return nil
	}
}

func DefaultStatus(kind string) string {
	s := Statuses(kind)
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func Open(kind, status string) bool {
	switch kind {
	case "activity", "action":
		return status != "complete" && status != "cancelled"
	case "milestone":
		return status != "achieved" && status != "cancelled"
	case "risk":
		return status != "closed"
	case "issue":
		return status != "resolved" && status != "closed"
	case "decision":
		return status != "decided" && status != "superseded"
	case "approval":
		return status != "approved" && status != "rejected" && status != "withdrawn"
	default:
		return true
	}
}

func Validate(c Content) error {
	if !slices.Contains(Statuses(c.Kind), c.Status) {
		return fmt.Errorf("invalid status for delivery kind")
	}
	if strings.TrimSpace(c.Title) == "" || !bounded(c.Title, 200) || !bounded(c.OwnerText, 200) {
		return fmt.Errorf("invalid title or owner text")
	}
	if c.StageID != "" && c.PackageID == "" {
		return fmt.Errorf("stage requires its package")
	}
	for _, d := range []*string{c.BaselineDate, c.TargetDate, c.ForecastDate, c.ActualDate, c.AsOf} {
		if d != nil && !date(*d) {
			return fmt.Errorf("dates must use YYYY-MM-DD")
		}
	}
	return validateDetails(c)
}

type RiskDetails struct {
	Likelihood  string `json:"likelihood"`
	Consequence string `json:"consequence"`
	UCRecordID  string `json:"uc_record_id,omitempty"`
}
type ApprovalDetails struct {
	Authority    string  `json:"authority"`
	Reference    string  `json:"reference"`
	SubmittedOn  *string `json:"submitted_on"`
	DeterminedOn *string `json:"determined_on"`
}
type DecisionDetails struct {
	Options []string `json:"options"`
	Chosen  string   `json:"chosen"`
}

func validateDetails(c Content) error {
	if len(c.Details) > 32000 {
		return fmt.Errorf("delivery details too large")
	}
	decode := func(v any) error {
		if len(c.Details) == 0 || bytes.Equal(bytes.TrimSpace(c.Details), []byte("null")) {
			return fmt.Errorf("details must be an object")
		}
		d := json.NewDecoder(bytes.NewReader(c.Details))
		d.DisallowUnknownFields()
		if err := d.Decode(v); err != nil {
			return fmt.Errorf("invalid details: %w", err)
		}
		if err := d.Decode(new(any)); err != io.EOF {
			return fmt.Errorf("expected one details object")
		}
		return nil
	}
	switch c.Kind {
	case "risk":
		var d RiskDetails
		if err := decode(&d); err != nil {
			return err
		}
		if !bounded(d.Likelihood, 200) || !bounded(d.Consequence, 2000) || !bounded(d.UCRecordID, 200) {
			return fmt.Errorf("invalid risk details")
		}
	case "approval":
		var d ApprovalDetails
		if err := decode(&d); err != nil {
			return err
		}
		if !bounded(d.Authority, 200) || !bounded(d.Reference, 200) {
			return fmt.Errorf("invalid approval details")
		}
		for _, v := range []*string{d.SubmittedOn, d.DeterminedOn} {
			if v != nil && !date(*v) {
				return fmt.Errorf("invalid approval date")
			}
		}
		if d.SubmittedOn != nil && d.DeterminedOn != nil && *d.DeterminedOn < *d.SubmittedOn {
			return fmt.Errorf("approval determination precedes submission")
		}
	case "decision":
		var d DecisionDetails
		if err := decode(&d); err != nil {
			return err
		}
		if len(d.Options) > 100 || !bounded(d.Chosen, 2000) {
			return fmt.Errorf("invalid decision options")
		}
		seen := map[string]bool{}
		for _, option := range d.Options {
			if strings.TrimSpace(option) == "" || !bounded(option, 2000) || seen[option] {
				return fmt.Errorf("invalid or duplicate decision option")
			}
			seen[option] = true
		}
		if d.Chosen != "" && !seen[d.Chosen] {
			return fmt.Errorf("chosen decision is not an option")
		}
		if c.Status == "decided" && d.Chosen == "" {
			return fmt.Errorf("decided record requires a chosen option")
		}
	default:
		var d struct{}
		return decode(&d)
	}
	return nil
}

func bounded(s string, n int) bool { return utf8.ValidString(s) && utf8.RuneCountInString(s) <= n }
func date(s string) bool {
	d, err := time.Parse("2006-01-02", s)
	return err == nil && d.Year() > 0 && d.Format("2006-01-02") == s
}
