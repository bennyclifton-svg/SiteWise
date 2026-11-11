import { useEffect, useLayoutEffect, useRef, useState, type FocusEvent, type KeyboardEvent } from "react";
import { createPortal } from "react-dom";
import { submissionApi, type Firm } from "./submissionApi";
import "./table/interaction.css";

export type NamedFirm = { firm_id: string } | { name: string };

/** FirmNameEditor is a Firm cell being named before the firm's quote
 * arrives, or an invited firm's name being corrected
 * (docs/plans/2026-11-11-procurement-invited-firms.md). The cell is the
 * field, as with every inline editor in a table (web/DESIGN.md): Enter or a
 * suggestion saves, Esc cancels, leaving with typed text saves. Firms
 * already in the directory are offered under the cell as plain buttons. */
export function FirmNameEditor({ label, initial = "", exclude, busy, error, onCommit, onCancel }: {
  label: string;
  /** The invited firm's name, when correcting it. */
  initial?: string;
  /** Firms already on the row, not offered again. */
  exclude: string[];
  busy: boolean;
  error: string;
  onCommit: (firm: NamedFirm) => void;
  onCancel: () => void;
}) {
  const [text, setText] = useState(initial);
  const [found, setFound] = useState<Firm[]>([]);
  const [at, setAt] = useState<DOMRect | null>(null);
  const input = useRef<HTMLInputElement>(null);
  const list = useRef<HTMLDivElement>(null);
  const done = useRef(false);
  const skip = exclude.join(",");

  useEffect(() => { input.current?.focus({ preventScroll: true }); input.current?.select(); }, []);
  // A failed save keeps the typed name so the PM can try again.
  useEffect(() => {
    if (error) { done.current = false; input.current?.focus({ preventScroll: true }); }
  }, [error]);
  // The directory search as the name is typed: a database read, no Jev.
  useEffect(() => {
    const q = text.trim();
    // The name as it stands needs no suggestions until it is changed.
    if (!q || q === initial) { setFound([]); return; }
    let stopped = false;
    const t = setTimeout(() => submissionApi.firms(q)
      .then(r => { if (!stopped) setFound(r.firms.filter(f => !skip.split(",").includes(f.id)).slice(0, 6)); })
      .catch(() => { if (!stopped) setFound([]); }), 150);
    return () => { stopped = true; clearTimeout(t); };
  }, [text, skip, initial]);
  // Table cells clip what overflows them, so the list floats under the cell.
  useLayoutEffect(() => {
    if (!found.length) return;
    const place = () => setAt(input.current?.getBoundingClientRect() ?? null);
    place();
    window.addEventListener("scroll", place, true);
    window.addEventListener("resize", place);
    return () => { window.removeEventListener("scroll", place, true); window.removeEventListener("resize", place); };
  }, [found.length]);

  const commit = (firm: NamedFirm) => {
    if (done.current || busy) return;
    done.current = true;
    onCommit(firm);
  };
  const cancel = () => { done.current = true; onCancel(); };
  // A blank or unchanged name leaves the cell as it was; X removes a firm.
  const typed = () => { const name = text.trim(); if (name && name !== initial) commit({ name }); else cancel(); };
  const leave = (e: FocusEvent) => {
    const next = e.relatedTarget as Node | null;
    if (next && (input.current?.contains(next) || list.current?.contains(next))) return;
    if (!done.current && !busy) typed();
  };
  const fieldKeys = (e: KeyboardEvent) => {
    e.stopPropagation();
    if (busy) return;
    if (e.key === "Enter") { e.preventDefault(); typed(); }
    else if (e.key === "Escape") { e.preventDefault(); cancel(); }
    else if (e.key === "ArrowDown" && found.length) { e.preventDefault(); list.current?.querySelector<HTMLButtonElement>("button")?.focus(); }
  };
  const listKeys = (e: KeyboardEvent) => {
    e.stopPropagation();
    const all = Array.from(list.current?.querySelectorAll<HTMLButtonElement>("button") ?? []);
    const i = all.indexOf(document.activeElement as HTMLButtonElement);
    if (e.key === "Escape") { e.preventDefault(); cancel(); }
    else if (e.key === "ArrowDown") { e.preventDefault(); all[Math.min(i + 1, all.length - 1)]?.focus(); }
    else if (e.key === "ArrowUp") { e.preventDefault(); if (i <= 0) input.current?.focus(); else all[i - 1]?.focus(); }
  };

  return <>
    <input ref={input} className="inline-cell-editor field-plain firm-name-editor" aria-label={label}
      aria-invalid={error ? true : undefined} aria-description={error || undefined} title={error || undefined}
      placeholder="Firm name" maxLength={200} value={text} disabled={busy}
      onChange={e => { done.current = false; setText(e.target.value); }} onKeyDown={fieldKeys} onBlur={leave}
      onClick={e => e.stopPropagation()} />
    {found.length > 0 && at && createPortal(
      <div ref={list} className="page-action-menu firm-suggest" role="group" aria-label="Firms in your directory"
        style={{ top: at.bottom + 2, left: at.left, minWidth: at.width }} onKeyDown={listKeys}>
        {/* Mouse down keeps the field focused, so choosing is not leaving. */}
        {found.map(f => <button key={f.id} type="button" disabled={busy} onMouseDown={e => e.preventDefault()} onBlur={leave}
          onClick={() => commit({ firm_id: f.id })}>
          <span className="firm-suggest-name">{f.name}</span>{f.abn && <span className="firm-suggest-abn">ABN {f.abn}</span>}
        </button>)}
      </div>, document.body)}
  </>;
}
