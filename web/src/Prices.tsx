import { ConsultantComparison } from "./ConsultantComparison";
import { Pending } from "./Activity";
import { BuilderQuotedTotals, BuilderWorkSections } from "./BuilderComparison";
// The price table: schedule items down the side, firms across. With one firm
// it is the review screen; with several it is the comparison. Every total
// arrives computed by the server; a cell a user corrects is final.
import { useCallback, useEffect, useMemo, useState, type KeyboardEvent } from "react";
import { ApiError } from "./api";
import { AwardDialog } from "./AwardDialog";
import { ComparisonRowEditor } from "./ComparisonAlignment";
import { fmt } from "./CostGrid";
import { Chip } from "./Invoices";
import { ProcurementReport } from "./ProcurementReport";
import { packageApi, type PackageRecord } from "./packageApi";
import { FLAG_WORDS, STATUS_WORDS, submissionApi, uploadProposal, type Cell, type CellState, type Column, type FirmFields, type PriceTable, type ScheduleItem } from "./submissionApi";
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
  const [rowEditor, setRowEditor] = useState<ScheduleItem | "new" | null>(null);
  const [moving, setMoving] = useState<{ sub: string; cell: Cell; version: number; target: string; rate: boolean } | null>(null);

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
    try { setTable(await action()); return true; } catch (e) { fail(e); return false; } finally { setBusy(false); }
  }

  async function reassign(sub: string, destination: string) {
    setBusy(true); setError("");
    try { await submissionApi.edit(projectId, sub, { package_id: destination }); onBack(); }
    catch (e) { fail(e); } finally { setBusy(false); }
  }

  const [uploading, setUploading] = useState("");
  async function attach(submissionId: string, file?: File) {
    if (!file || busy) return;
    setUploading(submissionId);
    await run(async () => {
      await uploadProposal(projectId, pkg, file, submissionId);
      return submissionApi.table(projectId, pkg);
    });
    setUploading("");
  }

  const items = (table?.items ?? []).filter(i => i.key !== "comparison:appointment");
  const base = useMemo(() => items.filter(i => (i.kind === "stage" || i.kind === "element" || i.kind === "reimbursable") && i.base), [items]);
  const cols = table?.columns ?? [];
  const allOptions = useMemo(() => items.filter(i => i.kind === "option"), [items]);
  const allRates = useMemo(() => items.filter(i => i.kind === "rate"), [items]);
  const options = allOptions;
  const rates = allRates;
  const unpriced: ScheduleItem[] = [];
  const compare = cols.length > 1;
  const quoted = !table?.builder && cols.some(c => c.quote);
  const selected = cols.find(c => c.submission.id === panel);

  const moveColumn = cols.find(c => c.submission.id === moving?.sub);
  const destination = items.find(i => i.key === moving?.target);
  const destinationCell = destination && moveColumn?.cells[destination.key];
  const combining = destinationCell?.state === "priced" && destinationCell.amount != null;
  const locked = busy || !!table?.award;
  function startMove(col: Column, cell: Cell) {
    setEditing(null);
    setMoving({ sub: col.submission.id, cell, version: col.submission.version, target: "", rate: items.some(i => i.key === cell.item_key && i.kind === "rate") });
  }
  async function finishMove(exclude = false) {
    if (!moving) return;
    const ids = [moving.cell.id];
    if (!exclude && combining && destinationCell && destinationCell.id !== moving.cell.id) ids.push(destinationCell.id);
    if (await run(() => submissionApi.align(projectId, moving.sub, moving.version, moving.target, ids, false, exclude))) setMoving(null);
  }
  function renderRow(it: ScheduleItem) {
    if (table?.builder && it.key === "stage:lump_sum" && it.label === "Lump sum fee") it = { ...it, label: "Construction price used" };
    return <Row key={it.key} item={it} cols={cols} editing={editing} onEdit={setEditing}
      onSave={(sub, state, amount, note) => void run(() => submissionApi.setCell(projectId, sub, it.key, state, amount, note)).then(ok => { if (ok) setEditing(null); })}
      busy={locked} perHour={it.kind === "rate"} moving={moving} onMove={startMove}
      onTarget={target => setMoving(m => m ? { ...m, target } : null)} onRename={() => setRowEditor(it)}
      onPlacement={() => void run(() => submissionApi.row(projectId, pkg, it.key, it.label, it.base ? "option" : "base", it.unit || ""))} />;
  }

  if (showReport && table) return <ProcurementReport projectId={projectId} table={table} onBack={() => setShowReport(false)} onSignedOut={onSignedOut} />;

  return <section className="prices" aria-label="Price table" aria-busy={busy}>
    <header className="prices-head">
      <button type="button" className="btn btn-small" onClick={onBack}>← Procurement</button>
      <h1>{table?.title ?? "Package"} <span>{compare ? `· comparing ${cols.length} firms` : cols.length === 1 ? "· review" : ""}</span></h1>
      <div className="prices-actions"><span role="status">{busy && !uploading && <Pending delayed>Saving…</Pending>}</span>
        {cols.length > 0 && <button className="btn btn-small" disabled={locked || !!moving} onClick={() => setRowEditor("new")}>+ Add row</button>}
        <button type="button" className="btn btn-small" onClick={() => setAdding(a => !a)} aria-expanded={adding}>+ Add firm</button>
        {reportId && <a className="btn btn-small" href={submissionApi.responseFormURL(reportId)} download>Response form</a>}
        {cols.length > 0 && <a className="btn btn-small" href={submissionApi.exportURL(projectId, pkg)} download>Export to Excel</a>}
        {cols.length > 0 && <button type="button" className="btn btn-small" onClick={() => setShowReport(true)}>Report</button>}
      </div>
    </header>
    {error && <div className="banner" role="alert"><p>{error}</p><button type="button" className="btn btn-small" onClick={() => setError("")}>Dismiss</button></div>}
    {adding && <AddFirm projectId={projectId} pkg={pkg} onDone={t => { setTable(t); setAdding(false); }} onFail={fail} onCancel={() => setAdding(false)} />}
    {rowEditor && <ComparisonRowEditor key={rowEditor === "new" ? "new" : rowEditor.key} item={rowEditor === "new" ? undefined : rowEditor} busy={locked} onCancel={() => setRowEditor(null)} onSave={async (key, label, section, unit) => { if (await run(() => submissionApi.row(projectId, pkg, key, label, section, unit))) setRowEditor(null); }} />}
    {moving && <div className="prices-move-bar" role="region" aria-label="Move or combine fee">
      <p><strong>{moveColumn?.firm?.name ?? moveColumn?.submission.filename}</strong>: {money(moving.cell.amount)} · {moving.cell.label || items.find(i => i.key === moving.cell.item_key)?.label}<br />
      {destination ? `${combining ? "Combine with the existing fee in" : "Move to"} ${destination.label}.` : "Choose a destination in this firm’s column."}</p>
      <button className="btn btn-small" disabled={locked || !destination} onClick={() => void finishMove()}>{combining ? "Combine fees" : "Move fee"}</button>
      <button className="table-action" disabled={locked} onClick={() => void finishMove(true)}>Set aside</button>
      <button className="table-action" disabled={busy} onClick={() => setMoving(null)}>Cancel</button>
    </div>}
    {table?.award && <AwardBanner table={table} busy={busy} onUndo={() => void run(() => submissionApi.undoAward(projectId, pkg))} onOpenCostPlan={onOpenCostPlan} />}
    {table && quoted && <ConsultantComparison table={table} projectId={projectId} busy={busy} onOpen={setPanel} onAward={setAwardFor} onCoverage={(sub, key, state, note) => run(() => submissionApi.setCell(projectId, sub, key, state, "", note))} onConfirm={(sub, amount, note) => run(() => submissionApi.setCell(projectId, sub, "comparison:appointment", "priced", amount, note))} />}
    <details open={!quoted}><summary>{quoted ? "Schedule details and scope corrections" : "Fee schedule"}</summary>
    {!table ? !error && <p role="status"><Pending delayed>Loading prices…</Pending></p> : cols.length === 0 ? <div className="prices-empty">
      <p><strong>No proposals yet.</strong> Drop a fee proposal on this package's row in Procurement, or drop it anywhere in the document register: proposals are recognised when they are filed.</p>
      <p>The table will list the {base.length + options.length} fee rows of this package's schedule{rates.length ? ` and ${rates.length} hourly rates` : ""}.</p>
    </div> : <div className="prices-scroll">
      <table className={`prices-table${table.builder ? " builder-compact" : ""}`}>
        <colgroup><col className="prices-label-col" />{cols.map(c => <col key={c.submission.id} />)}<col className="prices-action-col" /></colgroup>
        <thead><tr>
          <th scope="col" className="prices-item">{table.builder ? "Work section" : "Item"} <span>ex GST</span></th>
          {cols.map(c => <th scope="col" key={c.submission.id} className={c.totals.lowest ? "prices-firm lowest" : "prices-firm"}
            onDragOver={e => { if (!c.submission.document_id) { e.preventDefault(); e.stopPropagation(); } }}
            onDrop={e => { if (!c.submission.document_id) { e.preventDefault(); e.stopPropagation(); void attach(c.submission.id, e.dataTransfer.files[0]); } }}>
            <button type="button" className="prices-firm-name" onClick={() => setPanel(panel === c.submission.id ? "" : c.submission.id)} aria-expanded={panel === c.submission.id}>
              {c.firm?.name ?? c.submission.filename.replace(/\.pdf$/i, "")}{!table.builder && c.firm?.status === "draft" && <span className="prices-draft" title="New to your firm directory: check the details">new</span>}
            </button>
            <details className={table.builder ? "builder-firm-actions" : "prices-firm-actions"} open={table.builder ? undefined : true}>
            <summary hidden={!table.builder} aria-label={`Options for ${c.firm?.name ?? c.submission.filename}`}>⋯</summary>
            <span className="prices-firm-meta">{day(c.submission.received_on)} · <span className="prices-status" data-status={c.submission.status}>{c.submission.status === "reading" ? <Pending>Reading proposal…</Pending> : STATUS_WORDS[c.submission.status]}</span></span>
            {!c.submission.document_id && c.submission.status !== "awarded" && <label className="prices-upload">
              <span>{uploading === c.submission.id ? <Pending>Uploading proposal…</Pending> : "Upload proposal"}</span>
              <input type="file" aria-label={`Upload proposal for ${c.firm?.name ?? c.submission.filename}`} accept=".pdf,.docx,.xlsx" disabled={busy} onChange={e => { void attach(c.submission.id, e.target.files?.[0]); e.target.value = ""; }} />
              <small>Or drop here. Also saved in Documents.</small>
            </label>}
            {!quoted && !table.award && (c.submission.status === "received" || c.submission.status === "needs_review") && <button type="button" className="btn btn-small prices-award" disabled={busy || (table.builder && (c.cells["stage:lump_sum"]?.decided_by !== "user" || c.cells["stage:lump_sum"]?.state !== "priced"))} onClick={() => setAwardFor(c.submission.id)}>Award</button>}
            {c.undo_alignment && !table.award && <button className="table-action prices-undo" disabled={busy} onClick={() => void run(() => submissionApi.align(projectId, c.submission.id, c.submission.version, "", [], true)).then(ok => { if (ok) setMoving(null); })}>Undo last alignment</button>}
            </details>
          </th>)}
          <th scope="col"><span className="prices-sr-only">Row placement</span></th>
        </tr></thead>
        <tbody>
          {table.builder && <BuilderWorkSections cols={cols} projectId={projectId} />}
          {base.filter(it=>!table.builder || it.key!=="stage:lump_sum").map(renderRow)}
          {!table.builder && <tr className="prices-total">
            <th scope="row">{table.builder ? "Comparison total" : quoted ? "Confirmed comparison fee — independent of schedule breakdown" : "Base fee total"}</th>
            {cols.map(c => <td key={c.submission.id} className={c.totals.lowest ? "lowest" : undefined}>
              <span className="prices-figure">{c.totals.base_total != null ? money(c.totals.base_total) : c.totals.known_base && c.totals.known_base !== "0.00" ? `${money(c.totals.known_base)} known` : "Unknown"}</span>
              {table.builder && c.cells["stage:lump_sum"]?.decided_by !== "user" && <span className="prices-note warn">Not confirmed by you</span>}
              {c.totals.base_total == null && <span className="prices-note warn">{table.builder ? "Comparison amount unresolved" : "Incomplete fee"}</span>}
              {!table.builder && c.totals.stated_total != null && <span className="prices-note">Proposal states {money(c.totals.stated_total)}</span>}
              {c.totals.lowest && <span className="prices-note ok">Lowest complete price</span>}
              {c.totals.not_priced > 0 && <span className="prices-note warn">{c.totals.not_priced} item{c.totals.not_priced === 1 ? "" : "s"} not priced</span>}
              {c.totals.unread > 0 && <span className="prices-note warn">{c.totals.unread} not read</span>}
              {!table.builder && c.totals.gap != null && <span className="prices-note warn" title={`Stated total ${money(c.totals.stated_total)}`}>Stated total differs by {money(c.totals.gap)}</span>}
            </td>)}<td />
          </tr>}
          {options.length > 0 && <tr className="prices-group"><th colSpan={cols.length + 2} scope="rowgroup">Options, priced separately</th></tr>}
          {options.map(renderRow)}
          {!table.builder && cols.some(c => c.others.length > 0) && <tr className="prices-others">
            <th scope="row">Fees awaiting alignment <span>not included in comparison totals</span></th>
            {cols.map(c => <td key={c.submission.id}>{c.others.length ? <ul>{c.others.map(o => <li key={o.id} title={o.passage}>{o.state === "excluded" ? "Set aside" : money(o.amount)} <span>{o.label}</span>{o.state === "priced" && o.amount != null && <button className="price-icon" disabled={locked || !!moving} aria-label={`Move or combine ${o.label}, ${c.firm?.name ?? c.submission.filename}`} title="Move or combine fee" onClick={() => startMove(c, o)}><PriceIcon kind="move" /></button>}</li>)}</ul> : "—"}</td>)}<td />
          </tr>}
          {rates.length > 0 && <tr className="prices-group"><th colSpan={cols.length + 2} scope="rowgroup">Rates, for variations</th></tr>}
          {rates.map(renderRow)}
          {table.builder && <BuilderQuotedTotals cols={cols} busy={locked} onUse={(sub,amount,note)=>void run(()=>submissionApi.setCell(projectId,sub,"stage:lump_sum","priced",amount,note))} />}
          {!table.builder && cols.length > 0 && unpriced.length > 0 && <tr className="prices-unpriced"><th scope="row" colSpan={cols.length + 2}>
            Not priced by any firm: {unpriced.map(i => i.label).join("; ")}.
          </th></tr>}
          {!table.builder && cols.some(c => c.submission.flags.length > 0) && <tr className="prices-flags">
            <th scope="row">Read notes</th>
            {cols.map(c => <td key={c.submission.id}>{c.submission.flags.filter(f => FLAG_WORDS[f] && !(table.builder && f.startsWith("gst_"))).map(f => <p key={f}>{FLAG_WORDS[f]}</p>)}</td>)}<td />
          </tr>}
        </tbody>
      </table>
    </div>}
    </details>
    {cols.length > 0 && !quoted && <p className="prices-legend">{table?.builder ? "Review the quoted total, then use it or enter your confirmed comparison amount. Expand a work section to check the source wording. Allowances, options and components are not added to the quoted total. Your corrections are preserved." : "Figures are read from each proposal and are ex GST. The warning triangle means review needed. Click a fee for its source and notes. Use row arrows to change base/option placement, or the button beside a fee to move or combine it. Your corrections are final."}</p>}
    {table && table.earlier.length > 0 && <details className="prices-earlier"><summary>{table.earlier.length} earlier or set-aside proposal{table.earlier.length === 1 ? "" : "s"}</summary>
      <ul>{table.earlier.map(s => <li key={s.id}>{s.filename} · {day(s.received_on)} · {STATUS_WORDS[s.status]}
        {(s.status === "withdrawn" || s.status === "not_proposal") && <button type="button" className="table-action" onClick={() => void run(() => submissionApi.edit(projectId, s.id, { status: "received" }))}>Restore</button>}</li>)}</ul>
    </details>}
    {selected && <SubmissionPanel key={selected.submission.id} projectId={projectId} col={selected} busy={busy} error={error} onClose={() => setPanel("")}
      onReassign={destination => void reassign(selected.submission.id, destination)}
      onSave={edit => void run(() => submissionApi.edit(projectId, selected.submission.id, edit))} />}
    {awardFor && table && <AwardDialog projectId={projectId} table={table} submissionId={awardFor} onClose={() => setAwardFor("")} onAwarded={t => { setTable(t); setAwardFor(""); }} onSignedOut={onSignedOut} />}
  </section>;
}

function PriceIcon({ kind }: { kind: "up" | "down" | "move" | "warning" }) {
  return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    {kind === "warning" ? <><path d="M12 3 22 21H2Z" /><path d="M12 9v5m0 3h.01" /></> : kind === "move" ? <><path d="M4 5h5v7h11M16 8l4 4-4 4M4 19h5v-7" /></> : <path d={kind === "up" ? "M12 20V4m-6 6 6-6 6 6" : "M12 4v16m-6-6 6 6 6-6"} />}
  </svg>;
}

function Row({ item, cols, editing, onEdit, onSave, busy, perHour, moving, onMove, onTarget, onRename, onPlacement }: {
  item: ScheduleItem; cols: Column[]; editing: { sub: string; key: string } | null; busy: boolean; perHour?: boolean;
  moving: { sub: string; cell: Cell; target: string; rate: boolean } | null;
  onMove: (col: Column, cell: Cell) => void; onTarget: (key: string) => void; onRename: () => void; onPlacement: () => void;
  onEdit: (e: { sub: string; key: string } | null) => void; onSave: (sub: string, state: CellState, amount: string, note: string) => void;
}) {
  return <tr>
    <th scope="row" className="prices-item"><button className="prices-row-name" title="Edit service description" disabled={busy || !!moving} onClick={onRename}>{item.label}</button>{perHour && <span> per {item.unit || "hour"}</span>}</th>
    {cols.map(c => {
      const cell = c.cells[item.key];
      const open = editing?.sub === c.submission.id && editing.key === item.key;
      const sameFirm = moving?.sub === c.submission.id;
      const source = sameFirm && moving?.cell.id === cell?.id;
      const eligible = sameFirm && !source && (!moving?.rate || perHour) && !(perHour && cell?.state === "priced");
      const review = cell && cell.state !== "" && cell.decided_by !== "user" && cell.band !== "green";
      return <td key={c.submission.id} className={`${cell?.decided_by === "user" ? "user" : ""} ${source ? "prices-source-selected" : ""} ${sameFirm && moving?.target === item.key ? "prices-target-selected" : ""}`}>
        {open ? <CellEditor cell={cell} busy={busy} onCancel={() => onEdit(null)} onSave={(s, a, note) => onSave(c.submission.id, s, a, note)} />
          : <div className="prices-cell-line"><button type="button" className="prices-cell" disabled={busy || !!moving || c.submission.status === "awarded" || c.submission.status === "unsuccessful"} onClick={() => onEdit({ sub: c.submission.id, key: item.key })}
            aria-label={`${item.label}, ${c.firm?.name ?? c.submission.filename}: ${cellWords(cell)}. Change`} title={[provenance(cell), cell?.basis].filter(Boolean).join(". ")}>
            <span className={cell?.state === "priced" ? "prices-figure" : "prices-word"}>{cellWords(cell)}</span>
            {review && <span className="prices-warning" aria-label="Review needed"><PriceIcon kind="warning" /></span>}
            {cell?.basis && <span className="prices-sr-only">{cell.basis}</span>}
          </button>
          {!moving && cell?.state === "priced" && cell.amount != null && <button className="price-icon" disabled={busy} aria-label={`Move or combine ${item.label}, ${c.firm?.name ?? c.submission.filename}`} title="Move or combine fee" onClick={() => onMove(c, cell)}><PriceIcon kind="move" /></button>}
          {eligible && <button className="btn btn-small prices-destination" disabled={busy} aria-pressed={moving?.target === item.key} onClick={() => onTarget(item.key)}>{cell?.state === "priced" ? "Combine here" : "Move here"}</button>}
          {source && <span className="prices-note">Selected</span>}
          </div>}
      </td>;
    })}
    <td className="prices-row-action">{!perHour && <button className="price-icon" disabled={busy || !!moving} title={item.base ? "Move row below the base total" : "Move row into the base fee"} aria-label={`Move ${item.label} to ${item.base ? "options" : "base fee"}`} onClick={onPlacement}><PriceIcon kind={item.base ? "down" : "up"} /></button>}</td>
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

function CellEditor({ cell, busy, onSave, onCancel }: { cell?: Cell; busy: boolean; onSave: (state: CellState, amount: string, note: string) => void; onCancel: () => void }) {
  const [state, setState] = useState<CellState>(cell?.state ? cell.state : "priced");
  const [amount, setAmount] = useState(cell?.amount ?? "");
  const [note, setNote] = useState(cell?.basis ?? "");
  const key = (e: KeyboardEvent) => { if (e.key === "Enter") onSave(state, amount.replace(/[,$\s]/g, ""), note); if (e.key === "Escape") onCancel(); };
  return <div className="prices-editor" onKeyDown={key}>
    {cell?.passage && <p className="prices-source-note">{provenance(cell)}</p>}
    <select aria-label="Answer" value={state} onChange={e => setState(e.target.value as CellState)} disabled={busy}>
      <option value="priced">Amount</option><option value="not_priced">Not priced</option><option value="included">Included elsewhere</option><option value="excluded">Excluded</option>
    </select>
    {state === "priced" && <input aria-label="Amount ex GST" inputMode="decimal" autoFocus value={amount} onChange={e => setAmount(e.target.value)} disabled={busy} placeholder="0.00" />}
    <input aria-label="Scope or allowance note" placeholder="Scope, allowance or bundled fee reference" value={note} maxLength={500} onChange={e=>setNote(e.target.value)} disabled={busy} />
    <button type="button" className="btn btn-small" disabled={busy} onClick={() => onSave(state, amount.replace(/[,$\s]/g, ""), note)}>Save</button>
    <button type="button" className="table-action" onClick={onCancel}>Cancel</button>
  </div>;
}

function SubmissionPanel({ projectId, col, busy, error, onClose, onSave, onReassign }: { projectId: string; col: Column; busy: boolean; error: string; onClose: () => void; onSave: (e: { firm?: FirmFields; status?: string; received_on?: string }) => void; onReassign: (destination: string) => void }) {
  const f = col.firm;
  const [packages, setPackages] = useState<PackageRecord[] | null>(null);
  const [destination, setDestination] = useState(col.submission.package_id);
  const [packageError, setPackageError] = useState("");
  useEffect(() => {
    let stopped = false;
    packageApi.list(projectId).then(p => { if (!stopped) setPackages(p.items.filter(p => !p.retired_at && p.lifecycle_status !== "cancelled")); })
      .catch(() => { if (!stopped) setPackageError("Could not load packages. Close and reopen this proposal to retry."); });
    return () => { stopped = true; };
  }, [projectId]);
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
        <h3>Proposal package</h3>
        <p>{col.submission.filename}</p>
        {error && <p role="alert">{error}</p>}
        {packageError && <p role="alert">{packageError}</p>}
        <label className="prices-field">Package<select aria-label="Package" value={destination} disabled={busy || !packages || col.submission.status === "awarded"} onChange={e => setDestination(e.target.value)}>
          <option value="">Unassigned — detach from package</option>
          {!packages && <option value={col.submission.package_id}>Loading packages…</option>}
          {packages?.map(p => <option key={p.id} value={p.id}>{p.title}</option>)}
        </select></label>
        <p className="prices-note">The document, firm details and quoted fees are kept. Detached proposals appear under “needs a package” in Procurement. Review fee alignment after moving.</p>
        <button type="button" className="btn btn-small" disabled={busy || !packages || destination === col.submission.package_id || col.submission.status === "awarded"} onClick={() => onReassign(destination)}>{destination ? "Move proposal" : "Detach proposal"}</button>
        {col.submission.status === "awarded" && <p className="prices-note">Undo the award before moving this proposal.</p>}
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
