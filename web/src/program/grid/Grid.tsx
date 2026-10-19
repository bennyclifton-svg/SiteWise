// The Program grid (plan 9.1, 9.5): a treegrid of stages and activities.
// Only the rows in the render window are in the DOM; the block is offset by
// first × rowH so it lines up with the chart's rows. Read-only in WP-P7:
// WP-P8 adds rename, typed cells and row drag on these same rows (each cell
// already carries data-col so an editor can mount in place).

import { memo } from "react";
import type { Cal } from "../calendar";
import type { Times } from "../types";
import { shownDates, type Row } from "../layout";

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
}

export const GRID_COLUMNS = ["#", "Name", "Duration", "Start", "Finish"] as const;

export function durationText(row: Row, t: Times | undefined): string {
  const a = row.act;
  if (row.hasChildren) return t && t.es !== null && t.ef !== null ? `${t.ef - t.es}d` : "Unknown";
  if (a.duration_days === null) return "Unknown";
  if (a.duration_low !== null && a.duration_high !== null && a.duration_low !== a.duration_high) {
    return `${a.duration_days}d (${a.duration_low}–${a.duration_high})`;
  }
  return a.duration_days === 0 ? "Milestone" : `${a.duration_days}d`;
}

export const Grid = memo(function Grid({ rows, times, cal, first, last, rowH, narrow, selected, onToggle, onSelect }: GridProps) {
  const out = [];
  for (let i = first; i < last; i++) {
    const r = rows[i];
    const t = times[r.id];
    const d = shownDates(cal, t);
    const unknown = durationText(r, t) === "Unknown";
    const proposed = r.act.review_status === "proposed";
    out.push(
      <div
        key={r.id}
        role="row"
        aria-level={r.stage ? 1 : 2}
        aria-expanded={r.hasChildren ? r.expanded : undefined}
        aria-selected={selected.has(r.id)}
        aria-rowindex={i + 2}
        data-id={r.id}
        className={`pg-row${r.stage ? " is-stage" : ""}${selected.has(r.id) ? " is-selected" : ""}${t?.critical ? " is-critical" : ""}`}
        style={{ height: rowH }}
        onClick={(e) => onSelect(r.id, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })}
      >
        {!narrow && (
          <span role="gridcell" data-col="number" className="pg-cell pg-num">
            {r.number}
          </span>
        )}
        <span role="gridcell" data-col="name" className="pg-cell pg-name">
          {r.hasChildren ? (
            <button
              type="button"
              className="pg-chevron"
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
          <span className="pg-label" title={r.act.label}>
            {r.act.label}
          </span>
          {proposed && (
            <span className="pg-proposed" title="Proposed: not yet accepted">
              <span className="sr-only">, proposed</span>
            </span>
          )}
          {narrow && unknown && <span className="pg-unknown-tag">?</span>}
        </span>
        {!narrow && (
          <>
            <span role="gridcell" data-col="duration" className={`pg-cell pg-dur${unknown ? " is-unknown" : ""}`}>
              {durationText(r, t)}
            </span>
            <span role="gridcell" data-col="start" className="pg-cell pg-date">
              {d.start}
            </span>
            <span role="gridcell" data-col="finish" className="pg-cell pg-date">
              {d.finish}
            </span>
          </>
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
