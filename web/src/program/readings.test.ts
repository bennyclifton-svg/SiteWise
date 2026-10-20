import { expect, test } from "vitest";
import type { ProgramImport, ProgramReading } from "../programApi";
import { fileLinks, groupReadings, importable, linkText, orphanEvidence, proposedText, reviewWords } from "./readings";

const reading = (over: Partial<ProgramReading>): ProgramReading => ({
  id: "r1",
  import_id: "i1",
  target_activity_id: "a1",
  catalogue_stage_id: null,
  field: "start",
  value: "15/03/2027",
  parsed: { row: "r2", date: "2027-03-15" },
  band: "green",
  decided_by: "rule",
  document_id: "d1",
  document_title: "Smith program.xlsx",
  applies: true,
  source: "program_file",
  ...over,
});

test("a target's start, finish and duration from one file are one group, accepted together", () => {
  const groups = groupReadings([
    reading({ id: "s" }),
    reading({ id: "f", field: "finish", value: "26/03/2027", parsed: { row: "r2", date: "2027-03-26" }, applies: false }),
    reading({ id: "x", target_activity_id: "a2" }),
    reading({ id: "d", field: "duration", value: "10 days", parsed: { row: "r2", low: 10, high: 10 } }),
  ]);
  expect(groups.map((g) => g.ids)).toEqual([["s", "f", "d"], ["x"]]);
  expect(groups[0].applies).toBe(true);
  expect(proposedText(groups[0], (d) => `<${d}>`)).toBe("<2027-03-15> – <2027-03-26>");
});

test("a date that did not parse shows its raw text; a duration alone shows the duration", () => {
  const [g] = groupReadings([reading({ value: "early March", parsed: { row: "r2" } })]);
  expect(proposedText(g, (d) => d)).toBe("from early March");
  const [r7] = groupReadings([reading({ field: "duration", value: "6 weeks", import_id: "", source: "proposal_r7", text: "From the awarded proposal: Design development 6 weeks" })]);
  expect(proposedText(r7, (d) => d)).toBe("6 weeks");
  expect(r7.text).toBe("From the awarded proposal: Design development 6 weeks");
});

test("evidence whose stage was deleted is an orphan; evidence on a live row is not", () => {
  const groups = groupReadings([
    reading({ id: "o", target_activity_id: null, catalogue_stage_id: "approval_documentation" }),
    reading({ id: "l", catalogue_stage_id: "concept_design" }),
    reading({ id: "n", document_id: "d2", target_activity_id: null }),
  ]);
  expect(orphanEvidence(groups).map((g) => g.ids)).toEqual([["o"]]);
});

test("the review banner names each kind of change with its count (plan 7.4)", () => {
  expect(reviewWords({ add: 3, remove: 1, rename: 2, duration: 0, link: 0 }, 6)).toBe("Scope and procurement changed: 3 to add, 1 to remove, 2 renamed.");
  expect(reviewWords({ add: 0, remove: 0, rename: 0, duration: 1, link: 2 }, 3)).toBe("Scope and procurement changed: 1 duration changed, 2 links changed.");
  expect(reviewWords(null, 1)).toBe("Scope and procurement changed: 1 change.");
});

test("only PDF, DOCX and XLSX files are imported", () => {
  expect(importable("Smith program.XLSX")).toBe(true);
  expect(importable("program.docx")).toBe(true);
  expect(importable("scan.pdf")).toBe(true);
  expect(importable("program.mpp")).toBe(false);
  expect(importable("notes.xlsx.txt")).toBe(false);
});

test("a file's proposed links: added on accept while an end waits, otherwise offered to add (plan 10.2)", () => {
  const imp = {
    id: "i1",
    links: [
      { pred_id: "slab", succ_id: "frame", type: "FS", lag_days: 0, lag_unit: "working" },
      { pred_id: "frame", succ_id: "roof", type: "SS", lag_days: 2, lag_unit: "working" },
      { pred_id: "roof", succ_id: "gone", type: "FS", lag_days: 0, lag_unit: "working" },
    ],
  } as unknown as ProgramImport;
  const live = (id: string) => id !== "gone";
  const links = fileLinks(imp, new Set(["slab"]), live);
  expect(links.map((l) => [l.link.pred_id, l.link.succ_id, l.text, l.onAccept])).toEqual([
    ["slab", "frame", "FS", true],
    ["frame", "roof", "SS+2d", false],
  ]);
  expect(fileLinks({ ...imp, links: null }, new Set(), live)).toEqual([]);
});

test("a link's type and lag in the typed format", () => {
  expect(linkText({ type: "FS", lag_days: 0, lag_unit: "working" })).toBe("FS");
  expect(linkText({ type: "FF", lag_days: -1, lag_unit: "working" })).toBe("FF-1d");
  expect(linkText({ type: "FS", lag_days: 3, lag_unit: "calendar" })).toBe("FS+3cd");
});
