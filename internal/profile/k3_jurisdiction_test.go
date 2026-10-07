package profile_test

import (
	"sitewise/internal/profile"
	"testing"
)

func TestJurisdictionProposalInputUsesAuthoredStateAndPartUnknown(t *testing.T) {
	cat := repoCatalog(t)
	rows := []profile.Row{{PartID: whole, Key: "det.state", Value: "NSW", Band: "user", Origin: "user", Meaning: "stated"}}
	in, err := profile.ProposalInputs(rows, input().Parts, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	if in.Parts["p-b"].Values["state"].Value != "NSW" {
		t.Fatal("site jurisdiction not inherited")
	}
	rows = append(rows, profile.Row{PartID: "p-b", Key: "det.state", Band: "user", Origin: "user", ValueState: profile.StateUnknown})
	in, err = profile.ProposalInputs(rows, input().Parts, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	if in.Parts["p-b"].Values["state"].Value != "" || in.Parts[whole].Values["state"].Value != "NSW" {
		t.Fatal("part unknown leaked or lost")
	}
	rows = rows[:1]
	rows[0].Origin = "assumption"
	in, err = profile.ProposalInputs(rows, input().Parts, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	if in.Parts["p-b"].Values["state"].Value != "" {
		t.Fatal("assumed jurisdiction treated as fact")
	}
}
