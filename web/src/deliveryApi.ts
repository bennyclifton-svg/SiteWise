import { request } from "./api";

// The register (docs/plans/2026-11-17-rfis-review-comments.md). Reads come
// from GET /delivery; every write goes through POST /ops in area delivery,
// so each screen action has one Undo.

export interface DeliveryContent {
  kind: string; title: string; status: string; owner_text: string;
  baseline_date: string | null; target_date: string | null; forecast_date: string | null; actual_date: string | null; as_of: string | null;
  package_id: string; work_item_id: string; stage_id: string; system_id?: string;
  details: Record<string, unknown>;
}
export interface Reading { id: string; document_id: string; document: string; decided_by: string; band: string; basis?: string; filed_on?: string }
export interface DeliveryRecord extends DeliveryContent {
  owner_user_id?: string;
  id: string; version: number; origin: string; review_status: string; meaning: string; source_proposal_key?: string;
  number?: number; with?: string; late_days?: number; readings?: Reading[];
  provenance: { actor?: string; at?: string; last_edited_by?: string; last_edited_at?: string; proposal?: { kind: string; draft: boolean; label: string };
    sources?: { document_id?: string; filename?: string; excerpt?: string; page?: number }[];
    status_history?: { from: string; status: string; at: string }[] };
}
export interface RegisterKind { kind: string; rows: number; open: number; to_check: number }
export interface AlsoOpen { system: string; label: string; spec_unknowns: number; exclusions: number; exclusion_packages: string[]; main_works_exclusions: number; main_works_package?: string }
export interface RegisterView {
  items: DeliveryRecord[]; kinds: RegisterKind[];
  response_days: number; response_typical: boolean; calendar: "program" | "weekdays"; today: string;
  systems: { id: string; label: string }[]; also_open?: AlsoOpen;
}
export interface OpCall { op: string; fields: Record<string, unknown> }
export interface OpsReply { change_id?: string; undo: { kind: string } }

export const deliveryApi = {
  view: (project: string, system = "") =>
    request<RegisterView>("GET", `/projects/${project}/delivery${system ? `?system=${encodeURIComponent(system)}` : ""}`),
  ops: (project: string, ops: OpCall[]) => request<OpsReply>("POST", `/projects/${project}/ops`, { area: "delivery", ops }),
  undo: (project: string, change: string) => request<{ undone: boolean }>("POST", `/projects/${project}/changes/${change}/undo`, {}),
};

/** "RFI-007" */
export const rfiNumber = (n?: number) => (n ? `RFI-${String(n).padStart(3, "0")}` : "");

/** Today in Sydney, YYYY-MM-DD: the raised date of an RFI typed in the grey row. */
export function sydneyToday(): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Australia/Sydney", year: "numeric", month: "2-digit", day: "2-digit" }).format(new Date());
}
