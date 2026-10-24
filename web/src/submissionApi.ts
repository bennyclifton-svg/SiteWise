import { ApiError, request } from "./api";

// Money is an exact decimal string from the server; null is unknown. The
// browser never sums it: every total arrives computed.
export type Money = string | null;
export type CellState = "" | "priced" | "not_priced" | "included" | "excluded";
export type SubmissionStatus = "reading" | "received" | "needs_review" | "awarded" | "unsuccessful" | "superseded" | "withdrawn" | "not_proposal";

export interface ScheduleItem { unit?: string; key: string; kind: "stage" | "element" | "option" | "reimbursable" | "rate" | "criterion"; label: string; detail?: string; base: boolean; ordinal: number; source: string; retired: boolean }
export interface Cell { id: string; item_key: string; label: string; state: CellState; basis?: string; page?: number; passage?: string; band: "green" | "amber" | "blank" | "grey"; decided_by: "rule" | "jev" | "user"; amount: Money }
export interface Firm { id: string; name: string; abn: string; address: string; contact: string; email: string; phone: string; status: "draft" | "confirmed"; version: number }
export interface Submission { id: string; project_id: string; package_id: string; firm_id: string; document_id: string; filename: string; kind: string; status: SubmissionStatus; received_on: string; superseded_by: string; gst_basis: string; flags: string[]; created_at: string; version: number }
export interface Totals { base_total: Money; known_base: Money; stated_total: Money; not_priced: number; unread: number; options_total: Money; others_total: Money; stated_total: Money; gap: Money; lowest: boolean }
export interface FieldDecision { field: string; value: string; band: "green" | "amber" | "blank" | "grey"; decided_by: "rule" | "jev" | "user" }
export interface BuilderDetail { text: string; relationship: string; section_page: number }
export interface BuilderSection { key: string; label: string; summary?: string; details: BuilderDetail[] }
export interface BuilderReview { ex_gst: Money; inc_gst: Money; basis: string; sources: string[]; issues: string[]; sections: BuilderSection[] }
export interface QuoteFee { id: string; label: string; printed: string; amount: Money; role: string; page: number; source: string; heading: string; suggestion?: string }
export interface QuoteReview { fees: QuoteFee[]; coverage: Record<string, {fee_id: string; status: string}>; issues: string[] }
export interface Column { quote?: QuoteReview; builder?: BuilderReview; undo_alignment: boolean; submission: Submission; firm: Firm | null; cells: Record<string, Cell>; others: Cell[]; totals: Totals; needs_review: boolean; fields: Record<string, FieldDecision> }
export interface AwardLine { cost_item_id?: string; label: string; item_keys: string[]; price_ids: string[]; amount: Money; invoiced: boolean }
export interface Award { id?: string; submission_id: string; firm_name: string; parent_label: string; parent_item_id?: string; budget: Money; total: Money; lines: AwardLine[]; created_at?: string; locked: boolean }
export interface PriceTable { builder?: boolean; package_id: string; title: string; discipline_id: string; lifecycle_status: string; items: ScheduleItem[]; columns: Column[]; earlier: Submission[]; award: Award | null }
export interface FirmCard { submission_id: string; firm_name: string; filename: string; received_on: string; status: SubmissionStatus; base_total: Money; known_base: Money; not_priced: number; unread: number; lowest: boolean; needs_review: boolean; flags: string[] }
export interface Board { packages: Record<string, FirmCard[]>; unassigned: FirmCard[] }
export interface FirmFields { name: string; abn: string; address: string; contact: string; email: string; phone: string }
export interface ResponsePassage { ordinal: number; heading: string; page: number; excerpt: string }
export interface CriterionResponse { passages: ResponsePassage[]; rating: "" | "strong" | "acceptable" | "concern"; band: string; decided_by: string }
export interface ReportData {
  table: PriceTable;
  report: { criteria: ScheduleItem[]; columns: { submission_id: string; responses: Record<string, CriterionResponse> }[]; recommendation: string; draft: string; edited_by: string; signed_off_by: string; signed_off_at: string | null; version: number };
}
export interface AwardRequest { submission_id: string; lines: { label: string; item_keys: string[]; price_ids: string[] }[]; total: string }

const p = (project: string) => `/projects/${project}`;

export const submissionApi = {
  row: (project: string, pkg: string, key: string, label: string, section: string, unit = "") => request<PriceTable>("PUT", `${p(project)}/procurement/packages/${pkg}/comparison-row`, { key, label, section, unit }),
  align: (project: string, sub: string, version: number, target: string, price_ids: string[], undo = false, exclude = false) => request<PriceTable>("POST", `${p(project)}/submissions/${sub}/alignment`, { version, target, price_ids, undo, exclude }),
  board: (project: string) => request<Board>("GET", `${p(project)}/procurement/board`),
  table: (project: string, pkg: string) => request<PriceTable>("GET", `${p(project)}/procurement/packages/${pkg}/prices`),
  setCell: (project: string, sub: string, key: string, state: CellState, amount = "", note = "") =>
    request<PriceTable>("PUT", `${p(project)}/submissions/${sub}/cells/${encodeURIComponent(key)}`, { state, amount, note }),
  edit: (project: string, sub: string, edit: { package_id?: string; firm_id?: string; firm?: FirmFields; status?: string; received_on?: string }) =>
    request<PriceTable>("PATCH", `${p(project)}/submissions/${sub}`, edit),
  addFirm: (project: string, pkg: string, firm: { firm_id: string } | { firm: FirmFields }) =>
    request<PriceTable>("POST", `${p(project)}/procurement/packages/${pkg}/firms`, firm),
  firms: (q: string) => request<{ firms: Firm[] }>("GET", `/firms?q=${encodeURIComponent(q)}`),
  previewAward: (project: string, pkg: string, a: AwardRequest) => request<Award>("POST", `${p(project)}/procurement/packages/${pkg}/award/preview`, a),
  award: (project: string, pkg: string, a: AwardRequest) => request<PriceTable>("POST", `${p(project)}/procurement/packages/${pkg}/award`, a),
  undoAward: (project: string, pkg: string) => request<PriceTable>("DELETE", `${p(project)}/procurement/packages/${pkg}/award`),
  report: (project: string, pkg: string) => request<ReportData>("GET", `${p(project)}/procurement/packages/${pkg}/report`),
  rate: (project: string, pkg: string, sub: string, key: string, rating: string) =>
    request<ReportData>("PUT", `${p(project)}/procurement/packages/${pkg}/submissions/${sub}/ratings/${encodeURIComponent(key)}`, { rating }),
  recommend: (project: string, pkg: string, text: string, signOff: boolean) =>
    request<ReportData>("PUT", `${p(project)}/procurement/packages/${pkg}/report`, { text, sign_off: signOff }),
  fileURL: (project: string, sub: string, page?: number) => `/api${p(project)}/submissions/${sub}/file?view=1${page ? `#page=${page}` : ""}`,
  exportURL: (project: string, pkg: string) => `/api${p(project)}/procurement/packages/${pkg}/prices/export`,
  responseFormURL: (report: string) => `/api/reports/${report}/response-form`,
};

/** uploadProposal files one proposal dropped on a package row. */
export function uploadProposal(project: string, pkg: string, file: File, submissionId = ""): Promise<{ submission_id: string }> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", `/api${p(project)}/procurement/packages/${pkg}/submissions?name=${encodeURIComponent(file.name)}${submissionId ? `&submission_id=${encodeURIComponent(submissionId)}` : ""}`);
    xhr.withCredentials = true;
    xhr.onload = () => xhr.status === 201 ? resolve(JSON.parse(xhr.responseText)) : reject(new ApiError(xhr.status, xhr.responseText.trim() || "Upload failed"));
    xhr.onerror = () => reject(new ApiError(0, "Upload interrupted. Drop the file again."));
    xhr.send(file);
  });
}

export const STATUS_WORDS: Record<SubmissionStatus, string> = {
  reading: "Reading…", received: "Received", needs_review: "Needs review", awarded: "Awarded", unsuccessful: "Unsuccessful",
  superseded: "Superseded", withdrawn: "Withdrawn", not_proposal: "Not a proposal",
};

export const FLAG_WORDS: Record<string, string> = {
  ocr_review: "Read from scanned pages: check the figures against the PDF, especially handwriting.",
  incomplete_text: "Some pages could not be read. Missing fees remain unknown; check the PDF.",
  shared_fee: "The same quoted fee was selected for several rows. Counted once; check which scope it covers.",
  rates_only: "Hourly rates only: no lump sum is offered.",
  options_ab: "Prices more than one option (A and B): check which one is compared.",
  gst_not_stated: "GST basis not stated; read as excluding GST.",
  gst_inc_normalised: "Quoted including GST; shown ex GST (÷ 1.1).",
  total_gap: "The firm's stated total differs from its items.",
  no_text: "No readable text: type the prices from the file.",
  no_schedule: "The package has no schedule yet.",
  response_form: "Returned on the response form: read row by row.",
};
