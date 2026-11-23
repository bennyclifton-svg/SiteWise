package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"sitewise/internal/delivery"
	"sitewise/internal/invoices"
	"sitewise/internal/knowledge"
	"sitewise/internal/program"
)

// The register (docs/plans/2026-11-17-rfis-review-comments.md, Task 2):
// RFIs, notices and the older delivery kinds in one table. Numbers, due
// dates, days late, who holds a row and the also-open line are Go's,
// computed here; nothing on these paths calls Jev or the model.

// RegisterQuery narrows a register read. Both are optional.
type RegisterQuery struct {
	Kind   string
	System string
}

// RegisterView is the register as the screen and chat read it.
type RegisterView struct {
	Items []delivery.Item `json:"items"`
	// Kinds lists only the kinds that have rows, so the Kind filter never
	// offers one the project does not have.
	Kinds []RegisterKind `json:"kinds"`
	// ResponseDays is the RFI response time; Typical is true until the PM
	// sets the project's own (shown blue).
	ResponseDays    int  `json:"response_days"`
	ResponseTypical bool `json:"response_typical"`
	// EOTDays is the calendar days to assess an EOT claim; Typical until
	// the PM sets the project's own (SiteWise's default, check the contract).
	EOTDays    int  `json:"eot_days"`
	EOTTypical bool `json:"eot_typical"`
	// Completion is the agreed completion date, the days granted on EOT
	// claims and the date after them, computed here.
	Completion Completion `json:"completion"`
	Calendar   string     `json:"calendar"` // "program", or "weekdays" with no program
	Today      string     `json:"today"`
	// Systems are the top-level systems in building order, for the System filter.
	Systems  []RegisterSystem `json:"systems"`
	AlsoOpen *AlsoOpen        `json:"also_open,omitempty"`
	// Consents are the project's consent reads (Read conditions), newest
	// first, and Consent the Profile's consent number, date and lapse date
	// for the conditions' header.
	Consents []ConsentRead `json:"consents"`
	Consent  ConsentFacts  `json:"consent"`
}

// ConsentFacts are the Profile's consent facts (knowledge/profile/project_facts.yaml).
type ConsentFacts struct {
	Number    string `json:"number,omitempty"`
	Date      string `json:"date,omitempty"`
	LapseDate string `json:"lapse_date,omitempty"`
}

type RegisterKind struct {
	Kind    string `json:"kind"`
	Rows    int    `json:"rows"`
	Open    int    `json:"open"`
	ToCheck int    `json:"to_check"`
}

type RegisterSystem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// AlsoOpen is what else is unsettled on a system: specification rows with
// an unknown inclusion or an undecided selection, and the exclusions of the
// tenders for trades that build under it. Main Works exclusions are not
// sorted by system, so they are one count of their own.
type AlsoOpen struct {
	System              string   `json:"system"`
	Label               string   `json:"label"`
	SpecUnknowns        int      `json:"spec_unknowns"`
	Exclusions          int      `json:"exclusions"`
	ExclusionPackages   []string `json:"exclusion_packages"`
	MainWorksExclusions int      `json:"main_works_exclusions"`
	MainWorksPackage    string   `json:"main_works_package,omitempty"`
}

// listedSQL is what the register lists: every kind but Program's own
// activities and milestones. An accepted hold point is a milestone that
// came from the implications table, and it stays.
const listedSQL = `(kind NOT IN ('activity','milestone') OR source_proposal_key IS NOT NULL)`

func sydneyToday() string { return time.Now().In(invoices.Sydney).Format("2006-01-02") }

// ReadDelivery is the register's rows with With, days late and possible
// answers, as of today in Sydney.
func (s *Store) ReadDelivery(ctx context.Context, org, project string) ([]delivery.Item, error) {
	v, err := s.ReadRegister(ctx, org, project, RegisterQuery{}, sydneyToday())
	return v.Items, err
}

// ReadRegister reads the register in one read-only snapshot.
func (s *Store) ReadRegister(ctx context.Context, org, project string, q RegisterQuery, today string) (RegisterView, error) {
	view := RegisterView{Items: []delivery.Item{}, Kinds: []RegisterKind{}, Systems: []RegisterSystem{}, Consents: []ConsentRead{}, Today: today}
	cat := s.workCatalog()
	if q.System != "" && !topSystem(cat, q.System) {
		return view, fmt.Errorf("%w: unknown system", ErrInvalidDelivery)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return view, err
	}
	defer tx.Rollback(ctx)
	if view, err = s.readRegisterTx(ctx, tx, org, project, q, view); err != nil {
		return view, err
	}
	if err := tx.Commit(ctx); err != nil {
		return view, err
	}
	view.Consents, err = s.ConsentReads(ctx, org, project)
	return view, err
}

// readRegisterTx is the register on the caller's snapshot, so the project
// report lists exactly the register's rows, with With, days late and
// possible answers computed the same way.
func (s *Store) readRegisterTx(ctx context.Context, tx pgx.Tx, org, project string, q RegisterQuery, view RegisterView) (RegisterView, error) {
	today, cat := view.Today, s.workCatalog()
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&exists); err != nil {
		return view, err
	}
	if !exists {
		return view, ErrNotFound
	}
	rows, err := tx.Query(ctx, `SELECT to_jsonb(i) FROM project_delivery_items i WHERE org_id=$1::uuid AND project_id=$2::uuid AND retired_at IS NULL AND `+listedSQL+` ORDER BY number NULLS LAST,target_date NULLS LAST,title,id`, org, project)
	if err != nil {
		return view, err
	}
	var all []delivery.Item
	for rows.Next() {
		var raw []byte
		var item delivery.Item
		if err := rows.Scan(&raw); err != nil {
			rows.Close()
			return view, err
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			rows.Close()
			return view, err
		}
		all = append(all, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return view, err
	}
	readings, err := proposedReadings(ctx, tx, org, project)
	if err != nil {
		return view, err
	}
	terms, err := deliveryTerms(ctx, tx, org, project)
	if err != nil {
		return view, err
	}
	periods := terms.Resolve()
	cal, hasProgram, err := projectCalendar(ctx, tx, org, project)
	if err != nil {
		return view, err
	}
	view.ResponseDays, view.ResponseTypical, view.Calendar = periods.RFIDays, periods.RFITypical, "weekdays"
	view.EOTDays, view.EOTTypical = periods.EOTDays, periods.EOTTypical
	if hasProgram {
		view.Calendar = "program"
	}
	attached, err := rfiTransmittals(ctx, tx, org, project, all)
	if err != nil {
		return view, err
	}
	threads, err := rfiThreads(ctx, tx, org, project, all, readings)
	if err != nil {
		return view, err
	}
	raisers, err := rowRaisers(ctx, tx, org, all)
	if err != nil {
		return view, err
	}
	counts := map[string]*RegisterKind{}
	var order []string
	for i := range all {
		it := &all[i]
		it.Transmittal = attached[it.ID]
		it.Thread = threads[it.ID]
		it.RaisedBy = raisers[it.ID]
		it.With = delivery.With(it.Content)
		it.LateDays = lateDays(cal, *it, today)
		it.DueBasis = dueBasis(it.Content, periods)
		it.Readings = readings[it.ID]
		k := counts[it.Kind]
		if k == nil {
			k = &RegisterKind{Kind: it.Kind}
			counts[it.Kind] = k
			order = append(order, it.Kind)
		}
		k.Rows++
		if delivery.Open(it.Kind, it.Status) {
			k.Open++
		}
		if delivery.ToCheck(*it) {
			k.ToCheck++
		}
		if (q.Kind == "" || q.Kind == it.Kind) && (q.System == "" || q.System == it.SystemID) {
			view.Items = append(view.Items, *it)
		}
	}
	view.Completion = completionOf(all, terms.AgreedCompletion)
	for _, kind := range kindOrder(order) {
		view.Kinds = append(view.Kinds, *counts[kind])
	}
	if cat != nil {
		for _, sys := range cat.BuildingOrderSystems() {
			view.Systems = append(view.Systems, RegisterSystem{ID: sys.ID, Label: sys.Label})
		}
	}
	if q.System != "" {
		also, err := s.alsoOpen(ctx, tx, org, project, q.System)
		if err != nil {
			return view, err
		}
		view.AlsoOpen = &also
	}
	if view.Consent, err = consentFacts(ctx, tx, org, project); err != nil {
		return view, err
	}
	return view, nil
}

// consentFacts reads the Profile's consent number, date and lapse date as
// it holds them (a value the PM or a read document stated).
func consentFacts(ctx context.Context, tx pgx.Tx, org, project string) (ConsentFacts, error) {
	var f ConsentFacts
	rows, err := tx.Query(ctx, `SELECT r.key, COALESCE(r.value,'') FROM profile_rows r JOIN project_parts p ON p.org_id=r.org_id AND p.site_id=r.site_id AND p.id=r.part_id
 WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND p.kind='whole' AND r.key IN ('fact.consent_number','fact.consent_date','fact.consent_lapse_date')
 AND r.value_state NOT IN ('unknown','cleared') AND r.review_status<>'superseded'`, org, project)
	if err != nil {
		return f, err
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return f, err
		}
		switch key {
		case "fact.consent_number":
			f.Number = value
		case "fact.consent_date":
			f.Date = value
		case "fact.consent_lapse_date":
			f.LapseDate = value
		}
	}
	return f, rows.Err()
}

// kindOrder puts RFIs, notices and consent conditions first, then the older kinds as they came.
func kindOrder(kinds []string) []string {
	rank := map[string]int{"rfi": 0, "eot_claim": 1, "delay_notice": 2, "letter": 3, "notice": 4, "condition": 5, "action": 6, "risk": 7, "issue": 8, "decision": 9, "approval": 10, "milestone": 11}
	out := append([]string(nil), kinds...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && rank[out[j]] < rank[out[j-1]]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// owed reports a row that is still waiting on its response: a Draft or
// Open RFI, an Open EOT claim, an Open letter.
func owed(it delivery.Item) bool {
	switch it.Kind {
	case "rfi":
		return it.Status == "draft" || it.Status == "open"
	case delivery.KindEOTClaim, delivery.KindLetter:
		return it.Status == "open"
	}
	return false
}

// dueBasis says how a row's due date was reached, for the screen's "28
// days, the default". Empty when the row has no worked-out due date.
func dueBasis(c delivery.Content, p Periods) string {
	if c.TargetDate == nil {
		return ""
	}
	typed, typical := false, p.RFITypical
	switch c.Kind {
	case "rfi":
		typed = delivery.RFI(c).DueTyped
	case delivery.KindLetter:
		typed = delivery.Letter(c).DueTyped
	case delivery.KindEOTClaim:
		typed, typical = delivery.EOT(c).DueTyped, p.EOTTypical
	default:
		return ""
	}
	switch {
	case typed:
		return "typed"
	case typical:
		return "default"
	}
	return "project"
}

// lateDays is how far a row still owed a response is past its due date:
// working days on the project's calendar for an RFI or a letter, calendar
// days for an EOT claim (its period is in calendar days). Nil when it is
// not late.
func lateDays(cal program.Calendar, it delivery.Item, today string) *int {
	if !owed(it) || it.TargetDate == nil || *it.TargetDate >= today {
		return nil
	}
	if it.Kind == delivery.KindEOTClaim {
		due, err1 := time.Parse("2006-01-02", *it.TargetDate)
		now, err2 := time.Parse("2006-01-02", today)
		if err1 != nil || err2 != nil {
			return nil
		}
		n := int(now.Sub(due).Hours() / 24)
		return &n
	}
	c := program.NewCalendar(program.Calendar{Start: *it.TargetDate, WorkingWeek: cal.WorkingWeek, Days: cal.Days})
	n, ok := c.IndexOnOrBefore(today)
	if !ok {
		return nil
	}
	if first, _ := c.IndexOnOrBefore(*it.TargetDate); first != 0 {
		n++ // the due date was not a working day, so index 0 is already late
	}
	if n < 1 {
		return nil
	}
	return &n
}

// projectCalendar is the Program's working week and days, or Monday to
// Friday when the project has no program.
func projectCalendar(ctx context.Context, tx pgx.Tx, org, project string) (program.Calendar, bool, error) {
	var week int
	var days []byte
	err := tx.QueryRow(ctx, `SELECT p.working_week, COALESCE((SELECT jsonb_agg(jsonb_build_object('day',to_char(d.day,'YYYY-MM-DD'),'kind',d.kind)) FROM program_calendar_days d WHERE d.org_id=p.org_id AND d.program_id=p.id),'[]')
 FROM programs p WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid`, org, project).Scan(&week, &days)
	if errors.Is(err, pgx.ErrNoRows) {
		return program.Calendar{}, false, nil
	}
	if err != nil {
		return program.Calendar{}, false, err
	}
	cal := program.Calendar{WorkingWeek: week}
	if err := json.Unmarshal(days, &cal.Days); err != nil {
		return cal, false, err
	}
	return cal, true, nil
}

// responseDays is the project's RFI response time, or the typical five.
func responseDays(ctx context.Context, tx pgx.Tx, org, project string) (int, bool, error) {
	t, err := deliveryTerms(ctx, tx, org, project)
	p := t.Resolve()
	return p.RFIDays, p.RFITypical, err
}

// proposedReadings are the possible answers the screen shows: rule matches
// and amber Jev picks, by row. Blank and grey readings are stored for
// calibration and never shown (owner decision 4).
func proposedReadings(ctx context.Context, tx pgx.Tx, org, project string) (map[string][]delivery.Reading, error) {
	rows, err := tx.Query(ctx, `SELECT r.item_id::text, r.id::text, r.document_id::text,
 COALESCE((SELECT t.value FROM decisions t WHERE t.org_id=d.org_id AND t.document_id=d.id AND t.field='title' LIMIT 1), d.filename),
 r.decided_by, r.band, r.probability, r.basis, to_char(d.created_at AT TIME ZONE 'Australia/Sydney','YYYY-MM-DD'), r.field
 FROM delivery_readings r JOIN documents d ON d.org_id=r.org_id AND d.id=r.document_id
 WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.status='proposed' AND (r.decided_by='rule' OR r.band='amber')
 ORDER BY r.created_at, r.id`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]delivery.Reading{}
	for rows.Next() {
		var item string
		var r delivery.Reading
		if err := rows.Scan(&item, &r.ID, &r.DocumentID, &r.Document, &r.DecidedBy, &r.Band, &r.Probability, &r.Basis, &r.FiledOn, &r.Field); err != nil {
			return nil, err
		}
		out[item] = append(out[item], r)
	}
	return out, rows.Err()
}

// topSystem reports a top-level system of the catalogue. The register
// files rows against top-level systems only (child systems are out of
// scope).
func topSystem(cat *knowledge.Catalog, id string) bool {
	if cat == nil {
		return false
	}
	sys, ok := cat.System(id)
	return ok && sys.Parent == "" && sys.Status != "deprecated"
}

// topOf is the top-level system above id, or "".
func topOf(cat *knowledge.Catalog, id string) string {
	for i := 0; cat != nil && id != "" && i < 20; i++ {
		sys, ok := cat.System(id)
		if !ok {
			return ""
		}
		if sys.Parent == "" {
			return sys.ID
		}
		id = sys.Parent
	}
	return ""
}

// registerFill is what Go fills on a new or edited row: an RFI's number,
// its due date from the raised date, With from the appointed firm, and the
// system from the work item or the trade package. It also checks the
// building link and the documents against this org, project and catalogue.
func (s *Store) registerFill(ctx context.Context, tx pgx.Tx, org, project string, c *delivery.Content, before *delivery.Content, dueEdited bool) error {
	cat := s.workCatalog()
	if c.SystemID != "" && !topSystem(cat, c.SystemID) {
		return fmt.Errorf("%w: unknown system", ErrInvalidDelivery)
	}
	if c.DeterminantID != "" {
		if cat == nil {
			return fmt.Errorf("%w: unknown determinant", ErrInvalidDelivery)
		}
		if _, ok := cat.Determinant(c.DeterminantID); !ok {
			return fmt.Errorf("%w: unknown determinant", ErrInvalidDelivery)
		}
	}
	if c.SystemID == "" && (before == nil || c.WorkItemID != before.WorkItemID || c.PackageID != before.PackageID) {
		sys, err := s.rowSystem(ctx, tx, org, project, *c)
		if err != nil {
			return err
		}
		c.SystemID = sys
	}
	if c.OwnerText == "" && c.PackageID != "" && (before == nil || c.PackageID != before.PackageID) {
		var firm string
		err := tx.QueryRow(ctx, `SELECT f.name FROM submissions s JOIN firms f ON f.org_id=s.org_id AND f.id=s.firm_id
 WHERE s.org_id=$1::uuid AND s.project_id=$2::uuid AND s.package_id=$3::uuid AND s.status='awarded' ORDER BY f.name LIMIT 1`, org, project, c.PackageID).Scan(&firm)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if len([]rune(firm)) > 200 {
			firm = string([]rune(firm)[:200])
		}
		c.OwnerText = firm
	}
	switch c.Kind {
	case "condition":
		d := delivery.Condition(*c)
		if before != nil && before.Kind == "condition" {
			// The number, heading and words are the instrument's; an edit
			// changes who acts, the gate and evidence needed only.
			d = delivery.MergeConditionEdit(delivery.Condition(*before), d)
		} else {
			if strings.TrimSpace(d.Number) == "" {
				return fmt.Errorf("%w: a condition needs its number as printed on the consent", ErrInvalidDelivery)
			}
			// Evidence is linked and an RFP line is written by their own
			// actions, never typed into a new row.
			d.Evidence, d.RFP = nil, nil
			if d.Source != nil {
				if err := projectDocument(ctx, tx, org, project, d.Source.DocumentID); err != nil {
					return err
				}
			}
		}
		raw, err := json.Marshal(d)
		if err != nil {
			return err
		}
		c.Details = raw
	case "notice":
		var d delivery.NoticeDetails
		_ = json.Unmarshal(c.Details, &d)
		if d.MessageID != "" {
			// A notice from a message (correspondence plan, the chip's Add to
			// register): the message must be a filed message of this project.
			// Its pick is then used, in this transaction, so the chip goes.
			if err := useMessagePick(ctx, tx, org, project, d.MessageID); err != nil {
				return err
			}
		}
	case delivery.KindEOTClaim:
		d := delivery.EOT(*c)
		var was *delivery.EOTClaimDetails
		if before != nil && before.Kind == c.Kind {
			b := delivery.EOT(*before)
			was = &b
		}
		if err := s.fillReceived(ctx, tx, org, project, c, dueEdited, &d.DueTyped, d.ReceivedOn, was != nil && was.DueTyped, was == nil || !sameDate(was.ReceivedOn, d.ReceivedOn)); err != nil {
			return err
		}
		if d.MessageID != "" && (was == nil || was.MessageID != d.MessageID) {
			if err := useMessagePick(ctx, tx, org, project, d.MessageID); err != nil {
				return err
			}
		}
		if d.NoticeID != "" {
			var ok bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_delivery_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND kind='delay_notice')`, org, project, d.NoticeID).Scan(&ok); err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("%w: the linked notice of delay is not on this project", ErrInvalidDelivery)
			}
		}
		// A decided result keeps the day it was recorded when none is given.
		if d.Result.Decided() && d.Result.AssessedOn == nil {
			today := sydneyToday()
			d.Result.AssessedOn = &today
		}
		raw, err := json.Marshal(d)
		if err != nil {
			return err
		}
		c.Details = raw
	case delivery.KindLetter:
		d := delivery.Letter(*c)
		var was *delivery.LetterDetails
		if before != nil && before.Kind == c.Kind {
			b := delivery.Letter(*before)
			was = &b
		}
		if err := s.fillReceived(ctx, tx, org, project, c, dueEdited, &d.DueTyped, d.ReceivedOn, was != nil && was.DueTyped, was == nil || !sameDate(was.ReceivedOn, d.ReceivedOn)); err != nil {
			return err
		}
		if d.MessageID != "" && (was == nil || was.MessageID != d.MessageID) {
			if err := useMessagePick(ctx, tx, org, project, d.MessageID); err != nil {
				return err
			}
		}
		raw, err := json.Marshal(d)
		if err != nil {
			return err
		}
		c.Details = raw
	case delivery.KindDelayNotice:
		d := delivery.DelayNotice(*c)
		if d.MessageID != "" && (before == nil || delivery.DelayNotice(*before).MessageID != d.MessageID) {
			if err := useMessagePick(ctx, tx, org, project, d.MessageID); err != nil {
				return err
			}
		}
	case "rfi":
		d := delivery.RFI(*c)
		for _, id := range d.AnswerDocuments {
			if err := projectDocument(ctx, tx, org, project, id); err != nil {
				return err
			}
		}
		// The message an RFI went out on is Go's (LinkRFIMessage); an edit
		// keeps it as it was.
		d.SentMessageID = ""
		if before != nil && before.Kind == "rfi" {
			d.SentMessageID = delivery.RFI(*before).SentMessageID
		}
		if dueEdited {
			d.DueTyped = c.TargetDate != nil
		} else if before != nil && delivery.RFI(*before).DueTyped {
			// The screen sends the details it shows; the typed mark is Go's.
			d.DueTyped = true
		}
		raisedMoved := before == nil || !sameDate(delivery.RFI(*before).RaisedOn, d.RaisedOn)
		if !d.DueTyped && (raisedMoved || dueEdited) {
			days, _, err := responseDays(ctx, tx, org, project)
			if err != nil {
				return err
			}
			cal, _, err := projectCalendar(ctx, tx, org, project)
			if err != nil {
				return err
			}
			c.TargetDate = delivery.DueDate(cal, d.RaisedOn, days)
		}
		raw, err := json.Marshal(d)
		if err != nil {
			return err
		}
		c.Details = raw
	}
	return nil
}

// fillReceived works out a received item's due date from its received
// date and the project's period unless the PM typed it: the rule an RFI's
// raised date follows. An EOT claim counts calendar days; a letter counts
// working days on the Program's calendar.
func (s *Store) fillReceived(ctx context.Context, tx pgx.Tx, org, project string, c *delivery.Content, dueEdited bool, dueTyped *bool, received *string, wasTyped, moved bool) error {
	if dueEdited {
		*dueTyped = c.TargetDate != nil
	} else if wasTyped {
		*dueTyped = true // the screen sends the details it shows; the mark is Go's
	}
	if *dueTyped || !(moved || dueEdited) {
		return nil
	}
	t, err := deliveryTerms(ctx, tx, org, project)
	if err != nil {
		return err
	}
	p := t.Resolve()
	if c.Kind == delivery.KindEOTClaim {
		c.TargetDate = delivery.AddDays(received, p.EOTDays)
		return nil
	}
	cal, _, err := projectCalendar(ctx, tx, org, project)
	if err != nil {
		return err
	}
	c.TargetDate = delivery.DueDate(cal, received, p.RFIDays)
	return nil
}

func sameDate(a, b *string) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

// rowSystem is the system a row's work item is on, or the one top-level
// system a trade package builds under; "" when neither says.
func (s *Store) rowSystem(ctx context.Context, tx pgx.Tx, org, project string, c delivery.Content) (string, error) {
	cat := s.workCatalog()
	if cat == nil {
		return "", nil
	}
	if c.WorkItemID != "" {
		var sys string
		err := tx.QueryRow(ctx, `SELECT system_id FROM work_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, c.WorkItemID).Scan(&sys)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
		if top := topOf(cat, sys); top != "" {
			return top, nil
		}
	}
	if c.PackageID != "" {
		var discipline string
		err := tx.QueryRow(ctx, `SELECT COALESCE(discipline_id,'') FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, c.PackageID).Scan(&discipline)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
		if t, ok := cat.Trade(discipline); ok {
			tops := map[string]bool{}
			for _, b := range t.Builds {
				if top := topOf(cat, b); top != "" {
					tops[top] = true
				}
			}
			if len(tops) == 1 {
				for top := range tops {
					return top, nil
				}
			}
		}
	}
	return "", nil
}

// projectDocument checks that a document id is a document of this project.
func projectDocument(ctx context.Context, tx pgx.Tx, org, project, id string) error {
	var uuid pgtype.UUID
	if uuid.Scan(id) != nil {
		return ErrInvalidDelivery
	}
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid)`, org, project, id).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

// nextNumber is the project's next RFI number, under the project lock the
// caller holds. Retired rows keep their numbers, so a number is never
// reused; the unique index backs the lock.
func nextNumber(ctx context.Context, tx pgx.Tx, org, project, kind string) (*int, error) {
	if !delivery.Numbered(kind) {
		return nil, nil
	}
	var n int
	err := tx.QueryRow(ctx, `SELECT COALESCE(max(number),0)+1 FROM project_delivery_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND kind=$3`, org, project, kind).Scan(&n)
	return &n, err
}

// SetDeliverySettings sets the project's RFI response time. Open RFIs whose
// due date the PM did not type keep the date they have: the new time
// applies to RFIs raised or re-dated after it.
func (s *Store) SetDeliverySettings(ctx context.Context, org, project, actor string, days int) error {
	return s.SetDeliveryTerms(ctx, org, project, actor, TermsPatch{RFIDays: Optional[int]{Present: true, Value: &days}})
}

// alsoOpen counts what else is unsettled on a system.
func (s *Store) alsoOpen(ctx context.Context, tx pgx.Tx, org, project, system string) (AlsoOpen, error) {
	cat := s.workCatalog()
	out := AlsoOpen{System: system, ExclusionPackages: []string{}}
	if sys, ok := cat.System(system); ok {
		out.Label = sys.Label
	}
	// Specification rows under the system with an unknown inclusion or an
	// undecided selection. Rows with no application yet read as both.
	err := tx.QueryRow(ctx, `SELECT count(*) FROM specification_types t
 LEFT JOIN specification_applications a ON a.org_id=t.org_id AND a.type_id=t.id
 WHERE t.org_id=$1::uuid AND t.project_id=$2::uuid AND COALESCE(t.content->>'status','')<>'retired' AND t.content->>'system_id' = ANY($3::text[])
 AND (a.id IS NULL OR COALESCE(a.content->>'inclusion','unknown')='unknown' OR COALESCE(a.content->>'selection','undecided')='undecided')`, org, project, systemsUnder(cat, system)).Scan(&out.SpecUnknowns)
	if err != nil {
		return out, err
	}
	rows, err := tx.Query(ctx, `SELECT p.id::text, COALESCE(p.discipline_id,''), COALESCE(sum(jsonb_array_length(s.tender_exclusions)),0)::int
 FROM packages p JOIN submissions s ON s.org_id=p.org_id AND s.project_id=p.project_id AND s.package_id=p.id
 WHERE p.org_id=$1::uuid AND p.project_id=$2::uuid AND p.retired_at IS NULL AND jsonb_array_length(s.tender_exclusions)>0
 GROUP BY p.id, p.discipline_id, p.title ORDER BY p.title, p.id`, org, project)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, discipline string
		var n int
		if err := rows.Scan(&id, &discipline, &n); err != nil {
			return out, err
		}
		if discipline == "trade.main_works" {
			out.MainWorksExclusions += n
			out.MainWorksPackage = id
			continue
		}
		t, ok := cat.Trade(discipline)
		if !ok {
			continue
		}
		for _, b := range t.Builds {
			if topOf(cat, b) == system {
				out.Exclusions += n
				out.ExclusionPackages = append(out.ExclusionPackages, id)
				break
			}
		}
	}
	return out, rows.Err()
}

// systemsUnder is a top-level system and every system below it.
func systemsUnder(cat *knowledge.Catalog, top string) []string {
	out := []string{top}
	queue := []string{top}
	for len(queue) > 0 && cat != nil {
		next := queue[0]
		queue = queue[1:]
		for _, child := range cat.Children(next) {
			out = append(out, child.ID)
			queue = append(queue, child.ID)
		}
	}
	return out
}
