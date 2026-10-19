// Link line routing (plan 9.3). A path leaves the predecessor's anchor
// (finish x for F, start x for S) by an 8 px stub, then either goes straight
// down to the successor row and in, or, when the successor anchor is behind
// the stub, routes round through the gap between rows. The arrowhead points
// into the successor anchor: rightwards into a start, leftwards into a finish.

export const STUB = 8;
export const ARROW = 5;

export interface Anchors {
  /** Predecessor anchor x and row centre y. */
  px: number;
  py: number;
  /** Successor anchor x and row centre y. */
  sx: number;
  sy: number;
  type: string;
  rowH: number;
}

export interface Route {
  d: string;
  arrow: string;
  /** Midpoint of the middle segment, for the hold-point diamond and lag label. */
  mid: { x: number; y: number };
  /** True when the line had to go round. */
  back: boolean;
}

/** Which end of each bar a link type joins: [predecessor, successor]. */
export function anchorEnds(type: string): ["S" | "F", "S" | "F"] {
  switch (type) {
    case "SS":
      return ["S", "S"];
    case "FF":
      return ["F", "F"];
    case "SF":
      return ["S", "F"];
    default:
      return ["F", "S"];
  }
}

const r1 = (n: number) => Math.round(n * 10) / 10;

export function route(a: Anchors): Route {
  const [predEnd, succEnd] = anchorEnds(a.type);
  const exitX = a.px + (predEnd === "F" ? STUB : -STUB);
  // A start is entered from the left, a finish from the right.
  const intoStart = succEnd === "S";
  const forward = intoStart ? exitX <= a.sx : exitX >= a.sx;
  let d: string;
  let mid: { x: number; y: number };
  if (forward) {
    d = `M${r1(a.px)},${r1(a.py)}H${r1(exitX)}V${r1(a.sy)}H${r1(a.sx)}`;
    mid = { x: exitX, y: (a.py + a.sy) / 2 };
  } else {
    const down = a.sy >= a.py ? 1 : -1;
    const gapY = a.sy - (down * a.rowH) / 2;
    const entryX = a.sx + (intoStart ? -STUB : STUB);
    d = `M${r1(a.px)},${r1(a.py)}H${r1(exitX)}V${r1(gapY)}H${r1(entryX)}V${r1(a.sy)}H${r1(a.sx)}`;
    mid = { x: (exitX + entryX) / 2, y: gapY };
  }
  const tip = a.sx;
  const backX = intoStart ? tip - ARROW : tip + ARROW;
  const arrow = `M${r1(tip)},${r1(a.sy)}L${r1(backX)},${r1(a.sy - ARROW / 1.4)}L${r1(backX)},${r1(a.sy + ARROW / 1.4)}Z`;
  return { d, arrow, mid, back: !forward };
}
