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
  /** The progress claim this tax invoice is for; link_note says it waits for the schedule, or invoiced the claimed amount.
   * held: not counted until the PM places it. not_claim: the PM said it is not for a claim. */
  settles_claim?: string; settles_by?: string; link_note?: "waiting" | "claimed_amount"; held: boolean; not_claim: boolean;
  fields: Record<string, InvoiceField>; allocations: InvoiceAllocation[] | null; rows: InvoiceRow[] | null;
}
export interface LedgerLine { id: string; label: string; section: string; category: string; archived: boolean; budget: Money; invoiced: Money; remaining: Money; percent_spent: string; over_budget: boolean; invoices: number; contract_remaining: Money }
export interface LedgerSummary { lines: LedgerLine[]; unallocated: LedgerLine; invoiced: Money; budget: Money; budget_unknown: number; remaining: Money; percent_spent: string; counted: number; excluded: number; comparable: boolean; sections: Record<string, Money>; sections_contract_remaining: Record<string, Money>; contract_remaining: Money }
export interface LedgerPackage { id: string; title: string; kind: string; discipline: string; abn: string }
export interface Ledger { groups?: Record<string, LedgerLine & { line_ids: string[] }>; project: string; tax_basis: string; summary: LedgerSummary; invoices: Invoice[]; packages: LedgerPackage[]; lines: LedgerLine[]; contracts?: Contract[]; variations?: Variation[]; claims?: Claim[]; sheet?: Sheet }
/** A live contract (an award) as the Variations and Progress claims tabs show it; every figure is the server's. */
export interface Contract {
  award_id: string; package_id: string; title: string; firm: string; claims: boolean;
  lines: { id: string; label: string; amount: Money }[];
  contract_sum: Money; approved_variations: Money; contract_sum_to_date: Money; certified_to_date: Money; retention_held: Money;
  schedule_days: number | null; schedule_basis: "business" | "calendar"; payment_days: number | null; payment_basis: "business" | "calendar";
  /** The sum to date with provisional sums marked final taken at what was completed; paid is the payment list's total ex GST. */
  provisional_adjustment: Money; adjusted: Money; paid_to_date: Money;
  latest_schedule: { claim_id: string; number: string; period: string; issued_on: string; report_id: string; version_id: string; ex_gst: string; gst: string; inc_gst: string } | null;
  /** What a schedule prints of the contract, typed once: its date, and retention as a percent of a base. "" is not entered. */
  /** A consultant's or other services contract; act_on: its schedules name the Act (act_applies is the PM's switch, "" the default by kind). */
  services: boolean; act_on: boolean; act_applies: "" | "yes" | "no";
  contract_date: string; retention_percent: string; retention_base: "" | "works" | "let" | "to_date"; firm_abn: string;
  /** What the contract's payment schedules print; stored is false while it is still the cost lines. */
  breakdown: BreakdownLine[];
  /** Cost lines the breakdown does not add up to: diff is the breakdown less the contract line. */
  breakdown_gaps: { cost_line_id: string; label: string; diff: string }[];
}
export interface BreakdownLine { id: string; cost_line_id: string; section: "works" | "provisional"; label: string; note: string; amount: Money; final: boolean; retired: boolean; source: "award" | "tender" | "user"; stored: boolean }
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
  /** As the schedule prints them: the claim's number on its contract, its period (AUG26) and its tax invoice. */
  display_number: string; period: string; invoice_no: string; invoice_date: string;
  /** The PM's own words on the schedule; retention_open: the contract has retention terms and this claim has not said what it holds. */
  schedule_note: string; retention_open: boolean;
  /** The PM typed or confirmed the day served; a day set when the schedule's message was sent is not. */
  schedule_served_confirmed: boolean;
  /** The claim is its own tax invoice; or no invoice is linked to it yet. */
  tax_invoice: boolean; awaiting_invoice: boolean;
  /** The message the issued schedule last went out on. */
  schedule_send?: { message_id: string; status: "queued" | "sent" | "failed"; sent_on: string };
}
/** One line of a payment schedule sheet: completed is to date, prior what was certified before, this_claim the difference. */
export interface SheetLine {
  key: string; row_id: string; label: string; note: string; total_let: Money; completed: Money; percent: string; prior: Money; this_claim: Money;
  claimed_to_date: Money; below_claimed: boolean; reason: string; not_approved: boolean;
  /** How far completed is over the sum; final is a provisional sum marked final, final_claimed one this claim still moves. */
  over_by: Money; final: boolean; final_claimed: boolean; retired: boolean;
}
export interface SheetSection { kind: "works" | "provisional" | "variations" | "unplaced"; lines: SheetLine[]; total_let: Money; completed: Money; prior: Money; this_claim: Money }
export interface PaymentRow { claim_id: string; period: string; number: string; invoice_no: string; invoice_date: string; ex: Money; gst: Money; inc: Money; current: boolean; unserved: boolean }
/** A claim's payment schedule, every figure the server's. retention.held is what this schedule withholds; a return is negative. */
export interface Sheet {
  claim_id: string; sections: SheetSection[]; total_let: Money; completed: Money; prior: Money; this_claim: Money;
  retention: { rate_percent: string; base_kind: "" | "works" | "let" | "to_date"; base: Money; open: boolean; limit: Money; held_before: Money; held: Money; percent: string; held_to_date: Money };
  header?: SheetHeader;
  net: Money; gst: Money; payable: Money; payments: PaymentRow[]; paid: { ex: Money; gst: Money; inc: Money }; paid_in_full: boolean;
  /** What the server has to say about the contract's records, such as a breakdown that does not add up. */
  notes: string[];
}
/** What a schedule's header prints. A party is typed (Project terms) or stands in: the Profile's client, the organisation's name, the signed-in user. */
export interface SheetHeader {
  project: string; address: string; principal: string; principal_abn: string; principal_email: string; principal_typed: boolean;
  contractor: string; contractor_abn: string; contractor_email: string;
  assessor_org: string; assessor_company_no: string; assessor_typed: boolean; approver: string; approver_typed: boolean;
  contract_date: string; act: string; warnings: string[]; typed: Record<string, string>;
}
export interface OpCall { op: string; fields: Record<string, unknown> }
export interface OpsReply { change_id?: string; undo: { kind: string } }

export const invoiceApi = {
  /** Variation and claim writes go through POST /ops, so each has one Undo. */
  ops: (p: string, area: "variations" | "claims" | "delivery", ops: OpCall[]) => request<OpsReply>("POST", `/projects/${p}/ops`, { area, ops }),
  /** A contract's own terms: its date and its retention. Only the fields sent are changed. */
  terms: (p: string, pkg: string, body: Record<string, string>) => request<Ledger>("PATCH", `/projects/${p}/procurement/packages/${pkg}/award`, body),
  undo: (p: string, change: string) => request<{ undone: boolean }>("POST", `/projects/${p}/changes/${change}/undo`, {}),
  /** Issue is the PM's press only; chat never calls it. */
  issueSchedule: (p: string, inv: string) => request<{ report_id: string; version_id: string; number: number }>("POST", `${base(p)}/${inv}/schedule`, {}),
  periods: (p: string, pkg: string, body: { schedule_days: number | null; schedule_basis: string; payment_days: number | null; payment_basis: string }) =>
    request<Ledger>("PATCH", `/projects/${p}/procurement/packages/${pkg}/award`, body),
  /** With a claim's id the read also carries that claim's payment schedule sheet. */
  ledger: (p: string, sheet = "") => request<Ledger>("GET", sheet ? `${base(p)}?sheet=${encodeURIComponent(sheet)}` : base(p)),
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
