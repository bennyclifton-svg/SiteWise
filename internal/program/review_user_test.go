package program

import "testing"

// Plan section 2 rule 2 and 7.4: the review raises only differences its
// inputs made. The user's own label, link shape and deletions are user
// values, so they are never offered back.

func userEditInput() GenInput {
	return GenInput{WorkTypes: []string{"new"}, Values: map[string]string{"flood_hazard": "true"}, WorkItems: []WorkItem{
		{ID: "wi-frame", System: "structure", Action: "new", Title: "Timber frame"},
		{ID: "wi-roof", System: "envelope", Action: "new", Title: "Roof"}}}
}

// without drops the rows (and every link touching them) the user deleted.
func without(rows []LiveRow, links []LiveLink, gone func(LiveRow) bool) ([]LiveRow, []LiveLink) {
	dropped := map[string]bool{}
	var keptRows []LiveRow
	for _, r := range rows {
		if gone(r) {
			dropped[r.ID] = true
			continue
		}
		keptRows = append(keptRows, r)
	}
	var keptLinks []LiveLink
	for _, l := range links {
		if !dropped[l.PredID] && !dropped[l.SuccID] {
			keptLinks = append(keptLinks, l)
		}
	}
	return keptRows, keptLinks
}

func TestDiffLeavesTheUsersOwnEdits(t *testing.T) {
	cat := testCatalog(t, testLibrary, testHoldPoint)
	p := mustGenerate(t, cat, userEditInput())
	hold, frame := "hold:if.test-frame-before-roof", "fragment:fr.frame:wi-frame:frame"
	if _, ok := p.Row(hold); !ok {
		t.Fatal("no hold point generated")
	}
	rows, links := liveFrom(p)

	// The user deleted the hold point (with a reason), renamed the frame
	// row and changed a generated link's lag.
	rows, links = without(rows, links, func(r LiveRow) bool { return r.SourceKey == hold })
	fi := -1
	for i := range rows {
		if rows[i].SourceKey == frame {
			fi = i
			rows[i].GenLabel = rows[i].Label
			rows[i].Label, rows[i].Origin, rows[i].ReviewStatus = "Frame (steel)", OriginUser, ReviewAccepted
		}
	}
	li := -1
	for i := range links {
		if links[i].SourceKey != "" {
			li = i
			break
		}
	}
	if fi < 0 || li < 0 {
		t.Fatalf("frame row %d, generated link %d", fi, li)
	}
	links[li].GenShape = &LinkShape{Type: links[li].Type, Lag: links[li].Lag, LagUnit: links[li].LagUnit}
	links[li].Lag, links[li].Origin = links[li].Lag+3, OriginUser

	gone := UserRetired{Rows: map[string]bool{hold: true}}
	if cs := Diff(rows, links, p, gone); len(cs) != 0 {
		t.Fatalf("the user's own edits were raised: %+v", cs)
	}

	// Without the record of the user's delete the hold point would be
	// offered back with Adopt, which is the defect.
	if c, ok := findChange(Diff(rows, links, p, UserRetired{}), ChangeAdd, hold); !ok || c.DefaultChoice != ChoiceAdopt {
		t.Fatalf("control: hold point add %+v", c)
	}

	// When generation's label moves on (an input changed), the rename is a
	// real difference: raised, defaulting to Keep.
	rows[fi].GenLabel = "Frame (as generated before)"
	c, ok := findChange(Diff(rows, links, p, gone), ChangeRename, frame)
	if !ok || c.DefaultChoice != ChoiceKeep || !c.Edited {
		t.Fatalf("input-driven rename %+v", c)
	}
	rows[fi].GenLabel = rows[fi].Label // renamed back: nothing to compare
	links[li].GenShape.Lag++
	if c, ok := findChange(Diff(rows, links, p, gone), ChangeLink, links[li].SourceKey); !ok || c.DefaultChoice != ChoiceKeep {
		t.Fatalf("input-driven link change %+v", c)
	}
}

func TestDiffLeavesADeletedStageAndItsRows(t *testing.T) {
	cat := testCatalog(t, testLibrary, testHoldPoint)
	p := mustGenerate(t, cat, userEditInput())
	frame, ok := p.Row("fragment:fr.frame:wi-frame:frame")
	if !ok || frame.ParentKey == "" {
		t.Fatalf("frame %+v", frame)
	}
	stage := frame.ParentKey
	rows, links := liveFrom(p)
	// The user deleted the stage: its rows went with it. Only the stage's
	// delete is recorded here; its rows must not be offered without it.
	rows, links = without(rows, links, func(r LiveRow) bool { return r.SourceKey == stage || r.ParentID == "id:"+stage })
	if cs := Diff(rows, links, p, UserRetired{Rows: map[string]bool{stage: true}}); len(cs) != 0 {
		t.Fatalf("a deleted stage was offered back: %+v", cs)
	}
	if _, ok := findChange(Diff(rows, links, p, UserRetired{}), ChangeAdd, stage); !ok {
		t.Fatal("control: the stage is not offered without the record")
	}
}

func TestDiffLeavesADeletedLink(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	p := mustGenerate(t, cat, reviewInput())
	rows, links := liveFrom(p)
	key := links[0].SourceKey
	links = links[1:]
	if cs := Diff(rows, links, p, UserRetired{Links: map[string]bool{key: true}}); len(cs) != 0 {
		t.Fatalf("a deleted link was offered back: %+v", cs)
	}
	if _, ok := findChange(Diff(rows, links, p, UserRetired{}), ChangeLink, key); !ok {
		t.Fatal("control: the link is not offered without the record")
	}
}
