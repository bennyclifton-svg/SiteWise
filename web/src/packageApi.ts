import { request } from "./api";
export interface PackageStage { id: string; label: string; novation_phase: string }
/** no; option: the delivery phase is priced and the client may novate later; yes: the client has elected. */
export type Novation = "no" | "option" | "yes";
/** Who engages a consultant: the client directly, or the lead design consultant as a subconsultant. Absent: not decided. */
export type EngagedBy = "client" | "lead_consultant";
export interface PackageRecord { id: string; title: string; kind: string; works_scope?: string; lifecycle_status: string; novation: Novation; engaged_by?: EngagedBy; version: number; stages: PackageStage[]; discipline_id?: string; retired_at?: string }
export interface ScopeRecord { id: string; item_kind: string; user_text?: string; clause_id?: string; clause_version?: number; work_item_id?: string; role?: string; stage_id?: string; inclusion: string; deliverable?: string; version: number; provisional: boolean }
export interface ScopeWork { id: string; title: string; action: string; inclusion: string; is_group: boolean; review_status: string }
export interface ScopeInput { item_kind: string; user_text: string; work_item_id?: string; role?: string; stage_id?: string; inclusion: string; deliverable: string; interface_ids: string[] }
export interface PackageSuggestion { key: string; kind: string; title: string; novation: Novation; draft: boolean }
export interface ProposedWorks { key: string; title: string; package_id?: string; lines: { work_item_id: string; role: string; text: string }[] }
export interface PackageGap { work_item_id: string; state: string; role?: string }
export interface PackageOverview { items: PackageRecord[]; suggestions: PackageSuggestion[]; route: { value: string; builder_designs: boolean; confirm: boolean }; proposed_works: ProposedWorks | null; gaps: PackageGap[] }
export const packageApi = {
  list: (project: string) => request<PackageOverview>("GET", `/projects/${project}/packages`),
  accept: (project: string, key: string) => request<PackageRecord>("POST", `/projects/${project}/packages/proposed/accept`, { key }),
  create: (project: string, title: string, kind: string, worksScope: string, novation: Novation) => request<PackageRecord>("POST", `/projects/${project}/packages`, { title, kind, works_scope: kind === "works" ? worksScope : "", novation: kind === "services" ? novation : "no" }),
  // "" clears the answer back to not decided.
  engagedBy: (project: string, pkg: string, version: number, engagedBy: EngagedBy | "") => request<PackageRecord>("PATCH", `/projects/${project}/packages/${pkg}`, { version, engaged_by: engagedBy }),
  works: (project: string) => request<{ items: ScopeWork[] }>("GET", `/projects/${project}/works`),
  scope: (project: string, pkg: string) => request<{ items: ScopeRecord[] }>("GET", `/projects/${project}/packages/${pkg}/scope`),
  addScope: (project: string, pkg: string, input: ScopeInput) => request<ScopeRecord>("POST", `/projects/${project}/packages/${pkg}/scope`, input),
  patchScope: (project: string, pkg: string, id: string, version: number, patch: Partial<ScopeInput>) => request<ScopeRecord>("PATCH", `/projects/${project}/packages/${pkg}/scope/${id}`, { ...patch, version }),
};
