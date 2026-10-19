import { expect, test } from "vitest";
import { ARROW, STUB, anchorEnds, route } from "./routing";

const rowH = 28;

// Coordinates of a path's corners, in order.
const points = (d: string) => {
  const out: [number, number][] = [];
  let x = 0;
  let y = 0;
  for (const m of d.matchAll(/([MHVL])([-\d.]+)(?:,([-\d.]+))?/g)) {
    if (m[1] === "M" || m[1] === "L") [x, y] = [+m[2], +m[3]];
    else if (m[1] === "H") x = +m[2];
    else y = +m[2];
    out.push([x, y]);
  }
  return out;
};

test("forward FS leaves the finish by the stub, drops to the successor row and enters its start from the left", () => {
  const r = route({ px: 100, py: 14, sx: 140, sy: 70, type: "FS", rowH });
  expect(r.back).toBe(false);
  expect(points(r.d)).toEqual([
    [100, 14],
    [100 + STUB, 14],
    [100 + STUB, 70],
    [140, 70],
  ]);
  // Arrow tip on the anchor, base to its left: points right, into the start.
  const tip = points(r.arrow);
  expect(tip[0]).toEqual([140, 70]);
  expect(tip[1][0]).toBe(140 - ARROW);
});

test("backward FS routes round through the gap above the successor row", () => {
  // The successor starts before the predecessor's stub: it must go round.
  const r = route({ px: 200, py: 14, sx: 120, sy: 70, type: "FS", rowH });
  expect(r.back).toBe(true);
  const p = points(r.d);
  expect(p[1]).toEqual([200 + STUB, 14]);
  expect(p[2]).toEqual([200 + STUB, 70 - rowH / 2]); // into the gap between rows
  expect(p[3]).toEqual([120 - STUB, 70 - rowH / 2]); // across, past the anchor
  expect(p[4]).toEqual([120 - STUB, 70]);
  expect(p[5]).toEqual([120, 70]);
  expect(r.mid.y).toBe(70 - rowH / 2);
});

test("backward route upwards uses the gap below the successor row", () => {
  const r = route({ px: 200, py: 98, sx: 120, sy: 14, type: "FS", rowH });
  expect(points(r.d)[2]).toEqual([200 + STUB, 14 + rowH / 2]);
});

test("each anchor pair joins the right ends and points the right way", () => {
  expect(anchorEnds("FS")).toEqual(["F", "S"]);
  expect(anchorEnds("SS")).toEqual(["S", "S"]);
  expect(anchorEnds("FF")).toEqual(["F", "F"]);
  expect(anchorEnds("SF")).toEqual(["S", "F"]);

  // SS: leaves the predecessor's start leftwards, enters the successor start from the left.
  const ss = points(route({ px: 100, py: 14, sx: 130, sy: 42, type: "SS", rowH }).d);
  expect(ss[1]).toEqual([100 - STUB, 14]);
  expect(ss.at(-1)).toEqual([130, 42]);

  // FF: leaves the finish rightwards; the successor finish is entered from the right.
  const ff = route({ px: 200, py: 14, sx: 180, sy: 42, type: "FF", rowH });
  expect(ff.back).toBe(false);
  expect(points(ff.d)[1]).toEqual([200 + STUB, 14]);
  expect(points(ff.arrow)[1][0]).toBe(180 + ARROW); // left-pointing

  // FF whose successor finishes later must go round to come in from the right.
  const ffBack = route({ px: 200, py: 14, sx: 260, sy: 42, type: "FF", rowH });
  expect(ffBack.back).toBe(true);
  expect(points(ffBack.d).at(-2)).toEqual([260 + STUB, 42]);

  // SF: leaves the start leftwards and enters the successor finish from the right.
  const sf = route({ px: 100, py: 14, sx: 60, sy: 42, type: "SF", rowH });
  expect(sf.back).toBe(false);
  expect(points(sf.d)[1]).toEqual([100 - STUB, 14]);
  expect(points(sf.arrow)[1][0]).toBe(60 + ARROW);
});
