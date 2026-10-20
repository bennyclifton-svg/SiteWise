import { expect, test } from "vitest";
import { fitHeight, fitSpan, LEFT_ROOM, scaleFor, zoomAbout } from "./fit";

test("fit to a span leaves one day and the dot room free on the left and the margin on the right", () => {
  // 100 working days on a 1,088 px chart: (1088 − 48 − 40) / (100 + 2).
  const f = fitSpan([10, 110], 1088);
  expect(f.pxPerDay).toBeCloseTo(1000 / 102);
  expect(f.viewStart).toBeCloseTo(9 - LEFT_ROOM / f.pxPerDay);
  // The first bar starts one day plus the room in, so its start dot is visible.
  expect((10 - f.viewStart) * f.pxPerDay).toBeCloseTo(f.pxPerDay + LEFT_ROOM);
  // The last index then lands inside the chart with the margin to spare.
  expect((110 - f.viewStart) * f.pxPerDay).toBeLessThanOrEqual(1088 - 48 + f.pxPerDay);
});

test("at a tight zoom a bar starting on day 0 still has room for its start dot", () => {
  // Two years of working days on a narrow chart: well under 2 px a day.
  const f = fitSpan([0, 520], 700);
  expect(f.pxPerDay).toBeLessThan(2);
  // Start dot: end zone 24 px (coarse) + 6 px + its 4 px radius.
  expect((0 - f.viewStart) * f.pxPerDay).toBeGreaterThanOrEqual(24 + 6 + 4);
});

test("a one-day span does not divide by zero and stays within the zoom limits", () => {
  const f = fitSpan([5, 5], 400);
  expect(Number.isFinite(f.pxPerDay)).toBe(true);
  expect(f.pxPerDay).toBeLessThanOrEqual(96);
});

test("timescale tiers switch at 24, 4 and 1 px a day", () => {
  expect(scaleFor(24)).toBe("days");
  expect(scaleFor(23.9)).toBe("weeks");
  expect(scaleFor(4)).toBe("weeks");
  expect(scaleFor(3.9)).toBe("months");
  expect(scaleFor(1)).toBe("months");
  expect(scaleFor(0.9)).toBe("quarters");
});

test("fit height clamps rows to 20–28 px and says when they still do not fit", () => {
  expect(fitHeight(600, 10)).toEqual({ rowH: 28, tooMany: false });
  expect(fitHeight(600, 25)).toEqual({ rowH: 24, tooMany: false });
  expect(fitHeight(600, 30)).toEqual({ rowH: 20, tooMany: false });
  expect(fitHeight(600, 31)).toEqual({ rowH: 20, tooMany: true });
});

test("zoom keeps the anchor index under the same x", () => {
  const before = { pxPerDay: 10, viewStart: 0 };
  const z = zoomAbout(before.pxPerDay, before.viewStart, 2, 50);
  expect((50 - z.viewStart) * z.pxPerDay).toBeCloseTo((50 - before.viewStart) * before.pxPerDay);
  expect(z.pxPerDay).toBe(20);
});
