package intake_test

import (
	"encoding/json"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"strings"
	"testing"
)

func TestUnderscoreSheetAndParenthesisedRevision(t *testing.T) {
	name := "15123_S0003_AFS Wall Details-(02).pdf"
	got := intake.Harvest(name, identity.Text{})
	for field, value := range map[string]string{"number": "S0003", "revision": "02"} {
		if _, ok := findCandidate(got, field, value); !ok {
			t.Fatalf("missing %s %s", field, value)
		}
	}
}

func TestCopySuffixIsNotARevision(t *testing.T) {
	for _, c := range intake.Harvest("CC-A-601 WINDOW SCHEDULE (2).pdf", identity.Text{}) {
		if c.Field == intake.FieldRevision {
			t.Fatal("download copy suffix is not a stated revision")
		}
	}
}

func TestConsultantContactCaptionIsNotAuthorship(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "Waterproofing"}, {Text: "Architectural"}}}
	d := intake.NewDraft(draftCatalog(t), "plan.pdf", text, nil, nil)
	d.Plan(nil, "self")
	call, _ := d.Call()
	if _, ok := call.Questions[intake.FieldDiscipline]; !ok {
		t.Fatal("contact-panel caption settled author discipline")
	}
}

func TestTechnicalDrawingLifecycleUsesPurpose(t *testing.T) {
	name := "A-100 Floor Plan.pdf"
	text := identity.Text{Runs: []identity.Run{{Text: "Drawing"}, {Text: "ISSUED FOR CONSTRUCTION"}}}
	d := intake.NewDraft(draftCatalog(t), name, text, intake.Harvest(name, text), nil)
	d.Plan(nil, "self")
	for _, v := range d.Decisions() {
		if v.Field == intake.FieldLifecycle && v.Value == "construction" {
			t.Fatal("issue stamp incorrectly settled filing purpose")
		}
	}
	if call, ok := d.Call(); ok {
		q, asked := call.Questions[intake.FieldLifecycle]
		b, _ := json.Marshal(q)
		if asked && !strings.Contains(string(b), "Design") {
			t.Fatal("question omits agreed Design policy")
		}
	}
}

func TestFilingStateRetainsLateIdentityContext(t *testing.T) {
	text := identity.Text{}
	for i := 0; i < 80; i++ {
		text.Runs = append(text.Runs, identity.Run{Text: "General coordination note for this building project."})
	}
	text.Runs = append(text.Runs, identity.Run{Text: "Prepared by Example Architects"}, identity.Run{Text: "Drawing No A-100"}, identity.Run{Text: "Issue date 06.11.2023"})
	d := intake.NewDraft(draftCatalog(t), "plan.pdf", text, intake.Harvest("plan.pdf", text), nil)
	d.Plan(nil, "self")
	call, _ := d.Call()
	b, _ := json.Marshal(call.State)
	if !strings.Contains(string(b), "Example Architects") {
		t.Fatal("late author context discarded")
	}
}

func TestIncidentalVocabularyDoesNotBecomeGreen(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "Acoustical recommendations to satisfy BCA requirements"}, {Text: "The parties accept the contract terms in Schedule 1."}}}
	d := intake.NewDraft(draftCatalog(t), "advice.pdf", text, intake.Harvest("advice.pdf", text), nil)
	d.Plan(nil, "self")
	call, ok := d.Call()
	if !ok {
		t.Fatal("expected vocabulary questions")
	}
	for _, field := range []string{intake.FieldKind, intake.FieldDiscipline} {
		if _, ok := call.Questions[field]; !ok {
			t.Fatalf("incidental text settled %s", field)
		}
	}
}

func TestFilenameReportDoesNotOverrideDeclaration(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "Design compliance declaration - single regulated design"}}}
	d := intake.NewDraft(draftCatalog(t), "DCD Facade Report.pdf", text, intake.Harvest("DCD Facade Report.pdf", text), nil)
	d.Plan(nil, "self")
	call, _ := d.Call()
	if _, ok := call.Questions[intake.FieldKind]; !ok {
		t.Fatal("filename word overrode document identity")
	}
}

func TestCombinedDrawingNumberCaptionKeepsDesignLifecycle(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "Project Number/Drawing Number"}, {Text: "ISSUED FOR CONSTRUCTION"}}}
	d := intake.NewDraft(draftCatalog(t), "plan.pdf", text, intake.Harvest("plan.pdf", text), nil)
	d.Plan(nil, "self")
	assertDecision(t, d.Decisions(), intake.FieldLifecycle, "design", intake.BandGreen, intake.DecidedByRule)
}
