// Fit to screen, fit to selection and fit height (plan 9.6). Browser only,
// pure and cheap: each is a handful of arithmetic over an extent the caller
// already has, so all three finish well inside one frame.

import { ROW_H, ROW_H_MIN, type Scale } from "./layout";

export const PX_MIN = 0.05;
export const PX_MAX = 96;
export const PX_DEFAULT = 12;
/** Room kept free at the chart's right edge (plan 9.6). */
const MARGIN = 48;

export function clampPx(px: number): number {
  return Math.min(PX_MAX, Math.max(PX_MIN, px));
}

/**
 * Room kept free at the chart's left edge, beyond the one day before the
 * first bar: the start link dot sits past the bar's end zone (8 px fine,
 * 24 px coarse, plus 6 px and the 4 px dot), so at a fitted zoom a bar
 * starting on day 0 would otherwise put its dot under the grid divider.
 */
export const LEFT_ROOM = 40;

/**
 * pxPerDay = (chartWidth − 48 − 40) / (last − first + 2) and viewStart =
 * first − 1 − 40 / pxPerDay, over an extent [first, last) of working-day
 * indices: one day and 40 px free on the left, 48 px on the right.
 */
export function fitSpan(span: [number, number], chartWidth: number): { pxPerDay: number; viewStart: number } {
  const [first, last] = span;
  const days = Math.max(1, last - first) + 2;
  const pxPerDay = clampPx(Math.max(1, chartWidth - MARGIN - LEFT_ROOM) / days);
  return { pxPerDay, viewStart: first - 1 - LEFT_ROOM / pxPerDay };
}

/** Timescale tier from the zoom: ≥ 24 px a day shows days, ≥ 4 weeks, ≥ 1 months, else quarters. */
export function scaleFor(pxPerDay: number): Scale {
  if (pxPerDay >= 24) return "days";
  if (pxPerDay >= 4) return "weeks";
  if (pxPerDay >= 1) return "months";
  return "quarters";
}

/**
 * Row height that fits every visible row: floor(viewportH / rows) clamped to
 * 20–28 px. tooMany when even 20 px rows overflow.
 */
export function fitHeight(viewportH: number, rows: number): { rowH: number; tooMany: boolean } {
  if (rows <= 0) return { rowH: ROW_H, tooMany: false };
  const h = Math.floor(viewportH / rows);
  return { rowH: Math.min(ROW_H, Math.max(ROW_H_MIN, h)), tooMany: h < ROW_H_MIN };
}

/** Zoom by a factor about an index kept under the same x (the view centre). */
export function zoomAbout(pxPerDay: number, viewStart: number, factor: number, anchorIndex: number): { pxPerDay: number; viewStart: number } {
  const next = clampPx(pxPerDay * factor);
  const x = (anchorIndex - viewStart) * pxPerDay;
  return { pxPerDay: next, viewStart: anchorIndex - x / next };
}
