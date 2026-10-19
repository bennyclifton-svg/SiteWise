import { expect, test } from "vitest";
import { Cal } from "./calendar";
import type { Times } from "./types";
import type { ProgramActivity, ProgramLink } from "../programApi";
import { barOf, breaks, crisp, drawnLinks, extent, labelPlacement, outlineRows, rowWindow, shownDates, ticks, xOf } from "./layout";

const act = (id: string, parent: string | null = null): ProgramActivity => ({
  id, parent_id: parent, ordinal: 0, kind: parent ? "activity" : "stage", label: id,
  duration_days: null, duration_low: null, duration_high: null, duration_origin: null,
  constraint_kind: "asap", constraint_date: null, review_status: "proposed", origin: "library", version: 1,
});
const link = (id: string, pred: string, succ: string, type = "FS"): ProgramLink => ({
  id, pred_id: pred, succ_id: succ, type, lag_days: 0, lag_unit: "working", origin: "library", version: 1,
});
const times = (es: number | null, ef: number | null): Times => ({
  es, ef, ls: es, lf: ef, tf: 0, ff: 0, critical: false, start: null, finish: null, unknown: es === null,
});

// Design (a1, a2), Approvals (no activities), Build (b1).
const acts = [act("design"), act("a1", "design"), act("a2", "design"), act("approvals"), act("build"), act("b1", "build")];
const parents = new Map(acts.map((a) => [a.id, a.parent_id]));

test("collapsed shows stages only; expanding a stage shows its activities in outline order", () => {
  const collapsed = outlineRows(acts, new Set());
  expect(collapsed.map((r) => r.id)).toEqual(["design", "approvals", "build"]);
  expect(collapsed.map((r) => r.hasChildren)).toEqual([true, false, true]);
  const open = outlineRows(acts, new Set(["design"]));
  expect(open.map((r) => r.id)).toEqual(["design", "a1", "a2", "approvals", "build"]);
  // Row numbers stay the outline position whatever is collapsed.
  expect(open.map((r) => r.number)).toEqual([1, 2, 3, 4, 5]);
  expect(outlineRows(acts, new Set(["build"])).map((r) => r.number)).toEqual([1, 4, 5, 6]);
});

test("bars: x from the view start, at least 2 px wide, a zero-length row is a diamond, unknown has none", () => {
  const v = { pxPerDay: 10, viewStart: -1, rowH: 28 };
  expect(xOf(4, v)).toBe(50);
  expect(barOf(times(4, 9), v)).toEqual({ x: 50, w: 50, milestone: false });
  expect(barOf(times(4, 5), { ...v, pxPerDay: 0.5 })?.w).toBe(2);
  expect(barOf(times(4, 4), v)).toEqual({ x: 50, w: 10, milestone: true });
  expect(barOf(times(null, null), v)).toBeNull();
  expect(barOf(undefined, v)).toBeNull();
});

test("crisp edges land on half pixels", () => {
  expect(crisp(10.2)).toBe(10.5);
  expect(crisp(10.7)).toBe(11.5);
});

test("the render window pads ten rows each side and clamps to the row count", () => {
  expect(rowWindow(0, 280, 28, 2000)).toEqual([0, 20]);
  expect(rowWindow(2800, 280, 28, 2000)).toEqual([90, 120]);
  expect(rowWindow(56000, 280, 28, 2000)).toEqual([1990, 2000]);
  expect(rowWindow(0, 280, 28, 3)).toEqual([0, 3]);
});

test("a label goes inside the bar when it fits, else after the end", () => {
  expect(labelPlacement(40, { x: 100, w: 60, milestone: false })).toEqual({ x: 106, inside: true });
  expect(labelPlacement(50, { x: 100, w: 60, milestone: false })).toEqual({ x: 166, inside: false });
  expect(labelPlacement(10, { x: 100, w: 10, milestone: true }).inside).toBe(false);
});

test("links into collapsed stages fold onto the stage bars, faint and once per pair", () => {
  const links = [link("l1", "a1", "b1"), link("l2", "a2", "b1"), link("l3", "a1", "a2"), link("l4", "design", "approvals")];
  const rows = outlineRows(acts, new Set());
  const drawn = drawnLinks(links, rows, parents);
  // l1 and l2 both become design → build; l3 is inside one stage and is not drawn.
  expect(drawn.map((d) => [d.link.id, d.pred, d.succ, d.faint])).toEqual([
    ["l1", 0, 2, true],
    ["l4", 0, 1, false],
  ]);
  // Expanded, every link draws at its own rows, none faint.
  const all = drawnLinks(links, outlineRows(acts, new Set(["design", "build"])), parents);
  expect(all.map((d) => d.faint)).toEqual([false, false, false, false]);
});

test("a link is drawn only when an end is inside the window", () => {
  const rows = outlineRows(acts, new Set(["design", "build"]));
  const links = [link("near", "a1", "a2"), link("far", "b1", "build", "SS"), link("cross", "a1", "b1")];
  expect(drawnLinks(links, rows, parents, 0, 3).map((d) => d.link.id)).toEqual(["near", "cross"]);
  expect(drawnLinks(links, rows, parents, 4, 6).map((d) => d.link.id)).toEqual(["far", "cross"]);
});

test("extent spans known rows and widens to the high finish", () => {
  const t = { a: times(2, 7), b: times(5, 20), c: times(null, null) };
  expect(extent(["a", "b", "c"], t)).toEqual([2, 20]);
  expect(extent(["a"], t, 30)).toEqual([2, 30]);
  expect(extent(["c"], t)).toBeNull();
});

test("shown dates: the last working day as finish; Week N, day M without a start date", () => {
  const dated = new Cal({ start: "2027-03-01", working_week: 31 });
  const t: Times = { ...times(0, 5), start: "2027-03-01", finish: "2027-03-05" };
  expect(shownDates(dated, t)).toEqual({ start: "1 Mar 2027", finish: "5 Mar 2027" });
  const relative = new Cal({ start: "", working_week: 31 });
  expect(shownDates(relative, times(0, 5))).toEqual({ start: "Week 1, day 1", finish: "Week 1, day 5" });
  expect(shownDates(relative, times(5, 6))).toEqual({ start: "Week 2, day 1", finish: "Week 2, day 1" });
  // A milestone shows its start only.
  expect(shownDates(relative, times(7, 7))).toEqual({ start: "Week 2, day 3", finish: "" });
});

test("ticks: months over days, Monday weeks, years over quarters, and weeks on a relative program", () => {
  const cal = new Cal({ start: "2027-03-29", working_week: 31 }); // a Monday
  const d = ticks(cal, 0, 6, "days");
  expect(d.major.map((t) => t.label)).toEqual(["Mar 2027", "Apr 2027"]);
  expect(d.major[1].index).toBe(3); // Thu 1 April
  expect(d.minor.map((t) => t.label)).toEqual(["29", "30", "31", "1", "2", "5"]);
  const w = ticks(cal, 0, 10, "weeks");
  expect(w.minor.map((t) => [t.index, t.label])).toEqual([[0, "29"], [5, "5"]]);
  const q = ticks(cal, 0, 300, "quarters");
  expect(q.minor.map((t) => t.label).slice(0, 3)).toEqual(["Q1", "Q2", "Q3"]);
  expect(q.major.map((t) => t.label)).toEqual(["2027", "2028"]);
  const rel = ticks(new Cal({ start: "", working_week: 31 }), 0, 11, "days");
  expect(rel.major.map((t) => t.label)).toEqual(["Week 1", "Week 2", "Week 3"]);
  expect(rel.minor[5].label).toBe("d1");
});

test("breaks mark the working day after a holiday, not after a plain weekend", () => {
  const days = [{ day: "2027-04-02", kind: "public_holiday" }]; // Good Friday
  const cal = new Cal({ start: "2027-03-29", working_week: 31, days });
  const plain = new Cal({ start: "2027-03-29", working_week: 31 });
  // Index 4 is Mon 5 April: after the Good Friday gap. Index 9 follows a plain weekend.
  expect(breaks(cal, plain, 0, 12)).toEqual([4]);
});
