// The Program view (plan docs/plans/2026-10-18-program.md section 9):
// toolbar, the one question, the completion header and a split pane with
// the grid on the left and the chart on the right sharing one vertical
// scroll. Every date comes from the server's `computed`, except during a
// drag, when the TypeScript engine previews it (chart/useDrag.ts).
//
// Editing (WP-P8): every edit is one POST /program/ops batch through
// `apply`, the one place a write response replaces state. Writes queue so
// each carries the version the last answer returned; a drag shows its
// preview at once and is replaced by the server's answer, or restored with
// its message (422) or the newer version (409). An SSE refetch that lands
// during a drag waits for the release (plan 8.3).

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type DragEvent, type KeyboardEvent, type PointerEvent as ReactPointerEvent, type ReactElement } from "react";
import { flushSync } from "react-dom";
import { ApiError } from "./api";
import { ConfirmDialog } from "./ConfirmDialog";
import { needsRefetch, programApi, ProgramConflict, ProgramRefused, type ProgramChange, type ProgramHoldPoint, type ProgramOp, type ProgramView, type RemoteProgram } from "./programApi";
import { Cal } from "./program/calendar";
import { deletePlan, dropTarget, inOutline, predecessorOps, predecessorTexts, reorderStep, rowNumbers, ZONE_COARSE, ZONE_FINE, type DeletePlan } from "./program/edit";
import { clampPx, fitHeight, fitSpan, LEFT_ROOM, PX_DEFAULT, scaleFor, zoomAbout } from "./program/fit";
import {
  drawnLinks,
  extent,
  fmtDate,
  indexAt,
  outlineRows,
  ROW_H,
  rowWindow,
  shownFinish,
  stagesWithChildren,
  xOf,
  type Row,
  type Viewport,
} from "./program/layout";
import { clickRow, emptySelection, prune, selectAll, type Selection } from "./program/selection";
import { Grid, GRID_COLUMNS, type CellFocus, type CommitKey, type EditCol, type Editing } from "./program/grid/Grid";
import { parseDuration, parsePredecessors } from "./program/grid/parse";
import { SCALE_H, TimescaleGrid, TimescaleHeader } from "./program/chart/Timescale";
import { Bars } from "./program/chart/Bars";
import { Links, lagText } from "./program/chart/Links";
import { Overlay, type OverlayHandle } from "./program/chart/Overlay";
import { useDrag } from "./program/chart/useDrag";
import { EvidenceNotice, HoldPointsPanel, ReadingsPanel, ReviewBanner, ReviewPanel } from "./program/Panels";
import { groupReadings, importable, orphanEvidence, proposedText, reviewWords } from "./program/readings";
import type { RowNote } from "./program/grid/Grid";
import type { Times } from "./program/types";
import "./program.css";

interface Props {
  projectId: string;
  /** Newest program version announced by `program.changed` (plan 8.3), and its program. */
  remoteVersion: RemoteProgram;
  /** Bumped by `program.review` and `program.readings`: the review or the readings changed, not the version. */
  remoteTick?: number;
  /** A strip answer wrote a Profile value: the Profile view refetches. */
  onProfileChanged?: () => void;
  /** The view is on screen; a hidden view never takes over printing. */
  active: boolean;
  onSignedOut: () => void;
}

const GRID_W = 620;
const GRID_W_NARROW = 150;
const GRID_W_MIN = 220;
/** Below this pane width the grid shows the name column only (plan 9.1). */
const NARROW = 720;
const X_CHUNK = 256;
/** Rows on one A3 landscape page at 28 px with the header (plan 9.7). */
// A3 landscape less the 10 mm margins is 400 × 277 mm: 1511 × 1046 CSS px.
// Grid plus chart fit that width; rows per page fit that height under the
// 44 px timescale, and the first page leaves room for the heading and the
// completion summary printed above it.
const PRINT_PAGE_W = 1511;
const PRINT_PAGE_H = 1046;
const PRINT_ROWS = 30;
const PRINT_GRID_W = 520;
const PRINT_CHART_W = PRINT_PAGE_W - PRINT_GRID_W - 11;
/** Room the shell's printed title takes above the program when it cannot be measured. */
const PRINT_SHELL_H = 80;
/** Grid columns the arrow keys move between (plan 9.5). */
const COLS = ["name", "duration", "start", "finish", "predecessors"];

const expandKey = (p: string) => `sitewise.program.expanded.v1.${p}`;

// Per-browser view preference (plan 7.5): never on the program, no undo.
function readExpanded(p: string): Set<string> {
  try {
    const raw = localStorage.getItem(expandKey(p));
    const ids = raw ? (JSON.parse(raw) as unknown) : [];
    return new Set(Array.isArray(ids) ? ids.filter((x): x is string => typeof x === "string") : []);
  } catch {
    return new Set();
  }
}

function writeExpanded(p: string, ids: Set<string>) {
  try {
    localStorage.setItem(expandKey(p), JSON.stringify([...ids]));
  } catch {
    // A private window can refuse storage; the view still applies for this visit.
  }
}

// Today as YYYY-MM-DD in local time: display only (the today line), never
// program logic (plan 2, product rule 9).
function todayString(): string {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

const reducedMotion = () => {
  try {
    return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  } catch {
    return false;
  }
};

interface Win {
  first: number;
  last: number;
  x0: number;
  x1: number;
}

interface Toast {
  text: string;
  /** The write can be undone from the toast. */
  undo?: boolean;
  /** A refusal: announced assertively. */
  alert?: boolean;
}

interface ApplyOpts {
  optimistic?: ProgramView;
  /** An edit: one undo step, so the toast offers Undo. */
  undoable?: boolean;
  /** Toast when the server sends none. */
  note?: string;
  then?: (v: ProgramView) => void;
  refused?: (message: string) => void;
}

type Dialog =
  | { kind: "delete"; ids: string[]; plan: DeletePlan }
  | { kind: "durations"; ids: string[]; days: number | null }
  | { kind: "milestone"; ops: ProgramOp[]; label: string; days: number; o: CommitOpts };

interface CommitOpts {
  optimistic?: ProgramView;
  /** Toast when the server sends none (it still offers Undo). */
  note?: string;
  added?: { temp: string; parent: string | null };
  then?: (v: ProgramView) => void;
  refused?: (message: string) => void;
}

export function Program({ projectId, remoteVersion, remoteTick = 0, active, onSignedOut, onProfileChanged }: Props) {
  const [view, setView] = useState<ProgramView | null | undefined>(undefined);
  const [error, setError] = useState("");
  const [toast, setToast] = useState<Toast | null>(null);
  const [busy, setBusy] = useState(false);
  const [expanded, setExpanded] = useState<Set<string>>(() => readExpanded(projectId));
  const [sel, setSel] = useState<Selection>(emptySelection);
  const [ppd, setPpd] = useState(PX_DEFAULT);
  const [rowH, setRowH] = useState(ROW_H);
  const [tooMany, setTooMany] = useState(false);
  const [showLinks, setShowLinks] = useState(true);
  const [gridW, setGridW] = useState(GRID_W);
  const [box, setBox] = useState({ w: 0, h: 0 });
  const [win, setWin] = useState<Win>({ first: 0, last: 40, x0: 0, x1: 2048 });
  const [printing, setPrinting] = useState(false);
  const [printFirst, setPrintFirst] = useState(PRINT_ROWS);
  const [checksOpen, setChecksOpen] = useState(false);
  const [reviewOpen, setReviewOpen] = useState(false);
  const [changes, setChanges] = useState<{ list: ProgramChange[]; counts: Record<string, number> } | null>(null);
  const [readingsOpen, setReadingsOpen] = useState(true);
  const [dropping, setDropping] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);
  const [startInput, setStartInput] = useState("");
  const [editing, setEditing] = useState<Editing | null>(null);
  const [focus, setFocus] = useState<CellFocus | null>(null);
  const [dialog, setDialog] = useState<Dialog | null>(null);
  const [reason, setReason] = useState("");
  // `back`: the bar a keyboard user reached the link from; focus returns there when the popover closes.
  const [linkPop, setLinkPop] = useState<{ id: string; x: number; y: number; back?: string } | null>(null);
  const [coarse] = useState(() => {
    try {
      return window.matchMedia("(pointer: coarse)").matches;
    } catch {
      return false;
    }
  });

  const scroller = useRef<HTMLDivElement>(null);
  const [scrollEl, setScrollEl] = useState<HTMLDivElement | null>(null);
  const gridcol = useRef<HTMLDivElement>(null);
  const insert = useRef<HTMLDivElement>(null);
  const overlay = useRef<OverlayHandle>(null);
  const pendingScroll = useRef<{ left?: number; top?: number } | null>(null);
  const fitted = useRef(false);
  const raf = useRef(0);
  const dragging = useRef(false);
  const refetchOnRelease = useRef(false);
  const divider = useRef<{ x: number; w: number } | null>(null);
  /** The view on screen (optimistic included) and the last one the server sent. */
  const viewRef = useRef<ProgramView | null>(null);
  const confirmed = useRef<ProgramView | null>(null);
  const queue = useRef<Promise<void>>(Promise.resolve());
  /** Bar x positions before a state change, for the settle animation. */
  const settleFrom = useRef<Map<string, number> | null>(null);
  /** Where DOM focus goes after the next render: a grid cell or a bar. */
  const focusNext = useRef<{ kind: "cell" | "bar"; id: string } | null>(null);

  const fail = useCallback(
    (e: unknown) => {
      if (e instanceof ApiError && e.status === 401) return onSignedOut();
      setError(e instanceof Error ? e.message : "Could not load the program.");
    },
    [onSignedOut],
  );

  const show = useCallback((v: ProgramView | null) => {
    viewRef.current = v;
    setView(v);
  }, []);

  const load = useCallback(() => {
    programApi.read(projectId).then(
      (v) => {
        confirmed.current = v;
        captureBars();
        show(v);
        setError("");
      },
      (e) => fail(e),
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, fail, show]);

  useEffect(load, [load]);

  // A newer version from another viewer: refetch now, or on release when a
  // drag is in progress (plan 8.3).
  // Each announcement refetches at most once, so a GET that answers with a
  // lower version (a recreated program) can never refetch in a loop.
  const fetchedFor = useRef<RemoteProgram | null>(null);
  useEffect(() => {
    if (!view || !needsRefetch(remoteVersion, view.program, fetchedFor.current)) return;
    fetchedFor.current = remoteVersion;
    if (dragging.current) refetchOnRelease.current = true;
    else load();
  }, [remoteVersion, view, load]);

  // The review or the readings changed (plan 8.3 events that carry no
  // version): refetch, after the drag when one is in progress.
  const seenTick = useRef(remoteTick);
  useEffect(() => {
    if (remoteTick === seenTick.current) return;
    seenTick.current = remoteTick;
    if (dragging.current) refetchOnRelease.current = true;
    else load();
  }, [remoteTick, load]);

  // The review list and its counts, read whenever the pending count or the
  // program changes while something is pending (plan 7.4).
  const pendingChanges = view?.changes_pending ?? 0;
  const viewVersion = view?.program.version ?? 0;
  useEffect(() => {
    if (pendingChanges <= 0) {
      setChanges(null);
      setReviewOpen(false);
      return;
    }
    let stale = false;
    programApi.changes(projectId).then(
      (c) => !stale && setChanges({ list: c.changes ?? [], counts: c.counts ?? {} }),
      (e) => !stale && fail(e),
    );
    return () => {
      stale = true;
    };
  }, [projectId, pendingChanges, viewVersion, fail]);

  /** Ends a drag and runs a refetch deferred during it. */
  const releaseDrag = useCallback(() => {
    dragging.current = false;
    if (refetchOnRelease.current) {
      refetchOnRelease.current = false;
      load();
    }
  }, [load]);

  // Record where every bar is drawn now, so the next render can ease from it.
  function captureBars() {
    const svg = scroller.current?.querySelector(".pg-chart");
    if (!svg || reducedMotion()) return;
    const m = new Map<string, number>();
    for (const el of svg.querySelectorAll<SVGGElement>(".pg-layer-bars .pg-bar")) m.set(el.dataset.id!, Number(el.dataset.x) + Number(el.dataset.dx ?? 0));
    settleFrom.current = m;
  }

  /**
   * Runs a write in order after any write before it, against the last
   * version the server sent. A 409 shows the other user's version; a 422
   * restores what was there and shows the server's message.
   */
  const apply = useCallback(
    (write: (v: ProgramView | null) => Promise<ProgramView>, o: ApplyOpts = {}) => {
      if (o.optimistic) flushSync(() => show(o.optimistic!));
      const run = async () => {
        setBusy(true);
        try {
          const next = await write(confirmed.current);
          confirmed.current = next;
          captureBars();
          show(next);
          setError("");
          const words = next.toasts ?? [];
          if (words.length || o.note) setToast({ text: words.length ? words.join(" ") : o.note!, undo: o.undoable });
          else if (o.undoable) setToast(null);
          o.then?.(next);
        } catch (e) {
          if (e instanceof ProgramConflict) {
            confirmed.current = e.current;
            show(e.current);
            setToast({ text: e.message, alert: true });
          } else if (e instanceof ProgramRefused) {
            captureBars();
            show(confirmed.current);
            setToast({ text: e.message, alert: true });
            o.refused?.(e.message);
          } else {
            if (o.optimistic) show(confirmed.current);
            fail(e);
          }
        } finally {
          setBusy(false);
        }
      };
      queue.current = queue.current.then(run, run);
    },
    [fail, show],
  );

  // --- derived model -------------------------------------------------------

  const model = useMemo(() => {
    if (!view) return null;
    const start = view.program.start_date ?? "";
    const days = (view.calendar.days ?? []).map((d) => ({ day: d.day, kind: d.kind, label: d.label }));
    const cal = new Cal({ start, working_week: view.program.working_week, days });
    const plain = new Cal({ start, working_week: view.program.working_week });
    const parentOf = new Map(view.activities.map((a) => [a.id, a.parent_id]));
    const withKids = stagesWithChildren(view.activities);
    const today = cal.relative ? null : cal.indexOnOrAfter(todayString());
    const byId = new Map(view.activities.map((a) => [a.id, a]));
    // Row → label of the row driving its start, for "held by Slab" (plan 9.5).
    const drivers = new Map<string, string>();
    const linkById = new Map(view.links.map((l) => [l.id, l]));
    for (const id of view.computed.driving_links ?? []) {
      const l = linkById.get(id);
      if (l && !drivers.has(l.succ_id)) drivers.set(l.succ_id, byId.get(l.pred_id)?.label ?? "");
    }
    return {
      cal,
      plain,
      parentOf,
      withKids,
      today,
      byId,
      drivers,
      preds: predecessorTexts(view.links, rowNumbers(view.activities)),
      times: view.computed.activities as Record<string, Times>,
      driving: new Set(view.computed.driving_links ?? []),
      conflicts: new Set((view.computed.conflicts ?? []).map((c) => c.activity_id)),
      proposed: view.activities.filter((a) => a.review_status === "proposed").length,
    };
  }, [view]);

  // Readings, hold-point evidence and releases, for the panels, the grid
  // markers and the chart's amber outlines (plan 10.2 to 10.4).
  const docs = useMemo(() => {
    if (!view || !model) return null;
    const groups = groupReadings(view.readings);
    const live = groups.filter((g) => g.targetId && model.byId.has(g.targetId));
    const notes = new Map<string, RowNote[]>();
    const add = (id: string, n: RowNote) => notes.set(id, [...(notes.get(id) ?? []), n]);
    const marks = new Map<string, readonly [number, number]>();
    for (const g of live) {
      const id = g.targetId!;
      const text = g.text || proposedText(g, fmtDate);
      add(id, { tone: "reading", text: g.source === "proposal_r7" ? "Proposal" : "File", title: `${g.documentTitle}: ${text}. Accept or dismiss it in Readings.` });
      const s = model.cal.relative ? null : model.cal.indexOnOrAfter(g.start?.date);
      const f = model.cal.relative ? null : model.cal.indexOnOrBefore(g.finish?.date);
      if (s !== null && f !== null && f >= s) marks.set(id, [s, f + 1]);
    }
    const holdInfo = new Map<string, ProgramHoldPoint>((view.hold_points ?? []).map((h) => [h.activity_id, h]));
    const released = new Set<string>();
    for (const h of holdInfo.values()) {
      if (!model.byId.has(h.activity_id)) continue;
      if (h.released) {
        released.add(h.activity_id);
        add(h.activity_id, { tone: "released", text: "Released", title: "Marked released by a user" });
      } else if ((h.evidence ?? []).length) {
        add(h.activity_id, { tone: "evidence", text: "Evidence found", title: `${h.evidence![0].text}. ${h.limit}` });
      }
    }
    const holds = view.activities.filter((a) => a.kind === "hold_point");
    const imports = view.imports ?? [];
    const proposalReadings = live.filter((g) => g.source === "proposal_r7").length;
    return { groups, live, notes, marks, holdInfo, released, holds, imports, proposalReadings, orphans: orphanEvidence(groups) };
  }, [view, model]);

  const rows: Row[] = useMemo(() => (view ? outlineRows(view.activities, expanded) : []), [view, expanded]);
  const order = useMemo(() => rows.map((r) => r.id), [rows]);
  useEffect(() => setSel((s) => prune(s, order)), [order]);
  const selected = useMemo(() => new Set(sel.ids), [sel]);

  const span = useMemo(() => (model ? extent(order, model.times, view?.computed.finish.high_index ?? null) : null), [model, order, view]);
  const allSpan = useMemo(
    () => (model && view ? extent(view.activities.map((a) => a.id), model.times, view.computed.finish.high_index) : null),
    [model, view],
  );
  // The chart's x = 0 sits two days and the fit's left room before the
  // program start (or the earliest bar), fixed for a zoom, so fitting only
  // scrolls and a start link dot on day 0 is never cut off.
  const originAt = useCallback((px: number) => Math.min(0, allSpan?.[0] ?? 0) - 2 - LEFT_ROOM / px, [allSpan]);
  const origin = originAt(ppd);
  const v: Viewport = useMemo(() => ({ pxPerDay: ppd, viewStart: origin, rowH }), [ppd, origin, rowH]);

  const narrow = box.w > 0 && box.w < NARROW;
  const gw = narrow ? GRID_W_NARROW : Math.min(gridW, Math.max(GRID_W_MIN, box.w - 200));
  const chartViewW = Math.max(0, box.w - gw);
  const endIndex = Math.max(allSpan?.[1] ?? 0, view?.computed.finish.high_index ?? 0, 0) + 20;
  const chartW = Math.max(chartViewW, Math.ceil(xOf(endIndex, v)));
  const height = rows.length * rowH;
  const scale = scaleFor(ppd);

  const first = printing ? 0 : Math.min(win.first, rows.length);
  const last = printing ? rows.length : Math.min(win.last, rows.length);
  const lo = Math.floor(indexAt(Math.max(0, win.x0), v));
  const hi = Math.ceil(indexAt(Math.min(chartW, win.x1), v)) + 1;

  const drawn = useMemo(
    () => (model && view && showLinks ? drawnLinks(view.links, rows, model.parentOf, first, last) : []),
    [model, view, rows, showLinks, first, last],
  );

  // --- settle and focus after a render ----------------------------------------

  useLayoutEffect(() => {
    const from = settleFrom.current;
    settleFrom.current = null;
    const svg = scroller.current?.querySelector(".pg-chart");
    if (!from || !svg) return;
    // Plan 9.4 "Settle": bars ease to where the server put them over 150 ms.
    const moved: SVGGElement[] = [];
    for (const el of svg.querySelectorAll<SVGGElement>(".pg-layer-bars .pg-bar")) {
      const old = from.get(el.dataset.id!);
      const now = Number(el.dataset.x);
      el.style.transition = "";
      delete el.dataset.dx;
      if (old === undefined || Math.abs(old - now) < 0.5) {
        el.style.transform = "";
        continue;
      }
      el.style.transform = `translateX(${old - now}px)`;
      moved.push(el);
    }
    if (!moved.length) return;
    void svg.getBoundingClientRect();
    for (const el of moved) {
      el.style.transition = "transform 150ms ease-out";
      el.style.transform = "";
      window.setTimeout(() => (el.style.transition = ""), 200);
    }
  }, [view]);

  useLayoutEffect(() => {
    const f = focusNext.current;
    if (!f || !scroller.current) return;
    const el =
      f.kind === "cell"
        ? scroller.current.querySelector<HTMLElement>('.pg-gridcol [data-focus="1"]')
        : scroller.current.querySelector<SVGGElement>(`.pg-layer-bars [data-id="${f.id}"]`);
    if (el) {
      focusNext.current = null;
      el.focus({ preventScroll: true });
    }
  });

  // --- scrolling, measuring and fitting -------------------------------------

  const live = useRef({ rows: 0, rowH: ROW_H, gw: GRID_W });
  live.current = { rows: rows.length, rowH, gw };

  const measure = useCallback(() => {
    const el = scroller.current;
    if (!el) return;
    const { rows: n, rowH: h, gw: g } = live.current;
    const [f, l] = rowWindow(el.scrollTop, el.clientHeight - SCALE_H, h, n);
    // Horizontal window in chunks so a small scroll renders nothing new.
    const cw = Math.max(1, el.clientWidth - g);
    const x0 = Math.floor(el.scrollLeft / X_CHUNK) * X_CHUNK - cw;
    const x1 = x0 + 3 * cw + X_CHUNK;
    setWin((p) => (p.first === f && p.last === l && p.x0 === x0 && p.x1 === x1 ? p : { first: f, last: l, x0, x1 }));
  }, []);

  const onScroll = useCallback(() => {
    if (raf.current) return;
    raf.current = requestAnimationFrame(() => {
      raf.current = 0;
      measure();
    });
  }, [measure]);

  useEffect(() => () => cancelAnimationFrame(raf.current), []);

  useLayoutEffect(() => {
    const el = scroller.current;
    setScrollEl(el);
    if (!el) return;
    const ro = new ResizeObserver(() => {
      setBox((b) => (b.w === el.clientWidth && b.h === el.clientHeight ? b : { w: el.clientWidth, h: el.clientHeight }));
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, [view === null || view === undefined]);

  // Re-measure the window when rows, row height or the pane change.
  useLayoutEffect(measure, [measure, rows.length, rowH, box, gw]);

  // Apply a scroll asked for by a fit or zoom once its layout has rendered.
  useLayoutEffect(() => {
    const el = scroller.current;
    const p = pendingScroll.current;
    if (!el || !p) return;
    pendingScroll.current = null;
    if (p.left !== undefined) el.scrollLeft = Math.max(0, p.left);
    if (p.top !== undefined) el.scrollTop = Math.max(0, p.top);
    measure();
  });

  const fitTo = useCallback(
    (s: [number, number] | null) => {
      if (!s || chartViewW <= 0) return false;
      const f = fitSpan(s, chartViewW);
      setPpd(f.pxPerDay);
      pendingScroll.current = { left: (f.viewStart - originAt(f.pxPerDay)) * f.pxPerDay };
      return true;
    },
    [chartViewW, originAt],
  );

  const note = useCallback((text: string) => setToast({ text }), []);

  const fitScreen = useCallback(() => {
    if (!fitTo(span)) note("Nothing to fit yet: no row has a known duration.");
  }, [fitTo, span, note]);

  const fitSelection = useCallback(() => {
    if (!model) return;
    if (sel.ids.length === 0) return note("Select rows to fit them.");
    if (!fitTo(extent(sel.ids, model.times))) note("The selected rows have no known dates to fit.");
  }, [fitTo, model, sel, note]);

  const fitRows = useCallback(() => {
    const el = scroller.current;
    if (!el) return;
    const f = fitHeight(el.clientHeight - SCALE_H, rows.length);
    setRowH(f.rowH);
    setTooMany(f.tooMany);
  }, [rows.length]);

  // Fit to screen once on first open, when there is something to fit.
  useEffect(() => {
    if (fitted.current || !view || chartViewW <= 0) return;
    fitted.current = true;
    fitTo(span);
  }, [view, chartViewW, fitTo, span]);

  const zoom = useCallback(
    (factor: number) => {
      const el = scroller.current;
      if (!el) return;
      const centre = indexAt(el.scrollLeft + chartViewW / 2, v);
      const next = clampPx(zoomAbout(ppd, origin, factor, centre).pxPerDay);
      setPpd(next);
      pendingScroll.current = { left: (centre - originAt(next)) * next - chartViewW / 2 };
    },
    [chartViewW, v, ppd, origin, originAt],
  );

  const goToday = useCallback(() => {
    if (!model || model.today === null) return note("Set a start date to see today on the program.");
    if (model.today < origin || model.today > endIndex) return note(`Today (${fmtDate(todayString())}) is outside the program.`);
    pendingScroll.current = { left: xOf(model.today, v) - chartViewW / 3 };
    setWin((w) => ({ ...w })); // render so the scroll applies
  }, [model, origin, endIndex, v, chartViewW, note]);

  /** Scrolls so row i is inside the visible rows. */
  const reveal = useCallback(
    (i: number) => {
      const el = scroller.current;
      if (!el) return;
      const top = i * rowH;
      const viewH = el.clientHeight - SCALE_H;
      if (top < el.scrollTop) el.scrollTop = top;
      else if (top + rowH > el.scrollTop + viewH) el.scrollTop = top + rowH - viewH;
    },
    [rowH],
  );

  // --- expand and collapse ---------------------------------------------------

  const setOpen = useCallback(
    (next: Set<string>) => {
      setExpanded(next);
      writeExpanded(projectId, next);
    },
    [projectId],
  );
  const toggle = useCallback(
    (id: string) => {
      const next = new Set(expanded);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      setOpen(next);
    },
    [expanded, setOpen],
  );
  const expandAll = useCallback(() => model && setOpen(new Set(model.withKids)), [model, setOpen]);
  const collapseAll = useCallback(() => setOpen(new Set()), [setOpen]);

  const onSelect = useCallback(
    (id: string, mods: { ctrl: boolean; shift: boolean }) => setSel((s) => clickRow(s, id, order, mods)),
    [order],
  );

  // --- edits ---------------------------------------------------------------------

  /**
   * Sends one batch. A milestone given days asks first (plan 13.1); a row
   * added by the batch opens for naming once the server has given it an id.
   */
  const commit = useCallback(
    (ops: ProgramOp[], o: CommitOpts = {}, confirmedMilestone = false) => {
      const cur = viewRef.current;
      if (!cur) return;
      if (!confirmedMilestone) {
        for (const op of ops) {
          const a = op.id ? cur.activities.find((x) => x.id === op.id) : undefined;
          if ((op.op === "resize" || op.op === "update") && a?.kind === "milestone" && typeof op.duration_days === "number" && op.duration_days > 0) {
            setDialog({ kind: "milestone", ops, label: a.label, days: op.duration_days, o });
            return;
          }
        }
      }
      apply((v) => programApi.ops(projectId, v!, ops), {
        optimistic: o.optimistic,
        undoable: true,
        note: o.note,
        refused: o.refused,
        then: (next) => {
          if (o.added) {
            const id = next.temp_ids?.[o.added.temp];
            if (id) {
              if (o.added.parent && !expanded.has(o.added.parent)) setOpen(new Set([...expanded, o.added.parent]));
              setSel({ ids: [id], anchor: id, base: [] });
              setFocus({ id, col: "name" });
              setEditing({ id, col: "name", error: "" });
            }
          }
          o.then?.(next);
        },
      });
    },
    [apply, projectId, expanded, setOpen],
  );

  const undo = useCallback(() => apply((v) => programApi.undo(projectId, v!), { note: "Undone." }), [apply, projectId]);
  const redo = useCallback(() => apply((v) => programApi.redo(projectId, v!), { note: "Redone." }), [apply, projectId]);

  const linkChain = useCallback(() => {
    if (!view) return;
    const ids = inOutline(sel.ids, view.activities);
    if (ids.length < 2) return note("Select two or more rows to link them in sequence.");
    commit([{ op: "link_chain", ids }]);
  }, [view, sel, commit, note]);

  const remove = useCallback(() => {
    if (!view || !sel.ids.length) return;
    // The focused row may go: keep keyboard focus in the program so Undo still works.
    const inChart = !!(document.activeElement as HTMLElement | null)?.closest?.(".pg-chartcol");
    ((inChart ? scroller.current?.querySelector(".pg-chartcol") : gridcol.current) as HTMLElement | null)?.focus({ preventScroll: true });
    const plan = deletePlan(sel.ids, view.activities, view.links);
    if (plan.stages.length || plan.holds.length) {
      setReason("");
      setDialog({ kind: "delete", ids: [...sel.ids], plan });
    } else commit([{ op: "delete", ids: [...sel.ids] }], { note: deletedNote(plan) });
  }, [view, sel, commit]);

  const nudge = useCallback((days: number) => sel.ids.length && commit([{ op: "move", ids: [...sel.ids], days }]), [sel, commit]);

  const tier = useCallback(
    (op: "indent" | "outdent") => view && sel.ids.length && commit([{ op, ids: inOutline(sel.ids, view.activities) }]),
    [view, sel, commit],
  );

  const reorder = useCallback(
    (dir: -1 | 1) => {
      if (!view || !sel.ids.length) return;
      const op = reorderStep(sel.ids, view.activities, dir);
      if (op) commit([op]);
      else note(sel.ids.some((id) => model?.byId.get(id)?.parent_id) && sel.ids.some((id) => !model?.byId.get(id)?.parent_id) ? "Move stages or activities, not both at once." : "It can't move further that way.");
    },
    [view, sel, commit, note, model],
  );

  /** A row's add op placed directly below it. */
  const addBelow = (r: Row, temp: string): ProgramOp =>
    r.stage ? { op: "add", temp_id: temp, after_id: r.id, label: "New stage" } : { op: "add", temp_id: temp, parent_id: r.parentId!, after_id: r.id, label: "New activity" };

  const onEdit = useCallback(
    (id: string, col: EditCol) => {
      const r = rows.find((x) => x.id === id);
      if (!r) return;
      if (col === "duration" && r.hasChildren) return note("A stage with activities takes its length from them.");
      setFocus({ id, col });
      setEditing({ id, col, error: "" });
    },
    [rows, note],
  );

  const onCommit = useCallback(
    (id: string, col: EditCol, text: string, key: CommitKey) => {
      if (!view || !model) return;
      const r = rows.find((x) => x.id === id);
      if (!r) return setEditing(null);
      const keep = (msg: string) => setEditing({ id, col, error: msg });
      const at = rows.indexOf(r);
      const next = (k: CommitKey) => {
        if (k === "tab" || k === "shift-tab") {
          const to = rows[at + (k === "tab" ? 1 : -1)];
          if (to) {
            setFocus({ id: to.id, col: "name" });
            setSel({ ids: [to.id], anchor: to.id, base: [] });
            setEditing({ id: to.id, col: "name", error: "" });
            reveal(at + (k === "tab" ? 1 : -1));
            return;
          }
        }
        setEditing(null);
        focusNext.current = { kind: "cell", id };
      };
      if (col === "name") {
        const label = text.trim();
        if (label.length < 1 || label.length > 200) return keep("Names are 1 to 200 characters.");
        if (label !== r.act.label) {
          const optimistic = { ...view, activities: view.activities.map((a) => (a.id === id ? { ...a, label } : a)) };
          commit([{ op: "update", id, label }], { optimistic });
        }
        return next(key);
      }
      if (col === "duration") {
        let days: number | null = null;
        if (text.trim() !== "") {
          const d = parseDuration(text, model.cal.perWeek);
          if (!d.ok) return keep(d.error);
          if (d.value.low !== d.value.high) return keep("Type one duration, for example 5, 5d or 3w.");
          days = d.value.low;
        }
        setEditing(null);
        focusNext.current = { kind: "cell", id };
        const many = selected.has(id) && sel.ids.length > 1;
        if (many) {
          const ids = sel.ids.filter((x) => !rows.find((y) => y.id === x)?.hasChildren);
          if (ids.length > 1) return setDialog({ kind: "durations", ids, days });
        }
        if (days !== r.act.duration_days) commit([{ op: "update", id, duration_days: days }]);
        if (key === "tab") setEditing({ id, col: "predecessors", error: "" });
        return;
      }
      // Predecessors: typed rows become link adds, updates and deletes.
      const refs = parsePredecessors(text, model.cal.perWeek);
      if (!refs.ok) return keep(refs.error);
      const diff = predecessorOps(id, refs.value, view.activities, view.links);
      if ("error" in diff) return keep(diff.error);
      const ops = [...diff.ops];
      setEditing(null);
      // Enter on the last cell adds a row below (plan 9.5).
      if (key === "enter") {
        ops.push(addBelow(r, "below-1"));
        commit(ops, { added: { temp: "below-1", parent: r.stage ? null : r.parentId } });
      } else {
        focusNext.current = { kind: "cell", id };
        if (ops.length) commit(ops);
      }
    },
    [view, model, rows, commit, selected, sel, reveal],
  );

  const onCancel = useCallback(() => {
    if (editing) focusNext.current = { kind: "cell", id: editing.id };
    setEditing(null);
  }, [editing]);

  const onFocusCell = useCallback((id: string, col: string) => setFocus((f) => (f && f.id === id && f.col === col ? f : { id, col })), []);

  const onCatalogue = useCallback((id: string, stage: string) => stage !== "other" && commit([{ op: "update", id, catalogue_stage_id: stage }]), [commit]);

  // --- dragging grid rows to reorder (plan 9.4) --------------------------------

  const onHandleDown = useCallback(
    (e: ReactPointerEvent<HTMLElement>, id: string) => {
      if (!view || (e.pointerType === "mouse" && e.button !== 0)) return;
      e.preventDefault();
      e.stopPropagation();
      const isStage = (x: string) => !model?.byId.get(x)?.parent_id;
      const ids = selected.has(id) ? inOutline(sel.ids.filter((x) => isStage(x) === isStage(id)), view.activities) : [id];
      if (!selected.has(id)) setSel({ ids: [id], anchor: id, base: [] });
      const col = gridcol.current;
      const line = insert.current;
      if (!col || !line) return;
      const y0 = e.clientY;
      let started = false;
      let drop: ReturnType<typeof dropTarget> = null;
      const hide = () => line.setAttribute("hidden", "");
      const stop = () => {
        window.removeEventListener("pointermove", move);
        window.removeEventListener("pointerup", up);
        window.removeEventListener("pointercancel", cancel);
        window.removeEventListener("keydown", key, true);
        hide();
        if (started) releaseDrag();
      };
      const move = (ev: PointerEvent) => {
        if (!started && Math.abs(ev.clientY - y0) < 4) return;
        if (!started) {
          started = true;
          dragging.current = true;
        }
        drop = dropTarget(rows, ids, ev.clientY - col.getBoundingClientRect().top, rowH, view.activities);
        if (!drop) return hide();
        line.removeAttribute("hidden");
        line.className = drop.line >= 0 ? "pg-insert" : "pg-insert is-into";
        line.style.top = `${drop.line >= 0 ? drop.line * rowH - 1 : drop.into * rowH}px`;
        line.style.height = drop.line >= 0 ? "2px" : `${rowH}px`;
      };
      const up = () => {
        stop();
        if (started && drop) commit([drop.op]);
      };
      const cancel = () => stop();
      const key = (ev: globalThis.KeyboardEvent) => {
        if (ev.key !== "Escape") return;
        ev.preventDefault();
        ev.stopPropagation();
        drop = null;
        stop();
      };
      window.addEventListener("pointermove", move);
      window.addEventListener("pointerup", up);
      window.addEventListener("pointercancel", cancel);
      window.addEventListener("keydown", key, true);
    },
    [view, model, selected, sel, rows, rowH, commit, releaseDrag],
  );

  // --- the chart's pointer gestures (chart/useDrag.ts) ---------------------------

  const drag = useDrag({
    view: view ?? ({ activities: [], links: [] } as unknown as ProgramView),
    rows,
    times: model?.times ?? {},
    v,
    cal: model?.cal ?? new Cal({}),
    selIds: sel.ids,
    overlay: () => overlay.current,
    scroller: scrollEl,
    select: onSelect,
    addToSelection: (ids) => setSel((s) => {
      const all = [...s.ids, ...ids.filter((x) => !s.ids.includes(x))];
      return { ids: all, anchor: s.anchor ?? all[0] ?? null, base: all };
    }),
    clearSelection: () => {
      setSel(emptySelection);
      (scroller.current?.querySelector(".pg-chartcol") as HTMLElement | null)?.focus({ preventScroll: true });
    },
    commit: (ops, o) => commit(ops, o ?? {}),
    openLink: (id, x, y) => setLinkPop({ id, x, y }),
    setDragging: (on) => (on ? (dragging.current = true) : releaseDrag()),
  });

  // --- print: every row, no virtualisation (plan 9.7) ------------------------

  useEffect(() => {
    if (!active) return;
    const before = () => {
      setPrintFirst(firstPrintRows(printReserve()));
      setPrinting(true);
    };
    const after = () => setPrinting(false);
    window.addEventListener("beforeprint", before);
    window.addEventListener("afterprint", after);
    return () => {
      window.removeEventListener("beforeprint", before);
      window.removeEventListener("afterprint", after);
    };
  }, [active]);

  // Once the print pages exist under print styles, measure where the first
  // one starts and give it only the rows that fit below the summary.
  useLayoutEffect(() => {
    if (!printing) return;
    const n = firstPrintRows(printReserve());
    if (n !== printFirst) setPrintFirst(n);
  }, [printing, printFirst]);

  // --- keyboard (plan 9.5) ----------------------------------------------------

  function moveFocus(dRow: number, dCol: number, extend: boolean, inChart: string | null) {
    const id = inChart ?? focus?.id ?? sel.anchor ?? order[0];
    if (!id) return;
    const i = order.indexOf(id);
    const j = Math.max(0, Math.min(order.length - 1, (i < 0 ? 0 : i) + dRow));
    const to = order[j];
    if (inChart) {
      setSel((s) => clickRow(s, to, order, { shift: extend }));
      focusNext.current = { kind: "bar", id: to };
      reveal(j);
      return;
    }
    const cols = narrow ? ["name"] : COLS;
    const c = Math.max(0, Math.min(cols.length - 1, cols.indexOf(focus?.col ?? "name") + dCol));
    setFocus({ id: to, col: cols[c] });
    if (dRow) setSel((s) => clickRow(s, to, order, { shift: extend }));
    focusNext.current = { kind: "cell", id: to };
    reveal(j);
  }

  // Links take focus from the bar they touch: Left and Right step through
  // that bar's drawn links, Enter opens the popover, Escape goes back.
  const linkFrom = useRef<string | null>(null);

  function focusBar(id: string) {
    linkFrom.current = null;
    scroller.current?.querySelector<SVGGElement>(`.pg-layer-bars [data-id="${id}"]`)?.focus({ preventScroll: true });
  }

  function stepLink(barId: string, fromLink: string | null, dir: 1 | -1) {
    const list = Array.from(scroller.current?.querySelectorAll<SVGGElement>(`.pg-layer-links [data-pred="${barId}"], .pg-layer-links [data-succ="${barId}"]`) ?? []);
    if (!list.length) return;
    const i = fromLink ? list.findIndex((el) => el.dataset.id === fromLink) + dir : dir > 0 ? 0 : list.length - 1;
    if (i < 0 || i >= list.length) return focusBar(barId);
    linkFrom.current = barId;
    list[i].focus({ preventScroll: true });
  }

  function closeLinkPop(to: "link" | "bar") {
    const p = linkPop;
    setLinkPop(null);
    if (!p?.back) return;
    const back = p.back;
    requestAnimationFrame(() => {
      const el = to === "link" ? scroller.current?.querySelector<SVGGElement>(`.pg-layer-links [data-id="${p.id}"]`) : null;
      if (el) {
        linkFrom.current = back;
        el.focus({ preventScroll: true });
      } else focusBar(back);
    });
  }

  function onKeyDown(e: KeyboardEvent) {
    const t = e.target as HTMLElement;
    if (t.closest("input, textarea, select, dialog, .pg-pop")) return;
    const mod = e.ctrlKey || e.metaKey;
    const k = e.key;
    // Row keys (Delete, arrows, Enter, Tab, Ctrl+A, Ctrl+L) act only with
    // focus in the grid or the chart: Backspace on a toolbar button must not
    // delete the selected row. View keys (undo, redo, collapse, expand, the
    // fits, Escape) also work from elsewhere in the section, but never on a
    // control or inside a panel, where the key belongs to that control.
    if (!t.closest(".pg-gridcol, .pg-chartcol") || t.closest("[role=separator]")) {
      if (t.closest("button, a, summary, [role=radio], [role=separator], .pg-panel, .pg-strip, .pg-toolbar")) return;
      const viewKey = (mod && ["[", "]", "z", "Z", "y", "Y"].includes(k)) || (!mod && !e.altKey && (k === "f" || k === "F")) || k === "Escape";
      if (!viewKey) return;
    }
    const bar = (t.closest(".pg-layer-bars [data-id]") as HTMLElement | null)?.dataset.id ?? null;
    const cell = t.closest(".pg-gridcol [data-col]") as HTMLElement | null;
    const link = t.closest(".pg-layer-links [data-link]") as SVGGElement | null;
    const act = (fn: () => void) => {
      e.preventDefault();
      fn();
    };
    if (link && !mod && !e.altKey) {
      const from = linkFrom.current ?? link.dataset.pred!;
      if (k === "Enter" || k === " ") {
        const b = link.getBBox();
        return act(() => setLinkPop({ id: link.dataset.id!, x: b.x + b.width / 2, y: b.y + b.height / 2, back: from }));
      }
      if (k === "ArrowLeft" || k === "ArrowRight") return act(() => stepLink(from, link.dataset.id!, k === "ArrowRight" ? 1 : -1));
      if (k === "Escape" || k === "ArrowUp" || k === "ArrowDown") return act(() => focusBar(from));
      // A link is deleted from its popover, never by a row key.
      if (k === "Delete" || k === "Backspace" || k === "Tab" || k === "F2") return;
    }
    // The chart's own tab stop: Up or Down goes to the selected row's bar.
    if (t.classList.contains("pg-chartcol") && !mod && !e.altKey && (k === "ArrowUp" || k === "ArrowDown")) {
      const start = sel.anchor && order.includes(sel.anchor) ? sel.anchor : order[0];
      if (start) act(() => moveFocus(0, 0, e.shiftKey, start));
      return;
    }
    if (mod && k === "[") act(collapseAll);
    else if (mod && k === "]") act(expandAll);
    else if (mod && (k === "a" || k === "A")) act(() => setSel(selectAll(order)));
    else if (mod && (k === "z" || k === "Z")) act(e.shiftKey ? redo : undo);
    else if (mod && (k === "y" || k === "Y")) act(redo);
    else if (mod && (k === "l" || k === "L")) act(linkChain);
    else if (!mod && !e.altKey && (k === "f" || k === "F")) act(e.shiftKey ? fitSelection : fitScreen);
    else if (k === "Delete" || k === "Backspace") act(remove);
    else if (e.altKey && (k === "ArrowUp" || k === "ArrowDown")) act(() => reorder(k === "ArrowUp" ? -1 : 1));
    else if (e.shiftKey && (k === "ArrowLeft" || k === "ArrowRight")) act(() => nudge(k === "ArrowLeft" ? -1 : 1));
    else if (k === "ArrowUp" || k === "ArrowDown") act(() => moveFocus(k === "ArrowUp" ? -1 : 1, 0, e.shiftKey, bar));
    else if ((k === "ArrowLeft" || k === "ArrowRight") && cell) act(() => moveFocus(0, k === "ArrowLeft" ? -1 : 1, false, null));
    else if ((k === "ArrowLeft" || k === "ArrowRight") && bar && !mod && !e.altKey) act(() => stepLink(bar, null, k === "ArrowRight" ? 1 : -1));
    else if ((k === "Enter" || k === "F2") && cell && focus) {
      const col = focus.col as EditCol;
      if (col === "name" || col === "duration" || col === "predecessors") act(() => onEdit(focus.id, col));
    } else if (k === "Tab" && cell && focus?.col === "name" && sel.ids.length) act(() => tier(e.shiftKey ? "outdent" : "indent"));
    else if (k === "Escape") {
      setLinkPop(null);
      setSel(emptySelection);
    }
  }

  // --- divider ----------------------------------------------------------------

  function onDividerDown(e: ReactPointerEvent<HTMLDivElement>) {
    if (e.button !== 0) return;
    e.preventDefault();
    divider.current = { x: e.clientX, w: gw };
    try {
      e.currentTarget.setPointerCapture(e.pointerId);
    } catch {
      // The pointer may already be gone; moves still resize.
    }
  }
  function onDividerMove(e: ReactPointerEvent<HTMLDivElement>) {
    if (!divider.current) return;
    setGridW(Math.max(GRID_W_MIN, Math.min(box.w - 200, divider.current.w + e.clientX - divider.current.x)));
  }
  function onDividerKey(e: KeyboardEvent<HTMLDivElement>) {
    const step = e.shiftKey ? 48 : 16;
    if (e.key === "ArrowLeft") setGridW(Math.max(GRID_W_MIN, gw - step));
    else if (e.key === "ArrowRight") setGridW(Math.min(box.w - 200, gw + step));
  }

  // --- render -------------------------------------------------------------------

  if (view === undefined) {
    return (
      <section className="program" aria-label="Program">
        <header className="program-head">
          <h1>Program</h1>
        </header>
        {error ? <Failure error={error} retry={load} /> : <p className="program-empty">Opening the program…</p>}
      </section>
    );
  }

  if (view === null) {
    return (
      <section className="program" aria-label="Program">
        <header className="program-head">
          <h1>Program</h1>
        </header>
        {error && <Failure error={error} retry={load} />}
        <div className="program-empty">
          <p>No program yet. SiteWise builds one from the Profile, the included work and the Procurement packages; nothing is guessed, and unknown durations stay unknown.</p>
          <button type="button" className="btn pg-build" disabled={busy} onClick={() => apply(() => programApi.prepare(projectId))}>
            {busy ? "Building…" : "Build program"}
          </button>
        </div>
      </section>
    );
  }

  const m = model!;
  const d = docs!;
  const finish = view.computed.finish;
  const typical = shownFinish(m.cal, finish.typical, finish.typical_index);
  const low = shownFinish(m.cal, finish.low, finish.low_index);
  const high = shownFinish(m.cal, finish.high, finish.high_index);
  // Evidence that lost its stage shows as its own notice with Dismiss (7.7).
  const checks = (view.checks ?? []).filter((c) => c.code !== "evidence_without_stage");
  const orphaned = (view.checks ?? []).some((c) => c.code === "evidence_without_stage") ? d.orphans : [];
  const acceptCount = m.proposed + d.live.length;
  const labelOf = (id: string) => m.byId.get(id)?.label ?? "";
  const stageChoices = view.activities.filter((a) => !a.parent_id).map((a) => ({ id: a.id, label: a.label }));

  // Program files: dropped on the view or picked with Import, filed through
  // intake, then read in the background (plan 10.2).
  const importFiles = (files: File[]) => {
    const ok = files.filter((f) => importable(f.name));
    if (ok.length < files.length) setToast({ text: "Only PDF, DOCX and XLSX program files can be imported.", alert: true });
    if (!ok.length) return;
    setReadingsOpen(true);
    setToast({ text: `Filing ${ok.map((f) => f.name).join(", ")}…` });
    Promise.all(ok.map((f) => programApi.importFile(projectId, f))).then(
      () => load(),
      (e) => {
        if (e instanceof ApiError && e.status === 401) return onSignedOut();
        setToast({ text: e instanceof Error ? `Import failed: ${e.message}` : "Import failed.", alert: true });
      },
    );
  };
  const hasFiles = (e: DragEvent) => Array.from(e.dataTransfer.types).includes("Files");
  const q = view.question;
  const askStart = !view.program.start_date;
  const anyOpen = rows.some((r) => !r.stage);
  const scaleProps = { cal: m.cal, v, lo, hi, scale, width: chartW, version: view.program.version };
  const zone = coarse ? ZONE_COARSE : ZONE_FINE;
  const popLink = linkPop ? view.links.find((l) => l.id === linkPop.id) : undefined;

  const chart = (from: number, to: number, w: number, vp: Viewport, ticksLo: number, ticksHi: number, onScreen: boolean) => {
    const links = printing ? (showLinks ? drawnLinks(view.links, rows, m.parentOf, from, to) : []) : drawn;
    return (
      <svg
        className="pg-chart"
        width={w}
        height={rows.length * vp.rowH}
        role="presentation"
        ref={onScreen ? drag.svg : undefined}
        onPointerDown={onScreen ? drag.onPointerDown : undefined}
        onPointerMove={onScreen ? drag.onPointerMove : undefined}
        onPointerUp={onScreen ? drag.onPointerUp : undefined}
        onPointerCancel={onScreen ? drag.onPointerCancel : undefined}
      >
        <TimescaleGrid {...scaleProps} v={vp} lo={ticksLo} hi={ticksHi} width={w} plain={m.plain} height={rows.length * vp.rowH} today={m.today} />
        <Bars rows={rows} times={m.times} v={vp} first={from} last={to} selected={onScreen ? selected : new Set()} conflicts={m.conflicts} relative={m.cal.relative} version={view.program.version} drivers={m.drivers} zone={zone} marks={d.marks} released={d.released} />
        <Links drawn={links} rows={rows} times={m.times} v={vp} driving={m.driving} version={view.program.version} />
        {onScreen && <Overlay ref={overlay} height={rows.length * vp.rowH} />}
      </svg>
    );
  };

  return (
    <section
      className={`program${dropping ? " is-dropping" : ""}`}
      aria-label="Program"
      onKeyDown={onKeyDown}
      onDragOver={(e) => {
        if (!hasFiles(e)) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = "copy";
        if (!dropping) setDropping(true);
      }}
      onDragLeave={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setDropping(false);
      }}
      onDrop={(e) => {
        if (!hasFiles(e)) return;
        e.preventDefault();
        setDropping(false);
        importFiles(Array.from(e.dataTransfer.files));
      }}
    >
      <header className="program-head">
        <h1>Program</h1>
        {sel.ids.length > 0 && <span className="pg-chip">{sel.ids.length} selected</span>}
        {dropping && <span className="pg-chip pg-drop-hint">Drop a PDF, DOCX or XLSX program to read its dates</span>}
      </header>

      <div className="pg-toolbar" role="toolbar" aria-label="Program tools">
        <div className="pg-tools">
          <button type="button" className="btn btn-small" onClick={collapseAll} title="Ctrl+[">Collapse all</button>
          <button type="button" className="btn btn-small" onClick={expandAll} title="Ctrl+]">Expand all</button>
        </div>
        <div className="pg-tools">
          <button type="button" className="btn btn-small" onClick={fitScreen} title="F">Fit to screen</button>
          <button type="button" className="btn btn-small" onClick={fitSelection} title="Shift+F">Fit to selection</button>
          <button type="button" className="btn btn-small" onClick={fitRows}>Fit height</button>
          <button type="button" className="btn btn-small" aria-pressed={showLinks} onClick={() => setShowLinks((s) => !s)}>
            Links {showLinks ? "on" : "off"}
          </button>
          <button type="button" className="btn btn-small pg-icon" aria-label="Zoom out" onClick={() => zoom(1 / 1.5)}>−</button>
          <button type="button" className="btn btn-small pg-icon" aria-label="Zoom in" onClick={() => zoom(1.5)}>+</button>
          <button type="button" className="btn btn-small" onClick={goToday}>Today</button>
        </div>
        <div className="pg-tools">
          <button type="button" className="btn btn-small" onClick={undo} title="Ctrl+Z">Undo</button>
          <button type="button" className="btn btn-small" onClick={redo} title="Ctrl+Shift+Z">Redo</button>
          <button type="button" className="btn btn-small" disabled={sel.ids.length < 2} onClick={linkChain} title="Ctrl+L">Link in sequence</button>
        </div>
        <div className="pg-tools">
          {acceptCount > 0 && (
            <button type="button" className="btn btn-small pg-accept" disabled={busy} onClick={() => apply((cur) => programApi.acceptAll(projectId, cur!), { undoable: true })}>
              Accept all ({acceptCount})
            </button>
          )}
          {view.changes_pending > 0 && (
            <button type="button" className="btn btn-small pg-accept" aria-expanded={reviewOpen} onClick={() => setReviewOpen((o) => !o)}>
              Review changes ({view.changes_pending})
            </button>
          )}
          {d.imports.map((imp) => (
            <button key={imp.id} type="button" className={`btn btn-small pg-import-text is-${imp.status}`} aria-expanded={readingsOpen} onClick={() => setReadingsOpen((o) => !o)}>
              {imp.text}
            </button>
          ))}
          {d.proposalReadings > 0 && (
            <button type="button" className="btn btn-small pg-import-text" aria-expanded={readingsOpen} onClick={() => setReadingsOpen((o) => !o)}>
              {d.proposalReadings} reading{d.proposalReadings === 1 ? "" : "s"} from awarded proposals
            </button>
          )}
          <button type="button" className="btn btn-small" title="Read the dates of a PDF, DOCX or XLSX program, or drop one on the Program" onClick={() => fileInput.current?.click()}>
            Import
          </button>
          <input
            ref={fileInput}
            type="file"
            accept=".pdf,.docx,.xlsx"
            multiple
            hidden
            aria-label="Import a program file"
            onChange={(e) => {
              const files = Array.from(e.target.files ?? []);
              e.target.value = "";
              importFiles(files);
            }}
          />
          <a className="btn btn-small" href={programApi.exportHref(projectId, anyOpen ? "expanded" : "collapsed")} download>
            Export
          </a>
          <button type="button" className="btn btn-small" onClick={() => window.print()}>Print</button>
        </div>
      </div>

      {error && <Failure error={error} retry={load} />}
      {view.changes_pending > 0 && !reviewOpen && <ReviewBanner words={reviewWords(changes?.counts, view.changes_pending)} onReview={() => setReviewOpen(true)} />}
      {reviewOpen && (
        <ReviewPanel
          changes={changes?.list ?? null}
          busy={busy}
          onClose={() => setReviewOpen(false)}
          onAdoptAll={() => apply((cur) => programApi.adoptAll(projectId, cur!), { undoable: true, note: "Adopted every change that defaults to Adopt; edited rows wait for your choice." })}
          onApply={(decisions) => {
            const adopted = decisions.filter((x) => x.choice === "adopt").length;
            apply((cur) => programApi.decide(projectId, cur!, decisions), {
              undoable: adopted > 0,
              note: adopted ? `Adopted ${adopted} change${adopted === 1 ? "" : "s"}; kept ${decisions.length - adopted}.` : "Kept the program as it is.",
            });
          }}
        />
      )}
      {orphaned.map((g) => (
        <EvidenceNotice
          key={g.key}
          label={view.catalogue_stages?.find((c) => c.id === g.catalogueStageId)?.label ?? g.catalogueStageId!}
          busy={busy}
          onDismiss={() => apply((cur) => programApi.dismissReadings(projectId, cur!, g.ids), { undoable: true, note: "Dismissed." })}
        />
      ))}
      {readingsOpen && (d.imports.length > 0 || d.proposalReadings > 0) && (
        <ReadingsPanel
          imports={d.imports}
          groups={d.live}
          labelOf={labelOf}
          stages={stageChoices}
          busy={busy}
          onClose={() => setReadingsOpen(false)}
          onAccept={(ids) => apply((cur) => programApi.accept(projectId, cur!, ids), { undoable: true, note: "Accepted." })}
          onDismiss={(ids) => apply((cur) => programApi.dismissReadings(projectId, cur!, ids), { undoable: true, note: "Dismissed." })}
          onAddRow={(importId, row, parentId) =>
            apply((cur) => programApi.addRows(projectId, cur!, [{ import_id: importId, row, parent_id: parentId }]), { undoable: true, note: "Added." })
          }
          onAddLinks={(links) =>
            commit(
              links.map((l) => ({ op: "link_add", pred_id: l.pred_id, succ_id: l.succ_id, type: l.type, lag_days: l.lag_days, lag_unit: l.lag_unit })),
              { note: `Added ${links.length} link${links.length === 1 ? "" : "s"} from the file.` },
            )
          }
        />
      )}
      {d.holds.length > 0 && (
        <HoldPointsPanel
          holds={d.holds}
          info={d.holdInfo}
          busy={busy}
          onRelease={(id, readingId) => apply((cur) => programApi.release(projectId, cur!, id, readingId), { undoable: true, note: "Marked released." })}
        />
      )}

      {(q || askStart) && (
        <div className="pg-strip" role="region" aria-label="Question">
          {q && (
            <div className="pg-q">
              <span className="pg-q-label">{q.label}</span>
              {q.number ? (
                <NumberAnswer disabled={busy} onSave={(n) => apply((cur) => programApi.answer(projectId, cur!, q.key, n), { undoable: true, then: onProfileChanged })} />
              ) : (
                q.options.map((label, i) => (
                  <button
                    key={label}
                    type="button"
                    className="btn btn-small"
                    disabled={busy}
                    // An answer's program changes are one undoable batch (plan 7.3); "Not sure" changes nothing.
                    onClick={() => apply((cur) => programApi.answer(projectId, cur!, q.key, q.values[i] ?? "not_sure"), i < q.values.length ? { undoable: true, then: onProfileChanged } : {})}
                  >
                    {label}
                  </button>
                ))
              )}
              {q.narrows_by_days > 0 && <span className="pg-q-why">Narrows the finish by up to {q.narrows_by_days} working days</span>}
            </div>
          )}
          {askStart && (
            <div className="pg-q">
              {view.start_suggestion ? (
                <button type="button" className="btn btn-small" disabled={busy} onClick={() => apply((cur) => programApi.setStart(projectId, cur!, view.start_suggestion))}>
                  Start on {fmtDate(view.start_suggestion)} (consent date)
                </button>
              ) : (
                <span className="pg-q-label">No start date yet.</span>
              )}
              <label className="pg-start">
                <span>Start date</span>
                <input type="date" className="input" value={startInput} onChange={(e) => setStartInput(e.target.value)} />
              </label>
              <button
                type="button"
                className="btn btn-small"
                disabled={busy || !/^\d{4}-\d{2}-\d{2}$/.test(startInput)}
                onClick={() => apply((cur) => programApi.setStart(projectId, cur!, startInput))}
              >
                Set start
              </button>
            </div>
          )}
        </div>
      )}

      <div className="pg-summary">
        <div className="pg-finish">
          <span className="pg-k">Finish</span>
          {typical ? (
            <strong className="pg-v" data-testid="program-finish">{typical}</strong>
          ) : (
            <strong className="pg-v is-unknown" data-testid="program-finish">Not known yet</strong>
          )}
          {low && high && low !== high && (
            <span className="pg-range" data-testid="program-range">
              range {low} – {high}
            </span>
          )}
        </div>
        <div className="pg-status" data-testid="program-status">{view.computed.status.words}</div>
        {view.skeleton.label && (
          <div className="pg-skeleton">
            <span className="pg-k">Stages</span>
            <span data-testid="program-skeleton">{view.skeleton.reason || view.skeleton.label}</span>
          </div>
        )}
        {m.cal.relative && <div className="pg-relative">No start date: dates show as weeks from the start.</div>}
        {view.computed.cycle && <div className="pg-alert" role="alert">The links form a loop, so no dates can be worked out.</div>}
        {checks.length > 0 && (
          <div className="pg-checks">
            <button type="button" className="pg-checks-toggle" aria-expanded={checksOpen} onClick={() => setChecksOpen((o) => !o)}>
              {checks.length} check{checks.length === 1 ? "" : "s"}
            </button>
            {checksOpen && (
              <ul>
                {checks.map((c, i) => (
                  <li key={`${c.code}-${i}`}>{c.message}</li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>

      {tooMany && <div className="pg-note" role="status">Too many rows to fit; try collapsing stages.</div>}

      <div
        className={`pg-body${narrow ? " is-narrow" : ""}`}
        ref={scroller}
        onScroll={onScroll}
        hidden={printing}
        style={{ ["--pg-grid-w" as string]: `${gw}px`, ["--pg-row-h" as string]: `${rowH}px` }}
      >
        <div className="pg-canvas" style={{ width: gw + chartW, height: SCALE_H + height }}>
          <div className="pg-corner" aria-hidden="true">
            <GridHeader narrow={narrow} />
          </div>
          <div className="pg-scale">
            <TimescaleHeader {...scaleProps} />
          </div>
          <div className="pg-gridcol" ref={gridcol} role="treegrid" tabIndex={0} aria-label="Program rows" aria-rowcount={rows.length + 1} aria-multiselectable="true" style={{ height }}>
            <div role="row" className="sr-only" aria-rowindex={1}>
              {(narrow ? ["Name"] : GRID_COLUMNS).map((c) => (
                <span key={c} role="columnheader">{c}</span>
              ))}
            </div>
            <Grid
              rows={rows}
              times={m.times}
              cal={m.cal}
              first={first}
              last={last}
              rowH={rowH}
              narrow={narrow}
              selected={selected}
              version={view.program.version}
              onToggle={toggle}
              onSelect={onSelect}
              preds={m.preds}
              editing={editing}
              focus={focus}
              catalogue={view.catalogue_stages}
              onEdit={onEdit}
              onCommit={onCommit}
              onCancel={onCancel}
              onFocusCell={onFocusCell}
              onHandleDown={onHandleDown}
              onCatalogue={onCatalogue}
              notes={d.notes}
            />
            <div ref={insert} className="pg-insert" hidden aria-hidden="true" />
            {!narrow && (
              <div
                className="pg-divider"
                role="separator"
                aria-orientation="vertical"
                aria-label="Resize grid"
                aria-valuenow={gw}
                tabIndex={0}
                onPointerDown={onDividerDown}
                onPointerMove={onDividerMove}
                onPointerUp={() => (divider.current = null)}
                onPointerCancel={() => (divider.current = null)}
                onKeyDown={onDividerKey}
              />
            )}
          </div>
          <div className="pg-chartcol" tabIndex={0} role="group" aria-label="Program chart: Up and Down move between bars, Left and Right between a bar's links">
            {!printing && chart(first, last, chartW, v, lo, hi, true)}
            {popLink && linkPop && (
              <LinkPopover
                key={popLink.id}
                x={linkPop.x}
                y={linkPop.y}
                type={popLink.type}
                lag={popLink.lag_days}
                unit={popLink.lag_unit}
                title={`${m.byId.get(popLink.pred_id)?.label ?? ""} → ${m.byId.get(popLink.succ_id)?.label ?? ""}`}
                onClose={() => closeLinkPop("link")}
                onSave={(type, lag, unit) => {
                  closeLinkPop("link");
                  if (type !== popLink.type || lag !== popLink.lag_days || unit !== popLink.lag_unit) commit([{ op: "link_update", id: popLink.id, type, lag_days: lag, lag_unit: unit }]);
                }}
                onDelete={() => {
                  closeLinkPop("bar");
                  commit([{ op: "link_delete", ids: [popLink.id] }]);
                }}
              />
            )}
          </div>
        </div>
      </div>

      {printing && <PrintPages rows={rows} times={m.times} cal={m.cal} span={span} render={chart} version={view.program.version} firstRows={printFirst} />}

      <div className={`pg-toast${toast?.alert ? " is-alert" : ""}`} role={toast?.alert ? "alert" : "status"} aria-live={toast?.alert ? "assertive" : "polite"}>
        {toast?.text}
        {toast?.undo && (
          <button type="button" className="pg-toast-undo" onClick={() => (setToast(null), undo())}>
            Undo
          </button>
        )}
      </div>

      {dialog?.kind === "delete" && (
        <ConfirmDialog
          title={dialog.plan.holds.length ? "Remove hold point" : "Delete stage"}
          message={deleteWords(dialog.plan)}
          confirmLabel="Delete"
          confirmDisabled={dialog.plan.holds.length > 0 && !reason.trim()}
          onCancel={() => setDialog(null)}
          onConfirm={() => {
            const ids = dialog.ids;
            setDialog(null);
            commit([{ op: "delete", ids, ...(reason.trim() ? { reason: reason.trim() } : {}) }], { note: deletedNote(dialog.plan) });
          }}
        >
          {dialog.plan.holds.length > 0 && (
            <label className="pg-reason">
              <span>Why is it not needed?</span>
              <input className="input" value={reason} maxLength={200} onChange={(e) => setReason(e.target.value)} />
            </label>
          )}
        </ConfirmDialog>
      )}
      {dialog?.kind === "durations" && (
        <ConfirmDialog
          title="Set duration"
          message={`Set the duration of ${dialog.ids.length} rows to ${dialog.days === null ? "unknown" : dialog.days === 0 ? "0 days (milestones)" : `${dialog.days} days`}?`}
          confirmLabel={`Set ${dialog.ids.length} rows`}
          tone="plain"
          onCancel={() => setDialog(null)}
          onConfirm={() => {
            const d = dialog;
            setDialog(null);
            commit(d.ids.map((id) => ({ op: "update", id, duration_days: d.days })));
          }}
        />
      )}
      {dialog?.kind === "milestone" && (
        <ConfirmDialog
          title="Milestone to activity"
          message={`${dialog.label} is a milestone. Giving it ${dialog.days} day${dialog.days === 1 ? "" : "s"} makes it an activity.`}
          confirmLabel="Make it an activity"
          tone="plain"
          onCancel={() => setDialog(null)}
          onConfirm={() => {
            const d = dialog;
            setDialog(null);
            commit(d.ops, d.o, true);
          }}
        />
      )}
    </section>
  );
}

/** "Delete Build and its 3 activities? 4 rows and 6 links are removed." (plan 13.1) */
function deleteWords(p: DeletePlan): string {
  const parts: string[] = [];
  if (p.stages.length) parts.push(`Delete ${p.stages.join(", ")} with ${p.stages.length === 1 ? "its" : "their"} activities? ${p.rows} row${p.rows === 1 ? "" : "s"} and ${p.links} link${p.links === 1 ? "" : "s"} are removed.`);
  if (p.holds.length) parts.push(`${p.holds.join(", ")} ${p.holds.length === 1 ? "is a hold point" : "are hold points"}: say why it is not needed. It is retired, not deleted, and a check stays visible.`);
  parts.push("Undo restores it all.");
  return parts.join(" ");
}

function deletedNote(p: DeletePlan): string {
  return `Deleted ${p.rows} row${p.rows === 1 ? "" : "s"}.`;
}

function Failure({ error, retry }: { error: string; retry: () => void }) {
  return (
    <div className="banner" role="alert">
      <p>{error}</p>
      <button type="button" className="btn btn-small" onClick={retry}>Reload</button>
    </div>
  );
}

function GridHeader({ narrow }: { narrow: boolean }) {
  return (
    <div className="pg-gridhead">
      {(narrow ? (["Name"] as const) : GRID_COLUMNS).map((c) => (
        <span key={c} className={`pg-cell pg-h-${c === "#" ? "num" : c.toLowerCase()}`}>
          {c}
        </span>
      ))}
    </div>
  );
}

function NumberAnswer({ disabled, onSave }: { disabled: boolean; onSave: (v: string) => void }) {
  const [n, setN] = useState("");
  return (
    <>
      <input className="input pg-q-number" inputMode="decimal" aria-label="Answer" value={n} onChange={(e) => setN(e.target.value)} />
      <button type="button" className="btn btn-small" disabled={disabled || !/^\d+(\.\d+)?$/.test(n.trim())} onClick={() => onSave(n.trim())}>
        Save
      </button>
    </>
  );
}

// Click a link line (plan 9.4): type, lag, unit, delete.
function LinkPopover({
  x,
  y,
  type,
  lag,
  unit,
  title,
  onSave,
  onDelete,
  onClose,
}: {
  x: number;
  y: number;
  type: string;
  lag: number;
  unit: string;
  title: string;
  onSave: (type: string, lag: number, unit: string) => void;
  onDelete: () => void;
  onClose: () => void;
}) {
  const [t, setT] = useState(type);
  const [l, setL] = useState(String(lag));
  const [u, setU] = useState(unit);
  const first = useRef<HTMLSelectElement>(null);
  useEffect(() => first.current?.focus(), []);
  const n = Number(l.replace("−", "-"));
  const ok = /^[-−]?\d{1,3}$/.test(l.trim()) && n >= -365 && n <= 365;
  return (
    <div
      className="pg-pop"
      role="dialog"
      aria-label={`Link ${title}`}
      style={{ left: Math.round(x), top: Math.round(y) + 8 }}
      onKeyDown={(e) => {
        e.stopPropagation();
        if (e.key === "Escape") onClose();
        if (e.key === "Enter" && ok) onSave(t, n, u);
      }}
    >
      <p className="pg-pop-title">{title}</p>
      <label>
        <span>Type</span>
        <select ref={first} className="input" value={t} onChange={(e) => setT(e.target.value)}>
          <option value="FS">Finish to start</option>
          <option value="SS">Start to start</option>
          <option value="FF">Finish to finish</option>
          <option value="SF">Start to finish</option>
        </select>
      </label>
      <label>
        <span>Lag</span>
        <input className="input" inputMode="numeric" value={l} onChange={(e) => setL(e.target.value)} aria-invalid={!ok || undefined} />
      </label>
      <label>
        <span>Unit</span>
        <select className="input" value={u} onChange={(e) => setU(e.target.value)}>
          <option value="working">Working days</option>
          <option value="calendar">Calendar days</option>
        </select>
      </label>
      {!ok && <p className="pg-pop-error">Lag is a whole number of days from −365 to 365.</p>}
      <p className="pg-pop-now">Now: {lagText(lag, unit)}</p>
      <div className="pg-pop-actions">
        <button type="button" className="btn btn-small btn-danger" onClick={onDelete}>Delete link</button>
        <button type="button" className="btn btn-small" onClick={onClose}>Cancel</button>
        <button type="button" className="btn btn-small pg-build" disabled={!ok} onClick={() => onSave(t, n, u)}>Save</button>
      </div>
    </div>
  );
}

/**
 * How far down the first printed page the rows start. Under print styles
 * the print block's own position says it; otherwise (the moment before
 * printing) the heading and summary heights plus the shell's title, which
 * over-estimates at a narrower screen width and so stays safe.
 */
function printReserve(): number {
  const pg = document.querySelector<HTMLElement>(".pg-print");
  if (pg && pg.getClientRects().length) return pg.getBoundingClientRect().top + window.scrollY;
  const outer = (sel: string) => {
    const el = document.querySelector<HTMLElement>(sel);
    if (!el) return 0;
    const cs = getComputedStyle(el);
    return el.getBoundingClientRect().height + parseFloat(cs.marginTop) + parseFloat(cs.marginBottom);
  };
  return PRINT_SHELL_H + outer(".program .program-head") + outer(".program .pg-summary");
}

/** Rows that fit on the first page below `reserve` px: at least 4, at most a full page. */
function firstPrintRows(reserve: number): number {
  return Math.max(4, Math.min(PRINT_ROWS, Math.floor((PRINT_PAGE_H - reserve - SCALE_H - 8) / ROW_H)));
}

// A3 landscape pages (plan 9.7): every row in the current view, the
// timescale header repeated on each page, the chart fitted to the page.
function PrintPages({
  rows,
  times,
  cal,
  span,
  render,
  version,
  firstRows,
}: {
  rows: Row[];
  times: Record<string, Times>;
  cal: Cal;
  span: [number, number] | null;
  render: (from: number, to: number, w: number, v: Viewport, lo: number, hi: number, onScreen: boolean) => ReactElement;
  version: number;
  /** Rows on the first page, which shares its height with the summary. */
  firstRows: number;
}) {
  const s = span ?? [0, 20];
  const f = fitSpan(s, PRINT_CHART_W);
  const v: Viewport = { pxPerDay: f.pxPerDay, viewStart: f.viewStart, rowH: ROW_H };
  const pages: [number, number][] = [];
  for (let i = 0; i < Math.max(rows.length, 1); i += i === 0 ? firstRows : PRINT_ROWS) pages.push([i, Math.min(rows.length, i + (i === 0 ? firstRows : PRINT_ROWS))]);
  const scale = scaleFor(v.pxPerDay);
  return (
    <div className="pg-print" aria-hidden="true">
      {pages.map(([from, to]) => (
        <div key={from} className="pg-print-page" style={{ ["--pg-grid-w" as string]: `${PRINT_GRID_W}px`, ["--pg-print-chart-w" as string]: `${PRINT_CHART_W}px` }}>
          <div className="pg-print-head">
            <GridHeader narrow={false} />
            <TimescaleHeader cal={cal} v={v} lo={Math.floor(f.viewStart)} hi={s[1] + 2} scale={scale} width={PRINT_CHART_W} version={version} />
          </div>
          <div className="pg-print-body" style={{ height: (to - from) * v.rowH }}>
            <div className="pg-print-grid" style={{ transform: `translateY(${-from * v.rowH}px)` }}>
              <Grid rows={rows} times={times} cal={cal} first={from} last={to} rowH={v.rowH} narrow={false} selected={new Set()} version={version} onToggle={() => {}} onSelect={() => {}} />
            </div>
            <div className="pg-print-chart" style={{ transform: `translateY(${-from * v.rowH}px)` }}>
              {render(from, to, PRINT_CHART_W, v, Math.floor(f.viewStart), s[1] + 2, false)}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
