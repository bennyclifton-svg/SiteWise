import { request } from "./api";

export interface DeliveryContent {
  kind: string; title: string; status: string; owner_text: string;
  baseline_date: string | null; target_date: string | null; forecast_date: string | null; actual_date: string | null; as_of: string | null;
  package_id: string; work_item_id: string; stage_id: string;
  details: Record<string, unknown>;
}
export interface DeliveryRecord extends DeliveryContent {
  owner_user_id?: string;
  id: string; version: number; origin: string; review_status: string; meaning: string; source_proposal_key?: string;
  provenance: { actor?: string; at?: string; last_edited_by?: string; last_edited_at?: string; proposal?: { kind: string; draft: boolean; label: string }; sources?: { filename?: string; excerpt?: string; page?: number }[] };
}
export interface DeliveryDependency { predecessor_id: string; successor_id: string; type: string; lag_days: number }
export const deliveryApi = {
  dependencies: (p: string) => request<{items: DeliveryDependency[]}>("GET", `/projects/${p}/delivery/dependencies`),
  dependency: (p: string, d: DeliveryDependency, version: number, remove = false) => request(remove ? "DELETE" : "POST", `/projects/${p}/delivery/dependencies`, {...d,version}),
  list: (project: string) => request<{ items: DeliveryRecord[] }>("GET", `/projects/${project}/delivery`),
  create: (project: string, body: DeliveryContent) => request<DeliveryRecord>("POST", `/projects/${project}/delivery`, body),
  patch: (project: string, item: string, version: number, body: Partial<DeliveryContent>) => request<DeliveryRecord>("PATCH", `/projects/${project}/delivery/${item}`, { ...body, version }),
};
