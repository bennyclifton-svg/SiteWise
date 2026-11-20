import { describe, expect, it } from "vitest";
import type { DeliveryRecord } from "./deliveryApi";
import { rfiHistory } from "./registerHistory";

const base = (over: Partial<DeliveryRecord>): DeliveryRecord => ({
  kind: "rfi", title: "Membrane", status: "answered", owner_text: "", baseline_date: null, target_date: "2026-11-27", forecast_date: null, actual_date: "2026-11-24", as_of: null,
  package_id: "", work_item_id: "", stage_id: "", details: {}, id: "r1", version: 4, origin: "user", review_status: "accepted_for_planning", meaning: "requirement",
  provenance: {}, ...over,
});

describe("rfiHistory", () => {
  it("lists raised, status changes, what went out, the reply as the answer, and possible answers, oldest first", () => {
    const row = base({
      details: { raised_on: "2026-11-20", answer_documents: ["doc-reply", "doc-other"], answer: "Laps 150 mm." },
      provenance: {
        at: "2026-11-20T00:10:00Z", initial_status: "draft",
        status_history: [{ from: "draft", status: "open", at: "2026-11-20T01:00:00Z" }, { from: "open", status: "answered", at: "2026-11-23T23:30:00Z" }],
      },
      thread: [
        { id: "m1", direction: "out", at: "2026-11-20T02:00:00Z", with: "eng@consult.test", number: 1, send_status: "sent", documents: 2 },
        { id: "m2", direction: "in", at: "2026-11-21T03:00:00Z", with: "Consult Eng", excerpt: "Use the sheet membrane.", documents: 1, body_document: "doc-reply" },
      ],
      readings: [{ id: "rd", document_id: "doc-maybe", document: "Site memo", decided_by: "rule", band: "amber", filed_on: "2026-11-22" },
        { id: "rd2", document_id: "doc-reply", document: "Re: Membrane", decided_by: "rule", band: "amber", filed_on: "2026-11-21" }],
    });
    const lines = rfiHistory(row, id => (id === "doc-other" ? "Sketch SK-01" : "Document"));
    expect(lines.map(l => [l.date, l.tone, l.text])).toEqual([
      ["2026-11-20", "note", "Raised · draft"],
      ["2026-11-20", "note", "Draft → Open"],
      ["2026-11-20", "out", "Sent to eng@consult.test"],
      ["2026-11-21", "answer", "Reply from Consult Eng · the answer"],
      ["2026-11-22", "check", "Possible answer: Site memo"],
      ["2026-11-24", "note", "Open → Answered"],
      ["2026-11-24", "answer", "Answered by Sketch SK-01"],
      ["2026-11-24", "answer", "Answer"],
    ]);
    expect(lines.find(l => l.kind === "sent")?.detail).toBe("2 documents · TR-001");
    expect(lines.find(l => l.kind === "reply")?.quote).toBe("Use the sheet membrane.");
  });

  it("with mail off: raised, status changes and the accepted document only", () => {
    const row = base({ status: "open", actual_date: null, details: { raised_on: "2026-11-20" }, provenance: { at: "2026-11-20T00:10:00Z", status_history: [{ from: "draft", status: "open", at: "2026-11-21T00:00:00Z" }] } });
    expect(rfiHistory(row, () => "").map(l => l.text)).toEqual(["Raised · draft", "Draft → Open"]);
  });
});
