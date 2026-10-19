// Key cases of internal/program/calendar_test.go, plus the same day-by-day
// cross-check of the fast arithmetic and the lag boundary helpers.

import { describe, expect, test } from "vitest";
import { Cal, RelativeEpoch, dayNumber, dayString, weekdayBit } from "./calendar";
import type { CalendarDay } from "./types";

const cal = (start: string, week: number, ...days: CalendarDay[]) => new Cal({ start, working_week: week, days });
const day = (d: string, kind: string): CalendarDay => ({ day: d, kind });

test("dates parse strictly and format back", () => {
  expect(dayNumber("1970-01-01")).toBe(0);
  expect(dayNumber("2027-03-01")).toBe(20878);
  expect(dayString(20878)).toBe("2027-03-01");
  expect(dayString(-1)).toBe("1969-12-31");
  expect(dayNumber("2028-02-29")).not.toBeNull();
  for (const bad of ["2027-02-29", "2027-13-01", "2027-00-10", "2027-3-01", "27-03-01", "", "2027-03-01T00:00"]) {
    expect(dayNumber(bad), bad).toBeNull();
  }
  // Every day across four centuries round-trips.
  for (let d = -40000; d < 120000; d += 37) {
    expect(dayNumber(dayString(d))).toBe(d);
  }
});

test("weekend", () => {
  const c = cal("2027-03-05", 31); // a Friday
  expect(c.dateOf(0)).toBe("2027-03-05");
  expect(c.dateOf(1)).toBe("2027-03-08");
  expect(c.dateOf(-1)).toBe("2027-03-04");
  for (const [date, after, before] of [
    ["2027-03-06", 1, 0],
    ["2027-03-07", 1, 0],
    ["2027-03-08", 1, 1],
    ["2027-03-01", -4, -4],
  ] as const) {
    expect([c.indexOnOrAfter(date), c.indexOnOrBefore(date)], date).toStrictEqual([after, before]);
  }
  expect(cal("2027-03-07", 31).dateOf(0)).toBe("2027-03-08"); // Sunday start
});

test("holiday", () => {
  const c = cal("2027-03-01", 31, day("2027-03-08", "public_holiday"));
  expect(c.dateOf(5)).toBe("2027-03-09");
  expect(c.indexOnOrAfter("2027-03-08")).toBe(5);
  expect(c.indexOnOrBefore("2027-03-08")).toBe(4);
  // A holiday on a weekend changes nothing.
  expect(cal("2027-03-01", 31, day("2027-03-06", "public_holiday")).dateOf(5)).toBe("2027-03-08");
  // Fri 5 Mar plus 3 calendar days is Mon 8, a holiday, so Tue 9.
  expect(c.shiftCalendar(4, 3)).toBe(5);
});

test("shutdown across the year end", () => {
  const days = [
    "2027-12-24", "2027-12-27", "2027-12-28", "2027-12-29", "2027-12-30", "2027-12-31",
    "2028-01-03", "2028-01-04", "2028-01-05", "2028-01-06", "2028-01-07",
  ].map((d) => day(d, "shutdown"));
  const c = cal("2027-12-20", 31, ...days);
  const want = ["2027-12-20", "2027-12-21", "2027-12-22", "2027-12-23", "2028-01-10", "2028-01-11"];
  want.forEach((w, i) => expect(c.dateOf(i), `index ${i}`).toBe(w));
  expect(c.indexOnOrAfter("2027-12-25")).toBe(4);
  expect(c.indexOnOrBefore("2028-01-09")).toBe(3);
  // The warm cache gives the same dates as a search per date.
  c.warm(-3, 40, 10);
  want.forEach((w, i) => expect(c.dateOf(i)).toBe(w));
});

test("working override and a six-day week", () => {
  const c = cal("2027-03-01", 31, day("2027-03-06", "working_override"), day("2027-03-03", "working_override"));
  const want = ["2027-03-01", "2027-03-02", "2027-03-03", "2027-03-04", "2027-03-05", "2027-03-06", "2027-03-08"];
  want.forEach((w, i) => expect(c.dateOf(i)).toBe(w));
  expect(cal("2027-03-01", 63).dateOf(5)).toBe("2027-03-06");
});

test("relative program", () => {
  const c = cal("", 0, day("2027-03-02", "public_holiday"), day("2001-01-02", "public_holiday"));
  expect(c.relative).toBe(true);
  // Dates count from RelativeEpoch (a Monday) for stored constraints; holiday
  // rows are ignored.
  expect(c.dateOf(0)).toBe(RelativeEpoch);
  expect(c.dateOf(1)).toBe("2001-01-02");
  expect(c.dateOf(5)).toBe("2001-01-08");
  expect(c.indexOnOrAfter("2001-01-06")).toBe(5);
  expect(c.indexOnOrBefore("2001-01-06")).toBe(4);
  expect(c.weekLabel(4)).toBe("Week 1");
  expect(c.weekLabel(5)).toBe("Week 2");
  expect(c.dayLabel(6)).toBe("Week 2, day 2");
  expect(c.display(0, 5)).toStrictEqual(["", ""]);
  expect(c.indexOnOrAfter("")).toBeNull();
});

test("display is half-open", () => {
  const c = cal("2027-03-01", 31);
  expect(c.display(0, 5)).toStrictEqual(["2027-03-01", "2027-03-05"]);
  expect(c.display(3, 1)).toStrictEqual(["2027-03-04", "2027-03-04"]);
  expect(c.display(5, 0)).toStrictEqual(["2027-03-08", ""]);
});

// Deterministic generator (mulberry32) so failures reproduce.
function rng(seed: number): (n: number) => number {
  let a = seed >>> 0;
  return (n) => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return Math.floor((((t ^ (t >>> 14)) >>> 0) / 4294967296) * n);
  };
}

const kindsList = ["public_holiday", "rdo", "shutdown", "working_override"];

function randomCalendar(r: (n: number) => number) {
  const mask = 1 + r(127);
  const start = (dayNumber("2027-03-01") as number) + r(14);
  const kinds = new Map<number, string>();
  const days: CalendarDay[] = [];
  const count = r(60);
  for (let k = 0; k < count; k++) {
    const off = r(1400) - 300;
    const kind = kindsList[r(4)];
    if (kinds.has(off)) continue;
    kinds.set(off, kind);
    days.push(day(dayString(start + off), kind));
  }
  return { c: new Cal({ start: dayString(start), working_week: mask, days }), mask, kinds, start };
}

describe("fast arithmetic agrees with a day-by-day walk", () => {
  test("dayOf, indexOnOrAfter and indexOnOrBefore", () => {
    const r = rng(3);
    for (let n = 0; n < 200; n++) {
      const { c, mask, kinds, start } = randomCalendar(r);
      const work: number[] = [];
      for (let d = -400; d < 1200; d++) {
        let works = (weekdayBit(start + d) & mask) !== 0;
        const k = kinds.get(d);
        if (k !== undefined) works = k === "working_override";
        if (works) work.push(d);
      }
      let zero = 0;
      while (zero < work.length && work[zero] < 0) zero++;
      work.forEach((d, k) => {
        const i = k - zero;
        if (i < -100 || i > 600) return;
        if (c.dayOf(i) !== d) throw new Error(`case ${n}: index ${i} is day ${c.dayOf(i)}, walk says ${d}`);
        const date = dayString(start + d);
        if (c.indexOnOrAfter(date) !== i || c.indexOnOrBefore(date) !== i) {
          throw new Error(`case ${n}: ${date} round trip`);
        }
      });
    }
  });

  // "On or after" is the least index meeting the bound, "on or before" the
  // greatest; being exact inverses keeps the backward pass from inventing float.
  test("lag boundary helpers", () => {
    const r = rng(5);
    const least = (ok: (i: number) => boolean) => {
      for (let i = -200; ; i++) if (ok(i)) return i;
    };
    const greatest = (ok: (i: number) => boolean) => {
      for (let i = 800; ; i--) if (ok(i)) return i;
    };
    for (let n = 0; n < 100; n++) {
      const { c } = randomCalendar(r);
      for (let k = 0; k < 50; k++) {
        const m = r(500) - 50;
        const d = [0, 1, 3, 10][r(4)];
        expect(c.startOnOrAfter(m)).toBe(least((s) => c.startDay(s) >= m));
        expect(c.startOnOrBefore(m)).toBe(greatest((s) => c.startDay(s) <= m));
        expect(c.finishOnOrAfter(m, d)).toBe(least((f) => c.finishDay(f, d) >= m));
        expect(c.finishOnOrBefore(m, d)).toBe(greatest((f) => c.finishDay(f, d) <= m));
      }
    }
  });
});
