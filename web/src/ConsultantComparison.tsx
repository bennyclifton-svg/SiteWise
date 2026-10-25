import { useState } from "react";
import { fmt } from "./CostGrid";
import { submissionApi, type Column, type PriceTable, type QuoteFee, type Cell, type CellState } from "./submissionApi";

const money = (value: string | null) => value === null ? "Tax basis unclear" : `$${fmt(value)}`;
const feeRole = (f: QuoteFee) => f.role.startsWith("Repeated") ? "Repeated reference · do not add again" : f.role.startsWith("Alternative") ? "Alternative · includes other work" : f.role.startsWith("Breakdown") ? "Breakdown · included in package" : f.role.startsWith("Conditional") ? "Conditional / optional" : f.role.includes("total") ? "Package total" : "Quoted fee";
const roleNote = (f: QuoteFee) => `${feeRole(f)}${f.suggestion && f.suggestion !== "Current offer" ? ` · Suggested: ${f.suggestion}` : ""}`;
function headlineFees(fees: QuoteFee[]) {
  const offers = fees.filter(f => (f.role.includes("Package") || f.role.includes("Alternative")) && !f.suggestion?.startsWith("Earlier"));
  // A bare repeated figure adds no scope information. Keep the named offer
  // prominent; every occurrence remains available in the source fee lines.
  return offers.filter(f => f.label !== "Quoted fee" || !offers.some(other => other.id !== f.id && (f.amount !== null ? other.amount === f.amount : other.printed === f.printed) && other.label !== "Quoted fee"));
}
const statedNote = (c: Column) => `Firm's stated total, ex GST, as printed in ${c.submission.filename}.`.slice(0, 500);

export function ConsultantComparison({ table, projectId, busy, onConfirm, onCoverage, onOpen, onAward, onSave }: {
  table: PriceTable; projectId: string; busy: boolean;
  onConfirm: (sub: string, amount: string, note: string) => Promise<boolean>;
  onCoverage: (sub: string, key: string, state: "included" | "excluded" | "not_priced", note: string) => Promise<boolean>;
  onSave: (sub: string, key: string, state: CellState, amount: string, note: string) => Promise<boolean>;
  onOpen: (sub: string) => void; onAward: (sub: string) => void;
}) {
  const [editing, setEditing] = useState("");
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  const [awardAfterConfirm, setAwardAfterConfirm] = useState(false);
  const [later, setLater] = useState(false);
  const locked = busy || !!table.award;
  const rows = table.items.filter(i => i.kind !== "criterion" && i.kind !== "rate" && !i.key.startsWith("comparison:") && (i.base || later));
  const additions = table.items.filter(i => i.custom && !i.retired && i.key !== "comparison:appointment" && i.kind !== "rate");
  const sourceCount = Math.max(0, ...table.columns.map(c => c.quote?.fees.length ?? 0));
  function edit(col: Column, fee?: QuoteFee, stated?: string | null) {
    setAwardAfterConfirm(false);
    setEditing(col.submission.id);
    setAmount(fee?.amount ?? col.cells["comparison:appointment"]?.amount ?? stated ?? "");
    setNote(fee ? `${fee.label}. Source page ${fee.page}: ${fee.source}`.slice(0, 500) : col.cells["comparison:appointment"]?.basis ?? (stated ? statedNote(col) : ""));
  }
  // Award in one click: an unconfirmed stated total is recorded as the PM's
  // comparison fee (the click is the confirmation), then the award opens.
  async function awardAt(col: Column, stated: string | null) {
    if (stated && !(await onConfirm(col.submission.id, stated, statedNote(col)))) return;
    onAward(col.submission.id);
  }
  return <div className="consultant-comparison">
    <div className="prices-scroll"><table className="prices-table">
      <caption className="prices-sr-only">Quoted fees and scope coverage</caption>
      <thead><tr><th scope="col"><span className="prices-sr-only">Fee item</span></th>{table.columns.map(c => <th scope="col" key={c.submission.id}>
        <button className="table-action" onClick={() => onOpen(c.submission.id)}>{c.firm?.name || c.submission.filename}</button>
        {c.firm && table.columns.some(other => other.submission.id !== c.submission.id && other.firm?.id === c.firm?.id) && <small>Another document for this firm is shown. Do not add their fees together.</small>}
      </th>)}</tr></thead>
      <tbody>
        {Array.from({ length: sourceCount }, (_, index) => <tr key={`source-${index}`} className="quote-source-row">
          <th scope="row">{table.columns.length === 1 ? table.columns[0].quote?.fees[index]?.label : `Source fee ${index + 1}`}</th>
          {table.columns.map(c => { const fee = c.quote?.fees[index]; return <td key={c.submission.id}>{fee ? <>
            {table.columns.length > 1 && <span>{fee.label}</span>}
            <strong title={`${roleNote(fee)}. ${fee.source}`}>{fee.amount != null ? money(fee.amount) : fee.printed}</strong>
          </> : "—"}</td>; })}
        </tr>)}
        {additions.filter(i => i.base).map(row => <tr key={row.key}><th scope="row">{row.label}<span>Added fee</span></th>{table.columns.map(c => <td key={c.submission.id}><AddedFee cell={c.cells[row.key]} label={`${row.label}, ${c.firm?.name || c.submission.filename}`} busy={locked} onSave={(state, amount, note) => onSave(c.submission.id, row.key, state, amount, note)} /></td>)}</tr>)}
        <tr className="prices-total"><th scope="row">Total<span>ex GST</span></th>{table.columns.map(c => {
          const confirmed = c.cells["comparison:appointment"];
          // The firm's own printed total, read from its proposal. Offered only
          // when it is known to be ex GST: the server stores it as printed.
          const stated = !confirmed?.amount && c.submission.gst_basis === "ex" ? c.totals.stated_total : null;
          const firm = c.firm?.name || "this consultant";
          return <td key={c.submission.id}>
            <strong>{c.appointment_total != null ? money(c.appointment_total) : "Not confirmed"}</strong>
            {!table.award && <button className="btn btn-primary btn-small" disabled={busy || additions.some(i => i.base && c.cells[i.key]?.state !== "included" && (c.cells[i.key]?.state !== "priced" || c.cells[i.key]?.amount == null))} onClick={() => {
              if (confirmed?.amount != null || stated != null) void awardAt(c, stated);
              else { edit(c); setAwardAfterConfirm(true); }
            }}>Award to {firm}</button>}

          </td>;
        })}</tr>
        <tr className="prices-group"><th scope="rowgroup" colSpan={table.columns.length + 1}>Proposal details</th></tr>
        <tr><th scope="row">Quoted packages and totals<span>Alternatives and breakdowns are not added together</span></th>
          {table.columns.map(c => <td key={c.submission.id}>
            {!c.quote ? <p>Source reading not yet available.</p> : <>
              <ul className="quoted-fees">{headlineFees(c.quote.fees).map(f => <li key={f.id}>
                <strong>{f.amount !== null ? money(f.amount) : f.printed}</strong> <span>{f.label}</span>
                {/^sub\s*total$/i.test(f.label) && <small>{f.heading}</small>}
                <small>{roleNote(f)}{f.amount === null ? " · GST basis unclear" : ""}</small>
                <a href={submissionApi.fileURL(projectId, c.submission.id, f.page)} target="_blank" rel="noreferrer">{f.page > 0 ? `Page ${f.page}` : "Open source"}</a>
                <button className="table-action" disabled={locked || !f.amount} onClick={() => edit(c, f)}>Review for comparison</button>
              </li>)}</ul>
              {!c.quote.fees.some(f => f.role.includes("Package") || f.role.includes("Alternative")) && <p>No single package total established. Review the quoted fee lines above.</p>}
              {c.quote.issues.map(i => <p className="prices-note warn" key={i}>{i}</p>)}
            </>}
          </td>)}
        </tr>
        {([['scope', 'Scope stated in the source'], ['exclusions and qualifications', 'Exclusions and qualifications'], ['conditions', 'Conditions']] as const).map(([kind, label]) => <tr key={kind}><th scope="row">{label}<span>Source wording · check before appointment</span></th>{table.columns.map(c => {
          const terms = c.quote?.terms?.filter(t => t.kind === kind) ?? [];
          return <td key={c.submission.id}>{terms.length ? terms.map((term, i) => <details key={i} open={kind === 'exclusions and qualifications'}><summary>{term.heading}</summary><p style={{ whiteSpace: 'pre-line' }}>{term.text}</p><a href={submissionApi.fileURL(projectId, c.submission.id, term.page)} target="_blank" rel="noreferrer">Open source</a></details>) : <span>{kind === 'exclusions and qualifications' ? 'No explicit exclusions or qualifications identified. This does not confirm full coverage.' : 'No labelled source section identified.'}</span>}</td>;
        })}</tr>)}
        <tr><th scope="row">Conditional fees and alternatives<span>Check before selecting the appointment</span></th>{table.columns.map(c => <td key={c.submission.id}><ul className="quoted-fees">{c.quote?.fees.filter(f => f.role.includes("Conditional") || f.suggestion?.startsWith("Conditional")).map(f => <li key={f.id}><strong>{f.amount ? money(f.amount) : f.printed}</strong> {f.label}<small>{f.source}</small><a href={submissionApi.fileURL(projectId, c.submission.id, f.page)} target="_blank" rel="noreferrer">Open source</a></li>)}</ul></td>)}</tr>
        <tr className="prices-group"><th scope="rowgroup" colSpan={table.columns.length + 1}>Scope coverage — a bundle may cover several services</th></tr>
        {rows.map(row => <tr key={row.key}><th scope="row">{row.label.replace(/^[A-Z]\d+\s+/, "")}<span>{row.base ? "Requested scope" : "Optional / later scope"}</span></th>{table.columns.map(c => {
          const coverage = c.quote?.coverage[row.key];
          const fee = c.quote?.fees.find(f => f.id === coverage?.fee_id);
          const cell = c.cells[row.key];
          const user = cell?.decided_by === "user";
          return <td key={c.submission.id}>{user ? <><strong>{cell.state === "included" ? "Included — confirmed by you" : cell.state === "excluded" ? "Excluded — confirmed by you" : cell.state === "priced" ? "Separately priced — confirmed by you" : "Not separately priced"}</strong><p>{cell.basis}</p></> : fee ? <><span>{coverage?.status.startsWith("Partial") ? "Partly covered by" : "Covered by"} {fee.label}</span><small className="prices-note">{fee.suggestion?.startsWith("Earlier") ? "Earlier source · check revised offer" : "Suggested coverage · check source"}</small><a href={submissionApi.fileURL(projectId, c.submission.id, fee.page)} target="_blank" rel="noreferrer">{fee.page > 0 ? `Page ${fee.page}` : "Open source"}</a></> : <span>{coverage?.status || "Coverage unclear"}</span>}
            {!user && fee && !coverage?.status.startsWith("Partial") && !fee.suggestion?.startsWith("Earlier") && <button className="table-action" disabled={locked} onClick={() => void onCoverage(c.submission.id, row.key, "included", `Included in ${fee.label}; source page ${fee.page}. ${fee.source}`.slice(0, 500))}>Confirm coverage</button>}
          </td>;
        })}</tr>)}
      </tbody>
    </table></div>
    <button className="table-action" aria-expanded={later} onClick={() => setLater(!later)}>{later ? "Hide optional and later scope" : "Show optional and later scope"}</button>
    {editing && <form className="quote-confirm" onSubmit={async e => { e.preventDefault(); if (await onConfirm(editing, amount, note)) { setEditing(""); if (awardAfterConfirm) onAward(editing); } }}>
      <h2>Confirm comparison fee · {table.columns.find(c => c.submission.id === editing)?.firm?.name || "Proposal"}</h2>
      <p>Record the selected appointment, including any alternatives or exclusions. Do not add a package total to its breakdown.</p>
      <label>Amount excluding GST<input autoFocus inputMode="decimal" required value={amount} onChange={e => setAmount(e.target.value)} /></label>
      <label>Scope and source<textarea required maxLength={500} value={note} onChange={e => setNote(e.target.value)} /></label>
      <button className="btn btn-primary" disabled={busy || !note.trim()}>{awardAfterConfirm ? "Continue to award" : "Confirm fee"}</button>
      <button type="button" className="btn" disabled={busy} onClick={() => setEditing("")}>Cancel</button>
    </form>}
  </div>;
}

function AddedFee({ cell, label, busy, onSave }: { cell?: Cell; label: string; busy: boolean; onSave: (state: CellState, amount: string, note: string) => Promise<boolean> }) {
  const [open, setOpen] = useState(false);
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  const [state, setState] = useState<CellState>("priced");
  if (!open) return <button className="prices-cell" disabled={busy} aria-label={`Edit ${label}`} onClick={() => { setAmount(cell?.amount ?? ""); setNote(cell?.basis ?? ""); setState(cell?.state || "priced"); setOpen(true); }}>{cell?.state === "included" ? "Included" : cell?.amount != null ? money(cell.amount) : "Enter amount"}</button>;
  return <form className="prices-editor" onSubmit={async e => { e.preventDefault(); if (await onSave(state, amount.replace(/[,$\s]/g, ""), note)) setOpen(false); }}>
    <select aria-label="Fee treatment" value={state} disabled={busy} onChange={e => setState(e.target.value as CellState)}><option value="priced">Additional amount</option><option value="included">Included in existing fee</option><option value="not_priced">Not priced</option></select>
    {state === "priced" && <input autoFocus aria-label="Amount ex GST" inputMode="decimal" required value={amount} disabled={busy} onChange={e => setAmount(e.target.value)} />}
    <input aria-label="Agreement note" required maxLength={500} placeholder="Agreed scope or allowance" value={note} disabled={busy} onChange={e => setNote(e.target.value)} />
    <button className="btn btn-small" disabled={busy}>Save fee</button><button type="button" className="table-action" disabled={busy} onClick={() => setOpen(false)}>Cancel</button>
  </form>;
}
