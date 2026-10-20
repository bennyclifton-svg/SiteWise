// Bars layer (plan 9.2): one bar per row in the render window. Colour says
// who owns the value and a second cue says it again: ember solid fill for a
// user or evidenced duration, azure light fill for library inference, amber
// dashed outline for a proposal, alert red 2 px outline for the critical
// path. A row with an unknown duration has no bar, only a dotted
// "duration unknown" marker. useDrag finds bars by data-id and moves them
// with transforms during a drag; nothing here holds drag state. Each bar has
// a transparent hit area covering its end zones and two link dots.

import { memo } from "react";
import type { Times } from "../types";
import { MILESTONE, barOf, labelPlacement, xOf, type Row, type Viewport } from "../layout";
import { fmtDate } from "../layout";

export interface BarsProps {
  rows: readonly Row[];
  times: Record<string, Times>;
  v: Viewport;
  first: number;
  last: number;
  selected: ReadonlySet<string>;
  /** Rows held against their own constraint (computed.conflicts). */
  conflicts: ReadonlySet<string>;
  relative: boolean;
  version: number;
  /** Row id → label of the row driving its start, for "held by Slab" (plan 9.5). */
  drivers: ReadonlyMap<string, string>;
  /** End-zone width: the hit area reaches this far past each end (plan 9.4). */
  zone: number;
}

// The shell sets labels in var(--sans) (Instrument Sans; IBM Plex Sans is not
// bundled), so measure with the same face.
const LABEL_FONT = '12px "Instrument Sans", sans-serif';
const widths = new Map<string, number>();
let ctx: CanvasRenderingContext2D | null | undefined;

/** Label width from one cached canvas; a rough estimate where canvas is missing. */
export function textWidth(s: string): number {
  const hit = widths.get(s);
  if (hit !== undefined) return hit;
  if (ctx === undefined) {
    try {
      ctx = document.createElement("canvas").getContext("2d");
      if (ctx) ctx.font = LABEL_FONT;
    } catch {
      ctx = null;
    }
  }
  const w = ctx ? ctx.measureText(s).width : s.length * 6.5;
  widths.set(s, w);
  return w;
}

const OWNED = new Set(["user", "document", "reading", "evidence", "profile"]);

/** Ember for user or evidenced values, azure for library inference (plan 9.2). */
export function ownerClass(origin: string, durationOrigin: string | null): "o-user" | "o-library" {
  return origin === "user" || OWNED.has(durationOrigin ?? "") ? "o-user" : "o-library";
}

function ariaLabel(r: Row, t: Times | undefined, relative: boolean, held: string | undefined): string {
  const parts = [r.act.label];
  if (t && !t.unknown && t.es !== null && t.ef !== null) {
    if (!relative && t.start) parts.push(t.finish ? `${fmtDate(t.start)} to ${fmtDate(t.finish)}` : fmtDate(t.start));
    else parts.push(`${t.ef - t.es} working days`);
  } else parts.push("duration unknown");
  if (t?.critical) parts.push("critical");
  if (r.act.review_status === "proposed") parts.push("proposed");
  if (held) parts.push(`held by ${held}`);
  return parts.join(", ");
}

export const Bars = memo(function Bars({ rows, times, v, first, last, selected, conflicts, relative, drivers, zone }: BarsProps) {
  const out = [];
  const h = v.rowH;
  for (let i = first; i < last; i++) {
    const r = rows[i];
    const t = times[r.id];
    const b = barOf(t, v);
    const y = i * h;
    const mid = y + h / 2;
    if (!b) {
      // No bar: say so where the row would start (its stage's start, else the program start).
      const stageT = r.parentId ? times[r.parentId] : undefined;
      const x = stageT && stageT.es !== null ? xOf(stageT.es, v) : xOf(0, v);
      out.push(
        <g key={r.id} className="pg-unknown" data-id={r.id} role="img" aria-label={ariaLabel(r, t, relative, undefined)}>
          <rect x={Math.round(x) + 0.5} y={Math.round(mid - 6) + 0.5} width={14} height={12} rx={2} />
          <text x={x + 7} y={mid + 4} textAnchor="middle">?</text>
          <text className="pg-unknown-text" x={x + 20} y={mid + 4}>duration unknown</text>
        </g>,
      );
      continue;
    }
    const cls = [
      "pg-bar",
      r.hasChildren ? "is-summary" : "",
      r.act.kind === "milestone" || b.milestone ? "is-milestone" : "",
      r.act.kind === "hold_point" ? "is-hold" : "",
      ownerClass(r.act.origin, r.act.duration_origin),
      r.act.review_status === "proposed" ? "is-proposed" : "",
      t?.critical ? "is-critical" : "",
      selected.has(r.id) ? "is-selected" : "",
      conflicts.has(r.id) ? "is-conflict" : "",
    ]
      .filter(Boolean)
      .join(" ");
    const label = r.act.label;
    // A stage band is too thin for text inside it.
    const place = r.hasChildren ? { x: b.x + b.w + 6, inside: false } : labelPlacement(textWidth(label), b);
    const x = Math.round(b.x) + 0.5;
    let shape;
    if (b.milestone) {
      const s = MILESTONE / 2;
      shape = <path className="pg-shape" d={`M${b.x},${mid - s}L${b.x + s},${mid}L${b.x},${mid + s}L${b.x - s},${mid}Z`} />;
    } else if (r.hasChildren) {
      // A stage spanning its activities: a thin band with end ticks.
      const top = Math.round(y + h * 0.3) + 0.5;
      const bh = Math.max(4, Math.round(h * 0.22));
      const w = Math.max(1, Math.round(b.w) - 1);
      shape = <path className="pg-shape" d={`M${x},${top}h${w}v${bh + 4}l-4,-4H${x + 4}l-4,4Z`} />;
    } else {
      const top = Math.round(y + h * 0.22) + 0.5;
      shape = <rect className="pg-shape" x={x} y={top} width={Math.max(1, Math.round(b.w) - 1)} height={Math.round(h * 0.56) - 1} rx={2} />;
    }
    // Hit area: the row band from one end zone past the start to one past the finish.
    const [hx, hw] = b.milestone ? [b.x - MILESTONE / 2, MILESTONE] : [b.x, b.w];
    // End zones sit outside a narrow bar; the link dots sit beyond them, and
    // the hit area reaches past the dots so moving onto one keeps the hover.
    const reach = hw >= 3 * zone ? 0 : zone;
    const dotOff = reach + 6;
    const out1 = reach + 12;
    out.push(
      <g key={r.id} className={cls} data-id={r.id} data-x={b.x} role="button" tabIndex={-1} aria-label={ariaLabel(r, t, relative, drivers.get(r.id))}>
        <rect className="pg-hit" x={hx - out1} y={y + 2} width={hw + 2 * out1} height={h - 4} />
        {shape}
        {r.act.kind === "hold_point" && !b.milestone && (
          <path className="pg-hold-mark" d={`M${b.x},${mid - 4}l4,4l-4,4l-4,-4Z`} />
        )}
        <text className={place.inside ? "pg-label-in" : "pg-label-out"} x={b.milestone ? b.x + MILESTONE : place.x} y={mid + 4}>
          {label}
        </text>
        {/* Link dots (plan 9.4): shown on hover, or once selected on touch. */}
        <circle className="pg-dot" data-dot="S" cx={hx - dotOff} cy={mid} r={4} />
        <circle className="pg-dot" data-dot="F" cx={hx + hw + dotOff} cy={mid} r={4} />
      </g>,
    );
  }
  return <g className="pg-layer-bars">{out}</g>;
});
