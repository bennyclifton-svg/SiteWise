// Chart pointer handling (plan 9.4 "Pointer handling"): Pointer Events only,
// captured on down; a press becomes a drag after 4 px (or, on touch, after a
// 350 ms long press, so one-finger scrolling still scrolls). Moves are
// batched into one requestAnimationFrame: the frame snaps the pointer, runs
// the TypeScript engine on the edited input and writes transforms straight
// onto the bar nodes and the overlay. No React state changes during a drag.
// On release the hook builds the op batch and hands it to the view, which
// updates optimistically from the preview, posts it and replaces state with
// the server's answer (or restores it with the server's message). Escape or
// pointercancel restores and sends nothing.

import { useCallback, useEffect, useRef, type PointerEvent as ReactPointerEvent } from "react";
import type { Cal } from "../calendar";
import { compute } from "../engine";
import {
  boxHits,
  completionBadge,
  DRAG_THRESHOLD,
  heldMessage,
  heldRows,
  kidsOf,
  linkType,
  LONG_PRESS_MS,
  moveTargets,
  snapDays,
  snapUnit,
  withMoves,
  withResize,
  ZONE_COARSE,
  ZONE_FINE,
  zoneAt,
  type Zone,
} from "../edit";
import { barOf, fmtDate, indexAt, xOf, type Row, type Viewport } from "../layout";
import type { Input, Output, Times } from "../types";
import { toInput, type ProgramActivity, type ProgramOp, type ProgramView } from "../../programApi";
import { showTip, type OverlayHandle } from "./Overlay";

export interface DragCtx {
  view: ProgramView;
  rows: readonly Row[];
  times: Record<string, Times>;
  v: Viewport;
  cal: Cal;
  selIds: readonly string[];
  overlay: () => OverlayHandle | null;
  scroller: HTMLDivElement | null;
  select: (id: string, mods: { ctrl: boolean; shift: boolean }) => void;
  addToSelection: (ids: string[]) => void;
  clearSelection: () => void;
  /** Sends a batch; `optimistic` is shown at once (plan 9.4 step 4). */
  commit: (ops: ProgramOp[], o?: { optimistic?: ProgramView; added?: { temp: string; parent: string | null } }) => void;
  openLink: (id: string, x: number, y: number) => void;
  setDragging: (on: boolean) => void;
}

type Hit =
  | { kind: "dot"; id: string; end: "S" | "F" }
  | { kind: "bar"; id: string; zone: Zone }
  | { kind: "link"; id: string }
  | { kind: "empty"; row: number };

interface Move {
  kind: "move";
  primary: string;
  send: string[];
  targets: string[];
  input: Input;
  delta: number;
  out: Output | null;
}
interface Resize {
  kind: "resize";
  id: string;
  end: "start" | "end";
  es: number;
  d: number;
  input: Input;
  delta: number;
  out: Output | null;
}
interface LinkDrag {
  kind: "link";
  id: string;
  end: "S" | "F";
  ax: number;
  ay: number;
  over: Element | null;
}
interface Draw {
  kind: "draw";
  row: number;
}
interface Box {
  kind: "box";
}
type Gesture = Move | Resize | LinkDrag | Draw | Box;

interface Press {
  pointer: number;
  touch: boolean;
  x0: number;
  y0: number;
  cx0: number;
  cy0: number;
  x: number;
  y: number;
  cx: number;
  cy: number;
  alt: boolean;
  mods: { ctrl: boolean; shift: boolean };
  hit: Hit;
  g: Gesture | null;
  armed: boolean;
  timer: number;
  /** Bars in the DOM at drag start, by id. */
  els: Map<string, SVGGElement>;
}

const reducedMotion = () => {
  try {
    return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  } catch {
    return false;
  }
};

/** Frame times of the last drag, for the browser budget (plan 11.2). */
function frameLog(): number[] {
  const w = window as unknown as { __pgDragFrames?: number[] };
  return (w.__pgDragFrames ??= []);
}

export function useDrag(ctx: DragCtx) {
  const live = useRef(ctx);
  live.current = ctx;
  const svg = useRef<SVGSVGElement | null>(null);
  const press = useRef<Press | null>(null);
  const raf = useRef(0);
  const tipTimer = useRef(0);
  const hovered = useRef<Element | null>(null);

  const local = (cx: number, cy: number): [number, number] => {
    const r = svg.current!.getBoundingClientRect();
    return [cx - r.left, cy - r.top];
  };

  const coarse = (touch: boolean) => {
    if (touch) return true;
    try {
      return window.matchMedia("(pointer: coarse)").matches;
    } catch {
      return false;
    }
  };

  // What a press is on. Links are drawn over bars, so look through the whole
  // stack under the pointer: a link dot first, then a bar's end zone, then
  // a link line, then a bar's body. A link stub leaving a finish would
  // otherwise cover that bar's finish dot and end zone.
  const hitAt = (cx: number, cy: number, x: number, y: number, touch: boolean): Hit => {
    const c = live.current;
    const stack = document.elementsFromPoint(cx, cy);
    let bar: Hit | null = null;
    let link: Hit | null = null;
    for (const el of stack) {
      const g = el.closest(".pg-bar") as SVGGElement | null;
      if (el.closest("[data-dot]") && g) return { kind: "dot", id: g.dataset.id!, end: el.closest("[data-dot]")!.getAttribute("data-dot") as "S" | "F" };
      if (g && !bar) {
        const id = g.dataset.id!;
        const b = barOf(c.times[id], c.v);
        const row = c.rows.find((r) => r.id === id);
        const [hx, hw] = b ? (b.milestone ? [b.x - b.w / 2, b.w] : [b.x, b.w]) : [0, 0];
        bar = { kind: "bar", id, zone: b && row ? zoneAt(x, hx, hw, coarse(touch) ? ZONE_COARSE : ZONE_FINE, { milestone: b.milestone, summary: row.hasChildren }) : "body" };
      }
      const l = el.closest("[data-link]") as SVGGElement | null;
      if (l && !link) link = { kind: "link", id: l.dataset.id! };
      if (!el.closest(".pg-chart")) break;
    }
    if (bar && bar.kind === "bar" && bar.zone !== "body") return bar;
    return link ?? bar ?? { kind: "empty", row: Math.floor(y / c.v.rowH) };
  };

  const dateLabel = (i: number) => {
    const c = live.current;
    return c.cal.relative ? c.cal.dayLabel(Math.max(0, i)) : fmtDate(c.cal.dateOf(Math.max(0, i)));
  };

  const hideOverlay = () => {
    const o = live.current.overlay();
    if (!o) return;
    for (const el of [o.ghost, o.band, o.guide, o.box]) el?.setAttribute("visibility", "hidden");
    showTip(o.tip, 0, 0, "");
  };

  /** Puts every moved bar back, easing over 150 ms unless motion is reduced. */
  const restoreBars = (p: Press, ease: boolean) => {
    const anim = ease && !reducedMotion();
    for (const el of p.els.values()) {
      el.classList.remove("is-ghosted");
      if (!el.style.transform) continue;
      if (anim) {
        el.style.transition = "transform 150ms ease-out";
        window.setTimeout(() => (el.style.transition = ""), 200);
      }
      el.style.transform = "";
      delete el.dataset.dx;
    }
    svg.current?.classList.remove("is-moving", "is-linking", "is-resizing");
  };

  const flashTip = (x: number, y: number, text: string) => {
    window.clearTimeout(tipTimer.current);
    showTip(live.current.overlay()?.tip ?? null, x, y, text);
    tipTimer.current = window.setTimeout(() => showTip(live.current.overlay()?.tip ?? null, 0, 0, ""), 2500);
  };

  const end = useCallback((p: Press) => {
    window.clearTimeout(p.timer);
    cancelAnimationFrame(raf.current);
    raf.current = 0;
    press.current = null;
    try {
      svg.current?.releasePointerCapture(p.pointer);
    } catch {
      // Already released.
    }
    if (p.g) live.current.setDragging(false);
  }, []);

  /** Escape or pointercancel: restore, send nothing. */
  const cancel = useCallback(() => {
    const p = press.current;
    if (!p) return;
    restoreBars(p, true);
    hideOverlay();
    end(p);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [end]);

  // Escape during a drag cancels it before the view's own Escape (clear selection).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && press.current?.g) {
        e.preventDefault();
        e.stopPropagation();
        cancel();
      }
    };
    window.addEventListener("keydown", onKey, true);
    return () => window.removeEventListener("keydown", onKey, true);
  }, [cancel]);

  // After a long press arms a touch drag, stop the browser panning (plan 9.4).
  useEffect(() => {
    const el = ctx.scroller;
    if (!el) return;
    const onMove = (e: TouchEvent) => {
      if (press.current?.armed && e.cancelable) e.preventDefault();
    };
    el.addEventListener("touchmove", onMove, { passive: false });
    return () => el.removeEventListener("touchmove", onMove);
  }, [ctx.scroller]);

  // --- starting a gesture ----------------------------------------------------

  const begin = (p: Press) => {
    const c = live.current;
    const h = p.hit;
    let g: Gesture | null = null;
    if (h.kind === "dot") {
      const b = barOf(c.times[h.id], c.v)!;
      const i = c.rows.findIndex((r) => r.id === h.id);
      const [hx, hw] = b.milestone ? [b.x - b.w / 2, b.w] : [b.x, b.w];
      g = { kind: "link", id: h.id, end: h.end, ax: h.end === "S" ? hx : hx + hw, ay: i * c.v.rowH + c.v.rowH / 2, over: null };
      svg.current?.classList.add("is-linking");
    } else if (h.kind === "bar") {
      const t = c.times[h.id];
      if (!t || t.es === null || t.ef === null) return;
      if (h.zone !== "body") {
        g = { kind: "resize", id: h.id, end: h.zone, es: t.es, d: t.ef - t.es, input: toInput(c.view), delta: 0, out: null };
        p.els.get(h.id)?.classList.add("is-ghosted");
        svg.current?.classList.add("is-resizing");
      } else {
        const inSel = c.selIds.includes(h.id);
        if (!inSel) c.select(h.id, { ctrl: false, shift: false });
        const send = inSel ? [...c.selIds] : [h.id];
        const targets = moveTargets(send, kidsOf(c.view.activities)).filter((id) => c.times[id]?.es != null);
        g = { kind: "move", primary: h.id, send, targets, input: toInput(c.view), delta: 0, out: null };
        svg.current?.classList.add("is-moving");
      }
    } else if (h.kind === "empty") {
      g = p.mods.shift ? { kind: "box" } : { kind: "draw", row: h.row };
    }
    if (!g) return;
    p.g = g;
    if (g.kind === "move" || g.kind === "resize") frameLog().length = 0;
    c.setDragging(true);
  };

  // --- one animation frame -----------------------------------------------------

  const frame = () => {
    raf.current = 0;
    const p = press.current;
    if (!p?.g) return;
    const c = live.current;
    const o = c.overlay();
    const g = p.g;
    const ppd = c.v.pxPerDay;
    const unit = snapUnit(ppd, p.alt, c.cal.perWeek);
    const t0 = performance.now();
    if (g.kind === "move") {
      const delta = snapDays(p.x - p.x0, ppd, unit);
      if (delta !== g.delta || !g.out) {
        g.delta = delta;
        g.out = compute(withMoves(g.input, c.times, g.targets, delta, c.cal));
        writeShifts(p, g.out, null);
        frameLog().push(performance.now() - t0);
      }
      const t = c.times[g.primary];
      const to = (t.es as number) + delta;
      const held = heldRows(g.targets, c.times, g.out, delta);
      guide(o, xOf(Math.max(0, to), c.v));
      const text = held.length
        ? heldMessage(held, c.times, g.out, delta, c.view.activities, c.view.links)
        : `${dateLabel(to)} · ${completionBadge(c.view.computed.finish.typical_index, g.out.finish.typical_index)}`;
      showTip(o?.tip ?? null, p.x + 12, p.y - 10, text);
    } else if (g.kind === "resize") {
      const raw = snapDays(p.x - p.x0, ppd, unit);
      const delta = g.end === "end" ? Math.max(-g.d, raw) : Math.min(g.d, raw);
      if (delta !== g.delta || !g.out) {
        g.delta = delta;
        const [start, d] = g.end === "end" ? [null, g.d + delta] : [g.es + delta, g.d - delta];
        g.out = compute(withResize(g.input, g.id, d, start, c.cal));
        writeShifts(p, g.out, g.id);
        frameLog().push(performance.now() - t0);
      }
      const nt = g.out.activities[g.id];
      const s = nt?.es ?? g.es;
      const d = g.end === "end" ? g.d + delta : g.d - delta;
      const i = c.rows.findIndex((r) => r.id === g.id);
      if (o?.ghost) {
        o.ghost.setAttribute("x", String(xOf(s, c.v)));
        o.ghost.setAttribute("y", String(i * c.v.rowH + c.v.rowH * 0.22));
        o.ghost.setAttribute("width", String(Math.max(2, d * ppd)));
        o.ghost.setAttribute("height", String(c.v.rowH * 0.56));
        o.ghost.setAttribute("visibility", "visible");
      }
      guide(o, xOf(g.end === "end" ? s + d : s, c.v));
      const held = g.end === "start" && heldRows([g.id], c.times, g.out, delta).length > 0;
      const text = held
        ? heldMessage([g.id], c.times, g.out, delta, c.view.activities, c.view.links)
        : `${d === 0 ? "Milestone" : `${d} days`} · ${completionBadge(c.view.computed.finish.typical_index, g.out.finish.typical_index)}`;
      showTip(o?.tip ?? null, p.x + 12, p.y - 10, text);
    } else if (g.kind === "link") {
      if (o?.band) {
        o.band.setAttribute("x1", String(g.ax));
        o.band.setAttribute("y1", String(g.ay));
        o.band.setAttribute("x2", String(p.x));
        o.band.setAttribute("y2", String(p.y));
        o.band.setAttribute("visibility", "visible");
      }
      // A dot of another bar wins over that bar's body; links drawn on top are looked through.
      let target: Element | null = null;
      for (const el of document.elementsFromPoint(p.cx, p.cy)) {
        const b = el.closest(".pg-bar") as SVGGElement | null;
        if (!b || b.dataset.id === g.id) continue;
        const d = el.closest("[data-dot]");
        if (d) {
          target = d;
          break;
        }
        target ??= b;
      }
      if (target !== g.over) {
        g.over?.classList.remove("is-target");
        target?.classList.add("is-target");
        g.over = target;
      }
    } else if (g.kind === "draw") {
      const [a, b] = drawSpan(p, c.v, unit);
      if (o?.ghost) {
        o.ghost.setAttribute("x", String(xOf(a, c.v)));
        o.ghost.setAttribute("y", String(g.row * c.v.rowH + c.v.rowH * 0.22));
        o.ghost.setAttribute("width", String(Math.max(2, (b - a) * ppd)));
        o.ghost.setAttribute("height", String(c.v.rowH * 0.56));
        o.ghost.setAttribute("visibility", "visible");
      }
      showTip(o?.tip ?? null, p.x + 12, p.y - 10, `${dateLabel(a)} · ${b - a} days`);
    } else if (o?.box) {
      o.box.setAttribute("x", String(Math.min(p.x0, p.x)));
      o.box.setAttribute("y", String(Math.min(p.y0, p.y)));
      o.box.setAttribute("width", String(Math.abs(p.x - p.x0)));
      o.box.setAttribute("height", String(Math.abs(p.y - p.y0)));
      o.box.setAttribute("visibility", "visible");
    }
  };

  // Moves each bar in the DOM by its preview shift; `skip` is the bar shown
  // by the ghost instead.
  const writeShifts = (p: Press, out: Output, skip: string | null) => {
    const c = live.current;
    for (const [id, el] of p.els) {
      if (id === skip) continue;
      const a = c.times[id]?.es;
      const b = out.activities[id]?.es;
      const dx = a != null && b != null ? (b - a) * c.v.pxPerDay : 0;
      const s = dx ? `translateX(${dx}px)` : "";
      if (el.style.transform !== s) {
        el.style.transform = s;
        el.dataset.dx = String(dx);
      }
    }
  };

  const schedule = () => {
    if (!raf.current) raf.current = requestAnimationFrame(frame);
  };

  // --- release -------------------------------------------------------------------

  const release = (p: Press) => {
    const c = live.current;
    const g = p.g;
    if (!g) {
      click(p);
      return;
    }
    // Run the last pending frame so the release acts on what was shown.
    if (raf.current) {
      cancelAnimationFrame(raf.current);
      frame();
    }
    hideOverlay();
    if (g.kind === "move") {
      if (g.delta === 0 || !g.out) return restoreBars(p, false);
      const held = heldRows(g.targets, c.times, g.out, g.delta);
      if (held.length) {
        // Plan 13.1: snap back, say what holds it, write nothing.
        restoreBars(p, true);
        flashTip(p.x + 12, p.y - 10, heldMessage(held, c.times, g.out, g.delta, c.view.activities, c.view.links));
        return;
      }
      const moved = new Map(g.targets.map((id) => [id, (c.times[id].es as number) + g.delta]));
      const optimistic = withActs(c.view, g.out, (a) =>
        moved.has(a.id) ? { ...a, constraint_kind: a.constraint_kind === "mso" ? "mso" : "snet", constraint_date: c.cal.dateOf(Math.max(0, moved.get(a.id)!)) } : a,
      );
      c.commit([{ op: "move", ids: g.send, days: g.delta }], { optimistic });
      return restoreBars(p, false);
    }
    if (g.kind === "resize") {
      if (g.delta === 0 || !g.out) return restoreBars(p, false);
      const d = g.end === "end" ? g.d + g.delta : g.d - g.delta;
      if (g.end === "start" && heldRows([g.id], c.times, g.out, g.delta).length) {
        restoreBars(p, true);
        flashTip(p.x + 12, p.y - 10, heldMessage([g.id], c.times, g.out, g.delta, c.view.activities, c.view.links));
        return;
      }
      const ops: ProgramOp[] = [];
      if (g.end === "start") ops.push({ op: "move", ids: [g.id], days: g.delta });
      ops.push({ op: "resize", id: g.id, duration_days: d });
      const start = g.end === "start" ? g.es + g.delta : null;
      const optimistic = withActs(c.view, g.out, (a) => {
        if (a.id !== g.id) return a;
        const b = { ...a, duration_days: d, duration_low: null, duration_high: null, duration_origin: "user" };
        if (b.kind === "milestone" && d > 0) b.kind = "activity";
        return start === null ? b : { ...b, constraint_kind: a.constraint_kind === "mso" ? "mso" : "snet", constraint_date: c.cal.dateOf(Math.max(0, start)) };
      });
      c.commit(ops, { optimistic });
      return restoreBars(p, false);
    }
    if (g.kind === "link") {
      g.over?.classList.remove("is-target");
      restoreBars(p, false);
      const target = g.over?.closest(".pg-bar") as SVGGElement | null;
      if (!g.over || !target) return;
      const to = g.over.getAttribute("data-dot") as "S" | "F" | null;
      // A drop on a bar's middle is finish-to-start with no lag (plan 9.4).
      const type = to ? linkType(g.end, to) : "FS";
      c.commit([{ op: "link_add", pred_id: g.id, succ_id: target.dataset.id!, type, lag_days: 0, lag_unit: "working" }]);
      return;
    }
    if (g.kind === "draw") {
      const [a, b] = drawSpan(p, c.v, snapUnit(c.v.pxPerDay, p.alt, c.cal.perWeek));
      const row = c.rows[g.row];
      const temp = "new-1";
      const ops: ProgramOp[] = [];
      let parent: string | null = null;
      if (!row) ops.push({ op: "add", temp_id: temp, label: "New stage", duration_days: b - a });
      else {
        parent = row.stage ? row.id : (row.parentId as string);
        // A stage's first activity takes the stage's own days (plan 6.5), so
        // the drawn length only applies once the stage has activities.
        const first = row.stage && !row.hasChildren && row.act.duration_days !== null;
        ops.push({ op: "add", temp_id: temp, parent_id: parent, after_id: row.stage ? undefined : row.id, label: "New activity", ...(first ? {} : { duration_days: b - a }) });
      }
      if (a > 0) ops.push({ op: "update", id: temp, constraint_kind: "snet", constraint_date: c.cal.dateOf(a) });
      c.commit(ops, { added: { temp, parent } });
      return;
    }
    // Selection box: add every bar it touches.
    c.addToSelection(boxHits(c.rows, c.times, c.v, p.x0, p.y0, p.x, p.y));
  };

  // A press that never became a drag.
  const click = (p: Press) => {
    const c = live.current;
    const h = p.hit;
    if (h.kind === "bar" || h.kind === "dot") {
      c.select(h.id, p.mods);
      (svg.current?.querySelector(`.pg-layer-bars [data-id="${h.id}"]`) as SVGGElement | null)?.focus({ preventScroll: true });
    } else if (h.kind === "link") c.openLink(h.id, p.x, p.y);
    else c.clearSelection();
  };

  // --- React handlers --------------------------------------------------------------

  const onPointerDown = useCallback((e: ReactPointerEvent<SVGSVGElement>) => {
    if (press.current || (e.pointerType === "mouse" && e.button !== 0)) return;
    const touch = e.pointerType === "touch";
    const [x, y] = local(e.clientX, e.clientY);
    const p: Press = {
      pointer: e.pointerId,
      touch,
      x0: x,
      y0: y,
      cx0: e.clientX,
      cy0: e.clientY,
      x,
      y,
      cx: e.clientX,
      cy: e.clientY,
      alt: e.altKey,
      mods: { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey },
      hit: hitAt(e.clientX, e.clientY, x, y, touch),
      g: null,
      armed: false,
      timer: 0,
      els: new Map(),
    };
    for (const el of svg.current!.querySelectorAll<SVGGElement>(".pg-layer-bars .pg-bar")) p.els.set(el.dataset.id!, el);
    press.current = p;
    if (!touch) {
      e.preventDefault(); // no text selection while dragging
      try {
        e.currentTarget.setPointerCapture(e.pointerId);
      } catch {
        // The pointer may already be gone.
      }
      return;
    }
    // Touch: a long press arms the drag; moving first lets the page scroll.
    p.timer = window.setTimeout(() => {
      if (press.current !== p) return;
      p.armed = true;
      try {
        svg.current?.setPointerCapture(p.pointer);
      } catch {
        // Lost already; pointercancel will tidy up.
      }
      begin(p);
      svg.current?.classList.add("is-armed");
      schedule();
    }, LONG_PRESS_MS);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const onPointerMove = useCallback((e: ReactPointerEvent<SVGSVGElement>) => {
    const p = press.current;
    if (!p) {
      if (e.pointerType === "mouse") hover(e);
      return;
    }
    if (e.pointerId !== p.pointer) return;
    const [x, y] = local(e.clientX, e.clientY);
    Object.assign(p, { x, y, cx: e.clientX, cy: e.clientY, alt: e.altKey });
    if (!p.g) {
      const dist = Math.hypot(e.clientX - p.cx0, e.clientY - p.cy0);
      if (p.touch) {
        if (!p.armed && dist > 8) {
          window.clearTimeout(p.timer);
          press.current = null; // the browser scrolls
        }
        return;
      }
      if (dist < DRAG_THRESHOLD) return;
      begin(p);
      if (!p.g) return;
    }
    schedule();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const onPointerUp = useCallback((e: ReactPointerEvent<SVGSVGElement>) => {
    const p = press.current;
    if (!p || e.pointerId !== p.pointer) return;
    svg.current?.classList.remove("is-armed");
    end(p);
    release(p);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [end]);

  const onPointerCancel = useCallback(() => {
    svg.current?.classList.remove("is-armed");
    cancel();
  }, [cancel]);

  // Cursor by what the pointer is over (plan 9.4).
  const hover = (e: ReactPointerEvent<SVGSVGElement>) => {
    const el = svg.current;
    if (!el) return;
    const [x, y] = local(e.clientX, e.clientY);
    const h = hitAt(e.clientX, e.clientY, x, y, false);
    // Hover by hit test, not CSS :hover: a link drawn over a bar would take
    // the hover and hide the bar's link dots.
    const g = h.kind === "bar" || h.kind === "dot" ? el.querySelector(`.pg-layer-bars [data-id="${h.id}"]`) : null;
    if (g !== hovered.current || (g && !g.classList.contains("is-hover"))) {
      hovered.current?.classList.remove("is-hover");
      g?.classList.add("is-hover");
      hovered.current = g;
    }
    const cursor = h.kind === "dot" ? "crosshair" : h.kind === "bar" ? (h.zone === "body" ? "grab" : "ew-resize") : h.kind === "link" ? "pointer" : "";
    if (el.style.cursor !== cursor) el.style.cursor = cursor;
  };

  return { svg, onPointerDown, onPointerMove, onPointerUp, onPointerCancel };
}

function guide(o: OverlayHandle | null, x: number) {
  if (!o?.guide) return;
  o.guide.setAttribute("x1", String(x));
  o.guide.setAttribute("x2", String(x));
  o.guide.setAttribute("visibility", "visible");
}

// The snapped [start, finish) indices of a bar being drawn: at least one unit.
function drawSpan(p: Press, v: Viewport, unit: number): [number, number] {
  const snap = (x: number) => Math.round(indexAt(x, v) / unit) * unit;
  let a = Math.max(0, snap(Math.min(p.x0, p.x)));
  let b = snap(Math.max(p.x0, p.x));
  if (b <= a) b = a + unit;
  if (a < 0) a = 0;
  return [a, b];
}

// The optimistic view: rows edited, times from the preview (plan 9.4 step 4).
function withActs(view: ProgramView, out: Output, edit: (a: ProgramActivity) => ProgramActivity): ProgramView {
  return { ...view, activities: view.activities.map(edit), computed: { ...view.computed, ...out, status: view.computed.status } };
}
