// The Program view (plan docs/plans/2026-10-18-program.md section 9):
// toolbar, the one question, the completion header and a split pane with
// the grid on the left and the chart on the right sharing one vertical
// scroll. WP-P7 is read-only apart from Build, Accept all, the question and
// the one-tap start. Every date comes from the server's `computed`; the
// TypeScript engine is kept for WP-P8's drag preview (toInput in
// programApi.ts feeds it the raw inputs).
//
// WP-P8 hooks: `dragging` (a ref) suppresses SSE refetches until release,
// and `releaseDrag` runs the deferred refetch; bar and link groups carry
// data-id for pointer handlers; the Overlay ref exposes the ghost, guide and
// selection-box nodes; `apply` is the one place a write response replaces
// state (use it for POST /program/ops batches).

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type KeyboardEvent, type PointerEvent as ReactPointerEvent, type ReactElement } from "react";
import { ApiError } from "./api";
import { programApi, ProgramConflict, type ProgramView } from "./programApi";
import { Cal } from "./program/calendar";
import { clampPx, fitHeight, fitSpan, PX_DEFAULT, scaleFor, zoomAbout } from "./program/fit";
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
import { Grid, GRID_COLUMNS } from "./program/grid/Grid";
import { SCALE_H, TimescaleGrid, TimescaleHeader } from "./program/chart/Timescale";
import { Bars } from "./program/chart/Bars";
import { Links } from "./program/chart/Links";
import { Overlay, type OverlayHandle } from "./program/chart/Overlay";
import type { Times } from "./program/types";
import "./program.css";

interface Props {
  projectId: string;
  /** Newest program version announced by `program.changed` (plan 8.3). */
  remoteVersion: number;
  /** The view is on screen; a hidden view never takes over printing. */
  active: boolean;
  onSignedOut: () => void;
}

const GRID_W = 520;
const GRID_W_NARROW = 150;
const GRID_W_MIN = 220;
/** Below this pane width the grid shows the name column only (plan 9.1). */
const NARROW = 720;
const X_CHUNK = 256;
/** Rows on one A3 landscape page at 28 px with the header (plan 9.7). */
const PRINT_ROWS = 30;
const PRINT_GRID_W = 320;
const PRINT_CHART_W = 1160;

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

interface Win {
  first: number;
  last: number;
  x0: number;
  x1: number;
}

export function Program({ projectId, remoteVersion, active, onSignedOut }: Props) {
  const [view, setView] = useState<ProgramView | null | undefined>(undefined);
  const [error, setError] = useState("");
  const [toast, setToast] = useState("");
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
  const [checksOpen, setChecksOpen] = useState(false);
  const [reviewOpen, setReviewOpen] = useState(false);
  const [tooltip] = useState<{ x: number; y: number; text: string } | null>(null);
  const [startInput, setStartInput] = useState("");

  const scroller = useRef<HTMLDivElement>(null);
  const overlay = useRef<OverlayHandle>(null);
  const pendingScroll = useRef<{ left?: number; top?: number } | null>(null);
  const fitted = useRef(false);
  const raf = useRef(0);
  const dragging = useRef(false);
  const refetchOnRelease = useRef(false);
  const divider = useRef<{ x: number; w: number } | null>(null);

  const fail = useCallback(
    (e: unknown) => {
      if (e instanceof ApiError && e.status === 401) return onSignedOut();
      setError(e instanceof Error ? e.message : "Could not load the program.");
    },
    [onSignedOut],
  );

  const load = useCallback(() => {
    programApi.read(projectId).then(
      (v) => {
        setView(v);
        setError("");
      },
      (e) => fail(e),
    );
  }, [projectId, fail]);

  useEffect(load, [load]);

  // A newer version from another viewer: refetch now, or on release when a
  // drag is in progress (plan 8.3).
  useEffect(() => {
    if (!view || remoteVersion <= view.program.version) return;
    if (dragging.current) refetchOnRelease.current = true;
    else load();
  }, [remoteVersion, view, load]);

  /** For WP-P8: call when a drag ends to run a refetch deferred during it. */
  const releaseDrag = useCallback(() => {
    dragging.current = false;
    if (refetchOnRelease.current) {
      refetchOnRelease.current = false;
      load();
    }
  }, [load]);
  void releaseDrag;

  /** Replaces state with a write response; a 409 shows the newer version. */
  const apply = useCallback(
    async (write: () => Promise<ProgramView>) => {
      setBusy(true);
      try {
        const next = await write();
        setView(next);
        setError("");
        if (next.toasts?.length) setToast(next.toasts.join(" "));
      } catch (e) {
        if (e instanceof ProgramConflict) {
          setView(e.current);
          setToast(e.message);
        } else fail(e);
      } finally {
        setBusy(false);
      }
    },
    [fail],
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
    return {
      cal,
      plain,
      parentOf,
      withKids,
      today,
      times: view.computed.activities as Record<string, Times>,
      driving: new Set(view.computed.driving_links ?? []),
      conflicts: new Set((view.computed.conflicts ?? []).map((c) => c.activity_id)),
      proposed: view.activities.filter((a) => a.review_status === "proposed").length,
    };
  }, [view]);

  const rows: Row[] = useMemo(() => (view ? outlineRows(view.activities, expanded) : []), [view, expanded]);
  const order = useMemo(() => rows.map((r) => r.id), [rows]);
  useEffect(() => setSel((s) => prune(s, order)), [order]);
  const selected = useMemo(() => new Set(sel.ids), [sel]);

  const span = useMemo(() => (model ? extent(order, model.times, view?.computed.finish.high_index ?? null) : null), [model, order, view]);
  const allSpan = useMemo(
    () => (model && view ? extent(view.activities.map((a) => a.id), model.times, view.computed.finish.high_index) : null),
    [model, view],
  );
  // The chart's x = 0 sits two days before the program start (or the
  // earliest bar), fixed for a zoom, so fitting only scrolls.
  const origin = Math.min(0, allSpan?.[0] ?? 0) - 2;
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
      pendingScroll.current = { left: (f.viewStart - origin) * f.pxPerDay };
      return true;
    },
    [chartViewW, origin],
  );

  const fitScreen = useCallback(() => {
    if (!fitTo(span)) setToast("Nothing to fit yet: no row has a known duration.");
  }, [fitTo, span]);

  const fitSelection = useCallback(() => {
    if (!model) return;
    if (sel.ids.length === 0) return setToast("Select rows to fit them.");
    if (!fitTo(extent(sel.ids, model.times))) setToast("The selected rows have no known dates to fit.");
  }, [fitTo, model, sel]);

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
      const z = zoomAbout(ppd, origin, factor, centre);
      setPpd(clampPx(z.pxPerDay));
      pendingScroll.current = { left: (centre - origin) * z.pxPerDay - chartViewW / 2 };
    },
    [chartViewW, v, ppd, origin],
  );

  const goToday = useCallback(() => {
    if (!model || model.today === null) return setToast("Set a start date to see today on the program.");
    if (model.today < origin || model.today > endIndex) return setToast(`Today (${fmtDate(todayString())}) is outside the program.`);
    pendingScroll.current = { left: xOf(model.today, v) - chartViewW / 3 };
    setWin((w) => ({ ...w })); // render so the scroll applies
  }, [model, origin, endIndex, v, chartViewW]);

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

  // --- print: every row, no virtualisation (plan 9.7) ------------------------

  useEffect(() => {
    if (!active) return;
    const before = () => setPrinting(true);
    const after = () => setPrinting(false);
    window.addEventListener("beforeprint", before);
    window.addEventListener("afterprint", after);
    return () => {
      window.removeEventListener("beforeprint", before);
      window.removeEventListener("afterprint", after);
    };
  }, [active]);

  // --- keyboard (plan 9.5; WP-P7 subset) --------------------------------------

  function onKeyDown(e: KeyboardEvent) {
    const t = e.target as HTMLElement;
    if (t.closest("input, textarea, select")) return;
    const mod = e.ctrlKey || e.metaKey;
    if (mod && e.key === "[") {
      e.preventDefault();
      collapseAll();
    } else if (mod && e.key === "]") {
      e.preventDefault();
      expandAll();
    } else if (mod && (e.key === "a" || e.key === "A")) {
      e.preventDefault();
      setSel(selectAll(order));
    } else if (!mod && !e.altKey && (e.key === "f" || e.key === "F")) {
      e.preventDefault();
      if (e.shiftKey) fitSelection();
      else fitScreen();
    } else if (e.key === "Escape") {
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
          <button type="button" className="btn pg-build" disabled={busy} onClick={() => void apply(() => programApi.prepare(projectId))}>
            {busy ? "Building…" : "Build program"}
          </button>
        </div>
      </section>
    );
  }

  const m = model!;
  const finish = view.computed.finish;
  const typical = shownFinish(m.cal, finish.typical, finish.typical_index);
  const low = shownFinish(m.cal, finish.low, finish.low_index);
  const high = shownFinish(m.cal, finish.high, finish.high_index);
  const checks = view.checks ?? [];
  const q = view.question;
  const askStart = !view.program.start_date;
  const anyOpen = rows.some((r) => !r.stage);
  const scaleProps = { cal: m.cal, v, lo, hi, scale, width: chartW, version: view.program.version };

  const chart = (from: number, to: number, w: number, vp: Viewport, ticksLo: number, ticksHi: number, withOverlay: boolean) => {
    const links = printing ? (showLinks ? drawnLinks(view.links, rows, m.parentOf, from, to) : []) : drawn;
    return (
      <svg className="pg-chart" width={w} height={rows.length * vp.rowH} role="presentation">
        <TimescaleGrid {...scaleProps} v={vp} lo={ticksLo} hi={ticksHi} width={w} plain={m.plain} height={rows.length * vp.rowH} today={m.today} />
        <Bars rows={rows} times={m.times} v={vp} first={from} last={to} selected={selected} conflicts={m.conflicts} relative={m.cal.relative} version={view.program.version} />
        <Links drawn={links} rows={rows} times={m.times} v={vp} driving={m.driving} version={view.program.version} />
        {withOverlay && <Overlay ref={overlay} tooltip={tooltip} height={rows.length * vp.rowH} />}
      </svg>
    );
  };

  return (
    <section className="program" aria-label="Program" onKeyDown={onKeyDown}>
      <header className="program-head">
        <h1>Program</h1>
        {sel.ids.length > 0 && <span className="pg-chip">{sel.ids.length} selected</span>}
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
          <button type="button" className="btn btn-small" disabled title="Editing arrives in the next build">Undo</button>
          <button type="button" className="btn btn-small" disabled title="Editing arrives in the next build">Redo</button>
          <button type="button" className="btn btn-small" disabled title="Editing arrives in the next build">Link in sequence</button>
        </div>
        <div className="pg-tools">
          {m.proposed > 0 && (
            <button type="button" className="btn btn-small pg-accept" disabled={busy} onClick={() => void apply(() => programApi.acceptAll(projectId, view))}>
              Accept all ({m.proposed})
            </button>
          )}
          {view.changes_pending > 0 && (
            <button type="button" className="btn btn-small" aria-expanded={reviewOpen} onClick={() => setReviewOpen((o) => !o)}>
              Review changes ({view.changes_pending})
            </button>
          )}
          <button type="button" className="btn btn-small" disabled title="Importing a program file arrives in a later build">Import</button>
          <a className="btn btn-small" href={programApi.exportHref(projectId, anyOpen ? "expanded" : "collapsed")} download>
            Export
          </a>
          <button type="button" className="btn btn-small" onClick={() => window.print()}>Print</button>
        </div>
      </div>

      {error && <Failure error={error} retry={load} />}
      {reviewOpen && (
        <div className="pg-note" role="note">
          {view.changes_pending} change{view.changes_pending === 1 ? "" : "s"} from the Profile or Procurement wait for review. The review list is not in this build yet; nothing has changed on the program.
        </div>
      )}

      {(q || askStart) && (
        <div className="pg-strip" role="region" aria-label="Question">
          {q && (
            <div className="pg-q">
              <span className="pg-q-label">{q.label}</span>
              {q.number ? (
                <NumberAnswer disabled={busy} onSave={(n) => void apply(() => programApi.answer(projectId, view, q.key, n))} />
              ) : (
                q.options.map((label, i) => (
                  <button
                    key={label}
                    type="button"
                    className="btn btn-small"
                    disabled={busy}
                    onClick={() => void apply(() => programApi.answer(projectId, view, q.key, q.values[i] ?? "not_sure"))}
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
                <button type="button" className="btn btn-small" disabled={busy} onClick={() => void apply(() => programApi.setStart(projectId, view, view.start_suggestion))}>
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
                onClick={() => void apply(() => programApi.setStart(projectId, view, startInput))}
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
          <div className="pg-gridcol" role="treegrid" tabIndex={0} aria-label="Program rows" aria-rowcount={rows.length + 1} aria-multiselectable="true" style={{ height }}>
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
            />
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
          <div className="pg-chartcol">{!printing && chart(first, last, chartW, v, lo, hi, true)}</div>
        </div>
      </div>

      {printing && <PrintPages rows={rows} times={m.times} cal={m.cal} span={span} render={chart} version={view.program.version} />}

      <p className="pg-toast" role="status" aria-live="polite">
        {toast}
      </p>
    </section>
  );
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

// A3 landscape pages (plan 9.7): every row in the current view, the
// timescale header repeated on each page, the chart fitted to the page.
function PrintPages({
  rows,
  times,
  cal,
  span,
  render,
  version,
}: {
  rows: Row[];
  times: Record<string, Times>;
  cal: Cal;
  span: [number, number] | null;
  render: (from: number, to: number, w: number, v: Viewport, lo: number, hi: number, overlay: boolean) => ReactElement;
  version: number;
}) {
  const s = span ?? [0, 20];
  const f = fitSpan(s, PRINT_CHART_W);
  const v: Viewport = { pxPerDay: f.pxPerDay, viewStart: f.viewStart, rowH: ROW_H };
  const pages: [number, number][] = [];
  for (let i = 0; i < Math.max(rows.length, 1); i += PRINT_ROWS) pages.push([i, Math.min(rows.length, i + PRINT_ROWS)]);
  const scale = scaleFor(v.pxPerDay);
  return (
    <div className="pg-print" aria-hidden="true">
      {pages.map(([from, to]) => (
        <div key={from} className="pg-print-page" style={{ ["--pg-grid-w" as string]: `${PRINT_GRID_W}px` }}>
          <div className="pg-print-head">
            <GridHeader narrow={false} />
            <TimescaleHeader cal={cal} v={v} lo={s[0] - 1} hi={s[1] + 2} scale={scale} width={PRINT_CHART_W} version={version} />
          </div>
          <div className="pg-print-body" style={{ height: (to - from) * v.rowH }}>
            <div className="pg-print-grid" style={{ transform: `translateY(${-from * v.rowH}px)` }}>
              <Grid rows={rows} times={times} cal={cal} first={from} last={to} rowH={v.rowH} narrow={false} selected={new Set()} version={version} onToggle={() => {}} onSelect={() => {}} />
            </div>
            <div className="pg-print-chart" style={{ transform: `translateY(${-from * v.rowH}px)` }}>
              {render(from, to, PRINT_CHART_W, v, s[0] - 1, s[1] + 2, false)}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
