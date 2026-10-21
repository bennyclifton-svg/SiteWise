import { request } from "./api";
export interface CostSettings { currency: string; tax_basis: string; tax_rate: string | null; price_date: string | null; coverage: string; funding_target: string | null; funding_target_provenance: Record<string, unknown> }
export interface CostContent { system_id?: string; parent_item_id: string; code: string; label: string; line_kind: string; category: string; work_item_id: string; package_id: string; package_stage_id: string; posting: boolean; quantity: string | null; unit: string; rate: string | null; rate_basis: string; excluded: boolean; origin: string; meaning: string; rationale: string }
export interface CostValue { amount: string | null; low: string | null; high: string | null; value_state: string; as_of: string | null; origin: string; meaning: string; rationale: string; version?: number; provenance?: { spread?: { budget: string; split: string } } }
/** The construction budget the profile states, and the split it is spread by. */
export interface ConstructionBudget { stated: { text: string; amount: string; gst: "ex" | "inc" | "not_stated"; converted: boolean }; known: boolean; origin: "document" | "user"; split: string; split_confidence: string; split_basis: string; no_split?: string }
/** Budget against the construction lines: difference is budget less allocated. */
export interface BudgetCheck { amount: string | null; allocated: string; unpriced: number; difference: string | null }
export interface CostItem extends CostContent { source_key?: string; resolved_package_id?: string; resolved_category?: string; scope_work_items?: CostScope[]; cost_item_id: string; version: number; system_id: string; part_id: string; values: Record<string, CostValue> }
export interface CostScope { id: string; system_id: string; title: string; part_id: string }
export interface CostSuggestion extends CostContent { source_key: string; parent_key: string; resolved_category: string; scope_work_items: CostScope[] }
export interface CostPlan extends CostSettings { totals?: CostTotals; suggestions?: CostSuggestion[]; dismissed?: CostSuggestion[]; systems?: {id: string; label: string}[]; id: string; baseline_version_id?: string; revision: number; version: number; status: string; items: CostItem[]; links: { package_scope_item_id: string; cost_item_id: string }[]; construction_budget?: ConstructionBudget }
export interface BenchmarkSuggestion { benchmark: { id: string; version: number; geography: string; quality: string; price_date: string; inclusions: string[]; exclusions: string[]; sources: unknown }; amount: string; limitations: string }
export interface CostTotal { amount: string | null; low: string | null; high: string | null; known_subtotal: string; unknown: number; complete: boolean; lines: number }
// budget_variance is budget − forecast (positive is under budget); remaining is
// forecast − claimed. Both are null when an input is unknown or a range.
export interface CostDerived { budget_variance: string | null; remaining: string | null }
export interface CostFigures extends CostDerived { forecast: CostValue; basis: string }
export interface CostTotals { sections: Record<string, Record<string, CostTotal>>; section_items: Record<string, string>; section_figures: Record<string, CostDerived>; overall: Record<string, CostTotal>; works: Record<string, CostTotal>; fees: Record<string, CostTotal>; project_wide: Record<string, CostTotal>; groups: Record<string, Record<string, CostTotal>>; variance: string | null; items: Record<string, CostFigures>; buckets: Record<string, CostDerived>; construction_budget?: BudgetCheck }
const path = (p: string) => `/projects/${p}/cost-plan`;
const pin = (p: CostPlan) => ({ plan_version_id: p.id || "", version: p.version || 0 });
export const costApi = {
  prepare: (p: string) => request<{plan: CostPlan; totals: CostTotals}>("POST", `${path(p)}/prepare`, {}),
  remove: (p: string, plan: CostPlan, id: string) => request<CostPlan>("DELETE", `${path(p)}/items/${id}`, pin(plan)),
  accept: (p: string, plan: CostPlan, key = "", budget?: string | null) => request<CostPlan>("POST", `${path(p)}/suggestions/${key ? `${encodeURIComponent(key)}/` : ""}accept`, {...pin(plan), budget}),
  dismiss: (p: string, plan: CostPlan, key: string, restore = false) => request<CostPlan>(restore ? "DELETE" : "POST", `${path(p)}/suggestions/${encodeURIComponent(key)}/dismiss`, pin(plan)),
  read: (p: string, v = "") => request<CostPlan>("GET", path(p) + (v ? `?version=${v}` : "")),
  totals: (p: string, by = "overall", v = "") => request<CostTotals>("GET", `${path(p)}/totals?by=${by}${v ? `&version=${v}` : ""}`),
  settings: (p: string, plan: CostPlan, settings: CostSettings) => request<CostPlan>("PUT", path(p), { ...pin(plan), ...settings }),
  item: (p: string, plan: CostPlan, content: CostContent, id = "", calculate_budget = false) => request<CostPlan>(id ? "PATCH" : "POST", `${path(p)}/items${id ? `/${id}` : ""}`, { ...pin(plan), ...content, calculate_budget }),
  value: (p: string, plan: CostPlan, id: string, metric: string, value: CostValue, calculate = false) => request<CostPlan>("PUT", `${path(p)}/items/${id}/values/${metric}`, { plan_version_id: plan.id, plan_version: plan.version, ...value, calculate }),
  baseline: (p: string, plan: CostPlan) => request<CostPlan>("POST", `${path(p)}/baseline`, pin(plan)),
  subdivide: (p: string, plan: CostPlan, item: CostItem, children: { content: CostContent; amount: string }[], metric: string, residual: string) => request<CostPlan>("POST", `${path(p)}/items/${item.cost_item_id}/subdivide`, { ...pin(plan), residual, children: children.map(c => ({ ...c.content, values: { [metric]: { amount: c.amount || null, low: null, high: null, value_state: c.amount ? "known" : "unknown", as_of: null, origin: "user", meaning: "allowance", rationale: "Explicit subdivision" } } })) }),
  benchmarks: (p: string, id: string, geography: string, quality: string) => request<{suggestions: BenchmarkSuggestion[]; message: string}>("GET", `${path(p)}/items/${id}/benchmarks?${new URLSearchParams({geography,quality})}`),
  applyBenchmark: (p: string, plan: CostPlan, id: string, suggestion: BenchmarkSuggestion, geography: string, quality: string) => request<CostPlan>("POST", `${path(p)}/items/${id}/benchmarks`, {...pin(plan),benchmark_id:suggestion.benchmark.id,benchmark_version:suggestion.benchmark.version,geography,quality,acknowledge_basis:true}),
  link: (p: string, plan: CostPlan, item: string, scope: string, remove: boolean) => request<CostPlan>("PUT", `${path(p)}/links`, { ...pin(plan), cost_item_id: item, package_scope_item_id: scope, remove }),
};
