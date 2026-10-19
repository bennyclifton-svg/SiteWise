// Same cases as internal/program/parse_test.go: the grid and the Go importer
// must read every typed value the same way.

import { describe, expect, test } from "vitest";
import {
  type PredRef,
  DayMonth,
  MonthDay,
  decideDayOrder,
  formatPredecessor,
  isMilestone,
  parseDate,
  parseDuration,
  parsePredecessors,
} from "./parse";

describe("parseDuration", () => {
  const good: [string, number, number, number][] = [
    ["5", 5, 5, 5],
    ["5d", 5, 5, 5],
    [" 5 D ", 5, 5, 5],
    ["10 days", 5, 10, 10],
    ["1 day", 5, 1, 1],
    ["3w", 5, 15, 15],
    ["3w", 6, 18, 18], // weeks are working weeks of the program's calendar
    ["3 wks", 5, 15, 15],
    ["2 weeks", 4, 8, 8],
    ["1.5w", 4, 6, 6],
    ["0", 5, 0, 0],
    ["4–6 wks", 5, 20, 30],
    ["4-6 weeks", 5, 20, 30],
    ["4 to 6 wks", 5, 20, 30],
    ["2w-15d", 5, 10, 15],
    ["3650", 5, 3650, 3650],
  ];
  for (const [input, perWeek, low, high] of good) {
    test(`${JSON.stringify(input)} at ${perWeek} days a week`, () => {
      expect(parseDuration(input, perWeek)).toStrictEqual({ ok: true, value: { low, high } });
    });
  }

  test("0 makes a milestone", () => {
    const d = parseDuration("0", 5);
    expect(d.ok && isMilestone(d.value)).toBe(true);
    const five = parseDuration("5", 5);
    expect(five.ok && isMilestone(five.value)).toBe(false);
  });

  test("a missing week length is five days", () => {
    expect(parseDuration("2w", 0)).toStrictEqual({ ok: true, value: { low: 10, high: 10 } });
  });

  for (const bad of ["", "abc", "-3", "5x", "1.5d", "1.5", "6–4 wks", "3651", "2 months", "5 edays", "4–"]) {
    test(`refuses ${JSON.stringify(bad)} with a message`, () => {
      const r = parseDuration(bad, 5);
      expect(r.ok).toBe(false);
      expect(r.ok ? "" : r.error).not.toBe("");
    });
  }

  test("messages name the problem", () => {
    expect(parseDuration("", 5)).toStrictEqual({ ok: false, error: "Enter a duration, for example 5, 5d or 3w." });
    expect(parseDuration("6–4 wks", 5)).toStrictEqual({
      ok: false,
      error: "A range runs from low to high, for example 4–6 wks.",
    });
    expect(parseDuration("1.5d", 5)).toStrictEqual({ ok: false, error: "1.5d is not a whole number of working days." });
    expect(parseDuration("3651", 5)).toStrictEqual({ ok: false, error: "Duration must be at most 3650 working days." });
    expect(parseDuration("2 months", 5)).toStrictEqual({
      ok: false,
      error: '"months" is not a unit. Use d for working days or w for weeks.',
    });
  });
});

const fs = (row: number, lag = 0, unit = "working"): PredRef => ({ row, type: "FS", lag, lag_unit: unit });

describe("parsePredecessors", () => {
  const good: [string, PredRef[]][] = [
    ["12", [fs(12)]],
    ["12FS", [fs(12)]],
    ["12SS+2d", [{ row: 12, type: "SS", lag: 2, lag_unit: "working" }]],
    ["8FF-1d", [{ row: 8, type: "FF", lag: -1, lag_unit: "working" }]],
    ["4SF", [{ row: 4, type: "SF", lag: 0, lag_unit: "working" }]],
    ["7FS+3cd", [fs(7, 3, "calendar")]],
    ["7fs+3ed", [fs(7, 3, "calendar")]],
    ["5+1w", [fs(5, 5)]],
    ["3ss + 2", [{ row: 3, type: "SS", lag: 2, lag_unit: "working" }]],
    [
      "12, 8FF-1d; 4SF",
      [fs(12), { row: 8, type: "FF", lag: -1, lag_unit: "working" }, { row: 4, type: "SF", lag: 0, lag_unit: "working" }],
    ],
    ["", []],
    [" , ", []],
  ];
  for (const [input, want] of good) {
    test(JSON.stringify(input), () => {
      expect(parsePredecessors(input, 5)).toStrictEqual({ ok: true, value: want });
    });
  }

  for (const bad of ["0", "12XX", "12FS+", "a", "12,12FS", "1FS+400d", "1FS+80w", "12FS+2x", "-3"]) {
    test(`refuses ${JSON.stringify(bad)}`, () => {
      expect(parsePredecessors(bad, 5).ok).toBe(false);
    });
  }

  test("messages name the problem", () => {
    expect(parsePredecessors("0", 5)).toStrictEqual({ ok: false, error: "Row numbers start at 1." });
    expect(parsePredecessors("12,12FS", 5)).toStrictEqual({ ok: false, error: "Row 12 is listed twice." });
    expect(parsePredecessors("1FS+80w", 5)).toStrictEqual({ ok: false, error: "Lag must be between −365 and 365 days." });
  });
});

// What export writes, import must read back as the same link.
test("formatPredecessor round-trips through parsePredecessors", () => {
  const refs: PredRef[] = [
    fs(12),
    { row: 12, type: "SS", lag: 2, lag_unit: "working" },
    { row: 8, type: "FF", lag: -1, lag_unit: "working" },
    fs(7, 3, "calendar"),
    { row: 4, type: "SF", lag: 0, lag_unit: "working" },
  ];
  for (const r of refs) {
    expect(parsePredecessors(formatPredecessor(r), 5)).toStrictEqual({ ok: true, value: [r] });
  }
  expect(refs.map(formatPredecessor)).toStrictEqual(["12", "12SS+2d", "8FF-1d", "7FS+3cd", "4SF"]);
});

describe("parseDate", () => {
  const good: [string, typeof DayMonth, string][] = [
    ["1/3/2027", DayMonth, "2027-03-01"],
    ["1/3/2027", MonthDay, "2027-01-03"],
    ["1/3/27", DayMonth, "2027-03-01"],
    ["01.03.2027", DayMonth, "2027-03-01"],
    ["12 Mar 2027", DayMonth, "2027-03-12"],
    ["12 March 2027", MonthDay, "2027-03-12"], // order applies to numeric dates only
    ["12-Mar-27", DayMonth, "2027-03-12"],
    ["2027-03-12", MonthDay, "2027-03-12"],
    ["Mon 1/03/27", DayMonth, "2027-03-01"],
    ["Monday, 1 March 2027", DayMonth, "2027-03-01"],
    ["1/03/2027 8:00 AM", DayMonth, "2027-03-01"],
    ["29/2/2028", DayMonth, "2028-02-29"],
  ];
  for (const [input, order, want] of good) {
    test(`${JSON.stringify(input)} as ${order}`, () => {
      expect(parseDate(input, order)).toStrictEqual({ ok: true, value: want });
    });
  }
  for (const bad of ["", "tomorrow", "31/4/2027", "29/2/2027", "12 Foo 2027", "13/13/2027", "2027-02-30", "1/3", "1 constructor 2027"]) {
    test(`refuses ${JSON.stringify(bad)}`, () => {
      expect(parseDate(bad, DayMonth).ok).toBe(false);
    });
  }
});

describe("decideDayOrder", () => {
  const cases: [string[] | null, typeof DayMonth, boolean][] = [
    [["13/2/2027", "1/3/2027"], DayMonth, false],
    [["2/13/2027", "3/1/2027"], MonthDay, false],
    [["1/3/2027", "5/5/2027"], DayMonth, true],
    [["5/5/2027", "2027-01-02", "12 Mar 2027", "TBA"], DayMonth, false],
    [["13/2/2027", "2/13/2027"], DayMonth, false], // a first part over 12 decides first
    [null, DayMonth, false],
  ];
  for (const [values, order, assumed] of cases) {
    test(JSON.stringify(values), () => {
      expect(decideDayOrder(values)).toStrictEqual({ order, assumed });
    });
  }
});
