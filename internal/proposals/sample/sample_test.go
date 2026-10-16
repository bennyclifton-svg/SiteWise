package sample

import (
	"bytes"
	"context"
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/proposals"
	"sitewise/internal/rfp"
)

// A sample proposal reads back through the real harvest and questions.
func TestSampleReadsIntoItsSchedule(t *testing.T) {
	p := Proposal{Firm: "Example Planning Pty Ltd", ABN: "88071762537", Email: "office@example-planning.test",
		Lines: []Line{{"Preliminary advice", 300000}, {"Planning report", 900000}, {"Lodgement", -1}},
		Rates: []Line{{"Director", 45000}}, Extra: &Line{"Heritage statement", 120000}}
	b := PDF(p)
	text, err := identity.Extract(context.Background(), "pdf", bytes.NewReader(b), int64(len(b)), proposals.ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	items := []rfp.ScheduleItem{
		{Key: "stage:advice", Kind: rfp.ItemStage, Label: "Preliminary advice", Base: true},
		{Key: "stage:report", Kind: rfp.ItemStage, Label: "Planning report", Base: true},
		{Key: "stage:lodgement", Kind: rfp.ItemStage, Label: "Lodgement", Base: true},
		{Key: "rate:director", Kind: rfp.ItemRate, Label: "Director"},
	}
	pc := proposals.Context{Packages: []proposals.Package{{ID: "p", Discipline: "consultant.town_planner"}}, Discipline: "consultant.town_planner", Schedules: map[string][]rfp.ScheduleItem{"p": items}}
	lo, hi, g := 2, 255, 0.5
	th := proposals.Thresholds{QuestionVersion: proposals.QuestionVersion, Questions: map[string][]proposals.Threshold{}}
	for _, f := range []string{"price", "rate", "firm"} {
		th.Questions[f] = []proposals.Threshold{{Options: &lo, MaxOptions: &hi, Green: &g, Amber: &g}}
	}
	out, err := proposals.Read(context.Background(), Jev{}, th, "Fee proposal (Example Planning).pdf", text, pc, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]proposals.Price{}
	for _, pr := range out.Prices {
		got[pr.Key] = pr
	}
	if c := got["stage:report"].Cents; c == nil || *c != 900000 {
		t.Fatalf("report %+v", got["stage:report"])
	}
	if got["stage:lodgement"].State != proposals.StateNotPriced || got["rate:director"].Cents == nil || *got["rate:director"].Cents != 45000 {
		t.Fatalf("prices %+v", got)
	}
	if c := got[""].Cents; c == nil || *c != 120000 {
		t.Fatalf("extra %+v", got[""])
	}
	if out.Kind != proposals.KindFeeProposal || out.FirmFields[proposals.FieldFirmName] == "" || out.StatedTotal == nil || *out.StatedTotal != 1200000 {
		t.Fatalf("reading %+v", out.Reading)
	}
}
