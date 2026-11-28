import { useEffect, useState, type ReactNode } from "react";
import { ApiError } from "./api";
import { fmt } from "./CostGrid";
import { dialogOf, useChatTable } from "./chatView";
import { emptySelection, selectAll } from "./table/selection";
import { invoiceApi, type BreakdownLine, type Claim, type Contract, type DueDate, type Ledger, type OpCall, type Sheet, type SheetHeader, type SheetLine, type SheetSection } from "./invoiceApi";
import { BackButton, MoreMenu, type MenuItem } from "./PageActions";
import { PanelHead, PrimaryButton } from "./PanelHead";
import { ALERT, cents, claimStatus, day, nilDash } from "./ProgressClaims";
import { Band } from "./RegisterSheet";
import "./payment-schedule.css";

// A claim's payment schedule as one sheet in the middle panel
// (docs/plans/2026-11-26-payment-schedules.md, section 6.3), laid out in the
// order the owner's own schedule prints: details, works, provisional sums,
// variations, the total and retention, the payment list. The PM types only
// what is complete to date, as a percent or as dollars, and a reason where
// that is less than claimed; every other figure is the server's. Each edit
// goes through POST /ops and has one Undo. Issue is the PM's press only.
//
// The lines themselves are the contract's breakdown (slice 3). "Edit
// breakdown" in the three dots turns each line's words and sum into fields:
// rename, note, sum, works or provisional sum, remove, and a grey row to add
// one. A provisional sum carries a Final tick whenever the sheet is open.

const SECTION: Record<SheetSection["kind"], [title: string, sum: string]> = {
  works: ["Contract summary: works", "Total let ($)"],
  provisional: ["Provisional sum adjustment", "Sum ($)"],
  variations: ["Variations", "Sum ($)"],
  unplaced: ["Not placed on the contract", ""],
};
/** One set of column widths, so every section's figures line up. */
const COLS = <colgroup><col /><col className="ps-c-num" /><col className="ps-c-pct" /><col className="ps-c-done" /><col className="ps-c-num" /><col className="ps-c-num" /></colgroup>;
const plain = (s: string) => s.replace(/[,$%\s]/g, "").replace("−", "-");
/** A sheet line's key names its breakdown line (b:), its cost line (l:) or its variation (v:). */
const keyId = (key: string) => key.slice(2);

interface Props {
  projectId: string;
  ledger: Ledger | null;
  claimId: string;
  onClose: () => void;
  reload: () => void;
  onSignedOut: () => void;
}

export function PaymentSchedule({ projectId, ledger, claimId, onClose, reload, onSignedOut }: Props) {
  const claim = ledger?.claims?.find(c => c.invoice_id === claimId);
  const contract = ledger?.contracts?.find(c => c.package_id === claim?.package_id);
  const sheet = ledger?.sheet?.claim_id === claimId ? ledger.sheet : undefined;
  const [busy, setBusy] = useState(false), [error, setError] = useState("");
  const [last, setLast] = useState<{ change: string; label: string } | null>(null);
  const [selection, setSelection] = useState(emptySelection);
  const [editing, setEditing] = useState(false);
  const [terms, setTerms] = useState(false);
  const rowIds = claim?.rows.map(r => r.id) ?? [];
  useChatTable("claim_rows", "costs", {
    select: list => setSelection(selectAll(list.filter(id => rowIds.includes(id)))),
    only: list => setSelection(selectAll(list.filter(id => rowIds.includes(id)))),
    clear: () => setSelection(emptySelection),
  }, selection.ids, !!claim);
  useEffect(() => {
    const key = (e: KeyboardEvent) => { if (e.key === "Escape" && !(e.target as HTMLElement).closest("input,textarea,select,[role=menu],[role=dialog],.project-side-panel")) onClose(); };
    document.addEventListener("keydown", key);
    return () => document.removeEventListener("keydown", key);
  }, [onClose]);

  const number = claim?.display_number || "";
  const crumbs = <BackButton to="Progress claims" onClick={onClose} title="Back to Progress claims (Esc)" here={`Schedule ${number}`.trim()} />;
  if (!ledger) return <section className="ps" aria-label="Payment schedule"><PanelHead crumbs={crumbs} title="Payment schedule" /><p role="status">Opening…</p></section>;
  if (!claim || !contract) return <section className="ps" aria-label="Payment schedule"><PanelHead crumbs={crumbs} title="Payment schedule" /><p className="reg-empty">This claim is no longer on a live contract.</p></section>;

  const issued = claim.issued.length > 0;
  const latest = claim.issued[claim.issued.length - 1];
  async function run(work: () => Promise<{ change_id?: string } | void>, label: string) {
    if (busy) return;
    setBusy(true); setError("");
    try {
      const out = await work();
      setLast(out && out.change_id ? { change: out.change_id, label } : null);
      reload();
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
      setError(e instanceof ApiError && e.message ? e.message : "Could not save. Try again.");
    } finally { setBusy(false); }
  }
  const apply = (ops: OpCall[], label: string) => run(() => invoiceApi.ops(projectId, "claims", ops), label);
  const assess = (row: string, fields: Record<string, unknown>, label: string) => apply([{ op: "claims.assess", fields: { claim: claim.invoice_id, row, ...fields } }], label);
  const edit = (field: string, value: string) => run(async () => { await invoiceApi.edit(projectId, claim.invoice_id, { [field]: value }); }, "");
  const issue = () => run(async () => { await invoiceApi.issueSchedule(projectId, claim.invoice_id); }, "");
  // Terms are typed once: the contract's on its award, the parties on the project.
  const setContract = (body: Record<string, string>) => run(async () => { await invoiceApi.terms(projectId, contract.package_id, body); }, "");
  const setParty = (field: string, value: string) => run(() => invoiceApi.ops(projectId, "delivery", [{ op: "delivery.settings", fields: { [field]: value || null } }]), "Saved the project's terms.");
  const move = (fields: Record<string, string>) => apply([{ op: "claims.assess", fields: { claim: claim.invoice_id, ...fields } }], "Retention saved.");
  // The contract's breakdown: each edit is one operation with Undo.
  const breakdown: Breakdown = {
    lines: contract.breakdown ?? [],
    edit: (line, fields, label) => void apply([{ op: "claims.breakdown_edit", fields: { package: contract.package_id, line, ...fields } }], label),
    add: (section, label, amount) => void apply([{ op: "claims.breakdown_add", fields: { package: contract.package_id, section, label, ...(amount ? { amount } : {}) } }], `Added ${label}.`),
  };
  // Once a schedule is issued on the contract, "less prior" rests on its lines.
  const rebuildable = !(ledger.claims ?? []).some(c => c.package_id === contract.package_id && c.issued.length > 0);

  // Send opens the one Send dialog filled in: the contractor and the
  // principal, the issued PDF. The PM presses Send there.
  const sent = claim.schedule_send;
  const address = (name: string, email: string) => !email ? "" : name && !/[,<>"]/.test(name) ? `${name} <${email}>` : email;
  const send = () => dialogOf("send")?.open({ claim: claim.invoice_id, subject: `Payment schedule ${number} · ${contract.title}`,
    to: [address(contract.firm, sheet?.header?.contractor_email ?? ""), address(sheet?.header?.principal ?? "", sheet?.header?.principal_email ?? "")].filter(Boolean).join(", ") });
  const blocked = claim.unassessed ? `${claim.unassessed} ${claim.unassessed === 1 ? "line is" : "lines are"} not assessed` : claim.missing_reasons ? `${claim.missing_reasons} ${claim.missing_reasons === 1 ? "line needs" : "lines need"} a reason` : claim.retention_open ? "Retention needs an answer" : "";
  const items: MenuItem[] = [
    { label: "Accept claimed", icon: null, disabled: busy || issued, onSelect: () => void apply([{ op: "claims.accept_claimed", fields: { claim: claim.invoice_id } }], `Accepted claim ${number} as claimed.`) },
    { label: "Open the claim", icon: null, href: invoiceApi.fileURL(projectId, claim.invoice_id), newTab: true },
    { label: editing ? "Done editing the breakdown" : "Edit breakdown", icon: null, onSelect: () => setEditing(e => !e) },
    { label: terms ? "Close contract terms" : "Contract terms", icon: null, onSelect: () => setTerms(t => !t) },
    ...(latest ? [{ label: "Download PDF", icon: null, href: latest.file, download: true }] : []),
    { label: "Print", icon: null, onSelect: () => window.print() },
    ...(issued && sent?.status === "sent" ? [{ label: "Send again", icon: null, onSelect: send }] : []),
    ...(issued ? [{ label: "Issue again", icon: null, disabled: busy || !!blocked, onSelect: () => void issue() }] : []),
  ];
  const retentionRow = claim.rows.find(r => r.kind === "retention");
  const due = !issued && (claim.schedule_due.date || claim.alert) ? `Schedule due ${claim.schedule_due.date ? day(claim.schedule_due.date) : "unknown"}` : "";

  return <section className="ps" aria-label={`Payment schedule ${number}`} aria-busy={busy}>
    <PanelHead className="ps-head" crumbs={crumbs} title={<>Payment schedule {number} · {contract.title}{contract.firm ? ` · ${contract.firm}` : ""}</>}
      action={!issued ? <PrimaryButton disabled={busy || !!blocked} title={blocked || undefined} onClick={() => void issue()}>Issue payment schedule</PrimaryButton>
        : sent?.status !== "sent" && sent?.status !== "queued" ? <PrimaryButton disabled={busy} onClick={send}>Send</PrimaryButton> : undefined}
      more={<MoreMenu label={`Schedule ${number} actions`} items={items} />}>
      <span className="register-chip">{claimStatus(claim)}</span>
      {due && <span className="register-chip" data-band={claim.alert === "overdue" ? "late" : claim.alert || claim.schedule_due.to_check ? "amber" : undefined}>{due}{claim.alert ? ` · ${ALERT[claim.alert].toLowerCase()}` : claim.schedule_due.to_check ? " · to check" : ""}</span>}
      {latest && <span className="register-chip">Issued {day(latest.issued_on)} · version {latest.number}</span>}
      {sent?.status === "failed" && <span className="register-chip" data-band="late">Not sent · see Messages</span>}
    </PanelHead>
    {error && <p role="alert" className="inv-error">{error}</p>}
    {last && <p className="reg-notice" role="status">{last.label} <button type="button" className="cell-link" disabled={busy} onClick={() => { const c = last.change; setLast(null); void invoiceApi.undo(projectId, c).then(reload).catch(() => setError("Can't undo: this has changed since.")); }}>Undo</button></p>}
    {!issued && blocked && <p className="ps-check" role="status">{blocked} before Issue.</p>}
    {claim.flags.length > 0 && <ul className="inv-flags" aria-label="Checks">{claim.flags.map((f, i) => <li key={`${f.code}:${i}`}>{f.reason}</li>)}</ul>}

    <Details claim={claim} contract={contract} header={sheet?.header} issued={issued} busy={busy} edit={edit} />
    {terms && <TermsPanel contract={contract} header={sheet?.header} busy={busy} setContract={setContract} setParty={setParty} periods={body => run(async () => { await invoiceApi.periods(projectId, contract.package_id, body); }, "")} onClose={() => setTerms(false)} />}

    {!sheet ? <p role="status">Opening the schedule…</p> : <>
      {sheet.notes?.length > 0 && <ul className="inv-flags" aria-label="Breakdown checks">{sheet.notes.map(n => <li key={n}>{n}</li>)}</ul>}
      {editing && <p className="ps-editing" role="status">Editing {contract.title}'s breakdown: the lines every schedule on this contract prints.
        <button type="button" className="btn btn-small" disabled={busy || !rebuildable} title={rebuildable ? "Replace the lines with the awarded tender's own lines, where they add up" : "A schedule is issued on this contract, so its lines stay"}
          onClick={() => void apply([{ op: "claims.breakdown_rebuild", fields: { package: contract.package_id } }], "Rebuilt the breakdown from the tender.")}>Rebuild from the tender</button>
        <button type="button" className="btn btn-small" onClick={() => setEditing(false)}>Done</button></p>}
      {sections(sheet, editing).map(sec => <SectionTable key={sec.kind} section={sec} contract={contract} locked={issued || busy} busy={busy} editing={editing} breakdown={breakdown} selected={selection.ids} assess={assess} />)}
      <Totals key={`${sheet.retention.held}:${sheet.retention.open}`} sheet={sheet} locked={issued || busy} held={retentionRow?.assessed_to_date ?? null}
        onHeld={v => void assess(retentionRow?.id ?? "retention", { assessed_to_date: v }, "Retention saved.")} onMove={f => void move(f)}
        why={retentionRow ? { below: retentionRow.below_claimed, reason: retentionRow.reason } : null} onWhy={v => void assess(retentionRow?.id ?? "retention", { reason: v }, "Reason saved.")} />
      <Band title="Note" className="ps-note">
        <p>{sheet.paid_in_full ? "Progress claim to be paid in full." : claim.unassessed ? "Not every line is assessed yet." : "Scheduled for less than claimed. The reasons print with the schedule."}</p>
        {issued ? claim.schedule_note && <p>{claim.schedule_note}</p> : <textarea className="ps-note-own" aria-label="Your note, printed on the schedule" key={`note:${claim.schedule_note}`} defaultValue={claim.schedule_note} maxLength={500} disabled={busy}
          placeholder="Your own words, printed after the sentence above" onBlur={e => { if (e.target.value.trim() !== claim.schedule_note) void edit("schedule_note", e.target.value.trim()); }} />}
      </Band>
      <Payments sheet={sheet} />
    </>}

    {sheet?.header && <Band title="Approval" className="ps-approval">
      <dl className="ps-grid">
        <div><Row label="Approved by">{sheet.header.approver}<span className="ps-src">{sheet.header.approver_typed ? <><span className="ps-dot" data-origin="user" />Project terms</> : "your sign-in"}</span></Row></div>
        <div><Row label="Date">{latest ? day(latest.issued_on) : <span className="ps-src">the day it is issued</span>}</Row></div>
      </dl>
      <p className="ps-src">The PDF leaves a line to sign.</p>
    </Band>}

    {issued && <Band title="Issued" className="ps-issued">
      <ul>{claim.issued.map(i => <li key={i.version_id}>Version {i.number}, issued {day(i.issued_on)} · <a href={i.file}>Download PDF</a></li>)}</ul>
      <p className="ps-src">{sent?.status === "sent" ? `Sent from SiteWise on ${day(sent.sent_on)}; the message is in Correspondence.`
        : sent?.status === "queued" ? "Sending…"
        : "Send it from here with the PDF attached, or download it and send it yourself, then enter the day it was served."}</p>
      <label className="ps-served">Schedule served on <input type="date" aria-label="Schedule served on" key={`ss:${claim.schedule_served_on}`} defaultValue={claim.schedule_served_on}
        onBlur={e => { if (e.target.value !== claim.schedule_served_on) void edit("schedule_served_on", e.target.value); }} /></label>
      {claim.schedule_served_on && !claim.schedule_served_confirmed && <p className="ps-flag">To confirm: the day its message was sent. <button type="button" className="cell-link" disabled={busy} onClick={() => void edit("schedule_served_on", claim.schedule_served_on)}>Confirm {day(claim.schedule_served_on)}</button></p>}
    </Band>}
  </section>;
}

// The schedule's header, in the order the owner's own prints it: the
// parties on the left, the claim on the right. Each value says where it
// came from; nothing is filled in for the PM.
function Details({ claim, contract, header, issued, busy, edit }: { claim: Claim; contract: Contract; header?: SheetHeader; issued: boolean; busy: boolean; edit: (field: string, value: string) => void }) {
  const number = claim.display_number;
  const abn = (n: string, word = "ABN") => n ? ` (${word} ${n})` : "";
  const companyNo = (n: string) => abn(n, n.replace(/\D/g, "").length === 11 ? "ABN" : "ACN");
  const from = (typed: boolean, standIn: string) => <span className="ps-src">{typed ? <><span className="ps-dot" data-origin="user" />Project terms</> : standIn}</span>;
  const warned = (n: string) => !!header?.warnings.some(w => n !== "" && w.startsWith(n));
  return <Band title="Details" className="ps-details">
    <dl className="ps-grid">
      <div>
        <Row label="Project">{header?.project ? `${header.project} · ${contract.title}` : contract.title}</Row>
        {header?.address && <Row label="Address">{header.address}</Row>}
        <Row label="Principal" check={warned(header?.principal_abn ?? "")}>{header?.principal ? <>{header.principal}{abn(header.principal_abn)}{from(header.principal_typed, "the Profile's client")}</> : <span className="ps-src">not entered · Contract terms</span>}</Row>
        <Row label="Contractor">{contract.firm ? <>{contract.firm}{abn(header?.contractor_abn ?? contract.firm_abn)}<span className="ps-src">the firm's record</span></> : <span className="ps-src">no firm on the award</span>}</Row>
        <Row label="Principal's rep" check={warned(header?.assessor_company_no ?? "")}>{header?.assessor_org ? <>{header.assessor_org}{companyNo(header.assessor_company_no)}{from(header.assessor_typed, "your organisation's name")}</> : <span className="ps-src">not entered · Contract terms</span>}</Row>
        {header?.warnings.map(w => <p key={w} className="ps-flag">To check: {w} It prints as typed.</p>)}
      </div>
      <div>
        <Row label="Claim no.">{issued ? number : <input className="field-plain" aria-label="Claim number" key={`n:${claim.number}`} defaultValue={claim.number} placeholder={number} maxLength={100} disabled={busy} onBlur={e => { if (e.target.value.trim() !== claim.number) edit("number", e.target.value.trim()); }} />}</Row>
        <Row label="Claim received" check={!claim.served_confirmed}>
          {issued ? day(claim.served_on) : <input type="date" className="field-plain" aria-label="Claim received on" key={`s:${claim.served_on}`} defaultValue={claim.served_on} disabled={busy} onBlur={e => { if (e.target.value && (e.target.value !== claim.served_on || !claim.served_confirmed)) edit("served_on", e.target.value); }} />}
          {!claim.served_confirmed && <span className="ps-how">The day it was dropped here. Confirm the day it was served.</span>}
        </Row>
        <Row label="Period">{claim.period}</Row>
        <Row label="Reference date">{issued ? day(claim.reference_date) : <input type="date" className="field-plain" aria-label="Reference date" key={`r:${claim.reference_date}`} defaultValue={claim.reference_date} disabled={busy} onBlur={e => { if (e.target.value !== claim.reference_date) edit("reference_date", e.target.value); }} />}</Row>
        <Row label="Invoice no.">{claim.invoice_no ? `${claim.invoice_no}${claim.invoice_date ? ` · ${day(claim.invoice_date)}` : ""}` : <span className="ps-src">awaiting invoice</span>}</Row>
        <Row label="Contract date">{contract.contract_date ? <>{day(contract.contract_date)}<span className="ps-src"><span className="ps-dot" data-origin="user" />Contract terms</span></> : <span className="ps-src">not entered · Contract terms</span>}</Row>
        <Due label="Schedule due" due={claim.schedule_due} />
        <Due label="Payment due" due={claim.payment_due} />
      </div>
    </dl>
    <p className="ps-act">{header?.act ? `This is a payment schedule made under the ${header.act}.` : "This is a payment schedule."}</p>
  </Band>;
}

// Terms typed once: the contract's own (its date, retention and periods) on
// its award, and the parties on the project. A field saves when it is left.
function TermsPanel({ contract, header, busy, setContract, setParty, periods, onClose }: { contract: Contract; header?: SheetHeader; busy: boolean; setContract: (body: Record<string, string>) => void; setParty: (field: string, value: string) => void;
  periods: (body: { schedule_days: number | null; schedule_basis: string; payment_days: number | null; payment_basis: string }) => void; onClose: () => void }) {
  const typed = header?.typed ?? {};
  const enter = (e: React.KeyboardEvent<HTMLInputElement>) => { if (e.key === "Enter") e.currentTarget.blur(); };
  const days = (v: string) => v.trim() === "" ? null : Number(v);
  const savePeriods = (next: Partial<{ schedule_days: string; schedule_basis: string; payment_days: string; payment_basis: string }>) => periods({
    schedule_days: next.schedule_days !== undefined ? days(next.schedule_days) : contract.schedule_days, schedule_basis: next.schedule_basis ?? contract.schedule_basis,
    payment_days: next.payment_days !== undefined ? days(next.payment_days) : contract.payment_days, payment_basis: next.payment_basis ?? contract.payment_basis });
  const party = (field: string, label: string, hint = "") => <Row label={label}><input className="field-plain ps-term" aria-label={label} key={`${field}:${typed[field] ?? ""}`} defaultValue={typed[field] ?? ""} placeholder={hint || "not entered"} disabled={busy}
    onBlur={e => { if (e.target.value.trim() !== (typed[field] ?? "")) setParty(field, e.target.value.trim()); }} onKeyDown={enter} /></Row>;
  return <Band title="Contract terms" className="ps-terms">
    <p className="ps-src ps-terms-lead">Typed once. The contract's own terms apply to every schedule on {contract.title}; the parties apply to every contract on the project. SiteWise has no default retention rate and no release rule. <button type="button" className="cell-link" onClick={onClose}>Close</button></p>
    <dl className="ps-grid">
      <div>
        <Row label="Contract date"><input type="date" className="field-plain" aria-label="Contract date" key={`cd:${contract.contract_date}`} defaultValue={contract.contract_date} disabled={busy} onBlur={e => { if (e.target.value !== contract.contract_date) setContract({ contract_date: e.target.value }); }} /></Row>
        <Row label="Retention"><input className="field-plain ps-term-short" inputMode="decimal" aria-label="Retention rate, percent" key={`rp:${contract.retention_percent}`} defaultValue={contract.retention_percent} placeholder="none" disabled={busy}
          onBlur={e => { const v = e.target.value.replace(/[%\s]/g, ""); if (v !== contract.retention_percent) setContract({ retention_percent: v }); }} onKeyDown={enter} /> % of{" "}
          <select className="field-plain" aria-label="Retention is a percent of" value={contract.retention_base || "works"} disabled={busy || !contract.retention_percent} onChange={e => setContract({ retention_base: e.target.value })}>
            <option value="works">the works sub total</option><option value="let">works and provisional sums</option><option value="to_date">the contract sum to date</option></select></Row>
        <Row label="Names the Act"><select className="field-plain" aria-label="Name the Security of Payment Act on this contract's schedules" value={contract.act_applies} disabled={busy} onChange={e => setContract({ act_applies: e.target.value })}>
          <option value="">{contract.services ? "No (a consultant's contract)" : "Yes (a works contract)"}</option><option value="yes">Yes</option><option value="no">No</option></select>
          <span className="ps-how">Only from a state row the owner has verified, and never with a section number.</span></Row>
        <Row label="Schedule within"><input className="field-plain ps-term-short" inputMode="numeric" aria-label="Payment schedule days" key={`sd:${contract.schedule_days}`} defaultValue={contract.schedule_days ?? ""} placeholder="the Act's" disabled={busy}
          onBlur={e => { if (e.target.value.trim() !== String(contract.schedule_days ?? "")) savePeriods({ schedule_days: e.target.value }); }} onKeyDown={enter} />{" "}
          <select className="field-plain" aria-label="Payment schedule day basis" value={contract.schedule_basis} disabled={busy} onChange={e => savePeriods({ schedule_basis: e.target.value })}><option value="business">business days</option><option value="calendar">calendar days</option></select></Row>
        <Row label="Payment within"><input className="field-plain ps-term-short" inputMode="numeric" aria-label="Payment days" key={`pd:${contract.payment_days}`} defaultValue={contract.payment_days ?? ""} placeholder="the Act's" disabled={busy}
          onBlur={e => { if (e.target.value.trim() !== String(contract.payment_days ?? "")) savePeriods({ payment_days: e.target.value }); }} onKeyDown={enter} />{" "}
          <select className="field-plain" aria-label="Payment day basis" value={contract.payment_basis} disabled={busy} onChange={e => savePeriods({ payment_basis: e.target.value })}><option value="business">business days</option><option value="calendar">calendar days</option></select></Row>
      </div>
      <div>
        {party("principal_name", "Principal", header && !header.principal_typed ? header.principal : "")}
        {party("principal_abn", "Principal's ABN")}
        {party("principal_email", "Principal's email")}
        {party("assessor_org", "Principal's rep", header && !header.assessor_typed ? header.assessor_org : "")}
        {party("assessor_company_no", "Rep's ACN or ABN")}
        {party("assessor_name", "Approved by", header && !header.approver_typed ? header.approver : "")}
      </div>
    </dl>
  </Band>;
}

function Row({ label, check, children }: { label: string; check?: boolean; children: ReactNode }) {
  return <div className="ps-row" data-band={check ? "amber" : undefined}><dt>{label}</dt><dd>{children}</dd></div>;
}

/** A date Go worked out: the date, "to check" while it is not settled, and what it was worked out from. */
function Due({ label, due }: { label: string; due: DueDate }) {
  return <Row label={label} check={due.to_check}>
    {due.date ? day(due.date) : "Unknown"}{due.to_check && <span className="ps-src">to check</span>}
    <span className="ps-how">{due.words}{due.section && !due.words.includes(due.section) ? ` (${due.section})` : ""}{due.notes.map(n => ` ${n}`)}</span>
  </Row>;
}

type Assess = (row: string, fields: Record<string, unknown>, label: string) => void;
interface Breakdown {
  lines: BreakdownLine[];
  edit: (line: string, fields: Record<string, unknown>, label: string) => void;
  add: (section: "works" | "provisional", label: string, amount: string) => void;
}

/** While the breakdown is edited, works and provisional sums both show, so a line can be added to either. */
function sections(sheet: Sheet, editing: boolean): SheetSection[] {
  if (!editing) return sheet.sections;
  const blank = (kind: SheetSection["kind"]): SheetSection => ({ kind, lines: [], total_let: "0.00", completed: "0.00", prior: "0.00", this_claim: "0.00" });
  const has = (kind: SheetSection["kind"]) => sheet.sections.find(x => x.kind === kind);
  return [has("works") ?? blank("works"), has("provisional") ?? blank("provisional"), ...sheet.sections.filter(x => x.kind !== "works" && x.kind !== "provisional")];
}

// One section in the schedule's own columns. The section's name is its
// table's first heading, so each section wears one band, not two.
function SectionTable({ section, contract, locked, busy, editing, breakdown, selected, assess }: { section: SheetSection; contract: Contract; locked: boolean; busy: boolean; editing: boolean; breakdown: Breakdown; selected: readonly string[]; assess: Assess }) {
  const [title, sum] = SECTION[section.kind];
  const unplaced = section.kind === "unplaced";
  const own = section.kind === "works" || section.kind === "provisional";
  const [draft, setDraft] = useState({ label: "", amount: "" });
  function add() {
    if (!own || !draft.label.trim()) return;
    breakdown.add(section.kind as "works" | "provisional", draft.label.trim(), plain(draft.amount));
    setDraft({ label: "", amount: "" });
  }
  return <div className="inv-table-scroll ps-section"><table className="inv-table var-table ps-table">
    {COLS}
    <thead>
      <tr><th scope="col" rowSpan={2} className="ps-band">{title}</th><th scope="col" rowSpan={2} className="num">{sum}</th><th scope="colgroup" colSpan={2} className="ps-ctr">Completed to date</th><th scope="col" rowSpan={2} className="num">Less prior ($)</th><th scope="col" rowSpan={2} className="num">This claim ($)</th></tr>
      <tr><th scope="col" className="num">(%)</th><th scope="col" className="num">($)</th></tr>
    </thead>
    <tbody>
      {section.lines.map(l => <Line key={l.key || l.row_id} line={l} contract={contract} kind={section.kind} locked={locked} busy={busy} editing={editing && own} breakdown={breakdown} selected={!!l.row_id && selected.includes(l.row_id)} assess={assess} />)}
      {editing && own && <tr className="cost-ghost ps-ghost">
        <td><input aria-label={section.kind === "works" ? "New works line" : "New provisional sum"} placeholder={section.kind === "works" ? "Add a works line" : "Add a provisional sum"} value={draft.label} maxLength={500} disabled={busy}
          onChange={e => setDraft({ ...draft, label: e.target.value })} onKeyDown={e => { if (e.key === "Enter") add(); }} /></td>
        <td className="num"><input aria-label={section.kind === "works" ? "New works line sum" : "New provisional sum amount"} inputMode="decimal" placeholder="sum" value={draft.amount} disabled={busy}
          onChange={e => setDraft({ ...draft, amount: e.target.value })} onKeyDown={e => { if (e.key === "Enter") add(); }} /></td>
        <td /><td /><td /><td />
      </tr>}
      <tr className="ps-subtotal"><th scope="row">Sub total ex GST ($)</th><td className="num">{unplaced ? "" : section.total_let == null ? "unknown" : nilDash(section.total_let)}</td><td /><td className="num">{section.completed == null ? "" : nilDash(section.completed)}</td><td className="num">{nilDash(section.prior)}</td><td className="num">{section.this_claim == null ? "" : nilDash(section.this_claim)}</td></tr>
    </tbody>
  </table></div>;
}

function Line({ line, contract, kind, locked, busy, editing, breakdown, selected, assess }: { line: SheetLine; contract: Contract; kind: SheetSection["kind"]; locked: boolean; busy: boolean; editing: boolean; breakdown: Breakdown; selected: boolean; assess: Assess }) {
  const unplaced = kind === "unplaced";
  // A line the claim left out is named by its breakdown line or variation; its row is added by the first edit.
  const id = keyId(line.key);
  const row = line.row_id || id;
  const done = line.completed ? fmt(line.completed, true) : "";
  const save = (fields: Record<string, unknown>) => assess(row, fields, `Assessed ${line.label}.`);
  const enter = (e: React.KeyboardEvent<HTMLInputElement>) => { if (e.key === "Enter") e.currentTarget.blur(); };
  // What the contractor claims to date, where it is not what is assessed. A
  // line the claim left out claims nothing, so it has nothing to show.
  const differs = line.claimed_to_date != null && line.claimed_to_date !== line.completed && line.claimed_to_date !== line.prior;
  const own = kind === "works" || kind === "provisional";
  const sum = line.total_let ? fmt(line.total_let, true) : "";
  const change = (fields: Record<string, unknown>, label: string) => breakdown.edit(id, fields, label);
  return <>
    <tr data-row={line.row_id || undefined} data-key={line.key || undefined} data-retired={line.retired || undefined} aria-selected={selected || undefined}>
      <th scope="row" className="ps-desc">
        {editing ? <span className="ps-edit">
          <input className="field-plain" aria-label={`Description of ${line.label}`} key={`${line.key}:l:${line.label}`} defaultValue={line.label} maxLength={500} disabled={busy}
            onBlur={e => { const v = e.target.value.trim(); if (v && v !== line.label) change({ label: v }, `Renamed ${line.label}.`); }} onKeyDown={enter} />
          <input className="field-plain ps-edit-note" aria-label={`Note for ${line.label}`} key={`${line.key}:n:${line.note}`} defaultValue={line.note} maxLength={300} placeholder="note" disabled={busy}
            onBlur={e => { const v = e.target.value.trim(); if (v !== line.note) change({ note: v }, `Changed ${line.label}.`); }} onKeyDown={enter} />
          <select className="field-plain" aria-label={`Section of ${line.label}`} value={kind} disabled={busy} onChange={e => change({ section: e.target.value }, `Moved ${line.label}.`)}>
            <option value="works">Works</option><option value="provisional">Provisional sum</option></select>
          <button type="button" className="cell-link" disabled={busy} onClick={() => change({ remove: line.retired ? "no" : "yes" }, line.retired ? `Put ${line.label} back.` : `Removed ${line.label}.`)}>{line.retired ? "Put back" : "Remove"}</button>
        </span> : <>
          {line.label}{line.note && <span className="ps-desc-note"> ({line.note})</span>}
          {line.retired && <span className="cost-tag">removed</span>}
          {differs && <span className="cost-tag">claimed {cents(line.claimed_to_date)}</span>}
          {line.not_approved && <span className="cost-tag" data-alert="check">not approved</span>}
          {kind === "provisional" && !line.retired && <label className="ps-final"><input type="checkbox" checked={line.final} disabled={busy} aria-label={`${line.label} is final`}
            onChange={e => change({ final: e.target.checked ? "yes" : "no" }, e.target.checked ? `${line.label} is final.` : `${line.label} is no longer final.`)} /> Final</label>}
          {unplaced && !locked && <select className="field-plain ps-place" aria-label={`Line for ${line.label}`} value="" onChange={e => { if (e.target.value) assess(line.row_id, { line: e.target.value }, `Placed ${line.label}.`); }}>
            <option value="">Place on…</option>{(contract.breakdown ?? []).filter(b => !b.retired).map(b => <option key={b.id} value={b.id}>{b.label}</option>)}<option value="retention">Retention</option></select>}
        </>}
        {line.over_by && <span className="ps-flag">{kind === "provisional" ? `Over the provisional sum by $${cents(line.over_by)}; the excess needs a variation.` : `Over this line by $${cents(line.over_by)}.`}</span>}
        {line.final_claimed && <span className="ps-flag">Marked final before this claim, and this claim moves it. Clear the mark or cut the line.</span>}
      </th>
      <td className="num">{editing ? <input className="field-plain num" inputMode="decimal" aria-label={`Sum for ${line.label}`} key={`${line.key}:s:${line.total_let}`} defaultValue={sum} placeholder="unknown" disabled={busy}
        onBlur={e => { const v = plain(e.target.value); if (v !== plain(sum)) change({ amount: v }, `Changed ${line.label}.`); }} onKeyDown={enter} />
        : unplaced ? "" : line.not_approved ? "not approved" : line.total_let == null ? "unknown" : nilDash(line.total_let)}</td>
      <td className="num ps-pct">{locked || editing || unplaced || line.total_let == null || (own && line.retired) ? line.percent : <input className="field-plain num" inputMode="decimal" aria-label={`${line.label}: completed to date, percent`} key={`${line.key}:p:${line.percent}`} defaultValue={line.percent} placeholder=""
        onBlur={e => { const v = plain(e.target.value); if (v && v !== line.percent) save({ percent: v }); }} onKeyDown={enter} />}</td>
      <td className="num">{locked || editing ? (line.completed == null ? "" : nilDash(line.completed)) : <input className="field-plain num" inputMode="decimal" aria-label={`${line.label}: completed to date, dollars`} key={`${line.key}:d:${line.completed}`} defaultValue={done} placeholder={line.claimed_to_date ? `claimed ${cents(line.claimed_to_date)}` : ""}
        onBlur={e => { const v = plain(e.target.value); if (v !== plain(done)) save({ assessed_to_date: v }); }} onKeyDown={enter} />}</td>
      <td className="num">{nilDash(line.prior)}</td>
      <td className="num">{line.this_claim == null ? "" : nilDash(line.this_claim)}</td>
    </tr>
    {(line.below_claimed || line.reason) && <tr className="ps-reason"><td colSpan={6}><label><span>Reason</span>
      {locked ? <span>{line.reason}</span> : <input className="field-plain" aria-label={`Reason for ${line.label}`} key={`${line.key}:why:${line.reason}`} defaultValue={line.reason} maxLength={1000} placeholder="Why this is less than claimed. It prints on the schedule."
        onBlur={e => { if (e.target.value.trim() !== line.reason) save({ reason: e.target.value.trim() }); }} onKeyDown={enter} />}</label></td></tr>}
  </>;
}

const BASE: Record<string, string> = { works: "the works sub total", let: "works and provisional sums", to_date: "the contract sum to date" };

// The total and retention. With terms on the contract the PM says what this
// claim holds or returns, as a percent of the base or as dollars, and the
// server adds it to what was held before. With no terms retention is typed
// as dollars held to date, as it always was.
function Totals({ sheet, locked, held, onHeld, onMove, why, onWhy }: { sheet: Sheet; locked: boolean; held: string | null; onHeld: (v: string) => void; onMove: (fields: Record<string, string>) => void;
  why: { below: boolean; reason: string } | null; onWhy: (v: string) => void }) {
  const r = sheet.retention;
  const returning = r.held != null && r.held.startsWith("-");
  const none = r.held != null && !/[1-9]/.test(r.held);
  const moved = r.held == null || r.open ? "" : none ? "No retention change this claim" : `Retention ${returning ? "returned" : "held"} this claim${r.percent ? ` (${returning ? "+" : "−"}${r.percent}%)` : ""}`;
  const shown = held ? fmt(held, true) : "";
  const amount = r.held && !none ? fmt(r.held.replace("-", ""), true) : "";
  const [mode, setMode] = useState<"" | "hold" | "return" | "none">(r.open ? "" : none ? "none" : returning ? "return" : "hold");
  const say = (next: "hold" | "return", by: "percent" | "dollars", v: string) => { if (v) onMove({ [`retention_${next}${by === "percent" ? "_percent" : ""}`]: v }); };
  const enter = (e: React.KeyboardEvent<HTMLInputElement>) => { if (e.key === "Enter") e.currentTarget.blur(); };
  const withTerms = r.rate_percent !== "";
  return <div className="inv-table-scroll ps-section"><table className="inv-table var-table ps-table ps-totals">
    {COLS}
    <thead><tr><th scope="col" className="ps-band">Total and retention</th><th scope="col" className="num">Total let ($)</th><th scope="col" className="num">{withTerms ? "(%)" : ""}</th><th scope="col" className="num">Completed ($)</th><th scope="col" className="num">Less prior ($)</th><th scope="col" className="num">This claim ($)</th></tr></thead>
    <tbody>
      <tr className="ps-subtotal"><th scope="row">Total (ex GST)</th><td className="num">{sheet.total_let == null ? "unknown" : nilDash(sheet.total_let)}</td><td /><td className="num">{sheet.completed == null ? "" : nilDash(sheet.completed)}</td><td className="num">{nilDash(sheet.prior)}</td><td className="num">{sheet.this_claim == null ? "" : nilDash(sheet.this_claim)}</td></tr>
      {withTerms && <tr><th scope="row" className="ps-desc">Retention limit: {r.rate_percent}% of {BASE[r.base_kind] ?? "the base"}<span className="ps-how">Contract terms</span></th><td className="num">{r.limit == null ? "unknown" : `−${cents(r.limit)}`}</td><td /><td /><td /><td /></tr>}
      <tr><th scope="row">Retention held before this claim</th><td /><td /><td /><td className="num">{nilDash(r.held_before)}</td><td /></tr>
      {withTerms ? <tr data-open={r.open || undefined}>
        <th scope="row" className="ps-desc">{locked ? moved || "Retention this claim" : "Retention this claim"}
          {!locked && <select className="field-plain ps-ret-mode" aria-label="Retention this claim" value={mode} onChange={e => {
            const next = e.target.value as "hold" | "return" | "none";
            setMode(next);
            if (next === "none") onMove({ retention_hold: "0" });
            else if (amount) say(next, "dollars", plain(amount));
          }}>{mode === "" && <option value="">Hold or return…</option>}<option value="hold">Hold</option><option value="return">Return</option><option value="none">No change</option></select>}
          {r.open && <span className="ps-flag">Say what this claim holds or returns before Issue.{!locked && <> <button type="button" className="cell-link" onClick={() => { setMode("none"); onMove({ retention_hold: "0" }); }}>No change this claim</button></>}</span>}
        </th>
        <td />
        <td className="num">{locked || mode === "none" || mode === "" ? (none ? "" : r.percent) : <input className="field-plain num" inputMode="decimal" aria-label="Retention this claim, percent" key={`rp:${r.percent}:${mode}`} defaultValue={none ? "" : r.percent} placeholder="%"
          onBlur={e => { const v = plain(e.target.value); if (v && v !== r.percent) say(mode as "hold" | "return", "percent", v); }} onKeyDown={enter} />}</td>
        <td /><td />
        <td className="num">{locked || mode === "none" || mode === "" ? (r.held == null ? "" : none ? "–" : returning ? `+${amount}` : `−${amount}`) : <input className="field-plain num" inputMode="decimal" aria-label="Retention this claim, dollars" key={`rd:${r.held}:${mode}`} defaultValue={amount} placeholder="0.00"
          onBlur={e => { const v = plain(e.target.value); if (v && v !== plain(amount)) say(mode as "hold" | "return", "dollars", v); }} onKeyDown={enter} />}</td>
      </tr> : <>
        <tr><th scope="row">Retention held to date</th><td /><td /><td className="num">{locked ? (r.held_to_date == null ? "" : nilDash(r.held_to_date)) : <input className="field-plain num" inputMode="decimal" aria-label="Retention held to date, dollars" key={`ret:${held}`} defaultValue={shown} placeholder="0.00"
          onBlur={e => { const v = plain(e.target.value); if (v !== plain(shown)) onHeld(v); }} onKeyDown={enter} />}</td><td /><td /></tr>
        <tr><th scope="row">{moved || "Retention this claim"}</th><td /><td /><td /><td /><td className="num">{r.held == null ? "" : none ? "–" : returning ? `+${amount}` : `−${amount}`}</td></tr>
      </>}
      {why && (why.below || why.reason) && <tr className="ps-reason"><td colSpan={6}><label><span>Reason</span>
        {locked ? <span>{why.reason}</span> : <input className="field-plain" aria-label="Reason for retention" key={`ret:why:${why.reason}`} defaultValue={why.reason} maxLength={1000} placeholder="Why more is held than claimed. It prints on the schedule."
          onBlur={e => { if (e.target.value.trim() !== why.reason) onWhy(e.target.value.trim()); }} onKeyDown={enter} />}</label></td></tr>}
      <tr className="ps-subtotal"><th scope="row" colSpan={5}>Total claim inc retention (ex GST)</th><td className="num">{sheet.net == null ? "" : nilDash(sheet.net)}</td></tr>
    </tbody>
  </table></div>;
}

function Payments({ sheet }: { sheet: Sheet }) {
  return <div className="inv-table-scroll ps-section"><table className="inv-table var-table ps-table ps-payments">
    <thead><tr><th scope="col" className="ps-band">Payment list</th><th scope="col">No.</th><th scope="col">Inv no.</th><th scope="col">Invoice date</th><th scope="col" className="num">Value ($)</th><th scope="col" className="num">GST ($)</th><th scope="col" className="num">Value inc GST ($)</th></tr></thead>
    <tbody>
      {sheet.payments.map(p => <tr key={p.claim_id} className={p.current ? "ps-current" : undefined}>
        <td>{p.period}{p.unserved && <span className="cost-tag" data-alert="overdue">no schedule served</span>}</td><td>{p.number}</td>
        <td>{p.invoice_no || <span className="ps-src">awaiting invoice</span>}</td><td>{day(p.invoice_date)}</td>
        <td className="num">{p.ex == null ? "" : nilDash(p.ex)}</td><td className="num">{p.gst == null ? "" : nilDash(p.gst)}</td><td className="num">{p.inc == null ? "" : nilDash(p.inc)}</td>
      </tr>)}
      <tr className="ps-subtotal"><th scope="row" colSpan={4}>Total</th><td className="num">{sheet.paid.ex == null ? "" : nilDash(sheet.paid.ex)}</td><td className="num">{sheet.paid.gst == null ? "" : nilDash(sheet.paid.gst)}</td><td className="num">{sheet.paid.inc == null ? "" : nilDash(sheet.paid.inc)}</td></tr>
    </tbody>
  </table></div>;
}
