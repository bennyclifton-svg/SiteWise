// Pure editing helpers for WP-P8 (plan 9.4, 9.5): hit zones, snapping, the
// drag preview input, held rows, op-batch builders and the grid drop target.
// Nothing here touches the DOM, so each rule is unit-tested; the chart and
// grid only measure pointers and write what these return.

import type { Cal } from "./calendar";
import { formatPredecessor, type PredRef } from "./grid/parse";
import { barOf, type Row, type Viewport } from "./layout";
import type { Input, Output, Times } from "./types";
import type { ProgramActivity, ProgramLink, ProgramOp } from "../programApi";

/** End-zone width by pointer kind (plan 9.4). */
export const ZONE_FINE = 8;
export const ZONE_COARSE = 24;
/** Movement before a press becomes a drag, so a click stays a click. */
export const DRAG_THRESHOLD = 4;
/** Touch long-press before a drag starts, so one-finger scroll still scrolls. */
export const LONG_PRESS_MS = 350;

export type Zone = "start" | "end" | "body";

/**
 * Which part of a bar a pointer x is on. Zones sit inside a bar wide enough
 * for three of them, otherwise just outside its ends. A stage spanning its
 * activities only moves; a milestone only grows from its right side.
 */
export function zoneAt(x: number, bx: number, bw: number, z: number, o: { milestone?: boolean; summary?: boolean } = {}): Zone {
  if (o.summary) return "body";
  if (o.milestone) return x > bx + bw / 2 ? "end" : "body";
  if (bw >= 3 * z) {
    if (x < bx + z) return "start";
    if (x > bx + bw - z) return "end";
    return "body";
  }
  if (x < bx) return "start";
  if (x > bx + bw) return "end";
  return "body";
}

/** Snap unit in working days: a day when pxPerDay ≥ 12 or Alt is held, else a week. */
export function snapUnit(pxPerDay: number, alt: boolean, perWeek: number): number {
  return alt || pxPerDay >= 12 ? 1 : Math.max(1, perWeek);
}

/** A pointer offset in px as a whole number of snap units, in working days. */
export function snapDays(dx: number, pxPerDay: number, unit: number): number {
  const d = Math.round(dx / pxPerDay / unit) * unit;
  return d === 0 ? 0 : d; // no −0
}

/** Link type from the dragged end of the predecessor and the end dropped on. */
export function linkType(from: "S" | "F", to: "S" | "F"): string {
  return from + to === "FF" ? "FF" : from + to === "SS" ? "SS" : from + to === "SF" ? "SF" : "FS";
}

/** Stage id → its activity ids, in outline order. */
export function kidsOf(acts: readonly ProgramActivity[]): Map<string, string[]> {
  const m = new Map<string, string[]>();
  for (const a of acts) {
    if (!a.parent_id) continue;
    const k = m.get(a.parent_id);
    if (k) k.push(a.id);
    else m.set(a.parent_id, [a.id]);
  }
  return m;
}

/** The rows a move shifts: a stage with activities moves its activities (plan 8.1). */
export function moveTargets(ids: readonly string[], kids: ReadonlyMap<string, string[]>): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const id of ids) {
    for (const t of kids.get(id) ?? [id]) {
      if (!seen.has(t)) (seen.add(t), out.push(t));
    }
  }
  return out;
}

/**
 * The engine input after moving targets by delta working days: each target
 * gets a start-no-earlier-than its new start (an MSO keeps its kind and takes
 * the new date), as the server's move does (owner answer Q2).
 */
export function withMoves(input: Input, times: Record<string, Times>, targets: readonly string[], delta: number, cal: Cal): Input {
  const want = new Map<string, number>();
  for (const id of targets) {
    const t = times[id];
    if (t && t.es !== null) want.set(id, t.es + delta);
  }
  return {
    ...input,
    activities: (input.activities ?? []).map((a) => {
      const to = want.get(a.id);
      if (to === undefined) return a;
      return { ...a, constraint: a.constraint === "mso" ? "mso" : "snet", constraint_date: cal.dateOf(Math.max(0, to)) };
    }),
  };
}

/** The engine input after one row's duration and, from its left end, its start change. */
export function withResize(input: Input, id: string, duration: number, start: number | null, cal: Cal): Input {
  return {
    ...input,
    activities: (input.activities ?? []).map((a) => {
      if (a.id !== id) return a;
      const b = { ...a, duration, low: null, high: null };
      if (b.kind === "milestone" && duration > 0) b.kind = "activity";
      if (start !== null) Object.assign(b, { constraint: a.constraint === "mso" ? "mso" : "snet", constraint_date: cal.dateOf(Math.max(0, start)) });
      return b;
    }),
  };
}

/** Targets whose preview start is not where they were asked to go: held by a link or the program start. */
export function heldRows(targets: readonly string[], times: Record<string, Times>, out: Output, delta: number): string[] {
  const held: string[] = [];
  for (const id of targets) {
    const t = times[id];
    if (!t || t.es === null) continue;
    const want = t.es + delta;
    if (want < 0 || out.activities[id]?.es !== want) held.push(id);
  }
  return held;
}

/** "Held by Slab", or every held row named, as the server says it (store heldError). */
export function heldMessage(held: readonly string[], times: Record<string, Times>, out: Output, delta: number, acts: readonly ProgramActivity[], links: readonly ProgramLink[]): string {
  const byId = new Map(acts.map((a) => [a.id, a]));
  const linkById = new Map(links.map((l) => [l.id, l]));
  const by = (id: string) => {
    const t = times[id];
    if (t && t.es !== null && t.es + delta < 0) return "the program start";
    for (const lid of out.driving_links) {
      const l = linkById.get(lid);
      if (l && l.succ_id === id) return byId.get(l.pred_id)?.label ?? "a link";
    }
    const p = byId.get(id)?.parent_id;
    return p ? (byId.get(p)?.label ?? "its stage") : "the program start";
  };
  if (held.length === 1) return `Held by ${by(held[0])}`;
  return held.map((id) => `${byId.get(id)?.label ?? "A row"} is held by ${by(id)}`).join("; ") + ".";
}

/** "+9 days to completion" from the finish before and after a preview. */
export function completionBadge(before: number | null, after: number | null): string {
  if (before === null || after === null) return "Completion not known yet";
  const d = after - before;
  if (d === 0) return "No change to completion";
  return `${d > 0 ? "+" : "−"}${Math.abs(d)} day${Math.abs(d) === 1 ? "" : "s"} to completion`;
}

/** Outline position (1-based row number) of every row, as the grid's # column shows it. */
export function rowNumbers(acts: readonly ProgramActivity[]): Map<string, number> {
  return new Map(acts.map((a, i) => [a.id, i + 1]));
}

/** A row's predecessor cell in the typed format: "3, 7SS+2d". */
export function predecessorText(id: string, links: readonly ProgramLink[], numbers: ReadonlyMap<string, number>): string {
  return predecessorTexts(links.filter((l) => l.succ_id === id), numbers).get(id) ?? "";
}

/** Every row's predecessor cell, in one pass over the links. */
export function predecessorTexts(links: readonly ProgramLink[], numbers: ReadonlyMap<string, number>): Map<string, string> {
  const into = new Map<string, PredRef[]>();
  for (const l of links) {
    const row = numbers.get(l.pred_id);
    if (row === undefined) continue;
    const r = { row, type: l.type, lag: l.lag_days, lag_unit: l.lag_unit };
    const list = into.get(l.succ_id);
    if (list) list.push(r);
    else into.set(l.succ_id, [r]);
  }
  const out = new Map<string, string>();
  for (const [id, refs] of into) out.set(id, refs.sort((a, b) => a.row - b.row).map(formatPredecessor).join(", "));
  return out;
}

/**
 * The ops that make a row's predecessors what was typed: adds for new rows,
 * updates where the type or lag changed, deletes for rows no longer listed.
 */
export function predecessorOps(id: string, refs: readonly PredRef[], acts: readonly ProgramActivity[], links: readonly ProgramLink[]): { ops: ProgramOp[] } | { error: string } {
  const existing = new Map(links.filter((l) => l.succ_id === id).map((l) => [l.pred_id, l]));
  const ops: ProgramOp[] = [];
  const kept = new Set<string>();
  for (const r of refs) {
    const pred = acts[r.row - 1];
    if (!pred) return { error: `There is no row ${r.row}.` };
    if (pred.id === id) return { error: "A row can't follow itself." };
    kept.add(pred.id);
    const l = existing.get(pred.id);
    if (!l) ops.push({ op: "link_add", pred_id: pred.id, succ_id: id, type: r.type, lag_days: r.lag, lag_unit: r.lag_unit });
    else if (l.type !== r.type || l.lag_days !== r.lag || l.lag_unit !== r.lag_unit) ops.push({ op: "link_update", id: l.id, type: r.type, lag_days: r.lag, lag_unit: r.lag_unit });
  }
  const gone = [...existing.values()].filter((l) => !kept.has(l.pred_id)).map((l) => l.id);
  if (gone.length) ops.push({ op: "link_delete", ids: gone });
  return { ops };
}

export interface DeletePlan {
  rows: number;
  links: number;
  /** Stages that take activities with them: these need a confirm. */
  stages: string[];
  /** Hold points removed: these need a reason (plan 13.1). */
  holds: string[];
}

/** What deleting a selection removes, for the confirm (plan 9.4, 13.1). */
export function deletePlan(ids: readonly string[], acts: readonly ProgramActivity[], links: readonly ProgramLink[]): DeletePlan {
  const kids = kidsOf(acts);
  const byId = new Map(acts.map((a) => [a.id, a]));
  const gone = new Set<string>();
  const stages: string[] = [];
  for (const id of ids) {
    if (!byId.has(id)) continue;
    gone.add(id);
    const k = kids.get(id) ?? [];
    if (k.length) stages.push(byId.get(id)!.label);
    for (const c of k) gone.add(c);
  }
  const holds = [...gone].filter((id) => byId.get(id)?.kind === "hold_point").map((id) => byId.get(id)!.label);
  const n = links.filter((l) => gone.has(l.pred_id) || gone.has(l.succ_id)).length;
  return { rows: gone.size, links: n, stages, holds };
}

/** Ids in outline (grid) order, for Link in sequence and reorder. */
export function inOutline(ids: Iterable<string>, acts: readonly ProgramActivity[]): string[] {
  const want = new Set(ids);
  return acts.filter((a) => want.has(a.id)).map((a) => a.id);
}

/**
 * Alt+Up / Alt+Down: the reorder op moving the selection one place (plan
 * 9.5). Stages move among stages; activities move among their siblings and
 * past the first or last one into the stage above or below. Null when there
 * is nowhere to go or the selection mixes stages and activities.
 */
export function reorderStep(ids: readonly string[], acts: readonly ProgramActivity[], dir: -1 | 1): ProgramOp | null {
  const sel = inOutline(ids, acts);
  if (!sel.length) return null;
  const byId = new Map(acts.map((a) => [a.id, a]));
  const stages = acts.filter((a) => !a.parent_id).map((a) => a.id);
  const isStage = (id: string) => !byId.get(id)!.parent_id;
  if (sel.every(isStage)) {
    const step = slide(stages, sel, dir);
    return step === null ? null : { op: "reorder", ids: sel, after_id: step || undefined };
  }
  if (sel.some(isStage)) return null;
  const parent = byId.get(sel[0])!.parent_id as string;
  if (sel.some((id) => byId.get(id)!.parent_id !== parent)) return null;
  const kids = kidsOf(acts);
  const step = slide(kids.get(parent) ?? [], sel, dir);
  if (step !== null) return { op: "reorder", ids: sel, parent_id: parent, after_id: step || undefined };
  // Past the end of the stage: into the neighbouring stage.
  const s = stages[stages.indexOf(parent) + dir];
  if (!s) return null;
  if (dir < 0) {
    const k = kids.get(s) ?? [];
    return { op: "reorder", ids: sel, parent_id: s, after_id: k[k.length - 1] };
  }
  return { op: "reorder", ids: sel, parent_id: s };
}

// The id to place a block after when it slides one place in seq ("" for
// first), or null when it cannot move that way.
function slide(seq: readonly string[], block: readonly string[], dir: -1 | 1): string | null {
  const moving = new Set(block);
  const rest = seq.filter((x) => !moving.has(x));
  const before = seq.slice(0, seq.indexOf(block[0])).filter((x) => !moving.has(x)).length;
  const to = before + dir;
  if (to < 0 || to > rest.length) return null;
  return to === 0 ? "" : rest[to - 1];
}

export interface Drop {
  /** Insertion line at this row boundary, or -1 when dropping onto a stage row. */
  line: number;
  /** Row index highlighted when dropping into a stage. */
  into: number;
  op: ProgramOp;
}

/**
 * Where dragged grid rows land for a pointer y (plan 9.4): stages between
 * stages; activities between activities, or onto a stage row's middle to go
 * to its end. Null when the drop would change nothing or is not allowed.
 */
export function dropTarget(rows: readonly Row[], drag: readonly string[], y: number, rowH: number, acts: readonly ProgramActivity[]): Drop | null {
  const moving = new Set(drag);
  const kids = kidsOf(acts);
  const stages = drag.every((id) => !acts.find((a) => a.id === id)?.parent_id);
  const i = Math.floor(y / rowH);
  const frac = y / rowH - i;
  if (!stages && i >= 0 && i < rows.length && rows[i].stage && frac > 0.25 && frac < 0.75 && !moving.has(rows[i].id)) {
    const k = (kids.get(rows[i].id) ?? []).filter((x) => !moving.has(x));
    return { line: -1, into: i, op: { op: "reorder", ids: [...drag], parent_id: rows[i].id, after_id: k[k.length - 1] } };
  }
  const line = Math.max(0, Math.min(rows.length, Math.round(y / rowH)));
  if (stages) {
    // After the stage of the row above the line, skipping dragged stages.
    let after = "";
    for (let j = line - 1; j >= 0; j--) {
      const s = rows[j].stage ? rows[j].id : (rows[j].parentId as string);
      if (!moving.has(s)) {
        after = s;
        break;
      }
    }
    return { line, into: -1, op: { op: "reorder", ids: [...drag], after_id: after || undefined } };
  }
  if (line === 0) return null; // nothing holds an activity above the first stage
  for (let j = line - 1; j >= 0; j--) {
    const r = rows[j];
    if (r.stage) return { line, into: -1, op: { op: "reorder", ids: [...drag], parent_id: r.id } };
    if (!moving.has(r.id)) return { line, into: -1, op: { op: "reorder", ids: [...drag], parent_id: r.parentId as string, after_id: r.id } };
  }
  return null;
}

/** Rows whose bars a selection box [x0,x1]×[y0,y1] touches (Shift-drag, plan 9.4). */
export function boxHits(rows: readonly Row[], times: Record<string, Times>, v: Viewport, x0: number, y0: number, x1: number, y1: number): string[] {
  const [left, right] = [Math.min(x0, x1), Math.max(x0, x1)];
  const first = Math.max(0, Math.floor(Math.min(y0, y1) / v.rowH));
  const last = Math.min(rows.length - 1, Math.floor(Math.max(y0, y1) / v.rowH));
  const out: string[] = [];
  for (let i = first; i <= last; i++) {
    const b = barOf(times[rows[i].id], v);
    if (!b) continue;
    const bl = b.milestone ? b.x - b.w / 2 : b.x;
    if (bl <= right && bl + b.w >= left) out.push(rows[i].id);
  }
  return out;
}
