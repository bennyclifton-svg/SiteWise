package delivery

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Correspondence between parties, first slice
// (docs/plans/2026-11-23-correspondence-between-parties.md, section 18):
// a notice of delay, an extension of time claim and a letter are register
// kinds, each numbered in its own run. The host records them as they
// arrive and records the result; every date and total is Go's.

// Kinds of this slice.
const (
	KindDelayNotice = "delay_notice"
	KindEOTClaim    = "eot_claim"
	KindLetter      = "letter"
)

// Prefix is a numbered kind's prefix as the screen shows it (EOT-4).
func Prefix(kind string) string {
	switch kind {
	case "rfi":
		return "RFI"
	case KindDelayNotice:
		return "NOD"
	case KindEOTClaim:
		return "EOT"
	case KindLetter:
		return "LTR"
	}
	return ""
}

// Label is a number as the screen shows it: RFI-007, EOT-4. Only an RFI is
// padded, as it has been since plan 4.
func Label(kind string, n int) string {
	p := Prefix(kind)
	if p == "" || n < 1 {
		return ""
	}
	if kind == "rfi" {
		return fmt.Sprintf("RFI-%03d", n)
	}
	return fmt.Sprintf("%s-%d", p, n)
}

// SiteWise's own default response periods, shown as "default, check your
// contract" (section 18: no AS 4902 table until the owner reads the form).
const (
	// DefaultEOTResponseDays is calendar days from receipt to assess an
	// EOT claim.
	DefaultEOTResponseDays = 28
	// A letter takes the RFI response time (working days).
)

// Directions. A received item came in from another party and the host owes
// the response; a raised one is the host's own, lodged with another party,
// who owes it. Blank reads as received, as every row before this did.
const (
	DirectionIn  = "in"
	DirectionOut = "out"
)

// Raised reports an item the host lodged with another party.
func Raised(c Content) bool {
	var d struct {
		Direction string `json:"direction"`
	}
	if c.Kind == KindEOTClaim || c.Kind == KindLetter || c.Kind == KindDelayNotice {
		_ = json.Unmarshal(c.Details, &d)
	}
	return d.Direction == DirectionOut
}

func checkDirection(d string) error {
	if d != "" && d != DirectionIn && d != DirectionOut {
		return fmt.Errorf("the direction is received or raised")
	}
	return nil
}

// EOT outcomes. A decided outcome assesses the claim; "more_information"
// is a response that keeps it open.
const (
	OutcomeGranted       = "granted"
	OutcomeGrantedInPart = "granted_in_part"
	OutcomeRejected      = "rejected"
	OutcomeMoreInfo      = "more_information"
)

// EOTResult is what a formal response decides, kept as data.
type EOTResult struct {
	Outcome     string  `json:"outcome"`
	DaysGranted *int    `json:"days_granted"`
	Reasons     string  `json:"reasons"`
	AssessedOn  *string `json:"assessed_on"`
}

// Decided reports an outcome that assesses the claim.
func (r *EOTResult) Decided() bool {
	return r != nil && (r.Outcome == OutcomeGranted || r.Outcome == OutcomeGrantedInPart || r.Outcome == OutcomeRejected)
}

// DelayNoticeDetails is a notice of delay. No response is owed; it is
// recorded, and an EOT claim may point at it.
type DelayNoticeDetails struct {
	// Direction is "in" (received, the default) or "out" (raised by the host).
	Direction  string  `json:"direction,omitempty"`
	Ref        string  `json:"ref"`
	RaisedBy   string  `json:"raised_by"`
	ReceivedOn *string `json:"received_on"`
	Cause      string  `json:"cause"`
	DelayFrom  *string `json:"delay_from"`
	MessageID  string  `json:"message_id,omitempty"`
}

// EOTClaimDetails is an extension of time claim. Its due date is
// target_date, worked out from received_on unless the PM typed it.
type EOTClaimDetails struct {
	// Direction is "in" (received, the default) or "out" (raised by the host).
	Direction   string     `json:"direction,omitempty"`
	Ref         string     `json:"ref"`
	RaisedBy    string     `json:"raised_by"`
	ReceivedOn  *string    `json:"received_on"`
	Cause       string     `json:"cause"`
	DelayFrom   *string    `json:"delay_from"`
	DelayTo     *string    `json:"delay_to"`
	DaysClaimed *int       `json:"days_claimed"`
	NoticeID    string     `json:"notice_id,omitempty"`
	MessageID   string     `json:"message_id,omitempty"`
	DueTyped    bool       `json:"due_typed,omitempty"`
	Result      *EOTResult `json:"result,omitempty"`
}

// LetterDetails is a letter owed a response.
type LetterDetails struct {
	// Direction is "in" (received, the default) or "out" (raised by the host).
	Direction  string  `json:"direction,omitempty"`
	Ref        string  `json:"ref"`
	RaisedBy   string  `json:"raised_by"`
	ReceivedOn *string `json:"received_on"`
	Body       string  `json:"body"`
	Answer     string  `json:"answer"`
	MessageID  string  `json:"message_id,omitempty"`
	DueTyped   bool    `json:"due_typed,omitempty"`
}

// EOT reads an EOT claim's details; anything else reads as none.
func EOT(c Content) EOTClaimDetails {
	var d EOTClaimDetails
	if c.Kind == KindEOTClaim {
		_ = json.Unmarshal(c.Details, &d)
	}
	return d
}

// Letter reads a letter's details; anything else reads as none.
func Letter(c Content) LetterDetails {
	var d LetterDetails
	if c.Kind == KindLetter {
		_ = json.Unmarshal(c.Details, &d)
	}
	return d
}

// DelayNotice reads a notice of delay's details; anything else reads as none.
func DelayNotice(c Content) DelayNoticeDetails {
	var d DelayNoticeDetails
	if c.Kind == KindDelayNotice {
		_ = json.Unmarshal(c.Details, &d)
	}
	return d
}

// ReceivedOn is the date a received item came in, for any kind that has one.
func ReceivedOn(c Content) *string {
	switch c.Kind {
	case KindEOTClaim:
		return EOT(c).ReceivedOn
	case KindLetter:
		return Letter(c).ReceivedOn
	case KindDelayNotice:
		return DelayNotice(c).ReceivedOn
	}
	return nil
}

func checkDates(ds ...*string) error {
	for _, d := range ds {
		if d != nil && !date(*d) {
			return fmt.Errorf("dates must use YYYY-MM-DD")
		}
	}
	return nil
}

func (d DelayNoticeDetails) check() error {
	if err := checkDirection(d.Direction); err != nil {
		return err
	}
	if !bounded(d.Ref, 100) || !bounded(d.RaisedBy, 200) || !bounded(d.Cause, 8000) || (d.MessageID != "" && !uuidLike(d.MessageID)) {
		return fmt.Errorf("invalid notice of delay details")
	}
	return checkDates(d.ReceivedOn, d.DelayFrom)
}

func (d LetterDetails) check() error {
	if err := checkDirection(d.Direction); err != nil {
		return err
	}
	if !bounded(d.Ref, 100) || !bounded(d.RaisedBy, 200) || !bounded(d.Body, 8000) || !bounded(d.Answer, 8000) || (d.MessageID != "" && !uuidLike(d.MessageID)) {
		return fmt.Errorf("invalid letter details")
	}
	return checkDates(d.ReceivedOn)
}

// check holds the claim's own rules. A result less than what was claimed
// needs a reason, as an assessed claim row does; an assessed claim needs a
// decided result.
func (d EOTClaimDetails) check(status string) error {
	if err := checkDirection(d.Direction); err != nil {
		return err
	}
	if !bounded(d.Ref, 100) || !bounded(d.RaisedBy, 200) || !bounded(d.Cause, 8000) ||
		(d.MessageID != "" && !uuidLike(d.MessageID)) || (d.NoticeID != "" && !uuidLike(d.NoticeID)) {
		return fmt.Errorf("invalid EOT claim details")
	}
	if err := checkDates(d.ReceivedOn, d.DelayFrom, d.DelayTo); err != nil {
		return err
	}
	if d.DelayFrom != nil && d.DelayTo != nil && *d.DelayTo < *d.DelayFrom {
		return fmt.Errorf("the delay ends before it starts")
	}
	if d.DaysClaimed != nil && (*d.DaysClaimed < 0 || *d.DaysClaimed > 3650) {
		return fmt.Errorf("days claimed are 0 to 3650")
	}
	if status == "assessed" && !d.Result.Decided() {
		return fmt.Errorf("an assessed claim needs its result: granted, granted in part or rejected")
	}
	r := d.Result
	if r == nil {
		return nil
	}
	if !bounded(r.Reasons, 8000) {
		return fmt.Errorf("the reasons are too long")
	}
	if err := checkDates(r.AssessedOn); err != nil {
		return err
	}
	if r.DaysGranted != nil && (*r.DaysGranted < 0 || *r.DaysGranted > 3650) {
		return fmt.Errorf("days granted are 0 to 3650")
	}
	reasons := strings.TrimSpace(r.Reasons) != ""
	switch r.Outcome {
	case OutcomeGranted:
		if r.DaysGranted == nil || *r.DaysGranted < 1 {
			return fmt.Errorf("a granted claim needs the days granted")
		}
		if d.DaysClaimed != nil && *r.DaysGranted < *d.DaysClaimed {
			return fmt.Errorf("fewer days than claimed is granted in part, with reasons")
		}
	case OutcomeGrantedInPart:
		if r.DaysGranted == nil || *r.DaysGranted < 1 {
			return fmt.Errorf("granted in part needs the days granted")
		}
		if d.DaysClaimed != nil && *r.DaysGranted >= *d.DaysClaimed {
			return fmt.Errorf("granted in part is fewer days than claimed")
		}
		if !reasons {
			return fmt.Errorf("a result less than what was claimed needs a reason")
		}
	case OutcomeRejected:
		if r.DaysGranted != nil && *r.DaysGranted != 0 {
			return fmt.Errorf("a rejected claim grants no days")
		}
		if !reasons {
			return fmt.Errorf("a rejected claim needs a reason")
		}
	case OutcomeMoreInfo:
		if r.DaysGranted != nil {
			return fmt.Errorf("asking for more information grants no days")
		}
	default:
		return fmt.Errorf("unknown result")
	}
	return nil
}

// DaysGranted is what an EOT claim's standing result grants: 0 unless the
// claim is assessed or closed with a decided result, and never for a
// withdrawn claim.
func DaysGranted(c Content) int {
	if c.Kind != KindEOTClaim || (c.Status != "assessed" && c.Status != "closed") {
		return 0
	}
	r := EOT(c).Result
	if !r.Decided() || r.DaysGranted == nil {
		return 0
	}
	return *r.DaysGranted
}

// AddDays is a date plus calendar days, or nil when the date is unknown.
func AddDays(day *string, n int) *string {
	if day == nil {
		return nil
	}
	t, err := time.Parse("2006-01-02", *day)
	if err != nil {
		return nil
	}
	out := t.AddDate(0, 0, n).Format("2006-01-02")
	return &out
}
