import { describe, expect, it } from "vitest";
import { findTargets, mergeRun, type ProgressDoc, type RunDoc } from "./profileRun";

const doc = (id: string, over: Partial<ProgressDoc> = {}): ProgressDoc => ({
  document_id: id, filename: `${id}.pdf`, state: "queued", pass: "label", units: 96, labelled: 0, evidence: 0, ...over,
});

describe("mergeRun", () => {
  it("adds a document the moment its reading is queued", () => {
    expect(mergeRun([], [doc("ds")])).toEqual<RunDoc[]>([
      { id: "ds", filename: "ds.pdf", state: "queued", pass: "label", units: 96, labelled: 0, evidence: 0, ticks: 0 },
    ]);
  });

  it("ticks the label pass once the evidence pass is the one waiting or reading", () => {
    const reading = mergeRun([], [doc("ds", { state: "reading", labelled: 41 })]);
    expect(reading[0]).toMatchObject({ state: "reading", labelled: 41, ticks: 0 });
    const next = mergeRun(reading, [doc("ds", { state: "queued", pass: "evidence", labelled: 96 })]);
    expect(next[0]).toMatchObject({ state: "queued", pass: "evidence", ticks: 1 });
  });

  it("marks a document done when it leaves the list, its sheet full", () => {
    const before = mergeRun([], [doc("ds", { state: "reading", pass: "evidence", labelled: 90, evidence: 40 }), doc("sp")]);
    const after = mergeRun(before, [doc("sp", { state: "reading" })]);
    expect(after.map((d) => d.id)).toEqual(["ds", "sp"]);
    expect(after[0]).toMatchObject({ state: "done", ticks: 2, labelled: 96, evidence: 96 });
    expect(after[1]).toMatchObject({ state: "reading" });
  });

  it("keeps a failed pass failed until the server says otherwise", () => {
    const failed = mergeRun([], [doc("gt", { state: "failed", pass: "evidence" })]);
    expect(failed[0]).toMatchObject({ state: "failed", pass: "evidence", ticks: 1 });
  });
});

describe("findTargets", () => {
  it("lights card fields from the label pass and systems from the evidence pass", () => {
    expect(findTargets({ stage: "label", keys: ["hdr.scale.storeys", "proj.zone", "source.scope", "implied.x"], systems: ["substructure"] }))
      .toEqual({ fields: ["hdr.scale.storeys", "proj.zone"], leaves: [], entries: ["substructure"] });
    expect(findTargets({ stage: "evidence", keys: ["sys.substructure.footings.presence", "sig.flood", "hdr.scale.storeys"], systems: ["substructure.footings", "fire-active"] }))
      .toEqual({ fields: [], leaves: ["substructure.footings"], entries: ["substructure", "fire-active"] });
  });
});
