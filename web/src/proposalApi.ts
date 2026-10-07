import { request } from "./api";

export interface ProposalRecord {
  key: string; kind: string; label: string; action?: string; state: string;
  target_system_id?: string; target_part_id?: string;
  critical: boolean; draft: boolean; unaccepted_triggers: boolean; inputs_changed: boolean;
  inputs_fingerprint: string; knowledge_version: string;
  reason: {
    record: string; interface?: string; rules?: string[];
    triggers: { work_item_id: string; action: string; system: string; part: string }[];
    determinants: { key: string; value: string; origin: string }[];
    signals: { id: string; state: string; sources?: unknown }[];
    other_side_existence?: string;
  };
  decision?: { decision: string; version: number; actor: string; decided_at: string; rationale: string; created_record_type?: string; created_record_id?: string };
}
export interface ProposalAssignment { package_id: string; work_item_id: string; stage_id: string; role: string }
const path = (project: string, key: string) => `/projects/${project}/proposals/${encodeURIComponent(key)}`;
export const proposalApi = {
  list: (project: string, all: boolean) => request<{ items: ProposalRecord[]; total: number }>("GET", `/projects/${project}/proposals${all ? "?show=all" : ""}`),
  accept: (project: string, proposal: ProposalRecord, assignment: Partial<ProposalAssignment>) => request("POST", `${path(project, proposal.key)}/accept`, { inputs_fingerprint: proposal.inputs_fingerprint, ...assignment }),
  dismiss: (project: string, proposal: ProposalRecord, rationale: string) => request("POST", `${path(project, proposal.key)}/dismiss`, { inputs_fingerprint: proposal.inputs_fingerprint, rationale }),
  undo: (project: string, key: string, version: number) => request("DELETE", `${path(project, key)}/decision`, { version }),
};
