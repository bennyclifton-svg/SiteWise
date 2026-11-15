# Seven Hills demo corpus

This is the complete synthetic evidence set and operator run book for one SiteWise
demonstration project. It is designed to make a large, credible project legible through
one narrow story: an on-site detention (OSD) requirement is established in planning,
carried into design, exposed as a tender exclusion, revised during construction and then
claimed before the related variation is approved.

**Everything in this folder is fabricated.** The address, land, organisations, people,
contact details, identifiers, proposals, appointments, drawings, reports, tenders,
invoices and advice are fictional. Nothing here is a real customer record, quotation,
professional opinion or construction document.

The public story and exact capture choreography are in the
[storyboard](00-storyboard/README.md) and [run sheet](00-storyboard/run-sheet.md).

---

## Project truth

| Field | Current fact |
| --- | --- |
| Project | 14–18 Wianamatta Avenue, Seven Hills NSW 2147 — explicitly fictional |
| Client | Wianamatta Developments Pty Ltd |
| PM / Superintendent | Ridgeline Project Management Pty Ltd |
| Development | 11 attached two-storey Class 1a townhouses; Torrens subdivision |
| Site | 3,240 m² with a 3.0 m rear drainage easement |
| Delivery | AS 4000–1997 construct-only; no consultant novation |
| Construction budget | $9,800,000 excluding GST |
| Accepted builder | Ironbark Building Group Pty Ltd — $9,340,000 excluding GST |
| Current demo state | Construction month 4; Progress Claim 04 dated 18 August 2026 |

The acquisition and DA evidence deliberately contains older facts. In particular, the
feasibility and DA Rev C position is 12 dwellings; the RFI response and Rev D planning set
reduce the approved scheme to 11. Never rewrite an older source with the current fact.

---

## What is here

There are **140 ingestible project-evidence documents**. The prose documents are only as
long as the evidence requires; the drawing files are lean title-block records containing
the metadata needed to form a credible register.

| Folder | Evidence | Count |
| --- | --- | ---: |
| [`01-briefing-and-planning/`](01-briefing-and-planning/) | Client brief, handover email, deposited-plan and title summaries, desktop geotechnical advice, planning advice, pre-DA record and initial cost advice | 8 |
| [`02-consultant-procurement/`](02-consultant-procurement/) | Three proposals and one separate appointment for each of five disciplines | 20 |
| [`03-consultant-invoices/`](03-consultant-invoices/) | Five invoices for each appointed consultant | 25 |
| [`04-design-documents/`](04-design-documents/) | 52 current drawings, 13 reports and one staged prior structural revision | 66 |
| [`05-planning-and-approvals/`](05-planning-and-approvals/) | Council RFI, consultant responses, Rev D transmittal, determination and conditions tracker | 7 |
| [`06-builder-procurement/`](06-builder-procurement/) | Three whole-builder tenders, one priced clarification and one letter of acceptance | 5 |
| [`07-construction-commercial/`](07-construction-commercial/) | Builder Progress Claims 01–04 | 4 |
| [`08-project-controls/`](08-project-controls/) | Reviewed programme state, structural change, QS advice, programme note and VO-007 request | 5 |
| **Total** | | **140** |

Control material is intentionally outside that total:

- [`00-prompts/`](00-prompts/) contains the seven operator prompts.
- [`00-answer-keys/`](00-answer-keys/) contains deterministic facts, chronology,
  reconciliations and the expected live-change result.
- [`09-email-scenarios/`](09-email-scenarios/) contains payload and staging instructions
  for inbound and outbound Pulse/email proof; these are not project uploads.
- [`04-design-documents/document-register.md`](04-design-documents/document-register.md)
  is the expected register and upload sequence, not project evidence.

### Stage the evidence; do not bulk-upload outcomes

The 140 files span the whole project, but they are not one baseline upload:

1. Establishment: upload the three standalone folder-01 documents named in the run sheet,
   then seed the handover email with its four attachments. Do not duplicate them.
2. Procurement: ingest the 15 proposals, then the first four separate appointment letters;
   proposals alone do not prove appointment.
3. Planning: ingest folder 05 in date order, review 12→11 and create PMP v1. Ingest the
   civil appointment letter for PMP v2, add the inline edit for v3, then process all 25 invoices.
4. Design baseline: follow Run 1 in the design register—51 current drawings plus staged
   S-202 Rev B, and 13 reports. Keep S-202 Rev C out.
5. Tender: ingest the three tenders and Redgum clarification. Keep WD-LOA-001 out until
   Tender Comparison QA and the user decision are complete; then ingest it as award evidence.
   Next ingest `08-project-controls/00-IBG-PROG-B01-reviewed-construction-programme.md` and
   apply the reviewed 5 May 2026 / 58-week programme state.
6. Construction baseline: process Progress Claims 01–03.
7. Change: deliver the email carrying S-202 Rev C, DCN-007, CA-014 and PN-006 exactly once;
   canonical attachment intake replaces Rev B. Apply reviewed updates, then compare the
   application-created CHG-007 record with its control answer key.
8. Claim: ingest the unapproved VO-007 request, then deliver Progress Claim 04 through its
   staged email and open invoice review.

---

## The commercial model

### Consultants

| Discipline | Appointed firm | Fee excl. GST | Invoices |
| --- | --- | ---: | ---: |
| Architectural services | Axis Studio | $286,000 | 5 |
| Town planning | Civic Pattern Planning | $48,000 | 5 |
| Structural engineering | Northline Structures | $132,000 | 5 |
| Civil and stormwater engineering | Catchment Works | $96,000 | 5 |
| Building services engineering | Flux Services | $158,000 | 5 |
| **Total** | | **$720,000** | **25** |

Each discipline has three proposals. An appointment is established only by its separate
appointment letter; no unsuccessful proposal contains later outcome knowledge.

### Builder tender

| Bidder | Submitted excl. GST | 120 m³ OSD evidence | Evidence-backed comparable |
| --- | ---: | --- | ---: |
| Redgum Constructions | $9,080,000 | Explicitly excluded; later priced at $420,000 | $9,500,000 |
| Ironbark Building Group | $9,340,000 | Included | $9,340,000 |
| Calderline Projects | $9,460,000 | Included | $9,460,000 |

The comparison does not infer an OSD allowance or use the house-scale seed benchmark.
Redgum's only adjustment comes from its own clarification. Ironbark's appointment is
proved separately by WD-LOA-001; the arithmetic does not make the decision.

The four builder claims total **$3,408,500 excluding GST**. Progress Claim 04 includes
**VO-007 at $68,500**, expressly unapproved, and must open invoice review rather than imply
approval, certification or a payment schedule.

---

## The live change loop

The final beat starts from a stable PMP v3, Cost Plan and programme. Before the change,
add and save one ordinary human inline edit to the PMP.

1. Stage the inbound structural transmittal described in
   [`09-email-scenarios/01-inbound-structural-transmittal.md`](09-email-scenarios/01-inbound-structural-transmittal.md).
2. Pulse presents S-202 Rev C as superseding Rev B and links the source thread. Receipt
   alone changes no PMP, cost, programme or correspondence state.
3. Review S-202 Rev C, design-change notice DCN-007, QS change advice CA-014 and
   programme note PN-006 together.
4. Explicitly authorise Update PMP and the bounded Cost Plan/programme operations. Confirm
   that the prior inline edit survives and PMP v3 becomes v4.
5. Confirm exactly one **+$68,500 excluding GST forecast variation** and one
   **10-calendar-day OSD redesign and coordination** activity are applied.
6. Create the populated reply as **Draft — not sent**. The user reviews and sends it.
7. Ingest Progress Claim 04 and hold the unapproved VO-007 item in invoice review.

The [live-change answer key](00-answer-keys/live-change-loop.md) records the expected before,
new-evidence and after states. It is never uploaded.

---

## Generate and validate

Run these from the repository root:

```powershell
python docs/demo-corpus/seven-hills/generate-scenario-documents.py
python docs/demo-corpus/seven-hills/generate-commercial.py
python docs/demo-corpus/seven-hills/generate-design-documents.py
python docs/demo-corpus/seven-hills/validate.py
```

The three generators own separate output sets and are safe to rerun. Validation checks
counts, legal identities, naming, commercial facts, revision choreography, forbidden
legacy facts and local Markdown links.

Before a public capture, also complete the product/environment gates in the
[run sheet](00-storyboard/run-sheet.md), including Tender Comparison townhouse acceptance
and honest labelling of any staged email provider.
