package invoices

import (
	"testing"
	"time"
)

// A combined engagement is one supplier: while grouped, a member is left
// out of supplier matching, so the firm's ABN settles on the group even when
// the member still carries the same firm (combined engagements, slice 1).
func TestGroupMemberLeftOutOfSupplierMatching(t *testing.T) {
	firm := []Firm{{NameKey: "karrakengineers", ABN: "77650369205"}}
	ctx := Context{Packages: []Package{
		{ID: "member", Title: "Structural engineer", Kind: "services", Firms: firm, ABN: "77650369205", Grouped: true},
		{ID: "group", Title: "Structural and civil", Kind: "services", Firms: firm},
	}}
	ev := Evidence{ABNs: []Candidate{{Value: "77650369205", Labels: []string{"ABN"}, Labeled: true, Count: 1}}}
	d := NewDraft(ev, ctx, nil)
	if got := d.Finish(time.Now()).PackageID; got != "group" {
		t.Fatalf("supplier %q, want the group", got)
	}
	// Without the group flag the two rows tie and no rule decides.
	ctx.Packages[0].Grouped = false
	d = NewDraft(ev, ctx, nil)
	if got := d.Finish(time.Now()).PackageID; got != "" {
		t.Fatalf("tied supplier settled on %q", got)
	}
	// A member is not offered to Jev either.
	ctx.Packages[0].Grouped = true
	d = NewDraft(Evidence{}, ctx, nil)
	if q, ok := d.question(FieldSupplier); !ok || len(q.Options) != 2 || q.Options[0].Value != "group" {
		t.Fatalf("supplier options %+v", q.Options)
	}
}
