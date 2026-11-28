import { Fragment, useState, type KeyboardEvent, type ReactNode } from "react";
import { reportApi, type ReportTableData, type RowFacts } from "./reportApi";
import { MoreMenu, type MenuItem } from "./PageActions";

// A table of the project report in the owner's format
// (docs/plans/2026-11-26-project-report-golden-format.md, sections 6 and 7).
// Rows come from records (Go's, every Update), are the PM's own, or are the
// template's grey placeholders. Click a cell to type over it; the words
// stay through Update, and when SiteWise's value under them moves the row
// says so and issue waits for Keep mine or Use SiteWise's. Nothing here
// works out a figure.

/** One change to a table, as the row routes take it. */
export interface RowChange {
  table?: string; key?: string; row?: string;
  cells?: Record<string, string | null>; keep?: string[]; hidden?: boolean; move?: "up" | "down"; after?: string;
}

interface Props {
  reportId: string;
  table: ReportTableData;
  label: string;
  busy: boolean;
  onWrite: (change: RowChange) => Promise<void>;
  /** Give a laid row back to SiteWise, or delete the PM's own row. */
  onDelete: (row: string) => Promise<void>;
}

/** A cell's few marks: "- " bullets, "**bold**" lines. */
const IMAGE = /^!\[([^\]]*)\]\(image:([0-9a-f]{64})\)$/;

/** A cell's few marks: "- " bullets, "**bold**" lines, and image lines. */
export function CellText({ text, reportId }: { text: string; reportId?: string }): ReactNode {
  const lines = text.split("\n");
  const out: ReactNode[] = [];
  let bullets: string[] = [];
  const flush = () => { if (bullets.length) { out.push(<ul key={`u${out.length}`} className="rt-bullets">{bullets.map((b, i) => <li key={i}>{b}</li>)}</ul>); bullets = []; } };
  lines.forEach((line, i) => {
    if (line.startsWith("- ")) { bullets.push(line.slice(2)); return; }
    flush();
    const img = reportId ? IMAGE.exec(line.trim()) : null;
    if (img) { out.push(<figure key={i} className="rt-figure"><img src={reportApi.imageURL(reportId!, img[2])} alt={img[1]} loading="lazy" /><figcaption>{img[1]}</figcaption></figure>); return; }
    const bold = line.length > 4 && line.startsWith("**") && line.endsWith("**");
    out.push(bold ? <strong key={i} className="rt-line">{line.slice(2, -2)}</strong> : <span key={i} className="rt-line">{line}</span>);
  });
  flush();
  return <>{out}</>;
}

function target(t: ReportTableData, f: RowFacts): RowChange {
  return f.id && (!f.key || f.kind === "report" && !f.key) ? { row: f.id } : { table: t.id, key: f.key };
}

export function ReportTable({ reportId, table, label, busy, onWrite, onDelete }: Props) {
  const [editing, setEditing] = useState<{ row: number; col: number; text: string } | null>(null);
  const [adding, setAdding] = useState("");
  const [dropError, setDropError] = useState("");
  // A PNG or JPEG dropped on a cell goes under its words.
  const dropImage = async (r: number, c: number, files: FileList) => {
    const file = Array.from(files).find(f => /^image\/(png|jpeg)$/.test(f.type));
    if (!file) { setDropError("Only a PNG or JPEG image can go in the report."); return; }
    setDropError("");
    try {
      const saved = await reportApi.image(reportId, file);
      const f = table.meta[r] ?? {};
      const was = table.rows[r]?.[c] ?? "";
      const keep = f.kind === "placeholder" && /^(TBD|Nil to report)$/.test(was) ? "" : was;
      await onWrite({ ...target(table, f), cells: { [String(c)]: keep ? `${keep}
${saved.line}` : saved.line } });
    } catch (e) { setDropError(e instanceof Error ? e.message : "The image could not be saved."); }
  };
  const save = async () => {
    if (!editing) return;
    const f = table.meta[editing.row] ?? {};
    const was = table.rows[editing.row]?.[editing.col] ?? "";
    const text = editing.text.trim();
    setEditing(null);
    if (text === was.trim()) return;
    await onWrite({ ...target(table, f), cells: { [String(editing.col)]: text } });
  };
  const keys = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Escape") { e.preventDefault(); setEditing(null); }
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); void save(); }
  };
  const hidden = table.hidden ?? 0;
  return <div className="report-table-scroll rt-scroll">
    {dropError && <p className="rt-flag" role="alert">{dropError}</p>}
    <table className="report-table rt" aria-label={label}>
      <thead><tr>{table.columns.map((c, i) => <th key={i} scope="col">{c}</th>)}<th scope="col" className="rt-act"><span className="sr-only">Row actions</span></th></tr></thead>
      <tbody>
        {table.rows.map((row, r) => {
          const f = table.meta[r] ?? {};
          const items: MenuItem[] = [
            { label: "Move up", icon: null, disabled: busy || r === 0, onSelect: () => void onWrite({ ...target(table, f), move: "up" }) },
            { label: "Move down", icon: null, disabled: busy || r === table.rows.length - 1, onSelect: () => void onWrite({ ...target(table, f), move: "down" }) },
            ...(f.kind === "report" && !f.key && f.id ? [{ label: "Delete row", icon: null, disabled: busy, onSelect: () => void onDelete(f.id!) }]
              : [{ label: "Hide row", icon: null, disabled: busy, onSelect: () => void onWrite({ ...target(table, f), hidden: true }) }]),
            ...(f.id && f.key ? [{ label: "Give back to SiteWise", icon: null, disabled: busy, onSelect: () => void onDelete(f.id!) }] : []),
          ];
          return <Fragment key={`${f.key ?? ""}:${f.id ?? r}`}><tr data-kind={f.kind} data-new={f.new || undefined} data-flag={f.flag || undefined}>
            {row.map((cell, c) => {
              const isTyped = !!f.typed?.[c];
              const open = editing?.row === r && editing.col === c;
              return <td key={c} data-tone={f.tones?.[c] || undefined} data-changed={f.changed?.[c] || undefined}
                onDragOver={e => { if (e.dataTransfer.types.includes("Files")) { e.preventDefault(); e.stopPropagation(); } }}
                onDrop={e => { if (!e.dataTransfer.files.length) return; e.preventDefault(); e.stopPropagation(); void dropImage(r, c, e.dataTransfer.files); }}
                title={isTyped && f.base?.[c] !== undefined ? `SiteWise has: ${f.base[c] || "nothing"}` : undefined}>
                {open
                  ? <textarea className="rt-edit" aria-label={`${table.columns[c]}, ${row[0] || "row"}`} autoFocus rows={Math.max(1, editing.text.split("\n").length)} value={editing.text} disabled={busy}
                      onChange={e => setEditing({ ...editing, text: e.target.value })} onKeyDown={keys} onBlur={() => void save()} />
                  : <button type="button" className="rt-cell" disabled={busy || !!editing} aria-label={`Edit ${table.columns[c]}, ${row[0] || "row"}`}
                      onClick={() => setEditing({ row: r, col: c, text: f.kind === "placeholder" && /^(TBD|Nil to report)$/.test(cell) ? "" : cell })}>
                      {isTyped && <span className="rt-dot" data-origin="user" aria-label="Your words" />}<CellText text={cell} reportId={reportId} />
                      {c === row.length - 1 && f.source && <span className="rt-source">Words read from a filed document</span>}
                    </button>}
              </td>;
            })}
            <td className="rt-act"><MoreMenu label={`Row ${row[0] || r + 1} actions`} items={items} /></td>
          </tr>
            {f.flag && <tr className="rt-flag-row"><td colSpan={row.length + 1}>{f.flag === "missing" ? <p className="rt-flag">Its source is no longer in SiteWise. <button type="button" className="cell-link" disabled={busy} onClick={() => f.id && void onDelete(f.id)}>Remove it</button></p>
              : <p className="rt-flag">SiteWise's value has changed under your words{(f.typed ?? []).map((t, c) => t ? `: “${f.base?.[c] ?? ""}”` : "").filter(Boolean).slice(0, 1).join("")}.{" "}
                <button type="button" className="cell-link" disabled={busy} onClick={() => void onWrite({ ...target(table, f), keep: (f.typed ?? []).map((t, c) => t ? String(c) : "").filter(Boolean) })}>Keep mine</button>{" · "}
                <button type="button" className="cell-link" disabled={busy} onClick={() => void onWrite({ ...target(table, f), cells: Object.fromEntries((f.typed ?? []).map((t, c) => t ? [String(c), null] : null).filter(Boolean) as [string, null][]) })}>Use SiteWise's</button></p>}</td></tr>}
          </Fragment>;
        })}
        <tr className="rt-ghost">
          <td colSpan={table.columns.length + 1}>
            <input className="field-plain" aria-label={`New row in ${label}`} placeholder="Add a row" value={adding} disabled={busy}
              onChange={e => setAdding(e.target.value)}
              onKeyDown={e => { if (e.key === "Enter" && adding.trim()) { const text = adding.trim(); setAdding(""); void onWrite({ table: table.id, cells: { "0": text } }); } }} />
          </td>
        </tr>
      </tbody>
    </table>
    {hidden > 0 && <p className="rt-hidden">{hidden === 1 ? "1 row hidden" : `${hidden} rows hidden`}. {table.hidden_ids?.length ? <button type="button" className="cell-link" disabled={busy} onClick={() => void Promise.all((table.hidden_ids ?? []).map(id => onDelete(id)))}>Show them again</button> : null}</p>}
  </div>;
}
