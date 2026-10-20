// Links layer (plan 9.3): driving links solid, non-driving dashed, critical
// 2 px in the critical colour, a diamond at the midpoint of a link into or
// out of a hold point, the lag on hover. Under each path a transparent
// 10 px stroke is the hit target that opens the link popover (data-id).

import { memo } from "react";
import type { Times } from "../types";
import { xOf, type DrawnLink, type Row, type Viewport } from "../layout";
import { anchorEnds, route } from "../routing";

export interface LinksProps {
  drawn: readonly DrawnLink[];
  rows: readonly Row[];
  times: Record<string, Times>;
  v: Viewport;
  driving: ReadonlySet<string>;
  version: number;
}

export function lagText(lag: number, unit: string): string {
  if (lag === 0) return "No lag";
  const sign = lag > 0 ? "+" : "−";
  const n = Math.abs(lag);
  return `${sign}${n} ${unit === "calendar" ? "calendar" : "working"} day${n === 1 ? "" : "s"}`;
}

export const Links = memo(function Links({ drawn, rows, times, v, driving }: LinksProps) {
  const h = v.rowH;
  const out = [];
  for (const d of drawn) {
    const pt = times[rows[d.pred].id];
    const st = times[rows[d.succ].id];
    // A link with an unknown end has no bar to attach to.
    if (!pt || !st || pt.es === null || pt.ef === null || st.es === null || st.ef === null) continue;
    const [pe, se] = anchorEnds(d.link.type);
    const r = route({
      px: xOf(pe === "F" ? pt.ef : pt.es, v),
      py: d.pred * h + h / 2,
      sx: xOf(se === "F" ? st.ef : st.es, v),
      sy: d.succ * h + h / 2,
      type: d.link.type,
      rowH: h,
    });
    const isDriving = driving.has(d.link.id);
    const critical = !d.faint && isDriving && pt.critical && st.critical;
    const hold = rows[d.pred].act.kind === "hold_point" || rows[d.succ].act.kind === "hold_point";
    const name = `${rows[d.pred].act.label} → ${rows[d.succ].act.label}, ${d.link.type}, ${lagText(d.link.lag_days, d.link.lag_unit)}${isDriving ? ", driving" : ""}`;
    const cls = ["pg-link", isDriving ? "is-driving" : "is-free", critical ? "is-critical" : "", d.faint ? "is-faint" : ""].filter(Boolean).join(" ");
    out.push(
      <g key={d.link.id} className={cls} data-id={d.link.id} data-link="1" role="button" tabIndex={-1} aria-label={`Link ${name}`}>
        <title>{name}</title>
        <path className="pg-link-hit" d={r.d} />
        <path className="pg-link-line" d={r.d} />
        <path className="pg-link-arrow" d={r.arrow} />
        {hold && <path className="pg-link-hold" d={`M${r.mid.x},${r.mid.y - 4}l4,4l-4,4l-4,-4Z`} />}
        {d.link.lag_days !== 0 && (
          <text className="pg-link-lag" x={r.mid.x + 6} y={r.mid.y - 4}>
            {lagText(d.link.lag_days, d.link.lag_unit)}
          </text>
        )}
      </g>,
    );
  }
  return <g className="pg-layer-links">{out}</g>;
});
