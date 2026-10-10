import { request } from "./api";

export interface SavedReport { id: string; title: string; kind: string; package_id: string; current_draft_version_id: string }
export interface ReportReference { citation_id: string; label: "E" | "U" | "C" | "A"; anchor_id: string; basis: { text: string; source: Record<string, unknown>; material_assumption: boolean; protected_edit: boolean } }
export interface ReportBlock { table?: { columns: string[]; rows: string[][]; display?: string }; generated_text?: string; id: string; label: string; text: string; provisional: boolean; edited: boolean; conflict: boolean; source_missing: boolean; citation_ids: string[] }
export interface ReportDraft { id: string; version: number; status: string; reporting_date: string; sections: { id: string; title: string; internal?: boolean; blocks: ReportBlock[] }[]; references: ReportReference[]; material_assumptions: ReportReference[] }
export interface ReportView { rfp?: RfpControls; rfp_suggestion?: RfpSuggestion; report: SavedReport; draft: ReportDraft | null; stale: string[]; issues: { id: string; number: number; reporting_date: string; snapshot_sha256: string }[]; changes_since_issue: { target_id: string; kind: string; before?: string; after?: string }[]; page_fit?: PageFit }
export interface PageFit { pages: number; limit: number; target: number; fits: boolean; error?: string }
export interface ReportPackage { id: string; title: string; kind: string; retired_at?: string; discipline_id?: string; version: number }
// Consultant RFP editor controls (internal/rfp/controls.go).
export interface RfpCheck { id: string; label: string; text: string; blocking: boolean; control?: string; changes?: string[] }
export interface RfpChoice { value: string; label: string }
export type RfpControlKind = "choice" | "tristate" | "parts" | "number" | "text" | "date" | "money" | "stage" | "element" | "count" | "position" | "weight";
export interface RfpControl { key: string; label: string; kind: RfpControlKind; value: unknown; options?: RfpChoice[]; source: string; why?: string; help?: string; locked?: boolean; set: boolean; party?: string; owner?: string; home?: RfpHome }
/** Where a fact: control saves (detail, header, site, determinant), or where it is changed instead (locked). */
export type RfpHome = "detail" | "header" | "site" | "determinant" | "cost_plan" | "link" | "derived";
export interface RfpGroup { id: string; title: string; controls: RfpControl[] }
export interface RfpControls {
  discipline: string; title: string; status: string;
  scale: { value: string; source: string; why: string; set: boolean };
  boundary: { start: string; end: string; source: string; confirmed: boolean; set: boolean; stages: RfpChoice[] };
  decisions: RfpCheck[]; checks: RfpCheck[]; groups: RfpGroup[];
}
export interface RfpSuggestion { discipline: string; title: string; discipline_id: string }
export interface RfpDiscipline { id: string; title: string; status: string; discipline_ids: string[]; title_aliases: string[] }
export const reportApi = {
  list: (project: string) => request<SavedReport[]>("GET", `/projects/${project}/reports`),
  packages: (project: string) => request<{ items: ReportPackage[] }>("GET", `/projects/${project}/packages`),
  create: (project: string, packageId: string, kind = "rfp") => request<SavedReport>("POST", `/projects/${project}/reports`, { kind, package_id: kind === "pmp" ? "" : packageId }),
  issue: (id: string, options: { version: number; reporting_date: string; budget_disclosed: boolean; accept_stale: boolean; stale_reason: string }) => request("POST", `/reports/${id}/issue`, options),
  read: (id: string) => request<ReportView>("GET", `/reports/${id}`),
  refresh: (id: string, useLastCompleted = false) => request<ReportDraft>("POST", `/reports/${id}/draft`, { use_last_completed: useLastCompleted }),
  edit: (id: string, target: string, text: string, version: number) => request<ReportDraft>("PUT", `/reports/${id}/edits/${encodeURIComponent(target)}`, { text, version }),
	reset: (id: string, target: string, version: number) => request<ReportDraft>("DELETE", `/reports/${id}/edits/${encodeURIComponent(target)}`, { version }),
  setting: (id: string, key: string, value: unknown, version: number) => request<ReportView>("PUT", `/reports/${id}/settings/${encodeURIComponent(key)}`, { value, version }),
  clearSetting: (id: string, key: string, version: number) => request<ReportView>("DELETE", `/reports/${id}/settings/${encodeURIComponent(key)}`, { version }),
  /** Saves a project fact to Profile and returns the refreshed view; null clears the user's word. */
  fact: (id: string, fact: string, value: string | null, version: number) => request<ReportView>("PUT", `/reports/${id}/facts/${encodeURIComponent(fact)}`, { value, version }),
  disciplines: () => request<{ items: RfpDiscipline[] }>("GET", "/rfp/disciplines"),
  confirmDiscipline: (project: string, pkg: string, disciplineId: string, version: number) => request<ReportPackage>("PATCH", `/projects/${project}/packages/${pkg}`, { discipline_id: disciplineId, version }),
};
