import { request } from "./api";

// The project's correspondence record (docs/plans/2026-11-14-email-intake.md
// and docs/plans/2026-11-17-correspondence-transmittals.md). Every word of a
// message is shown as text: a body is never rendered as HTML.

export interface MailPart {
  n: number;
  filename: string;
  content_type: string;
  byte_size: number;
  skipped: string;
  document_id?: string;
  number?: string;
  revision?: string;
  title?: string;
}

export interface MailMessage {
  id: string;
  sender: string;
  sender_name: string;
  subject: string;
  to: string;
  cc: string;
  body?: string;
  received_at: string;
  status: "filed" | "held";
  reason: string;
  parts: MailPart[];
  direction: "in" | "out";
  number?: number;
  with: string;
  thread_id: string;
  firm_id?: string;
  firm?: string;
  package_id?: string;
  package?: string;
  linked_by?: string;
  replied: boolean;
  send_status?: "" | "queued" | "sent" | "failed";
  send_error?: string;
  sent_at?: string;
  version: number;
  /** Jev's pick, shown as a chip only when amber and not acted on. */
  notice?: string;
  pick_state?: string;
}

export interface MailList { address: string; held: number; messages: MailMessage[] }

export interface DocumentSend { message_id: string; number: number; sent_at: string; to: string; cc: string; document_id?: string; revision: string; send_status: string; this: boolean }
export interface Holder { address: string; firm?: string; firm_id?: string; revision: string; number: number }
export interface DocumentSends { sends: DocumentSend[]; newest: boolean; holding_older: Holder[] }

export const readMail = (projectId: string) => request<MailList>("GET", `/projects/${projectId}/mail`);
export const readMessage = (projectId: string, id: string) => request<MailMessage>("GET", `/projects/${projectId}/mail/${id}`);
export const readDocumentSends = (projectId: string, documentId: string) => request<DocumentSends>("GET", `/projects/${projectId}/mail?document=${encodeURIComponent(documentId)}`);

export const transmittalNumber = (n?: number) => (n ? `TR-${String(n).padStart(3, "0")}` : "");

/** Held reasons in plain words (email intake, Task 9). */
export const HELD_WORDS: Record<string, string> = {
  unknown_sender: "Not from your team",
  unverified_sender: "Could not confirm the sender",
  plan_limit: "Monthly document limit reached",
  archived: "Project is archived",
  held_full: "Held mail is full",
  busy: "Too much mail this hour",
};

/** Jev's picks in the PM's words, for the chip. */
export const NOTICE_WORDS: Record<string, string> = {
  rfi: "RFI",
  direction: "direction",
  variation_claim: "variation claim",
  eot_delay: "delay notice",
  payment_claim: "payment claim",
  dispute: "notice of dispute",
  other_notice: "notice",
};

export function shortDay(iso?: string) {
  if (!iso) return "";
  return new Date(iso).toLocaleDateString("en-AU", { day: "numeric", month: "short" });
}
