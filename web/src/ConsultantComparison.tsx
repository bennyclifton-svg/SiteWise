import { useLayoutEffect, useRef, useState } from "react";
import { fmt } from "./CostGrid";
import { submissionApi, type AppointmentRow, type Column, type PriceTable, type CellState } from "./submissionApi";

const money = (value: string | null | undefined) => value == null ? "Not priced" : `$${fmt(value)}`;

export function ConsultantComparison({ table, projectId, busy, onOpen, onAward, onSave, onSelect, onAdd }: {
  table: PriceTable; projectId: string; busy: boolean; onAdd: () => void;
  onOpen: (sub: string) => void; onAward: (sub: string) => void;
  onSave: (sub: string, key: string, state: CellState, amount: string, note: string) => Promise<boolean>;
  onSelect: (col: Column, row: AppointmentRow, included: boolean, keys?: string[]) => Promise<boolean>;
}) {
  const slots = Array.from({ length: Math.max(0, 3 - table.columns.length) }, (_, i) => table.columns.length + i + 1);
  const columnCount = 1 + table.columns.length + slots.length;
  const emptyCells = () => slots.map(slot => <td key={slot} className="comparison-placeholder" />);
  const locked = busy || !!table.award;
  const arrows = useRef(new Map<string, HTMLButtonElement>());
  const moved = useRef("");
  useLayoutEffect(() => {
    if (moved.current) { arrows.current.get(moved.current)?.focus(); moved.current = ""; }
  }, [table]);
  // Source positions in different proposals do not imply matching scope.
  const identities = [...new Set(table.columns.flatMap(c => (c.appointment_rows ?? []).map(r => rowKey(c, r))))];
  function rowKey(c: Column, r: AppointmentRow) { return r.id === "fee" ? "fee" : r.keys.length === 1 ? `scope:${r.keys[0]}` : `${c.submission.id}:${r.id}`; }
  function schedule(included: boolean) {
    return identities.map(key => {
      const matches = table.columns.map(c => (c.appointment_rows ?? []).find(r => rowKey(c, r) === key && r.included === included));
      const row = matches.find(Boolean);
      if (!row) return null;
      return <tr key={key} className="appointment-row">
        <th scope="row">{row.label}</th>
        {table.columns.map((col, index) => {
          const r = matches[index];
          return <td key={col.submission.id} className="appointment-cell">{r ? <>
              <button ref={el => { const key = `${col.submission.id}:${r.id}`; if (el) arrows.current.set(key, el); else arrows.current.delete(key); }} className="price-icon appointment-move" disabled={locked} aria-label={`${included ? "Exclude" : "Include"} ${r.label}, ${col.firm?.name ?? col.submission.filename}`} title={included ? "Move below the line" : "Move into appointment"} onClick={async () => { moved.current = `${col.submission.id}:${r.id}`; if (!await onSelect(col, r, !included)) moved.current = ""; }}>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true"><path d={included ? "M12 4v16m-6-6 6 6 6-6" : "M12 20V4m-6 6 6-6 6 6"} /></svg>
              </button>
            <details className="appointment-detail"><summary title={`Fee details for ${r.label}`}>
              <strong>{r.state === "included" ? "Included" : r.amount == null && r.printed ? `${r.printed} printed` : money(r.amount)}</strong>
              <svg className="appointment-chevron" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true"><path d="m9 6 6 6-6 6" /></svg>
              <span className="prices-sr-only">Fee details for {r.label}, {col.firm?.name ?? col.submission.filename}</span>
            </summary>
              {r.amount == null && r.state !== "included" && (r.printed || included) && <small className="prices-note">{r.printed ? "GST basis unclear; not added" : "Confirm a fee or mark included before award"}</small>}
              {r.alternative && <small className="prices-note">Alternative offer</small>}
              {r.mapping_suggested && <p>Check mapping</p>}
              {r.keys.length > 1 && <p>Covers {r.keys.length} stages</p>}
              {r.source_label && <p>Quoted as: {r.source_label}</p>}
              {r.source && <p className="appointment-source">{r.source}</p>}
              {r.keys.length > 1 && <ul>{r.keys.map(k => <li key={k}>{table.items.find(i => i.key === k)?.label}</li>)}</ul>}
              <a href={submissionApi.fileURL(projectId, col.submission.id, r.page)} target="_blank" rel="noreferrer">Open source</a>
              {(r.quote_id || r.id === "fee") && <Mapping key={`${r.id}:${r.keys.join()}`} row={r} table={table} busy={locked} onSave={keys => onSelect(col, r, r.included, keys)} />}
              {r.item_key && <FeeEditor row={r} busy={locked} onSave={(state, amount, note) => onSave(col.submission.id, r.item_key!, state, amount, note)} />}
            </details>
          </> : <span className="prices-note">—</span>}</td>;
        })}
        {emptyCells()}
      </tr>;
    });
  }
  return <div className={`consultant-comparison${table.columns.length === 1 ? " appointment-single" : ""}`}>
    <div className="prices-scroll"><table className="prices-table">
      <caption className="prices-sr-only">Appointment scope and fees</caption>
      <colgroup><col className="comparison-scope-col" />{table.columns.map(c => <col key={c.submission.id} />)}{slots.map(slot => <col key={slot} className="comparison-placeholder-col" />)}</colgroup>
      <thead><tr><th scope="col">Scope</th>{table.columns.map(c => <th scope="col" key={c.submission.id}><button type="button" className="comparison-firm" title="Open the proposal" onClick={() => onOpen(c.submission.id)}>{c.firm?.name || c.submission.filename}</button></th>)}{slots.map(slot => <th key={slot} scope="col" className="comparison-placeholder"><button className="comparison-add" disabled={busy || !!table.award} onClick={onAdd} aria-label={`Add consultant ${slot}`}><span>Consultant {slot}</span><span>+ Add consultant</span></button></th>)}</tr></thead>
      <tbody>
        <tr className="prices-group"><th scope="rowgroup" colSpan={columnCount}>Included in appointment</th></tr>
        {schedule(true)}
        <tr className="prices-total"><th scope="row">Total <span>excluding GST</span></th>{table.columns.map(c => <td key={c.submission.id}><span className="comparison-total">
          <strong>{c.appointment_total != null ? money(c.appointment_total) : "Not confirmed"}</strong>
          {/* A reconciled total needs no note; only the unprinted case is worth a word. */}
          {c.appointment_total != null && c.appointment_basis && c.appointment_basis !== "reconciled" && <small className="prices-note" title="No total printed; sum of the listed fees">Sum of listed fees</small>}
          {!table.award && <button className="btn btn-primary btn-small comparison-award" aria-label={`Award to ${c.firm?.name || "this consultant"}`} disabled={locked || c.appointment_total == null || !c.appointment_rows?.some(r => r.included && r.state === "priced" && r.amount != null)} onClick={() => onAward(c.submission.id)}>Award</button>}
        </span></td>)}{emptyCells()}</tr>
        {schedule(false)}
      </tbody>
    </table></div>
    <div className="appointment-terms">
      {([['scope', 'Scope'], ['exclusions and qualifications', 'Exclusions'], ['conditions', 'Conditions']] as const).map(([kind, label]) => <details key={kind}><summary>{label}</summary>
        {table.columns.map(c => <div key={c.submission.id}>
          {table.columns.length > 1 && <h3>{c.firm?.name || c.submission.filename}</h3>}
          {c.quote?.terms?.some(t => t.kind === kind) ? c.quote.terms.filter(t => t.kind === kind).map((t, i) => <div key={i}><h4>{t.heading}</h4><p className="appointment-source">{t.text}</p><a href={submissionApi.fileURL(projectId, c.submission.id, t.page)} target="_blank" rel="noreferrer">Open source</a></div>) : <p>No explicit {label.toLowerCase()} identified in the source.</p>}
        </div>)}
      </details>)}
      {table.columns.some(c => c.quote?.issues.length) && <details><summary>Reading notes</summary>{table.columns.map(c => c.quote?.issues.map(i => <p key={`${c.submission.id}:${i}`}>{i}</p>))}</details>}
    </div>
  </div>;
}

function Mapping({ row, table, busy, onSave }: { row: AppointmentRow; table: PriceTable; busy: boolean; onSave: (keys: string[]) => Promise<boolean> }) {
  const [keys, setKeys] = useState(row.keys);
  return <form className="appointment-mapping" onSubmit={e => { e.preventDefault(); void onSave(keys); }}>
    <fieldset disabled={busy}><legend>Covered stages</legend>{table.items.filter(i => !i.retired && i.kind !== "rate" && i.kind !== "criterion" && !i.key.startsWith("comparison:")).map(i => <label key={i.key}><input type="checkbox" checked={keys.includes(i.key)} onChange={e => setKeys(e.target.checked ? [...keys, i.key] : keys.filter(k => k !== i.key))} />{i.label}</label>)}</fieldset>
    <button className="btn btn-small" disabled={busy}>Save mapping</button>
  </form>;
}

function FeeEditor({ row, busy, onSave }: { row: AppointmentRow; busy: boolean; onSave: (state: CellState, amount: string, note: string) => Promise<boolean> }) {
  const [amount, setAmount] = useState(row.amount ?? "");
  const [note, setNote] = useState("");
  const [state, setState] = useState<CellState>(row.state === "included" ? "included" : "priced");
  return <form className="prices-editor" onSubmit={async e => { e.preventDefault(); if (await onSave(state, amount.replace(/[,$\s]/g, ""), note)) setNote(""); }}>
    <label>Fee treatment<select value={state} disabled={busy} onChange={e => setState(e.target.value as CellState)}><option value="priced">Agreed amount</option><option value="included">Included in existing fee</option><option value="not_priced">Not priced</option></select></label>
    {state === "priced" && <label>Amount ex GST<input inputMode="decimal" required value={amount} disabled={busy} onChange={e => setAmount(e.target.value)} /></label>}
    <label>Agreement note<input required maxLength={500} value={note} disabled={busy} onChange={e => setNote(e.target.value)} /></label>
    <button className="btn btn-small" disabled={busy}>Save fee</button>
  </form>;
}
