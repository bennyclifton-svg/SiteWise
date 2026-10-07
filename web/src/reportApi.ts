import { request } from "./api";

export interface SavedReport { id: string; title: string; kind: string; package_id: string; current_draft_version_id: string }
export interface ReportReference { citation_id: string; label: "E" | "U" | "C" | "A"; anchor_id: string; basis: { text: string; source: Record<string, unknown>; material_assumption: boolean; protected_edit: boolean } }
export interface ReportBlock { generated_text?: string; id: string; label: string; text: string; provisional: boolean; edited: boolean; conflict: boolean; source_missing: boolean; citation_ids: string[] }
export interface ReportDraft { id: string; version: number; status: string; reporting_date: string; sections: { id: string; title: string; blocks: ReportBlock[] }[]; references: ReportReference[]; material_assumptions: ReportReference[] }
export interface ReportView { report: SavedReport; draft: ReportDraft | null; stale: string[] }
export interface ReportPackage { id: string; title: string; kind: string; retired_at?: string }
export const reportApi = {
  list: (project: string) => request<SavedReport[]>("GET", `/projects/${project}/reports`),
  packages: (project: string) => request<{ items: ReportPackage[] }>("GET", `/projects/${project}/packages`),
  create: (project: string, packageId: string) => request<SavedReport>("POST", `/projects/${project}/reports`, { kind: "rfp", package_id: packageId }),
  read: (id: string) => request<ReportView>("GET", `/reports/${id}`),
  refresh: (id: string, useLastCompleted = false) => request<ReportDraft>("POST", `/reports/${id}/draft`, { use_last_completed: useLastCompleted }),
  edit: (id: string, target: string, text: string, version: number) => request<ReportDraft>("PUT", `/reports/${id}/edits/${encodeURIComponent(target)}`, { text, version }),
	reset: (id: string, target: string, version: number) => request<ReportDraft>("DELETE", `/reports/${id}/edits/${encodeURIComponent(target)}`, { version }),
};
