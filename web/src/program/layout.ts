// Chart geometry as pure functions (plan 9.2). The x axis is the working-day
// index (plan 6.1): non-working days take no width. Nothing here touches the
// DOM, so every rule is unit-tested.

import { Cal, dayString } from "./calendar";
import type { Times } from "./types";
import type { ProgramActivity, ProgramLink } from "../programApi";

export const ROW_H = 28;
export const ROW_H_MIN = 20;
export const MILESTONE = 10;
export const WINDOW_PAD = 10;

/** One grid and chart row in the current view (collapsed or expanded). */
export interface Row {
  id: string;
  /** Position in the full outline, 1-based: the number a typed predecessor refers to. */
  number: number;
  stage: boolean;
  parentId: string | null;
  hasChildren: boolean;
  expanded: boolean;
  act: ProgramActivity;
}

/**
 * The rows the view shows, in outline order: every stage, and the
 * activities of expanded stages (plan 7.5). Activities arrive in outline
 * order from the server (each stage, then its children).
 */
export function outlineRows(acts: readonly ProgramActivity[], expanded: ReadonlySet<string>): Row[] {
  const kids = new Set<string>();
  for (const a of acts) if (a.parent_id) kids.add(a.parent_id);
  const out: Row[] = [];
  acts.forEach((a, i) => {
    const stage = !a.parent_id;
    if (!stage && !expanded.has(a.parent_id as string)) return;
    out.push({
      id: a.id,
      number: i + 1,
      stage,
      parentId: a.parent_id,
      hasChildren: stage && kids.has(a.id),
      expanded: stage && expanded.has(a.id),
      act: a,
    });
  });
  return out;
}

/** Ids of stages that have activities: what Expand all opens. */
export function stagesWithChildren(acts: readonly ProgramActivity[]): string[] {
  const kids = new Set<string>();
  for (const a of acts) if (a.parent_id) kids.add(a.parent_id);
  return acts.filter((a) => !a.parent_id && kids.has(a.id)).map((a) => a.id);
}

export interface Viewport {
  pxPerDay: number;
  /** Working-day index at x = 0. */
  viewStart: number;
  rowH: number;
}

export function xOf(index: number, v: Viewport): number {
  return (index - v.viewStart) * v.pxPerDay;
}

/** Inverse of xOf, for Today and scroll anchoring. */
export function indexAt(x: number, v: Viewport): number {
  return x / v.pxPerDay + v.viewStart;
}

/** Half-pixel offset so a 1 px line lands on one device pixel (plan 9.2). */
export function crisp(x: number): number {
  return Math.round(x) + 0.5;
}

export interface BarGeom {
  x: number;
  w: number;
  milestone: boolean;
}

/**
 * A row's bar from its computed times, or null when the duration is unknown.
 * Width is at least 2 px; a zero-length row is a 10 px milestone diamond
 * centred on its start.
 */
export function barOf(t: Times | undefined, v: Viewport): BarGeom | null {
  if (!t || t.unknown || t.es === null || t.ef === null) return null;
  const x = xOf(t.es, v);
  if (t.ef - t.es <= 0) return { x, w: MILESTONE, milestone: true };
  return { x, w: Math.max((t.ef - t.es) * v.pxPerDay, 2), milestone: false };
}

/**
 * Rows to render for a scroll position: from floor(top / rowH) − 10 to
 * ceil((top + height) / rowH) + 10, clamped; last is exclusive (plan 9.2).
 */
export function rowWindow(scrollTop: number, viewportH: number, rowH: number, count: number, pad = WINDOW_PAD): [number, number] {
  const first = Math.max(0, Math.floor(scrollTop / rowH) - pad);
  const last = Math.min(count, Math.ceil((scrollTop + viewportH) / rowH) + pad);
  return [first, Math.max(first, last)];
}

/** Bar label inside the bar when it fits with 6 px each side, else just after its end. */
export function labelPlacement(textW: number, bar: BarGeom, pad = 6): { x: number; inside: boolean } {
  if (!bar.milestone && textW + 2 * pad <= bar.w) return { x: bar.x + pad, inside: true };
  return { x: bar.x + bar.w + pad, inside: false };
}

/** One drawn link: its row positions and whether it stands in for hidden activities. */
export interface DrawnLink {
  link: ProgramLink;
  pred: number;
  succ: number;
  /** An end was folded onto its collapsed stage: drawn faint (plan 7.5). */
  faint: boolean;
}

/**
 * Links to draw for the visible rows and window. An end inside a collapsed
 * stage is drawn on that stage's bar; links folded onto the same pair of
 * rows draw once, and a link folded onto one row is not drawn. A link is
 * drawn only when either end is inside [first, last) (plan 9.2).
 */
export function drawnLinks(
  links: readonly ProgramLink[],
  rows: readonly Row[],
  parentOf: ReadonlyMap<string, string | null>,
  first = 0,
  last = Infinity,
): DrawnLink[] {
  const at = new Map<string, number>();
  rows.forEach((r, i) => at.set(r.id, i));
  const place = (id: string): [number, boolean] | null => {
    const i = at.get(id);
    if (i !== undefined) return [i, false];
    const p = parentOf.get(id);
    const j = p ? at.get(p) : undefined;
    return j === undefined ? null : [j, true];
  };
  const seen = new Set<string>();
  const out: DrawnLink[] = [];
  for (const l of links) {
    const a = place(l.pred_id);
    const b = place(l.succ_id);
    if (!a || !b || a[0] === b[0]) continue;
    const inWindow = (a[0] >= first && a[0] < last) || (b[0] >= first && b[0] < last);
    if (!inWindow) continue;
    const faint = a[1] || b[1];
    if (faint) {
      const key = `${a[0]}:${b[0]}`;
      if (seen.has(key)) continue;
      seen.add(key);
    }
    out.push({ link: l, pred: a[0], succ: b[0], faint });
  }
  return out;
}

/**
 * The index span [lo, hi) of the known rows given, widened to the high
 * finish when it is later. Null when no row has known times.
 */
export function extent(ids: Iterable<string>, times: Record<string, Times>, highFinish: number | null = null): [number, number] | null {
  let lo = Infinity;
  let hi = -Infinity;
  for (const id of ids) {
    const t = times[id];
    if (!t || t.es === null || t.ef === null) continue;
    lo = Math.min(lo, t.es);
    hi = Math.max(hi, t.ef, t.es + 1);
  }
  if (lo === Infinity) return null;
  if (highFinish !== null) hi = Math.max(hi, highFinish);
  return [lo, hi];
}

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

/** "2027-05-12" → "12 May 2027"; display only, never parsed back. */
export function fmtDate(ymd: string | null | undefined): string {
  if (!ymd || ymd.length !== 10) return "";
  return `${+ymd.slice(8, 10)} ${MONTHS[+ymd.slice(5, 7) - 1]} ${ymd.slice(0, 4)}`;
}

/**
 * Shown start and finish of a row: the server's dates on a dated program,
 * "Week N, day M" on a relative one (plan 6.1: the finish is the last
 * working day, start + duration − 1; a milestone shows its start only).
 */
export function shownDates(cal: Cal, t: Times | undefined): { start: string; finish: string } {
  if (!t || t.unknown || t.es === null || t.ef === null) return { start: "", finish: "" };
  if (!cal.relative) return { start: fmtDate(t.start), finish: fmtDate(t.finish) };
  return { start: cal.dayLabel(t.es), finish: t.ef - t.es > 0 ? cal.dayLabel(t.ef - 1) : "" };
}

/** The finish shown for the project: a date, or "Week N, day M" on a relative program. */
export function shownFinish(cal: Cal, date: string | null, index: number | null): string {
  if (date) return fmtDate(date);
  if (index === null) return "";
  return cal.dayLabel(Math.max(0, index - 1));
}

export type Scale = "days" | "weeks" | "months" | "quarters";

export interface Tick {
  index: number;
  label: string;
}

/**
 * Timescale ticks over [lo, hi): a minor tier and the major tier above it
 * (days under months, weeks under months, months under years, quarters
 * under years). A relative program has no months, so it counts weeks.
 * A tick sits on the first working day of its period.
 */
export function ticks(cal: Cal, lo: number, hi: number, scale: Scale): { major: Tick[]; minor: Tick[] } {
  const major: Tick[] = [];
  const minor: Tick[] = [];
  lo = Math.floor(lo);
  hi = Math.ceil(hi);
  let prevMajor: number | null = null;
  let prevMinor: number | null = null;
  for (let i = lo; i < hi; i++) {
    const off = cal.dayOf(i);
    const [majorKey, majorLabel, minorKey, minorLabel] = cal.relative ? relativeKeys(cal, i, off, scale) : datedKeys(cal.origin + off, scale);
    if (majorKey !== prevMajor) major.push({ index: i, label: majorLabel });
    if (minorKey !== prevMinor) minor.push({ index: i, label: minorLabel });
    prevMajor = majorKey;
    prevMinor = minorKey;
  }
  return { major, minor };
}

function datedKeys(n: number, scale: Scale): [number, string, number, string] {
  const s = dayString(n);
  const y = +s.slice(0, 4);
  const m = +s.slice(5, 7);
  const d = +s.slice(8, 10);
  const q = Math.floor((m - 1) / 3);
  switch (scale) {
    case "days":
      return [y * 12 + m, `${MONTHS[m - 1]} ${y}`, n, String(d)];
    case "weeks": {
      // 1970-01-05 (day 4) was a Monday: weeks start on Monday.
      const wk = Math.floor((n - 4) / 7);
      return [y * 12 + m, `${MONTHS[m - 1]} ${y}`, wk, String(d)];
    }
    case "months":
      return [y, String(y), y * 12 + m, MONTHS[m - 1]];
    default:
      return [y, String(y), y * 4 + q, `Q${q + 1}`];
  }
}

function relativeKeys(cal: Cal, i: number, off: number, scale: Scale): [number, string, number, string] {
  const week = Math.floor(off / 7);
  switch (scale) {
    case "days": {
      const day = cal.count(off + 1) - cal.count(week * 7);
      return [week, `Week ${week + 1}`, i, `d${day}`];
    }
    case "weeks":
      return [Math.floor(week / 4), `Week ${4 * Math.floor(week / 4) + 1}`, week, `W${week + 1}`];
    case "months":
      return [Math.floor(week / 13), `Week ${13 * Math.floor(week / 13) + 1}`, Math.floor(week / 4), `W${4 * Math.floor(week / 4) + 1}`];
    default:
      return [Math.floor(week / 52), `Year ${Math.floor(week / 52) + 1}`, Math.floor(week / 13), `W${13 * Math.floor(week / 13) + 1}`];
  }
}

/**
 * Indices in [lo, hi) that come right after a public holiday, RDO or
 * shutdown: the gap before them holds a day the plain working week would
 * work. `plain` is the same calendar without its exception days.
 */
export function breaks(cal: Cal, plain: Cal, lo: number, hi: number): number[] {
  if (cal.relative) return [];
  const out: number[] = [];
  for (let i = Math.max(lo, 1); i < hi; i++) {
    const prev = cal.dayOf(i - 1);
    const cur = cal.dayOf(i);
    if (cur - prev > 1 && plain.count(cur) - plain.count(prev + 1) > 0) out.push(i);
  }
  return out;
}
