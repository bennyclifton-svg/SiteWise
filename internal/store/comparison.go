package store

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/invoices"
	"sitewise/internal/proposals"
	"sitewise/internal/rfp"
)

// FirmColumn is one firm in a package's price table. Totals are computed in
// Go from the stored cells.
type FirmColumn struct {
	Quote         *proposals.QuoteReview        `json:"quote,omitempty"`
	Builder       *proposals.BuilderReview      `json:"builder,omitempty"`
	Submission    Submission                    `json:"submission"`
	Firm          *Firm                         `json:"firm"`
	Cells         map[string]proposals.Cell     `json:"cells"`
	Others        []proposals.Cell              `json:"others"`
	Totals        proposals.Column              `json:"-"`
	NeedsReview   bool                          `json:"needs_review"`
	Decisions     map[string]proposals.Decision `json:"decisions"`
	UndoAlignment bool                          `json:"undo_alignment"`
}

// PriceTable is a package's schedule down the side and its firms across.
type PriceTable struct {
	Builder    bool           `json:"builder"`
	PackageID  string         `json:"package_id"`
	Title      string         `json:"title"`
	Discipline string         `json:"discipline_id"`
	Lifecycle  string         `json:"lifecycle_status"`
	Items      []ScheduleItem `json:"items"`
	Columns    []FirmColumn   `json:"columns"`
	// Earlier is superseded, withdrawn and not-a-proposal submissions,
	// kept and shown below the table.
	Earlier []Submission `json:"earlier"`
}

// liveColumn is a submission that appears in the table.
func liveColumn(status string) bool {
	switch status {
	case SubmissionReading, SubmissionReceived, SubmissionNeedsReview, SubmissionAwarded, SubmissionUnsuccess:
		return true
	}
	return false
}

// ReadPriceTable is one package's price table in one read-only snapshot.
func (s *Store) ReadPriceTable(ctx context.Context, org, project, pkg string) (PriceTable, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return PriceTable{}, err
	}
	defer tx.Rollback(ctx)
	t, err := s.priceTable(ctx, tx, org, project, pkg)
	if err != nil {
		return t, err
	}
	return t, tx.Commit(ctx)
}

func (s *Store) priceTable(ctx context.Context, tx pgx.Tx, org, project, pkg string) (PriceTable, error) {
	t := PriceTable{PackageID: pkg, Items: []ScheduleItem{}, Columns: []FirmColumn{}, Earlier: []Submission{}}
	err := tx.QueryRow(ctx, `SELECT title, COALESCE(discipline_id,''), lifecycle_status FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3`, org, project, pkg).Scan(&t.Title, &t.Discipline, &t.Lifecycle)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	if err != nil {
		return t, err
	}
	if t.Builder, err = builderPackage(ctx, tx, org, pkg); err != nil {
		return t, err
	}
	if err := s.ensureSchedule(ctx, tx, org, project, pkg); err != nil {
		return t, err
	}
	items, err := readSchedule(ctx, tx, org, pkg)
	if err != nil {
		return t, err
	}
	var live []rfp.ScheduleItem
	for _, it := range items {
		if !it.Retired {
			t.Items = append(t.Items, it)
			live = append(live, it.ScheduleItem)
		}
	}
	cols, earlier, err := s.packageColumns(ctx, tx, org, project, pkg, live)
	if err != nil {
		return t, err
	}
	t.Columns, t.Earlier = cols, earlier
	return t, nil
}

// packageColumns loads every submission of one package with its firm, cells
// and decisions, and computes each live column's totals.
func (s *Store) packageColumns(ctx context.Context, tx pgx.Tx, org, project, pkg string, items []rfp.ScheduleItem) ([]FirmColumn, []Submission, error) {
	isBuilder, err := builderPackage(ctx, tx, org, pkg)
	if err != nil {
		return nil, nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text FROM submissions WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id::text=$3 ORDER BY received_on, created_at`, org, project, pkg)
	if err != nil {
		return nil, nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	cols := []FirmColumn{}
	earlier := []Submission{}
	var totals []proposals.Column
	for _, id := range ids {
		sub, err := readSubmission(ctx, tx, org, id)
		if err != nil {
			return nil, nil, err
		}
		if !liveColumn(sub.Status) {
			earlier = append(earlier, sub)
			continue
		}
		col := FirmColumn{Submission: sub, Cells: map[string]proposals.Cell{}, Others: []proposals.Cell{}}
		if sub.FirmID != "" {
			f, err := readFirm(ctx, tx, org, sub.FirmID)
			if err != nil {
				return nil, nil, err
			}
			col.Firm = &f
		}
		cells, err := tx.Query(ctx, `SELECT id::text, COALESCE(item_key,''), label, amount_cents, state, basis, page, passage, band, decided_by
FROM submission_prices WHERE org_id=$1::uuid AND submission_id::text=$2 ORDER BY ordinal, id`, org, id)
		if err != nil {
			return nil, nil, err
		}
		for cells.Next() {
			var c proposals.Cell
			var page int32
			if err := cells.Scan(&c.ID, &c.Key, &c.Label, &c.Cents, &c.State, &c.Basis, &page, &c.Passage, &c.Band, &c.DecidedBy); err != nil {
				cells.Close()
				return nil, nil, err
			}
			c.Page = int(page)
			// An unread cell is stored without a figure; show it as unread.
			if c.Band == proposals.BandBlank && c.DecidedBy == proposals.ByJev || c.Band == proposals.BandGrey {
				c.State = ""
				c.Cents = nil
			}
			if c.Key == "" {
				col.Others = append(col.Others, c)
				continue
			}
			col.Cells[c.Key] = c
		}
		cells.Close()
		if err := cells.Err(); err != nil {
			return nil, nil, err
		}
		if col.Decisions, err = submissionDecisions(ctx, tx, org, id); err != nil {
			return nil, nil, err
		}
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM comparison_alignment WHERE org_id=$1::uuid AND submission_id=$2::uuid AND version=$3)`, org, id, sub.Version).Scan(&col.UndoAlignment); err != nil {
			return nil, nil, err
		}
		col.Totals = proposals.Totals(items, col.Cells, col.Others, sub.StatedTotal)
		if !isBuilder {
			col.Quote, err = quotedReview(ctx, tx, org, id)
			if err != nil {
				return nil, nil, err
			}
			if _, confirmed := col.Cells["comparison:appointment"]; col.Quote != nil || confirmed {
				col.Totals = proposals.Totals([]rfp.ScheduleItem{{Key: "comparison:appointment", Kind: rfp.ItemStage, Base: true}}, col.Cells, nil, nil)
			}
		}
		if isBuilder {
			col.Builder, err = builderReview(ctx, tx, org, id)
			if err != nil {
				return nil, nil, err
			}
		}
		col.NeedsReview = needsReview(items, col)
		cols = append(cols, col)
		totals = append(totals, col.Totals)
	}
	hasQuoted := false
	for _, col := range cols {
		hasQuoted = hasQuoted || col.Quote != nil
	}
	if !isBuilder && !hasQuoted {
		proposals.MarkLowest(totals)
	}
	for i := range cols {
		cols[i].Totals = totals[i]
	}
	return cols, earlier, nil
}

// needsReview is a column a PM should look at before relying on it: a base
// cell unread or amber, a gap to the stated total, or a flag that changes
// how the price reads.
func needsReview(items []rfp.ScheduleItem, c FirmColumn) bool {
	if c.Submission.Status == SubmissionReading || c.Submission.Status == SubmissionNeedsReview || c.Totals.Gap != nil || c.Firm == nil || c.Firm.Status == "draft" {
		return true
	}
	for _, it := range items {
		if !it.Price() {
			continue
		}
		cell, ok := c.Cells[it.Key]
		if !ok || cell.State == "" || cell.Band == proposals.BandAmber || cell.Band == proposals.BandGrey {
			return true
		}
	}
	for _, f := range c.Submission.Flags {
		switch f {
		case proposals.FlagOptionsAB, proposals.FlagRatesOnly, proposals.FlagNoText, proposals.FlagTotalGap:
			return true
		}
	}
	return false
}

func readFirm(ctx context.Context, q querier, org, id string) (Firm, error) {
	var f Firm
	err := q.QueryRow(ctx, `SELECT id::text,name,COALESCE(abn,''),address,contact,email,phone,status,version FROM firms WHERE org_id=$1::uuid AND id::text=$2`, org, id).Scan(&f.ID, &f.Name, &f.ABN, &f.Address, &f.Contact, &f.Email, &f.Phone, &f.Status, &f.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, ErrNotFound
	}
	return f, err
}

// FirmCard is one firm under a package on the Procurement page.
type FirmCard struct {
	SubmissionID string   `json:"submission_id"`
	FirmName     string   `json:"firm_name"`
	Filename     string   `json:"filename"`
	ReceivedOn   string   `json:"received_on"`
	Status       string   `json:"status"`
	BaseTotal    *int64   `json:"-"`
	KnownBase    int64    `json:"-"`
	NotPriced    int      `json:"not_priced"`
	Unread       int      `json:"unread"`
	Lowest       bool     `json:"lowest"`
	NeedsReview  bool     `json:"needs_review"`
	Flags        []string `json:"flags"`
}

// ProcurementBoard is every package's live firm cards, and the submissions
// read but not yet tied to a package, in one snapshot.
func (s *Store) ProcurementBoard(ctx context.Context, org, project string) (map[string][]FirmCard, []FirmCard, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT DISTINCT package_id::text FROM submissions WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id IS NOT NULL`, org, project)
	if err != nil {
		return nil, nil, err
	}
	var pkgs []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			rows.Close()
			return nil, nil, err
		}
		pkgs = append(pkgs, p)
	}
	rows.Close()
	out := map[string][]FirmCard{}
	for _, pkg := range pkgs {
		items, err := readSchedule(ctx, tx, org, pkg)
		if err != nil {
			return nil, nil, err
		}
		var live []rfp.ScheduleItem
		for _, it := range items {
			if !it.Retired {
				live = append(live, it.ScheduleItem)
			}
		}
		cols, _, err := s.packageColumns(ctx, tx, org, project, pkg, live)
		if err != nil {
			return nil, nil, err
		}
		for _, c := range cols {
			out[pkg] = append(out[pkg], cardOf(c))
		}
	}
	loose := []FirmCard{}
	rows, err = tx.Query(ctx, `SELECT id::text FROM submissions WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id IS NULL AND status IN ('reading','received','needs_review') ORDER BY created_at`, org, project)
	if err != nil {
		return nil, nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		sub, err := readSubmission(ctx, tx, org, id)
		if err != nil {
			return nil, nil, err
		}
		c := FirmColumn{Submission: sub}
		if sub.FirmID != "" {
			if f, err := readFirm(ctx, tx, org, sub.FirmID); err == nil {
				c.Firm = &f
			}
		}
		card := cardOf(c)
		card.NeedsReview = true
		loose = append(loose, card)
	}
	return out, loose, tx.Commit(ctx)
}

func cardOf(c FirmColumn) FirmCard {
	card := FirmCard{SubmissionID: c.Submission.ID, Filename: c.Submission.Filename, ReceivedOn: c.Submission.ReceivedOn, Status: c.Submission.Status,
		BaseTotal: c.Totals.BaseTotal, KnownBase: c.Totals.KnownBase, NotPriced: c.Totals.NotPriced, Unread: c.Totals.Unread, Lowest: c.Totals.Lowest,
		NeedsReview: c.NeedsReview, Flags: stringsOrEmpty(c.Submission.Flags)}
	if c.Firm != nil {
		card.FirmName = c.Firm.Name
	}
	return card
}

// CellInput is a user's answer for one cell: an amount, not priced,
// included in another item, or excluded.
type CellInput struct {
	State  string `json:"state"`
	Amount string `json:"amount"`
	Note   string `json:"note"`
}

// SetSubmissionCell records a user's cell. The user's word is final: a
// later read keeps it. The decision keeps what the machine said.
func (s *Store) SetSubmissionCell(ctx context.Context, org, project, id, key string, in CellInput) error {
	if len(in.Note) > 500 {
		return ErrInvalidInput
	}
	var cents *int64
	switch in.State {
	case proposals.StatePriced:
		c, ok, err := invoices.ParseCents(in.Amount)
		if err != nil || !ok || c < 0 {
			return ErrInvalidInput
		}
		cents = &c
	case proposals.StateNotPriced, proposals.StateIncluded, proposals.StateExcluded:
	default:
		return ErrInvalidInput
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	initial, err := readSubmission(ctx, tx, org, id)
	if err != nil {
		return err
	}
	if initial.ProjectID != project {
		return ErrNotFound
	}
	if err = lockComparison(ctx, tx, org, project, initial.PackageID); err != nil {
		return err
	}
	sub, err := lockSubmission(ctx, tx, org, project, id)
	if err != nil {
		return err
	}
	if sub.Status != SubmissionReceived && sub.Status != SubmissionNeedsReview {
		return ErrConflict
	}
	if key == "comparison:appointment" {
		if strings.TrimSpace(in.Note) == "" {
			return ErrInvalidInput
		}
		_, err = tx.Exec(ctx, `INSERT INTO schedule_items(org_id,project_id,package_id,key,kind,label,detail,base,ordinal,source,comparison_custom) VALUES($1::uuid,$2::uuid,$3::uuid,'comparison:appointment','stage','Confirmed comparison fee','User-confirmed scope and amount; source breakdown is not additive',true,0,'generic',true) ON CONFLICT(org_id,package_id,key) DO NOTHING`, org, project, sub.PackageID)
		if err != nil {
			return err
		}
	}
	var label string
	err = tx.QueryRow(ctx, `SELECT label FROM schedule_items WHERE org_id=$1::uuid AND package_id::text=$2 AND key=$3`, org, sub.PackageID, key).Scan(&label)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	value := in.State
	if cents != nil {
		value = "cents:" + strconv.FormatInt(*cents, 10)
	}
	if err := upsertSubmissionDecision(ctx, tx, org, project, id, proposals.Decision{Field: proposals.PricePrefix + key, Value: value}, true); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO submission_prices (org_id,id,project_id,submission_id,item_key,label,amount_cents,state,band,decided_by,basis)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,'green','user',$9)
ON CONFLICT (org_id,submission_id,item_key) WHERE item_key IS NOT NULL DO UPDATE SET amount_cents=EXCLUDED.amount_cents, state=EXCLUDED.state,
 band='green', decided_by='user', basis=EXCLUDED.basis, updated_at=now()`, org, newID(), project, id, key, clipText(label, 500), cents, in.State, in.Note); err != nil {
		return err
	}
	return s.finishSubmissionWrite(ctx, tx, org, project, id)
}

func lockSubmission(ctx context.Context, tx pgx.Tx, org, project, id string) (Submission, error) {
	var got string
	err := tx.QueryRow(ctx, `SELECT id::text FROM submissions WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3 FOR UPDATE`, org, project, id).Scan(&got)
	if errors.Is(err, pgx.ErrNoRows) {
		return Submission{}, ErrNotFound
	}
	if err != nil {
		return Submission{}, err
	}
	return readSubmission(ctx, tx, org, id)
}

func (s *Store) finishSubmissionWrite(ctx context.Context, tx pgx.Tx, org, project, id string) error {
	if _, err := tx.Exec(ctx, `UPDATE submissions SET updated_at=now(), version=version+1 WHERE org_id=$1::uuid AND id::text=$2`, org, id); err != nil {
		return err
	}
	if err := procurementEvent(ctx, tx, s, org, project, id, "edited"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SubmissionEdit is a user's change to a submission. Nil leaves a field.
type SubmissionEdit struct {
	PackageID *string     `json:"package_id"`
	FirmID    *string     `json:"firm_id"`
	Firm      *FirmFields `json:"firm"`
	Status    *string     `json:"status"`
	Received  *string     `json:"received_on"`
}

// FirmFields are a firm's editable details.
type FirmFields struct {
	Name    string `json:"name"`
	ABN     string `json:"abn"`
	Address string `json:"address"`
	Contact string `json:"contact"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
}

// EditSubmission applies a user's change. Editing the firm's details
// confirms the directory firm; choosing a directory firm links it; a status
// may move between received and withdrawn only.
func (s *Store) EditSubmission(ctx context.Context, org, project, id string, e SubmissionEdit) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return err
	}
	sub, err := lockSubmission(ctx, tx, org, project, id)
	if err != nil {
		return err
	}
	if e.PackageID != nil {
		if *e.PackageID != "" {
			if err := packageInProject(ctx, tx, org, project, *e.PackageID); err != nil {
				return err
			}
		}
		if sub.Status == SubmissionAwarded {
			return ErrConflict
		}
		if _, err := tx.Exec(ctx, `UPDATE submissions SET package_id=NULLIF($3,'')::uuid WHERE org_id=$1::uuid AND id::text=$2`, org, id, *e.PackageID); err != nil {
			return err
		}
		if err := upsertSubmissionDecision(ctx, tx, org, project, id, proposals.Decision{Field: proposals.FieldPackage, Value: *e.PackageID}, true); err != nil {
			return err
		}
	}
	if e.FirmID != nil {
		if *e.FirmID != "" {
			if _, err := readFirm(ctx, tx, org, *e.FirmID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE submissions SET firm_id=NULLIF($3,'')::uuid WHERE org_id=$1::uuid AND id::text=$2`, org, id, *e.FirmID); err != nil {
			return err
		}
		sub.FirmID = *e.FirmID
	}
	if e.Firm != nil {
		if err := s.saveFirm(ctx, tx, org, project, id, sub.FirmID, *e.Firm); err != nil {
			return err
		}
	}
	if e.Status != nil {
		switch {
		case *e.Status == SubmissionWithdrawn && liveColumn(sub.Status) && sub.Status != SubmissionAwarded:
		case *e.Status == SubmissionReceived && (sub.Status == SubmissionWithdrawn || sub.Status == SubmissionNotProposal || sub.Status == SubmissionNeedsReview):
		default:
			return ErrInvalidInput
		}
		if _, err := tx.Exec(ctx, `UPDATE submissions SET status=$3 WHERE org_id=$1::uuid AND id::text=$2`, org, id, *e.Status); err != nil {
			return err
		}
	}
	if e.Received != nil {
		if _, err := time.Parse("2006-01-02", *e.Received); err != nil {
			return ErrInvalidInput
		}
		if _, err := tx.Exec(ctx, `UPDATE submissions SET received_on=$3::date WHERE org_id=$1::uuid AND id::text=$2`, org, id, *e.Received); err != nil {
			return err
		}
	}
	if err := supersedeEarlier(ctx, tx, org, project, id); err != nil {
		return err
	}
	return s.finishSubmissionWrite(ctx, tx, org, project, id)
}

// saveFirm writes the user's firm details: the linked firm is updated and
// confirmed, or a new confirmed firm is made and linked. An ABN already in
// the directory links that firm instead of making a second.
func (s *Store) saveFirm(ctx context.Context, tx pgx.Tx, org, project, sub, firmID string, f FirmFields) error {
	f.Name = strings.TrimSpace(f.Name)
	abn := invoices.NormalizeABN(f.ABN)
	if f.Name == "" || len(f.Name) > 200 || (strings.TrimSpace(f.ABN) != "" && (abn == "" || !invoices.ValidABN(abn))) || len(f.Address) > 500 || len(f.Contact) > 200 || len(f.Email) > 254 || len(f.Phone) > 50 {
		return ErrInvalidInput
	}
	if abn != "" {
		var other string
		err := tx.QueryRow(ctx, `SELECT id::text FROM firms WHERE org_id=$1::uuid AND abn=$2 AND id::text<>$3`, org, abn, firmID).Scan(&other)
		if err == nil {
			firmID = other
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	if firmID == "" {
		firmID = newID()
		if _, err := tx.Exec(ctx, `INSERT INTO firms (org_id,id,name,name_key,abn,address,contact,email,phone,status) VALUES ($1::uuid,$2::uuid,$3,$4,NULLIF($5,''),$6,$7,$8,$9,'confirmed')`,
			org, firmID, f.Name, nameKeyOrName(f.Name), abn, f.Address, f.Contact, f.Email, f.Phone); err != nil {
			return err
		}
	} else if _, err := tx.Exec(ctx, `UPDATE firms SET name=$3, name_key=$4, abn=NULLIF($5,''), address=$6, contact=$7, email=$8, phone=$9, status='confirmed', updated_at=now(), version=version+1
WHERE org_id=$1::uuid AND id::text=$2`, org, firmID, f.Name, nameKeyOrName(f.Name), abn, f.Address, f.Contact, f.Email, f.Phone); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE submissions SET firm_id=$3::uuid WHERE org_id=$1::uuid AND id::text=$2`, org, sub, firmID)
	return err
}

func nameKeyOrName(name string) string {
	if k := proposals.NameKey(name); k != "" {
		return k
	}
	return clipText(strings.ToLower(name), 200)
}

// ReadFirms is the org's directory, matching a query by name or ABN.
func (s *Store) ReadFirms(ctx context.Context, org, query string) ([]Firm, error) {
	q := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	rows, err := s.pool.Query(ctx, `SELECT id::text,name,COALESCE(abn,''),address,contact,email,phone,status,version FROM firms
WHERE org_id=$1::uuid AND (lower(name) LIKE $2 OR COALESCE(abn,'') LIKE $3) ORDER BY status='confirmed' DESC, name LIMIT 50`, org, q, strings.ReplaceAll(q, " ", ""))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Firm{}
	for rows.Next() {
		var f Firm
		if err := rows.Scan(&f.ID, &f.Name, &f.ABN, &f.Address, &f.Contact, &f.Email, &f.Phone, &f.Status, &f.Version); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// SubmissionFile is the proposal's file, for the viewer beside its fields.
func (s *Store) SubmissionFile(ctx context.Context, org, project, id string) (File, string, error) {
	var docID string
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(document_id::text,'') FROM submissions WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3`, org, project, id).Scan(&docID)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && docID == "" {
		return File{}, "", ErrNotFound
	}
	if err != nil {
		return File{}, "", err
	}
	doc, err := s.GetDocument(ctx, org, docID)
	if err != nil {
		return File{}, "", err
	}
	f, err := s.GetFile(ctx, org, doc.FileID)
	return f, doc.Filename, err
}

// SubmissionPassages is the proposal's sections, for the non-price report.
func (s *Store) SubmissionPassages(ctx context.Context, org, project, id string) ([]proposals.Passage, error) {
	rows, err := s.pool.Query(ctx, `SELECT ordinal, heading, page, body FROM submission_passages WHERE org_id=$1::uuid AND project_id=$2::uuid AND submission_id::text=$3 ORDER BY ordinal`, org, project, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []proposals.Passage
	for rows.Next() {
		var p proposals.Passage
		var ord, page int32
		if err := rows.Scan(&ord, &p.Heading, &page, &p.Body); err != nil {
			return nil, err
		}
		p.Ordinal, p.Page = int(ord), int(page)
		out = append(out, p)
	}
	return out, rows.Err()
}
