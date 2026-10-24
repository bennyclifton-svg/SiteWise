import { useState } from "react";
import { fmt } from "./CostGrid";
import { submissionApi, type Column, type PriceTable, type QuoteFee } from "./submissionApi";

const money = (value: string | null) => value === null ? "Tax basis unclear" : `$${fmt(value)}`;
const feeRole = (f: QuoteFee) => f.role.startsWith("Alternative") ? "Alternative · includes other work" : f.role.startsWith("Breakdown") ? "Breakdown · included in package" : f.role.startsWith("Conditional") ? "Conditional / optional" : f.role.includes("total") ? "Package total" : "Quoted fee";
const roleNote = (f: QuoteFee) => `${feeRole(f)}${f.suggestion && f.suggestion !== "Current offer" ? ` · Suggested: ${f.suggestion}` : ""}`;
function headlineFees(fees: QuoteFee[]) {
  const offers = fees.filter(f => (f.role.includes("Package") || f.role.includes("Alternative")) && !f.suggestion?.startsWith("Earlier"));
  // A bare repeated figure adds no scope information. Keep the named offer
  // prominent; every occurrence remains available in the source fee lines.
  return offers.filter(f => f.label !== "Quoted fee" || !offers.some(other => other.id !== f.id && (f.amount !== null ? other.amount === f.amount : other.printed === f.printed) && other.label !== "Quoted fee"));
}
const statedNote = (c: Column) => `Firm's stated total, ex GST, as printed in ${c.submission.filename}.`.slice(0, 500);

export function ConsultantComparison({ table, projectId, busy, onConfirm, onCoverage, onOpen, onAward }: {
  table: PriceTable; projectId: string; busy: boolean;
  onConfirm: (sub: string, amount: string, note: string) => Promise<boolean>;
  onCoverage: (sub: string, key: string, state: "included" | "excluded" | "not_priced", note: string) => Promise<boolean>;
  onOpen: (sub: string) => void; onAward: (sub: string) => void;
}) {
  const [editing, setEditing] = useState("");
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  const [later, setLater] = useState(false);
  const locked = busy || !!table.award;
  const rows = table.items.filter(i => i.kind !== "criterion" && i.kind !== "rate" && i.key !== "comparison:appointment" && (i.base || later));
  function edit(col: Column, fee?: QuoteFee, stated?: string | null) {
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
    <p className="prices-legend">Quoted fees retain each firm’s bundles and alternatives. Scope coverage is shown separately. Confirm the same appointment scope before comparing prices.</p>
    <div className="prices-scroll"><table className="prices-table">
      <caption className="prices-sr-only">Quoted fees and scope coverage</caption>
      <thead><tr><th scope="col">Quoted fees <span>ex GST where established</span></th>{table.columns.map(c => <th scope="col" key={c.submission.id}>
        <button className="table-action" onClick={() => onOpen(c.submission.id)}>{c.firm?.name || c.submission.filename}</button>
        <small className="quote-filename">{c.submission.filename}</small>
        <a href={submissionApi.fileURL(projectId, c.submission.id)} target="_blank" rel="noreferrer">Open proposal</a>
      </th>)}</tr></thead>
      <tbody>
        <tr><th scope="row">Quoted packages and totals<span>Alternatives and breakdowns are not added together</span></th>
          {table.columns.map(c => <td key={c.submission.id}>
            {!c.quote ? <p>Source reading not yet available.</p> : <>
              <ul className="quoted-fees">{headlineFees(c.quote.fees).map(f => <li key={f.id}>
                <strong>{f.amount !== null ? money(f.amount) : f.printed}</strong> <span>{f.label}</span>
                {/^sub\s*total$/i.test(f.label) && <small>{f.heading}</small>}
                <small>{roleNote(f)}{f.amount === null ? " · GST basis unclear" : ""}</small>
                <a href={submissionApi.fileURL(projectId, c.submission.id, f.page)} target="_blank" rel="noreferrer">Page {f.page}</a>
                <button className="table-action" disabled={locked || !f.amount} onClick={() => edit(c, f)}>Review for comparison</button>
              </li>)}</ul>
              {!c.quote.fees.some(f => f.role.includes("Package") || f.role.includes("Alternative")) && <p>No single package total established. Review the quoted fee lines below.</p>}
              {c.quote.issues.map(i => <p className="prices-note warn" key={i}>{i}</p>)}
            </>}
          </td>)}
        </tr>
        <tr><th scope="row">Conditions and alternatives<span>Check before selecting the appointment</span></th>{table.columns.map(c => <td key={c.submission.id}><ul className="quoted-fees">{c.quote?.fees.filter(f => f.role.includes("Conditional") || f.suggestion?.startsWith("Conditional")).map(f => <li key={f.id}><strong>{f.amount ? money(f.amount) : f.printed}</strong> {f.label}<small>{f.source}</small><a href={submissionApi.fileURL(projectId, c.submission.id, f.page)} target="_blank" rel="noreferrer">Page {f.page}</a></li>)}</ul></td>)}</tr>
        <tr><th scope="row">Quoted fee lines<span>Source amounts, not an additional total</span></th>{table.columns.map(c => <td key={c.submission.id}>
          <details><summary>{c.quote?.fees.length ?? 0} source fee lines</summary><ul className="quoted-fees">
            {c.quote?.fees.map(f => <li key={f.id}><strong>{f.amount !== null ? money(f.amount) : f.printed}</strong> {f.label}
              <small>{roleNote(f)}</small><p>{f.source}</p>
              <a href={submissionApi.fileURL(projectId, c.submission.id, f.page)} target="_blank" rel="noreferrer">Page {f.page}</a>
              <button className="table-action" disabled={locked || !f.amount} onClick={() => edit(c, f)}>Review for comparison</button>
            </li>)}
          </ul></details>
        </td>)}</tr>
        <tr className="prices-total"><th scope="row">Comparison fee<span>The firm's stated total unless you record another</span></th>{table.columns.map(c => {
          const confirmed = c.cells["comparison:appointment"];
          // The firm's own printed total, read from its proposal. Offered only
          // when it is known to be ex GST: the server stores it as printed.
          const stated = !confirmed?.amount && c.submission.gst_basis === "ex" ? c.totals.stated_total : null;
          const firm = c.firm?.name || "this consultant";
          return <td key={c.submission.id}>
            <strong>{confirmed?.amount ? money(confirmed.amount) : stated ? money(stated) : "Not confirmed"}</strong>
            {confirmed?.basis ? <p>{confirmed.basis}</p> : stated && <p>Stated total in the proposal, ex GST</p>}
            {!table.award && (confirmed?.amount || stated) && <button className="btn btn-primary btn-small" disabled={busy} onClick={() => void awardAt(c, stated)}>Award to {firm}</button>}
            <button className="table-action" disabled={locked} onClick={() => edit(c, undefined, stated)}>{confirmed?.amount || stated ? "Use a different amount" : "Confirm amount and scope"}</button>
          </td>;
        })}</tr>
        <tr className="prices-group"><th scope="rowgroup" colSpan={table.columns.length + 1}>Scope coverage — a bundle may cover several services</th></tr>
        {rows.map(row => <tr key={row.key}><th scope="row">{row.label.replace(/^[A-Z]\d+\s+/, "")}<span>{row.base ? "Requested scope" : "Optional / later scope"}</span></th>{table.columns.map(c => {
          const coverage = c.quote?.coverage[row.key];
          const fee = c.quote?.fees.find(f => f.id === coverage?.fee_id);
          const cell = c.cells[row.key];
          const user = cell?.decided_by === "user";
          return <td key={c.submission.id}>{user ? <><strong>{cell.state === "included" ? "Included — confirmed by you" : cell.state === "excluded" ? "Excluded — confirmed by you" : cell.state === "priced" ? "Separately priced — confirmed by you" : "Not separately priced"}</strong><p>{cell.basis}</p></> : fee ? <><span>Covered by {fee.label}</span><small className="prices-note">{fee.suggestion?.startsWith("Earlier") ? "Earlier source · check revised offer" : "Suggested coverage · check source"}</small><a href={submissionApi.fileURL(projectId, c.submission.id, fee.page)} target="_blank" rel="noreferrer">Page {fee.page}</a></> : <span>{coverage?.status || "Coverage unclear"}</span>}
            {!user && fee && !fee.suggestion?.startsWith("Earlier") && <button className="table-action" disabled={locked} onClick={() => void onCoverage(c.submission.id, row.key, "included", `Included in ${fee.label}; source page ${fee.page}. ${fee.source}`.slice(0, 500))}>Confirm coverage</button>}
          </td>;
        })}</tr>)}
      </tbody>
    </table></div>
    <button className="table-action" aria-expanded={later} onClick={() => setLater(!later)}>{later ? "Hide optional and later scope" : "Show optional and later scope"}</button>
    {editing && <form className="quote-confirm" onSubmit={async e => { e.preventDefault(); if (await onConfirm(editing, amount, note)) setEditing(""); }}>
      <h2>Confirm comparison fee · {table.columns.find(c => c.submission.id === editing)?.firm?.name || "Proposal"}</h2>
      <p>Record the selected appointment, including any alternatives or exclusions. Do not add a package total to its breakdown.</p>
      <label>Amount excluding GST<input autoFocus inputMode="decimal" required value={amount} onChange={e => setAmount(e.target.value)} /></label>
      <label>Scope and source<textarea required maxLength={500} value={note} onChange={e => setNote(e.target.value)} /></label>
      <button className="btn btn-primary" disabled={busy || !note.trim()}>Confirm fee</button>
      <button type="button" className="btn" disabled={busy} onClick={() => setEditing("")}>Cancel</button>
    </form>}
  </div>;
}
