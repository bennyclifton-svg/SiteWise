package store

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/costs"
	"sitewise/internal/db"
	"sitewise/internal/delivery"
	"sitewise/internal/knowledge"
	engine "sitewise/internal/program"
	"sitewise/internal/reports"
)

// The project report's loader (docs/plans/2026-11-17-monthly-report.md,
// Task 2). Every figure is read here, on the draft's repeatable-read
// snapshot, through the functions the screens use: the cost plan and
// costs.Summarize, the invoice ledger, the Program engine and the
// procurement board. Nothing here is a second query that could disagree
// with a screen, and nothing calls Jev or a model.

func readMonthlyReport(ctx context.Context, tx pgx.Tx, org, project string) (Report, error) {
	var r Report
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT to_jsonb(r) FROM reports r WHERE org_id=$1::uuid AND project_id=$2::uuid AND kind='monthly'`, org, project).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	return r, json.Unmarshal(raw, &r)
}

// MonthlyReport is the project's project report, or ErrNotFound.
func (s *Store) MonthlyReport(ctx context.Context, org, project string) (Report, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return Report{}, err
	}
	defer tx.Rollback(ctx)
	return readMonthlyReport(ctx, tx, org, project)
}

func (s *Store) monthlySectionsTx(ctx context.Context, tx pgx.Tx, org string, r Report, history []ReportIssueSummary, prior []reports.Section, number int, edits []reports.Edit, appBuild string, template knowledge.ReportTemplate) ([]reports.Section, reports.SourceState, error) {
	in, state, err := s.monthlyInputTx(ctx, tx, org, r.ProjectID, history, prior, number, appBuild, template)
	if err != nil {
		return nil, state, err
	}
	titles := map[string]string{}
	for _, section := range template.Sections {
		titles[section.ID] = section.Title
	}
	sections, err := reports.AssembleMonthly(in, titles, edits)
	return sections, state, err
}

// monthlyInputTx reads everything one draft prints. Every query is scoped to
// the org and the project.
func (s *Store) monthlyInputTx(ctx context.Context, tx pgx.Tx, org, project string, history []ReportIssueSummary, prior []reports.Section, number int, appBuild string, template knowledge.ReportTemplate) (reports.MonthlyInput, reports.SourceState, error) {
	var state reports.SourceState
	if s.profileBuild == nil {
		return reports.MonthlyInput{}, state, ErrInvalidReport
	}
	sydney := sydneyZone()
	today := time.Now().In(sydney).Format("2006-01-02")
	in := reports.MonthlyInput{AsAt: today, Number: number, Received: map[string]string{}}
	in.Attention.Today = today
	if err := tx.QueryRow(ctx, `SELECT name FROM projects WHERE org_id=$1::uuid AND id=$2::uuid`, org, project).Scan(&in.ProjectName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return in, state, ErrNotFound
		}
		return in, state, err
	}
	if len(history) > 0 {
		in.Previous = &reports.MonthlyPrevious{Number: history[0].Number, Date: history[0].ReportingDate, Sections: prior}
		in.Attention.LastReport = reports.LastCompletion(history[0].Number, prior)
	}
	v, err := readProfileTx(ctx, tx, org, project, s.profileBuild.ReadKinds)
	if err != nil {
		return in, state, err
	}
	in.PendingReadings, in.FailedReadings = v.PendingDocuments, v.FailedDocuments
	state = s.reportSourceState(v, template, appBuild)
	if err := monthlyDomains(ctx, tx, org, project, &state); err != nil {
		return in, state, err
	}

	// Cost: the plan and costs.Summarize, as the Cost Plan grid's totals.
	plan, err := readCostPlan(ctx, tx, org, project, "")
	if err != nil {
		return in, state, err
	}
	if len(plan.Items) > 0 {
		overall, err := costs.Summarize(plan, "overall")
		if err != nil {
			return in, state, err
		}
		byPackage, err := costs.Summarize(plan, "package")
		if err != nil {
			return in, state, err
		}
		in.Costs, in.PackageCosts = &overall, &byPackage
		in.Attention.Costs = &overall
	}
	state.CostPlanVersionID = plan.ID

	// Claimed to date, remaining and Unallocated: the invoice ledger.
	ledger, err := s.invoiceLedgerTx(ctx, tx, org, project, today)
	if err != nil {
		return in, state, err
	}
	in.Ledger = &reports.AttentionLedger{Invoices: ledger.Invoices, Status: ledger.Status, Summary: ledger.Summary, Lines: ledger.Lines, Groups: ledger.Groups}
	in.Attention.Ledger = in.Ledger
	for _, inv := range ledger.Invoices {
		in.Received[inv.ID] = inv.CreatedAt.In(sydney).Format("2006-01-02")
	}

	// Program: the engine over the stored inputs, as LoadProgram.
	p, acts, links, days, err := readProgramInputs(ctx, tx, org, project, false)
	switch {
	case errors.Is(err, ErrNotFound):
	case err != nil:
		return in, state, err
	default:
		st := computeState(p, acts, links, days)
		in.Program = MonthlyProgramOf(st)
		in.Attention.Completion = reports.CompletionOf(in.Program)
		for _, a := range st.Activities {
			in.Attention.Program = append(in.Attention.Program, reports.AttentionActivity{ID: a.ID, Kind: a.Kind, DurationDays: a.DurationDays})
		}
	}

	if err := s.monthlyPackagesTx(ctx, tx, org, project, &in); err != nil {
		return in, state, err
	}

	// The register's rows, read as the Register screen and chat read them.
	register, err := s.readRegisterTx(ctx, tx, org, project, RegisterQuery{}, RegisterView{Items: []delivery.Item{}, Kinds: []RegisterKind{}, Systems: []RegisterSystem{}, Today: today})
	if err != nil {
		return in, state, err
	}
	in.Delivery = register.Items
	for _, it := range in.Delivery {
		in.Attention.Delivery = append(in.Attention.Delivery, reports.AttentionRecordOf(it))
	}

	if err := monthlyDocumentsTx(ctx, tx, s.q.WithTx(tx), org, project, sydney, &in); err != nil {
		return in, state, err
	}

	implications, err := readProposals(ctx, tx, org, project)
	if err != nil {
		return in, state, err
	}
	for _, it := range implications {
		in.Attention.Implications = append(in.Attention.Implications, reports.AttentionImplication{Key: it.Key, Label: it.Label, Rank: it.Rank,
			Severity: it.Severity, Critical: it.Critical, Decided: it.Decision != nil})
	}
	return in, state, nil
}

// monthlyPackagesTx reads the live packages, the procurement board's firm
// cells, the live awards and each package's latest issued RFP or RFT.
func (s *Store) monthlyPackagesTx(ctx context.Context, tx pgx.Tx, org, project string, in *reports.MonthlyInput) error {
	pkgs, err := readPackages(ctx, tx, org, project)
	if err != nil {
		return err
	}
	board, _, err := s.procurementBoardTx(ctx, tx, org, project)
	if err != nil {
		return err
	}
	type award struct{ firm, on string }
	awards := map[string]award{}
	rows, err := tx.Query(ctx, `SELECT a.package_id::text, COALESCE(f.name, s.filename), a.created_at
FROM awards a JOIN submissions s ON s.org_id=a.org_id AND s.id=a.submission_id LEFT JOIN firms f ON f.org_id=s.org_id AND f.id=s.firm_id
WHERE a.org_id=$1::uuid AND a.project_id=$2::uuid AND a.undone_at IS NULL`, org, project)
	if err != nil {
		return err
	}
	for rows.Next() {
		var pkg, firm string
		var at time.Time
		if err := rows.Scan(&pkg, &firm, &at); err != nil {
			rows.Close()
			return err
		}
		awards[pkg] = award{firm, SydneyDate(at)}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	issued := map[string]string{}
	rows, err = tx.Query(ctx, `SELECT r.package_id::text, max(v.reporting_date)::text FROM reports r
JOIN report_versions v ON v.org_id=r.org_id AND v.report_id=r.id AND v.status='issued'
WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.kind IN ('rfp','rft') AND r.package_id IS NOT NULL GROUP BY r.package_id`, org, project)
	if err != nil {
		return err
	}
	for rows.Next() {
		var pkg, on string
		if err := rows.Scan(&pkg, &on); err != nil {
			rows.Close()
			return err
		}
		issued[pkg] = on
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, p := range pkgs {
		if p.RetiredAt != nil {
			continue
		}
		// The attention rules read every live package, as chat's read does.
		ap := reports.AttentionPackage{ID: p.ID, Title: p.Title}
		for _, c := range board[p.ID] {
			ap.Firms = append(ap.Firms, reports.AttentionFirm{Quoted: c.SubmissionID != "", NeedsReview: c.NeedsReview})
		}
		in.Attention.Packages = append(in.Attention.Packages, ap)
		// The report's table shows a combined engagement once, as its group
		// row, and leaves cancelled rows out.
		if p.GroupID != "" || p.LifecycleStatus == "cancelled" {
			continue
		}
		invited, quoted, review := reports.PackageQuotes(ap.Firms)
		a := awards[p.ID]
		in.Packages = append(in.Packages, reports.MonthlyPackage{ID: p.ID, Title: p.Title, Kind: p.Kind, Lifecycle: p.LifecycleStatus,
			Invited: invited, Quoted: quoted, ToCheck: review, AwardedTo: a.firm, AwardedOn: a.on, IssuedOn: issued[p.ID]})
	}
	return nil
}

// monthlyDocumentsTx reads the project's filings with their discipline, as
// the repository lists them.
func monthlyDocumentsTx(ctx context.Context, tx pgx.Tx, q *db.Queries, org, project string, sydney *time.Location, in *reports.MonthlyInput) error {
	docs, err := q.ListProjectDocumentViews(ctx, db.ListProjectDocumentViewsParams{OrgID: org, ProjectID: project})
	if err != nil {
		return err
	}
	ids := make([]string, len(docs))
	for i, d := range docs {
		ids[i] = d.ID
	}
	discipline := map[string]string{}
	if len(ids) > 0 {
		fields, err := q.ListDocumentDecisionViews(ctx, db.ListDocumentDecisionViewsParams{OrgID: org, DocumentIds: ids})
		if err != nil {
			return err
		}
		for _, f := range fields {
			if f.Field == "discipline" {
				discipline[f.DocumentID] = f.Value
			}
		}
	}
	for _, d := range docs {
		in.Documents = append(in.Documents, reports.MonthlyDocument{ID: d.ID, Received: d.CreatedAt.In(sydney).Format("2006-01-02"), Discipline: discipline[d.ID], Status: d.Status})
		in.Attention.Documents = append(in.Attention.Documents, reports.AttentionDocument{ID: d.ID, Status: d.Status})
	}
	return nil
}

// MonthlyProgramOf is the Program as the report and the attention rule read
// it: the engine's dates, with the labels the Program screen shows (dates,
// or weeks on a Program with no start date).
func MonthlyProgramOf(st ProgramState) *reports.MonthlyProgram {
	c := st.Computed
	out := &reports.MonthlyProgram{Cycle: len(c.Cycle) > 0, Complete: c.Finish.Complete, Known: c.Status.Known, Total: c.Status.Total}
	in := st.Input()
	cal := engine.NewCalendar(in.Calendar)
	out.Relative = cal.Relative()
	label := func(date *string, index *int) (string, string) {
		if index == nil {
			return "", ""
		}
		if out.Relative {
			return "", cal.WeekLabel(*index - 1)
		}
		if date == nil {
			return "", ""
		}
		return *date, reports.LongDate(*date)
	}
	if !out.Cycle {
		out.Typical, out.TypicalLabel = label(c.Finish.Typical, c.Finish.TypicalIndex)
		out.Low, out.LowLabel = label(c.Finish.Low, c.Finish.LowIndex)
		out.High, out.HighLabel = label(c.Finish.High, c.Finish.HighIndex)
		out.TypicalIndex = c.Finish.TypicalIndex
	}
	for i, a := range st.Activities {
		row := reports.MonthlyActivity{ID: a.ID, StageID: string(a.ParentID), Kind: a.Kind, Label: a.Label, Unknown: true}
		if i < len(c.Activities) && !out.Cycle {
			t := c.Activities[i]
			row.Critical, row.Unknown = t.Critical, t.Unknown
			if !t.Unknown {
				if out.Relative {
					row.StartLabel = cal.WeekLabel(t.ES)
					row.FinishLabel = cal.WeekLabel(max(t.EF-1, t.ES))
				} else {
					row.Start, row.Finish = t.Start, t.Finish
					row.StartLabel, row.FinishLabel = reports.LongDate(t.Start), reports.LongDate(t.Finish)
				}
			}
		}
		out.Activities = append(out.Activities, row)
	}
	return out
}

// monthlyDomains adds the two source stamps the project report depends on
// beyond the shared revision counters: the Program's version, and the
// invoices' count plus the sum of their versions (any new or edited invoice
// moves it). No new revision domain.
func monthlyDomains(ctx context.Context, tx pgx.Tx, org, project string, state *reports.SourceState) error {
	if state.Domains == nil {
		state.Domains = map[string]int64{}
	}
	var program, invoices int64
	err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT version FROM programs WHERE org_id=$1::uuid AND project_id=$2::uuid),0),
 (SELECT count(*)+COALESCE(sum(version),0) FROM invoices WHERE org_id=$1::uuid AND project_id=$2::uuid)`, org, project).Scan(&program, &invoices)
	if err != nil {
		return err
	}
	state.Domains["program"], state.Domains["invoices"] = program, invoices
	return nil
}

// currentSourceState is what a draft's saved state is compared with.
func (s *Store) currentSourceState(ctx context.Context, tx pgx.Tx, org string, r Report, v ProfileView, template knowledge.ReportTemplate, appBuild string) (reports.SourceState, error) {
	state := s.reportSourceState(v, template, appBuild)
	if r.Kind != reports.KindMonthly {
		return state, nil
	}
	return state, monthlyDomains(ctx, tx, org, r.ProjectID, &state)
}

// LastReportCompletion is the planned completion the project's last issued
// project report printed, for the attention rule; nil when there is none.
func (s *Store) LastReportCompletion(ctx context.Context, org, project string) (*reports.AttentionCompletion, error) {
	var number int
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT v.number, v.sections FROM report_versions v JOIN reports r ON r.org_id=v.org_id AND r.id=v.report_id
WHERE v.org_id=$1::uuid AND v.project_id=$2::uuid AND r.kind='monthly' AND v.status='issued' ORDER BY v.number DESC LIMIT 1`, org, project).Scan(&number, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var sections []reports.Section
	if err := json.Unmarshal(raw, &sections); err != nil {
		return nil, err
	}
	return reports.LastCompletion(number, sections), nil
}

// sydneyZone loads the zone once: the loader turns a thousand invoice times
// into Sydney dates on every Update.
var sydneyZone = sync.OnceValue(func() *time.Location {
	loc, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		return time.UTC
	}
	return loc
})
