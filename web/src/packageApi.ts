import { request } from "./api";
export interface PackageStage { id: string; label: string; novation_phase: string }
export interface PackageRecord { id: string; title: string; kind: string; works_scope?: string; lifecycle_status: string; novation: boolean; version: number; stages: PackageStage[] }
export interface ScopeRecord { id: string; item_kind: string; user_text?: string; clause_id?: string; clause_version?: number; work_item_id?: string; role?: string; stage_id?: string; inclusion: string; deliverable?: string; version: number; provisional: boolean }
export interface ScopeWork { id: string; title: string; action: string; inclusion: string; is_group: boolean; review_status: string }
export interface ScopeInput { item_kind: string; user_text: string; work_item_id?: string; role?: string; stage_id?: string; inclusion: string; deliverable: string; interface_ids: string[] }
export const packageApi = {
  list: (project: string) => request<{ items: PackageRecord[] }>("GET", `/projects/${project}/packages`),
  create: (project: string, title: string, kind: string, worksScope: string, novation: boolean) => request<PackageRecord>("POST", `/projects/${project}/packages`, { title, kind, works_scope: kind === "works" ? worksScope : "", novation: kind === "services" && novation }),
  works: (project: string) => request<{ items: ScopeWork[] }>("GET", `/projects/${project}/works`),
  scope: (project: string, pkg: string) => request<{ items: ScopeRecord[] }>("GET", `/projects/${project}/packages/${pkg}/scope`),
  addScope: (project: string, pkg: string, input: ScopeInput) => request<ScopeRecord>("POST", `/projects/${project}/packages/${pkg}/scope`, input),
  patchScope: (project: string, pkg: string, id: string, version: number, patch: Partial<ScopeInput>) => request<ScopeRecord>("PATCH", `/projects/${project}/packages/${pkg}/scope/${id}`, { ...patch, version }),
};
