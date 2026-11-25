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

	"sitewise/internal/correspondence"
)

type Content struct {
	Kind         string  `json:"kind"`
	Title        string  `json:"title"`
	OwnerText    string  `json:"owner_text"`
	OwnerUserID  string  `json:"owner_user_id,omitempty"`
	BaselineDate *string `json:"baseline_date"`
	TargetDate   *string `json:"target_date"`
	ForecastDate *string `json:"forecast_date"`
	ActualDate   *string `json:"actual_date"`
	Status       string  `json:"status"`
	AsOf         *string `json:"as_of"`
	PackageID    string  `json:"package_id,omitempty"`
	WorkItemID   string  `json:"work_item_id,omitempty"`
	StageID      string  `json:"stage_id,omitempty"`
	// SystemID and DeterminantID put a register row on the building
	// (knowledge/SCHEMA.md): a top-level system, and optionally one of its
	// determinants. The store checks them against the catalogue.
	SystemID      string          `json:"system_id,omitempty"`
	DeterminantID string          `json:"determinant_id,omitempty"`
	Details       json.RawMessage `json:"details"`
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
	// Number is a numbered kind's number in its project (RFI-007, EOT-4),
	// in that kind's own run, never reused.
	Number *int `json:"number,omitempty"`
	// With, LateDays and Readings are computed at read, never stored.
	With     string `json:"with,omitempty"`
	LateDays *int   `json:"late_days,omitempty"`
	// DueBasis says how a worked-out due date was reached: "default" (SiteWise's
	// own period), "project" (the project's), or "typed". Computed at read.
	DueBasis string    `json:"due_basis,omitempty"`
	Readings []Reading `json:"readings,omitempty"`
	// Transmittal is an RFI's attached documents as the RFP transmittal's
	// rows (doc number, title, rev, date, discipline), in the order attached.
	Transmittal [][]string `json:"transmittal,omitempty"`
	// Thread is an RFI's mail: the message it went out on and every filed
	// message in that thread, oldest first. Read with the register, never
	// stored on the row.
	Thread []ThreadMessage `json:"thread,omitempty"`
	// RaisedBy is who in the team added the row: the member's email (the
	// account has no name), or "a former team member" once they are no
	// longer in the organisation. Empty when the record names no member.
	RaisedBy string `json:"raised_by_member,omitempty"`
}

// ThreadMessage is one message of an RFI's thread as the sheet's history
// shows it. Excerpt is a reply's first lines, as plain text.
type ThreadMessage struct {
	ID         string `json:"id"`
	Direction  string `json:"direction"` // "out": the PM sent it; "in": a reply
	At         string `json:"at"`
	With       string `json:"with"` // the recipients of what went out, the sender of a reply
	Number     *int   `json:"number,omitempty"`
	SendStatus string `json:"send_status,omitempty"`
	Excerpt    string `json:"excerpt,omitempty"`
	// Documents is how many documents went out, or how many attachments a
	// reply carried.
	Documents int `json:"documents"`
	// BodyDocument is the filed document a reply's text became, and Reading
	// the possible answer that document already is on this RFI, if any.
	BodyDocument string `json:"body_document,omitempty"`
	Reading      string `json:"reading,omitempty"`
}

// Reading is a proposed answer to an open register row: a filed document
// that may answer an RFI. A reading is a proposal; only the PM's edit
// makes the RFI answered.
type Reading struct {
	ID          string   `json:"id"`
	DocumentID  string   `json:"document_id"`
	Document    string   `json:"document"`
	DecidedBy   string   `json:"decided_by"`
	Band        string   `json:"band"`
	Probability *float64 `json:"probability,omitempty"`
	Basis       string   `json:"basis,omitempty"`
	FiledOn     string   `json:"filed_on,omitempty"`
	// Field is what the reading proposes: "answers" (a reply to an RFI) or
	// "evidence" (a document that may show a consent condition is met).
	Field string `json:"field,omitempty"`
}

// Numbered kinds carry a project number that belongs to that kind alone.
func Numbered(kind string) bool { return Prefix(kind) != "" }

// CheckKindChange refuses a change to or from a numbered kind: RFI-007
// must not become a risk and later an RFI again under another number.
func CheckKindChange(from, to string) error {
	if from != to && (Numbered(from) || Numbered(to)) {
		return fmt.Errorf("a numbered record keeps its kind; add a new record instead")
	}
	return nil
}

// With is who holds the row now (Procore's ball in court, in one column).
// An answered RFI is with the PM, whose move it is to pass the answer on or
// close it; otherwise it is with whoever the row names.
func With(c Content) string {
	if (c.Kind == "rfi" || c.Kind == KindLetter) && c.Status == "answered" {
		return "You"
	}
	raised := Raised(c)
	if c.Kind == KindEOTClaim && c.Status == "assessed" {
		// Assessed: the raiser's move, to accept it or follow it up. A claim
		// the host raised comes back to the host.
		if raised {
			return "You"
		}
		if by := EOT(c).RaisedBy; by != "" {
			return by
		}
	}
	if c.OwnerText == "" && !raised && (c.Kind == KindEOTClaim || c.Kind == KindLetter) && Open(c.Kind, c.Status) {
		// Received with no responder named: the host answers it. A raised
		// item is with whoever the row names, the party who owes the reply.
		return "You"
	}
	return c.OwnerText
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
	case "rfi":
		return []string{"draft", "open", "answered", "closed", "withdrawn"}
	case "notice", KindDelayNotice:
		return []string{"open", "closed", "withdrawn"}
	case KindEOTClaim:
		return []string{"open", "assessed", "closed", "withdrawn"}
	case KindLetter:
		return []string{"open", "answered", "closed", "withdrawn"}
	case "condition":
		return []string{"open", "evidence_filed", "satisfied", "not_applicable"}
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
	case "rfi":
		// Answered stays open until the PM closes it (owner decision 7).
		return status == "draft" || status == "open" || status == "answered"
	case "notice", KindDelayNotice:
		return status == "open"
	case KindEOTClaim:
		// Assessed stays open until the PM settles it, as an answered RFI does.
		return status == "open" || status == "assessed"
	case KindLetter:
		return status == "open" || status == "answered"
	case "condition":
		// Only the PM marks a condition satisfied (owner decision 6); evidence
		// filed is still open until then.
		return status == "open" || status == "evidence_filed"
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
	if !bounded(c.SystemID, 200) || !bounded(c.DeterminantID, 200) || (c.DeterminantID != "" && c.SystemID == "") {
		return fmt.Errorf("a determinant needs its system")
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

// RFIDetails is what is particular to an RFI. The subject is the title,
// who it went to is owner_text and package_id, the response date is
// target_date and the date answered is actual_date.
type RFIDetails struct {
	Question        string   `json:"question"`
	Ref             string   `json:"ref"` // the sender's own number, shown as "Ref"
	RaisedBy        string   `json:"raised_by"`
	RaisedOn        *string  `json:"raised_on"`
	Answer          string   `json:"answer"`
	AnswerDocuments []string `json:"answer_documents"`
	// DueTyped is true once the PM typed the response date, so a later
	// change to the raised date keeps it.
	DueTyped bool `json:"due_typed,omitempty"`
	// SentMessageID is the message the RFI went out on (correspondence plan:
	// Send for an RFI). Go's: set when the PM's Send queues it, never typed.
	// A reply in that message's thread answers the RFI by rule.
	SentMessageID string `json:"sent_message_id,omitempty"`
	// To, Cc and Subject are what the sheet's Send sends (register panel,
	// 20 November 2026): saved so they survive a reload. Each address is
	// checked as the send route checks it.
	To      []string `json:"to,omitempty"`
	Cc      []string `json:"cc,omitempty"`
	Subject string   `json:"subject,omitempty"`
}

// MaxRecipients is the send route's limit (store.MailMaxRecipients): an
// RFI keeps at most this many addresses in To and Cc together.
const MaxRecipients = 20

// checkAddressing refuses what the send route would refuse: a line break
// or control character, an entry that is not exactly one address, more
// than MaxRecipients in all, or a subject over the route's 300.
func (d RFIDetails) checkAddressing() error {
	if len(d.To)+len(d.Cc) > MaxRecipients {
		return fmt.Errorf("an RFI goes to at most %d addresses", MaxRecipients)
	}
	for _, v := range append(append([]string(nil), d.To...), d.Cc...) {
		if len(v) > 320 {
			return fmt.Errorf("an address is too long")
		}
		list, err := correspondence.ParseRecipients([]string{v})
		if err != nil || len(list) != 1 {
			return fmt.Errorf("%q is not one email address", v)
		}
	}
	if !bounded(d.Subject, 300) || correspondence.CleanHeader(d.Subject) != nil {
		return fmt.Errorf("the subject is one line of at most 300 characters")
	}
	return nil
}

// NoticeDetails is what is particular to a notice. Its due date is the
// date the PM typed (contract time bars are deferred).
type NoticeDetails struct {
	MessageID  string  `json:"message_id"`
	Notice     string  `json:"notice"`
	ReceivedOn *string `json:"received_on"`
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
	case "rfi":
		var d RFIDetails
		if err := decode(&d); err != nil {
			return err
		}
		if !bounded(d.Question, 8000) || !bounded(d.Answer, 8000) || !bounded(d.Ref, 100) || !bounded(d.RaisedBy, 200) || len(d.AnswerDocuments) > 20 {
			return fmt.Errorf("invalid RFI details")
		}
		if d.RaisedOn != nil && !date(*d.RaisedOn) {
			return fmt.Errorf("invalid RFI raised date")
		}
		if err := d.checkAddressing(); err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, id := range d.AnswerDocuments {
			if !uuidLike(id) || seen[id] {
				return fmt.Errorf("invalid or repeated answer document")
			}
			seen[id] = true
		}
	case "condition":
		var d ConditionDetails
		if err := decode(&d); err != nil {
			return err
		}
		if err := d.check(); err != nil {
			return err
		}
	case "notice":
		var d NoticeDetails
		if err := decode(&d); err != nil {
			return err
		}
		if !bounded(d.Notice, 200) || (d.MessageID != "" && !uuidLike(d.MessageID)) {
			return fmt.Errorf("invalid notice details")
		}
		if d.ReceivedOn != nil && !date(*d.ReceivedOn) {
			return fmt.Errorf("invalid notice received date")
		}
	case KindDelayNotice:
		var d DelayNoticeDetails
		if err := decode(&d); err != nil {
			return err
		}
		return d.check()
	case KindEOTClaim:
		var d EOTClaimDetails
		if err := decode(&d); err != nil {
			return err
		}
		return d.check(c.Status)
	case KindLetter:
		var d LetterDetails
		if err := decode(&d); err != nil {
			return err
		}
		return d.check()
	default:
		var d struct{}
		return decode(&d)
	}
	return nil
}

// RFI reads an RFI's details; anything else reads as none.
func RFI(c Content) RFIDetails {
	var d RFIDetails
	if c.Kind == "rfi" {
		_ = json.Unmarshal(c.Details, &d)
	}
	return d
}

func uuidLike(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if r != '-' {
				return false
			}
		case (r < '0' || r > '9') && (r < 'a' || r > 'f'):
			return false
		}
	}
	return true
}

func bounded(s string, n int) bool { return utf8.ValidString(s) && utf8.RuneCountInString(s) <= n }
func date(s string) bool {
	d, err := time.Parse("2006-01-02", s)
	return err == nil && d.Year() > 0 && d.Format("2006-01-02") == s
}
