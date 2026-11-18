package httpapi

import (
	"context"
	"errors"
	"sync"

	"sitewise/internal/costs"
	"sitewise/internal/procurement"
	"sitewise/internal/reports"
	"sitewise/internal/store"
)

// attentionPart names the records loadAttention reads, so a named filter
// reads only what its rule needs and chat's attention read reads them all.
type attentionPart int

const (
	attentionImplications attentionPart = 1 << iota
	attentionCosts
	attentionLedger
	attentionPackages
	attentionDocuments
	attentionProgram
	attentionDelivery
	// attentionReport is the planned completion the last issued project
	// report printed; with attentionProgram it feeds the completion rule.
	attentionReport

	attentionAll = attentionImplications | attentionCosts | attentionLedger | attentionPackages | attentionDocuments | attentionProgram | attentionDelivery | attentionReport
)

// loadAttention reads the records reports.Attention needs, each read at the
// same time as the others, so the read takes as long as its slowest record
// and not their sum (chat_tool_read budget). A record that is not there
// (no cost plan, no Program) is left empty. The first other error is
// returned with whatever was read; chat's attention read shows what it has
// and a filter refuses.
func loadAttention(ctx context.Context, d Deps, who opActor, project string, parts attentionPart) (reports.AttentionInput, error) {
	in := reports.AttentionInput{Today: todaySydney()}
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		first error
	)
	run := func(part attentionPart, read func() error) {
		if parts&part == 0 {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := read(); err != nil && !errors.Is(err, store.ErrNotFound) {
				mu.Lock()
				if first == nil {
					first = err
				}
				mu.Unlock()
			}
		}()
	}
	// Each read writes only its own field of in.
	run(attentionImplications, func() error {
		items, err := d.Store.ReadProposals(ctx, who.OrgID, project)
		for _, it := range items {
			in.Implications = append(in.Implications, reports.AttentionImplication{Key: it.Key, Label: it.Label, Rank: it.Rank,
				Severity: it.Severity, Critical: it.Critical, Decided: it.Decision != nil})
		}
		return err
	})
	run(attentionCosts, func() error {
		p, err := d.Store.ReadCostPlan(ctx, who.OrgID, project, "")
		if err != nil || len(p.Items) == 0 {
			return err
		}
		totals, err := costs.Summarize(p, "overall")
		if err == nil {
			in.Costs = &totals
		}
		return err
	})
	run(attentionLedger, func() error {
		v, err := d.Store.InvoiceLedger(ctx, who.OrgID, project, todaySydney())
		if err == nil {
			in.Ledger = &reports.AttentionLedger{Invoices: v.Invoices, Status: v.Status, Summary: v.Summary, Lines: v.Lines, Groups: v.Groups}
		}
		return err
	})
	run(attentionPackages, func() error {
		pkgs, board, err := readPackageBoard(ctx, d, who, project)
		for _, p := range pkgs {
			if p.RetiredAt != nil {
				continue
			}
			ap := reports.AttentionPackage{ID: p.ID, Title: p.Title}
			for _, c := range board[p.ID] {
				ap.Firms = append(ap.Firms, reports.AttentionFirm{Quoted: c.SubmissionID != "", NeedsReview: c.NeedsReview})
			}
			in.Packages = append(in.Packages, ap)
		}
		return err
	})
	run(attentionDocuments, func() error {
		views, err := d.Store.ProjectDocumentViews(ctx, who.OrgID, project)
		for _, v := range views {
			in.Documents = append(in.Documents, reports.AttentionDocument{ID: v.ID, Status: v.Status})
		}
		return err
	})
	run(attentionProgram, func() error {
		st, err := d.Store.LoadProgram(ctx, who.OrgID, project)
		for _, a := range st.Activities {
			in.Program = append(in.Program, reports.AttentionActivity{ID: a.ID, Kind: a.Kind, DurationDays: a.DurationDays})
		}
		if err == nil {
			in.Completion = reports.CompletionOf(store.MonthlyProgramOf(st))
		}
		return err
	})
	run(attentionReport, func() error {
		last, err := d.Store.LastReportCompletion(ctx, who.OrgID, project)
		in.LastReport = last
		return err
	})
	run(attentionDelivery, func() error {
		items, err := d.Store.ReadDelivery(ctx, who.OrgID, project)
		for _, it := range items {
			if it.RetiredAt != nil {
				continue
			}
			target := ""
			if it.TargetDate != nil {
				target = *it.TargetDate
			}
			in.Delivery = append(in.Delivery, reports.AttentionRecord{ID: it.ID, Kind: it.Kind, Status: it.Status, TargetDate: target})
		}
		return err
	})
	wg.Wait()
	return in, first
}

// readPackageBoard reads the packages and the procurement board at the
// same time.
func readPackageBoard(ctx context.Context, d Deps, who opActor, project string) ([]procurement.Package, map[string][]store.FirmCard, error) {
	var (
		board    map[string][]store.FirmCard
		boardErr error
		done     = make(chan struct{})
	)
	go func() {
		defer close(done)
		board, _, boardErr = d.Store.ProcurementBoard(ctx, who.OrgID, project)
	}()
	pkgs, err := d.Store.ReadPackages(ctx, who.OrgID, project)
	<-done
	if err == nil {
		err = boardErr
	}
	return pkgs, board, err
}

// attentionRule is the rows of one single-item rule: the named filter that
// computes the same rule reads it here, so the filter and the attention
// list cannot disagree.
func attentionRule(parts attentionPart, rule string) func(context.Context, Deps, opActor, string) ([]string, error) {
	return func(ctx context.Context, d Deps, who opActor, project string) ([]string, error) {
		in, err := loadAttention(ctx, d, who, project, parts)
		if err != nil {
			return nil, err
		}
		it, _ := reports.AttentionFor(reports.Attention(in), rule)
		return it.IDs, nil
	}
}
