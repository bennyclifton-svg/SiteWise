import { request } from "./api";
import type { PackageRecord } from "./packageApi";
import type { SavedReport } from "./reportApi";
import type { CostPlan } from "./costApi";

export const procurementApi = {
  addRow: (project: string, disciplineId: string) => request<PackageRecord>("POST", `/projects/${project}/procurement/rows`, { discipline_id: disciplineId }),
  /** Exactly one of package_id or proposal_key. Returns the existing report when the row already has one. */
  create: (project: string, from: { package_id: string } | { proposal_key: string }) => request<{ package: PackageRecord; report: SavedReport }>("POST", `/projects/${project}/procurement/create`, from),
  retire: (project: string, pkg: string, version: number, costPlan?: CostPlan) => request<PackageRecord>("PATCH", `/projects/${project}/packages/${pkg}`, { retired: true, version, ...(costPlan ? {remove_cost_lines: true, cost_plan_id: costPlan.id, cost_plan_version: costPlan.version} : {}) }),
};
