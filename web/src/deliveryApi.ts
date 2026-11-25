import { request } from "./api";

// The register (docs/plans/2026-11-17-rfis-review-comments.md). Reads come
// from GET /delivery; every write goes through POST /ops in area delivery,
// so each screen action has one Undo.

export interface DeliveryContent {
  kind: string; title: string; status: string; owner_text: string;
  baseline_date: string | null; target_date: string | null; forecast_date: string | null; actual_date: string | null; as_of: string | null;
  package_id: string; work_item_id: string; stage_id: string; system_id?: string;
  details: Record<string, unknown>;
}
export interface Reading { id: string; document_id: string; document: string; decided_by: string; band: string; basis?: string; filed_on?: string; field?: "answers" | "evidence" }
export interface DeliveryRecord extends DeliveryContent {
  owner_user_id?: string;
  id: string; version: number; origin: string; review_status: string; meaning: string; source_proposal_key?: string;
  number?: number; with?: string; late_days?: number; readings?: Reading[];
  /** How a worked-out due date was reached (Go): SiteWise's default, the project's period, or typed. */
  due_basis?: "default" | "project" | "typed";
  /** An RFI's attached documents as the RFP transmittal's rows (Go). */
  transmittal?: string[][];
  /** An RFI's mail: what went out and the filed replies in its thread, oldest first (Go). */
  thread?: ThreadMessage[];
  /** Who in the team added the row (Go): their email, or "a former team member". */
  raised_by_member?: string;
  provenance: { actor?: string; at?: string; last_edited_by?: string; last_edited_at?: string; proposal?: { kind: string; draft: boolean; label: string };
    sources?: { document_id?: string; filename?: string; excerpt?: string; page?: number }[];
    status_history?: { from: string; status: string; at: string; from_kind?: string; kind?: string }[];
    initial_status?: string;
    // A consent condition's reading and the notices that wrote it (plan 6).
    reading?: { fields?: Record<string, { value: string; band: string; decided_by: string }> };
    flag?: string; deferred_commencement?: boolean };
}
/** One message of an RFI's thread, as the sheet's history shows it. */
export interface ThreadMessage {
  id: string; direction: "in" | "out"; at: string;
  /** The recipients of what went out; the sender of a reply. */
  with: string;
  number?: number; send_status?: "" | "queued" | "sent" | "failed";
  /** A reply's first lines, plain text. */
  excerpt?: string;
  /** Documents that went out, or a reply's attachments. */
  documents: number;
  /** The document a reply's text was filed as, and its possible answer on this RFI. */
  body_document?: string; reading?: string;
}
/** What is particular to an RFI. To, Cc and Subject are what its sheet sends. */
export interface RFIDetails {
  question?: string; answer?: string; ref?: string; raised_by?: string; raised_on?: string | null;
  answer_documents?: string[]; due_typed?: boolean; sent_message_id?: string;
  to?: string[]; cc?: string[]; subject?: string;
}
/** A consent condition's details: the notice's own number and words, and what the PM edits. */
export interface ConditionDetails {
  consent_ref?: string; number?: string; heading?: string; text?: string; stage?: string; stage_heading?: string;
  discipline?: string; evidence_needed?: boolean | null;
  evidence?: { document_id: string; linked_by: string; linked_at: string }[] | null;
  source?: { document_id: string; page: number } | null; rfp?: { report_id: string; stage: string; line_id: string } | null;
}
export interface ConsentRead {
  document_id: string; title: string; status: "queued" | "reading" | "done" | "failed"; error?: string; rows: number;
  notice?: { consent_ref: string; mod_ref?: string; determined?: string; conditions: number; gaps: string[] };
}
export interface RegisterKind { kind: string; rows: number; open: number; to_check: number }
export interface AlsoOpen { system: string; label: string; spec_unknowns: number; exclusions: number; exclusion_packages: string[]; main_works_exclusions: number; main_works_package?: string }
export interface RegisterView {
  items: DeliveryRecord[]; kinds: RegisterKind[];
  response_days: number; response_typical: boolean; calendar: "program" | "weekdays"; today: string;
  /** Calendar days to assess an EOT claim; typical while it is SiteWise's default. */
  eot_days: number; eot_typical: boolean;
  /** The agreed completion date, the days granted on EOT claims and the date after them (Go). */
  completion: Completion;
  systems: { id: string; label: string }[]; also_open?: AlsoOpen;
  consents?: ConsentRead[]; consent?: { number?: string; date?: string; lapse_date?: string };
  /** Why the RFI sheet's Send is off on this server, in Go's words; absent when sending is set up. */
  send_off?: string;
}
/** The project's agreed completion and the days granted, computed by Go; unknown stays unknown. */
export interface Completion { agreed?: string; days_granted: number; adjusted?: string; claims: number }
/** An EOT claim's result, as recorded. */
export interface EOTResult { outcome: "granted" | "granted_in_part" | "rejected" | "more_information"; days_granted?: number | null; reasons?: string; assessed_on?: string | null }
/** A received item's details: a notice of delay, an EOT claim or a letter. */
export interface ClaimDetails {
  /** "out" when the host raised it with another party; received otherwise. */
  direction?: "in" | "out";
  ref?: string; raised_by?: string; received_on?: string | null; cause?: string; delay_from?: string | null; delay_to?: string | null;
  days_claimed?: number | null; notice_id?: string; message_id?: string; due_typed?: boolean; result?: EOTResult | null;
  body?: string; answer?: string;
}

/** The register's gates in project order, and their Due words. */
export const GATES: [string, string][] = [["before_cc", "Before CC"], ["before_work", "Before work starts"], ["during_works", "During works"],
  ["before_occupation", "Before OC"], ["before_subdivision", "Before SC"], ["ongoing", "Ongoing after occupation"], ["throughout", "Throughout"]];
export interface OpCall { op: string; fields: Record<string, unknown> }
export interface OpsReply { change_id?: string; undo: { kind: string } }

export const deliveryApi = {
  view: (project: string, system = "") =>
    request<RegisterView>("GET", `/projects/${project}/delivery${system ? `?system=${encodeURIComponent(system)}` : ""}`),
  ops: (project: string, ops: OpCall[]) => request<OpsReply>("POST", `/projects/${project}/ops`, { area: "delivery", ops }),
  /** Read conditions: queues the background read of a filed notice. */
  readNotice: (project: string, documentId: string) => request<{ status: string }>("POST", `/projects/${project}/delivery/imports`, { document_id: documentId }),
  undo: (project: string, change: string) => request<{ undone: boolean }>("POST", `/projects/${project}/changes/${change}/undo`, {}),
};

/** "RFI-007" */
export const rfiNumber = (n?: number) => (n ? `RFI-${String(n).padStart(3, "0")}` : "");

const PREFIX: Record<string, string> = { delay_notice: "NOD", eot_claim: "EOT", letter: "LTR" };
/** A numbered row's number as shown: RFI-007, EOT-4, NOD-2, LTR-1; "" for a kind with no number. */
export const numberLabel = (kind: string, n?: number) => (kind === "rfi" ? rfiNumber(n) : n && PREFIX[kind] ? `${PREFIX[kind]}-${n}` : "");

/** Today in Sydney, YYYY-MM-DD: the raised date of an RFI typed in the grey row. */
export function sydneyToday(): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Australia/Sydney", year: "numeric", month: "2-digit", day: "2-digit" }).format(new Date());
}
