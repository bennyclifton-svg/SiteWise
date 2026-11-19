import { useEffect, useRef, useState, type DragEvent } from "react";
import { PageSidePanel } from "./PageActions";
import { ApiError } from "./api";
import { fmt } from "./CostGrid";
import { clickRow, emptySelection, prune, selectAll } from "./table/selection";
import { revealQuietly, showingLine, useChatTable } from "./chatView";
import { invoiceApi, uploadInvoice, type Claim, type ClaimRow, type Contract, type DueDate, type Ledger, type OpCall, type Variation, type VariationStatus } from "./invoiceApi";

// The Cost Plan's "Variations and claims" tab
// (docs/plans/2026-11-17-variations-progress-claims.md, Task 10). Every
// figure is the server's: the browser never sums money. Every write goes
// through POST /ops, so each has one Undo; approving waits for nothing here
// because pressing Approve is the PM's own click.

const STATUS: Record<VariationStatus, string> = { proposed: "Proposed", submitted: "Submitted", approved: "Approved", rejected: "Rejected", withdrawn: "Withdrawn" };
const money = (m: string | null | undefined) => fmt(m, false);
const plain = (s: string) => s.replace(/[,$\s]/g, "");

export function VariationsTab({ projectId, ledger, reload, onSignedOut, openClaim, onOpenClaim }: { projectId: string; ledger: Ledger | null; reload: () => void; onSignedOut: () => void; openClaim: string; onOpenClaim: (id: string) => void }) {
  const contracts = ledger?.contracts ?? [];
  const [chosen, setChosen] = useState("");
  const claimed = ledger?.claims?.find(c => c.invoice_id === openClaim);
  const contract = contracts.find(c => c.package_id === (claimed?.package_id || chosen)) ?? contracts[0];
  const [busy, setBusy] = useState(false), [error, setError] = useState("");
  const [last, setLast] = useState<{ change: string; label: string } | null>(null);
  useEffect(() => { if (chosen && !contracts.some(c => c.package_id === chosen)) setChosen(""); }, [contracts, chosen]);

  async function apply(area: "variations" | "claims", ops: OpCall[], label: string) {
    if (busy) return false;
    setBusy(true); setError("");
    try {
      const out = await invoiceApi.ops(projectId, area, ops);
      setLast(out.change_id ? { change: out.change_id, label } : null);
      reload();
      return true;
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) { onSignedOut(); return false; }
      setError(e instanceof ApiError && e.message ? e.message : "Could not save. Try again.");
      return false;
    } finally { setBusy(false); }
  }

  if (!ledger) return <p role="status">Loading…</p>;
  if (!contract) return <section className="inv-empty" aria-label="Variations and claims"><h2>No live contracts yet</h2><p>Variations and progress claims sit on an awarded contract. Award a package in Procurement and it appears here.</p></section>;
  return <section className="variations" aria-label="Variations and claims" aria-busy={busy}>
    {contracts.length > 1 && <div className="costs-toolbar"><label className="var-contract">Contract <select value={contract.package_id} onChange={e => setChosen(e.target.value)}>
      {contracts.map(c => <option key={c.package_id} value={c.package_id}>{c.title}{c.firm ? ` · ${c.firm}` : ""}</option>)}</select></label></div>}
    <ContractHeader contract={contract} projectId={projectId} onSaved={reload} onError={setError} />
    {error && <p role="alert" className="inv-error">{error}</p>}
    {last && <p className="reg-notice" role="status">{last.label} <button type="button" className="cell-link" disabled={busy} onClick={() => { const c = last.change; setLast(null); void invoiceApi.undo(projectId, c).then(reload).catch(() => setError("Can't undo: this has changed since.")); }}>Undo</button></p>}
    <VariationTable contract={contract} variations={(ledger.variations ?? []).filter(v => v.package_id === contract.package_id)} busy={busy} apply={apply} />
    {contract.claims && <ClaimsTable projectId={projectId} contract={contract} claims={(ledger.claims ?? []).filter(c => c.package_id === contract.package_id)} open={openClaim} onOpen={onOpenClaim} onUploaded={reload} onSignedOut={onSignedOut} />}
    {claimed && <ClaimPanel projectId={projectId} claim={claimed} contract={contract} busy={busy} apply={apply} onClose={() => onOpenClaim("")} reload={reload} onSignedOut={onSignedOut} />}
  </section>;
}

function ContractHeader({ contract, projectId, onSaved, onError }: { contract: Contract; projectId: string; onSaved: () => void; onError: (m: string) => void }) {
  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState({ schedule_days: "", schedule_basis: "business", payment_days: "", payment_basis: "business" });
  function edit() {
    setForm({ schedule_days: contract.schedule_days == null ? "" : String(contract.schedule_days), schedule_basis: contract.schedule_basis, payment_days: contract.payment_days == null ? "" : String(contract.payment_days), payment_basis: contract.payment_basis });
    setEditing(true);
  }
  async function save() {
    const days = (v: string) => v.trim() === "" ? null : Number(v);
    try { await invoiceApi.periods(projectId, contract.package_id, { schedule_days: days(form.schedule_days), schedule_basis: form.schedule_basis, payment_days: days(form.payment_days), payment_basis: form.payment_basis }); setEditing(false); onSaved(); }
    catch (e) { onError(e instanceof ApiError && e.message ? e.message : "Could not save the contract's periods."); }
  }
  const period = (days: number | null, basis: string) => days == null ? "not entered" : `${days} ${basis} days`;
  return <><dl className="var-head" aria-label={`${contract.title} figures`}>
    <div><dt>Contract</dt><dd>{contract.title}{contract.firm ? ` · ${contract.firm}` : ""}</dd></div>
    <div><dt>Contract sum</dt><dd className="num">{money(contract.contract_sum)}</dd></div>
    <div><dt>Approved variations</dt><dd className="num">{money(contract.approved_variations)}</dd></div>
    <div><dt>Contract sum to date</dt><dd className="num">{money(contract.contract_sum_to_date)}</dd></div>
    {contract.claims && <><div><dt>Certified to date</dt><dd className="num">{money(contract.certified_to_date)}</dd></div>
      <div><dt>Retention held</dt><dd className="num">{money(contract.retention_held)}</dd></div>
      <div><dt>Contract periods</dt><dd>Schedule {period(contract.schedule_days, contract.schedule_basis)}; payment {period(contract.payment_days, contract.payment_basis)} <button type="button" className="cell-link" onClick={edit}>Edit</button></dd></div></>}
  </dl>
  {editing && <form className="var-periods" onSubmit={e => { e.preventDefault(); void save(); }} aria-label="Contract periods">
    <label>Payment schedule within <input inputMode="numeric" aria-label="Payment schedule days" value={form.schedule_days} onChange={e => setForm({ ...form, schedule_days: e.target.value })} /></label>
    <select aria-label="Payment schedule day basis" value={form.schedule_basis} onChange={e => setForm({ ...form, schedule_basis: e.target.value })}><option value="business">business days</option><option value="calendar">calendar days</option></select>
    <label>Payment within <input inputMode="numeric" aria-label="Payment days" value={form.payment_days} onChange={e => setForm({ ...form, payment_days: e.target.value })} /></label>
    <select aria-label="Payment day basis" value={form.payment_basis} onChange={e => setForm({ ...form, payment_basis: e.target.value })}><option value="business">business days</option><option value="calendar">calendar days</option></select>
    <button className="btn btn-small">Save periods</button><button type="button" className="btn btn-small" onClick={() => setEditing(false)}>Cancel</button>
  </form>}</>;
}

type Apply = (area: "variations" | "claims", ops: OpCall[], label: string) => Promise<boolean>;

function VariationTable({ contract, variations, busy, apply }: { contract: Contract; variations: Variation[]; busy: boolean; apply: Apply }) {
  const [selection, setSelection] = useState(emptySelection);
  const [only, setOnly] = useState<ReadonlySet<string> | null>(null), [onlyShown, setOnlyShown] = useState("");
  const [draft, setDraft] = useState({ title: "", claimed: "", line: "" });
  const ids = variations.map(v => v.id);
  useEffect(() => setSelection(s => prune(s, ids)), [ids.join(",")]); // eslint-disable-line react-hooks/exhaustive-deps
  const shown = only ? variations.filter(v => only.has(v.id)) : variations;
  useChatTable("variations", "costs", {
    select: list => { const keep = list.filter(id => ids.includes(id)); setOnly(null); setSelection(selectAll(keep)); if (keep[0]) requestAnimationFrame(() => revealQuietly(document.querySelector(`tr[data-variation="${CSS.escape(keep[0])}"]`))); },
    only: (list, words) => { setOnly(new Set(list.filter(id => ids.includes(id)))); setOnlyShown(words ?? ""); },
    clear: () => setOnly(null),
  }, selection.ids);
  const edit = (v: Variation, fields: Record<string, unknown>, label: string) => apply("variations", [{ op: "variations.edit", fields: { variation: v.id, ...fields } }], label);
  function decide(v: Variation, status: string) {
    if (status === v.status) return;
    if (status === "approved") {
      const amount = window.prompt(`Approve ${v.number} at what amount, ex GST?`, v.estimate ?? v.claimed ?? "");
      if (amount == null || !plain(amount)) return;
      void apply("variations", [{ op: "variations.decide", fields: { variation: v.id, decision: "approve", approved: plain(amount) } }], `Approved ${v.number}.`);
      return;
    }
    if (status === "rejected") { void apply("variations", [{ op: "variations.decide", fields: { variation: v.id, decision: "reject" } }], `Rejected ${v.number}.`); return; }
    void edit(v, { status }, `${v.number} is ${STATUS[status as VariationStatus].toLowerCase()}.`);
  }
  async function add() {
    if (!draft.title.trim()) return;
    const fields: Record<string, unknown> = { package: contract.package_id, title: draft.title.trim() };
    if (plain(draft.claimed)) fields.claimed = plain(draft.claimed);
    if (draft.line) fields.line = draft.line;
    if (await apply("variations", [{ op: "variations.add", fields }], `Recorded "${draft.title.trim()}".`)) setDraft({ title: "", claimed: "", line: "" });
  }
  const cellInput = (v: Variation, field: "claimed" | "estimate" | "time_claimed_days", value: string, label: string) =>
    <input className="field-plain num" aria-label={`${label} for ${v.number}`} inputMode="decimal" key={`${v.id}:${field}:${value}`} defaultValue={field === "time_claimed_days" ? value : value ? money(value) : ""} disabled={busy || v.status === "approved" && field !== "time_claimed_days"}
      onBlur={e => { const next = plain(e.target.value); const was = field === "time_claimed_days" ? value : value ? plain(money(value)) : ""; if (next !== was) void edit(v, { [field]: next }, `Changed ${v.number}.`); }}
      onKeyDown={e => { if (e.key === "Enter") (e.target as HTMLInputElement).blur(); }} />;
  return <>
    <h2 className="var-title">Variations</h2>
    {only && <p className="reg-filter">{showingLine("variations", only.size, onlyShown)} · <button type="button" className="cell-link" onClick={() => setOnly(null)}>Clear</button></p>}
    <div className="inv-table-scroll"><table className="inv-table var-table" onKeyDown={e => { if (e.key === "Escape") { setSelection(emptySelection); setOnly(null); } }}>
      <caption className="sr-only">Variations on {contract.title}</caption>
      <thead><tr><th scope="col">No.</th><th scope="col">Description</th>{contract.lines.length > 1 && <th scope="col">Line</th>}<th scope="col">Status</th><th scope="col" className="num">Claimed</th><th scope="col" className="num">PM's estimate</th><th scope="col" className="num">Approved</th><th scope="col" className="num">Days</th><th scope="col">Documents</th></tr></thead>
      <tbody>
        {shown.map(v => <tr key={v.id} data-variation={v.id} aria-selected={selection.ids.includes(v.id)} onClick={e => { if (!(e.target as HTMLElement).closest("input,select,button")) setSelection(s => clickRow(s, v.id, ids, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })); }}>
          <td>{v.number}</td>
          <td><input className="field-plain" aria-label={`Description of ${v.number}`} key={`${v.id}:t:${v.title}`} defaultValue={v.title} disabled={busy} maxLength={200} onBlur={e => { const t = e.target.value.trim(); if (t && t !== v.title) void edit(v, { title: t }, `Changed ${v.number}.`); }} onKeyDown={e => { if (e.key === "Enter") (e.target as HTMLInputElement).blur(); }} /></td>
          {contract.lines.length > 1 && <td><select className="field-plain" aria-label={`Line for ${v.number}`} value={v.line_id} disabled={busy} onChange={e => void edit(v, { line: e.target.value }, `Moved ${v.number}.`)}>{contract.lines.map(l => <option key={l.id} value={l.id}>{l.label}</option>)}</select></td>}
          <td><select className="field-plain" aria-label={`Status of ${v.number}`} value={v.status} disabled={busy} onChange={e => decide(v, e.target.value)}>{(Object.keys(STATUS) as VariationStatus[]).map(s => <option key={s} value={s}>{STATUS[s]}</option>)}</select></td>
          <td className="num">{cellInput(v, "claimed", v.claimed ?? "", "Claimed")}</td>
          <td className="num">{cellInput(v, "estimate", v.estimate ?? "", "PM's estimate")}{v.forecast_basis === "claimed" && <span className="cost-tag" title="No estimate yet: the forecast uses the claimed amount">claimed</span>}</td>
          <td className="num">{money(v.approved)}</td>
          <td className="num">{cellInput(v, "time_claimed_days", v.time_claimed_days == null ? "" : String(v.time_claimed_days), "Days claimed")}</td>
          <td>{v.documents.length ? v.documents.map(d => d.name || "Document").join(", ") : "—"}</td>
        </tr>)}
        <tr className="cost-ghost">
          <td>New</td>
          <td><input aria-label="New variation description" placeholder="Describe a variation" value={draft.title} disabled={busy} onChange={e => setDraft({ ...draft, title: e.target.value })} onKeyDown={e => { if (e.key === "Enter") void add(); }} /></td>
          {contract.lines.length > 1 && <td><select aria-label="New variation line" value={draft.line} onChange={e => setDraft({ ...draft, line: e.target.value })}><option value="">line</option>{contract.lines.map(l => <option key={l.id} value={l.id}>{l.label}</option>)}</select></td>}
          <td>Submitted</td>
          <td className="num"><input aria-label="New variation claimed amount" inputMode="decimal" placeholder="claimed" value={draft.claimed} disabled={busy} onChange={e => setDraft({ ...draft, claimed: e.target.value })} onKeyDown={e => { if (e.key === "Enter") void add(); }} /></td>
          <td /><td /><td /><td />
        </tr>
      </tbody>
    </table></div>
  </>;
}

const day = (iso: string) => iso ? new Date(`${iso}T00:00:00`).toLocaleDateString("en-AU", { day: "numeric", month: "short", year: "numeric" }) : "—";
const ALERT: Record<string, string> = { overdue: "Past due", soon: "Due soon", check: "To check" };

/** A worked-out date: the date, "to check" when it is not settled, and how it was worked out. */
function Due({ due, label }: { due: DueDate; label: string }) {
  return <div className="var-due"><dt>{label}</dt><dd>{due.date ? day(due.date) : "Unknown"}{due.to_check && <span className="cost-tag">to check</span>}
    <p>{due.words}{due.section && !due.words.includes(due.section) ? ` (${due.section})` : ""}</p>{due.notes.map(n => <p key={n}>{n}</p>)}</dd></div>;
}

function ClaimsTable({ projectId, contract, claims, open, onOpen, onUploaded, onSignedOut }: { projectId: string; contract: Contract; claims: Claim[]; open: string; onOpen: (id: string) => void; onUploaded: () => void; onSignedOut: () => void }) {
  const [over, setOver] = useState(false), [note, setNote] = useState("");
  const [selection, setSelection] = useState(emptySelection);
  const [only, setOnly] = useState<ReadonlySet<string> | null>(null), [onlyShown, setOnlyShown] = useState("");
  const ids = claims.map(c => c.invoice_id);
  const input = useRef<HTMLInputElement>(null);
  useChatTable("claims", "costs", {
    select: list => { setOnly(null); setSelection(selectAll(list.filter(id => ids.includes(id)))); },
    only: (list, words) => { setOnly(new Set(list.filter(id => ids.includes(id)))); setOnlyShown(words ?? ""); },
    clear: () => setOnly(null),
    open: id => { if (ids.includes(id)) onOpen(id); },
  }, selection.ids);
  // A file dropped here is this contract's progress claim, read in about a second.
  function drop(files: File[]) {
    for (const f of files) {
      setNote(`Reading ${f.name}…`);
      uploadInvoice(projectId, f, () => {}, contract.package_id).done.then(r => { setNote(r.created ? `${f.name} is being read.` : `${f.name} is already filed.`); onUploaded(); })
        .catch(e => { if (e instanceof ApiError && e.status === 401) onSignedOut(); else setNote(`${f.name}: ${e instanceof Error ? e.message : "upload failed"}`); });
    }
  }
  const drag = {
    onDragOver: (e: DragEvent) => { if (!e.dataTransfer.types.includes("Files")) return; e.preventDefault(); e.stopPropagation(); setOver(true); },
    onDragLeave: (e: DragEvent) => { e.stopPropagation(); setOver(false); },
    onDrop: (e: DragEvent) => { if (!e.dataTransfer.types.includes("Files")) return; e.preventDefault(); e.stopPropagation(); setOver(false); drop(Array.from(e.dataTransfer.files)); },
  };
  const shown = only ? claims.filter(c => only.has(c.invoice_id)) : claims;
  return <section aria-label="Progress claims" className="var-claims" data-over={over || undefined} {...drag}>
    <h2 className="var-title">Progress claims <button type="button" className="cell-link" onClick={() => input.current?.click()}>Add a claim</button></h2>
    <input ref={input} type="file" hidden data-testid="claim-input" accept=".pdf,.docx,.xlsx,.md,.markdown,application/pdf" onChange={e => { if (e.target.files?.length) drop(Array.from(e.target.files)); e.target.value = ""; }} />
    {note && <p className="reg-notice" role="status">{note}</p>}
    {only && <p className="reg-filter">{showingLine("claims", only.size, onlyShown)} · <button type="button" className="cell-link" onClick={() => setOnly(null)}>Clear</button></p>}
    <div className="inv-table-scroll"><table className="inv-table var-table">
      <caption className="sr-only">Progress claims on {contract.title}</caption>
      <thead><tr><th scope="col">No.</th><th scope="col">Served</th><th scope="col">Reference date</th><th scope="col" className="num">Claimed</th><th scope="col" className="num">Scheduled</th><th scope="col">Schedule due</th><th scope="col">Payment due</th><th scope="col">Status</th></tr></thead>
      <tbody>{shown.length === 0 ? <tr><td colSpan={8}>No progress claims yet. Drop the builder's claim here.</td></tr> : shown.map(c =>
        <tr key={c.invoice_id} data-alert={c.alert || undefined} aria-selected={open === c.invoice_id || selection.ids.includes(c.invoice_id)}>
          <td><button type="button" className="inv-open" onClick={() => onOpen(c.invoice_id)}>{c.number || "Claim"}</button></td>
          <td>{day(c.served_on)}{!c.served_confirmed && <span className="cost-tag">confirm</span>}</td>
          <td>{day(c.reference_date)}</td>
          <td className="num">{money(c.claimed)}</td>
          <td className="num">{money(c.scheduled)}</td>
          <td>{c.schedule_due.date ? day(c.schedule_due.date) : "Unknown"}{c.schedule_due.to_check && <span className="cost-tag">to check</span>}</td>
          <td>{c.payment_due.date ? day(c.payment_due.date) : "Unknown"}{c.payment_due.to_check && <span className="cost-tag">to check</span>}</td>
          <td>{c.issued.length ? (c.schedule_served_on ? "Schedule served" : "Issued") : c.unassessed ? "Assessing" : "Assessed"}{c.alert && <span className="cost-tag" data-alert={c.alert}>{ALERT[c.alert]}</span>}{c.late && " · claimed amount counts"}</td>
        </tr>)}</tbody>
    </table></div>
  </section>;
}

function ClaimPanel({ projectId, claim, contract, busy, apply, onClose, reload, onSignedOut }: { projectId: string; claim: Claim; contract: Contract; busy: boolean; apply: Apply; onClose: () => void; reload: () => void; onSignedOut: () => void }) {
  const [error, setError] = useState(""), [issuing, setIssuing] = useState(false);
  const [selection, setSelection] = useState(emptySelection);
  const [only, setOnly] = useState<ReadonlySet<string> | null>(null), [onlyShown, setOnlyShown] = useState("");
  const ids = claim.rows.map(r => r.id);
  const issued = claim.issued.length > 0;
  useEffect(() => { const k = (e: KeyboardEvent) => { if (e.key === "Escape" && !(e.target instanceof HTMLInputElement)) onClose(); }; window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k); }, [onClose]);
  useChatTable("claim_rows", "costs", {
    select: list => { setOnly(null); setSelection(selectAll(list.filter(id => ids.includes(id)))); },
    only: (list, words) => { setOnly(new Set(list.filter(id => ids.includes(id)))); setOnlyShown(words ?? ""); },
    clear: () => setOnly(null),
  }, selection.ids);
  const assess = (r: ClaimRow, fields: Record<string, unknown>) => apply("claims", [{ op: "claims.assess", fields: { claim: claim.invoice_id, row: r.id, ...fields } }], `Assessed ${r.label}.`);
  async function issue() {
    setIssuing(true); setError("");
    try { await invoiceApi.issueSchedule(projectId, claim.invoice_id); reload(); }
    catch (e) { if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; } setError(e instanceof ApiError && e.message ? e.message : "Could not issue the schedule."); }
    finally { setIssuing(false); }
  }
  async function served(field: string, value: string) {
    try { await invoiceApi.edit(projectId, claim.invoice_id, { [field]: value }); reload(); }
    catch (e) { setError(e instanceof ApiError && e.message ? e.message : "Could not save the date."); }
  }
  const lineName = (r: ClaimRow) => r.kind === "retention" ? "Retention" : r.variation || r.line || "Not placed";
  const rows = only ? claim.rows.filter(r => only.has(r.id)) : claim.rows;
  return <PageSidePanel><aside className="inv-panel var-panel" aria-label={`Claim ${claim.number}`}>
    <div className="inv-panel-head"><h2>Claim {claim.number || ""} · {contract.title}</h2><button type="button" className="btn btn-small" onClick={onClose}>Close</button></div>
    <div className="var-panel-body">
      {error && <p role="alert" className="inv-error">{error}</p>}
      {claim.flags.length > 0 && <ul className="inv-flags" aria-label="Checks">{claim.flags.map((f, i) => <li key={`${f.code}:${i}`}>{f.reason}</li>)}</ul>}
      <dl className="var-head">
        <div><dt>Served</dt><dd><input type="date" aria-label="Claim served on" key={`s:${claim.served_on}`} defaultValue={claim.served_on} onBlur={e => { if (e.target.value && (e.target.value !== claim.served_on || !claim.served_confirmed)) void served("served_on", e.target.value); }} />{!claim.served_confirmed && <span className="cost-tag">confirm</span>}</dd></div>
        <div><dt>Reference date</dt><dd><input type="date" aria-label="Reference date" key={`r:${claim.reference_date}`} defaultValue={claim.reference_date} onBlur={e => { if (e.target.value !== claim.reference_date) void served("reference_date", e.target.value); }} /></dd></div>
        <Due due={claim.schedule_due} label="Payment schedule due" />
        <Due due={claim.payment_due} label="Payment due" />
      </dl>
      {only && <p className="reg-filter">{showingLine("claim_rows", only.size, onlyShown)} · <button type="button" className="cell-link" onClick={() => setOnly(null)}>Clear</button></p>}
      <div className="inv-table-scroll"><table className="inv-table var-table">
        <caption className="sr-only">Rows of claim {claim.number}</caption>
        <thead><tr><th scope="col">Row</th><th scope="col">Line</th><th scope="col" className="num">Claimed to date</th><th scope="col" className="num">Certified before</th><th scope="col" className="num">This claim</th><th scope="col" className="num">Assessed to date</th><th scope="col" className="num">This schedule</th><th scope="col">Reason</th></tr></thead>
        <tbody>{rows.map(r => <tr key={r.id} data-row={r.id} aria-selected={selection.ids.includes(r.id)} onClick={e => { if (!(e.target as HTMLElement).closest("input,select,button")) setSelection(s => clickRow(s, r.id, ids, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })); }}>
          <td>{r.label}</td>
          <td>{issued ? lineName(r) : <select className="field-plain" aria-label={`Line for ${r.label}`} value={r.kind === "retention" ? "retention" : r.line_id} disabled={busy} onChange={e => void assess(r, { line: e.target.value })}>
            <option value="">Not placed</option>{contract.lines.map(l => <option key={l.id} value={l.id}>{l.label}</option>)}<option value="retention">Retention</option></select>}
            {r.mapped_by === "jev" && r.band === "blank" && <span className="cost-tag" title="Jev's pick is withheld until the claim answer key is calibrated">not placed</span>}</td>
          <td className="num">{money(r.to_date)}</td>
          <td className="num">{money(r.certified_before)}</td>
          <td className="num">{money(r.claimed)}</td>
          <td className="num">{issued ? money(r.assessed_to_date) : <input className="field-plain num" inputMode="decimal" aria-label={`Assessed to date for ${r.label}`} key={`${r.id}:${r.assessed_to_date}`} defaultValue={r.assessed_to_date ? money(r.assessed_to_date) : ""} disabled={busy}
            onBlur={e => { const v = plain(e.target.value); if (v !== (r.assessed_to_date ? plain(money(r.assessed_to_date)) : "")) void assess(r, { assessed_to_date: v }); }} onKeyDown={e => { if (e.key === "Enter") (e.target as HTMLInputElement).blur(); }} />}</td>
          <td className="num">{money(r.scheduled)}</td>
          <td>{issued ? r.reason : <input className="field-plain" aria-label={`Reason for ${r.label}`} key={`${r.id}:why:${r.reason}`} defaultValue={r.reason} maxLength={1000} disabled={busy} placeholder={r.below_claimed ? "Reason needed" : ""}
            onBlur={e => { if (e.target.value.trim() !== r.reason) void assess(r, { reason: e.target.value.trim() }); }} onKeyDown={e => { if (e.key === "Enter") (e.target as HTMLInputElement).blur(); }} />}</td>
        </tr>)}</tbody>
      </table></div>
      <dl className="var-head" aria-label="Schedule totals">
        <div><dt>Scheduled (ex GST)</dt><dd className="num">{money(claim.scheduled)}</dd></div>
        <div><dt>Retention withheld</dt><dd className="num">{money(claim.retention)}</dd></div>
        <div><dt>Net</dt><dd className="num">{money(claim.net)}</dd></div>
        <div><dt>GST</dt><dd className="num">{money(claim.gst)}</dd></div>
        <div><dt>Payable</dt><dd className="num">{money(claim.payable)}</dd></div>
      </dl>
      <div className="inv-actions">
        {!issued && <button type="button" className="btn" disabled={busy} onClick={() => void apply("claims", [{ op: "claims.accept_claimed", fields: { claim: claim.invoice_id } }], `Accepted claim ${claim.number} as claimed.`)}>Accept claimed</button>}
        <button type="button" className="btn" disabled={busy || issuing || claim.unassessed > 0 || claim.missing_reasons > 0} title={claim.unassessed ? `${claim.unassessed} rows not assessed` : claim.missing_reasons ? `${claim.missing_reasons} rows need a reason` : undefined} onClick={() => void issue()}>{issued ? "Issue again" : "Issue payment schedule"}</button>
      </div>
      {issued && <><ul className="var-issued">{claim.issued.map(i => <li key={i.version_id}>Schedule {i.number}, issued {day(i.issued_on)} · <a href={i.file}>Download PDF</a></li>)}</ul>
        <p className="inv-note">Send it from your email with the PDF attached (SiteWise's Send arrives with correspondence), then enter the day it was served.</p>
        <label className="var-served">Schedule served on <input type="date" aria-label="Schedule served on" key={`ss:${claim.schedule_served_on}`} defaultValue={claim.schedule_served_on} onBlur={e => { if (e.target.value !== claim.schedule_served_on) void served("schedule_served_on", e.target.value); }} /></label></>}
    </div>
  </aside></PageSidePanel>;
}
