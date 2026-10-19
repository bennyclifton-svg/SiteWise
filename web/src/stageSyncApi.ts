import { request } from "./api";

// Stage plan shared with Procurement and the Cost Plan (program plan 7.6):
// a package's or an RFP's differences from the program's stages, and
// Update or Keep. Program is where stages are edited; these only follow.

export type StageChangeKind = "renamed" | "added" | "removed";
export interface StageSyncChange { kind: StageChangeKind; program_stage_id: string; old_label?: string; new_label?: string; action: string; held?: boolean }
export interface PickerStage { program_stage_id: string; label: string; catalogue_stage_id: string; ticked: boolean; package_stage_id?: string; package_stage_version?: number }
export interface StageRow { program_stage_id: string; label: string; removed_from_program?: boolean }
export interface KeptStage { id: string; label: string; version: number; removed_from_program?: string }
export interface StageSync {
  document: string;
  program_exists: boolean;
  revision: number;
  built_from: number;
  acknowledged: number;
  offer: boolean;
  changes: StageSyncChange[];
  reordered: boolean;
  proposals_flagged: number;
  awarded: boolean;
  package_id: string;
  stages: PickerStage[];
  kept: KeptStage[];
  report_id?: string;
  version_id?: string;
  rows: StageRow[];
}
export type StageSyncChoice = "update" | "keep";

export const stageSyncApi = {
  pkg: (project: string, pkg: string) => request<StageSync>("GET", `/projects/${project}/packages/${pkg}/stage-sync`),
  syncPkg: (project: string, pkg: string, choice: StageSyncChoice, revision: number) =>
    request<StageSync>("POST", `/projects/${project}/packages/${pkg}/stage-sync`, { choice, revision }),
  report: (report: string) => request<StageSync>("GET", `/reports/${report}/stage-sync`),
  syncReport: (report: string, choice: StageSyncChoice, revision: number) => request<StageSync>("POST", `/reports/${report}/stage-sync`, { choice, revision }),
  /** Creates the program from Procurement when the project has none. Idempotent. */
  prepare: (project: string) => request<unknown>("POST", `/projects/${project}/program/prepare`, { surface: "procurement" }),
  tick: (project: string, pkg: string, programStageId: string) => request<unknown>("POST", `/projects/${project}/packages/${pkg}/stages`, { program_stage_id: programStageId }),
  untick: (project: string, pkg: string, stage: string, version: number) => request<unknown>("PATCH", `/projects/${project}/packages/${pkg}/stages/${stage}`, { retired: true, version }),
};

/** One difference in words, by stage name. */
export function changeWords(c: StageSyncChange): string {
  switch (c.kind) {
    case "renamed": return `${c.old_label} renamed ${c.new_label}`;
    case "added": return `${c.new_label} added`;
    default: return c.held ? `${c.old_label} removed from the program; kept here because money or records are attached` : `${c.old_label} removed`;
  }
}
