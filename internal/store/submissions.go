package store

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/costs"
	"sitewise/internal/invoices"
	"sitewise/internal/proposals"
	"sitewise/internal/rfp"
)

// EventProcurement wakes the Procurement page: a submission was opened,
// read, edited or awarded.
const EventProcurement = "procurement"

// Submission statuses as stored.
const (
	SubmissionReading     = "reading"
	SubmissionReceived    = "received"
	SubmissionNeedsReview = "needs_review"
	SubmissionAwarded     = "awarded"
	SubmissionUnsuccess   = "unsuccessful"
	SubmissionSuperseded  = "superseded"
	SubmissionWithdrawn   = "withdrawn"
	SubmissionNotProposal = "not_proposal"
)

// Submission is one firm's proposal or tender for one package.
type Submission struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	PackageID    string    `json:"package_id"`
	FirmID       string    `json:"firm_id"`
	DocumentID   string    `json:"document_id"`
	Filename     string    `json:"filename"`
	Kind         string    `json:"kind"`
	Status       string    `json:"status"`
	ReceivedOn   string    `json:"received_on"`
	SupersededBy string    `json:"superseded_by"`
	StatedTotal  *int64    `json:"-"`
	GSTBasis     string    `json:"gst_basis"`
	Flags        []string  `json:"flags"`
	CreatedAt    time.Time `json:"created_at"`
	Version      int64     `json:"version"`
}

// Firm is one firm in the organisation's directory.
type Firm struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	ABN     string `json:"abn"`
	Address string `json:"address"`
	Contact string `json:"contact"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Status  string `json:"status"`
	Version int64  `json:"version"`
}

func procurementEvent(ctx context.Context, tx pgx.Tx, s *Store, org, project, submission, state string) error {
	payload, _ := json.Marshal(map[string]string{"project_id": project, "submission_id": submission, "state": state})
	_, err := appendEvent(ctx, s.q.WithTx(tx), org, EventProcurement, "", string(payload))
	return err
}

// SydneyDate is the calendar date in Australia/Sydney: a proposal's
// received date and an award's date are the PM's date, not the server's.
func SydneyDate(t time.Time) string {
	loc, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		return t.UTC().Format("2006-01-02")
	}
	return t.In(loc).Format("2006-01-02")
}

// OpenSubmission makes the submission for a filed document. packageID is
// set when the user dropped the file on a package row: that settles the
// package as the user's act. The same document always has one submission.
func (s *Store) OpenSubmission(ctx context.Context, org, project, documentID, packageID string, now time.Time) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var filename string
	err = tx.QueryRow(ctx, `SELECT filename FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3`, org, project, documentID).Scan(&filename)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if packageID != "" {
		if err := packageInProject(ctx, tx, org, project, packageID); err != nil {
			return "", err
		}
	}
	var id string
	err = tx.QueryRow(ctx, `SELECT id::text FROM submissions WHERE org_id=$1::uuid AND document_id=$2::uuid`, org, documentID).Scan(&id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		id = newID()
		if _, err := tx.Exec(ctx, `INSERT INTO submissions (org_id,id,project_id,package_id,document_id,filename,status,received_on)
VALUES ($1::uuid,$2::uuid,$3::uuid,NULLIF($4,'')::uuid,$5::uuid,$6,'reading',$7::date)`, org, id, project, packageID, documentID, clipText(filename, 255), SydneyDate(now)); err != nil {
			return "", err
		}
	case err != nil:
		return "", err
	case packageID != "":
		if _, err := tx.Exec(ctx, `UPDATE submissions SET package_id=$3::uuid, updated_at=now(), version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid`, org, id, packageID); err != nil {
			return "", err
		}
	}
	if packageID != "" {
		if err := upsertSubmissionDecision(ctx, tx, org, project, id, proposals.Decision{Field: proposals.FieldPackage, Value: packageID, Band: proposals.BandGreen, DecidedBy: proposals.ByUser}, true); err != nil {
			return "", err
		}
	}
	if err := procurementEvent(ctx, tx, s, org, project, id, "reading"); err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

// SubmissionForDocument is the submission opened for a document, or "".
func (s *Store) SubmissionForDocument(ctx context.Context, org, documentID string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id::text FROM submissions WHERE org_id=$1::uuid AND document_id::text=$2`, org, documentID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// FiledKind is a filed document's kind and discipline values, "" if unset.
func (s *Store) FiledKind(ctx context.Context, org, documentID string) (kind, discipline string, err error) {
	err = s.pool.QueryRow(ctx, `SELECT
 COALESCE((SELECT value FROM decisions WHERE org_id=$1::uuid AND document_id=$2::uuid AND field='kind'),''),
 COALESCE((SELECT value FROM decisions WHERE org_id=$1::uuid AND document_id=$2::uuid AND field='discipline'),'')`, org, documentID).Scan(&kind, &discipline)
	return kind, discipline, err
}

// SubmissionReadInput is what one read needs: the bytes to open and the
// project to read them into.
type SubmissionReadInput struct {
	Submission
	SHA256  []byte
	Size    int64
	Context proposals.Context
	Keep    map[string]proposals.Decision
}

// SubmissionForReading gathers the project context: live packages, their
// schedules, the org's own email domains, the site address and the
// construction budget, and the decisions a read must keep.
func (s *Store) SubmissionForReading(ctx context.Context, org, id string, recheck bool) (SubmissionReadInput, error) {
	var in SubmissionReadInput
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return in, err
	}
	defer tx.Rollback(ctx)
	sub, err := readSubmission(ctx, tx, org, id)
	if err != nil {
		return in, err
	}
	in.Submission = sub
	if sub.DocumentID == "" {
		return in, ErrNotFound
	}
	if err := tx.QueryRow(ctx, `SELECT f.sha256, f.byte_size FROM documents d JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id
WHERE d.org_id=$1::uuid AND d.id=$2::uuid`, org, sub.DocumentID).Scan(&in.SHA256, &in.Size); err != nil {
		return in, err
	}
	pc := proposals.Context{Schedules: map[string][]rfp.ScheduleItem{}}
	if err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT value FROM decisions WHERE org_id=$1::uuid AND document_id=$2::uuid AND field='discipline'),'')`, org, sub.DocumentID).Scan(&pc.Discipline); err != nil {
		return in, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text, title, COALESCE(discipline_id,'') FROM packages
WHERE org_id=$1::uuid AND project_id=$2::uuid AND retired_at IS NULL AND lifecycle_status <> 'cancelled' ORDER BY title, id`, org, sub.ProjectID)
	if err != nil {
		return in, err
	}
	for rows.Next() {
		var p proposals.Package
		if err := rows.Scan(&p.ID, &p.Title, &p.Discipline); err != nil {
			rows.Close()
			return in, err
		}
		pc.Packages = append(pc.Packages, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return in, err
	}
	in.Keep = map[string]proposals.Decision{}
	decs, err := submissionDecisions(ctx, tx, org, id)
	if err != nil {
		return in, err
	}
	for f, d := range decs {
		if d.DecidedBy == proposals.ByUser || recheck && d.Band != proposals.BandGrey {
			in.Keep[f] = d
		}
	}
	if u, ok := in.Keep[proposals.FieldPackage]; ok && u.DecidedBy == proposals.ByUser {
		pc.PackageID = u.Value
	} else if sub.PackageID != "" && decs[proposals.FieldPackage].DecidedBy == proposals.ByUser {
		pc.PackageID = sub.PackageID
	}
	// Schedules for the package a rule or the user will settle, or every
	// candidate when Jev must choose.
	for _, p := range pc.Packages {
		if pc.PackageID != "" && p.ID != pc.PackageID {
			continue
		}
		if pc.PackageID == "" && pc.Discipline != "" && p.Discipline != pc.Discipline && countDiscipline(pc.Packages, pc.Discipline) > 0 {
			continue
		}
		if err := s.ensureSchedule(ctx, tx, org, sub.ProjectID, p.ID); err != nil {
			return in, err
		}
		items, err := readSchedule(ctx, tx, org, p.ID)
		if err != nil {
			return in, err
		}
		for _, it := range items {
			if !it.Retired {
				pc.Schedules[p.ID] = append(pc.Schedules[p.ID], it.ScheduleItem)
			}
		}
	}
	domains, err := tx.Query(ctx, `SELECT DISTINCT lower(split_part(email,'@',2)) FROM users WHERE org_id=$1::uuid`, org)
	if err != nil {
		return in, err
	}
	for domains.Next() {
		var d string
		if err := domains.Scan(&d); err != nil {
			domains.Close()
			return in, err
		}
		pc.OwnDomains = append(pc.OwnDomains, d)
	}
	domains.Close()
	if err := tx.QueryRow(ctx, `SELECT COALESCE(s.address,'') FROM projects p JOIN sites s ON s.org_id=p.org_id AND s.id=p.site_id WHERE p.org_id=$1::uuid AND p.id=$2::uuid`, org, sub.ProjectID).Scan(&pc.SiteAddress); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return in, err
	}
	pc.ConstructionBudget, err = constructionBudget(ctx, tx, org, sub.ProjectID)
	if err != nil {
		return in, err
	}
	in.Context = pc
	return in, tx.Commit(ctx)
}

func countDiscipline(ps []proposals.Package, d string) int {
	n := 0
	for _, p := range ps {
		if p.Discipline == d {
			n++
		}
	}
	return n
}

// constructionBudget is the cost plan's construction budget in cents when it
// is complete and known, the basis for a percentage fee; nil otherwise.
func constructionBudget(ctx context.Context, tx pgx.Tx, org, project string) (*int64, error) {
	var has bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM cost_plans WHERE org_id=$1::uuid AND project_id=$2::uuid)`, org, project).Scan(&has); err != nil || !has {
		return nil, err
	}
	p, err := readCostPlan(ctx, tx, org, project, "")
	if err != nil {
		return nil, err
	}
	t, err := costs.Summarize(p, "overall")
	if err != nil {
		return nil, nil
	}
	b := t.Sections["construction"]["budget"]
	if !b.Complete || b.Amount == nil {
		return nil, nil
	}
	c, ok, err := invoices.ParseCents(string(*b.Amount))
	if err != nil || !ok || c <= 0 {
		return nil, nil
	}
	return &c, nil
}

// FirmLookup finds a directory firm by ABN, then by an exact name key.
func (s *Store) FirmLookup(ctx context.Context, org string) func(abns, keys []string) *proposals.Firm {
	return func(abns, keys []string) *proposals.Firm {
		var f proposals.Firm
		if len(abns) > 0 {
			err := s.pool.QueryRow(ctx, `SELECT id::text,name,COALESCE(abn,''),address,contact,email,phone FROM firms WHERE org_id=$1::uuid AND abn = ANY($2::text[]) ORDER BY status='confirmed' DESC, created_at LIMIT 1`, org, abns).Scan(&f.ID, &f.Name, &f.ABN, &f.Address, &f.Contact, &f.Email, &f.Phone)
			if err == nil {
				return &f
			}
		}
		if len(keys) > 0 {
			rows, err := s.pool.Query(ctx, `SELECT id::text,name,COALESCE(abn,''),address,contact,email,phone FROM firms WHERE org_id=$1::uuid AND name_key = ANY($2::text[]) LIMIT 2`, org, keys)
			if err != nil {
				return nil
			}
			defer rows.Close()
			var found []proposals.Firm
			for rows.Next() {
				var g proposals.Firm
				if rows.Scan(&g.ID, &g.Name, &g.ABN, &g.Address, &g.Contact, &g.Email, &g.Phone) == nil {
					found = append(found, g)
				}
			}
			// Two directory firms with one name is not a match.
			if len(found) == 1 {
				return &found[0]
			}
		}
		return nil
	}
}

func readSubmission(ctx context.Context, q querier, org, id string) (Submission, error) {
	var sub Submission
	var flags []byte
	err := q.QueryRow(ctx, `SELECT id::text, project_id::text, COALESCE(package_id::text,''), COALESCE(firm_id::text,''), COALESCE(document_id::text,''), filename,
 COALESCE(kind,''), status, received_on::text, COALESCE(superseded_by::text,''), stated_total_cents, gst_basis, flags, created_at, version
FROM submissions WHERE org_id=$1::uuid AND id::text=$2`, org, id).Scan(&sub.ID, &sub.ProjectID, &sub.PackageID, &sub.FirmID, &sub.DocumentID, &sub.Filename,
		&sub.Kind, &sub.Status, &sub.ReceivedOn, &sub.SupersededBy, &sub.StatedTotal, &sub.GSTBasis, &flags, &sub.CreatedAt, &sub.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return sub, ErrNotFound
	}
	if err == nil {
		err = json.Unmarshal(flags, &sub.Flags)
	}
	return sub, err
}

func submissionDecisions(ctx context.Context, q querier, org, id string) (map[string]proposals.Decision, error) {
	rows, err := q.Query(ctx, `SELECT field, value, band, decided_by, question_version, confidence FROM submission_decisions WHERE org_id=$1::uuid AND submission_id::text=$2`, org, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]proposals.Decision{}
	for rows.Next() {
		var d proposals.Decision
		if err := rows.Scan(&d.Field, &d.Value, &d.Band, &d.DecidedBy, &d.QuestionVersion, &d.Confidence); err != nil {
			return nil, err
		}
		out[d.Field] = d
	}
	return out, rows.Err()
}

// upsertSubmissionDecision stores one judgement. A machine decision never
// replaces a user's; a user's keeps what the machine said as proposed_value.
func upsertSubmissionDecision(ctx context.Context, tx pgx.Tx, org, project, id string, d proposals.Decision, user bool) error {
	if user {
		_, err := tx.Exec(ctx, `INSERT INTO submission_decisions (org_id,project_id,submission_id,field,value,band,decided_by,question_version)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5,'green','user','')
ON CONFLICT (org_id,submission_id,field) DO UPDATE SET
 proposed_value = CASE WHEN submission_decisions.decided_by='user' THEN submission_decisions.proposed_value ELSE submission_decisions.value END,
 value=EXCLUDED.value, band='green', decided_by='user', question_version='', confidence=NULL, version=submission_decisions.version+1, updated_at=now()`,
			org, project, id, clipText(d.Field, 220), clipText(d.Value, 2000))
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO submission_decisions (org_id,project_id,submission_id,field,value,band,decided_by,question_version,confidence)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8,$9)
ON CONFLICT (org_id,submission_id,field) DO UPDATE SET value=EXCLUDED.value, band=EXCLUDED.band, decided_by=EXCLUDED.decided_by,
 question_version=EXCLUDED.question_version, confidence=EXCLUDED.confidence, version=submission_decisions.version+1, updated_at=now()
WHERE submission_decisions.decided_by <> 'user'`,
		org, project, id, clipText(d.Field, 220), clipText(d.Value, 2000), d.Band, d.DecidedBy, d.QuestionVersion, d.Confidence)
	return err
}

// CommitSubmissionRead stores one read in one transaction: decisions (never
// over a user's), the firm (matched, or a new draft in the directory),
// prices, passages and responses, then supersedes an earlier submission
// from the same firm for the same package. A late read of a submission no
// longer waiting is dropped unless it is a re-check.
func (s *Store) CommitSubmissionRead(ctx context.Context, org, id string, out proposals.Outcome, recheck bool, now time.Time) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var project, status string
	err = tx.QueryRow(ctx, `SELECT project_id::text, status FROM submissions WHERE org_id=$1::uuid AND id::text=$2 FOR UPDATE`, org, id).Scan(&project, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !recheck && status != SubmissionReading {
		return false, nil
	}
	for _, d := range out.Decisions {
		if d.DecidedBy == proposals.ByUser {
			continue
		}
		if err := upsertSubmissionDecision(ctx, tx, org, project, id, d, false); err != nil {
			return false, err
		}
	}
	decs, err := submissionDecisions(ctx, tx, org, id)
	if err != nil {
		return false, err
	}
	kind := decs[proposals.FieldKind].Value
	next := SubmissionReceived
	switch {
	case kind == proposals.KindNone:
		next = SubmissionNotProposal
	case out.NotRead != "":
		next = SubmissionNeedsReview
	}
	if status != SubmissionReading && status != SubmissionReceived && status != SubmissionNeedsReview {
		next = status // an award, a withdrawal or a supersession stands
	}
	pkg := decs[proposals.FieldPackage].Value
	if pkg != "" {
		if err := packageInProject(ctx, tx, org, project, pkg); err != nil {
			pkg = ""
		}
	}
	firm, err := s.resolveFirm(ctx, tx, org, out.Firm, decs)
	if err != nil {
		return false, err
	}
	if err := replaceMachinePrices(ctx, tx, org, project, id, out.Prices); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM submission_passages WHERE org_id=$1::uuid AND submission_id=$2::uuid`, org, id); err != nil {
		return false, err
	}
	for _, p := range out.Passages {
		if _, err := tx.Exec(ctx, `INSERT INTO submission_passages (org_id,project_id,submission_id,ordinal,heading,page,body) VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7)`,
			org, project, id, p.Ordinal, clipText(p.Heading, 500), max(p.Page, 0), clipText(p.Body, 20000)); err != nil {
			return false, err
		}
	}
	for _, r := range out.Responses {
		passages := r.Passages
		if passages == nil {
			passages = []int{}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO submission_responses (org_id,project_id,submission_id,item_key,passages,band,decided_by)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7)
ON CONFLICT (org_id,submission_id,item_key) DO UPDATE SET passages=EXCLUDED.passages, band=EXCLUDED.band, decided_by=EXCLUDED.decided_by, updated_at=now()
WHERE submission_responses.decided_by <> 'user'`, org, project, id, r.Key, passages, r.Band, r.DecidedBy); err != nil {
			return false, err
		}
	}
	flags, _ := json.Marshal(stringsOrEmpty(out.Flags))
	var stated *int64
	if out.StatedTotal != nil {
		v := *out.StatedTotal
		stated = &v
	}
	var retry *time.Time
	if out.Grey {
		t := now.Add(30 * time.Second)
		retry = &t
	}
	if _, err := tx.Exec(ctx, `UPDATE submissions SET status=$3, kind=NULLIF($4,''), package_id=NULLIF($5,'')::uuid, firm_id=NULLIF($6,'')::uuid,
 stated_total_cents=$7, gst_basis=$8, flags=$9::jsonb, retry_after=$10, read_attempts=read_attempts + CASE WHEN $11 THEN 1 ELSE 0 END,
 updated_at=now(), version=version+1
WHERE org_id=$1::uuid AND id::text=$2`, org, id, next, kindColumn(kind), pkg, firm, stated, gstColumn(out.GSTBasis), string(flags), retry, recheck); err != nil {
		return false, err
	}
	if err := supersedeEarlier(ctx, tx, org, project, id); err != nil {
		return false, err
	}
	if err := procurementEvent(ctx, tx, s, org, project, id, next); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func stringsOrEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func kindColumn(kind string) string {
	if kind == proposals.KindFeeProposal || kind == proposals.KindTender {
		return kind
	}
	return ""
}

func gstColumn(g string) string {
	if g == "ex" || g == "inc" {
		return g
	}
	return "unknown"
}

// resolveFirm is the submission's directory firm: the read's match, else
// the directory firm with the decided ABN or name, else a new draft firm
// from the decided details. No firm name, no firm.
func (s *Store) resolveFirm(ctx context.Context, tx pgx.Tx, org string, match *proposals.Firm, decs map[string]proposals.Decision) (string, error) {
	if match != nil {
		return match.ID, nil
	}
	name := strings.TrimSpace(decs[proposals.FieldFirmName].Value)
	abn := invoices.NormalizeABN(decs[proposals.FieldFirmABN].Value)
	if abn != "" && !invoices.ValidABN(abn) {
		abn = ""
	}
	var id string
	if abn != "" {
		err := tx.QueryRow(ctx, `SELECT id::text FROM firms WHERE org_id=$1::uuid AND abn=$2`, org, abn).Scan(&id)
		if err == nil {
			return id, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
	}
	key := proposals.NameKey(name)
	if name == "" || key == "" {
		return "", nil
	}
	rows, err := tx.Query(ctx, `SELECT id::text FROM firms WHERE org_id=$1::uuid AND name_key=$2 LIMIT 2`, org, key)
	if err != nil {
		return "", err
	}
	var ids []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			rows.Close()
			return "", err
		}
		ids = append(ids, f)
	}
	rows.Close()
	if len(ids) == 1 {
		return ids[0], nil
	}
	id = newID()
	err = tx.QueryRow(ctx, `INSERT INTO firms (org_id,id,name,name_key,abn,address,contact,email,phone,status)
VALUES ($1::uuid,$2::uuid,$3,$4,NULLIF($5,''),$6,$7,$8,$9,'draft')
ON CONFLICT (org_id,abn) WHERE abn IS NOT NULL DO UPDATE SET updated_at=now()
RETURNING id::text`, org, id, clipText(name, 200), key, abn,
		clipText(decs[proposals.FieldFirmAddress].Value, 500), clipText(decs[proposals.FieldFirmContact].Value, 200),
		clipText(decs[proposals.FieldFirmEmail].Value, 254), clipText(decs[proposals.FieldFirmPhone].Value, 50)).Scan(&id)
	return id, err
}

// replaceMachinePrices rewrites the read's cells and keeps every cell a user
// set. Other items are rewritten whole: they have no key to keep.
func replaceMachinePrices(ctx context.Context, tx pgx.Tx, org, project, id string, prices []proposals.Price) error {
	if _, err := tx.Exec(ctx, `DELETE FROM submission_prices WHERE org_id=$1::uuid AND submission_id=$2::uuid AND decided_by <> 'user'`, org, id); err != nil {
		return err
	}
	for n, p := range prices {
		if p.State == "" && p.Key == "" {
			continue
		}
		state := p.State
		if state == "" {
			// Unread: the cell keeps its band and no figure.
			state = proposals.StateNotPriced
		}
		cents := p.Cents
		if state != proposals.StatePriced {
			cents = nil
		}
		band := p.Band
		if p.State == "" && band != proposals.BandGrey {
			band = proposals.BandBlank
		}
		if _, err := tx.Exec(ctx, `INSERT INTO submission_prices (org_id,id,project_id,submission_id,item_key,label,amount_cents,state,basis,page,passage,band,decided_by,ordinal)
SELECT $1::uuid,$2::uuid,$3::uuid,$4::uuid,NULLIF($5,''),$6,$7,$8,$9,$10,$11,$12,$13,$14
WHERE NOT EXISTS (SELECT 1 FROM submission_prices WHERE org_id=$1::uuid AND submission_id=$4::uuid AND item_key=NULLIF($5,''))`,
			org, newID(), project, id, p.Key, clipText(p.Label, 500), cents, state, clipText(p.Basis, 500), max(p.Page, 0), clipText(p.Passage, 2000), band, p.DecidedBy, n); err != nil {
			return err
		}
	}
	return nil
}

// supersedeEarlier marks an earlier live submission from the same firm for
// the same package superseded by this one. An awarded one is never moved.
func supersedeEarlier(ctx context.Context, tx pgx.Tx, org, project, id string) error {
	_, err := tx.Exec(ctx, `UPDATE submissions o SET status='superseded', superseded_by=n.id, updated_at=now(), version=o.version+1
FROM submissions n
WHERE n.org_id=$1::uuid AND n.id=$3::uuid AND n.package_id IS NOT NULL AND n.firm_id IS NOT NULL AND n.status IN ('received','needs_review')
  AND o.org_id=n.org_id AND o.project_id=$2::uuid AND o.id<>n.id AND o.package_id=n.package_id AND o.firm_id=n.firm_id
  AND o.status IN ('received','needs_review','unsuccessful') AND o.created_at < n.created_at`, org, project, id)
	return err
}

// PendingSubmissionReads are submissions still reading, for a restart.
func (s *Store) PendingSubmissionReads(ctx context.Context) ([]SubmissionRef, error) {
	return s.submissionRefs(ctx, `SELECT org_id::text, id::text FROM submissions WHERE status='reading' AND document_id IS NOT NULL`)
}

// DueSubmissionRechecks are grey reads to ask again, at most three times.
func (s *Store) DueSubmissionRechecks(ctx context.Context, now time.Time) ([]SubmissionRef, error) {
	return s.submissionRefs(ctx, `SELECT org_id::text, id::text FROM submissions WHERE retry_after IS NOT NULL AND retry_after <= $1 AND read_attempts < 3 AND document_id IS NOT NULL`, now)
}

// SubmissionRef names one submission in its org.
type SubmissionRef struct{ OrgID, SubmissionID string }

func (s *Store) submissionRefs(ctx context.Context, sql string, args ...any) ([]SubmissionRef, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SubmissionRef
	for rows.Next() {
		var r SubmissionRef
		if err := rows.Scan(&r.OrgID, &r.SubmissionID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ClaimSubmissionRecheck clears the retry so one sweep reads it once.
func (s *Store) ClaimSubmissionRecheck(ctx context.Context, ref SubmissionRef, now time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE submissions SET retry_after=NULL WHERE org_id=$1::uuid AND id=$2::uuid AND retry_after IS NOT NULL AND retry_after <= $3`, ref.OrgID, ref.SubmissionID, now)
	return tag.RowsAffected() == 1, err
}

// itoa keeps money JSON exact.
func centsText(c *int64) string {
	if c == nil {
		return ""
	}
	return strconv.FormatInt(*c, 10)
}

// ErrInvalidInput is a procurement edit a user must fix.
var ErrInvalidInput = errors.New("invalid procurement input")

// ErrConflict refuses an edit an award depends on.
var ErrConflict = errors.New("procurement record is in use")
