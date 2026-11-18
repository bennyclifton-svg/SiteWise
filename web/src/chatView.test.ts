import { describe, expect, it } from "vitest";
import { dialogOf, noteSelection, registerDialog, registerTable, screenSelection, showingLine, tableOf, whenReady, type ChatTable } from "./chatView";

function fake(place: ChatTable["place"], ids: string[]): ChatTable & { calls: string[] } {
  const calls: string[] = [];
  return {
    place, calls,
    select: x => { calls.push(`select ${x.join(",")}`); },
    only: x => { calls.push(`only ${x.join(",")}`); },
    clear: () => { calls.push("clear"); },
    open: id => { calls.push(`open ${id}`); },
    selected: () => ids,
  };
}

describe("chat view registry", () => {
  it("sends the newest selection on screen, capped at 200 with the full count", () => {
    const lines = Array.from({ length: 600 }, (_, i) => `line-${i}`);
    const offDocs = registerTable("documents", fake("documents", ["d1", "d2"]));
    const offCosts = registerTable("cost_lines", fake("costs", lines));
    noteSelection("documents");
    noteSelection("cost_lines");
    const sel = screenSelection("costs");
    expect(sel?.table).toBe("cost_lines");
    expect(sel?.ids.length).toBe(200);
    expect(sel?.count).toBe(600);
    // On another view the cost lines are hidden; the register beside it is not.
    expect(screenSelection("program")).toEqual({ table: "documents", ids: ["d1", "d2"], count: 2 });
    noteSelection("documents");
    expect(screenSelection("costs")?.table).toBe("documents");
    offDocs();
    offCosts();
    expect(screenSelection("costs")).toBeUndefined();
  });

  it("waits for a table that is not on screen yet, then hands it over", async () => {
    const waiting = whenReady(() => tableOf("program"), 1000);
    const table = fake("program", []);
    const off = registerTable("program", table);
    const got = await waiting;
    got?.only(["a", "b"]);
    expect(table.calls).toEqual(["only a,b"]);
    off();
    expect(await whenReady(() => tableOf("program"), 10)).toBeUndefined();
  });

  it("keeps a newer registration when an older one unmounts", () => {
    const newer = { open: () => undefined };
    const first = registerDialog("award:p1", { open: () => undefined });
    const second = registerDialog("award:p1", newer);
    first();
    expect(dialogOf("award:p1")).toBe(newer);
    second();
    expect(dialogOf("award:p1")).toBeUndefined();
  });

  it("says what a filter shows in plain words", () => {
    expect(showingLine("cost_lines", 1)).toBe("Showing 1 cost line chosen in chat.");
    expect(showingLine("program", 6)).toBe("Showing 6 Program rows chosen in chat.");
    // A named filter's words replace "chosen in chat".
    expect(showingLine("cost_lines", 6, "with no budget")).toBe("Showing 6 cost lines with no budget.");
    expect(showingLine("documents", 1, "the profile does not read")).toBe("Showing 1 document the profile does not read.");
  });
});
