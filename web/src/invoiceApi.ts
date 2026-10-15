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
export interface LedgerLine { id: string; label: string; section: string; category: string; archived: boolean; budget: Money; invoiced: Money; remaining: Money; percent_spent: string; over_budget: boolean; invoices: number }
export interface LedgerSummary { lines: LedgerLine[]; unallocated: LedgerLine; invoiced: Money; budget: Money; budget_unknown: number; remaining: Money; percent_spent: string; counted: number; excluded: number; comparable: boolean; sections: Record<string, Money> }
export interface LedgerPackage { id: string; title: string; kind: string; discipline: string; abn: string }
export interface Ledger { project: string; tax_basis: string; summary: LedgerSummary; invoices: Invoice[]; packages: LedgerPackage[]; lines: LedgerLine[] }
export interface InvoiceFilter { supplier: string; month: string; line: string; status: string }
export interface Uploaded { id: string; status: InvoiceStatus; created: boolean }

const base = (p: string) => `/projects/${p}/invoices`;

export const invoiceApi = {
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
export function uploadInvoice(projectId: string, file: File, onProgress: (fraction: number) => void): { done: Promise<Uploaded>; abort: () => void } {
  const xhr = new XMLHttpRequest();
  const done = new Promise<Uploaded>((resolve, reject) => {
    xhr.open("POST", `/api${base(projectId)}?name=${encodeURIComponent(file.name)}`);
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
