// Timescale layer (plan 9.2): the two-tier header, and in the body the
// period gridlines, holiday breaks and today line. Memoised on the state
// version, zoom and horizontal window; scrolling inside a window renders
// nothing new.

import { memo } from "react";
import type { Cal } from "../calendar";
import { breaks, crisp, ticks, xOf, type Scale, type Viewport } from "../layout";

export const SCALE_H = 44;

interface Common {
  cal: Cal;
  v: Viewport;
  /** Working-day index window [lo, hi) to draw ticks for. */
  lo: number;
  hi: number;
  scale: Scale;
  width: number;
  version: number;
}

export const TimescaleHeader = memo(function TimescaleHeader({ cal, v, lo, hi, scale, width }: Common) {
  const t = ticks(cal, lo, hi, scale);
  const half = SCALE_H / 2;
  return (
    <svg className="pg-scale-svg" width={width} height={SCALE_H} aria-hidden="true">
      <g className="pg-tier pg-major">
        {majorLabels(t.major, v).map((k) => (
          <g key={`M${k.index}`}>
            <line className="pg-tick" x1={crisp(xOf(k.index, v))} x2={crisp(xOf(k.index, v))} y1={0} y2={SCALE_H} />
            {k.show && (
              <text x={xOf(k.index, v) + 4} y={half - 7}>
                {k.label}
              </text>
            )}
          </g>
        ))}
      </g>
      <line className="pg-tick" x1={0} x2={width} y1={crisp(half)} y2={crisp(half)} />
      <g className="pg-tier pg-minor">
        {minorLabels(t.minor, v).map((k) => (
          <g key={`m${k.index}`}>
            <line className="pg-tick pg-tick-minor" x1={crisp(xOf(k.index, v))} x2={crisp(xOf(k.index, v))} y1={half} y2={SCALE_H} />
            {k.show && (
              <text x={xOf(k.index, v) + 3} y={SCALE_H - 7}>
                {k.label}
              </text>
            )}
          </g>
        ))}
      </g>
    </svg>
  );
});

// A major label runs right until the next period starts: hide it when the
// period is too narrow to hold it (the one before a month change, say).
function majorLabels(major: { index: number; label: string }[], v: Viewport) {
  return major.map((k, i) => {
    const next = major[i + 1];
    const room = next ? xOf(next.index, v) - xOf(k.index, v) : Infinity;
    return { ...k, show: room >= 7 * k.label.length + 8 };
  });
}

// Minor labels need room: skip a label that would overlap the one before.
function minorLabels(minor: { index: number; label: string }[], v: Viewport) {
  let lastX = -Infinity;
  return minor.map((k) => {
    const x = xOf(k.index, v);
    const show = x - lastX >= 7 * k.label.length + 6;
    if (show) lastX = x;
    return { ...k, show };
  });
}

interface GridLayerProps extends Common {
  plain: Cal;
  height: number;
  /** Working-day index of today, or null on a relative program. */
  today: number | null;
}

export const TimescaleGrid = memo(function TimescaleGrid({ cal, plain, v, lo, hi, scale, height, today }: GridLayerProps) {
  const t = ticks(cal, lo, hi, scale);
  const holes = breaks(cal, plain, lo, hi);
  return (
    <g className="pg-layer-scale" aria-hidden="true">
      {t.minor.map((k) => (
        <line key={k.index} className="pg-gridline" x1={crisp(xOf(k.index, v))} x2={crisp(xOf(k.index, v))} y1={0} y2={height} />
      ))}
      {t.major.map((k) => (
        <line key={`M${k.index}`} className="pg-gridline pg-gridline-major" x1={crisp(xOf(k.index, v))} x2={crisp(xOf(k.index, v))} y1={0} y2={height} />
      ))}
      {holes.map((i) => (
        <rect key={`h${i}`} className="pg-break" x={Math.round(xOf(i, v)) - 2} y={0} width={3} height={height}>
          <title>Non-working days before this day</title>
        </rect>
      ))}
      {today !== null && today >= lo && today < hi && (
        <line className="pg-today" x1={crisp(xOf(today, v))} x2={crisp(xOf(today, v))} y1={0} y2={height} />
      )}
    </g>
  );
});
