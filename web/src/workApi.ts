import { request } from "./api";
import type { Profile } from "./profileApi";

export interface WorkTarget { text?: string; values?: { key: string; value: string | number | boolean; unit?: string }[]; clause_refs?: { id: string; version: number }[] }
export interface WorkRecord {
  id: string; part_id: string; system_id: string; title: string; action: string;
  inclusion: string; is_group: boolean; review_status: string; origin: string; version: number;
  layout_change?: "unknown" | "yes" | "no"; existing_condition?: string; existing_condition_note?: string; target: WorkTarget;
  quantity?: string; unit?: string; deprecated?: boolean; source_proposal_key?: string;
  provenance: { actor?: string; last_edited_by?: string; last_edited_at?: string; rationale?: string; sources?: unknown };
}
export interface WorkCorrection { layout_change?: "unknown" | "yes" | "no"; version: number; title: string; action: string; inclusion: string; existing_condition_note: string; target: WorkTarget; quantity: string | null; unit?: string }
export const workApi = {
  split: (p: string, id: string, version: number, children: { title: string; part_id: string; action: string }[]) => request<WorkRecord[]>("POST", `/projects/${p}/works/${id}/split`, {version, children}),
  retire: (p: string, id: string, version: number) => request<{retired: boolean}>("POST", `/projects/${p}/works/${id}/retire`, {version}),
  list: (project: string) => request<{ items: WorkRecord[] }>("GET", `/projects/${project}/works`),
  profile: (project: string) => request<Profile>("GET", `/projects/${project}/profile`),
  patch: (project: string, item: string, correction: WorkCorrection) => request<WorkRecord>("PATCH", `/projects/${project}/works/${item}`, correction),
};
