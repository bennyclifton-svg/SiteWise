package program

import (
	"reflect"
	"testing"
)

func checksFor(t *testing.T, name string, ci CheckInput) []Check {
	t.Helper()
	f := fixtureByName(t, name)
	ci.Input = f.Input
	ci.Output = Compute(f.Input)
	return Checks(ci)
}

func codes(list []Check) []string {
	var out []string
	for _, c := range list {
		out = append(out, c.Code)
	}
	return out
}

func TestConsentLapse(t *testing.T) {
	// Construction starts Mon 5 April 2027 in leaf-stage.
	got := checksFor(t, "leaf-stage", CheckInput{ConstructionStageID: "construction", ConsentLapseDate: "2027-04-01", CalendarYears: []int{2027}})
	want := []Check{{Code: CheckConsentLapse, ActivityID: "construction", Message: "Construction starts on 5 April 2027, after consent lapses on 1 April 2027."}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
	// Starting on the lapse date is not after it.
	if got := checksFor(t, "leaf-stage", CheckInput{ConstructionStageID: "construction", ConsentLapseDate: "2027-04-05", CalendarYears: []int{2027}}); len(got) != 0 {
		t.Errorf("start on the lapse date flagged: %+v", got)
	}
}

func TestTargetCompletion(t *testing.T) {
	// leaf-stage finishes Fri 28 May 2027.
	for _, c := range []struct{ target, want string }{
		{"2027-05-24", "Finish on 28 May 2027 is 4 days after the target completion of 24 May 2027."},
		{"2027-05-14", "Finish on 28 May 2027 is 2 weeks after the target completion of 14 May 2027."},
		{"2027-05-13", "Finish on 28 May 2027 is 2 weeks and 1 day after the target completion of 13 May 2027."},
		{"2027-05-28", ""},
	} {
		got := checksFor(t, "leaf-stage", CheckInput{TargetCompletion: c.target, CalendarYears: []int{2027}})
		msg := ""
		if len(got) == 1 && got[0].Code == CheckTargetCompletion {
			msg = got[0].Message
		}
		if msg != c.want || (c.want == "" && len(got) != 0) {
			t.Errorf("target %s: %+v", c.target, got)
		}
	}
}

func TestMSOConflictMessage(t *testing.T) {
	got := checksFor(t, "mso-conflict", CheckInput{CalendarYears: []int{2027}})
	want := []Check{{Code: CheckMSOConflict, ActivityID: "b", Message: "Pour slab must start 8 March 2027 but Excavate holds it until 15 March 2027."}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
	// Without a start date the same conflict is told in weeks.
	f := fixtureByName(t, "mso-conflict")
	f.Calendar.Start = ""
	out := Output{Conflicts: []Conflict{{ActivityID: "b", Index: 5, HeldIndex: 10, HeldBy: "a"}}}
	got = Checks(CheckInput{Input: f.Input, Output: out})
	if len(got) != 1 || got[0].Message != "Pour slab must start Week 2, day 1 but Excavate holds it until Week 3, day 1." {
		t.Errorf("relative: %+v", got)
	}
}

func TestCalendarGap(t *testing.T) {
	// leaf-stage runs March to May 2027 and has no calendar rows.
	got := checksFor(t, "leaf-stage", CheckInput{})
	if len(got) != 1 || got[0].Message != "No holiday data for 2027; weekends only." {
		t.Errorf("got %+v", got)
	}
	// Years in the program's own calendar count as covered.
	if got := checksFor(t, "shutdown-year-end", CheckInput{}); len(got) != 0 {
		t.Errorf("shutdown-year-end has rows for 2027 and 2028: %+v", got)
	}
	// The high run decides how far the program reaches.
	f := fixtureByName(t, "three-run-ranges")
	f.Activities[0].High = ptr(400)
	got = Checks(CheckInput{Input: f.Input, Output: Compute(f.Input), CalendarYears: []int{2027}})
	if !reflect.DeepEqual(codes(got), []string{CheckCalendarGap}) || got[0].Message != "No holiday data for 2028; weekends only." {
		t.Errorf("high run into 2028: %+v", got)
	}
	// A relative program has no years.
	f = fixtureByName(t, "relative-weeks")
	if got := Checks(CheckInput{Input: f.Input, Output: Compute(f.Input)}); len(got) != 0 {
		t.Errorf("relative: %+v", got)
	}
}

func TestHoldPointUnlinked(t *testing.T) {
	in := Input{
		Calendar: Calendar{Start: "2027-03-01"},
		Activities: []Activity{
			{ID: "s", Kind: KindStage, Label: "Structure"},
			{ID: "frame", ParentID: "s", Duration: ptr(5)},
			{ID: "h1", ParentID: "s", Kind: KindHoldPoint, Duration: ptr(0)},
			{ID: "h2", ParentID: "s", Kind: KindHoldPoint, Duration: ptr(0)},
			{ID: "h3", ParentID: "s", Kind: KindHoldPoint, Duration: ptr(0)},
			{ID: "line", ParentID: "s", Duration: ptr(3)},
		},
		Links: []Link{
			{ID: "1", Pred: "frame", Succ: "h1"},
			{ID: "2", Pred: "frame", Succ: "h3"},
			{ID: "3", Pred: "h3", Succ: "line"},
		},
	}
	got := Checks(CheckInput{Input: in, Output: Compute(in), CalendarYears: []int{2027}})
	want := []Check{
		{Code: CheckHoldPointUnlinked, ActivityID: "h1", Message: "Hold point has no successor."},
		{Code: CheckHoldPointUnlinked, ActivityID: "h2", Message: "Hold point has no predecessor or successor."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestChecksOrderAndInputs(t *testing.T) {
	got := checksFor(t, "mso-conflict", CheckInput{TargetCompletion: "2027-03-01", SkeletonTie: true, RemovedPackageRows: []string{"c"}})
	want := []string{CheckTargetCompletion, CheckMSOConflict, CheckCalendarGap, CheckSkeletonTie, CheckPackageRemoved}
	if !reflect.DeepEqual(codes(got), want) {
		t.Errorf("order %v", codes(got))
	}
	if got[len(got)-1].ActivityID != "c" {
		t.Errorf("package_removed names the row: %+v", got[len(got)-1])
	}
	// While a loop stands nothing is dated, so only undated checks remain.
	if got := checksFor(t, "cycle", CheckInput{TargetCompletion: "2027-03-01", SkeletonTie: true}); !reflect.DeepEqual(codes(got), []string{CheckSkeletonTie}) {
		t.Errorf("cycle: %+v", got)
	}
}
