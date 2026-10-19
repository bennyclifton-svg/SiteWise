import { expect, test } from "vitest";
import { fitHeight, fitSpan, scaleFor, zoomAbout } from "./fit";

test("fit to a span puts first − 1 at the left edge and leaves the margin free", () => {
  // 100 working days on an 1,048 px chart: (1048 − 48) / (100 + 2).
  const f = fitSpan([10, 110], 1048);
  expect(f.viewStart).toBe(9);
  expect(f.pxPerDay).toBeCloseTo(1000 / 102);
  // The last index then lands inside the chart with the margin to spare.
  expect((110 - f.viewStart) * f.pxPerDay).toBeLessThanOrEqual(1048 - 48 + f.pxPerDay);
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
