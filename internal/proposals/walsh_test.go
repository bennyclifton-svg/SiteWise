package proposals

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sitewise/internal/identity"
)

// The Walsh files are synthetic Markdown documents a user dropped on
// 27 October 2026 (an engagement letter and the fee proposal it accepts).
func walshDraft(t *testing.T, file string) *Draft {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "walsh", file))
	if err != nil {
		t.Fatal(err)
	}
	text, err := identity.Extract(context.Background(), "md", bytes.NewReader(body), int64(len(body)), ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	return NewDraft(Harvest(file, text), Context{Discipline: "consultant.architect", SiteAddress: "42 Hargrave Street, Paddington NSW 2021"}, nil)
}

func walshFlag(r Reading, flag string) bool {
	for _, f := range r.Flags {
		if f == flag || strings.HasPrefix(f, flag+":") {
			return true
		}
	}
	return false
}

// A letter prints its date alone above the addressee.
func TestWalshLetterIsDatedFromItsHeader(t *testing.T) {
	if r := walshDraft(t, "01-engagement-letter-atelier-north.md").Finish(); r.Dated != "2026-03-03" {
		t.Fatalf("letter dated %q, want 2026-03-03", r.Dated)
	}
	if r := walshDraft(t, "02-fee-proposal-atelier-north.md").Finish(); r.Dated != "2026-02-18" {
		t.Fatalf("proposal dated %q, want 2026-02-18", r.Dated)
	}
}

// With no letterhead, the sign-off names the firm ("Director, Atelier North").
func TestWalshProposalFirmFromItsSignOff(t *testing.T) {
	d := walshDraft(t, "02-fee-proposal-atelier-north.md")
	got, ok := d.Decided(FieldFirmName)
	if !ok || got.Value != "Atelier North" || got.DecidedBy != ByRule {
		t.Fatalf("firm %+v", got)
	}
	if c, ok := d.Decided(FieldFirmContact); !ok || c.Value != "Lucia Ferraro" {
		t.Fatalf("contact %+v", c)
	}
}

// A forwarded email is the signer's: the builder, not the architect whose
// address forwarded it. Its date line carries a time.
func TestWalshBuilderEmailIsTheBuilders(t *testing.T) {
	architect := &Firm{ID: "f-an", Name: "Atelier North Pty Ltd"}
	body, err := os.ReadFile(filepath.Join("testdata", "walsh", "04-email-builder-preliminary-cost-advice.md"))
	if err != nil {
		t.Fatal(err)
	}
	text, err := identity.Extract(context.Background(), "md", bytes.NewReader(body), int64(len(body)), ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	lookup := func(abns, keys []string) *Firm {
		for _, k := range keys {
			if k == NameKey(architect.Name) || k == "ateliernorth" {
				return architect
			}
		}
		return nil
	}
	d := NewDraft(Harvest("04-email-builder-preliminary-cost-advice.md", text), Context{Discipline: "trade.main_works", Lookup: lookup}, nil)
	if got, ok := d.Decided(FieldFirmName); !ok || got.Value != "Deluca Building Pty Ltd" {
		t.Fatalf("firm %+v", got)
	}
	if r := d.Finish(); r.Dated != "2026-03-22" {
		t.Fatalf("dated %q", r.Dated)
	}
}

// Advice with no ABN prints its one registered name at the top.
func TestWalshHeritageAdviceFirmFromItsHeader(t *testing.T) {
	d := walshDraft(t, "05-email-heritage-advisor-desktop.md")
	if got, ok := d.Decided(FieldFirmName); !ok || got.Value != "Heritage Insight Pty Ltd" {
		t.Fatalf("firm %+v", got)
	}
	if r := d.Finish(); r.Dated != "2026-03-28" {
		t.Fatalf("dated %q", r.Dated)
	}
}

// A printed ABN that fails the checksum is said, not silently dropped.
func TestWalshInvalidABNIsFlagged(t *testing.T) {
	r := walshDraft(t, "01-engagement-letter-atelier-north.md").Finish()
	if !walshFlag(r, FlagABNInvalid) {
		t.Fatalf("flags %v: the printed ABN 92 456 789 012 fails the checksum", r.Flags)
	}
	if _, ok := walshDraft(t, "01-engagement-letter-atelier-north.md").Decided(FieldFirmABN); ok {
		t.Fatal("an invalid ABN was decided")
	}
	if walshFlag(walshDraft(t, "02-fee-proposal-atelier-north.md").Finish(), FlagABNInvalid) {
		t.Fatal("a proposal that prints no ABN was flagged")
	}
}

// "Construction administration | Monthly × est. 9 months | $3,500" adds to
// the total only as a lump sum: the reader asks the PM to confirm.
func TestWalshMonthlyFeeWordingIsFlagged(t *testing.T) {
	if r := walshDraft(t, "01-engagement-letter-atelier-north.md").Finish(); !walshFlag(r, FlagMonthlyFee) {
		t.Fatalf("flags %v", r.Flags)
	}
	if r := walshDraft(t, "02-fee-proposal-atelier-north.md").Finish(); walshFlag(r, FlagMonthlyFee) {
		t.Fatalf("proposal with no monthly row flagged: %v", r.Flags)
	}
}

// Professional indemnity: the cover limit and the period end, read in code.
func TestWalshProfessionalIndemnity(t *testing.T) {
	r := walshDraft(t, "01-engagement-letter-atelier-north.md").Finish()
	if r.PI == nil || r.PI.LimitCents == nil || *r.PI.LimitCents != 500000000 || r.PI.ExpiresOn != "2026-06-30" {
		t.Fatalf("PI %+v", r.PI)
	}
	if p := walshDraft(t, "02-fee-proposal-atelier-north.md").Finish(); p.PI != nil {
		t.Fatalf("PI read from a proposal that states none: %+v", p.PI)
	}
}
