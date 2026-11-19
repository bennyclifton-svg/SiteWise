import { ApiError, request } from "./api";

// Money is an exact decimal string from the server; null is unknown. The
// browser never sums it.
export type Money = string | null;
export type InvoiceBand = "green" | "amber" | "blank" | "grey";
export interface InvoiceField { value: string; band: InvoiceBand; decided_by: "rule" | "jev" | "user"; confidence?: number }
export interface InvoiceFlag { code: string; reason: string; info?: boolean }
export interface InvoiceRow { share: number; shares: number; line_id: string; line: string; ex_gst: Money; gst: Money; total: Money; band: InvoiceBand; decided_by: string }
export interface InvoiceAllocation { id: string; line_id: string; line: string; amount: Money; decided_by: string; band: InvoiceBand }
export type InvoiceStatus = "reading" | "filed" | "needs_review" | "not_read" | "void" | "superseded";
export interface Invoice {
  id: string; filename: string; status: InvoiceStatus; not_read_reason?: string; kind: string; number: string; date: string; period: string;
  supplier_id: string; supplier: string; abn: string; currency: string; ex_gst: Money; gst: Money; total: Money; invoice_count: number;
  superseded_by?: string; duplicate_ok: boolean; counted: boolean; created_at: string; flags: InvoiceFlag[];
  fields: Record<string, InvoiceField>; allocations: InvoiceAllocation[] | null; rows: InvoiceRow[] | null;
}
export interface LedgerLine { id: string; label: string; section: string; category: string; archived: boolean; budget: Money; invoiced: Money; remaining: Money; percent_spent: string; over_budget: boolean; invoices: number; contract_remaining: Money }
export interface LedgerSummary { lines: LedgerLine[]; unallocated: LedgerLine; invoiced: Money; budget: Money; budget_unknown: number; remaining: Money; percent_spent: string; counted: number; excluded: number; comparable: boolean; sections: Record<string, Money>; sections_contract_remaining: Record<string, Money>; contract_remaining: Money }
export interface LedgerPackage { id: string; title: string; kind: string; discipline: string; abn: string }
export interface Ledger { groups?: Record<string, LedgerLine & { line_ids: string[] }>; project: string; tax_basis: string; summary: LedgerSummary; invoices: Invoice[]; packages: LedgerPackage[]; lines: LedgerLine[]; contracts?: Contract[]; variations?: Variation[]; claims?: Claim[] }
/** A live contract (an award) as the Variations and claims tab shows it; every figure is the server's. */
export interface Contract {
  award_id: string; package_id: string; title: string; firm: string; claims: boolean;
  lines: { id: string; label: string; amount: Money }[];
  contract_sum: Money; approved_variations: Money; contract_sum_to_date: Money; certified_to_date: Money; retention_held: Money;
  schedule_days: number | null; schedule_basis: "business" | "calendar"; payment_days: number | null; payment_basis: "business" | "calendar";
}
export type VariationStatus = "proposed" | "submitted" | "approved" | "rejected" | "withdrawn";
export interface Variation {
  id: string; package_id: string; line_id: string; line: string; number: string; title: string; details: string; status: VariationStatus;
  claimed: Money; estimate: Money; approved: Money; forecast: Money; forecast_basis: "" | "estimate" | "claimed" | "unknown";
  time_claimed_days: number | null; submitted_on: string; decided_on: string; documents: { id: string; name: string }[]; version: number;
}
export interface InvoiceFilter { supplier: string; month: string; line: string; status: string }
export interface Uploaded { id: string; status: InvoiceStatus; created: boolean }

const base = (p: string) => `/projects/${p}/invoices`;

/** A date Go worked out, with how: never settled while to_check is true. */
export interface DueDate { date: string; words: string; section: string; source: string; to_check: boolean; notes: string[] }
export interface ClaimRow {
  id: string; ordinal: number; kind: "contract" | "variation" | "retention"; label: string; line_id: string; line: string; variation_id: string; variation: string;
  mapped_by: string; band: InvoiceBand; to_date: Money; previous: Money; this_claim: Money; certified_before: Money; claimed: Money;
  assessed_to_date: Money; scheduled: Money; below_claimed: boolean; reason: string; version: number;
}
export interface Claim {
  invoice_id: string; package_id: string; number: string; served_on: string; served_confirmed: boolean; reference_date: string; schedule_served_on: string;
  claimed: Money; scheduled: Money; retention: Money; net: Money; gst: Money; payable: Money; unassessed: number; missing_reasons: number;
  counts: boolean; late: boolean; alert: "" | "overdue" | "soon" | "check"; schedule_due: DueDate; payment_due: DueDate;
  issued: { report_id: string; version_id: string; number: number; issued_on: string; snapshot_sha256: string; file: string }[];
  settled_by: string[]; flags: InvoiceFlag[]; rows: ClaimRow[];
}
export interface OpCall { op: string; fields: Record<string, unknown> }
export interface OpsReply { change_id?: string; undo: { kind: string } }

export const invoiceApi = {
  /** Variation and claim writes go through POST /ops, so each has one Undo. */
  ops: (p: string, area: "variations" | "claims", ops: OpCall[]) => request<OpsReply>("POST", `/projects/${p}/ops`, { area, ops }),
  undo: (p: string, change: string) => request<{ undone: boolean }>("POST", `/projects/${p}/changes/${change}/undo`, {}),
  /** Issue is the PM's press only; chat never calls it. */
  issueSchedule: (p: string, inv: string) => request<{ report_id: string; version_id: string; number: number }>("POST", `${base(p)}/${inv}/schedule`, {}),
  periods: (p: string, pkg: string, body: { schedule_days: number | null; schedule_basis: string; payment_days: number | null; payment_basis: string }) =>
    request<Ledger>("PATCH", `/projects/${p}/procurement/packages/${pkg}/award`, body),
  ledger: (p: string) => request<Ledger>("GET", base(p)),
  edit: (p: string, id: string, fields: Record<string, string>, action = "", target = "") =>
    request<Ledger>("PATCH", `${base(p)}/${id}`, { fields, action, target }),
  split: (p: string, id: string, shares: { line_id: string; amount: string }[]) =>
    request<Ledger>("PUT", `${base(p)}/${id}/allocations`, { shares }),
  fileURL: (p: string, id: string) => `/api${base(p)}/${id}/file?view=1`,
  exportURL: (p: string, f: InvoiceFilter, format: "xlsx" | "csv" = "xlsx", sheet = "") => {
    const q = new URLSearchParams({ format });
    if (sheet) q.set("sheet", sheet);
    for (const [k, v] of Object.entries(f)) if (v) q.set(k, v);
    return `/api${base(p)}/export?${q}`;
  },
};

/** upload streams one file as the raw body, reporting progress. */
export function uploadInvoice(projectId: string, file: File, onProgress: (fraction: number) => void, claimPackage = ""): { done: Promise<Uploaded>; abort: () => void } {
  const xhr = new XMLHttpRequest();
  const done = new Promise<Uploaded>((resolve, reject) => {
    // A file dropped on a contract is that contract's progress claim.
    const claim = claimPackage ? `&kind=progress_claim&package=${encodeURIComponent(claimPackage)}` : "";
    xhr.open("POST", `/api${base(projectId)}?name=${encodeURIComponent(file.name)}${claim}`);
    xhr.withCredentials = true;
    xhr.upload.onprogress = (e) => { if (e.lengthComputable) onProgress(e.loaded / e.total); };
    xhr.onload = () => {
      if (xhr.status === 200 || xhr.status === 201) resolve(JSON.parse(xhr.responseText) as Uploaded);
      else reject(new ApiError(xhr.status, xhr.responseText.trim() || "Upload failed"));
    };
    xhr.onerror = () => reject(new ApiError(0, "Upload interrupted. Drop the file again; nothing is lost."));
    xhr.onabort = () => reject(new ApiError(0, "Upload cancelled"));
    xhr.send(file);
  });
  return { done, abort: () => xhr.abort() };
}
