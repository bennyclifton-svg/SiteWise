package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"sitewise/internal/costs"
	"sitewise/internal/procurement"
	"sitewise/internal/proposals"
	"sitewise/internal/store"
)

// Combined engagements, slice 3: proposals, price table and award for a group.

var memberLabels = map[string]string{
	"consultant.structural": "Structural engineer", "consultant.civil": "Civil engineer",
	"consultant.mechanical": "Mechanical engineer", "consultant.electrical": "Electrical engineer",
	"consultant.hydraulic": "Hydraulic engineer", "consultant.fire_services": "Fire services engineer",
	"consultant.town_planner": "Town Planner",
}

// combinedGroup makes rows for the disciplines, prepares the cost plan so
// each has its fee line, and combines them.
func combinedGroup(t *testing.T, s *store.Store, disciplines ...string) (procurement.Package, []procurement.Package) {
	t.Helper()
	ctx := context.Background()
	var members []procurement.Package
	var ids []string
	for _, d := range disciplines {
		row, ok := procurement.DisciplinePackage(d, memberLabels[d])
		if !ok {
			t.Fatal("unknown discipline", d)
		}
		p, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
		if err != nil {
			t.Fatal(err)
		}
		members = append(members, p)
		ids = append(ids, p.ID)
	}
	if _, err := s.PrepareCostPlan(ctx, orgA, projectA, userA); err != nil {
		t.Fatal(err)
	}
	group, err := s.CombinePackages(ctx, orgA, projectA, userA, ids, "")
	if err != nil {
		t.Fatal(err)
	}
	return group, members
}

// proposalDoc files one more fee proposal, optionally filed as a discipline.
func proposalDoc(t *testing.T, s *store.Store, n byte, discipline string) string {
	t.Helper()
	id := "11111111-1111-4111-8111-aaaaaaaa" + string("0123456789abcdef"[n>>4]) + string("0123456789abcdef"[n&15]) + "c1"
	addDocument(t, s, orgA, projectA, id, 0x80+n, "fee_proposal")
	if discipline != "" {
		if err := s.CreateDecision(context.Background(), orgA, store.Decision{ID: id[:24] + "ee" + id[26:], DocumentID: id, Field: "discipline", Value: discipline, Band: "green", DecidedBy: "rule"}); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

func groupKeys(t *testing.T, table store.PriceTable, member string) []string {
	t.Helper()
	var out []string
	for _, it := range table.Items {
		if strings.HasPrefix(it.Key, member+":stage:") && it.Base {
			out = append(out, it.Key)
		}
	}
	return out
}

func memberKey(t *testing.T, table store.PriceTable, pkg string) string {
	t.Helper()
	if table.Group == nil {
		t.Fatal("not a group table")
	}
	for _, m := range table.Group.Members {
		if m.PackageID == pkg {
			return m.Key
		}
	}
	t.Fatalf("member %s not in %+v", pkg, table.Group.Members)
	return ""
}

func submissionPackage(t *testing.T, s *store.Store, id string) string {
	t.Helper()
	pkg, err := s.SubmissionPackage(context.Background(), orgA, projectA, id)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

var feeText = pageText([]string{"Karrak Consulting Engineers Pty Ltd"}, []string{"ABN 77 650 369 205"}, []string{"FEE PROPOSAL"}, []string{"Concept design", "$5,000"})

func TestGroupProposalRouting(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	group, members := combinedGroup(t, s, "consultant.structural", "consultant.civil")
	services, _ := combinedGroup(t, s, "consultant.mechanical", "consultant.electrical")
	planner, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, mustRow(t, "consultant.town_planner", "Town Planner"))
	if err != nil {
		t.Fatal(err)
	}
	read := func(doc, drop string) string {
		t.Helper()
		id, err := s.OpenSubmission(ctx, orgA, projectA, doc, drop, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		readWithQuote(t, s, id, feeText, answers{"firm_name": "c1"}, false)
		return submissionPackage(t, s, id)
	}
	if got := read(proposalDoc(t, s, 1, ""), group.ID); got != group.ID {
		t.Fatalf("dropped on the group: %q", got)
	}
	if got := read(proposalDoc(t, s, 2, ""), members[1].ID); got != group.ID {
		t.Fatalf("dropped on a member: %q", got)
	}
	// Two groups on one project: the one holding the filed discipline.
	if got := read(proposalDoc(t, s, 3, "consultant.civil"), ""); got != group.ID {
		t.Fatalf("filed as a member's discipline: %q", got)
	}
	if got := read(proposalDoc(t, s, 4, "consultant.services"), ""); got != services.ID {
		t.Fatalf("combined services filing: %q", got)
	}
	// Without a group, behaviour is unchanged.
	if got := read(proposalDoc(t, s, 5, "consultant.town_planner"), ""); got != planner.ID {
		t.Fatalf("ungrouped discipline: %q", got)
	}
	if err := s.UncombinePackages(ctx, orgA, projectA, userA, services.ID); err != nil {
		t.Fatal(err)
	}
	if got := read(proposalDoc(t, s, 6, "consultant.services"), ""); got != "" {
		t.Fatalf("services filing with no group placed on %q", got)
	}
}

func TestGroupPriceTableAwardAndUndo(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	group, members := combinedGroup(t, s, "consultant.structural", "consultant.civil")
	table, err := s.ReadPriceTable(ctx, orgA, projectA, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	st, cv := memberKey(t, table, members[0].ID), memberKey(t, table, members[1].ID)
	// The stages both disciplines price, in structural's order.
	var stKeys, cvKeys []string
	civil := map[string]bool{}
	for _, k := range groupKeys(t, table, cv) {
		civil[strings.TrimPrefix(k, cv+":")] = true
	}
	for _, k := range groupKeys(t, table, st) {
		if stage := strings.TrimPrefix(k, st+":"); civil[stage] {
			stKeys, cvKeys = append(stKeys, k), append(cvKeys, cv+":"+stage)
		}
	}
	if len(stKeys) < 2 {
		t.Fatalf("shared stages %v", stKeys)
	}
	stage := strings.TrimPrefix(stKeys[0], st+":")
	combinedKey := proposals.CombinedKey(stage)
	var hasCombined bool
	for _, it := range table.Items {
		hasCombined = hasCombined || it.Key == combinedKey
	}
	if !hasCombined {
		t.Fatalf("no combined cell %s in %+v", combinedKey, table.Items)
	}
	second := strings.TrimPrefix(stKeys[1], st+":")

	// Bidder 1 prices each discipline for the first stage and gives the
	// second stage once; civil prices nothing else (unknown, not zero).
	bid1 := pageText([]string{"Karrak Consulting Engineers Pty Ltd"}, []string{"ABN 77 650 369 205"}, []string{"FEE PROPOSAL"},
		[]string{"Structural first stage", "$5,000"}, []string{"Civil first stage", "$3,000"}, []string{"Second stage, both disciplines", "$8,000"})
	a1 := answers{"firm_name": "c1", "gst_basis": "ex", "price:" + stKeys[0]: "a1", "price:" + cvKeys[0]: "a2", "price:" + st + ":" + second: "a3", "price:" + cv + ":" + second: "a3"}
	sub1, err := s.OpenSubmission(ctx, orgA, projectA, proposalDoc(t, s, 7, ""), group.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	readWithQuote(t, s, sub1, bid1, a1, false)
	// Bidder 2 prices the first stage once for both.
	bid2 := pageText([]string{"Bligh Tanner Pty Ltd"}, []string{"FEE PROPOSAL"}, []string{"First stage, structural and civil", "$7,000"})
	sub2, err := s.OpenSubmission(ctx, orgA, projectA, proposalDoc(t, s, 8, ""), group.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	readWithQuote(t, s, sub2, bid2, answers{"firm_name": "c1", "gst_basis": "ex", "price:" + stKeys[0]: "a1", "price:" + cvKeys[0]: "a1"}, false)

	table, err = s.ReadPriceTable(ctx, orgA, projectA, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Columns) != 2 {
		t.Fatalf("columns %d", len(table.Columns))
	}
	c1, c2 := table.Columns[0], table.Columns[1]
	cents := func(c store.FirmColumn, key string) int64 {
		cell, ok := c.Cells[key]
		if !ok || cell.State != proposals.StatePriced || cell.Cents == nil {
			return -1
		}
		return *cell.Cents
	}
	if cents(c1, stKeys[0]) != 500000 || cents(c1, cvKeys[0]) != 300000 || cents(c1, proposals.CombinedKey(second)) != 800000 {
		t.Fatalf("bidder 1 cells %+v", c1.Cells)
	}
	if cents(c2, combinedKey) != 700000 || c2.Cells[stKeys[0]].State != proposals.StateIncluded {
		t.Fatalf("bidder 2 cells %+v", c2.Cells)
	}
	if c1.Totals.KnownBase != 1600000 || c2.Totals.KnownBase != 700000 || c1.Quote != nil {
		t.Fatalf("totals %+v %+v", c1.Totals, c2.Totals)
	}

	// Budgets: structural's was set by the PM, civil's and the group's are blank.
	plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	plan = setBudget(t, s, plan, feeLine(plan, members[0].ID).ID, "9000.00")

	// A member is not awarded on its own while grouped.
	if _, err := s.Award(ctx, orgA, projectA, members[0].ID, userA, store.AwardInput{SubmissionID: sub1, Lines: []store.AwardLine{{Label: "x", Keys: stKeys[:1]}}}); !errors.Is(err, store.ErrPackageGrouped) {
		t.Fatalf("member award %v", err)
	}
	lines := []store.AwardLine{
		{Label: "Structural first stage", Keys: stKeys[:1]},
		{Label: "Civil first stage", Keys: cvKeys[:1]},
		{Label: "Second stage", Keys: []string{proposals.CombinedKey(second)}},
	}
	// One line may not hold two disciplines' cells: it has no one parent.
	if _, err := s.PreviewAward(ctx, orgA, projectA, group.ID, store.AwardInput{SubmissionID: sub1, Lines: []store.AwardLine{{Label: "Both", Keys: []string{stKeys[0], cvKeys[0]}}, lines[2]}}); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("mixed line %v", err)
	}
	// Every priced cell is awarded; a contract never drops a priced stage.
	if _, err := s.PreviewAward(ctx, orgA, projectA, group.ID, store.AwardInput{SubmissionID: sub1, Lines: lines[:2]}); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("partial award %v", err)
	}
	// Wrong org and wrong project are refused.
	if _, err := s.Award(ctx, orgB, projectA, group.ID, userB, store.AwardInput{SubmissionID: sub1, Lines: lines}); err == nil {
		t.Fatal("other org awarded")
	}
	if _, err := s.Award(ctx, orgA, projectB, group.ID, userA, store.AwardInput{SubmissionID: sub1, Lines: lines}); err == nil {
		t.Fatal("other project awarded")
	}
	if _, err := s.Award(ctx, orgA, projectA, group.ID, userA, store.AwardInput{SubmissionID: sub1, Lines: lines, Total: "16000.01"}); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("wrong total %v", err)
	}
	before, _ := s.ReadCostPlan(ctx, orgA, projectA, "")
	v, err := s.Award(ctx, orgA, projectA, group.ID, userA, store.AwardInput{SubmissionID: sub1, Lines: lines, Total: "16000.00"})
	if err != nil {
		t.Fatal(err)
	}
	if v.Total != 1600000 {
		t.Fatalf("contract %d", v.Total)
	}
	plan, _ = s.ReadCostPlan(ctx, orgA, projectA, "")
	byID := map[string]costs.Item{}
	for _, it := range plan.Items {
		byID[it.ID] = it
	}
	groupLine := byID[v.ParentItemID]
	stLine, cvLine := feeLine(plan, members[0].ID), feeLine(plan, members[1].ID)
	if groupLine.PackageID != group.ID || stLine.ParentItemID != groupLine.ID || cvLine.ParentItemID != groupLine.ID {
		t.Fatalf("nesting group %+v st %+v cv %+v", groupLine, stLine, cvLine)
	}
	under := map[string]string{}
	for _, l := range v.Lines {
		under[l.Label] = byID[l.CostItemID].ParentItemID
		if c := byID[l.CostItemID].Values["commitment"]; c.Amount == nil {
			t.Fatalf("line %s has no contract value", l.Label)
		}
	}
	if under["Structural first stage"] != stLine.ID || under["Civil first stage"] != cvLine.ID || under["Second stage"] != groupLine.ID {
		t.Fatalf("lines under %+v", under)
	}
	budget := func(it costs.Item) string {
		b := it.Values["budget"]
		if b.Amount == nil {
			return b.ValueState
		}
		return string(*b.Amount)
	}
	// The PM's budget stays; a blank one adopts the contract beneath it.
	if budget(*stLine) != "9000.00" || budget(*cvLine) != "3000.00" || budget(groupLine) != "8000.00" {
		t.Fatalf("budgets st %s cv %s group %s", budget(*stLine), budget(*cvLine), budget(groupLine))
	}
	totals, err := costs.Summarize(plan, "overall")
	if err != nil {
		t.Fatal(err)
	}
	if r := totals.Rollups[groupLine.ID]; r.Values["commitment"].Amount == nil || *r.Values["commitment"].Amount != "16000.00" {
		t.Fatalf("group contract sum %+v", r.Values["commitment"])
	}
	// The group is appointed; the other bid is unsuccessful; uncombine waits.
	if p := packageByID(t, s, group.ID); p.LifecycleStatus != "appointed" {
		t.Fatalf("group status %s", p.LifecycleStatus)
	}
	if err := s.UncombinePackages(ctx, orgA, projectA, userA, group.ID); !errors.Is(err, store.ErrPackageAwarded) {
		t.Fatalf("uncombine while awarded %v", err)
	}
	if got, err := s.ReadAward(ctx, orgA, projectA, group.ID); err != nil || got == nil || len(got.Lines) != 3 || got.Total != 1600000 {
		t.Fatalf("read award %+v %v", got, err)
	}

	if err := s.UndoAward(ctx, orgA, projectA, group.ID, userA); err != nil {
		t.Fatal(err)
	}
	after, _ := s.ReadCostPlan(ctx, orgA, projectA, "")
	if len(after.Items) != len(before.Items) {
		t.Fatalf("undo left %d lines, before %d", len(after.Items), len(before.Items))
	}
	for _, it := range before.Items {
		var got *costs.Item
		for i := range after.Items {
			if after.Items[i].ID == it.ID {
				got = &after.Items[i]
			}
		}
		if got == nil || got.Posting != it.Posting || got.Label != it.Label || got.ParentItemID != it.ParentItemID || budget(*got) != budget(it) {
			t.Fatalf("undo changed %s: %+v, before %+v", it.Label, got, it)
		}
	}
	if p := packageByID(t, s, group.ID); p.LifecycleStatus == "appointed" {
		t.Fatal("group still appointed after undo")
	}
}

// T03: one building services proposal covering four disciplines, its
// stages priced once. After combining it is one fee line with the four
// disciplines beneath it and nothing counted twice.
func TestT03CombinedServicesProposalIsOneFeeLine(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	group, members := combinedGroup(t, s, "consultant.mechanical", "consultant.electrical", "consultant.hydraulic", "consultant.fire_services")
	table, err := s.ReadPriceTable(ctx, orgA, projectA, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string][]string{}
	for _, m := range members {
		keys[m.ID] = groupKeys(t, table, memberKey(t, table, m.ID))
	}
	text := pageText(
		[]string{"MERIDALE BUILDING SERVICES ENGINEERS PTY LTD"},
		[]string{"Mechanical | Electrical | Hydraulic | Fire"},
		[]string{"ABN 78 303 243 509"},
		[]string{"Fee proposal: building services engineering"},
		[]string{"Fee schedule"},
		[]string{"Item", "Description", "Fee (inc GST)"},
		[]string{"1", "Concept and design development", "$19,800.00"},
		[]string{"2", "Documentation for the CDC and tender", "$26,400.00"},
		[]string{"Total", "$46,200.00"},
		[]string{"Total fee $46,200 including GST."},
	)
	ask := answers{"firm_name": "c1", "gst_basis": "inc"}
	// Jev matches each stage's line to that stage under every discipline.
	for _, m := range members {
		ask["price:"+keys[m.ID][0]] = "a1"
		ask["price:"+keys[m.ID][1]] = "a2"
	}
	id, err := s.OpenSubmission(ctx, orgA, projectA, proposalDoc(t, s, 9, "consultant.services"), "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	readWithQuote(t, s, id, text, ask, false)
	if got := submissionPackage(t, s, id); got != group.ID {
		t.Fatalf("services proposal placed on %q", got)
	}
	table, err = s.ReadPriceTable(ctx, orgA, projectA, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	col := table.Columns[0]
	var lines []store.AwardLine
	for _, it := range table.Items {
		if c := col.Cells[it.Key]; c.State == proposals.StatePriced && it.Base {
			if _, _, combined := proposals.SplitItemKey(it.Key); !combined {
				t.Fatalf("an undivided stage was placed under a discipline: %s", it.Key)
			}
			lines = append(lines, store.AwardLine{Label: it.Label, Keys: []string{it.Key}})
		}
	}
	if len(lines) != 2 || col.Totals.KnownBase != 4200000 {
		t.Fatalf("combined cells %d, base %d", len(lines), col.Totals.KnownBase)
	}
	v, err := s.Award(ctx, orgA, projectA, group.ID, userA, store.AwardInput{SubmissionID: id, Lines: lines, Total: "42000.00"})
	if err != nil {
		t.Fatal(err)
	}
	plan, _ := s.ReadCostPlan(ctx, orgA, projectA, "")
	top, beneath := 0, 0
	for _, it := range plan.Items {
		if it.LineKind != "fee" {
			continue
		}
		switch {
		case it.ParentItemID == "":
			top++
			if it.ID != v.ParentItemID {
				t.Fatalf("a second top-level fee line: %+v", it)
			}
		case it.ParentItemID == v.ParentItemID && it.PackageID != group.ID:
			beneath++
		}
	}
	if top != 1 || beneath != 4 {
		t.Fatalf("top %d, members beneath %d", top, beneath)
	}
	totals, err := costs.Summarize(plan, "overall")
	if err != nil {
		t.Fatal(err)
	}
	if r := totals.Rollups[v.ParentItemID]; r.Values["commitment"].Amount == nil || *r.Values["commitment"].Amount != "42000.00" {
		t.Fatalf("contract sum %+v", r.Values["commitment"])
	}
}

// A member removed after a proposal was read: its cells show as left the
// group and are not awarded until the PM places them again.
func TestGroupMemberThatLeftIsShownAndNotAwarded(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	group, members := combinedGroup(t, s, "consultant.mechanical", "consultant.electrical", "consultant.hydraulic")
	table, err := s.ReadPriceTable(ctx, orgA, projectA, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	mech, hyd := memberKey(t, table, members[0].ID), memberKey(t, table, members[2].ID)
	mechKey, hydKey := groupKeys(t, table, mech)[0], groupKeys(t, table, hyd)[0]
	text := pageText([]string{"Karrak Consulting Engineers Pty Ltd"}, []string{"FEE PROPOSAL"}, []string{"Mechanical design", "$5,000"}, []string{"Hydraulic design", "$2,000"})
	id, err := s.OpenSubmission(ctx, orgA, projectA, proposalDoc(t, s, 10, ""), group.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	readWithQuote(t, s, id, text, answers{"firm_name": "c1", "gst_basis": "ex", "price:" + mechKey: "a1", "price:" + hydKey: "a2"}, false)
	if err := s.RemoveFromGroup(ctx, orgA, projectA, userA, members[2].ID); err != nil {
		t.Fatal(err)
	}
	table, err = s.ReadPriceTable(ctx, orgA, projectA, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	left := false
	for _, it := range table.Group.Left {
		left = left || it.Key == hydKey
	}
	if !left || len(table.Group.Members) != 2 {
		t.Fatalf("left rows %+v members %+v", table.Group.Left, table.Group.Members)
	}
	if _, err := s.PreviewAward(ctx, orgA, projectA, group.ID, store.AwardInput{SubmissionID: id, Lines: []store.AwardLine{{Label: "Mech", Keys: []string{mechKey}}, {Label: "Hyd", Keys: []string{hydKey}}}}); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("left member awarded: %v", err)
	}
	if v, err := s.PreviewAward(ctx, orgA, projectA, group.ID, store.AwardInput{SubmissionID: id, Lines: []store.AwardLine{{Label: "Mech", Keys: []string{mechKey}}}}); err != nil || v.Total != 500000 {
		t.Fatalf("award without the left member: %+v %v", v, err)
	}
}
