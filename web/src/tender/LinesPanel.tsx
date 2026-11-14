// The firm's own lines behind one cell of the first table: wording, amount,
// page. Here the manager moves a line to another row, splits it, corrects
// it, adds one and takes one out of the count. Selection, the right-click
// menu and editing follow the document register and the cost plan
// (states.css). The browser never adds: every figure comes from the server.
import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import { SYSTEM_ROW, type TenderColumn, type TenderLine, type TenderOp, type TenderRow } from "./tenderApi";
import { tenderDrag } from "./useTenderEdits";
import { dollars, printedWords } from "./words";

const SCOPE = new Set(["base", "ps", "pc"]);

/** The lines one cell shows: those on its row, counted first. The system
 * row shows the lines that sit on no row of the schedule. */
export function linesOf(col: TenderColumn, row: TenderRow, rows: TenderRow[]): TenderLine[] {
  const known = new Set(rows.filter(r => !r.system).map(r => r.key));
  const here = col.lines.filter(l => {
    if (l.kind === "subtotal" || l.kind === "gst" || l.kind === "rate") return false;
    if (row.system) return (!l.row || !known.has(l.row)) && (l.counted || SCOPE.has(l.kind) || (l.kind === "total" && l.decided_by === "user"));
    return l.row === row.key && l.kind !== "total";
  });
  return [...here.filter(l => l.counted), ...here.filter(l => !l.counted)];
}

interface Props {
  col: TenderColumn;
  firm: string;
  row: TenderRow;
  rows: TenderRow[];
  busy: boolean;
  /** False for an awarded firm: its lines are read only. */
  editable: boolean;
  run: (ops: TenderOp[]) => Promise<boolean>;
  onOpenPage: (page: number) => void;
}

export function LinesPanel({ col, firm, row, rows, busy, editable, run, onOpenPage }: Props) {
  const lines = linesOf(col, row, rows);
  const [selected, setSelected] = useState("");
  const [menu, setMenu] = useState("");
  const [editing, setEditing] = useState("");
  const [splitting, setSplitting] = useState("");
  const rowKey = row.system ? "" : row.key;
  const hasParts = (id: string) => col.lines.some(l => l.split_of === id);
  const own = (l: TenderLine) => l.decided_by === "user" && (l.note === "Added by you" || l.note === "Your figure for this row" || l.kind === "total");

  function keys(e: KeyboardEvent, l: TenderLine) {
    if (!editable || editing || splitting || (e.target as HTMLElement).closest("input, select, [role=menu]")) return;
    if (e.key === "m" || e.key === "M") { e.preventDefault(); setSelected(l.id); setMenu(l.id); }
    if (e.key === "F2") { e.preventDefault(); setEditing(l.id); }
    if (e.key === "Delete" && own(l) && !hasParts(l.id)) { e.preventDefault(); void run([{ op: "delete_line", line: l.id }]); }
  }

  return <div className="tender-lines" role="region" aria-label={`${firm}: lines in ${row.label}`}>
    {lines.length === 0 && <p className="prices-note">No lines here.</p>}
    {lines.length > 0 && <table className="tender-lines-table">
      <thead><tr><th scope="col">The firm's wording</th><th scope="col" className="tender-num">Amount ex GST</th><th scope="col">Page</th><th scope="col">Notes</th><th scope="col"><span className="prices-sr-only">Actions</span></th></tr></thead>
      <tbody>{lines.map(l => {
        const toType = l.state === "priced" && l.amount == null;
        if (editing === l.id) return <tr key={l.id} aria-selected="true"><td colSpan={5}>
          <LineEditor line={l} busy={busy} onCancel={() => setEditing("")} onSave={async change => { if (await run([{ op: "set_line", line: l.id, ...change }])) setEditing(""); }} />
        </td></tr>;
        return <tr key={l.id} aria-selected={selected === l.id} className={l.counted ? undefined : "tender-uncounted"} tabIndex={0} draggable={editable && l.counted && !busy}
          onClick={() => setSelected(l.id)} onDoubleClick={() => { if (editable) setEditing(l.id); }} onKeyDown={e => keys(e, l)}
          onContextMenu={e => { if (!editable) return; e.preventDefault(); setSelected(l.id); setMenu(l.id); }}
          onDragStart={e => { tenderDrag.current = { sub: col.submission_id, lines: [l.id] }; e.dataTransfer.effectAllowed = "move"; e.dataTransfer.setData("text/plain", l.label); }}
          onDragEnd={() => { tenderDrag.current = null; }}>
          <td>{l.label}{l.heading && <span className="tender-heading"> · {l.heading}</span>}{!l.counted && <span className="tender-tag">not counted</span>}</td>
          <td className="tender-num">{toType ? <button type="button" className="table-action" disabled={!editable || busy} onClick={() => setEditing(l.id)}>Amount to type</button> : l.amount == null ? l.state === "included" ? "Included" : l.state === "excluded" ? "Excluded" : "Not priced" : dollars(l.amount)}
            {printedWords(l) && <span className="prices-note">{printedWords(l)}</span>}</td>
          <td>{l.page > 0 ? <button type="button" className="table-action" onClick={e => { e.stopPropagation(); onOpenPage(l.page); }}>p. {l.page}</button> : "—"}</td>
          <td>{l.flags.map(f => <span key={f.code} className="prices-note warn">{f.text}</span>)}{l.note && <span className="prices-note">{l.note}</span>}</td>
          <td className="tender-line-actions">
            {editable && <button type="button" className="price-icon" aria-label={`Actions for ${l.label}`} aria-haspopup="menu" aria-expanded={menu === l.id} disabled={busy} onClick={e => { e.stopPropagation(); setSelected(l.id); setMenu(menu === l.id ? "" : l.id); }}>⋯</button>}
            {menu === l.id && <LineMenu line={l} rows={rows} current={rowKey} parts={hasParts(l.id)} own={own(l)} onClose={() => setMenu("")}
              onChoose={async action => {
                setMenu("");
                if (action.kind === "edit") setEditing(l.id);
                else if (action.kind === "split") setSplitting(l.id);
                else await run([action.op]);
              }} />}
          </td>
        </tr>;
      })}</tbody>
    </table>}
    {splitting && <SplitDialog line={lines.find(l => l.id === splitting)!} rows={rows} busy={busy} onCancel={() => setSplitting("")}
      onSplit={async (parts, rest) => { if (await run([{ op: "split_line", line: splitting, parts, rest_row: rest }])) setSplitting(""); }} />}
    {editable && <AddLine busy={busy} onAdd={(label, amount, gst) => run([{ op: "add_line", submission: col.submission_id, label, amount, gst, row: rowKey }])} />}
  </div>;
}

type Action = { kind: "op"; op: TenderOp } | { kind: "edit" } | { kind: "split" };

// The right-click menu of a line. Arrow keys move through it, Enter chooses,
// Escape closes: pressing M on a selected line opens the same list.
function LineMenu({ line, rows, current, parts, own, onChoose, onClose }: { line: TenderLine; rows: TenderRow[]; current: string; parts: boolean; own: boolean; onChoose: (a: Action) => void; onClose: () => void }) {
  const root = useRef<HTMLDivElement>(null);
  useEffect(() => {
    root.current?.querySelector<HTMLElement>("button")?.focus();
    const away = (e: PointerEvent) => { if (!root.current?.contains(e.target as Node)) onClose(); };
    document.addEventListener("pointerdown", away);
    return () => document.removeEventListener("pointerdown", away);
  }, [onClose]);
  function arrows(e: KeyboardEvent) {
    const items = [...(root.current?.querySelectorAll<HTMLElement>("button") ?? [])];
    const at = items.indexOf(document.activeElement as HTMLElement);
    if (e.key === "ArrowDown") { e.preventDefault(); items[(at + 1) % items.length]?.focus(); }
    if (e.key === "ArrowUp") { e.preventDefault(); items[(at - 1 + items.length) % items.length]?.focus(); }
    if (e.key === "Escape") { e.preventDefault(); e.stopPropagation(); onClose(); }
  }
  const scope = SCOPE.has(line.kind);
  const op = (o: TenderOp): Action => ({ kind: "op", op: o });
  return <div className="page-action-menu tender-menu" role="menu" aria-label={`Actions for ${line.label}`} ref={root} onKeyDown={arrows} onClick={e => e.stopPropagation()}>
    {scope && line.amount != null && rows.filter(r => (r.system ? "" : r.key) !== current).map(r =>
      <button type="button" role="menuitem" key={r.key} onClick={() => onChoose(op({ op: "move_line", line: line.id, row: r.system ? "" : r.key }))}>Move to {r.label}</button>)}
    {line.counted && !line.split_of && !parts && <button type="button" role="menuitem" onClick={() => onChoose({ kind: "split" })}>Split…</button>}
    {parts && <button type="button" role="menuitem" onClick={() => onChoose(op({ op: "unsplit_line", line: line.id }))}>Put back together</button>}
    {!parts && <button type="button" role="menuitem" onClick={() => onChoose({ kind: "edit" })}>Correct…</button>}
    {line.counted && <button type="button" role="menuitem" onClick={() => onChoose(op({ op: "count_line", line: line.id, counted: false }))}>Take out of the count</button>}
    {!line.counted && scope && line.amount != null && !parts && <button type="button" role="menuitem" onClick={() => onChoose(op({ op: "count_line", line: line.id, counted: true }))}>Count</button>}
    {line.counted && <button type="button" role="menuitem" onClick={() => onChoose(op({ op: "take_out_line", line: line.id }))}>Take out as a subtraction</button>}
    {own && !parts && <button type="button" role="menuitem" onClick={() => onChoose(op({ op: "delete_line", line: line.id }))}>Delete</button>}
  </div>;
}

function LineEditor({ line, busy, onSave, onCancel }: { line: TenderLine; busy: boolean; onSave: (change: { label?: string; amount?: string; gst?: string }) => void; onCancel: () => void }) {
  const [label, setLabel] = useState(line.label);
  const [amount, setAmount] = useState(line.amount ?? "");
  const [gst, setGst] = useState(line.gst === "inc" ? "inc" : "ex");
  const toType = line.amount == null;
  function save() {
    const change: { label?: string; amount?: string; gst?: string } = {};
    if (label.trim() !== line.label) change.label = label.trim();
    const typed = amount.replace(/[,$\s]/g, "");
    // The amount is typed in the basis chosen, so the basis goes with it.
    if (typed !== (line.amount ?? "") || gst !== (line.gst === "inc" ? "inc" : "ex")) {
      if (gst !== line.gst) change.gst = gst;
      if (typed !== "") change.amount = typed;
    }
    if (Object.keys(change).length === 0) { onCancel(); return; }
    onSave(change);
  }
  return <div className="tender-editor" role="group" aria-label={`Correct ${line.label}`} onKeyDown={e => { if (e.key === "Enter") { e.preventDefault(); save(); } if (e.key === "Escape") { e.stopPropagation(); onCancel(); } }}>
    <input aria-label="The firm's wording" value={label} maxLength={500} disabled={busy} autoFocus={!toType} onChange={e => setLabel(e.target.value)} />
    <input aria-label="Amount" inputMode="decimal" placeholder="0.00" value={amount} disabled={busy} autoFocus={toType} onChange={e => setAmount(e.target.value)} />
    <select aria-label="GST" value={gst} disabled={busy} onChange={e => setGst(e.target.value)}><option value="ex">ex GST</option><option value="inc">inc GST</option></select>
    <button type="button" className="btn btn-small" disabled={busy} onClick={save}>Save</button>
    <button type="button" className="table-action" onClick={onCancel}>Cancel</button>
  </div>;
}

// Split a line across rows. The manager names parts; the rest goes to one
// row. The browser does not add: the server works out the rest, so the
// parts always add to the line, and says so when they are too much.
function SplitDialog({ line, rows, busy, onSplit, onCancel }: { line: TenderLine; rows: TenderRow[]; busy: boolean; onSplit: (parts: { row: string; amount: string }[], rest: string) => void; onCancel: () => void }) {
  const first = rows.find(r => !r.system)?.key ?? "";
  const [parts, setParts] = useState([{ row: first, amount: "" }]);
  const [rest, setRest] = useState(line.row || SYSTEM_ROW);
  const option = (r: TenderRow) => <option key={r.key} value={r.key}>{r.label}</option>;
  const send = () => onSplit(parts.filter(p => p.amount.trim()).map(p => ({ row: p.row === SYSTEM_ROW ? "" : p.row, amount: p.amount.replace(/[,$\s]/g, "") })), rest === SYSTEM_ROW ? "" : rest);
  return <div className="tender-split" role="dialog" aria-label="Split line" onKeyDown={e => { if (e.key === "Escape") { e.stopPropagation(); onCancel(); } }}>
    <p><strong>{line.label}</strong> · {dollars(line.amount)}{line.gst === "inc" && line.printed ? ` (printed ${line.printed} inc GST; type parts inc GST)` : ""}</p>
    {parts.map((p, i) => <div key={i} className="tender-split-part">
      <input aria-label={`Part ${i + 1} amount`} inputMode="decimal" placeholder="0.00" value={p.amount} disabled={busy} autoFocus={i === 0} onChange={e => setParts(parts.map((x, j) => j === i ? { ...x, amount: e.target.value } : x))} />
      <select aria-label={`Part ${i + 1} row`} value={p.row} disabled={busy} onChange={e => setParts(parts.map((x, j) => j === i ? { ...x, row: e.target.value } : x))}>{rows.map(option)}</select>
    </div>)}
    <button type="button" className="table-action" disabled={busy || parts.length >= 10} onClick={() => setParts([...parts, { row: first, amount: "" }])}>Another part</button>
    <label className="tender-split-rest">The rest goes to <select aria-label="The rest goes to" value={rest} disabled={busy} onChange={e => setRest(e.target.value)}>{rows.map(option)}</select></label>
    <button type="button" className="btn btn-small" disabled={busy || !parts.some(p => p.amount.trim())} onClick={send}>Split</button>
    <button type="button" className="table-action" onClick={onCancel}>Cancel</button>
  </div>;
}

// The ghost row at the foot of the panel, as in the cost plan: a label, an
// amount and Enter add the manager's own line to this row.
function AddLine({ busy, onAdd }: { busy: boolean; onAdd: (label: string, amount: string, gst: string) => Promise<boolean> }) {
  const [label, setLabel] = useState("");
  const [amount, setAmount] = useState("");
  const [gst, setGst] = useState("ex");
  async function add() {
    if (!label.trim() || !amount.trim()) return;
    if (await onAdd(label.trim(), amount.replace(/[,$\s]/g, ""), gst)) { setLabel(""); setAmount(""); }
  }
  return <div className="tender-add" role="group" aria-label="Add a line" onKeyDown={e => { if (e.key === "Enter") { e.preventDefault(); void add(); } }}>
    <input aria-label="New line wording" placeholder="Add a line" value={label} maxLength={500} disabled={busy} onChange={e => setLabel(e.target.value)} />
    <input aria-label="New line amount" inputMode="decimal" placeholder="0.00" value={amount} disabled={busy} onChange={e => setAmount(e.target.value)} />
    <select aria-label="New line GST" value={gst} disabled={busy} onChange={e => setGst(e.target.value)}><option value="ex">ex GST</option><option value="inc">inc GST</option></select>
    <button type="button" className="table-action" disabled={busy || !label.trim() || !amount.trim()} onClick={() => void add()}>Add line</button>
  </div>;
}
