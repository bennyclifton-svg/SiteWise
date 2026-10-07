import { request } from "./api";

export interface SavedReport { id: string; title: string; kind: string; package_id: string; current_draft_version_id: string }
export interface ReportReference { citation_id: string; label: "E" | "U" | "C" | "A"; anchor_id: string; basis: { text: string; source: Record<string, unknown>; material_assumption: boolean; protected_edit: boolean } }
export interface ReportBlock { table?: { columns: string[]; rows: string[][] }; generated_text?: string; id: string; label: string; text: string; provisional: boolean; edited: boolean; conflict: boolean; source_missing: boolean; citation_ids: string[] }
export interface ReportDraft { id: string; version: number; status: string; reporting_date: string; sections: { id: string; title: string; blocks: ReportBlock[] }[]; references: ReportReference[]; material_assumptions: ReportReference[] }
export interface ReportView { report: SavedReport; draft: ReportDraft | null; stale: string[]; issues: { id: string; number: number; reporting_date: string; snapshot_sha256: string }[]; changes_since_issue: { target_id: string; kind: string; before?: string; after?: string }[] }
export interface ReportPackage { id: string; title: string; kind: string; retired_at?: string }
export const reportApi = {
  list: (project: string) => request<SavedReport[]>("GET", `/projects/${project}/reports`),
  packages: (project: string) => request<{ items: ReportPackage[] }>("GET", `/projects/${project}/packages`),
  create: (project: string, packageId: string, kind = "rfp") => request<SavedReport>("POST", `/projects/${project}/reports`, { kind, package_id: kind === "pmp" ? "" : packageId }),
  issue: (id: string, options: { version: number; reporting_date: string; budget_disclosed: boolean; accept_stale: boolean; stale_reason: string }) => request("POST", `/reports/${id}/issue`, options),
  read: (id: string) => request<ReportView>("GET", `/reports/${id}`),
  refresh: (id: string, useLastCompleted = false) => request<ReportDraft>("POST", `/reports/${id}/draft`, { use_last_completed: useLastCompleted }),
  edit: (id: string, target: string, text: string, version: number) => request<ReportDraft>("PUT", `/reports/${id}/edits/${encodeURIComponent(target)}`, { text, version }),
	reset: (id: string, target: string, version: number) => request<ReportDraft>("DELETE", `/reports/${id}/edits/${encodeURIComponent(target)}`, { version }),
};
