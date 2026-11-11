// Run-time motion for any surface (web/DESIGN.md, "Run-time motion and
// marks"; the classes are in trace-motion.css). One motion per change, played
// on the element that shows it, and only for a real event. Elements are found
// by data attribute, so React keeps owning them; a class comes off when its
// motion ends. The value is already on the page: motion never delays it.

export type Motion =
  | "ink-write" // a value written in, left to right (text, including one now to check)
  | "ink-pop" // a small mark or count that changed
  | "ink-on" // a block shaded in (positioned; arrives at --ink-fill)
  | "ink-off" // a block rubbed out
  | "ink-check" // a small element (cell, dot) that now needs a decision
  | "ink-held" // the user's value, read again, standing
  | "ink-glint" // a passage just spoke to this (a table row washes its cells)
  | "ink-flash" // a summary pointing at an item
  | "ink-reading"; // work on this is running now: the ring before its [data-ink-ring] child

// Matches trace-motion.css, so a class comes off once its motion has played.
const DURATION: Record<Motion, number> = {
  "ink-write": 360, "ink-pop": 320, "ink-on": 360, "ink-off": 380, "ink-check": 420,
  "ink-held": 900, "ink-glint": 800, "ink-flash": 1500, "ink-reading": 1400,
};

/** Decorative motion plays only for a visible tab and without reduced motion. */
export function motionAllowed(): boolean {
  if (typeof document !== "undefined" && document.hidden) return false;
  return !(window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false);
}

const running = new WeakMap<Element, Partial<Record<Motion, number>>>();

/** Plays a motion now or after delay ms; replaying restarts it. */
export function play(el: Element | null | undefined, motion: Motion, delay = 0) {
  if (!(el instanceof HTMLElement)) return;
  const timers = running.get(el) ?? {};
  window.clearTimeout(timers[motion]);
  el.classList.remove(motion);
  void el.offsetWidth; // restart the animation
  el.style.setProperty("--ink-delay", `${Math.round(delay)}ms`);
  el.classList.add(motion);
  timers[motion] = window.setTimeout(() => el.classList.remove(motion), delay + DURATION[motion] + 60);
  running.set(el, timers);
}

/** The part of the page the workbench view scrolls in. */
export function viewBox(from: Element): DOMRect {
  const scroller = from.closest(".cockpit-scroll");
  const view = new DOMRect(0, 0, window.innerWidth, window.innerHeight);
  if (!scroller) return view;
  const r = scroller.getBoundingClientRect();
  const top = Math.max(r.top, 0), bottom = Math.min(r.bottom, view.bottom);
  return new DOMRect(r.left, top, r.width, Math.max(0, bottom - top));
}

export function inView(el: Element, box: DOMRect): boolean {
  const r = el.getBoundingClientRect();
  return r.width > 0 && r.bottom > box.top + 2 && r.top < box.bottom - 2;
}

export interface InkItem { el: Element; motion: Motion }

/**
 * Plays items in page order, staggered within 900 ms (at once over 40), so a
 * change reads top to bottom (on Profile: the summary, the card by column,
 * the scope in building order). Items out of view keep only their mark.
 */
export function inkInOrder(items: InkItem[], box: DOMRect) {
  if (!motionAllowed()) return;
  const shown = items
    .filter((i) => inView(i.el, box))
    .sort((a, b) => (a.el.compareDocumentPosition(b.el) & Node.DOCUMENT_POSITION_FOLLOWING ? -1 : 1));
  const step = shown.length > 40 ? 0 : Math.min(45, 900 / Math.max(1, shown.length));
  shown.forEach((item, n) => play(item.el, item.motion, n * step));
}
