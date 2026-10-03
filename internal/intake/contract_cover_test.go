package intake_test

import (
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
)

func TestContractCoverChallengesRenamedFilename(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "General conditions of contract for", Source: identity.Source{Page: 1, X: 200, Y: 470, Width: 285, Height: 18}},
		{Text: "design and construct", Source: identity.Source{Page: 1, X: 200, Y: 448, Width: 180, Height: 18}},
		{Text: "Copyright", Source: identity.Source{Page: 1, X: 200, Y: 300, Width: 80, Height: 9}},
		{Text: "Licence conditions", Source: identity.Source{Page: 1, X: 200, Y: 280, Width: 100, Height: 9}},
		{Text: "Publisher", Source: identity.Source{Page: 1, X: 200, Y: 260, Width: 80, Height: 9}},
	}}
	got := intake.Harvest("Building contract FINAL.pdf", text)
	if _, ok := findCandidate(got, intake.FieldTitle, "General conditions of contract for design and construct"); !ok {
		t.Fatal("missing complete printed contract heading")
	}
	if r := fieldResult(t, intake.Decide(got), intake.FieldTitle); r.Settled {
		t.Fatalf("renamed filename incorrectly settled: %+v", r)
	}
}

func TestAmendedFromStandardIsNotOwnDocumentNumber(t *testing.T) {
	text := identity.Text{Runs: []identity.Run{{Text: "(Amended from AS 4321-2000)"}, {Text: "Amended from AS4321-2000"}}}
	got := intake.Harvest("AS4321 for Building FINAL.pdf", text)
	if r := fieldResult(t, intake.Decide(got), intake.FieldNumber); r.Settled {
		t.Fatalf("referenced standard became own number: %+v", r)
	}
	// An unrelated AS-prefixed drawing identifier is not globally blacklisted.
	got = intake.Harvest("AS4321 - Layout.pdf", identity.Text{})
	if _, ok := findCandidate(got, intake.FieldNumber, "AS4321"); !ok {
		t.Fatal("unreferenced AS drawing identifier removed")
	}
}

func TestAgreementHeadingAtBodySizeRemainsACandidate(t *testing.T) {
	text := identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: "FORMAL INSTRUMENT OF AGREEMENT", Source: identity.Source{Page: 1, X: 50, Y: 720, Width: 250, Height: 12}},
		{Text: "DATE:", Source: identity.Source{Page: 1, X: 50, Y: 690, Width: 45, Height: 12}},
		{Text: "Parties", Source: identity.Source{Page: 1, X: 50, Y: 650, Width: 50, Height: 12}},
		{Text: "Principal", Source: identity.Source{Page: 1, X: 50, Y: 610, Width: 70, Height: 12}},
	}}
	got := intake.Harvest("FIOA (Received yesterday).pdf", text)
	if _, ok := findCandidate(got, intake.FieldTitle, "FORMAL INSTRUMENT OF AGREEMENT"); !ok {
		t.Fatal("missing literal all-capital agreement heading")
	}
	text.Runs[0].Text = "Formal instrument of agreement"
	if _, ok := findCandidate(intake.Harvest("other.pdf", text), intake.FieldTitle, text.Runs[0].Text); ok {
		t.Fatal("ordinary mixed-case body phrase became cover heading")
	}
}
