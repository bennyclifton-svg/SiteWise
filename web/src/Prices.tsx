// The price table: schedule items down the side, firms across. With one firm
// it is the review screen; with several it is the comparison. Every total
// arrives computed by the server; a cell a user corrects is final.
import { useCallback, useEffect, useMemo, useState, type KeyboardEvent } from "react";
import { ApiError } from "./api";
import { AwardDialog } from "./AwardDialog";
import { fmt } from "./CostGrid";
import { Chip } from "./Invoices";
import { ProcurementReport } from "./ProcurementReport";
import { FLAG_WORDS, STATUS_WORDS, submissionApi, type Cell, type CellState, type Column, type FirmFields, type PriceTable, type ScheduleItem } from "./submissionApi";
import "./prices.css";

const money = (m: string | null | undefined) => (m == null ? "—" : `$${fmt(m)}`);
const day = (iso: string) => (iso ? new Date(`${iso}T00:00:00`).toLocaleDateString("en-AU", { day: "numeric", month: "short", year: "numeric" }) : "—");

const CELL_WORDS: Record<string, string> = { not_priced: "Not priced", included: "Included", excluded: "Excluded", "": "Not read" };

interface Props {
  projectId: string;
  pkg: string;
  reportId?: string;
  tick: number;
  onBack: () => void;
  onSignedOut: () => void;
  onOpenCostPlan?: () => void;
}

export function Prices({ projectId, pkg, reportId, tick, onBack, onSignedOut, onOpenCostPlan }: Props) {
  const [table, setTable] = useState<PriceTable | null>(null);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<{ sub: string; key: string } | null>(null);
  const [panel, setPanel] = useState("");
  const [awardFor, setAwardFor] = useState("");
  const [busy, setBusy] = useState(false);
  const [adding, setAdding] = useState(false);
  const [showReport, setShowReport] = useState(false);

  const fail = useCallback((e: unknown) => {
    if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
    setError(e instanceof ApiError && e.status !== 500 && e.message ? e.message : "Could not save. Try again.");
  }, [onSignedOut]);

  useEffect(() => {
    let stopped = false;
    submissionApi.table(projectId, pkg).then(t => { if (!stopped) setTable(t); }).catch(e => { if (!stopped) fail(e); });
    return () => { stopped = true; };
  }, [projectId, pkg, tick, fail]);

  async function run(action: () => Promise<PriceTable>) {
    setBusy(true); setError("");
    try { setTable(await action()); } catch (e) { fail(e); } finally { setBusy(false); }
  }

  const items = table?.items ?? [];
  const base = useMemo(() => items.filter(i => (i.kind === "stage" || i.kind === "element" || i.kind === "reimbursable") && i.base), [items]);
  const cols = table?.columns ?? [];
  // A row no firm priced is one line, not a column of "Not priced".
  const priced = (i: ScheduleItem) => cols.some(c => { const cell = c.cells[i.key]; return !cell || cell.state !== "not_priced"; });
  const allOptions = useMemo(() => items.filter(i => i.kind === "option"), [items]);
  const allRates = useMemo(() => items.filter(i => i.kind === "rate"), [items]);
  const options = allOptions.filter(priced);
  const rates = allRates.filter(priced);
  const unpriced = [...allOptions, ...allRates].filter(i => !priced(i));
  const compare = cols.length > 1;
  const selected = cols.find(c => c.submission.id === panel);

  if (showReport && table) return <ProcurementReport projectId={projectId} table={table} onBack={() => setShowReport(false)} onSignedOut={onSignedOut} />;

  return <section className="prices" aria-label="Price table" aria-busy={busy}>
    <header className="prices-head">
      <button type="button" className="btn btn-small" onClick={onBack}>← Procurement</button>
      <h1>{table?.title ?? "Package"} <span>{compare ? `· comparing ${cols.length} firms` : cols.length === 1 ? "· review" : ""}</span></h1>
      <div className="prices-actions">
        <button type="button" className="btn btn-small" onClick={() => setAdding(a => !a)} aria-expanded={adding}>+ Add firm</button>
        {reportId && <a className="btn btn-small" href={submissionApi.responseFormURL(reportId)} download>Response form</a>}
        {cols.length > 0 && <a className="btn btn-small" href={submissionApi.exportURL(projectId, pkg)} download>Export to Excel</a>}
        {cols.length > 0 && <button type="button" className="btn btn-small" onClick={() => setShowReport(true)}>Report</button>}
      </div>
    </header>
    {error && <div className="banner" role="alert"><p>{error}</p><button type="button" className="btn btn-small" onClick={() => setError("")}>Dismiss</button></div>}
    {adding && <AddFirm projectId={projectId} pkg={pkg} onDone={t => { setTable(t); setAdding(false); }} onFail={fail} onCancel={() => setAdding(false)} />}
    {table?.award && <AwardBanner table={table} busy={busy} onUndo={() => void run(() => submissionApi.undoAward(projectId, pkg))} onOpenCostPlan={onOpenCostPlan} />}
    {!table ? !error && <p role="status">Loading prices…</p> : cols.length === 0 ? <div className="prices-empty">
      <p><strong>No proposals yet.</strong> Drop a fee proposal on this package's row in Procurement, or drop it anywhere in the document register: proposals are recognised when they are filed.</p>
      <p>The table will list the {base.length + options.length} fee rows of this package's schedule{rates.length ? ` and ${rates.length} hourly rates` : ""}.</p>
    </div> : <div className="prices-scroll">
      <table className="prices-table">
        <thead><tr>
          <th scope="col" className="prices-item">Item <span>ex GST</span></th>
          {cols.map(c => <th scope="col" key={c.submission.id} className={c.totals.lowest ? "prices-firm lowest" : "prices-firm"}>
            <button type="button" className="prices-firm-name" onClick={() => setPanel(panel === c.submission.id ? "" : c.submission.id)} aria-expanded={panel === c.submission.id}>
              {c.firm?.name ?? c.submission.filename}{c.firm?.status === "draft" && <span className="prices-draft" title="New to your firm directory: check the details">new</span>}
            </button>
            <span className="prices-firm-meta">{day(c.submission.received_on)} · <span className="prices-status" data-status={c.submission.status}>{STATUS_WORDS[c.submission.status]}</span></span>
            {!table.award && (c.submission.status === "received" || c.submission.status === "needs_review") && <button type="button" className="btn btn-small prices-award" disabled={busy} onClick={() => setAwardFor(c.submission.id)}>Award</button>}
          </th>)}
        </tr></thead>
        <tbody>
          {base.map(it => <Row key={it.key} item={it} cols={cols} editing={editing} onEdit={setEditing} onSave={(sub, state, amount) => void run(() => submissionApi.setCell(projectId, sub, it.key, state, amount)).then(() => setEditing(null))} busy={busy} />)}
          <tr className="prices-total">
            <th scope="row">Base fee total</th>
            {cols.map(c => <td key={c.submission.id} className={c.totals.lowest ? "lowest" : undefined}>
              <span className="prices-figure">{c.totals.base_total != null ? money(c.totals.base_total) : `${money(c.totals.known_base)} + ?`}</span>
              {c.totals.lowest && <span className="prices-note ok">Lowest complete price</span>}
              {c.totals.not_priced > 0 && <span className="prices-note warn">{c.totals.not_priced} item{c.totals.not_priced === 1 ? "" : "s"} not priced</span>}
              {c.totals.unread > 0 && <span className="prices-note warn">{c.totals.unread} not read</span>}
              {c.totals.gap != null && <span className="prices-note warn" title={`Stated total ${money(c.totals.stated_total)}`}>Stated total differs by {money(c.totals.gap)}</span>}
            </td>)}
          </tr>
          {options.length > 0 && <tr className="prices-group"><th colSpan={cols.length + 1} scope="rowgroup">Options, priced separately</th></tr>}
          {options.map(it => <Row key={it.key} item={it} cols={cols} editing={editing} onEdit={setEditing} onSave={(sub, state, amount) => void run(() => submissionApi.setCell(projectId, sub, it.key, state, amount)).then(() => setEditing(null))} busy={busy} />)}
          {cols.some(c => c.others.length > 0) && <tr className="prices-others">
            <th scope="row">Offered, not asked for <span>kept out of totals</span></th>
            {cols.map(c => <td key={c.submission.id}>{c.others.length ? <ul>{c.others.map(o => <li key={o.id} title={o.passage}>{money(o.amount)} <span>{o.label}</span></li>)}</ul> : "—"}</td>)}
          </tr>}
          {rates.length > 0 && <tr className="prices-group"><th colSpan={cols.length + 1} scope="rowgroup">Hourly rates, for variations</th></tr>}
          {rates.map(it => <Row key={it.key} item={it} cols={cols} editing={editing} onEdit={setEditing} onSave={(sub, state, amount) => void run(() => submissionApi.setCell(projectId, sub, it.key, state, amount)).then(() => setEditing(null))} busy={busy} perHour />)}
          {cols.length > 0 && unpriced.length > 0 && <tr className="prices-unpriced"><th scope="row" colSpan={cols.length + 1}>
            Not priced by any firm: {unpriced.map(i => i.label).join("; ")}.
          </th></tr>}
          {cols.some(c => c.submission.flags.length > 0) && <tr className="prices-flags">
            <th scope="row">Read notes</th>
            {cols.map(c => <td key={c.submission.id}>{c.submission.flags.filter(f => FLAG_WORDS[f]).map(f => <p key={f}>{FLAG_WORDS[f]}</p>)}</td>)}
          </tr>}
        </tbody>
      </table>
    </div>}
    {cols.length > 0 && <p className="prices-legend">Figures are read from each proposal and are ex GST. A cell marked <em>Check</em> or <em>Not read</em> needs your eye; click any cell to correct it. Your corrections are final.</p>}
    {table && table.earlier.length > 0 && <details className="prices-earlier"><summary>{table.earlier.length} earlier or set-aside proposal{table.earlier.length === 1 ? "" : "s"}</summary>
      <ul>{table.earlier.map(s => <li key={s.id}>{s.filename} · {day(s.received_on)} · {STATUS_WORDS[s.status]}
        {(s.status === "withdrawn" || s.status === "not_proposal") && <button type="button" className="table-action" onClick={() => void run(() => submissionApi.edit(projectId, s.id, { status: "received" }))}>Restore</button>}</li>)}</ul>
    </details>}
    {selected && <SubmissionPanel projectId={projectId} col={selected} busy={busy} onClose={() => setPanel("")}
      onSave={edit => void run(() => submissionApi.edit(projectId, selected.submission.id, edit))} />}
    {awardFor && table && <AwardDialog projectId={projectId} table={table} submissionId={awardFor} onClose={() => setAwardFor("")} onAwarded={t => { setTable(t); setAwardFor(""); }} onSignedOut={onSignedOut} />}
  </section>;
}

function Row({ item, cols, editing, onEdit, onSave, busy, perHour }: {
  item: ScheduleItem; cols: Column[]; editing: { sub: string; key: string } | null; busy: boolean; perHour?: boolean;
  onEdit: (e: { sub: string; key: string } | null) => void; onSave: (sub: string, state: CellState, amount: string) => void;
}) {
  return <tr>
    <th scope="row" className="prices-item" title={item.detail}>{item.label}{perHour && <span> per hour</span>}</th>
    {cols.map(c => {
      const cell = c.cells[item.key];
      const open = editing?.sub === c.submission.id && editing.key === item.key;
      return <td key={c.submission.id} className={cell?.decided_by === "user" ? "user" : undefined}>
        {open ? <CellEditor cell={cell} busy={busy} onCancel={() => onEdit(null)} onSave={(s, a) => onSave(c.submission.id, s, a)} />
          : <button type="button" className="prices-cell" disabled={c.submission.status === "awarded" || c.submission.status === "unsuccessful"} onClick={() => onEdit({ sub: c.submission.id, key: item.key })}
            aria-label={`${item.label}, ${c.firm?.name ?? c.submission.filename}: ${cellWords(cell)}. Change`} title={provenance(cell)}>
            <span className={cell?.state === "priced" ? "prices-figure" : "prices-word"}>{cellWords(cell)}</span>
            {cell && cell.state !== "" && !(cell.band === "green" && cell.decided_by !== "user") && <Chip field={{ value: cell.state, band: cell.band, decided_by: cell.decided_by }} />}
            {cell?.basis && <span className="prices-basis">{cell.basis}</span>}
          </button>}
      </td>;
    })}
  </tr>;
}

function provenance(cell: Cell | undefined): string | undefined {
  if (!cell || cell.state === "") return undefined;
  const how = cell.decided_by === "user" ? "Set by you" : cell.decided_by === "rule" ? "Read by rule" : cell.band === "green" ? "Read by Jev, confirmed" : "Read by Jev, check";
  return cell.passage ? `${how}. Printed: ${cell.passage}${cell.page ? ` (page ${cell.page})` : ""}` : how;
}

function cellWords(cell: Cell | undefined): string {
  if (!cell) return "Not read";
  if (cell.state === "priced") return cell.amount == null ? "Priced, amount unknown" : money(cell.amount);
  return CELL_WORDS[cell.state] ?? cell.state;
}

function CellEditor({ cell, busy, onSave, onCancel }: { cell?: Cell; busy: boolean; onSave: (state: CellState, amount: string) => void; onCancel: () => void }) {
  const [state, setState] = useState<CellState>(cell?.state ? cell.state : "priced");
  const [amount, setAmount] = useState(cell?.amount ?? "");
  const key = (e: KeyboardEvent) => { if (e.key === "Enter") onSave(state, amount); if (e.key === "Escape") onCancel(); };
  return <div className="prices-editor" onKeyDown={key}>
    <select aria-label="Answer" value={state} onChange={e => setState(e.target.value as CellState)} disabled={busy}>
      <option value="priced">Amount</option><option value="not_priced">Not priced</option><option value="included">Included elsewhere</option><option value="excluded">Excluded</option>
    </select>
    {state === "priced" && <input aria-label="Amount ex GST" inputMode="decimal" autoFocus value={amount} onChange={e => setAmount(e.target.value)} disabled={busy} placeholder="0.00" />}
    <button type="button" className="btn btn-small" disabled={busy} onClick={() => onSave(state, amount.replace(/[,$\s]/g, ""))}>Save</button>
    <button type="button" className="table-action" onClick={onCancel}>Cancel</button>
  </div>;
}

function SubmissionPanel({ projectId, col, busy, onClose, onSave }: { projectId: string; col: Column; busy: boolean; onClose: () => void; onSave: (e: { firm?: FirmFields; status?: string; received_on?: string }) => void }) {
  const f = col.firm;
  const [firm, setFirm] = useState<FirmFields>({ name: f?.name ?? "", abn: f?.abn ?? "", address: f?.address ?? "", contact: f?.contact ?? "", email: f?.email ?? "", phone: f?.phone ?? "" });
  useEffect(() => { const k = (e: globalThis.KeyboardEvent) => { if (e.key === "Escape" && !(e.target instanceof HTMLInputElement)) onClose(); }; window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k); }, [onClose]);
  const field = (k: keyof FirmFields, label: string) => <label className="prices-field">{label}
    <input value={firm[k]} onChange={e => setFirm({ ...firm, [k]: e.target.value })} disabled={busy} />
    {col.fields[`firm_${k === "name" ? "name" : k}`] && <Chip field={col.fields[`firm_${k}`]} />}
  </label>;
  return <aside className="prices-panel" aria-label="Proposal">
    <header><h2>{f?.name ?? col.submission.filename}</h2><button type="button" className="btn btn-small" onClick={onClose}>Close</button></header>
    <div className="prices-panel-body">
      {col.submission.document_id ? <iframe title={`Proposal ${col.submission.filename}`} src={submissionApi.fileURL(projectId, col.submission.id)} /> : <p className="prices-empty">No file: prices for this firm are typed in the table.</p>}
      <form onSubmit={e => { e.preventDefault(); onSave({ firm }); }}>
        <h3>Firm {f?.status === "draft" && <span className="prices-draft">new to your directory</span>}</h3>
        {field("name", "Name")}{field("abn", "ABN")}{field("contact", "Contact")}{field("email", "Email")}{field("phone", "Phone")}{field("address", "Address")}
        <button className="btn btn-small" disabled={busy || !firm.name.trim()}>{f?.status === "draft" ? "Confirm firm" : "Save firm"}</button>
        <label className="prices-field">Received<input type="date" defaultValue={col.submission.received_on} disabled={busy} onBlur={e => { if (e.target.value && e.target.value !== col.submission.received_on) onSave({ received_on: e.target.value }); }} /></label>
        {col.submission.status !== "awarded" && <button type="button" className="table-action" disabled={busy} onClick={() => onSave({ status: "withdrawn" })}>Mark withdrawn</button>}
      </form>
    </div>
  </aside>;
}

function AddFirm({ projectId, pkg, onDone, onFail, onCancel }: { projectId: string; pkg: string; onDone: (t: PriceTable) => void; onFail: (e: unknown) => void; onCancel: () => void }) {
  const [q, setQ] = useState("");
  const [found, setFound] = useState<{ id: string; name: string; abn: string }[]>([]);
  useEffect(() => {
    let stopped = false;
    const t = setTimeout(() => submissionApi.firms(q).then(r => { if (!stopped) setFound(r.firms); }).catch(() => undefined), 150);
    return () => { stopped = true; clearTimeout(t); };
  }, [q]);
  const add = (body: { firm_id: string } | { firm: FirmFields }) => submissionApi.addFirm(projectId, pkg, body).then(onDone).catch(onFail);
  return <div className="prices-add" role="dialog" aria-label="Add firm">
    <input autoFocus placeholder="Search your firms, or type a new name" value={q} onChange={e => setQ(e.target.value)} />
    <ul>{found.slice(0, 8).map(f => <li key={f.id}><button type="button" onClick={() => void add({ firm_id: f.id })}>{f.name}{f.abn && <span> ABN {f.abn}</span>}</button></li>)}</ul>
    {q.trim() && !found.some(f => f.name.toLowerCase() === q.trim().toLowerCase()) && <button type="button" className="btn btn-small" onClick={() => void add({ firm: { name: q.trim(), abn: "", address: "", contact: "", email: "", phone: "" } })}>Add “{q.trim()}” as a new firm</button>}
    <button type="button" className="table-action" onClick={onCancel}>Cancel</button>
    <p className="prices-hint">Or drop the firm's proposal on the package row: it is read and its prices fill the column.</p>
  </div>;
}

function AwardBanner({ table, busy, onUndo, onOpenCostPlan }: { table: PriceTable; busy: boolean; onUndo: () => void; onOpenCostPlan?: () => void }) {
  const a = table.award!;
  return <div className="prices-awarded" role="status">
    <p><strong>Awarded to {a.firm_name}</strong>{a.created_at && ` on ${new Date(a.created_at).toLocaleDateString("en-AU", { day: "numeric", month: "short", year: "numeric" })}`}: {a.lines.length} cost plan line{a.lines.length === 1 ? "" : "s"} under “{a.parent_label}”, {money(a.total)} ex GST.
      {a.budget != null ? ` Budget ${money(a.budget)}.` : " No budget set, so no variance is shown."}</p>
    <div>
      {onOpenCostPlan && <button type="button" className="btn btn-small" onClick={onOpenCostPlan}>Open cost plan</button>}
      {a.locked ? <span className="prices-note">Undo is locked: an invoice is allocated to these lines.</span>
        : <button type="button" className="btn btn-small" disabled={busy} onClick={onUndo}>Undo award</button>}
    </div>
  </div>;
}
