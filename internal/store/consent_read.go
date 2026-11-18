package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"sitewise/internal/delivery"
)

// The consent read (docs/plans/2026-11-17-consent-conditions.md, Tasks 4
// and 5). A notice of determination is read into proposed condition rows by
// a background job: Go split it, Jev picked who acts, the system and whether
// evidence is needed. UpsertConditions writes the rows in one transaction.

// JobKindConsent is the background read of a notice's conditions.
const JobKindConsent = "consent"

// ErrConsentBusy is a read already queued or running for the document.
var ErrConsentBusy = errors.New("conditions are already being read from this document")

// ConsentNotice is what the split found in one notice.
type ConsentNotice struct {
	DocumentID   string   `json:"document_id"`
	ConsentRef   string   `json:"consent_ref"`
	ModRef       string   `json:"mod_ref,omitempty"`
	Modification bool     `json:"modification"`
	Determined   string   `json:"determined,omitempty"`
	Conditions   int      `json:"conditions"`
	Gaps         []string `json:"gaps"`
}

// ConsentRow is one condition to write: the instrument's words and Go's
// and Jev's picks (blank when a pick is below its floor).
type ConsentRow struct {
	Number, Heading, Text, StageHeading, Stage string
	Page                                       int
	Deferred                                   bool
	Marker, MarkerBy                           string
	Discipline, OwnerText, SystemID            string
	EvidenceNeeded                             *bool
	// Reading is the consent.Reading as stored in provenance, and
	// Fingerprint its cache key.
	Reading     json.RawMessage
	Fingerprint string
}

// consentKey is the read's key for a notice's conditions: its consent
// reference, or the document when the notice names none.
func consentKey(n ConsentNotice) string {
	if n.ConsentRef != "" {
		return "consent:" + n.ConsentRef + ":"
	}
	return "consent:doc:" + n.DocumentID + ":"
}

// QueueConsent queues the read of a filed document of this project for the
// PM who asked. A read already queued or running is ErrConsentBusy.
func (s *Store) QueueConsent(ctx context.Context, org, project, document, actor string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return err
	}
	var busy bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE org_id=$1::uuid AND document_id=$2::uuid AND kind='consent' AND status IN ('queued','leased'))`, org, document).Scan(&busy)
	if err != nil {
		return err
	}
	if busy {
		return ErrConsentBusy
	}
	tag, err := tx.Exec(ctx, `INSERT INTO jobs(org_id,id,document_id,kind,status,requested_by)
 SELECT d.org_id,$3::uuid,d.id,'consent','queued',$4::uuid FROM documents d WHERE d.org_id=$1::uuid AND d.id=$2::uuid AND d.project_id=$5::uuid AND d.status='filed'
 ON CONFLICT (org_id,document_id,kind) DO UPDATE SET status='queued',attempts=0,run_after=now(),locked_until=NULL,lease_token=NULL,last_error='',requested_by=EXCLUDED.requested_by,outcome=NULL`, org, document, newID(), actor, project)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

// QueueConsentAfterFiling queues the read of a notice filing recognised:
// the full_text job calls it once the text is stored. No one asked, so the
// rows are not one undoable change (no actor); the PM removes them in one
// bulk action instead. A read already done is not queued again.
func (s *Store) QueueConsentAfterFiling(ctx context.Context, org, document string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO jobs(org_id,id,document_id,kind,status)
 SELECT d.org_id,$3::uuid,d.id,'consent','queued' FROM documents d WHERE d.org_id=$1::uuid AND d.id=$2::uuid
 ON CONFLICT (org_id,document_id,kind) DO NOTHING`, org, document, newID())
	return err
}

// DocumentKind is a filed document's kind as filing decided it, or "".
func (s *Store) DocumentKind(ctx context.Context, org, document string) (string, error) {
	var kind string
	err := s.pool.QueryRow(ctx, `SELECT COALESCE((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='kind' LIMIT 1),'')
 FROM documents d WHERE d.org_id=$1::uuid AND d.id=$2::uuid`, org, document).Scan(&kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return kind, err
}

// ConsentDocument is a document the route may read: of this project, filed,
// with its text.
func (s *Store) ConsentDocument(ctx context.Context, org, project, document string) ([]SourcePage, error) {
	var uuid pgtype.UUID
	if uuid.Scan(document) != nil {
		return nil, ErrNotFound
	}
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid)`, org, project, document).Scan(&ok)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	pages, err := s.SpecificationSource(ctx, org, document)
	if errors.Is(err, ErrNotFound) {
		return nil, nil // filed without text yet, or a scan waiting for OCR
	}
	return pages, err
}

// ConsentPrior is the reading fingerprint of every condition row a notice's
// key already has, by number: a re-read with the same fingerprint asks Jev
// nothing.
func (s *Store) ConsentPrior(ctx context.Context, org, document string, n ConsentNotice) (map[string]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT i.details->>'number', COALESCE(i.provenance->'reading'->>'fingerprint','')
 FROM project_delivery_items i JOIN documents d ON d.org_id=i.org_id AND d.project_id=i.project_id
 WHERE i.org_id=$1::uuid AND d.id=$2::uuid AND i.kind='condition' AND left(i.source_proposal_key,length($3))=$3`, org, document, consentKey(n))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var number, print string
		if err := rows.Scan(&number, &print); err != nil {
			return nil, err
		}
		out[number] = print
	}
	return out, rows.Err()
}

type conditionRow struct {
	ID, Status, Review string
	Retired            bool
	Version            int64
	Content            delivery.Content
	Provenance         map[string]any
	Keyed              bool
}

// UpsertConditions writes one notice's conditions, while the job still holds
// its lease, in one transaction:
//   - a new number is a proposed row;
//   - the same words are left alone (a proposed row takes new picks);
//   - changed words replace a proposed row, but only flag one the PM has
//     accepted ("Changed by MOD/2023/0177"), and a satisfied one says
//     "changed since satisfied, check";
//   - "Deleted" proposes not applicable by a flag, never by status;
//   - an older notice never overwrites a row a newer one wrote;
//   - a row the PM deleted is not brought back.
//
// When a PM asked for the read, the rows are one change that PM can undo.
func (s *Store) UpsertConditions(ctx context.Context, job ClaimedJob, n ConsentNotice, conditions []ConsentRow) (int, error) {
	var actor *string
	var project string
	err := s.pool.QueryRow(ctx, `SELECT d.project_id::text, j.requested_by::text FROM jobs j JOIN documents d ON d.org_id=j.org_id AND d.id=j.document_id
 WHERE j.org_id=$1::uuid AND j.id=$2::uuid AND j.lease_token=$3::uuid AND j.status='leased' AND j.locked_until>now()`, job.OrgID, job.ID, job.Token).Scan(&project, &actor)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	change := Change{}
	if actor != nil {
		change = Change{ID: newID(), OrgID: job.OrgID, ProjectID: project, Actor: *actor, Via: "screen", Area: "delivery"}
		ctx = WithChange(ctx, change)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, job.OrgID, project); err != nil {
		return 0, err
	}
	// The lease is checked again under the project lock.
	var held bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE org_id=$1::uuid AND id=$2::uuid AND lease_token=$3::uuid AND status='leased' AND locked_until>now())`, job.OrgID, job.ID, job.Token).Scan(&held); err != nil {
		return 0, err
	}
	if !held {
		return 0, ErrNotFound
	}
	existing, err := conditionRows(ctx, tx, job.OrgID, project, n)
	if err != nil {
		return 0, err
	}
	packages := map[string]string{}
	notice := map[string]any{"document_id": n.DocumentID, "determined_on": n.Determined}
	label := n.ModRef
	if label == "" {
		label = "the notice of " + orDate(n.Determined)
	}
	written := 0
	for _, c := range conditions {
		row, found := existing[c.Number]
		switch {
		case found && row.Retired:
			continue // the PM removed it; a re-read does not bring it back
		case found && newerNotice(row.Provenance, n.Determined):
			continue // a newer notice wrote this row
		}
		d := delivery.ConditionDetails{ConsentRef: n.ConsentRef, Number: c.Number, Heading: c.Heading, Text: c.Text, Stage: c.Stage,
			StageHeading: c.StageHeading, Discipline: c.Discipline, EvidenceNeeded: c.EvidenceNeeded,
			Source: &delivery.ConditionSource{DocumentID: n.DocumentID, Page: c.Page}}
		flag := ""
		switch c.Marker {
		case "deleted":
			flag = "Deleted by " + orLabel(c.MarkerBy, label) + "; propose not applicable"
		case "changed":
			flag = "Changed by " + orLabel(c.MarkerBy, label)
		}
		pkg := ""
		if c.Discipline != "" {
			if p, ok := packages[c.Discipline]; ok {
				pkg = p
			} else if pkg, err = onePackage(ctx, tx, job.OrgID, project, c.Discipline); err != nil {
				return written, err
			}
			packages[c.Discipline] = pkg
		}
		if !found {
			if err := insertCondition(ctx, tx, job.OrgID, project, n, c, d, pkg, flag, notice); err != nil {
				return written, err
			}
			written++
			continue
		}
		before := delivery.Condition(row.Content)
		same := strings.TrimSpace(before.Text) == strings.TrimSpace(c.Text) && before.Heading == c.Heading
		proposed := row.Review == "proposed" && row.Keyed
		switch {
		case proposed:
			// SiteWise's own row: the newer reading replaces it.
			d.Evidence, d.RFP = before.Evidence, before.RFP
			if same && flag == "" {
				flag, _ = row.Provenance["flag"].(string)
			}
			if err := replaceCondition(ctx, tx, job.OrgID, project, row, c, d, pkg, flag, notice); err != nil {
				return written, err
			}
			written++
		case same && flag == "":
			// The PM's row and the same words: left as it is.
		default:
			if !same && flag == "" {
				flag = "Changed by " + label
			}
			if !same && row.Status == "satisfied" {
				flag += "; changed since satisfied, check"
			}
			if err := flagCondition(ctx, tx, job.OrgID, project, row.ID, flag, notice); err != nil {
				return written, err
			}
			written++
		}
	}
	outcome, _ := json.Marshal(n)
	if _, err := tx.Exec(ctx, `UPDATE jobs SET outcome=$3::jsonb WHERE org_id=$1::uuid AND id=$2::uuid`, job.OrgID, job.ID, string(outcome)); err != nil {
		return written, err
	}
	if err := s.finishDeliveryWrite(ctx, tx, job.OrgID, project); err != nil {
		return written, err
	}
	if change.ID != "" {
		ops, _ := json.Marshal([]map[string]any{{"op": "delivery.read_notice", "fields": map[string]any{"document": n.DocumentID, "conditions": written}}})
		if err := s.SetChangeOps(context.WithoutCancel(ctx), job.OrgID, change.ID, ops); err != nil && !errors.Is(err, ErrNotFound) {
			return written, err
		}
	}
	return written, nil
}

func orDate(d string) string {
	if d == "" {
		return "an unknown date"
	}
	return d
}

func orLabel(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// conditionRows are a notice's existing rows by number: the ones its key
// wrote (retired ones too), and conditions the PM added by hand with the
// same number and consent.
func conditionRows(ctx context.Context, tx pgx.Tx, org, project string, n ConsentNotice) (map[string]conditionRow, error) {
	rows, err := tx.Query(ctx, `SELECT to_jsonb(i), source_proposal_key IS NOT NULL FROM project_delivery_items i
 WHERE org_id=$1::uuid AND project_id=$2::uuid AND kind='condition'
 AND (left(source_proposal_key,length($3))=$3 OR source_proposal_key IS NULL AND COALESCE(details->>'consent_ref','') IN ('',$4))`, org, project, consentKey(n), n.ConsentRef)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]conditionRow{}
	for rows.Next() {
		var raw []byte
		var keyed bool
		if err := rows.Scan(&raw, &keyed); err != nil {
			return nil, err
		}
		var it delivery.Item
		if err := json.Unmarshal(raw, &it); err != nil {
			return nil, err
		}
		r := conditionRow{ID: it.ID, Status: it.Status, Review: it.ReviewStatus, Retired: it.RetiredAt != nil, Version: it.Version, Content: it.Content, Keyed: keyed}
		_ = json.Unmarshal(it.Provenance, &r.Provenance)
		number := delivery.Condition(it.Content).Number
		if prior, ok := out[number]; ok && prior.Keyed && !keyed {
			continue // the key's own row wins over a hand-added twin
		}
		out[number] = r
	}
	return out, rows.Err()
}

// newerNotice reports a row last written by a notice determined after this
// one. An undated notice is never newer.
func newerNotice(provenance map[string]any, determined string) bool {
	notices, _ := provenance["notices"].([]any)
	for _, raw := range notices {
		n, _ := raw.(map[string]any)
		if on, _ := n["determined_on"].(string); on != "" && determined != "" && on > determined {
			return true
		}
	}
	return false
}

// onePackage is the live package with the discipline, when there is
// exactly one; otherwise the PM picks.
func onePackage(ctx context.Context, tx pgx.Tx, org, project, discipline string) (string, error) {
	rows, err := tx.Query(ctx, `SELECT id::text FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND retired_at IS NULL AND lifecycle_status<>'cancelled' AND discipline_id=$3 LIMIT 2`, org, project, discipline)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		ids = append(ids, id)
	}
	if len(ids) != 1 {
		return "", rows.Err()
	}
	return ids[0], rows.Err()
}

func conditionProvenance(c ConsentRow, flag string, notice map[string]any, prior map[string]any) []byte {
	p := map[string]any{"reading": json.RawMessage(c.Reading)}
	if len(c.Reading) == 0 {
		p["reading"] = map[string]any{}
	}
	var notices []any
	if prior != nil {
		notices, _ = prior["notices"].([]any)
	}
	notices = append(notices, map[string]any{"document_id": notice["document_id"], "determined_on": notice["determined_on"], "change": c.Marker})
	if len(notices) > 20 {
		notices = notices[len(notices)-20:]
	}
	p["notices"] = notices
	if flag != "" {
		p["flag"] = flag
	}
	if c.Deferred {
		p["deferred_commencement"] = true
	}
	raw, _ := json.Marshal(p)
	return raw
}

func insertCondition(ctx context.Context, tx pgx.Tx, org, project string, n ConsentNotice, c ConsentRow, d delivery.ConditionDetails, pkg, flag string, notice map[string]any) error {
	details, err := json.Marshal(d)
	if err != nil {
		return err
	}
	content := delivery.Content{Kind: "condition", Title: delivery.ConditionTitle(c.Number, c.Heading, c.Text), OwnerText: c.OwnerText, Status: "open", Details: details}
	if err := delivery.Validate(content); err != nil {
		return fmt.Errorf("%w: condition %s: %v", ErrInvalidDelivery, c.Number, err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO project_delivery_items(org_id,id,project_id,kind,title,owner_text,status,package_id,details,origin,review_status,provenance,source_proposal_key,system_id)
 VALUES($1::uuid,$2::uuid,$3::uuid,'condition',$4,$5,'open',NULLIF($6,'')::uuid,$7::jsonb,'document','proposed',$8::jsonb||jsonb_build_object('at',now()),$9,NULLIF($10,''))
 ON CONFLICT (org_id,project_id,source_proposal_key) WHERE source_proposal_key IS NOT NULL DO NOTHING`,
		org, newID(), project, content.Title, c.OwnerText, pkg, details, conditionProvenance(c, flag, notice, nil), consentKey(n)+c.Number, c.SystemID)
	return err
}

func replaceCondition(ctx context.Context, tx pgx.Tx, org, project string, row conditionRow, c ConsentRow, d delivery.ConditionDetails, pkg, flag string, notice map[string]any) error {
	details, err := json.Marshal(d)
	if err != nil {
		return err
	}
	title := delivery.ConditionTitle(c.Number, c.Heading, c.Text)
	if err := delivery.Validate(delivery.Content{Kind: "condition", Title: title, Status: row.Status, Details: details}); err != nil {
		return fmt.Errorf("%w: condition %s: %v", ErrInvalidDelivery, c.Number, err)
	}
	_, err = tx.Exec(ctx, `UPDATE project_delivery_items SET title=$4,owner_text=$5,package_id=NULLIF($6,'')::uuid,details=$7::jsonb,system_id=NULLIF($8,''),provenance=$9::jsonb||jsonb_build_object('at',provenance->'at'),version=version+1
 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND review_status='proposed'`,
		org, project, row.ID, title, c.OwnerText, pkg, details, c.SystemID, conditionProvenance(c, flag, notice, row.Provenance))
	return err
}

func flagCondition(ctx context.Context, tx pgx.Tx, org, project, id, flag string, notice map[string]any) error {
	entry, _ := json.Marshal([]any{map[string]any{"document_id": notice["document_id"], "determined_on": notice["determined_on"]}})
	_, err := tx.Exec(ctx, `UPDATE project_delivery_items SET provenance=provenance||jsonb_build_object('flag',$4::text,'notices',COALESCE(provenance->'notices','[]'::jsonb)||$5::jsonb),version=version+1
 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, id, flag, string(entry))
	return err
}

// ConsentRead is one notice's read, for the register's header: its job's
// state and what it found.
type ConsentRead struct {
	DocumentID string         `json:"document_id"`
	Title      string         `json:"title"`
	Status     string         `json:"status"` // queued, reading, done, failed
	Error      string         `json:"error,omitempty"`
	Notice     *ConsentNotice `json:"notice,omitempty"`
	// Rows is how many live condition rows came from this notice.
	Rows int `json:"rows"`
}

// ConsentReads lists the project's consent reads, newest first.
func (s *Store) ConsentReads(ctx context.Context, org, project string) ([]ConsentRead, error) {
	rows, err := s.pool.Query(ctx, `SELECT j.document_id::text,
 COALESCE((SELECT t.value FROM decisions t WHERE t.org_id=d.org_id AND t.document_id=d.id AND t.field='title' LIMIT 1), d.filename),
 CASE WHEN j.status='leased' THEN 'reading' ELSE j.status END, j.last_error, j.outcome,
 (SELECT count(*) FROM project_delivery_items i WHERE i.org_id=j.org_id AND i.project_id=d.project_id AND i.kind='condition' AND i.retired_at IS NULL AND i.details->'source'->>'document_id'=j.document_id::text)::int
 FROM jobs j JOIN documents d ON d.org_id=j.org_id AND d.id=j.document_id
 WHERE j.org_id=$1::uuid AND d.project_id=$2::uuid AND j.kind='consent' ORDER BY j.created_at DESC, j.id`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ConsentRead{}
	for rows.Next() {
		var r ConsentRead
		var outcome []byte
		if err := rows.Scan(&r.DocumentID, &r.Title, &r.Status, &r.Error, &outcome, &r.Rows); err != nil {
			return nil, err
		}
		if len(outcome) > 0 {
			var n ConsentNotice
			if json.Unmarshal(outcome, &n) == nil {
				r.Notice = &n
			}
		}
		if r.Status != "failed" {
			r.Error = ""
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
