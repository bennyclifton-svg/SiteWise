// "As tendered": the schedule down the left, the firms across, a price in
// each cell and each firm's tender sum at the foot. A column always adds up
// to the figure the firm printed; what the firm did not break down sits in
// "Not broken down". Every figure arrives computed by the server.
import { Fragment, use, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { Pending } from "../Activity";
import { Chevron } from "../Chevron";
import type { Column, PriceTable } from "../submissionApi";
import { TenderCents } from "./cents";
import { spareCells, spareCols, spareFirms, spareHeads } from "./spare";
import { LinesPanel, linesOf } from "./LinesPanel";
import type { TenderColumn, TenderOp, TenderRow, TenderTable } from "./tenderApi";
import { tenderDrag, type TenderEdits } from "./useTenderEdits";
import { cellWords, columnState, dollars, markWords, remainderWords } from "./words";

const OLD_WORDS: Record<string, string> = { not_priced: "Not priced", included: "Included", excluded: "Excluded", "": "—" };

interface Props {
  table: PriceTable;
  tender: TenderTable;
  edits: TenderEdits;
  /** The firm's head as the old table renders it: name, status, upload, detach. */
  head: (c: Column, t: TenderColumn | undefined) => ReactNode;
  /** The Award button, shown beside the firm's tender sum; null when it cannot be awarded. */
  award: (c: Column, t: TenderColumn) => ReactNode;
  onOpenPage: (c: Column, page: number) => void;
}

export function AsTendered({ table, tender, edits, head, award, onOpenPage }: Props) {
  // Open rows, by key. Opening one leaves the others open (owner, 15 Nov 2026).
  const [open, setOpen] = useState<ReadonlySet<string>>(() => new Set());
  const toggle = (key: string) => setOpen(o => { const n = new Set(o); if (n.has(key)) n.delete(key); else n.add(key); return n; });
  const [sumEdit, setSumEdit] = useState("");
  const cols = table.columns;
  const spare = spareFirms(cols.length);
  const tcol = (c: Column) => tender.columns.find(t => t.submission_id === c.submission.id);
  const rows = tender.rows;
  const body = rows.filter(r => !r.system);
  const system = rows.find(r => r.system);
  const firm = (c: Column) => c.firm?.name ?? c.submission.filename.replace(/\.pdf$/i, "");
  const editable = (c: Column) => !table.award && c.submission.status !== "awarded" && c.submission.status !== "unsuccessful";
  const busy = edits.busy;
  const cents = use(TenderCents);
  // The open row a line is being dragged in gets one more, empty slot at
  // its foot, so a line can be moved below every other.
  const [dragRow, setDragRow] = useState<string | null>(null);
  // The firm whose column is being dragged by its grip.
  const colDrag = useRef("");
  // A combined engagement's rows are headed by discipline, as in the old table.
  const groupHead = (key: string) => !table.group ? "" : key.startsWith("combined:") ? "Combined stages" : table.group.members.find(m => key.startsWith(`${m.key}:`))?.title ?? "";

  // A line, or every counted line of a cell, dropped on another row of the
  // same firm moves there as one batch. Another firm's column refuses it.
  function drop(c: Column, row: TenderRow) {
    const d = tenderDrag.current;
    tenderDrag.current = null;
    setDragRow(null);
    if (!d || d.sub !== c.submission.id || d.counted === false || d.row === (row.system ? "" : row.key)) return;
    void edits.run(d.lines.map((line): TenderOp => ({ op: "move_line", line, row: row.system ? "" : row.key })));
  }

  // The slots of an open row: every place any firm's line holds, top to
  // bottom, so each firm's column lines up with the others.
  function slotsOf(row: TenderRow): number[] {
    const used = new Set<number>();
    for (const c of cols) {
      const t = tcol(c);
      if (t?.state === "read") for (const l of linesOf(t, row, rows)) if (l.slot >= 0) used.add(l.slot);
    }
    const out = [...used].sort((a, b) => a - b);
    if (dragRow === (row.system ? "" : row.key)) out.push(out.length ? out[out.length - 1] + 1 : 0);
    return out;
  }

  // Moving a line saves every line's place in the row, for every firm, so
  // the manager's layout holds from then on (owner, 15 Nov 2026). A line
  // dropped on a place its firm already uses swaps with that line.
  function place(row: TenderRow, sub: string, line: string, from: number, to: number) {
    const layout = new Map<string, number>();
    let next = Math.max(0, ...slotsOf(row)) + 1;
    for (const c of cols) {
      const t = tcol(c);
      if (t?.state !== "read") continue;
      for (const l of linesOf(t, row, rows)) layout.set(l.id, l.slot >= 0 ? l.slot : next++);
      if (c.submission.id === sub) {
        const there = linesOf(t, row, rows).find(l => l.id !== line && layout.get(l.id) === to);
        if (there) layout.set(there.id, from);
      }
    }
    layout.set(line, to);
    void edits.run([...layout].map(([id, slot]): TenderOp => ({ op: "set_slot", line: id, slot })));
  }

  // Dragging a firm's head by its grip, or Alt+Left and Alt+Right on the
  // grip, puts the firm in another column for everyone (owner, 15 Nov 2026).
  function moveColumn(sub: string, to: number) {
    const order = cols.map(c => c.submission.id).filter(id => id !== sub);
    order.splice(Math.max(0, Math.min(to, order.length)), 0, sub);
    if (order.join() !== cols.map(c => c.submission.id).join()) void edits.run([{ op: "order_columns", order }]);
  }
  function gripKeys(e: KeyboardEvent, sub: string) {
    const at = cols.findIndex(c => c.submission.id === sub);
    if (e.key === "ArrowLeft" || e.key === "ArrowRight") { e.preventDefault(); moveColumn(sub, at + (e.key === "ArrowLeft" ? -1 : 1)); }
  }

  function cell(c: Column, t: TenderColumn | undefined, row: TenderRow, first: boolean): ReactNode {
    if (!t) {
      // A submission read the old way, in the same package.
      const old = c.cells[row.key];
      return <td key={c.submission.id}><span className={old?.state === "priced" ? "prices-figure" : "prices-word"}>{old?.state === "priced" ? (old.amount == null ? "Priced, amount unknown" : dollars(old.amount, cents)) : OLD_WORDS[old?.state ?? ""]}</span></td>;
    }
    if (t.state !== "read") {
      // The firm's cells wait for its reading: the first says so, the rest are blank.
      return <td key={c.submission.id} className="tender-waiting">
        {first && (t.state === "failed" ? <><span className="prices-note warn">Prices not read</span><span className="prices-note">{columnState(t)}</span></> : <Pending>{columnState(t)}</Pending>)}
      </td>;
    }
    const tc = t.cells[row.key];
    const lines = linesOf(t, row, rows);
    const isOpen = open.has(row.key);
    const counted = lines.filter(l => l.counted).map(l => l.id);
    // No orange edge in the tender tables (owner, 15 Nov 2026): the dot says it.
    return <td key={c.submission.id}
      onDragOver={e => { if (tenderDrag.current?.sub === c.submission.id && editable(c)) e.preventDefault(); }}
      onDrop={e => { e.preventDefault(); if (editable(c)) drop(c, row); }}>
      <button type="button" className="prices-cell tender-cell" aria-expanded={isOpen} disabled={lines.length === 0 && !editable(c)}
        aria-label={`${row.label}, ${firm(c)}: ${cellWords(tc, cents)}${tc && markWords(tc.mark) ? `, ${markWords(tc.mark)}` : ""}. Show lines`}
        draggable={editable(c) && counted.length > 0 && !busy}
        onDragStart={e => { tenderDrag.current = { sub: c.submission.id, lines: counted }; e.dataTransfer.effectAllowed = "move"; e.dataTransfer.setData("text/plain", row.label); }}
        onDragEnd={() => { tenderDrag.current = null; }}
        onClick={() => toggle(row.key)}>
        <span className={tc?.state === "priced" && tc.amount != null ? "prices-figure" : "prices-word"}>{cellWords(tc, cents)}</span>
        {tc && markWords(tc.mark) && <span className="tender-mark" data-mark={tc.mark}>{markWords(tc.mark)}</span>}
        {tc?.to_check && <span className="tender-dot" title="Something here needs a look" aria-label="Something here needs a look" />}
      </button>
      {row.system && remainderWords(t.remainder_why) && <span className="prices-note">{remainderWords(t.remainder_why)}</span>}
    </td>;
  }

  // An open row shows every firm's own lines at once, each in its own column.
  function panel(row: TenderRow): ReactNode {
    if (!open.has(row.key)) return null;
    return <tr className="tender-lines-row"><td />
      {cols.map(c => {
        const t = tcol(c);
        return <td key={c.submission.id}>{t?.state === "read" && <LinesPanel col={t} firm={firm(c)} row={row} rows={rows} slots={slotsOf(row)} busy={busy} editable={editable(c)} run={edits.run}
          onPlace={(line, from, to) => place(row, c.submission.id, line, from, to)} onDragging={on => setDragRow(on ? (row.system ? "" : row.key) : null)} onOpenPage={page => onOpenPage(c, page)} />}</td>;
      })}
      {spareCells(spare)}
    </tr>;
  }

  // The row's label with its chevron: a click opens or closes the row.
  function rowHead(row: TenderRow): ReactNode {
    const isOpen = open.has(row.key);
    return <th scope="row" className="prices-item">
      <button type="button" className="tender-row-toggle" aria-expanded={isOpen} aria-label={`${isOpen ? "Hide" : "Show"} lines for ${row.label}`} onClick={() => toggle(row.key)}>
        <Chevron open={isOpen} />
        <span className="tender-row-label">{row.label}</span>
      </button>
    </th>;
  }

  return <div className="prices-scroll">
    <table className="prices-table tender-table">
      <caption>As tendered, ex GST</caption>
      <colgroup><col className="prices-label-col" />{cols.map(c => <col key={c.submission.id} />)}{spareCols(spare)}</colgroup>
      <thead><tr>
        <th scope="col" className="prices-item">Item</th>
        {cols.map(c => {
          const t = tcol(c);
          return <th scope="col" key={c.submission.id} className={t?.lowest ? "prices-firm lowest" : "prices-firm"}
            onDragOver={e => { if (colDrag.current && colDrag.current !== c.submission.id) e.preventDefault(); }}
            onDrop={e => {
              const from = colDrag.current;
              colDrag.current = "";
              if (!from || from === c.submission.id) return;
              e.preventDefault();
              moveColumn(from, cols.findIndex(x => x.submission.id === c.submission.id));
            }}>
            {/* A span, not a <button>: browsers will not start a drag from a button. */}
            {cols.length > 1 && <span className="tender-col-grip" role="button" tabIndex={0} draggable={!busy} aria-disabled={busy || undefined}
              aria-label={`Move ${firm(c)}: Alt and the left or right arrow`} title="Drag to move this firm left or right"
              onKeyDown={e => { if (e.altKey && !busy) gripKeys(e, c.submission.id); }}
              onDragStart={e => { colDrag.current = c.submission.id; e.dataTransfer.effectAllowed = "move"; e.dataTransfer.setData("text/plain", firm(c)); }}
              onDragEnd={() => { colDrag.current = ""; }}><span className="table-row-handle" aria-hidden="true" /></span>}
            {head(c, t)}
            {!t && <span className="prices-firm-meta">Read the old way</span>}
            {t && t.state !== "read" && <span className="tender-state" data-state={t.state}>{t.state === "failed" ? "Prices not read" : columnState(t)}</span>}
            {t?.state === "failed" && tender.on && <button type="button" className="btn btn-small" disabled={busy} onClick={() => void edits.readAgain(c.submission.id)}>Try again</button>}
          </th>;
        })}
        {spareHeads(spare)}
      </tr></thead>
      <tbody>
        {body.map((row, i) => <Fragment key={row.key}>
          {groupHead(row.key) && groupHead(row.key) !== (i > 0 ? groupHead(body[i - 1].key) : "") && <tr className="prices-group"><th colSpan={cols.length + spare.length + 1} scope="rowgroup">{groupHead(row.key)}</th></tr>}
          <tr className={open.has(row.key) ? "tender-row-open" : undefined}>{rowHead(row)}{cols.map(c => cell(c, tcol(c), row, i === 0))}{spareCells(spare)}</tr>
          {panel(row)}
        </Fragment>)}
        {system && <>
          <tr className={open.has(system.key) ? "tender-system tender-row-open" : "tender-system"}>{rowHead(system)}{cols.map(c => cell(c, tcol(c), system, body.length === 0))}{spareCells(spare)}</tr>
          {panel(system)}
        </>}
        <tr className="prices-total">
          <th scope="row">Tender sum</th>
          {cols.map(c => {
            const t = tcol(c);
            if (!t) return <td key={c.submission.id}><span className="prices-figure">{c.totals.base_total != null ? dollars(c.totals.base_total, cents) : "Unknown"}</span></td>;
            if (t.state !== "read") return <td key={c.submission.id} />;
            if (sumEdit === c.submission.id) return <td key={c.submission.id}><SumEditor sum={t.tender_sum} byUser={t.tender_sum_by === "user"} busy={busy} onCancel={() => setSumEdit("")}
              onSave={async (amount, gst) => { if (await edits.run([{ op: "set_tender_sum", submission: c.submission.id, amount, gst }])) setSumEdit(""); }} /></td>;
            return <td key={c.submission.id} className={t.lowest ? "lowest" : undefined} title={t.tender_sum_how} onDoubleClick={() => { if (editable(c)) setSumEdit(c.submission.id); }}>
              {/* Award sits beside the figure it commits to; the figure keeps the right edge. A
                  double-click still corrects a misread sum, without a button in the way. */}
              <span className="tender-sum-line">
                {award(c, t)}
                <span className="prices-figure" aria-label={`Tender sum, ${firm(c)}`}>{t.tender_sum == null ? "Unknown" : dollars(t.tender_sum, cents)}</span>
              </span>
              {t.tender_sum_by === "user" && <span className="prices-note">set by you</span>}
            </td>;
          })}
          {spareCells(spare)}
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
          {spareCells(spare)}
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
