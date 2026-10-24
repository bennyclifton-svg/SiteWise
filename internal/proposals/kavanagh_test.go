package proposals

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"sitewise/internal/identity"
)

// The Kavanagh files are synthetic Markdown fee proposals (each says so in
// its own banner). They exposed Markdown emphasis kept in firm names, a GST
// basis left unknown and a firm with no name, 24 October 2026.
var kavanagh = []struct {
	file, discipline, name, abn, address, dated string
	total                                       int64
}{
	{"01-fee-proposal-quoin-architecture.md", "consultant.architect", "Quoin Architecture Pty Ltd", "51000000680", "Level 4, 17 Hickson Road, Millers Point NSW 2000", "2026-01-20", 9600000},
	{"02-fee-proposal-catenary-structures.md", "consultant.structural", "Catenary Structures Pty Ltd", "69000000070", "Suite 6, 130 William Street, Woolloomooloo NSW 2011", "2026-01-22", 4180000},
	{"03-fee-proposal-flowline-hydraulics.md", "consultant.hydraulic", "Flowline Hydraulics Pty Ltd", "83000000570", "Level 2, 55 Miller Street, North Sydney NSW 2060", "2026-01-23", 3250000},
	{"04-fee-proposal-vertex-cost-advisory.md", "consultant.quantity_surveyor", "Vertex Cost Advisory Pty Ltd", "74000000006", "Level 8, 25 Martin Place, Sydney NSW 2000", "2026-01-24", 4500000},
}

func kavanaghText(t *testing.T, file string) identity.Text {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "kavanagh", file))
	if err != nil {
		t.Fatal(err)
	}
	text, err := identity.Extract(context.Background(), "md", bytes.NewReader(body), int64(len(body)), ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func TestKavanaghFirmGSTAndTotalSettleByRule(t *testing.T) {
	for _, k := range kavanagh {
		t.Run(k.file, func(t *testing.T) {
			ev := Harvest(k.file, kavanaghText(t, k.file))
			d := NewDraft(ev, Context{Discipline: k.discipline, SiteAddress: "8 Paperbark Lane, Northbridge NSW 2063"}, nil)
			want := map[string]string{FieldKind: KindFeeProposal, FieldFirmName: k.name, FieldFirmABN: k.abn, FieldFirmAddress: k.address, FieldGSTBasis: "ex"}
			for field, value := range want {
				got, ok := d.Decided(field)
				if !ok || got.Value != value || got.DecidedBy != ByRule || got.Band != BandGreen {
					t.Errorf("%s = %+v, want %q by rule", field, got, value)
				}
			}
			r := d.Finish()
			if r.StatedTotal == nil || *r.StatedTotal != k.total {
				t.Errorf("stated total %v, want %d", r.StatedTotal, k.total)
			}
			if r.Dated != k.dated {
				t.Errorf("dated %q, want %q", r.Dated, k.dated)
			}
			for _, f := range r.Flags {
				if f == FlagGSTNotStated {
					t.Errorf("GST is stated as exclusive; flag %s raised", f)
				}
			}
		})
	}
}

// A client block can carry its own ABN. The letterhead rule must not read
// the addressee as the firm.
func TestLetterheadRuleIgnoresLabelledClientBlock(t *testing.T) {
	md := "# Fee proposal\n\n**To:** Harbour Holdings Pty Ltd\nABN 51 000 000 680\nLevel 4, 17 Hickson Road, Millers Point NSW 2000\n\n## Professional fee\n\n| Stage | Fee ex GST |\n| --- | ---: |\n| **Total professional fee** | **$10,000.00** |\n"
	text, err := identity.Extract(context.Background(), "md", bytes.NewReader([]byte(md)), int64(len(md)), ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	d := NewDraft(Harvest("proposal.md", text), Context{Discipline: "consultant.architect"}, nil)
	if got, ok := d.Decided(FieldFirmName); ok && got.DecidedBy == ByRule {
		t.Fatalf("addressee read as the firm: %+v", got)
	}
	if got, ok := d.Decided(FieldFirmAddress); ok && got.DecidedBy == ByRule {
		t.Fatalf("addressee's address read as the firm's: %+v", got)
	}
}

// A stored directory firm saved with Markdown emphasis is not reused as-is:
// the read settles the clean name so the store can repair the draft firm.
func TestEmphasisedDirectoryNameIsNotReused(t *testing.T) {
	k := kavanagh[0]
	lookup := func(abns, keys []string) *Firm {
		return &Firm{ID: "f1", Name: "**" + k.name + "**", ABN: k.abn}
	}
	d := NewDraft(Harvest(k.file, kavanaghText(t, k.file)), Context{Discipline: k.discipline, Lookup: lookup}, nil)
	if got, _ := d.Decided(FieldFirmName); got.Value != k.name {
		t.Fatalf("firm name %+v", got)
	}
}
