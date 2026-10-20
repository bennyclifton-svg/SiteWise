// The Program grid (plan 9.1, 9.5): a treegrid of stages and activities.
// Only the rows in the render window are in the DOM; the block is offset by
// first × rowH so it lines up with the chart's rows. Names, durations and
// predecessors edit in place (click a selected name, double-click, F2 or
// Enter); the view validates what is typed and keeps the cell open with a
// message when it is not valid. A handle in the # cell drags rows to
// reorder them. Every cell carries data-col for focus and editing.

import { memo, useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";
import type { Cal } from "../calendar";
import type { Times } from "../types";
import { shownDates, type Row } from "../layout";

export type EditCol = "name" | "duration" | "predecessors";
export type CommitKey = "enter" | "tab" | "shift-tab" | "blur";

export interface Editing {
  id: string;
  col: EditCol;
  error: string;
}

export interface CellFocus {
  id: string;
  col: string;
}

export interface GridProps {
  rows: readonly Row[];
  times: Record<string, Times>;
  cal: Cal;
  first: number;
  last: number;
  rowH: number;
  /** Name column only (narrow screens, plan 9.1). */
  narrow: boolean;
  selected: ReadonlySet<string>;
  /** Changes when the program changes, so memo can skip scroll-only renders. */
  version: number;
  onToggle: (stageId: string) => void;
  onSelect: (id: string, mods: { ctrl: boolean; shift: boolean }) => void;
  /** Row id → predecessors in the typed format. */
  preds?: ReadonlyMap<string, string>;
  editing?: Editing | null;
  focus?: CellFocus | null;
  /** Delivery catalogue stages, for a user-made stage's control (owner answer Q5). */
  catalogue?: readonly { id: string; label: string }[] | null;
  onEdit?: (id: string, col: EditCol) => void;
  onCommit?: (id: string, col: EditCol, text: string, key: CommitKey) => void;
  onCancel?: () => void;
  onFocusCell?: (id: string, col: string) => void;
  onHandleDown?: (e: ReactPointerEvent<HTMLElement>, id: string) => void;
  onCatalogue?: (id: string, stage: string) => void;
}

export const GRID_COLUMNS = ["#", "Name", "Duration", "Start", "Finish", "Predecessors"] as const;

export function durationText(row: Row, t: Times | undefined): string {
  const a = row.act;
  if (row.hasChildren) return t && t.es !== null && t.ef !== null ? `${t.ef - t.es}d` : "Unknown";
  if (a.duration_days === null) return "Unknown";
  if (a.duration_low !== null && a.duration_high !== null && a.duration_low !== a.duration_high) {
    return `${a.duration_days}d (${a.duration_low}–${a.duration_high})`;
  }
  return a.duration_days === 0 ? "Milestone" : `${a.duration_days}d`;
}

/** The text an editor opens with: what the cell would accept back. */
export function editText(row: Row, col: EditCol, preds: ReadonlyMap<string, string> | undefined): string {
  if (col === "name") return row.act.label;
  if (col === "predecessors") return preds?.get(row.id) ?? "";
  return row.act.duration_days === null ? "" : String(row.act.duration_days);
}

export const Grid = memo(function Grid(p: GridProps) {
  const { rows, times, cal, first, last, rowH, narrow, selected, onToggle, onSelect, preds, editing, focus, catalogue } = p;
  const out = [];
  for (let i = first; i < last; i++) {
    const r = rows[i];
    const t = times[r.id];
    const d = shownDates(cal, t);
    const unknown = durationText(r, t) === "Unknown";
    const proposed = r.act.review_status === "proposed";
    const ed = editing && editing.id === r.id ? editing : null;
    const cell = (col: string) => ({
      "data-col": col,
      tabIndex: focus && focus.id === r.id && focus.col === col ? 0 : -1,
      "data-focus": focus && focus.id === r.id && focus.col === col ? "1" : undefined,
      onFocus: () => p.onFocusCell?.(r.id, col),
    });
    const editor = (col: EditCol) =>
      ed && ed.col === col ? (
        <CellEditor
          key={`${r.id}:${col}`}
          value={editText(r, col, preds)}
          label={`${col === "name" ? "Name" : col === "duration" ? "Duration" : "Predecessors"} of ${r.act.label}`}
          error={ed.error}
          onCommit={(text, key) => p.onCommit?.(r.id, col, text, key)}
          onCancel={() => p.onCancel?.()}
        />
      ) : null;
    out.push(
      <div
        key={r.id}
        role="row"
        aria-level={r.stage ? 1 : 2}
        aria-expanded={r.hasChildren ? r.expanded : undefined}
        aria-selected={selected.has(r.id)}
        aria-rowindex={i + 2}
        data-id={r.id}
        className={`pg-row${r.stage ? " is-stage" : ""}${selected.has(r.id) ? " is-selected" : ""}${t?.critical ? " is-critical" : ""}${ed ? " is-editing" : ""}`}
        style={{ height: rowH }}
        onClick={(e) => onSelect(r.id, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })}
      >
        {!narrow && (
          <span role="gridcell" className="pg-cell pg-num" {...cell("number")}>
            <span className="pg-handle" title="Drag to reorder" aria-hidden="true" onPointerDown={(e) => p.onHandleDown?.(e, r.id)} />
            {r.number}
          </span>
        )}
        <span
          role="gridcell"
          className="pg-cell pg-name"
          {...cell("name")}
          onDoubleClick={() => p.onEdit?.(r.id, "name")}
        >
          {r.hasChildren ? (
            <button
              type="button"
              className="pg-chevron"
              tabIndex={-1}
              aria-label={`${r.expanded ? "Collapse" : "Expand"} ${r.act.label}`}
              aria-expanded={r.expanded}
              onClick={(e) => {
                e.stopPropagation();
                onToggle(r.id);
              }}
            >
              <svg viewBox="0 0 12 12" aria-hidden="true">
                <path d={r.expanded ? "M2.5 4.5 6 8l3.5-3.5" : "M4.5 2.5 8 6l-3.5 3.5"} />
              </svg>
            </button>
          ) : (
            <span className="pg-chevron-gap" aria-hidden="true" />
          )}
          {editor("name") ?? (
            <span
              className="pg-label"
              title={r.act.label}
              onClick={(e) => {
                // A click on the name of the one selected row renames it (plan 9.4).
                if (selected.has(r.id) && selected.size === 1 && !e.ctrlKey && !e.metaKey && !e.shiftKey) {
                  e.stopPropagation();
                  p.onEdit?.(r.id, "name");
                }
              }}
            >
              {r.act.label}
            </span>
          )}
          {proposed && (
            <span className="pg-proposed" title="Proposed: not yet accepted">
              <span className="sr-only">, proposed</span>
            </span>
          )}
          {!narrow && !ed && r.stage && r.act.catalogue_stage_id === "other" && catalogue && catalogue.length > 0 && (
            <select
              className="pg-cat"
              aria-label={`Catalogue stage for ${r.act.label}`}
              title="Which catalogue stage this is"
              value="other"
              onClick={(e) => e.stopPropagation()}
              onChange={(e) => p.onCatalogue?.(r.id, e.target.value)}
            >
              {catalogue.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.id === "other" ? "Other stage" : c.label}
                </option>
              ))}
            </select>
          )}
          {narrow && unknown && <span className="pg-unknown-tag">?</span>}
        </span>
        {!narrow && (
          <>
            <span
              role="gridcell"
              className={`pg-cell pg-dur${unknown ? " is-unknown" : ""}`}
              {...cell("duration")}
              onDoubleClick={() => p.onEdit?.(r.id, "duration")}
            >
              {editor("duration") ?? durationText(r, t)}
            </span>
            <span role="gridcell" className="pg-cell pg-date" {...cell("start")}>
              {d.start}
            </span>
            <span role="gridcell" className="pg-cell pg-date" {...cell("finish")}>
              {d.finish}
            </span>
            <span role="gridcell" className="pg-cell pg-pred" {...cell("predecessors")} onDoubleClick={() => p.onEdit?.(r.id, "predecessors")}>
              {editor("predecessors") ?? preds?.get(r.id) ?? ""}
            </span>
          </>
        )}
        {ed?.error && (
          <span className="pg-cell-error" role="alert">
            {ed.error}
          </span>
        )}
      </div>,
    );
  }
  return (
    <div className="pg-rows" style={{ transform: `translateY(${first * rowH}px)` }}>
      {out}
    </div>
  );
});

// One cell editor: Enter saves, Escape cancels, Tab saves and moves on.
// Keys stay inside it so the grid's shortcuts never fire while typing.
function CellEditor({ value, label, error, onCommit, onCancel }: { value: string; label: string; error: string; onCommit: (text: string, key: CommitKey) => void; onCancel: () => void }) {
  const [text, setText] = useState(value);
  const input = useRef<HTMLInputElement>(null);
  const done = useRef(false);
  useEffect(() => {
    input.current?.focus({ preventScroll: true });
    input.current?.select();
  }, []);
  useEffect(() => {
    // A refused value reopens the editor for another try.
    if (error) done.current = false;
  }, [error]);
  const finish = (key: CommitKey) => {
    if (done.current) return;
    done.current = true;
    onCommit(text, key);
  };
  return (
    <input
      ref={input}
      className="pg-edit"
      aria-label={label}
      aria-invalid={error ? true : undefined}
      value={text}
      onChange={(e) => {
        done.current = false;
        setText(e.target.value);
      }}
      onClick={(e) => e.stopPropagation()}
      onDoubleClick={(e) => e.stopPropagation()}
      onKeyDown={(e) => {
        e.stopPropagation();
        if (e.key === "Enter") {
          e.preventDefault();
          finish("enter");
        } else if (e.key === "Escape") {
          e.preventDefault();
          done.current = true;
          onCancel();
        } else if (e.key === "Tab") {
          e.preventDefault();
          finish(e.shiftKey ? "shift-tab" : "tab");
        }
      }}
      onBlur={() => {
        if (done.current) return;
        if (text === value) {
          done.current = true;
          onCancel();
        } else finish("blur");
      }}
    />
  );
}
