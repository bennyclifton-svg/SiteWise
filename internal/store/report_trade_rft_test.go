package store_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"sitewise/internal/procurement"
	"sitewise/internal/reports"
	"sitewise/internal/store"
)

// A trade RFT is built by the RFP engine in the RFP format (owner, 8
// November 2026): Main Works opens with editor controls and section 2
// "Scope of works", a trade let separately leaves Main Works as "By
// separate contract", a trade's tender price rows become its schedule,
// and a consultant with no discipline file of its own still gets the new
// RFP format. Another org sees nothing.
func TestTradeRFTUsesTheRFPFormat(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	for key, value := range map[string]string{"hdr.building_class": "residential", "hdr.subclass": "house", "hdr.work_type": "new"} {
		v := value
		if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, key, store.UserWrite{Scope: "project", Value: &v, State: "set"}); err != nil {
			t.Fatal(key, err)
		}
	}
	create := func(id, label string) procurement.Package {
		t.Helper()
		row, ok := procurement.DisciplinePackage(id, label)
		if !ok {
			t.Fatalf("%s has no package", id)
		}
		p, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
		if err != nil {
			t.Fatal(id, err)
		}
		return p
	}
	open := func(p procurement.Package) (store.Report, store.ReportView) {
		t.Helper()
		_, r, _, err := s.ProcurementCreate(ctx, orgA, projectA, userA, p.ID, "", "")
		if err != nil {
			t.Fatal(p.Title, err)
		}
		if _, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test", true); err != nil {
			t.Fatal(p.Title, err)
		}
		v, err := s.ReadReport(ctx, orgA, r.ID, "test")
		if err != nil {
			t.Fatal(p.Title, err)
		}
		if v.RFP == nil || v.Draft == nil {
			t.Fatalf("%s opened in the old layout: %+v", p.Title, v.Report)
		}
		return r, v
	}
	text := func(v store.ReportView) string {
		var b strings.Builder
		for _, sec := range v.Draft.Sections {
			b.WriteString(sec.Title + "\n")
			for _, block := range sec.Blocks {
				b.WriteString(block.Text + "\n")
				if block.Table != nil {
					raw, _ := json.Marshal(block.Table.Rows)
					b.Write(raw)
				}
			}
		}
		return b.String()
	}

	works := create("trade.main_works", "Main Works")
	r, v := open(works)
	if r.Kind != "rft" || v.RFP.Discipline != "main-works" {
		t.Fatalf("main works report %+v discipline %q", r, v.RFP.Discipline)
	}
	body := text(v)
	for _, want := range []string{"2. Scope of works", "4. Terms", "Preliminaries and site establishment", "Mechanical services", "Tender price", "R1 Tender price"} {
		if !strings.Contains(body, want) {
			t.Fatalf("main works RFT lacks %q:\n%s", want, body)
		}
	}
	// It issues on the RFP's page allowance, not the old generic two pages.
	if v.PageFit == nil || v.PageFit.Limit != 4 || !v.PageFit.Fits {
		t.Fatalf("main works page fit %+v", v.PageFit)
	}
	for _, unwanted := range []string{"Scope of services", "Total fee", "Reimbursables", "Professional indemnity"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("main works RFT prints consultant wording %q", unwanted)
		}
	}

	// Mechanical let separately: Main Works prints it as by that contract.
	mech := create("trade.mechanical", "Mechanical Services")
	if _, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test", true); err != nil {
		t.Fatal(err)
	}
	v, err := s.ReadReport(ctx, orgA, r.ID, "test")
	if err != nil {
		t.Fatal(err)
	}
	body = text(v)
	if !strings.Contains(body, "By separate contract: Mechanical Services") || !strings.Contains(body, "The client lets these separately: Mechanical Services.") {
		t.Fatalf("separate mechanical contract not shown:\n%s", body)
	}
	mr, mv := open(mech)
	if mv.RFP.Discipline != "trade-mechanical" {
		t.Fatalf("mechanical discipline %q", mv.RFP.Discipline)
	}
	// Its Excel response form asks for a tender in tender words, on the
	// sheet and keys the proposal reader already knows.
	form, err := s.ReadResponseForm(ctx, orgA, mr.ID)
	if err != nil {
		t.Fatal(err)
	}
	var cells strings.Builder
	sheets := form.Sheets()
	for _, sheet := range sheets {
		cells.WriteString("[" + sheet.Name + "]")
		for _, row := range sheet.Rows {
			for _, c := range row {
				cells.WriteString(c.Text + "|")
			}
		}
	}
	for _, want := range []string{"[R1 Fees]", "Price $ excl. GST", "Tender price total", "RFT", "Contractor licence", "[R9 Subcontractors]", "Provisional sums and prime cost items"} {
		if !strings.Contains(cells.String(), want) {
			t.Fatalf("response form lacks %q: %s", want, cells.String())
		}
	}
	if strings.Contains(cells.String(), "Professional indemnity") || strings.Contains(cells.String(), "Fee $") {
		t.Fatalf("response form asks a tenderer for consultant items: %s", cells.String())
	}

	// A trade's tender price rows are its package schedule. (The head
	// contract keeps its one-lump-sum builder comparison.)
	items, err := s.PackageSchedule(ctx, orgA, projectA, mech.ID)
	if err != nil {
		t.Fatal(err)
	}
	prices := 0
	for _, it := range items {
		if it.Price() && !it.Retired && it.Source == store.ScheduleFromRFP {
			prices++
		}
	}
	if prices < 3 {
		t.Fatalf("schedule %+v", items)
	}

	// A supply and a consultant without a file use the general files.
	if _, sv := open(create("supplier.windows_glazing", "Windows and Glazing")); sv.RFP.Discipline != "trade-general" || !strings.Contains(text(sv), "Trade package: Windows and Glazing.") {
		t.Fatalf("supply RFT %q", sv.RFP.Discipline)
	}
	acoustic := create("consultant.acoustic", "Acoustic")
	if _, av := open(acoustic); av.RFP.Discipline != "consultant-general" || !strings.Contains(text(av), "Your job is to give the project") || !strings.Contains(text(av), "2. Scope of services") {
		t.Fatalf("acoustic RFP %q:\n%s", av.RFP.Discipline, text(av))
	}

	if _, err := s.ReadReport(ctx, orgB, r.ID, "test"); err == nil {
		t.Fatal("another org read the RFT")
	}
	_ = reports.ReviewSectionID
}
