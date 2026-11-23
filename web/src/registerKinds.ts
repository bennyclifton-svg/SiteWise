// What the register's list and its sheet share: each kind's words, its
// statuses, which count as closed, and a consent condition's status in
// words. Nothing here is computed from a date: Go gives those.

import { GATES, type ConditionDetails, type DeliveryRecord } from "./deliveryApi";

export const KIND: Record<string, [string, string]> = {
  rfi: ["RFI", "RFIs"], eot_claim: ["EOT claim", "EOT claims"], delay_notice: ["Notice of delay", "Notices of delay"], letter: ["Letter", "Letters"],
  notice: ["Notice", "Notices"], action: ["Action", "Actions"], risk: ["Risk", "Risks"], issue: ["Issue", "Issues"],
  decision: ["Decision", "Decisions"], approval: ["Approval", "Approvals"], milestone: ["Hold point", "Hold points"],
  condition: ["Condition", "Conditions"],
};
export const STATUSES: Record<string, string[]> = {
  rfi: ["draft", "open", "answered", "closed", "withdrawn"], notice: ["open", "closed", "withdrawn"],
  eot_claim: ["open", "assessed", "closed", "withdrawn"], delay_notice: ["open", "closed", "withdrawn"], letter: ["open", "answered", "closed", "withdrawn"],
  action: ["not_started", "in_progress", "blocked", "complete", "cancelled"], risk: ["open", "mitigating", "closed"],
  issue: ["open", "in_progress", "resolved", "closed"], decision: ["open", "decided", "superseded"],
  approval: ["not_submitted", "submitted", "approved", "rejected", "withdrawn"], milestone: ["planned", "achieved", "cancelled"],
  condition: ["open", "evidence_filed", "satisfied", "not_applicable"],
};
export const CLOSED = new Set(["closed", "withdrawn", "complete", "cancelled", "resolved", "decided", "superseded", "approved", "rejected", "achieved", "satisfied", "not_applicable"]);
export const words = (s: string) => s.replaceAll("_", " ");
/** The kinds the PM records as they arrive, and their sheet (correspondence plan, section 18). */
export const RECEIVED = new Set(["eot_claim", "delay_notice", "letter"]);
/** Kinds that take Close, Withdraw and Reopen from the menu. */
export const OWED = new Set(["rfi", "notice", "eot_claim", "delay_notice", "letter"]);
/** An EOT result in words: "2 of 5 days granted". */
export function resultWords(r: { outcome: string; days_granted?: number | null } | null | undefined, claimed?: number | null): string {
  if (!r) return "";
  const of = claimed != null ? ` of ${claimed}` : "";
  if (r.outcome === "rejected") return "Rejected";
  if (r.outcome === "more_information") return "More information asked for";
  const n = r.days_granted ?? 0;
  return `${n}${of} ${n === 1 && !of ? "day" : "days"} granted`;
}
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
