import { describe, expect, it } from "vitest";
import { diffSnapshots, receiptOf, snapshotOf, type Snapshot } from "./profileDiff";
import type { Profile, ProfileField, SystemRow } from "./profileApi";

const field = (key: string, value: string, band: ProfileField["band"], sources = 0): ProfileField => ({
  key, label: key, part_id: "whole", kind: "text", value, band,
  sources: Array.from({ length: sources }, (_, i) => ({ document_id: `d${i}` })), alternatives: [],
});

const system = (leaf: string, inScope: boolean, extra: Partial<SystemRow> = {}): SystemRow => ({
  leaf, label: leaf, part_id: "whole", in_scope: inScope, shown_by_default: true,
  presence: { value: "", band: "", sources: [], alternatives: [] },
  provider: { value: "", band: "", sources: [], alternatives: [] },
  note: { value: "", band: "", sources: [], alternatives: [] },
  ...extra,
});

function profile(fields: ProfileField[], systems: SystemRow[] = [], revision = 1): Profile {
  return {
    project_id: "p", built_at: null, revision, pending_documents: 0, active_documents: 0, unread_documents: 0,
    read_documents: 0, completed_documents: 0, skipped_documents: 0, skipped_kind: "", failed_documents: 0,
    payment_required: false, thresholds: { version: "", provisional: true, applied: true }, parts: [],
    header: fields, facts: [], details: [], site_details: [], compliance: [],
    systems: [{ id: "sub", label: "Substructure", rows: systems }], presets: [],
  };
}

const diff = (a: Snapshot, b: Snapshot) => diffSnapshots(a, b);

describe("diffSnapshots", () => {
  it("finds a value filled by the reading", () => {
    const before = snapshotOf(profile([field("hdr.scale.storeys", "", "")]));
    const after = snapshotOf(profile([field("hdr.scale.storeys", "5", "amber", 1)], [], 2));
    expect(diff(before, after)).toEqual([
      { kind: "field", id: "hdr.scale.storeys|whole", key: "hdr.scale.storeys", part: "whole", how: "filled", was: "", now: "5", added: 1 },
    ]);
  });

  it("finds a typical value the documents replaced", () => {
    const before = snapshotOf(profile([field("hdr.cond.procurement_route", "construct_only", "suggested")]));
    const after = snapshotOf(profile([field("hdr.cond.procurement_route", "dc", "amber", 1)]));
    const [c] = diff(before, after);
    expect(c).toMatchObject({ kind: "field", how: "changed", was: "construct_only", now: "dc" });
  });

  it("leaves the user's own edit and reset unmarked", () => {
    const typed = diff(snapshotOf(profile([field("proj.zone", "", "")])), snapshotOf(profile([field("proj.zone", "R2", "user")])));
    const reset = diff(snapshotOf(profile([field("proj.zone", "R2", "user")])), snapshotOf(profile([field("proj.zone", "MU1", "amber", 1)])));
    expect(typed).toEqual([]);
    expect(reset).toEqual([]);
  });

  it("holds the user's value when documents speak to it again", () => {
    const before = snapshotOf(profile([field("proj.budget_stated", "6900000", "user")]));
    const after = snapshotOf(profile([field("proj.budget_stated", "6900000", "user", 2)]));
    expect(diff(before, after)).toEqual([
      { kind: "field", id: "proj.budget_stated|whole", key: "proj.budget_stated", part: "whole", how: "held", was: "6900000", now: "6900000", added: 2 },
    ]);
  });

  it("marks a value the documents now disagree on", () => {
    const before = snapshotOf(profile([field("hdr.scale.site_sqm", "562.8", "amber", 1)]));
    const after = snapshotOf(profile([field("hdr.scale.site_sqm", "562.8", "red", 2)]));
    expect(diff(before, after)[0]).toMatchObject({ how: "conflict", added: 1 });
  });

  it("counts new sources on an unchanged value", () => {
    const before = snapshotOf(profile([field("proj.lga", "Inner West", "amber", 1)]));
    const after = snapshotOf(profile([field("proj.lga", "Inner West", "amber", 3)]));
    expect(diff(before, after)).toEqual([
      { kind: "field", id: "proj.lga|whole", key: "proj.lga", part: "whole", how: "sources", was: "Inner West", now: "Inner West", added: 2 },
    ]);
  });

  it("finds systems entering, leaving and needing a check, but not the user's ticks", () => {
    const out = { value: "not_included", band: "amber" as const, sources: [{ document_id: "d" }], alternatives: [] };
    const before = snapshotOf(profile([], [
      system("sub.basement", false), system("sub.termite", true), system("sub.underpin", true, { scope_origin: "user" }), system("sub.piles", false),
    ]));
    const after = snapshotOf(profile([], [
      system("sub.basement", true, { scope_origin: "document" }), system("sub.termite", false, { scope_origin: "document" }),
      system("sub.underpin", true, { scope_origin: "user", presence: out }), system("sub.piles", true, { scope_origin: "user" }),
    ]));
    expect(diff(before, after)).toEqual([
      { kind: "scope", leaf: "sub.basement", entry: "sub", how: "in" },
      { kind: "scope", leaf: "sub.termite", entry: "sub", how: "out" },
      { kind: "scope", leaf: "sub.underpin", entry: "sub", how: "check" },
    ]);
  });

  it("reports nothing between equal snapshots", () => {
    const p = profile([field("proj.lga", "Inner West", "amber", 1)], [system("sub.basement", true)]);
    expect(diff(snapshotOf(p), snapshotOf(p))).toEqual([]);
  });
});

describe("receiptOf", () => {
  it("counts what an update did", () => {
    const changes = diff(
      snapshotOf(profile([field("a", "", ""), field("b", "x", "suggested"), field("c", "1", "amber", 1), field("d", "9", "user")],
        [system("sub.one", false), system("sub.two", true), system("sub.three", false)])),
      snapshotOf(profile([field("a", "5", "amber", 1), field("b", "y", "amber", 1), field("c", "1", "red", 2), field("d", "9", "user", 1)],
        [system("sub.one", true), system("sub.two", false), system("sub.three", true)])),
    );
    expect(receiptOf(changes)).toEqual({ filled: 1, changed: 1, check: 1, held: 1, added: 2, removed: 1 });
  });
});
