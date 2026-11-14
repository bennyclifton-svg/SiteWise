// "Adds and subs": the short schedule of additions and subtractions that
// levels the tenders. The system suggests; only what the manager applies
// counts. The tender sum plus the applied adds and subs is the like-for-like
// total. The browser never adds: figures, ranges and totals all come from
// the server.
import { useState } from "react";
import type { Column, PriceTable } from "../submissionApi";
import type { TenderAdjustment, TenderAmount, TenderOp, TenderTable } from "./tenderApi";
import { tenderDrag, type TenderEdits } from "./useTenderEdits";
import { dollars, gapWords, signed, sourceWords, suggestedWords } from "./words";

interface Props {
  table: PriceTable;
  tender: TenderTable;
  edits: TenderEdits;
}

export function AddsSubs({ table, tender, edits }: Props) {
  const [choosing, setChoosing] = useState<{ adj: string; sub: string } | null>(null);
  const [typing, setTyping] = useState<{ adj: string; sub: string; gap: boolean } | null>(null);
  const [renaming, setRenaming] = useState("");
  const [noting, setNoting] = useState("");
  const [adding, setAdding] = useState("");
  const cols = table.columns;
  const tcol = (c: Column) => tender.columns.find(t => t.submission_id === c.submission.id);
  const firm = (c: Column) => c.firm?.name ?? c.submission.filename.replace(/\.pdf$/i, "");
  const read = (c: Column) => tcol(c)?.state === "read";
  const busy = edits.busy;
  const locked = !!table.award;
  const run = (op: TenderOp) => edits.run([op]);
  if (!tender.columns.some(t => t.state === "read")) return null;

  function amountCell(a: TenderAdjustment, c: Column) {
    if (!read(c)) return <td key={c.submission.id} />;
    const sub = c.submission.id;
    const m: TenderAmount | undefined = a.amounts[sub];
    const isTyping = typing?.adj === a.id && typing.sub === sub;
    if (isTyping) return <td key={sub}><AmountEditor start={m?.amount ?? ""} busy={busy} onCancel={() => setTyping(null)}
      onSave={async amount => {
        const op: TenderOp = typing.gap ? { op: "apply_adjustment", adjustment: a.id, submission: sub, applied: true, source: "typed", amount } : { op: "set_adjustment_amount", adjustment: a.id, submission: sub, amount: amount === "" ? null : amount };
        if (await run(op)) setTyping(null);
      }} /></td>;
    if (!m) return <td key={sub} className="tender-adj-empty" onDoubleClick={() => { if (!locked) setTyping({ adj: a.id, sub, gap: false }); }}>
      {a.origin === "user" && !locked ? <button type="button" className="table-action" disabled={busy} aria-label={`Type an amount for ${firm(c)} on ${a.label}`} onClick={() => setTyping({ adj: a.id, sub, gap: false })}>·</button> : <span className="prices-word">·</span>}
    </td>;
    if (m.state === "suggested") {
      const gap = m.amount == null;
      const isChoosing = choosing?.adj === a.id && choosing.sub === sub;
      return <td key={sub} className="tender-suggested">
        <span className="prices-word">{gap ? gapWords(a.others[sub]) : signed(m.amount)}</span>
        <span className="tender-tag">suggested</span>
        {!gap && <span className="prices-note">{sourceWords(m)}</span>}
        {!locked && !gap && <button type="button" className="btn btn-small" disabled={busy} aria-label={`Apply ${a.label} for ${firm(c)}`} onClick={() => void run({ op: "apply_adjustment", adjustment: a.id, submission: sub, applied: true })}>Apply</button>}
        {!locked && gap && !isChoosing && <button type="button" className="btn btn-small" disabled={busy} aria-expanded="false" aria-label={`Apply ${a.label} for ${firm(c)}`} onClick={() => setChoosing({ adj: a.id, sub })}>Apply</button>}
        {isChoosing && <div className="tender-choices" role="group" aria-label={`Figure for ${firm(c)}`}>
          {a.others[sub] && <button type="button" className="table-action" disabled={busy} onClick={() => void run({ op: "apply_adjustment", adjustment: a.id, submission: sub, applied: true, source: "others_highest" }).then(() => setChoosing(null))}>Use the highest</button>}
          {a.others[sub] && <button type="button" className="table-action" disabled={busy} onClick={() => void run({ op: "apply_adjustment", adjustment: a.id, submission: sub, applied: true, source: "others_average" }).then(() => setChoosing(null))}>Use the average</button>}
          <button type="button" className="table-action" disabled={busy} onClick={() => { setChoosing(null); setTyping({ adj: a.id, sub, gap: true }); }}>Type a figure</button>
          <button type="button" className="table-action" onClick={() => setChoosing(null)}>Cancel</button>
        </div>}
      </td>;
    }
    const own = a.origin === "user" || m.source === "typed";
    return <td key={sub} className="tender-applied" onDoubleClick={() => { if (!locked && own && a.reason !== "taken_out") setTyping({ adj: a.id, sub, gap: a.reason === "gap" && a.origin === "system" }); }}>
      <span className="prices-figure">{signed(m.amount)}</span>
      <span className="prices-note">{sourceWords(m) || "applied"}</span>
      {!locked && a.origin === "system" && <button type="button" className="table-action" disabled={busy} aria-label={`Undo apply of ${a.label} for ${firm(c)}`} onClick={() => void run({ op: "apply_adjustment", adjustment: a.id, submission: sub, applied: false })}>Undo apply</button>}
      {!locked && a.origin === "user" && a.reason !== "taken_out" && <button type="button" className="table-action" disabled={busy} aria-label={`Clear the amount for ${firm(c)} on ${a.label}`} onClick={() => void run({ op: "set_adjustment_amount", adjustment: a.id, submission: sub, amount: null })}>Clear</button>}
    </td>;
  }

  return <section className="tender-adds" aria-label="Adds and subs"
    onDragOver={e => { if (tenderDrag.current && tenderDrag.current.lines.length === 1 && !locked) e.preventDefault(); }}
    onDrop={e => {
      // A tendered line dragged down becomes a subtraction for that firm.
      const d = tenderDrag.current;
      tenderDrag.current = null;
      if (!d || d.lines.length !== 1 || locked) return;
      e.preventDefault();
      void run({ op: "take_out_line", line: d.lines[0] });
    }}>
    {suggestedWords(tender.suggested) && <p className="tender-suggested-count" role="status">{suggestedWords(tender.suggested)}</p>}
    <div className="prices-scroll">
      <table className="prices-table tender-table">
        <caption>Adds and subs</caption>
        <colgroup><col className="prices-label-col" />{cols.map(c => <col key={c.submission.id} />)}</colgroup>
        <thead><tr>
          <th scope="col" className="prices-item">Addition or subtraction</th>
          {cols.map(c => <th scope="col" key={c.submission.id} className="prices-firm">{firm(c)}
            {read(c) && !locked && tender.adjustments.some(a => a.amounts[c.submission.id]?.state === "suggested" && a.amounts[c.submission.id]?.source === "firm") &&
              <button type="button" className="table-action" disabled={busy} onClick={() => void run({ op: "apply_suggested", submission: c.submission.id })}>Apply this firm's own figures</button>}
          </th>)}
        </tr></thead>
        <tbody>
          {tender.adjustments.map(a => <tr key={a.id}>
            <th scope="row" className="prices-item">
              {renaming === a.id
                ? <TextEditor label="Name" start={a.label} busy={busy} onCancel={() => setRenaming("")} onSave={async label => { if (await run({ op: "set_adjustment", adjustment: a.id, label })) setRenaming(""); }} />
                : <span onDoubleClick={() => { if (a.origin === "user" && !locked) setRenaming(a.id); }}>{a.label}</span>}
              {noting === a.id
                ? <TextEditor label="Note" start={a.note} busy={busy} empty onCancel={() => setNoting("")} onSave={async note => { if (await run({ op: "set_adjustment", adjustment: a.id, note })) setNoting(""); }} />
                : a.note && <span className="prices-note">{a.note}</span>}
              {!locked && renaming !== a.id && noting !== a.id && <span className="tender-row-actions">
                {a.origin === "user" && <button type="button" className="table-action" disabled={busy} onClick={() => setRenaming(a.id)}>Rename</button>}
                <button type="button" className="table-action" disabled={busy} onClick={() => setNoting(a.id)}>Note…</button>
                <button type="button" className="table-action" disabled={busy} aria-label={`Remove ${a.label}`} onClick={() => void run({ op: "remove_adjustment", adjustment: a.id })}>Remove</button>
              </span>}
            </th>
            {cols.map(c => amountCell(a, c))}
          </tr>)}
          {!locked && <tr className="tender-ghost">
            <th scope="row" className="prices-item" colSpan={cols.length + 1}>
              <input aria-label="New add or sub" placeholder="New add or sub" value={adding} maxLength={300} disabled={busy} onChange={e => setAdding(e.target.value)}
                onKeyDown={e => { if (e.key === "Enter" && adding.trim()) { e.preventDefault(); void run({ op: "add_adjustment", label: adding.trim() }).then(ok => { if (ok) setAdding(""); }); } }} />
              <button type="button" className="table-action" disabled={busy || !adding.trim()} onClick={() => void run({ op: "add_adjustment", label: adding.trim() }).then(ok => { if (ok) setAdding(""); })}>Add row</button>
            </th>
          </tr>}
          <tr className="prices-total">
            <th scope="row">Like for like</th>
            {cols.map(c => {
              const t = tcol(c);
              if (t?.state !== "read") return <td key={c.submission.id} />;
              return <td key={c.submission.id} className={t.lowest ? "lowest" : undefined}>
                <span className="prices-figure" aria-label={`Like for like, ${firm(c)}`}>{t.like_for_like == null ? "Unknown" : dollars(t.like_for_like)}</span>
                {t.lowest && <span className="prices-note ok">Lowest</span>}
              </td>;
            })}
          </tr>
        </tbody>
      </table>
    </div>
    <p className="prices-legend">Only applied amounts count. A suggested amount is the firm's own printed figure, or a row a firm left out that others priced. Drag a tendered line down here to subtract it for that firm.</p>
  </section>;
}

function AmountEditor({ start, busy, onSave, onCancel }: { start: string; busy: boolean; onSave: (amount: string) => void; onCancel: () => void }) {
  const [amount, setAmount] = useState(start);
  const save = () => onSave(amount.replace(/[,$\s]/g, "").replace("−", "-"));
  return <div className="tender-editor" role="group" aria-label="Type an amount" onKeyDown={e => { if (e.key === "Enter") { e.preventDefault(); save(); } if (e.key === "Escape") { e.stopPropagation(); onCancel(); } }}>
    <input aria-label="Amount, minus for a subtraction" inputMode="decimal" autoFocus placeholder="-5000 or 5000" value={amount} disabled={busy} onChange={e => setAmount(e.target.value)} />
    <button type="button" className="btn btn-small" disabled={busy} onClick={save}>Save</button>
    <button type="button" className="table-action" onClick={onCancel}>Cancel</button>
  </div>;
}

function TextEditor({ label, start, busy, empty, onSave, onCancel }: { label: string; start: string; busy: boolean; empty?: boolean; onSave: (text: string) => void; onCancel: () => void }) {
  const [text, setText] = useState(start);
  const save = () => { if (text.trim() || empty) onSave(text.trim()); };
  return <span className="tender-editor" role="group" aria-label={label} onKeyDown={e => { if (e.key === "Enter") { e.preventDefault(); save(); } if (e.key === "Escape") { e.stopPropagation(); onCancel(); } }}>
    <input aria-label={label} autoFocus value={text} maxLength={300} disabled={busy} onChange={e => setText(e.target.value)} />
    <button type="button" className="btn btn-small" disabled={busy} onClick={save}>Save</button>
    <button type="button" className="table-action" onClick={onCancel}>Cancel</button>
  </span>;
}
