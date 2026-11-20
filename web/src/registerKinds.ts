// What the register's list and its sheet share: each kind's words, its
// statuses, which count as closed, and a consent condition's status in
// words. Nothing here is computed from a date: Go gives those.

import { GATES, type ConditionDetails, type DeliveryRecord } from "./deliveryApi";

export const KIND: Record<string, [string, string]> = {
  rfi: ["RFI", "RFIs"], notice: ["Notice", "Notices"], action: ["Action", "Actions"], risk: ["Risk", "Risks"], issue: ["Issue", "Issues"],
  decision: ["Decision", "Decisions"], approval: ["Approval", "Approvals"], milestone: ["Hold point", "Hold points"],
  condition: ["Condition", "Conditions"],
};
export const STATUSES: Record<string, string[]> = {
  rfi: ["draft", "open", "answered", "closed", "withdrawn"], notice: ["open", "closed", "withdrawn"],
  action: ["not_started", "in_progress", "blocked", "complete", "cancelled"], risk: ["open", "mitigating", "closed"],
  issue: ["open", "in_progress", "resolved", "closed"], decision: ["open", "decided", "superseded"],
  approval: ["not_submitted", "submitted", "approved", "rejected", "withdrawn"], milestone: ["planned", "achieved", "cancelled"],
  condition: ["open", "evidence_filed", "satisfied", "not_applicable"],
};
export const CLOSED = new Set(["closed", "withdrawn", "complete", "cancelled", "resolved", "decided", "superseded", "approved", "rejected", "achieved", "satisfied", "not_applicable"]);
export const words = (s: string) => s.replaceAll("_", " ");
/** "Open", "Not started": a status as the chip shows it. */
export const statusWords = (s: string) => { const w = words(s); return w.charAt(0).toUpperCase() + w.slice(1); };
export const GATE_WORDS: Record<string, string> = Object.fromEntries(GATES);
const GATE_RANK: Record<string, number> = Object.fromEntries(GATES.map(([g], i) => [g, i]));
export const cond = (r: DeliveryRecord) => r.details as ConditionDetails;

/** A condition's status in words: what is needed next. */
export function conditionStatus(r: DeliveryRecord): string {
  const d = cond(r);
  if (r.status === "open" && (r.readings ?? []).length > 0) return "Possible evidence · Link";
  if (r.status === "open") return d.evidence_needed ? "Open · evidence needed" : "Open";
  return ({ evidence_filed: "Evidence filed", satisfied: "Satisfied", not_applicable: "Not applicable" } as Record<string, string>)[r.status] ?? words(r.status);
}

/** Conditions in gate order, then by their printed number. */
export const conditionOrder = (a: DeliveryRecord, b: DeliveryRecord) =>
  (GATE_RANK[cond(a).stage ?? ""] ?? 99) - (GATE_RANK[cond(b).stage ?? ""] ?? 99) || (cond(a).number ?? "").localeCompare(cond(b).number ?? "", undefined, { numeric: true });

/** An operation's refusal in its own words. */
export function opMessage(raw: string): string {
  try { const m = (JSON.parse(raw) as { message?: string }).message; if (m) return m; } catch { /* plain text */ }
  return raw;
}
