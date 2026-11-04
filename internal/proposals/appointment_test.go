package proposals

import (
	"context"
	"strings"
	"testing"

	"sitewise/internal/identity"
)

// An owner's email announcing an appointment, with the signed letter's fee
// in an attachment summary. The Chen certifier email had this shape and was
// never read as a fee, 25 October 2026. Names here are invented.
const appointmentEmail = `From: Dana Owner <dana@example.com>
To: Lee Architect <lee@lanternarchitects.example>
Subject: Principal certifier — Northgate Certifiers appointed
Date: 2 June 2026 8:44 AM

Lee,

We have **appointed Northgate Certifiers Pty Ltd as principal certifier** for the project effective today.

Contact: Sam Ng, Senior Certifier — sam.ng@northgatecertifiers.example

Please coordinate the CC pathway with Sam. **Certifier engagement on file** — letter of appointment signed 1 June 2026 attached.

Dana

---

**Attachment summary (Northgate Certifiers letter of appointment):**

- Service: PCA for CC + OC pathway
- Fee: Owner direct — $6,800 + GST (mobilisation + CC stages)
- **Principal certifier appointed:** Northgate Certifiers Pty Ltd (Sam Ng) effective 1 June 2026
`

func mdText(t *testing.T, body string) identity.Text {
	t.Helper()
	text, err := identity.Extract(context.Background(), "md", strings.NewReader(body), int64(len(body)), ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func TestAppointmentWithAFeeIsCued(t *testing.T) {
	if !Cue("12-certifier-appointment.md", mdText(t, appointmentEmail), "correspondence") {
		t.Fatal("an appointment that states its fee was not read as a fee")
	}
}

func TestAppointmentWithoutAFeeIsNotCued(t *testing.T) {
	noFee := strings.Replace(appointmentEmail, "- Fee: Owner direct — $6,800 + GST (mobilisation + CC stages)\n", "", 1)
	if Cue("12-certifier-appointment.md", mdText(t, noFee), "correspondence") {
		t.Fatal("an appointment with no fee line was cued")
	}
	// A fee with no appointment stays as before: correspondence that only
	// mentions a fee is not read.
	chat := "From: a@example.com\nSubject: catch up\n\nThe fee was $500 last time, from memory.\n"
	if Cue("note.md", mdText(t, chat), "correspondence") {
		t.Fatal("correspondence mentioning a fee was cued")
	}
}

func TestAppointedFirmSettlesByRule(t *testing.T) {
	d := NewDraft(Harvest("appointment.md", mdText(t, appointmentEmail)), Context{}, nil)
	got, ok := d.Decided(FieldFirmName)
	if !ok || got.Value != "Northgate Certifiers Pty Ltd" || got.DecidedBy != ByRule {
		t.Fatalf("firm = %+v, want the appointed company by rule", got)
	}
	for _, q := range d.Questions() {
		if q == FieldFirmName {
			t.Fatal("firm name still asked of Jev")
		}
	}
}

func TestTwoAppointedFirmsLeaveTheFirmOpen(t *testing.T) {
	// No firm email either, so only the appointment could name the firm.
	two := strings.Replace(appointmentEmail, " — sam.ng@northgatecertifiers.example", "", 1) + "\nWe also appointed Harbour Survey Pty Ltd as surveyor.\n"
	d := NewDraft(Harvest("appointment.md", mdText(t, two)), Context{}, nil)
	if got, ok := d.Decided(FieldFirmName); ok && got.DecidedBy == ByRule && got.Value != "" {
		t.Fatalf("two appointed firms settled as %+v", got)
	}
}

// The PM's drop on a consultant row says what the document prices, so an
// appointment that states a fee is that consultant's fee.
func TestAppointmentOnAConsultantRowIsAFeeProposal(t *testing.T) {
	pc := Context{PackageID: "p1", Packages: []Package{{ID: "p1", Title: "Certifier", Discipline: "consultant.certifier"}}}
	d := NewDraft(Harvest("appointment.md", mdText(t, appointmentEmail)), pc, nil)
	if got, _ := d.Decided(FieldKind); got.Value != KindFeeProposal || got.DecidedBy != ByRule {
		t.Fatalf("kind = %+v, want fee_proposal by rule", got)
	}
}

// Real proposals and reports say "appointed" in their terms, credentials
// and recommendations; none of these is an appointment (shapes from the
// consultant fee proposal corpus, 25 October 2026).
func TestAppointedInPassingIsNotAnAppointment(t *testing.T) {
	for name, body := range map[string]string{
		"terms":          "# Proposal\n\nRefer the dispute to a mediator appointed by both parties.\n\nStatutory Administration Fee $350.00\n",
		"credentials":    "# Capability\n\nWe were first appointed to the Australia Post national panel in 2014.\n\nOur proposed fee for the above services is $5,000 plus GST.\n",
		"recommendation": "# Tender recommendation\n\nWe recommend Costwise Engineering Pty Ltd to be appointed as civil consultant.\n\nStage 1 fee amounting $36,000.00\n",
		"once appointed": "# Services\n\nPreliminary advice and feedback once appointed.\n\nThe fee is capped at $300,000 for liability.\n",
	} {
		text := mdText(t, body)
		if Cue("report.md", text, "correspondence") {
			t.Errorf("%s: cued as an appointment", name)
		}
		if k := kindRule(Harvest("report.md", text), "consultant.structural"); k != "" {
			t.Errorf("%s: kind settled as %q", name, k)
		}
	}
}
