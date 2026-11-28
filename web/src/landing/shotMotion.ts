// Plays a screen "being built" over its own picture: covers hide the values,
// cells and bars, and come off at the times the stage gives them. The boxes
// are measured from the live app (scripts/make-landing-shots.mjs, or by hand
// for the tender pictures in tenderShots.ts), and the motions are the
// workbench's own: a cyan glint where a passage is read, a left-to-right wipe
// where a value is written, a fade where a mark lands. The picture is
// complete without this; the stage lays no cover when motion is reduced.

export type How = "write" | "pop" | "glint";
/** Position and size as fractions of the picture, the colour that hides it, when it comes off (ms), how, and an optional mark drawn on the cover. */
export type Box = [x: number, y: number, w: number, h: number, colour: string, at: number, how: How, mark?: "upload"];
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
const GLINT = "rgba(11, 87, 208, .16)"; // --ink-spotted at the workbench's 16%
// The procurement table's empty-cell mark, as the app draws it.
const UPLOAD = `url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%23a29a91' stroke-width='1.6' stroke-linecap='round' stroke-linejoin='round'><path d='M12 15V4M7.5 8.5 12 4l4.5 4.5M4 14v5h16v-5'/></svg>")`;

export interface Covers {
  /** Starts every cover's motion from now, at the shot's times or the ones given. */
  play: (times?: number[]) => Animation[];
  /** Puts every cover back, so the screen is empty again. */
  reset: () => void;
  /** When the last cover is off, in ms from the start. */
  end: (times?: number[]) => number;
  destroy: () => void;
}

export function layCovers(frame: HTMLElement, shot: Shot): Covers {
  const layer = document.createElement("div");
  layer.className = "shot-layer";
  layer.setAttribute("aria-hidden", "true");
  const covers = shot.boxes.map(([x, y, w, h, colour, , how, mark]) => {
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
    if (mark === "upload") {
      el.style.backgroundImage = UPLOAD;
      el.style.backgroundRepeat = "no-repeat";
      el.style.backgroundPosition = "4% 50%";
      el.style.backgroundSize = "auto 46%";
    }
    layer.append(el);
    return el;
  });
  frame.append(layer);

  let running: Animation[] = [];
  const timeOf = (i: number, times?: number[]) => times?.[i] ?? shot.boxes[i][5];
  const duration = (i: number) => {
    const [, , w, , , , how] = shot.boxes[i];
    // A wide cover takes a little longer, so a long bar is drawn, not flashed.
    return how === "glint" ? 800 : how === "pop" ? 320 : 340 + w * 420;
  };
  const reset = () => { running.forEach(a => a.cancel()); running = []; };
  return {
    play: times => {
      reset();
      running = covers.map((el, i) => {
        const how = shot.boxes[i][6];
        const timing: KeyframeAnimationOptions = { delay: Math.max(0, timeOf(i, times)), duration: duration(i), fill: "both" };
        if (how === "glint") return el.animate([{ opacity: 0 }, { opacity: 1, offset: 0.12 }, { opacity: 0 }], { ...timing, easing: "ease-out" });
        if (how === "pop") return el.animate([{ opacity: 1 }, { opacity: 0 }], { ...timing, easing: EASE });
        return el.animate([{ clipPath: "inset(0 0 0 0)" }, { clipPath: "inset(0 0 0 100%)" }], { ...timing, easing: EASE });
      });
      return running;
    },
    reset,
    end: times => covers.reduce((n, _, i) => Math.max(n, timeOf(i, times) + duration(i)), 0),
    destroy: () => { reset(); layer.remove(); },
  };
}
