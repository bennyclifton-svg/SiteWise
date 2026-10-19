// Mirror of internal/program/engine.go, the authority. The browser uses this
// only to preview a drag; the server recomputes on release (plan section 3).
// Both engines pass the same JSON fixtures (internal/program/testdata/engine)
// and the Go golden outputs, so keep every rule here line for line with Go.
//
// The node graph, built once per compute (plan 6.3, 6.5):
//   - node i is activity i;
//   - a stage with children ("summary") reuses its own node as the hidden
//     Start(S), with zero duration, and adds one hidden Finish(S) node;
//   - containment edges tie each child between them: Start(S) →SS→ child and
//     child →FF→ Finish(S), both lag 0;
//   - links on a summary are re-anchored: into it on Start(S) (FS, SS), out
//     of it from Finish(S) (FS, FF) or Start(S) (SS, SF).
//
// Iterative, no recursion, O(V+E) per run. Working arrays are typed and kept
// in a module-level pool between calls: a drag recomputes every frame, and
// reallocating them each time costs more than the scheduling. Nothing in the
// returned Output points into the pool.

import { Cal } from "./calendar";
import {
  type Activity,
  type Conflict,
  type Input,
  type Link,
  type Output,
  type Status,
  type Times,
  ConstraintMSO,
  ConstraintSNET,
  KindMilestone,
  LagCalendar,
} from "./types";

const tFS = 0;
const tSS = 1;
const tFF = 2;
const tSF = 3;

const runTypical = 0;
const runLow = 1;
const runHigh = 2;

function linkType(s: string | undefined): number {
  switch (s ?? "") {
    case "FS":
    case "":
      return tFS;
    case "SS":
      return tSS;
    case "FF":
      return tFF;
    case "SF":
      return tSF;
  }
  return -1;
}

// Pooled scratch. Int32 is ample: durations are capped at 3,650 days and
// lags at 365, so indices stay far inside ±2^31.
const pool = {
  i32: new Map<string, Int32Array>(),
  u8: new Map<string, Uint8Array>(),
};

function i32(name: string, n: number): Int32Array {
  let a = pool.i32.get(name);
  if (!a || a.length < n) {
    a = new Int32Array(Math.max(n, 16) * 2);
    pool.i32.set(name, a);
  } else {
    a.fill(0, 0, n);
  }
  return a;
}

function u8(name: string, n: number): Uint8Array {
  let a = pool.u8.get(name);
  if (!a || a.length < n) {
    a = new Uint8Array(Math.max(n, 16) * 2);
    pool.u8.set(name, a);
  } else {
    a.fill(0, 0, n);
  }
  return a;
}

interface Graph {
  acts: Activity[];
  links: Link[];
  cal: Cal;
  n: number; // nodes: activities plus one Finish node per summary
  rows: number;
  act: Int32Array; // node → activity index (a Finish node maps to its stage)
  finish: Int32Array; // activity → its Finish node, −1 when not a summary
  parent: Int32Array; // activity → parent index, −1 when top level
  kidOff: Int32Array;
  kidIdx: Int32Array;
  m: number; // edges
  eFrom: Int32Array;
  eTo: Int32Array;
  eLink: Int32Array; // index into links, −1 for a containment edge
  eLag: Int32Array;
  eTyp: Uint8Array;
  eCal: Uint8Array;
  inOff: Int32Array;
  inIdx: Int32Array;
  outOff: Int32Array;
  outIdx: Int32Array;
  order: Int32Array;
  orderLen: number; // shorter than n when there is a cycle
  unknown: Uint8Array;
  snet: Int32Array;
  mso: Int32Array;
}

function buildGraph(acts: Activity[], links: Link[], cal: Cal): Graph {
  const rows = acts.length;
  const idx = new Map<string, number>();
  for (let i = 0; i < rows; i++) if (!idx.has(acts[i].id)) idx.set(acts[i].id, i);

  const parent = i32("parent", rows);
  const kidOff = i32("kidOff", rows + 1);
  for (let i = 0; i < rows; i++) {
    parent[i] = -1;
    const pid = acts[i].parent ?? "";
    const p = idx.get(pid);
    // Two tiers: only a top-level row can hold children. A row whose parent
    // is missing or is itself a child is scheduled as top level.
    if (pid !== "" && p !== undefined && p !== i && (acts[p].parent ?? "") === "") {
      parent[i] = p;
      kidOff[p + 1]++;
    }
  }
  for (let i = 0; i < rows; i++) kidOff[i + 1] += kidOff[i];
  const kidIdx = i32("kidIdx", kidOff[rows]);
  const next = i32("next", rows);
  for (let i = 0; i < rows; i++) next[i] = kidOff[i];
  for (let i = 0; i < rows; i++) {
    const p = parent[i];
    if (p >= 0) kidIdx[next[p]++] = i;
  }

  let summaries = 0;
  for (let i = 0; i < rows; i++) if (kidOff[i + 1] > kidOff[i]) summaries++;
  const n = rows + summaries;
  const finish = i32("finish", rows);
  const act = i32("act", n);
  for (let i = 0; i < rows; i++) {
    act[i] = i;
    finish[i] = -1;
  }
  let fn = rows;
  for (let i = 0; i < rows; i++) {
    if (kidOff[i + 1] > kidOff[i]) {
      finish[i] = fn;
      act[fn++] = i;
    }
  }

  const cap = links.length + 2 * kidOff[rows];
  const eFrom = i32("eFrom", cap);
  const eTo = i32("eTo", cap);
  const eLink = i32("eLink", cap);
  const eLag = i32("eLag", cap);
  const eTyp = u8("eTyp", cap);
  const eCal = u8("eCal", cap);
  let m = 0;
  for (let li = 0; li < links.length; li++) {
    const l = links[li];
    const p = idx.get(l.pred);
    const s = idx.get(l.succ);
    const t = linkType(l.type);
    if (p === undefined || s === undefined || t < 0 || p === s) continue; // Validate refuses these
    let from = p;
    let to = s;
    if (finish[p] >= 0 && (t === tFS || t === tFF)) from = finish[p];
    // FF and SF into a summary are refused by ValidateLink; one that still
    // arrives (a stage gaining its first child) anchors on Finish(S).
    if (finish[s] >= 0 && (t === tFF || t === tSF)) to = finish[s];
    eFrom[m] = from;
    eTo[m] = to;
    eTyp[m] = t;
    eLag[m] = l.lag ?? 0;
    eCal[m] = l.lag_unit === LagCalendar ? 1 : 0;
    eLink[m] = li;
    m++;
  }
  for (let s = 0; s < rows; s++) {
    for (let k = kidOff[s]; k < kidOff[s + 1]; k++) {
      const c = kidIdx[k];
      eFrom[m] = s;
      eTo[m] = c;
      eTyp[m] = tSS;
      eLink[m] = -1;
      m++;
      eFrom[m] = c;
      eTo[m] = finish[s];
      eTyp[m] = tFF;
      eLink[m] = -1;
      m++;
    }
  }

  const inOff = i32("inOff", n + 1);
  const inIdx = i32("inIdx", m);
  const outOff = i32("outOff", n + 1);
  const outIdx = i32("outIdx", m);
  csr(n, m, eTo, inOff, inIdx);
  csr(n, m, eFrom, outOff, outIdx);

  const g: Graph = {
    acts, links, cal, n, rows, act, finish, parent, kidOff, kidIdx, m,
    eFrom, eTo, eLink, eLag, eTyp, eCal, inOff, inIdx, outOff, outIdx,
    order: i32("order", n), orderLen: 0,
    unknown: u8("unknown", n), snet: i32("snet", n), mso: i32("mso", n),
  };
  topo(g);
  return g;
}

// csr groups edge indices by node, keeping input order within a node so
// "first link wins" ties are stable.
function csr(n: number, m: number, key: Int32Array, off: Int32Array, idx: Int32Array): void {
  for (let e = 0; e < m; e++) off[key[e] + 1]++;
  for (let v = 0; v < n; v++) off[v + 1] += off[v];
  const next = i32("csrNext", n);
  for (let v = 0; v < n; v++) next[v] = off[v];
  for (let e = 0; e < m; e++) idx[next[key[e]]++] = e;
}

// Kahn's algorithm with a FIFO queue seeded in node order.
function topo(g: Graph): void {
  const deg = i32("deg", g.n);
  for (let e = 0; e < g.m; e++) deg[g.eTo[e]]++;
  const order = g.order;
  let len = 0;
  for (let v = 0; v < g.n; v++) if (deg[v] === 0) order[len++] = v;
  for (let k = 0; k < len; k++) {
    const v = order[k];
    for (let j = g.outOff[v]; j < g.outOff[v + 1]; j++) {
      const w = g.eTo[g.outIdx[j]];
      if (--deg[w] === 0) order[len++] = w;
    }
  }
  g.orderLen = len;
}

function isSummary(g: Graph, i: number): boolean {
  return g.finish[i] >= 0;
}

/** A schedulable row's duration in one run, or null when unknown. */
function leafDuration(a: Activity, run: number): number | null {
  if (a.kind === KindMilestone) return 0;
  if (a.duration == null) return null;
  if (run === runLow && a.low != null) return a.low;
  if (run === runHigh && a.high != null) return a.high;
  return a.duration;
}

function durations(g: Graph, name: string, run: number): Int32Array {
  const d = i32(name, g.n);
  for (let i = 0; i < g.rows; i++) {
    if (!isSummary(g, i)) d[i] = leafDuration(g.acts[i], run) ?? 0;
  }
  return d;
}

// Unknown propagation and constraint indices; the same in every run.
function mark(g: Graph): void {
  const { unknown, snet, mso } = g;
  for (let v = 0; v < g.n; v++) {
    snet[v] = -1;
    mso[v] = -1;
  }
  for (let i = 0; i < g.rows; i++) {
    const a = g.acts[i];
    if (!isSummary(g, i)) unknown[i] = leafDuration(a, runTypical) === null ? 1 : 0;
    // A summary's constraint holds its hidden start, and so its children.
    const idx = g.cal.indexOnOrAfter(a.constraint_date);
    if (idx !== null) {
      if (a.constraint === ConstraintSNET) snet[i] = idx;
      else if (a.constraint === ConstraintMSO) mso[i] = Math.max(0, idx);
    }
  }
  for (let k = 0; k < g.orderLen; k++) {
    const v = g.order[k];
    for (let j = g.inOff[v]; j < g.inOff[v + 1]; j++) {
      if (unknown[g.eFrom[g.inIdx[j]]]) {
        unknown[v] = 1;
        break;
      }
    }
  }
}

interface Pass {
  dur: Int32Array;
  es: Int32Array;
  ef: Int32Array;
  ls: Int32Array;
  lf: Int32Array;
  res: Int32Array; // each edge's forward result, before the clamp at 0
  pf: number;
  known: boolean;
  pfLast: boolean; // some row with duration > 0 ends at pf
}

// Forward rule of plan 6.2 for edge e.
function fwd(g: Graph, e: number, p: Pass): number {
  const u = g.eFrom[e];
  const db = p.dur[g.eTo[e]];
  const lag = g.eLag[e];
  const typ = g.eTyp[e];
  if (!g.eCal[e]) {
    switch (typ) {
      case tFS:
        return p.ef[u] + lag;
      case tSS:
        return p.es[u] + lag;
      case tFF:
        return p.ef[u] + lag - db;
      default:
        return p.es[u] + lag - db;
    }
  }
  const c = g.cal;
  switch (typ) {
    case tFS:
      return c.startOnOrAfter(c.finishDay(p.ef[u], p.dur[u]) + lag);
    case tSS:
      return c.startOnOrAfter(c.startDay(p.es[u]) + lag);
    case tFF:
      return c.finishOnOrAfter(c.finishDay(p.ef[u], p.dur[u]) + lag, db) - db;
    default:
      return c.finishOnOrAfter(c.startDay(p.es[u]) + lag, db) - db;
  }
}

// Backward rule of plan 6.4: the latest finish edge e allows its predecessor.
function bwd(g: Graph, e: number, p: Pass): number {
  const u = g.eFrom[e];
  const w = g.eTo[e];
  const lag = g.eLag[e];
  const da = p.dur[u];
  const typ = g.eTyp[e];
  if (!g.eCal[e]) {
    switch (typ) {
      case tFS:
        return p.ls[w] - lag;
      case tSS:
        return p.ls[w] - lag + da;
      case tFF:
        return p.lf[w] - lag;
      default:
        return p.lf[w] - lag + da;
    }
  }
  const c = g.cal;
  switch (typ) {
    case tFS:
      return c.finishOnOrBefore(c.startDay(p.ls[w]) - lag, da);
    case tSS:
      return c.startOnOrBefore(c.startDay(p.ls[w]) - lag) + da;
    case tFF:
      return c.finishOnOrBefore(c.finishDay(p.lf[w], p.dur[w]) - lag, da);
    default:
      return c.startOnOrBefore(c.finishDay(p.lf[w], p.dur[w]) - lag) + da;
  }
}

type OnConflict = (v: number, held: number, e: number) => void;

function forward(g: Graph, p: Pass, conflict: OnConflict | null): void {
  for (let k = 0; k < g.orderLen; k++) {
    const v = g.order[k];
    if (g.unknown[v]) continue;
    let best = -Infinity;
    let bestEdge = -1;
    for (let j = g.inOff[v]; j < g.inOff[v + 1]; j++) {
      const e = g.inIdx[j];
      const r = fwd(g, e, p);
      p.res[e] = r;
      if (r > best) {
        best = r;
        bestEdge = e;
      }
    }
    let es = Math.max(0, g.snet[v], best);
    const m = g.mso[v];
    if (m >= 0) {
      es = m;
      if (best > m && conflict) conflict(v, best, bestEdge);
    }
    p.es[v] = es;
    const ef = es + p.dur[v];
    p.ef[v] = ef;
    if (!p.known || ef > p.pf) {
      p.pf = ef;
      p.known = true;
      p.pfLast = false;
    }
    if (ef === p.pf && p.dur[v] > 0) p.pfLast = true;
  }
}

function backward(g: Graph, p: Pass, free: Int32Array): void {
  for (let k = g.orderLen - 1; k >= 0; k--) {
    const v = g.order[k];
    if (g.unknown[v]) continue;
    let lf = p.pf;
    let ff = Infinity;
    let any = false;
    for (let j = g.outOff[v]; j < g.outOff[v + 1]; j++) {
      const e = g.outIdx[j];
      const w = g.eTo[e];
      if (g.unknown[w]) continue;
      lf = Math.min(lf, bwd(g, e, p));
      ff = Math.min(ff, p.es[w] - p.res[e]);
      any = true;
    }
    if (!any) ff = p.pf - p.ef[v];
    p.lf[v] = lf;
    p.ls[v] = lf - p.dur[v];
    free[v] = ff;
  }
}

function newPass(dur: Int32Array, es: Int32Array, ef: Int32Array, ls: Int32Array, lf: Int32Array, res: Int32Array): Pass {
  return { dur, es, ef, ls, lf, res, pf: 0, known: false, pfLast: false };
}

function hasRanges(acts: Activity[]): boolean {
  for (const a of acts) if (a.low != null || a.high != null) return true;
  return false;
}

function emptyFinish(): Output["finish"] {
  return { typical: null, low: null, high: null, typical_index: null, low_index: null, high_index: null, complete: false };
}

const unknownTimes = (): Times => ({
  es: null, ef: null, ls: null, lf: null, tf: null, ff: null,
  critical: false, start: null, finish: null, unknown: true,
});

/**
 * Schedules the program (plan 6.3–6.5): pure, O(V+E) per run. The forward
 * pass runs three times (typical, low, high) and the backward pass once, on
 * the typical durations the bars draw.
 */
export function compute(input: Input): Output {
  const acts = input.activities ?? [];
  const links = input.links ?? [];
  const cal = new Cal(input.calendar ?? {});
  const g = buildGraph(acts, links, cal);
  const out: Output = {
    status: status(g),
    finish: emptyFinish(),
    activities: {},
    driving_links: [],
    cycle: null,
    conflicts: [],
  };
  if (g.orderLen < g.n) {
    out.cycle = findCycle(g);
    return out;
  }
  mark(g);

  const n = g.n;
  const m = g.m;
  const typ = newPass(durations(g, "durT", runTypical), i32("es", n), i32("ef", n), i32("ls", n), i32("lf", n), i32("res", m));
  const held = new Map<number, Conflict>();
  forward(g, typ, (v, h, e) => {
    const c: Conflict = {
      activity_id: acts[g.act[v]].id,
      index: g.mso[v],
      held_index: h,
      held_by: acts[g.act[g.eFrom[e]]].id,
    };
    const l = g.eLink[e];
    if (l >= 0 && links[l].id) c.link_id = links[l].id;
    held.set(v, c);
  });
  const free = i32("free", n);
  backward(g, typ, free);
  let low = typ;
  let high = typ;
  if (hasRanges(acts)) {
    // Only the project finish is wanted from these runs, so they share one
    // set of arrays.
    const es2 = i32("es2", n);
    const ef2 = i32("ef2", n);
    const res2 = i32("res2", m);
    low = newPass(durations(g, "durL", runLow), es2, ef2, es2, ef2, res2);
    forward(g, low, null);
    high = newPass(durations(g, "durH", runHigh), es2, ef2, es2, ef2, res2);
    forward(g, high, null);
  }

  rowTimes(g, out, typ, free);
  out.finish.complete = out.status.known === out.status.total;
  [out.finish.typical, out.finish.typical_index] = finishOf(g, typ);
  [out.finish.low, out.finish.low_index] = finishOf(g, low);
  [out.finish.high, out.finish.high_index] = finishOf(g, high);

  // Driving links follow input link order, as in Go.
  const flags = u8("flags", links.length);
  for (let e = 0; e < m; e++) {
    const l = g.eLink[e];
    const w = g.eTo[e];
    if (l >= 0 && !g.unknown[w] && typ.res[e] === typ.es[w]) flags[l] = 1;
  }
  for (let li = 0; li < links.length; li++) if (flags[li]) out.driving_links.push(links[li].id ?? "");
  // Conflicts follow input row order, not the topological order.
  for (let i = 0; i < g.rows; i++) {
    const c = held.get(i);
    if (c) out.conflicts.push(c);
  }
  return out;
}

function finishOf(g: Graph, p: Pass): [string | null, number | null] {
  if (!p.known) return [null, null];
  if (g.cal.relative) return [null, p.pf];
  // The finish date is the last working day, unless only zero-length rows
  // reach pf (a closing milestone), which show their own day.
  return [g.cal.dateOf(p.pfLast ? p.pf - 1 : p.pf), p.pf];
}

// Summary roll-up: the bar spans the earliest start to the latest finish of
// its children; late dates likewise; total float is the least child float;
// free float is that of the hidden Finish node. Writes into r and returns
// false when the row is unknown.
const rowScratch = new Int32Array(6);

function rowNumbers(g: Graph, i: number, p: Pass, free: Int32Array, r: Int32Array): boolean {
  if (!isSummary(g, i)) {
    if (g.unknown[i]) return false;
    r[0] = p.es[i];
    r[1] = p.ef[i];
    r[2] = p.ls[i];
    r[3] = p.lf[i];
    r[4] = p.ls[i] - p.es[i];
    r[5] = free[i];
    return true;
  }
  const fin = g.finish[i];
  if (g.unknown[fin]) return false;
  let es = Infinity;
  let ef = -Infinity;
  let ls = Infinity;
  let lf = -Infinity;
  let tf = Infinity;
  for (let k = g.kidOff[i]; k < g.kidOff[i + 1]; k++) {
    const c = g.kidIdx[k];
    es = Math.min(es, p.es[c]);
    ef = Math.max(ef, p.ef[c]);
    ls = Math.min(ls, p.ls[c]);
    lf = Math.max(lf, p.lf[c]);
    tf = Math.min(tf, p.ls[c] - p.es[c]);
  }
  r[0] = es;
  r[1] = ef;
  r[2] = ls;
  r[3] = lf;
  r[4] = tf;
  r[5] = free[fin];
  return true;
}

function rowTimes(g: Graph, out: Output, p: Pass, free: Int32Array): void {
  const r = rowScratch;
  let lo = Infinity;
  let hi = -Infinity;
  for (let i = 0; i < g.rows; i++) {
    if (rowNumbers(g, i, p, free, r)) {
      lo = Math.min(lo, r[0]);
      hi = Math.max(hi, r[1] - 1, r[0]);
    }
  }
  if (lo !== Infinity) g.cal.warm(lo, hi, g.rows);
  const rowsOut = out.activities;
  for (let i = 0; i < g.rows; i++) {
    const id = g.acts[i].id;
    if (!rowNumbers(g, i, p, free, r)) {
      rowsOut[id] = unknownTimes();
      continue;
    }
    const [start, finish] = g.cal.display(r[0], r[1] - r[0]);
    rowsOut[id] = {
      es: r[0], ef: r[1], ls: r[2], lf: r[3], tf: r[4], ff: r[5],
      critical: r[4] <= 0,
      start: start === "" ? null : start,
      finish: finish === "" ? null : finish,
      unknown: false,
    };
  }
}

// Counts schedulable rows: every row except a stage with children.
function status(g: Graph): Status {
  let known = 0;
  let total = 0;
  for (let i = 0; i < g.rows; i++) {
    if (isSummary(g, i)) continue;
    total++;
    if (leafDuration(g.acts[i], runTypical) !== null) known++;
  }
  return { known, total, words: statusWords(known, total) };
}

/** The toolbar line, e.g. "Indicative: 31 of 44 durations known". */
export function statusWords(known: number, total: number): string {
  if (total === 0) return "No activities yet";
  if (known === total) return `All ${total} durations known`;
  return `Indicative: ${known} of ${total} durations known`;
}

// Go compares ids as UTF-8 bytes, which is code point order. JavaScript's <
// compares UTF-16 units, which differs only between astral characters and
// U+E000–U+FFFF; compare by code point so the reported cycle matches Go.
function compareIds(a: string, b: string): number {
  if (a === b) return 0;
  const n = Math.min(a.length, b.length);
  for (let k = 0; k < n; k++) {
    const x = a.codePointAt(k) as number;
    const y = b.codePointAt(k) as number;
    if (x !== y) return x < y ? -1 : 1;
    if (x > 0xffff) k++;
  }
  return a.length < b.length ? -1 : a.length > b.length ? 1 : 0;
}

// Outline key for the cycle walk: a stage sorts as (ordinal, −1), its child
// as (stage ordinal, ordinal), and a hidden Finish node as (stage ordinal, ∞).
function key(g: Graph, v: number): [number, number, string] {
  const i = g.act[v];
  const a = g.acts[i];
  if (v >= g.rows) return [a.ordinal ?? 0, Infinity, a.id];
  const p = g.parent[i];
  if (p >= 0) return [g.acts[p].ordinal ?? 0, a.ordinal ?? 0, a.id];
  return [a.ordinal ?? 0, -1, a.id];
}

function less(g: Graph, a: number, b: number): boolean {
  const ka = key(g, a);
  const kb = key(g, b);
  if (ka[0] !== kb[0]) return ka[0] < kb[0];
  if (ka[1] !== kb[1]) return ka[1] < kb[1];
  const c = compareIds(ka[2], kb[2]);
  if (c !== 0) return c < 0;
  return a < b; // the finish node of a stage sorts after the stage
}

// One loop as closed activity ids; see engine.go findCycle for why the walk
// from the least key, stepping to the least-key predecessor, finds a real
// loop that depends only on the input.
function findCycle(g: Graph): string[] {
  const ordered = new Uint8Array(g.n);
  for (let k = 0; k < g.orderLen; k++) ordered[g.order[k]] = 1;
  let start = -1;
  for (let v = 0; v < g.n; v++) {
    if (!ordered[v] && (start < 0 || less(g, v, start))) start = v;
  }
  const seen = new Map<number, number>();
  let walk: number[] = [];
  for (let v = start; ; ) {
    const at = seen.get(v);
    if (at !== undefined) {
      walk = walk.slice(at);
      break;
    }
    seen.set(v, walk.length);
    walk.push(v);
    let next = -1;
    for (let j = g.inOff[v]; j < g.inOff[v + 1]; j++) {
      const u = g.eFrom[g.inIdx[j]];
      if (!ordered[u] && (next < 0 || less(g, u, next))) next = u;
    }
    v = next;
  }
  walk.reverse(); // the walk runs against the links; read it forwards
  let least = 0;
  for (let k = 0; k < walk.length; k++) if (less(g, walk[k], walk[least])) least = k;
  walk = walk.slice(least).concat(walk.slice(0, least));
  // Start(S) → child → Finish(S) is the loop passing through stage S.
  const through = (k: number): boolean => {
    const v = walk[k];
    const p = v < g.rows ? g.parent[v] : -1;
    const prev = walk[(k + walk.length - 1) % walk.length];
    const nxt = walk[(k + 1) % walk.length];
    return p >= 0 && prev === p && nxt === g.finish[p];
  };
  const ids: string[] = [];
  for (let k = 0; k < walk.length; k++) {
    if (walk.length > 2 && through(k)) continue;
    const id = g.acts[g.act[walk[k]]].id;
    if (ids.length === 0 || ids[ids.length - 1] !== id) ids.push(id);
  }
  if (ids.length > 1 && ids[ids.length - 1] === ids[0]) ids.pop();
  ids.push(ids[0]);
  return ids;
}

/** One loop in the program as closed activity ids, or null. */
export function findCycleIn(input: Input): string[] | null {
  const g = buildGraph(input.activities ?? [], input.links ?? [], new Cal(input.calendar ?? {}));
  return g.orderLen === g.n ? null : findCycle(g);
}

/** Names a loop by label in order, e.g. "Framing → Roof → Framing". */
export function cycleWords(input: Input, cycle: string[] | null): string {
  const label = new Map<string, string>();
  for (const a of input.activities ?? []) label.set(a.id, a.label ? a.label : a.id);
  return (cycle ?? []).map((id) => label.get(id) ?? "").join(" → ");
}
