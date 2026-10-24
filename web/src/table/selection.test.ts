import { expect, test } from "vitest";
import { clickRow, emptySelection, prune, selectAll } from "./selection";

const order = ["s1", "a", "b", "c", "s2", "d", "e"];

test("a click selects one row and sets the anchor", () => {
  const s = clickRow(clickRow(emptySelection, "a", order), "c", order);
  expect(s.ids).toEqual(["c"]);
  expect(s.anchor).toBe("c");
});

test("Ctrl-click toggles one row and moves the anchor to it", () => {
  let s = clickRow(emptySelection, "a", order);
  s = clickRow(s, "c", order, { ctrl: true });
  expect(s.ids).toEqual(["a", "c"]);
  s = clickRow(s, "a", order, { ctrl: true });
  expect(s.ids).toEqual(["c"]);
  expect(s.anchor).toBe("a");
});

test("Shift-click selects from the anchor in grid order and replaces the previous range", () => {
  let s = clickRow(emptySelection, "b", order);
  s = clickRow(s, "e", order, { shift: true });
  expect(s.ids).toEqual(["b", "c", "s2", "d", "e"]);
  s = clickRow(s, "a", order, { shift: true });
  expect(s.ids).toEqual(["a", "b"]);
});

test("Shift-click keeps Ctrl-clicked rows", () => {
  let s = clickRow(emptySelection, "s1", order);
  s = clickRow(s, "d", order, { ctrl: true });
  s = clickRow(s, "e", order, { shift: true });
  // Ctrl-click moved the anchor to d, so the range is d..e, plus s1.
  expect([...s.ids].sort()).toEqual(["d", "e", "s1"]);
  // A new range from the same anchor replaces d..e and still keeps s1.
  s = clickRow(s, "s2", order, { shift: true });
  expect([...s.ids].sort()).toEqual(["d", "s1", "s2"]);
});

test("Ctrl+Shift adds a range to the selection", () => {
  let s = clickRow(emptySelection, "a", order);
  s = clickRow(s, "d", order, { ctrl: true });
  s = clickRow(s, "e", order, { ctrl: true, shift: true });
  expect(s.ids).toEqual(["a", "d", "e"]);
});

test("in the collapsed view a range covers stages only", () => {
  const collapsed = ["s1", "s2", "s3"];
  let s = clickRow(emptySelection, "s1", collapsed);
  s = clickRow(s, "s3", collapsed, { shift: true });
  expect(s.ids).toEqual(["s1", "s2", "s3"]);
});

test("select all and prune follow what is visible", () => {
  expect(selectAll(order).ids).toEqual(order);
  const s = prune({ ids: ["a", "d"], anchor: "d", base: ["a"] }, ["s1", "a", "s2"]);
  expect(s).toEqual({ ids: ["a"], anchor: null, base: ["a"] });
});
