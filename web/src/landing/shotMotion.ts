// Plays a screen "being built" over its own picture: covers hide the values,
// cells and bars, and come off in order when the picture scrolls into view.
// The boxes are measured from the live app (scripts/make-landing-shots.mjs),
// and the motions are the workbench's own: a cyan glint where a passage is
// read, a left-to-right wipe where a value is written, a fade where a mark
// lands. The picture is complete without this; no cover is laid when motion
// is reduced or the browser cannot play it.

export type Box = [x: number, y: number, w: number, h: number, colour: string, at: number, how: "write" | "pop" | "glint"];
export type Shot = {
  width: number;
  height: number;
  /** The CSS width the screen was captured at; one of its pixels is 1/cssWidth of the picture. */
  cssWidth: number;
  /** How an unshaded scope cell is drawn, so a "pop" cover looks like one. */
  emptyCell?: { background: string; border: string; radius: string };
  boxes: Box[];
};

const EASE = "cubic-bezier(.16, 1, .3, 1)";
const GLINT = "rgba(18, 96, 109, .16)"; // --ink-spotted at the workbench's 16%

/** Lays the covers over `frame` and plays them once it is in view. Returns replay and a cleanup. */
export function mountShot(frame: HTMLElement, shot: Shot): { replay: () => void; destroy: () => void } | null {
  const still = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
  if (still || !("IntersectionObserver" in window) || typeof frame.animate !== "function") return null;

  const layer = document.createElement("div");
  layer.className = "shot-layer";
  layer.setAttribute("aria-hidden", "true");
  const covers = shot.boxes.map(([x, y, w, h, colour, , how]) => {
    const el = document.createElement("i");
    el.style.cssText = `left:${x * 100}%;top:${y * 100}%;width:${w * 100}%;height:${h * 100}%`;
    if (how === "glint") {
      el.style.background = GLINT;
      el.style.opacity = "0";
    } else if (how === "pop" && shot.emptyCell) {
      el.style.background = shot.emptyCell.background;
      // The cell's 1px outline, at the scale the picture is shown.
      el.style.boxShadow = `inset 0 0 0 calc(100cqw / ${shot.cssWidth}) ${shot.emptyCell.border}`;
      el.style.borderRadius = shot.emptyCell.radius;
    } else {
      el.style.background = colour;
    }
    layer.append(el);
    return el;
  });
  frame.append(layer);

  let running: Animation[] = [];
  const play = () => {
    running.forEach(a => a.cancel());
    running = covers.map((el, i) => {
      const [, , w, , , at, how] = shot.boxes[i];
      if (how === "glint") return el.animate([{ opacity: 0 }, { opacity: 1, offset: 0.12 }, { opacity: 0 }], { delay: at, duration: 800, easing: "ease-out", fill: "both" });
      if (how === "pop") return el.animate([{ opacity: 1 }, { opacity: 0 }], { delay: at, duration: 320, easing: EASE, fill: "both" });
      // A wide cover takes a little longer, so a long bar is drawn, not flashed.
      return el.animate([{ clipPath: "inset(0 0 0 0)" }, { clipPath: "inset(0 0 0 100%)" }], { delay: at, duration: 340 + w * 420, easing: EASE, fill: "both" });
    });
  };

  const seen = new IntersectionObserver(entries => {
    if (!entries.some(e => e.isIntersecting)) return;
    seen.disconnect();
    play();
  }, { threshold: 0.35 });
  seen.observe(frame);

  return {
    replay: play,
    destroy: () => {
      seen.disconnect();
      running.forEach(a => a.cancel());
      layer.remove();
    },
  };
}
