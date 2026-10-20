import { describe, expect, it } from "vitest";
import { Cal } from "./calendar";
import { compute } from "./engine";
import {
  boxHits,
  completionBadge,
  deletePlan,
  dropTarget,
  heldMessage,
  heldRows,
  linkType,
  moveTargets,
  kidsOf,
  predecessorOps,
  predecessorText,
  reorderStep,
  rowNumbers,
  snapDays,
  snapUnit,
  withMoves,
  withResize,
  zoneAt,
} from "./edit";
import { parsePredecessors } from "./grid/parse";
import { outlineRows } from "./layout";
import type { ProgramActivity, ProgramLink, ProgramView } from "../programApi";
import { toInput } from "../programApi";

function act(id: string, parent: string | null, label: string, d: number | null = 5, kind = parent ? "activity" : "stage"): ProgramActivity {
  return { id, parent_id: parent, ordinal: 0, kind, label, duration_days: d, duration_low: null, duration_high: null, duration_origin: "user", constraint_kind: "asap", constraint_date: null, review_status: "accepted", origin: "user", version: 1 };
}
function link(id: string, pred: string, succ: string, type = "FS", lag = 0, unit = "working"): ProgramLink {
  return { id, pred_id: pred, succ_id: succ, type, lag_days: lag, lag_unit: unit, origin: "user", version: 1 };
}

// Design (A, B) then Build (Slab, Framing, Hold), Slab → Framing FS.
const acts = [
  act("D", null, "Design", null),
  act("A", "D", "Concept", 5),
  act("B", "D", "Approvals", 10),
  act("C", null, "Build", null),
  act("S", "C", "Slab", 10),
  act("F", "C", "Framing", 15),
  act("H", "C", "Inspection", 0, "hold_point"),
];
const links = [link("l1", "A", "B"), link("l2", "S", "F"), link("l3", "D", "C"), link("l4", "F", "H", "SS", 2, "calendar")];
const view = { program: { id: "p", project_id: "x", version: 1, start_date: "2027-03-01", start_origin: null, skeleton_id: null, stage_revision: 1, working_week: 31 }, activities: acts, links, calendar: { days: [] } } as unknown as ProgramView;
const input = toInput(view);
const cal = new Cal(input.calendar);
const base = compute(input);

describe("zoneAt", () => {
  it("puts end zones inside a wide bar and outside a narrow one", () => {
    expect(zoneAt(103, 100, 100, 8)).toBe("start");
    expect(zoneAt(195, 100, 100, 8)).toBe("end");
    expect(zoneAt(150, 100, 100, 8)).toBe("body");
    // 30 px is too narrow for three 24 px coarse zones: the zones move outside.
    expect(zoneAt(110, 100, 30, 24)).toBe("body");
    expect(zoneAt(95, 100, 30, 24)).toBe("start");
    expect(zoneAt(135, 100, 30, 24)).toBe("end");
  });
  it("only moves a stage with activities and only grows a milestone from its right", () => {
    expect(zoneAt(101, 100, 100, 8, { summary: true })).toBe("body");
    expect(zoneAt(97, 95, 10, 8, { milestone: true })).toBe("body");
    expect(zoneAt(106, 95, 10, 8, { milestone: true })).toBe("end");
  });
});

describe("snapping", () => {
  it("snaps to days when zoomed in or Alt is held, otherwise to working weeks", () => {
    expect(snapUnit(12, false, 5)).toBe(1);
    expect(snapUnit(6, false, 5)).toBe(5);
    expect(snapUnit(6, true, 5)).toBe(1);
    expect(snapDays(40, 12, 1)).toBe(3);
    expect(snapDays(40, 6, 5)).toBe(5); // 6.7 days → one week
    expect(snapDays(10, 6, 5)).toBe(0);
    expect(Object.is(snapDays(-1, 12, 1), 0)).toBe(true);
  });
});

describe("linkType", () => {
  it("reads the type from the two ends", () => {
    expect(linkType("F", "S")).toBe("FS");
    expect(linkType("S", "S")).toBe("SS");
    expect(linkType("F", "F")).toBe("FF");
    expect(linkType("S", "F")).toBe("SF");
  });
});

describe("move preview", () => {
  const kids = kidsOf(acts);
  it("moves a stage's activities with it", () => {
    expect(moveTargets(["C", "A"], kids)).toEqual(["S", "F", "H", "A"]);
  });
  it("dragging later shifts the row and its successors", () => {
    const out = compute(withMoves(input, base.activities, ["S"], 3, cal));
    expect(out.activities.S.es).toBe(base.activities.S.es! + 3);
    expect(out.activities.F.es).toBe(base.activities.F.es! + 3);
    expect(heldRows(["S"], base.activities, out, 3)).toEqual([]);
    expect(completionBadge(base.finish.typical_index, out.finish.typical_index)).toBe("+3 days to completion");
  });
  it("dragging earlier than a link allows is held, named like the server", () => {
    const out = compute(withMoves(input, base.activities, ["F"], -2, cal));
    expect(heldRows(["F"], base.activities, out, -2)).toEqual(["F"]);
    expect(heldMessage(["F"], base.activities, out, -2, acts, links)).toBe("Held by Slab");
  });
  it("a bulk move with one held row names each held row", () => {
    const out = compute(withMoves(input, base.activities, ["A", "B"], -1, cal));
    const held = heldRows(["A", "B"], base.activities, out, -1);
    expect(held).toEqual(["A", "B"]);
    expect(heldMessage(held, base.activities, out, -1, acts, links)).toBe("Concept is held by the program start; Approvals is held by Concept.");
  });
  it("resizing from the right lengthens; a milestone given days becomes an activity", () => {
    const out = compute(withResize(input, "S", 12, null, cal));
    expect(out.activities.S.ef! - out.activities.S.es!).toBe(12);
    const ms = withResize(input, "H", 3, null, cal).activities!.find((a) => a.id === "H")!;
    expect(ms.duration).toBe(3);
  });
});

describe("predecessors", () => {
  const numbers = rowNumbers(acts);
  it("writes the typed format", () => {
    expect(predecessorText("H", links, numbers)).toBe("6SS+2cd");
    expect(predecessorText("F", links, numbers)).toBe("5");
  });
  it("diffs typed predecessors into link ops", () => {
    const refs = parsePredecessors("2SS+2d, 5", 5);
    if (!refs.ok) throw new Error(refs.error);
    const r = predecessorOps("F", refs.value, acts, links);
    expect(r).toEqual({ ops: [{ op: "link_add", pred_id: "A", succ_id: "F", type: "SS", lag_days: 2, lag_unit: "working" }] });
    const changed = parsePredecessors("5FF-1d", 5);
    if (!changed.ok) throw new Error(changed.error);
    expect(predecessorOps("F", changed.value, acts, links)).toEqual({ ops: [{ op: "link_update", id: "l2", type: "FF", lag_days: -1, lag_unit: "working" }] });
    expect(predecessorOps("F", [], acts, links)).toEqual({ ops: [{ op: "link_delete", ids: ["l2"] }] });
  });
  it("refuses rows that do not exist and a row following itself", () => {
    expect(predecessorOps("F", [{ row: 99, type: "FS", lag: 0, lag_unit: "working" }], acts, links)).toEqual({ error: "There is no row 99." });
    expect(predecessorOps("F", [{ row: 6, type: "FS", lag: 0, lag_unit: "working" }], acts, links)).toEqual({ error: "A row can't follow itself." });
  });
});

describe("deletePlan", () => {
  it("counts a stage's activities and links, and names hold points", () => {
    expect(deletePlan(["C"], acts, links)).toEqual({ rows: 4, links: 3, stages: ["Build"], holds: ["Inspection"] });
    expect(deletePlan(["A"], acts, links)).toEqual({ rows: 1, links: 1, stages: [], holds: [] });
  });
});

describe("reorderStep", () => {
  it("moves stages among stages", () => {
    expect(reorderStep(["C"], acts, -1)).toEqual({ op: "reorder", ids: ["C"], after_id: undefined });
    expect(reorderStep(["D"], acts, 1)).toEqual({ op: "reorder", ids: ["D"], after_id: "C" });
    expect(reorderStep(["C"], acts, 1)).toBeNull();
  });
  it("moves activities among siblings and over a stage edge", () => {
    expect(reorderStep(["F"], acts, -1)).toEqual({ op: "reorder", ids: ["F"], parent_id: "C", after_id: undefined });
    expect(reorderStep(["B"], acts, 1)).toEqual({ op: "reorder", ids: ["B"], parent_id: "C" });
    expect(reorderStep(["S"], acts, -1)).toEqual({ op: "reorder", ids: ["S"], parent_id: "D", after_id: "B" });
    expect(reorderStep(["A", "C"], acts, 1)).toBeNull();
  });
});

describe("dropTarget", () => {
  const rows = outlineRows(acts, new Set(["D", "C"])); // D A B C S F H
  it("drops a stage after the stage above the line", () => {
    expect(dropTarget(rows, ["D"], 7 * 28, 28, acts)?.op).toEqual({ op: "reorder", ids: ["D"], after_id: "C" });
    expect(dropTarget(rows, ["C"], 0, 28, acts)?.op).toEqual({ op: "reorder", ids: ["C"], after_id: undefined });
  });
  it("drops activities between activities, or onto a stage row's middle to its end", () => {
    expect(dropTarget(rows, ["A"], 5 * 28, 28, acts)?.op).toEqual({ op: "reorder", ids: ["A"], parent_id: "C", after_id: "S" });
    expect(dropTarget(rows, ["A"], 3 * 28 + 14, 28, acts)?.op).toEqual({ op: "reorder", ids: ["A"], parent_id: "C", after_id: "H" });
    expect(dropTarget(rows, ["S"], 0, 28, acts)).toBeNull();
  });
});

describe("boxHits", () => {
  it("returns the bars a box touches", () => {
    const rows = outlineRows(acts, new Set(["D", "C"]));
    const v = { pxPerDay: 10, viewStart: 0, rowH: 28 };
    // Rows 1–2 (Concept 0–5, Approvals 5–15): a box over x 0–40 touches Concept only.
    expect(boxHits(rows, base.activities, v, 0, 30, 40, 70)).toEqual(["A"]);
    expect(boxHits(rows, base.activities, v, 0, 30, 60, 70)).toEqual(["A", "B"]);
  });
});
