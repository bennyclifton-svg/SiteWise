package httpapi_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"sitewise/internal/delivery"
	"sitewise/internal/invoices"
	"sitewise/internal/llm"
	"sitewise/internal/store"
)

// monthlyProgram is the heavy fixture's Program: 20 stages of 49 rows (1,000
// rows, one in ten a milestone) and 2,000 links, as the Program bench.
func monthlyProgram() store.ProgramCreate {
	in := store.ProgramCreate{Surface: "program", SkeletonID: "sk.generic", LibraryVersion: "bench", StartDate: "2026-11-02", StartOrigin: "user"}
	key := func(s, j int) string { return fmt.Sprintf("a%02d.%02d", s, j) }
	stage := func(s int) string { return fmt.Sprintf("s%02d", s) }
	for s := 0; s < 20; s++ {
		in.Rows = append(in.Rows, store.ProgramRowInput{Key: stage(s), Kind: "stage", Label: fmt.Sprintf("Stage %d", s+1), CatalogueStageID: "other", ReviewStatus: "proposed", Origin: "library"})
		for j := 0; j < 49; j++ {
			row := store.ProgramRowInput{Key: key(s, j), ParentKey: stage(s), Kind: "activity", Label: fmt.Sprintf("Activity %d.%d", s+1, j+1), ReviewStatus: "proposed", Origin: "library"}
			if j%10 == 9 {
				zero := 0
				row.Kind, row.Label, row.DurationDays, row.DurationOrigin = "milestone", fmt.Sprintf("Milestone %d.%d", s+1, j+1), &zero, "user"
			} else {
				d := 1 + (s*7+j)%10
				row.DurationDays, row.DurationOrigin = &d, "user"
			}
			in.Rows = append(in.Rows, row)
		}
	}
	add := func(pred, succ, typ, origin string) bool {
		if len(in.Links) == 2000 {
			return false
		}
		in.Links = append(in.Links, store.ProgramLinkInput{PredKey: pred, SuccKey: succ, Type: typ, Origin: origin})
		return true
	}
	for s := 1; s < 20; s++ {
		add(stage(s-1), stage(s), "FS", "chain")
	}
	for s := 0; s < 20; s++ {
		for j := 1; j < 49; j++ {
			add(key(s, j-1), key(s, j), "FS", "chain")
		}
	}
	for s := 1; s < 20; s++ {
		for j := 0; j < 49; j++ {
			add(key(s-1, j), key(s, j), "FS", "user")
		}
	}
	for s := 0; s < 20; s++ {
		for j := 2; j < 49; j++ {
			add(key(s, j-2), key(s, j), "SS", "user")
		}
	}
	return in
}

// TestProjectReportBudget holds report_assemble_monthly (monthly plan,
// Task 8) on the plan's heavy fixture: 1,000 Program rows and 2,000 links,
// 1,000 invoices, 30 packages, 200 register items and a previous issue. It
// samples report_read, report_write, report_issue and report_export on the
// same report, so those gates cover it too. Twenty rounds, so a query the
// database turns to a cached generic plan after five calls would show.
func TestProjectReportBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("heavy fixture")
	}
	ctx := context.Background()
	a, m, project, _ := chatApp(t, func(int, llm.Conversation) llm.Turn { return llm.Turn{Text: "ok"} })
	for i := 0; i < 30; i++ {
		body, _ := json.Marshal(map[string]string{"kind": "services", "title": fmt.Sprintf("Consultant %02d", i)})
		m.call(t, http.MethodPost, "/api/projects/"+project+"/packages", body, http.StatusCreated, nil)
	}
	var ready struct {
		Plan struct {
			Items []struct {
				ID      string `json:"cost_item_id"`
				Posting bool   `json:"posting"`
			} `json:"items"`
		} `json:"plan"`
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/cost-plan/prepare", []byte(`{}`), http.StatusOK, &ready)
	var lines []string
	for _, it := range ready.Plan.Items {
		if it.Posting {
			lines = append(lines, it.ID)
		}
	}
	if len(lines) < 2 {
		t.Fatalf("cost plan has %d lines", len(lines))
	}
	if _, err := a.store.CreateProgram(ctx, m.orgID, project, monthlyProgram()); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for i := 0; i < 1000; i++ {
		sum := sha256.Sum256([]byte(fmt.Sprintf("monthly bench %s %d", project, i)))
		created, err := a.store.CreateInvoice(ctx, m.orgID, project, store.InvoiceUpload{SHA256: sum[:], Size: 1000, MediaType: "application/pdf", Filename: fmt.Sprintf("INV-%05d.pdf", i)}, now)
		if err != nil {
			t.Fatal(err)
		}
		ex := int64(100000 + i*137)
		d := func(f, v string) invoices.Decision {
			return invoices.Decision{Field: f, Value: v, Band: invoices.BandGreen, DecidedBy: invoices.DecidedByRule}
		}
		out := invoices.Outcome{InvoiceCount: 1, Decisions: []invoices.Decision{d(invoices.FieldKind, "invoice"), d(invoices.FieldNumber, fmt.Sprintf("INV-%05d", i)),
			d(invoices.FieldDate, "2026-10-01"), d(invoices.FieldTotal, strconv.FormatInt(ex+invoices.GSTOf(ex), 10)), d(invoices.FieldGST, strconv.FormatInt(invoices.GSTOf(ex), 10)),
			d(invoices.FieldLine, lines[i%len(lines)]), d(invoices.FieldPeriod, invoices.PeriodOf(now)), d(invoices.FieldCurrency, "AUD")}}
		if _, err := a.store.CommitInvoiceRead(ctx, m.orgID, created.ID, out, false, now); err != nil {
			t.Fatal(err)
		}
	}
	kinds := []string{"risk", "issue", "decision", "approval", "action"}
	for i := 0; i < 200; i++ {
		kind := kinds[i%len(kinds)]
		target := now.AddDate(0, 0, i%60-20).Format("2006-01-02")
		c := delivery.Content{Kind: kind, Title: fmt.Sprintf("%s %d", kind, i), Status: delivery.DefaultStatus(kind), TargetDate: &target, OwnerText: "Architect", Details: json.RawMessage(`{}`)}
		body, _ := json.Marshal(c)
		m.call(t, http.MethodPost, "/api/projects/"+project+"/delivery", body, http.StatusCreated, nil)
	}
	m.ops(t, project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	report := projectReportID(t, m, project)
	base := "/api/reports/" + report
	issue := func(version int64) store.IssuedReport {
		body, _ := json.Marshal(store.IssueOptions{Version: version, ReportingDate: now.Format("2006-01-02"), AcceptStale: true, StaleReason: "Benchmark fixture"})
		var out store.IssuedReport
		m.call(t, http.MethodPost, base+"/issue", body, http.StatusCreated, &out)
		return out
	}
	var first store.ReportView
	m.getJSON(t, base, &first)
	if first.PageFit == nil || !first.PageFit.Fits {
		t.Fatalf("the heavy report does not fit: %+v", first.PageFit)
	}
	issue(first.Draft.Version)

	samples := map[string][]int64{}
	timed := func(name string, f func()) {
		start := time.Now()
		f()
		samples[name] = append(samples[name], time.Since(start).Microseconds())
	}
	for i := 0; i < 20; i++ {
		var d store.ReportDraft
		timed("report_assemble_monthly", func() {
			m.call(t, http.MethodPost, base+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK, &d)
		})
		var view store.ReportView
		timed("report_read", func() { m.getJSON(t, base, &view) })
		if view.PageFit == nil || !view.PageFit.Fits {
			t.Fatalf("round %d does not fit: %+v", i, view.PageFit)
		}
		version := view.Draft.Version
		// Every carried or stale block is kept: one save each, as the PM does.
		for _, s := range view.Draft.Sections {
			for _, b := range s.Blocks {
				if !b.Conflict {
					continue
				}
				body, _ := json.Marshal(map[string]any{"text": b.Text, "version": version})
				var saved store.ReportDraft
				timed("report_write", func() {
					m.call(t, http.MethodPut, base+"/edits/"+url.PathEscape(b.ID), body, http.StatusOK, &saved)
				})
				version = saved.Version
			}
		}
		body, _ := json.Marshal(map[string]any{"text": fmt.Sprintf("Round %d: as the cost plan shows.", i), "version": version})
		var saved store.ReportDraft
		timed("report_write", func() {
			m.call(t, http.MethodPut, base+"/edits/"+url.PathEscape("comment:summary"), body, http.StatusOK, &saved)
		})
		var issued store.IssuedReport
		timed("report_issue", func() { issued = issue(saved.Version) })
		timed("report_export", func() {
			if got := m.status(t, http.MethodGet, base+"/versions/"+issued.ID+"/file", nil); got != http.StatusOK {
				t.Fatalf("download %d", got)
			}
		})
	}
	gateTender(t, []string{"report_assemble_monthly", "report_read", "report_write", "report_issue", "report_export"}, samples)
}
