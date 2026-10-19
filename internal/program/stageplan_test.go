package program

import (
	"reflect"
	"testing"
)

func TestDiffStagesRenameIsNotDeleteAndAdd(t *testing.T) {
	recorded := []RecordedStage{{"s1", "Concept design"}, {"s2", "Approval documentation"}, {"s3", "Tender"}}
	current := []ProgramStage{{ID: "s1", Label: "Concept design"}, {ID: "s2", Label: "DA documentation"}, {ID: "s4", Label: "Construction"}}
	d := DiffStages(recorded, current)
	want := []StageChange{
		{Kind: StageRenamed, ProgramStageID: "s2", OldLabel: "Approval documentation", NewLabel: "DA documentation"},
		{Kind: StageRemoved, ProgramStageID: "s3", OldLabel: "Tender"},
		{Kind: StageAdded, ProgramStageID: "s4", NewLabel: "Construction"},
	}
	if !reflect.DeepEqual(d.Changes, want) || d.Reordered {
		t.Fatalf("diff %+v", d)
	}
	if d := DiffStages(recorded[:2], []ProgramStage{{ID: "s2", Label: "Approval documentation"}, {ID: "s1", Label: "Concept design"}}); len(d.Changes) != 0 || !d.Reordered {
		t.Fatalf("reorder %+v", d)
	}
}

func TestNeedsOffer(t *testing.T) {
	if NeedsOffer(3, 0, 3) || !NeedsOffer(3, 0, 4) || NeedsOffer(3, 4, 4) || !NeedsOffer(3, 4, 5) {
		t.Fatal("offer rule")
	}
}

// Every row of the plan 7.6 Update table.
func TestUpdateActions(t *testing.T) {
	ren := StageChange{Kind: StageRenamed}
	add := StageChange{Kind: StageAdded}
	rem := StageChange{Kind: StageRemoved}
	cases := []struct {
		doc   string
		c     StageChange
		money bool
		want  string
	}{
		{DocPackageStages, ren, false, CopyLabel},
		{DocPackageStages, add, false, OfferUnticked},
		{DocPackageStages, rem, false, Retire},
		{DocPackageStages, rem, true, MarkRemovedKeep},
		{DocDraftRFP, ren, false, RewriteRow},
		{DocDraftRFP, rem, false, RewriteRow},
		{DocIssuedRFP, ren, false, NewDraftRevision},
		{DocIssuedRFP, rem, true, NewDraftRevision},
		{DocProposal, ren, false, NoChange},
		{DocProposal, add, false, FlagMismatch},
		{DocProposal, rem, true, FlagMismatch},
		{DocAwardedContract, ren, true, CopyLabel},
		{DocAwardedContract, add, true, AddUnpricedLine},
		{DocAwardedContract, rem, true, MarkRemovedKeep},
		{DocAwardedContract, rem, false, MarkRemovedKeep},
		{DocFrozenBaseline, ren, false, NoChange},
		{DocFrozenBaseline, rem, true, NoChange},
	}
	for _, c := range cases {
		if got := UpdateAction(c.doc, c.c, c.money); got != c.want {
			t.Errorf("%s %s money=%v: %s, want %s", c.doc, c.c.Kind, c.money, got, c.want)
		}
	}
}
