"""Generate the narrative and control documents for the Seven Hills demo corpus.

The commercial and drawing registers have their own generators. This script owns the
small set of prose-bearing evidence that makes those registers tell one coherent story:
briefing, planning, the reviewed OSD change, email capture templates and demo prompts.

It overwrites only the named files below and never removes a directory.

    python docs/demo-corpus/seven-hills/generate-scenario-documents.py
"""

from __future__ import annotations

from pathlib import Path


ROOT = Path(__file__).parent
PROJECT = "14–18 Wianamatta Avenue, Seven Hills NSW 2147"
CLIENT = "Wianamatta Developments Pty Ltd"
PM = "Ridgeline Project Management Pty Ltd"
DA = "DA/2025/0733"
SYNTHETIC = (
    "*Synthetic demonstration document. The project, address, land, organisations, "
    "people, identifiers, amounts and advice are fabricated for SiteWise product testing. "
    "It is not a representation of any real property or development.*"
)


def page(title: str, metadata: list[tuple[str, str]], body: str) -> str:
    rows = "\n".join(f"| **{label}** | {value} |" for label, value in metadata)
    return f"""# {title}

| | |
| --- | --- |
{rows}

---

{body.strip()}

---

{SYNTHETIC}
"""


def control(title: str, body: str) -> str:
    return f"""# {title}

{body.strip()}

---

**Control file — do not upload as project evidence.**
"""


FILES: dict[str, str] = {
    "01-briefing-and-planning/01-client-development-brief.md": page(
        "Client Development Brief — Rev A",
        [
            ("Project", PROJECT),
            ("Client", CLIENT),
            ("Prepared by", "M. Kwan, Development Director"),
            ("Date", "2025-02-10"),
            ("Status", "Issued for project establishment"),
        ],
        """
## Purpose

Acquire, design, approve and construct a medium-density residential development across
three amalgamated lots. The feasibility target is **12 attached two-storey dwellings**;
that number is a target, not an approval or a direction to conceal non-compliance.

## Known project position

| Field | Brief position |
| --- | --- |
| Site | Three lots; combined area **3,240 m²** |
| Proposed use | Multi dwelling housing with Torrens subdivision |
| Building classification | Attached Class 1a dwellings, subject to certifier confirmation |
| Planning pathway | Development Application to Blacktown City Council |
| Construction budget | **$9,800,000 excl GST** |
| Delivery | Fully documented construct-only tender; AS 4000 form proposed |
| Target construction start | April 2026 |
| Target practical completion | 30 April 2027 |

## Accommodation and design intent

- Twelve dwellings at feasibility: eight 3-bedroom and four 4-bedroom.
- Two storeys, no basement, private open space to every dwelling.
- Durable brick and fibre-cement envelope; repeatable wet-area and kitchen layouts.
- Accessible path from the street to one adaptable dwelling.
- Waste collection, deep soil, tree canopy and stormwater must be resolved before DA.
- A below-ground OSD solution is anticipated, but its volume, location and construction
  are **not yet evidenced**.

## Project controls required in the PMP

- Monthly owner report covering scope, approvals, cost, programme and decisions.
- A live risk register with named owners and evidence links.
- Consultant appointments are not to be assumed from a proposal or a report.
- Cost movements require a stated basis; contingency is not consumed without approval.
- Programme changes require an explicit activity or dependency change.
- External correspondence may be drafted by SiteWise but is issued only by an authorised
  project representative.

## Initial risks and unknowns

| ID | Item | Current position | Owner |
| --- | --- | --- | --- |
| R-01 | Yield | 12 dwellings is a feasibility target only | Architect / planner |
| R-02 | Deep soil | Applicable area and calculation not confirmed | Planner / landscape |
| R-03 | Founding conditions | Desktop advice only; intrusive testing required | Structural engineer |
| R-04 | OSD | Volume and relationship to the rear easement unknown | Civil engineer |
| R-05 | Waste collection | Vehicle path and storage arrangement unresolved | Architect / waste adviser |
| R-06 | Budget | Early estimate; scope and escalation basis incomplete | PM / QS |

The PMP should preserve each unknown as unknown. It must not convert the feasibility target,
desktop ground assumption or anticipated OSD solution into a confirmed fact.
""",
    ),
    "01-briefing-and-planning/02-acquisition-handover-email.md": page(
        "FW: Wianamatta Ave — what we know so far",
        [
            ("Project", PROJECT),
            ("From", "m.kwan@wianamatta-developments.example"),
            ("To", "projects@ridgeline.example"),
            ("Date", "2025-02-19 07:18 AEDT"),
            ("Subject", "FW: Wianamatta Ave — what we know so far"),
        ],
        """
Team — the vendor folder is attached as it came to us. The useful items are the deposited
plan and title-search summaries, the planner's preliminary advice and the desktop ground
note. The filenames below match the staged Markdown evidence set.

The model still shows 12 dwellings. Please treat that as the acquisition case, not an
instruction. We need an honest position on deep soil, bins, OSD and the fill before the
board confirms the design budget.

Budget discussed with the board was **$9.8m construction excl GST**. I have not seen a cost
plan that supports it. Target is April 2026 on site.

Regards,

Mia

### Attachments as received

- `03-deposited-plan-extract.md`
- `04-title-search-summary.md`
- `05-preliminary-planning-advice.md`
- `06-desktop-geotechnical-advice.md`
""",
    ),
    "01-briefing-and-planning/03-deposited-plan-extract.md": page(
        "Deposited Plan Extract — DP 1187426",
        [
            ("Project", PROJECT),
            ("Document", "Deposited plan extract"),
            ("Reference", "DP 1187426 — Lots 41, 42 and 43"),
            ("Date supplied", "2025-02-11"),
            ("Source", "Vendor due-diligence folder"),
        ],
        """
## Plan particulars

| Lot | Area | Frontage | Title status |
| --- | ---: | ---: | --- |
| 41 | 1,075 m² | 18.0 m | Separate title |
| 42 | 1,080 m² | 18.0 m | Separate title |
| 43 | 1,085 m² | 18.0 m | Separate title |
| **Combined** | **3,240 m²** | **54.0 m** | Amalgamation required before subdivision |

A **3.0 m drainage easement** runs along the rear boundary. The extract records geometry
only. It does not establish the capacity of the drainage system or permit structures
inside the easement.
""",
    ),
    "01-briefing-and-planning/04-title-search-summary.md": page(
        "Title Search Summary — Lots 41–43 DP 1187426",
        [
            ("Project", PROJECT),
            ("Prepared for", CLIENT),
            ("Prepared by", "Ridge Legal Property"),
            ("Date", "2025-02-07"),
            ("Reference", "RLP-25-184"),
        ],
        """
## Interests relevant to project establishment

1. Lots 41, 42 and 43 are held under separate titles and require consolidation or a staged
   subdivision strategy.
2. A 3.0 m-wide easement for drainage burdens the rear boundary of all three lots.
3. No building envelope or covenant authorising an OSD structure within that easement was
   identified in the supplied title material.
4. Service locations, easement capacity and authority requirements remain to be confirmed.

This summary is not a survey and does not replace legal advice on the registered dealings.
""",
    ),
    "01-briefing-and-planning/05-preliminary-planning-advice.md": page(
        "Preliminary Planning Advice — Feasibility Review",
        [
            ("Project", PROJECT),
            ("Prepared by", "Civic Pattern Planning"),
            ("Author", "A. Serrano, Director"),
            ("Date", "2025-02-14"),
            ("Reference", "CPP-250214-WIA"),
        ],
        """
## Advice basis

This early review assumes an R3 Medium Density Residential scenario and a Council DA. The
site-specific statutory position must be confirmed from the current planning certificate,
mapping and instruments before lodgement. Low- and mid-rise Housing SEPP provisions and
exclusions also require a parcel-specific check.

For this synthetic project, the design team is to test the scheme against an applicable
deep-soil requirement of **486 m² (15% of the 3,240 m² site)**. That figure is the project
assessment basis recorded in this advice, not a general statement of current Blacktown
controls.

## Matters requiring evidence

- Twelve dwellings may be supportable but is not established by this desktop review.
- Deep-soil areas must exclude driveways, structures and the below-ground OSD tank.
- Waste storage, transfer path and collection geometry must be shown together.
- The OSD tank must be coordinated with the rear drainage easement and tree-retention zone.
- A traffic and parking assessment, arboricultural assessment, stormwater report and waste
  management plan are expected with the DA package.
- The proposed Torrens subdivision must be tested against the final dwelling layout.

## PMP instruction

Record yield, deep soil, waste and OSD as active approval risks. Do not state that the
12-dwelling scheme complies until the coordinated DA set and specialist reports evidence it.
""",
    ),
    "01-briefing-and-planning/06-desktop-geotechnical-advice.md": page(
        "Desktop Geotechnical Advice",
        [
            ("Project", PROJECT),
            ("Prepared by", "Groundframe Geotechnics"),
            ("Author", "T. Mensah, CPEng"),
            ("Date", "2025-02-18"),
            ("Reference", "GFG-L-250218"),
        ],
        """
Available mapping and nearby records suggest residual soil over weathered shale. Fill of
approximately **0.6–1.0 m may be present**, but no intrusive investigation has been carried
out on the site. That range is an assumption for early risk pricing only.

An intrusive investigation is required before footing type, founding depth, excavation
classification or OSD tank support can be confirmed. The structural engineer should not
adopt the desktop range as a design parameter.
""",
    ),
    "01-briefing-and-planning/07-pre-da-meeting-record.md": page(
        "Pre-DA Meeting Record",
        [
            ("Project", PROJECT),
            ("Council reference", "PDA/2025/0118"),
            ("Meeting date", "2025-04-09"),
            ("Prepared by", "Civic Pattern Planning"),
            ("Status", "Project record — not an approval"),
        ],
        """
## Attendees

Blacktown City Council duty planner and engineering officer; Civic Pattern Planning; Axis
Studio; Ridgeline Project Management Pty Ltd.

## Matters discussed

- Council officers did not endorse the 12-dwelling yield at pre-DA stage.
- The deep-soil calculation must distinguish genuine deep soil from general landscaped area.
- Waste storage and the collection path are to be resolved on one plan.
- A below-ground OSD tank appears likely; volume, access, easement clearance and structural
  support must be coordinated.
- The DA should explain the proposed subdivision pathway and staging.

The meeting was advisory. The assessment officer may raise additional matters after formal
lodgement.
""",
    ),
    "01-briefing-and-planning/08-feasibility-cost-and-programme-advice.md": page(
        "Feasibility Cost and Programme Advice — Rev 1",
        [
            ("Project", PROJECT),
            ("Prepared by", "Measureline Quantity Surveying"),
            ("Date", "2025-02-21"),
            ("Reference", "MQS-250221-R1"),
            ("Pricing basis", "February 2025 feasibility information"),
        ],
        """
## Construction forecast

| Element | Allowance excl GST |
| --- | ---: |
| Preliminaries and site establishment | $845,000 |
| Demolition, earthworks and remediation | $610,000 |
| Substructure and foundations | $920,000 |
| Structure and upper floors | $1,530,000 |
| Envelope, roofing, windows and doors | $1,410,000 |
| Internal finishes and fitout | $1,870,000 |
| Hydraulic, electrical and mechanical services | $1,255,000 |
| External works, landscaping and subdivision works | $900,000 |
| OSD allowance — scope not defined | $350,000 |
| Design development contingency | $380,000 |
| **Forecast construction cost** | **$10,070,000** |

The forecast is **$270,000 above** the client's $9.8m construction budget. Value management
should not remove unresolved statutory scope. OSD and founding allowances are provisional
and must be refreshed when specialist evidence arrives.

## Baseline milestone advice

| Milestone | Date |
| --- | --- |
| Consultant appointments complete | 2025-03-14 |
| DA lodgement target | 2025-07-25 |
| DA determination target | 2025-11-27 |
| Builder tender close | 2026-02-20 |
| Contract award | 2026-03-20 |
| Construction start | 2026-04-13 |
| Practical completion target | 2027-04-30 |

These are planning assumptions, not guarantees. Programme movements require an identified
activity or dependency change rather than a narrative statement alone.
""",
    ),
    "05-planning-and-approvals/01-da-lodgement-acknowledgement.md": page(
        "Development Application Lodgement Acknowledgement",
        [
            ("Project", PROJECT),
            ("Application", DA),
            ("Consent authority", "Blacktown City Council"),
            ("Lodged", "2025-07-25"),
            ("Proposal", "12 attached dwellings and Torrens subdivision — Rev C DA set"),
        ],
        """
The application was accepted for assessment. Acceptance for assessment is not approval and
does not confirm the submitted yield, deep-soil calculation, waste arrangement or OSD
solution.
""",
    ),
    "05-planning-and-approvals/02-council-request-for-information-rfi-01.md": page(
        "Council Request for Information — RFI 01",
        [
            ("Project", PROJECT),
            ("Application", DA),
            ("Issued", "2025-08-21"),
            ("Council-stated response due", "2025-09-18"),
            ("Officer reference", "RFI-01"),
        ],
        """
Please provide a coordinated response addressing the following matters.

1. **Deep soil.** The landscape calculation identifies **412 m²** as deep soil. The project
   planning advice records **486 m²** as the applicable assessment basis. Revise the scheme
   or substantiate the calculation.
2. **Waste.** The submitted 18 m² waste room conflicts with the swept path and collection
   route shown on the architectural plans.
3. **Stormwater.** Confirm the OSD volume, maintenance access and clearance from the rear
   drainage easement. The current architectural and civil drawings are not coordinated.
4. **Yield and subdivision.** Provide an updated schedule of dwellings and proposed lots if
   the design changes in response to the matters above.

The date above is the response date stated in this request. This document does not describe
it as a universal statutory period.
""",
    ),
    "05-planning-and-approvals/03-architect-rfi-design-advice.md": page(
        "Architect Design Advice — RFI 01 Response",
        [
            ("Project", PROJECT),
            ("Prepared by", "Axis Studio"),
            ("Date", "2025-09-02"),
            ("Reference", "AXS-26017-ADV-006"),
            ("Status", "Recommended for client approval"),
        ],
        """
Axis recommends removing Unit 12 and issuing Rev D. The coordinated change would:

- reduce yield from **12 to 11 dwellings**;
- reduce GFA from **1,824 m² to 1,676 m²**;
- increase genuine deep soil from **412 m² to 512 m²**;
- relocate and enlarge the waste room from **18 m² to 27 m²** beside the collection bay;
- retain the 120 m³ OSD tank outside the rear drainage easement; and
- require 20 architectural and two landscape sheets to be reissued.

The client is to approve the yield change. This advice is not itself an instruction to amend
the project profile, cost plan or programme.
""",
    ),
    "05-planning-and-approvals/04-rfi-response-transmittal-rev-d.md": page(
        "RFI 01 Response Transmittal — Rev D",
        [
            ("Project", PROJECT),
            ("Application", DA),
            ("Prepared by", "Axis Studio"),
            ("Issued", "2025-09-12"),
            ("Transmittal", "AXS-26017-TR-014"),
        ],
        """
The Rev D response comprises 20 architectural and two landscape sheets, the revised deep-soil
calculation, waste-management update and coordinated civil advice.

| Controlled fact | Rev C | Rev D |
| --- | ---: | ---: |
| Dwellings | 12 | **11** |
| GFA | 1,824 m² | **1,676 m²** |
| Deep soil | 412 m² | **512 m²** |
| Waste room | 18 m² | **27 m²** |
| OSD | Location conflicted | **120 m³, outside easement** |

Rev C BASIX/energy evidence is superseded by the yield change and requires replacement.
""",
    ),
    "05-planning-and-approvals/05-notice-of-determination.md": page(
        "Notice of Determination — Development Consent",
        [
            ("Project", PROJECT),
            ("Application", DA),
            ("Determined", "2025-11-27"),
            ("Approved development", "11 attached dwellings and Torrens subdivision"),
            ("Consent authority", "Blacktown City Council"),
        ],
        """
## Selected conditions carried into the demo

| Condition | Trigger | Requirement | Evidence to close |
| ---: | --- | --- | --- |
| 6 | Before CC | Updated BASIX/energy commitments for 11 dwellings | Current certificate and stamped plans |
| 12 | Before works | Approved waste collection layout retained | Architect and waste plans |
| 18 | Before works | Construct and certify the approved **120 m³ OSD system** | Civil/structural drawings and certification |
| 23 | During works | Protect the rear drainage easement from building work | Survey and inspection records |
| 31 | Before subdivision certificate | Complete required subdivision and authority works | Compliance certificates |

This extract is intentionally concise. It exists to provide structured, source-backed
approval obligations for the demonstration project.
""",
    ),
    "05-planning-and-approvals/06-section-7-11-contribution-notice.md": page(
        "Section 7.11 Contribution Notice",
        [
            ("Project", PROJECT),
            ("Application", DA),
            ("Issued", "2025-12-04"),
            ("Synthetic assessed amount", "$286,000"),
            ("Payment trigger", "Before Construction Certificate"),
        ],
        """
The assessed contribution is a client-direct project cost and is not part of the builder's
construction tender. The amount is fabricated for this corpus and must not be treated as a
published Council rate or precedent.
""",
    ),
    "05-planning-and-approvals/07-construction-certificate.md": page(
        "Construction Certificate",
        [
            ("Project", PROJECT),
            ("Certificate", "CC/2026/0184"),
            ("Issued", "2026-03-27"),
            ("Principal certifier", "Civic Certifiers"),
            ("Scope", "11 attached Class 1a dwellings and associated site works"),
        ],
        """
The certificate references the current architectural, structural, civil, hydraulic,
electrical, mechanical and landscape registers. It does not certify later revisions unless
they are reviewed and accepted through the project change process.
""",
    ),
    "08-project-controls/00-IBG-PROG-B01-reviewed-construction-programme.md": page(
        "Reviewed Construction Programme Extract — Rev 0",
        [
            ("Project", PROJECT),
            ("Prepared by", "Ironbark Building Group Pty Ltd"),
            ("Reviewed by", PM),
            ("Programme reference", "IBG-PROG-B01"),
            ("Issue date", "2026-05-06"),
            ("Site possession", "2026-05-05"),
            ("Time for completion", "58 calendar weeks"),
            ("Contractual practical completion", "2027-06-15"),
        ],
        """
This is the reviewed working programme state used for project coordination. It records the
accepted contract duration and the activity nodes needed for the OSD change demonstration;
it does not certify progress, an extension of time or critical-path status.

| ID | Activity | Duration | Planned start | Planned finish | Finish-to-start predecessor |
| --- | --- | ---: | --- | --- | --- |
| PRG-000 | Site possession | Milestone | 2026-05-05 | 2026-05-05 | — |
| PRG-210 | Approved civil OSD design — C-201 Rev C | Milestone | 2026-02-20 | 2026-02-20 | — |
| PRG-220 | OSD tank construction | 28 calendar days | 2026-08-24 | 2026-09-20 | PRG-210 |
| PRG-900 | Practical completion | Milestone | 2027-06-15 | 2027-06-15 | Downstream delivery chain from PRG-220 |

If a reviewed coordination activity is inserted at PRG-220's planned start and PRG-220 is
re-sequenced behind it, deterministic finish-to-start recalculation moves the downstream
dates. This statement defines arithmetic behavior only; it grants no time entitlement.
""",
    ),
    "08-project-controls/01-structural-design-change-notice-dcn-007.md": page(
        "Structural Design Change Notice — DCN-007",
        [
            ("Project", PROJECT),
            ("Prepared by", "Northline Structures"),
            ("Date", "2026-08-14"),
            ("Drawing", "S-202 Rev C — OSD Tank Base and Wall Reinforcement"),
            ("Status", "Design advice — commercial entitlement not determined"),
        ],
        """
Construction exposure confirmed variable founding material below the OSD footprint. Rev C
increases the tank base thickness, reinforcement and local piled support relative to the
For Construction Rev B issue dated 27 March 2026.

The design response is required for structural adequacy. This notice records the technical
change only; it does not direct the contractor, approve a variation or establish entitlement.
""",
    ),
    "08-project-controls/02-qs-cost-advice-ca-014.md": page(
        "QS Cost Advice — CA-014",
        [
            ("Project", PROJECT),
            ("Prepared by", "Measureline Quantity Surveying"),
            ("Date", "2026-08-15"),
            ("Change", "S-202 Rev B to Rev C"),
            ("Recommended forecast allowance", "$68,500 excl GST"),
        ],
        """
| Cost component | Allowance excl GST |
| --- | ---: |
| Additional reinforcement | $22,800 |
| Increased slab thickness, concrete and formwork | $18,700 |
| Local piled supports and pile caps | $19,500 |
| Shop-drawing and coordination allowance | $7,500 |
| **Total forecast movement** | **$68,500** |

The amount is a principal-side forecast allowance based on the revised design. It is not an
approved variation and must not be duplicated if the contractor's priced submission is later
accepted into the cost plan.
""",
    ),
    "08-project-controls/03-architect-programme-note-pn-006.md": page(
        "Architect Programme Note — PN-006",
        [
            ("Project", PROJECT),
            ("Prepared by", "Axis Studio"),
            ("Date", "2026-08-15"),
            ("Change", "OSD structural redesign coordination"),
            ("Proposed activity", "10 calendar days"),
            ("Finish-to-start predecessor", "PRG-210 · Approved civil OSD design — C-201 Rev C"),
            ("Successor to re-sequence", "PRG-220 · OSD tank construction"),
        ],
        """
At PRG-220's current planned start, insert a ten-calendar-day activity, **Coordinate and
approve OSD structural revision**, with **PRG-210 · Approved civil OSD design — C-201 Rev C**
as its finish-to-start predecessor. Re-sequence **PRG-220 · OSD tank construction** to start
finish-to-start after the new activity. The note does not assert critical-path or EOT
entitlement. SiteWise should add the activity and recalculate dependent dates only after the
user approves the programme operation.

For the demonstration answer key, the accepted contract programme runs for 58 calendar
weeks from site possession on 5 May 2026. Applying the proposed activity moves the
**forecast completion** from **15 June 2027** to **25 June 2027**. Contractual practical
completion remains **15 June 2027** unless a separate EOT or other authorised contract
adjustment changes it.
""",
    ),
    "08-project-controls/04-builder-variation-request-vo-007.md": page(
        "Builder Variation Request — VO-007",
        [
            ("Project", PROJECT),
            ("Submitted by", "Ironbark Building Group Pty Ltd"),
            ("Date", "2026-08-17"),
            ("Reference", "VO-007"),
            ("Claimed amount", "$68,500 excl GST"),
        ],
        """
Ironbark requests a variation for the S-202 Rev C OSD structural change. The submission uses
the same $68,500 amount as the QS forecast and remains **unapproved** at the date of Progress
Claim 04. Matching amounts do not prove entitlement or approval.
""",
    ),
    "00-answer-keys/reviewed-change-record-chg-007.md": control(
        "Reviewed Project Change Record — CHG-007",
        """
| | |
| --- | --- |
| **Project** | 14–18 Wianamatta Avenue, Seven Hills NSW 2147 |
| **Reviewed by** | Ridgeline Project Management Pty Ltd |
| **Date** | 2026-08-18 |
| **Source set** | S-202 Rev C, DCN-007, CA-014, PN-006 |
| **External status** | No variation approved; reply remains draft |

## User-authorised internal updates

| Surface | Before | Reviewed update |
| --- | --- | --- |
| PMP | OSD structure based on Rev B | Cite Rev C; risk/control row updated |
| Cost Plan | No forecast for CHG-007 | Add **$68,500** forecast allowance |
| Programme | No coordination activity | Add **10-calendar-day** predecessor activity |
| Email | No response | Draft reply saved; **not sent** |

The record demonstrates reviewed propagation. Receipt of the documents did not silently
change the PMP, Cost Plan, programme or external correspondence.
""",
    ),
    "09-email-scenarios/01-inbound-structural-transmittal.md": control(
        "Inbound Email Template — OSD Structural Revision",
        """
Send through the configured demo provider to the project alias.

| Header | Value |
| --- | --- |
| From | `projects@axis-studio.example` |
| To | `wianamatta-avenue@in.sitewise.au` |
| Date | `2026-08-15 16:42 AEST` |
| Subject | `WIA · Structural transmittal · S-202 Rev C · OSD tank` |
| Category expected | `document_transmittal` |

**Body**

Ridgeline — attached is Northline's revised OSD structural sheet and design-change note.
Measureline's cost advice and our programme note are included so the project position can be
reviewed together. Please confirm the internal records before anything is issued to Ironbark.

**Attachments**

- `04-design-documents/drawings/structural/S-202-osd-tank-base-and-wall-reinforcement.md`
- `08-project-controls/01-structural-design-change-notice-dcn-007.md`
- `08-project-controls/02-qs-cost-advice-ca-014.md`
- `08-project-controls/03-architect-programme-note-pn-006.md`

Expected: project match → canonical attachment intake → drawing revision + transmittal merged
into one Pulse card. No PMP, cost, programme or email mutation happens on receipt.
""",
    ),
    "09-email-scenarios/02-outbound-coordination-reply.md": control(
        "Outbound Email Template — Reviewed OSD Change",
        """
Create this as a populated Pi email draft after the internal updates are reviewed.

| Header | Value |
| --- | --- |
| To | `projects@axis-studio.example`; `admin@northline.example`; `pm@ironbark.example` |
| Subject | `RE: WIA · Structural transmittal · S-202 Rev C · OSD tank` |
| Required status before user action | `Draft — not sent` |

**Body**

We have recorded S-202 Rev C and updated the internal PMP, forecast Cost Plan and programme
for review. The Cost Plan carries $68,500 as a forecast allowance and the programme carries a
ten-calendar-day coordination activity. Neither entry approves VO-007 or determines time
entitlement. Please provide Ironbark's substantiation and proposed recovery response.

The user must explicitly issue the draft. A draft record alone is not evidence that the email
was sent.
""",
    ),
    "09-email-scenarios/03-inbound-builder-progress-claim-04.md": control(
        "Inbound Email Template — Progress Claim 04",
        """
| Header | Value |
| --- | --- |
| From | `accounts@ironbark.example` |
| To | `wianamatta-avenue@in.sitewise.au` |
| Date | `2026-08-20 09:06 AEST` |
| Subject | `Wianamatta Avenue · Progress Claim 04 and tax invoice` |
| Attachment | `07-construction-commercial/progress-claims/PC-04-IBG-PC-04.md` |

The claim includes $68,500 against VO-007 while that variation is unapproved. Expected Pulse
state: one `potential_cost_change` card linked to the invoice email, opening the existing
invoice review surface. Do not describe this as automated progress-claim certification.
""",
    ),
    "09-email-scenarios/04-builder-tender-return-wave.md": control(
        "Inbound Email Templates — Builder Tender Returns",
        """
Send the three bidder groups on separate threads so the ordered quote selection can be shown.

| Bidder | Subject | Attachments |
| --- | --- | --- |
| Redgum Constructions Pty Ltd | `WIA tender return · Redgum` | Redgum tender, then OSD clarification on same thread |
| Ironbark Building Group Pty Ltd | `WIA tender return · Ironbark` | Ironbark tender |
| Calderline Projects Pty Ltd | `WIA tender return · Calderline` | Calderline tender |

Expected: attachments follow canonical intake, link to the issued procurement request and
produce tender-received Pulse evidence. The comparison groups Redgum's tender and later
clarification together; it never treats the clarification as a fourth bidder.
""",
    ),
    "09-email-scenarios/05-unanswered-consultant-action.md": control(
        "Inbound Email Template — Action Required",
        """
| Header | Value |
| --- | --- |
| From | `services@flux-engineering.example` |
| To | `wianamatta-avenue@in.sitewise.au` |
| Subject | `Action required · confirm OSD pump power load` |
| Body | `Please confirm the selected OSD pump duty and power load before we close E-200.` |

An ordinary fresh consultant email appears in Pulse's compact other-activity roll-up. This
message becomes an individual unanswered-correspondence card only after five days. `Draft
reply` creates a draft; it does not send it.
""",
    ),
    "00-answer-keys/fact-ledger.md": control(
        "Seven Hills Demo — Fact Ledger",
        """
| Fact | Initial / Rev C | Current / Rev D or construction | Controlling evidence |
| --- | --- | --- | --- |
| Address | 14–18 Wianamatta Avenue — explicitly fictional | unchanged | Brief and DP extract |
| Site | 3,240 m²; rear 3.0 m drainage easement | unchanged | DP/title summaries |
| Yield | 12-dwelling feasibility / DA Rev C | **11 approved dwellings** | RFI response + determination |
| GFA | 1,824 m² | **1,676 m²** | Rev D transmittal |
| Deep soil | 412 m² | **512 m²** | RFI + Rev D response |
| Waste room | 18 m², collection conflict | **27 m²**, coordinated | RFI + Rev D response |
| OSD | anticipated; scope unknown | **120 m³**, outside easement | C-201 + consent condition 18 |
| Contract | proposed AS 4000 | Ironbark, **$9.340m excl GST** | acceptance / tender key |
| OSD structural change | S-202 Rev B | Rev C; **+$68,500 forecast**, **+10 calendar days** | DCN-007, CA-014, PN-006 |
| VO-007 | not raised | $68,500 requested, **unapproved** | VO-007 + PC-04 |

Never substitute a newer fact into an older document. The demo works because SiteWise can
show which position applied at which revision.
""",
    ),
    "00-answer-keys/timeline.md": control(
        "Seven Hills Demo — Master Timeline",
        """
| Date | Event |
| --- | --- |
| 2025-02-10 | Client brief issued — 12-dwelling feasibility target |
| 2025-07-07 | Five consultant appointments complete |
| 2025-07-25 | DA Rev C lodged for 12 dwellings |
| 2025-08-21 | Council RFI 01 issued |
| 2025-09-12 | Rev D response — 11 dwellings, 22 sheets reissued |
| 2025-11-27 | DA approved |
| 2026-03-27 | Construction Certificate and S-202 Rev B issued |
| 2026-04-17 | Three builder tenders returned |
| 2026-04-24 | Redgum prices its stated OSD exclusion in clarification RG-26031-ADD-01 |
| 2026-05-04 | Ironbark accepted at $9.340m excl GST |
| 2026-05-05 | Construction starts |
| 2026-05-06 | Reviewed 58-week programme state records 15 June 2027 completion |
| 2026-08-15 | S-202 Rev C change pack arrives by email |
| 2026-08-18 | Internal PMP, Cost Plan and programme updates reviewed |
| 2026-08-20 | Progress Claim 04 arrives with unapproved VO-007 |
| 2026-08-21 | Demo current date — construction month 4 |
""",
    ),
    "00-answer-keys/live-change-loop.md": control(
        "Seven Hills Demo — Live Change Loop Answer Key",
        """
## Before

- PMP v3 carries the approved 120 m³ OSD control but no S-202 Rev C reference or CHG-007 forecast.
- Cost Plan forecast has no $68,500 line for the OSD structural revision.
- Programme has no ten-day coordination activity; contractual practical completion is 15 June 2027.
- No outbound response exists.

## New evidence

One transmittal email carries S-202 Rev C, DCN-007, QS CA-014 and programme note PN-006.
Pulse should merge the structural drawing revision and transmittal into one attention card.

## Reviewed update

1. Run Update PMP and review the evidence delta and changed section.
2. Preserve an existing user inline edit; save the reviewed PMP as a new version.
3. Apply a typed Cost Plan operation for **$68,500**; Python recalculates the total.
4. Add the **10-calendar-day** programme activity and recalculate dependent dates.
5. Create a populated reply draft. Confirm `Draft — not sent` before any issue action.

## After

- PMP source/control row cites the current change pack.
- Cost forecast moves by exactly $68,500 once, not twice.
- Forecast completion moves to 25 June 2027; contractual practical completion remains
  15 June 2027 and the movement is not labelled a critical-path delay.
- VO-007 remains unapproved.
- Progress Claim 04 triggers invoice review when it claims the same $68,500.
""",
    ),
    "00-prompts/01-project-setup.md": control(
        "Prompt 1 — Establish the Project",
        """
Stage `01-briefing-and-planning/` in the run-sheet order: upload the client brief, pre-DA
record and feasibility advice, then seed the acquisition handover email with its four named
attachments. Do not upload the same attachment twice. Then ask:

> Set this project up from the evidence. Show me what is known, conflicted and missing.

Expected: residential / townhouses / new / NSW; 12 dwellings marked as a feasibility target;
$9.8m budget; DA pathway; April 2026 construction target. OSD volume, intrusive ground
conditions and consultant appointments remain unevidenced. Accept profile proposals only
after reviewing them.
""",
    ),
    "00-prompts/02-pmp-cost-programme-v1.md": control(
        "Prompt 2 — Create the Base Controls",
        """
> Create the Cost Plan and programme from what is currently on file. Keep assumptions visible
> and show the source beside each material project fact. Do not create the PMP yet.

Capture: deterministic Cost Plan total, editable programme and the OSD / ground risks staying
open. The PMP version choreography starts after the first four appointment letters.
""",
    ),
    "00-prompts/03-consultant-procurement.md": control(
        "Prompt 3 — Consultant Procurement and Appointments",
        """
Upload the 15 proposals found under each
`02-consultant-procurement/<discipline>/proposals/` folder.

> Compare the proposals by discipline. Show fee, inclusions, exclusions and project-specific
> gaps. Do not treat the lowest fee as the recommendation.

Then follow the version choreography: ingest and apply the architecture, planning, structural
and building-services appointment letters; ingest folder 05 and review the 12→11 planning
change; create PMP v1; ingest and apply the civil appointment letter for PMP v2; then save one
supported inline edit as PMP v3. Expected selected fees: Axis $286k; Civic Pattern $48k;
Northline $132k; Catchment Works $96k; Flux $158k; total **$720k excl GST**. Appointment
letters, not proposal wording, prove appointment.
""",
    ),
    "00-prompts/04-document-register.md": control(
        "Prompt 4 — Design Document Register",
        """
Follow Run 1 in `04-design-documents/document-register.md`: upload the 13 reports and all
current drawings except S-202 Rev C, then substitute the staged S-202 Rev B baseline. Do not
upload the answer-key register.

> Build the document register by discipline. Preserve drawing number, title, revision, issue
> purpose, date, author and job number. Show superseded revisions separately.

Compare the result with `04-design-documents/document-register.md`. Drawing bodies are not
fabricated; the title-block metadata is the test.
""",
    ),
    "00-prompts/05-builder-tender-comparison.md": control(
        "Prompt 5 — Builder Tender Comparison",
        """
Upload and group the three builder tender sets. Redgum has a base tender plus clarification.

> Compare the three builder tenders against the project scope. Reconcile each printed total,
> show stated exclusions and items not explicitly itemised, and keep every finding tied to
> the bidder page or project source.

Expected: Redgum $9.080m submitted, explicit OSD exclusion, +$420k clarification = $9.500m;
Ironbark $9.340m; Calderline $9.460m. The user selects Ironbark. SiteWise must not claim a
benchmark adjustment or recommend a builder.
""",
    ),
    "00-prompts/06-live-pmp-change.md": control(
        "Prompt 6 — Pulse, Update PMP, Cost and Programme",
        """
Seed/send the email in `09-email-scenarios/01-inbound-structural-transmittal.md`.

> Review the new OSD structural evidence. Update only the affected PMP content, show the
> change, then apply the evidenced cost and programme movements. Draft the coordination reply
> but do not send it.

Expected: evidence delta = the new attachment set; prior inline edit preserved; new PMP
version; one $68,500 cost movement; one ten-calendar-day programme activity; populated email
draft visibly `not sent`. No automatic propagation is claimed.
""",
    ),
    "00-prompts/07-builder-invoice-review.md": control(
        "Prompt 7 — Builder Invoice Review",
        """
Seed/send Progress Claim 04 after VO-007 exists but before it is approved.

> Review Ironbark Progress Claim 04 against the contract and approved cost position. Show me
> anything that needs a decision; do not approve, reject, pay or certify it.

Expected Pulse: one potential-cost-change card. Invoice review identifies the **$68,500
unapproved variation**. Use Hold/Reject/Approve only as explicit human actions. This is an
invoice-review demonstration, not a claim-certification or payment-schedule workflow.
""",
    ),
}


def write(relative_path: str, content: str) -> None:
    target = (ROOT / relative_path).resolve()
    if not target.is_relative_to(ROOT.resolve()):
        raise ValueError(f"refusing to write outside corpus: {relative_path}")
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(content, encoding="utf-8")


def main() -> None:
    for relative_path, content in FILES.items():
        write(relative_path, content)

    evidence = [
        path
        for path in FILES
        if path.startswith(("01-", "05-", "08-"))
    ]
    controls = [path for path in FILES if path not in evidence]
    assert len(evidence) == 20
    assert len(controls) == 16
    assert "$68,500" in FILES["00-answer-keys/live-change-loop.md"]
    assert "10 calendar days" in FILES["08-project-controls/03-architect-programme-note-pn-006.md"]

    print(f"wrote {len(FILES)} scenario files")
    print(f"  evidence: {len(evidence)}")
    print(f"  controls: {len(controls)}")


if __name__ == "__main__":
    main()
