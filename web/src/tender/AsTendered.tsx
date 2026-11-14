// "As tendered": the schedule down the left, the firms across, a price in
// each cell and each firm's tender sum at the foot. A column always adds up
// to the figure the firm printed; what the firm did not break down sits in
// "Not broken down". Every figure arrives computed by the server.
import { Fragment, useState, type ReactNode } from "react";
import { Pending } from "../Activity";
import type { Column, PriceTable } from "../submissionApi";
import { LinesPanel, linesOf } from "./LinesPanel";
import type { TenderColumn, TenderOp, TenderRow, TenderTable } from "./tenderApi";
import { tenderDrag, type TenderEdits } from "./useTenderEdits";
import { basisWords, cellWords, columnState, dollars, markWords, remainderWords } from "./words";

const OLD_WORDS: Record<string, string> = { not_priced: "Not priced", included: "Included", excluded: "Excluded", "": "—" };

interface Props {
  table: PriceTable;
  tender: TenderTable;
  edits: TenderEdits;
  /** The firm's head as the old table renders it: name, date, status, upload, Award. */
  head: (c: Column, t: TenderColumn | undefined) => ReactNode;
  onOpenPage: (c: Column, page: number) => void;
}

export function AsTendered({ table, tender, edits, head, onOpenPage }: Props) {
  const [open, setOpen] = useState<{ sub: string; row: string } | null>(null);
  const [sumEdit, setSumEdit] = useState("");
  const cols = table.columns;
  const tcol = (c: Column) => tender.columns.find(t => t.submission_id === c.submission.id);
  const rows = tender.rows;
  const body = rows.filter(r => !r.system);
  const system = rows.find(r => r.system);
  const firm = (c: Column) => c.firm?.name ?? c.submission.filename.replace(/\.pdf$/i, "");
  const editable = (c: Column) => !table.award && c.submission.status !== "awarded" && c.submission.status !== "unsuccessful";
  const busy = edits.busy;

  // A line, or every counted line of a cell, dropped on another row of the
  // same firm moves there as one batch. Another firm's column refuses it.
  function drop(c: Column, row: TenderRow) {
    const d = tenderDrag.current;
    tenderDrag.current = null;
    if (!d || d.sub !== c.submission.id) return;
    void edits.run(d.lines.map((line): TenderOp => ({ op: "move_line", line, row: row.system ? "" : row.key })));
  }

  function cell(c: Column, t: TenderColumn | undefined, row: TenderRow, first: boolean): ReactNode {
    if (!t) {
      // A submission read the old way, in the same package.
      const old = c.cells[row.key];
      return <td key={c.submission.id}><span className={old?.state === "priced" ? "prices-figure" : "prices-word"}>{old?.state === "priced" ? (old.amount == null ? "Priced, amount unknown" : dollars(old.amount)) : OLD_WORDS[old?.state ?? ""]}</span></td>;
    }
    if (t.state !== "read") {
      // The firm's cells wait for its reading: the first says so, the rest are blank.
      return <td key={c.submission.id} className="tender-waiting">
        {first && (t.state === "failed" ? <><span className="prices-note warn">Prices not read</span><span className="prices-note">{columnState(t)}</span></> : <Pending>{columnState(t)}</Pending>)}
      </td>;
    }
    const tc = t.cells[row.key];
    const lines = linesOf(t, row, rows);
    const isOpen = open?.sub === c.submission.id && open.row === row.key;
    const counted = lines.filter(l => l.counted).map(l => l.id);
    return <td key={c.submission.id} data-band={tc?.to_check ? "red" : undefined} className={isOpen ? "tender-cell-open" : undefined}
      onDragOver={e => { if (tenderDrag.current?.sub === c.submission.id && editable(c)) e.preventDefault(); }}
      onDrop={e => { e.preventDefault(); if (editable(c)) drop(c, row); }}>
      <button type="button" className="prices-cell tender-cell" aria-expanded={isOpen} disabled={lines.length === 0 && !editable(c)}
        aria-label={`${row.label}, ${firm(c)}: ${cellWords(tc)}${tc?.mark ? `, ${markWords(tc.mark)}` : ""}. Show lines`}
        draggable={editable(c) && counted.length > 0 && !busy}
        onDragStart={e => { tenderDrag.current = { sub: c.submission.id, lines: counted }; e.dataTransfer.effectAllowed = "move"; e.dataTransfer.setData("text/plain", row.label); }}
        onDragEnd={() => { tenderDrag.current = null; }}
        onClick={() => setOpen(isOpen ? null : { sub: c.submission.id, row: row.key })}>
        <span className={tc?.state === "priced" && tc.amount != null ? "prices-figure" : "prices-word"}>{cellWords(tc)}</span>
        {tc?.mark && <span className="tender-mark" data-mark={tc.mark}>{markWords(tc.mark)}</span>}
        {tc?.to_check && <span className="tender-dot" title="Something here needs a look" aria-label="Something here needs a look" />}
      </button>
      {row.system && remainderWords(t.remainder_why) && <span className="prices-note">{remainderWords(t.remainder_why)}</span>}
    </td>;
  }

  function panel(row: TenderRow): ReactNode {
    const c = cols.find(x => x.submission.id === open?.sub);
    const t = c && tcol(c);
    if (!c || !t || open?.row !== row.key) return null;
    return <tr className="tender-lines-row"><td colSpan={cols.length + 1}>
      <LinesPanel col={t} firm={firm(c)} row={row} rows={rows} busy={busy} editable={editable(c)} run={edits.run} onOpenPage={page => onOpenPage(c, page)} />
    </td></tr>;
  }

  return <div className="prices-scroll">
    <table className="prices-table tender-table">
      <caption>As tendered, ex GST</caption>
      <colgroup><col className="prices-label-col" />{cols.map(c => <col key={c.submission.id} />)}</colgroup>
      <thead><tr>
        <th scope="col" className="prices-item">Item</th>
        {cols.map(c => {
          const t = tcol(c);
          return <th scope="col" key={c.submission.id} className={t?.lowest ? "prices-firm lowest" : "prices-firm"}>
            {head(c, t)}
            {!t && <span className="prices-firm-meta">Read the old way
              {tender.on && c.submission.document_id && (c.submission.status === "received" || c.submission.status === "needs_review") &&
                <button type="button" className="table-action" disabled={busy} onClick={() => void edits.readAgain(c.submission.id)}>Read prices</button>}</span>}
            {t && <span className="tender-state" data-state={t.state === "read" ? (t.checked ? "checked" : "to-check") : t.state}>{t.state === "failed" ? "Prices not read" : columnState(t)}</span>}
            {t?.state === "read" && editable(c) && <button type="button" className="btn btn-small tender-check" disabled={busy} aria-pressed={t.checked}
              aria-label={t.checked ? `Checked: ${firm(c)}. Un-check` : `Mark ${firm(c)} checked`}
              onClick={() => void edits.run([{ op: "check_column", submission: c.submission.id, checked: !t.checked }])}>{t.checked ? "Checked" : "Mark checked"}</button>}
            {t?.state === "read" && tender.on && editable(c) && <button type="button" className="table-action" disabled={busy} onClick={() => void edits.readAgain(c.submission.id)}>Read again</button>}
            {t?.state === "failed" && tender.on && <button type="button" className="btn btn-small" disabled={busy} onClick={() => void edits.readAgain(c.submission.id)}>Try again</button>}
            {t && basisWords(t.basis) && <span className="prices-note warn">{basisWords(t.basis)}</span>}
          </th>;
        })}
      </tr></thead>
      <tbody>
        {body.map((row, i) => <Fragment key={row.key}>
          <tr><th scope="row" className="prices-item">{row.label}</th>{cols.map(c => cell(c, tcol(c), row, i === 0))}</tr>
          {panel(row)}
        </Fragment>)}
        {system && <>
          <tr className="tender-system"><th scope="row" className="prices-item">{system.label}</th>{cols.map(c => cell(c, tcol(c), system, body.length === 0))}</tr>
          {panel(system)}
        </>}
        <tr className="prices-total">
          <th scope="row">Tender sum</th>
          {cols.map(c => {
            const t = tcol(c);
            if (!t) return <td key={c.submission.id}><span className="prices-figure">{c.totals.base_total != null ? dollars(c.totals.base_total) : "Unknown"}</span></td>;
            if (t.state !== "read") return <td key={c.submission.id} />;
            if (sumEdit === c.submission.id) return <td key={c.submission.id}><SumEditor sum={t.tender_sum} byUser={t.tender_sum_by === "user"} busy={busy} onCancel={() => setSumEdit("")}
              onSave={async (amount, gst) => { if (await edits.run([{ op: "set_tender_sum", submission: c.submission.id, amount, gst }])) setSumEdit(""); }} /></td>;
            return <td key={c.submission.id} className={t.lowest ? "lowest" : undefined} title={t.tender_sum_how} onDoubleClick={() => { if (editable(c)) setSumEdit(c.submission.id); }}>
              <span className="prices-figure" aria-label={`Tender sum, ${firm(c)}`}>{t.tender_sum == null ? "Unknown" : dollars(t.tender_sum)}</span>
              {t.tender_sum_by === "user" && <span className="prices-note">set by you</span>}
              {t.tender_sum_how && t.tender_sum_by !== "user" && <span className="prices-note">{t.tender_sum_how}</span>}
              {editable(c) && <button type="button" className="table-action" disabled={busy} aria-label={`Correct the tender sum, ${firm(c)}`} onClick={() => setSumEdit(c.submission.id)}>Correct</button>}
            </td>;
          })}
        </tr>
        {tender.columns.some(t => t.flags.length + t.notes.length + t.exclusions.length > 0) && <tr className="prices-flags">
          <th scope="row">To look at</th>
          {cols.map(c => {
            const t = tcol(c);
            return <td key={c.submission.id}>
              {t?.flags.map(f => <p key={f.code + f.text}>{f.text}</p>)}
              {t && t.notes.length + t.exclusions.length > 0 && <details><summary>The firm says</summary>
                {t.notes.map((n, i) => <p key={`n${i}`}>{n}</p>)}
                {t.exclusions.map((x, i) => <p key={`x${i}`}>Excludes: {x.text}{x.p > 0 ? ` (p. ${x.p})` : ""}</p>)}
              </details>}
            </td>;
          })}
        </tr>}
      </tbody>
    </table>
  </div>;
}

// Correcting the tender sum is its own act: it never moves a line. An empty
// amount gives the sum the firm printed back.
function SumEditor({ sum, byUser, busy, onSave, onCancel }: { sum: string | null; byUser: boolean; busy: boolean; onSave: (amount: string, gst: string) => void; onCancel: () => void }) {
  const [amount, setAmount] = useState(sum ?? "");
  const [gst, setGst] = useState("ex");
  const save = () => onSave(amount.replace(/[,$\s]/g, ""), gst);
  return <div className="tender-editor" role="group" aria-label="Correct the tender sum" onKeyDown={e => { if (e.key === "Enter") { e.preventDefault(); save(); } if (e.key === "Escape") { e.stopPropagation(); onCancel(); } }}>
    <input aria-label="Tender sum" inputMode="decimal" autoFocus value={amount} disabled={busy} onChange={e => setAmount(e.target.value)} />
    <select aria-label="Tender sum GST" value={gst} disabled={busy} onChange={e => setGst(e.target.value)}><option value="ex">ex GST</option><option value="inc">inc GST</option></select>
    <button type="button" className="btn btn-small" disabled={busy || !amount.trim()} onClick={save}>Save</button>
    {byUser && <button type="button" className="table-action" disabled={busy} onClick={() => onSave("", "ex")}>Use the firm's printed sum</button>}
    <button type="button" className="table-action" onClick={onCancel}>Cancel</button>
  </div>;
}
