import type { DeliveryRecord, RFIDetails, ThreadMessage } from "./deliveryApi";
import { statusWords } from "./registerKinds";

// An RFI's history, oldest first, from what SiteWise records: raised, each
// status change, what went out and each reply in its thread, the answer
// and the possible answers to check (register panel, 20 November 2026).
// Every date here is a record's own; nothing is worked out from a calendar.

/** How a line is drawn: a ring for what the PM sent, amber for what came in, green for the answer. */
export type HistoryTone = "note" | "out" | "in" | "answer" | "check";

export interface HistoryLine {
  key: string;
  /** Sydney date, YYYY-MM-DD, or "" when the record has none. */
  date: string;
  tone: HistoryTone;
  kind: "raised" | "status" | "sent" | "reply" | "answer" | "typed" | "reading";
  text: string;
  /** A second line: what was sent, how many attachments, why it may answer. */
  detail?: string;
  /** A reply's first lines, as plain text. */
  quote?: string;
  message?: ThreadMessage;
  documentId?: string;
  readingId?: string;
  sort: string;
}

/** An instant as its Sydney date. */
export function sydneyDate(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Australia/Sydney", year: "numeric", month: "2-digit", day: "2-digit" }).format(d);
}

const plural = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;
const trNumber = (n?: number) => (n ? `TR-${String(n).padStart(3, "0")}` : "");

export function rfiHistory(row: DeliveryRecord, titleOf: (documentId: string) => string): HistoryLine[] {
  const d = row.details as RFIDetails;
  const p = row.provenance;
  const out: HistoryLine[] = [];
  const answers = new Set(d.answer_documents ?? []);
  const thread = row.thread ?? [];
  const replyDocs = new Set(thread.flatMap(m => m.direction === "in" && m.body_document ? [m.body_document] : []));
  // Answered at: the newest change to Answered, else the date answered.
  const toAnswered = [...(p.status_history ?? [])].reverse().find(h => h.status === "answered");
  const answeredAt = toAnswered ? sydneyDate(toAnswered.at) : row.actual_date ?? "";
  const answeredSort = toAnswered ? `${answeredAt}|2|${toAnswered.at}` : answeredAt ? `${answeredAt}|3` : "~";

  const raised = d.raised_on ?? sydneyDate(p.at);
  out.push({ key: "raised", kind: "raised", tone: "note", date: raised, sort: `${raised}|0`,
    text: `Raised${d.raised_by ? ` by ${d.raised_by}` : ""}${(p.initial_status ?? "draft") === "draft" ? " · draft" : ""}` });
  (p.status_history ?? []).forEach((h, i) => {
    if (h.from === h.status) return; // a change of kind only
    const date = sydneyDate(h.at);
    out.push({ key: `status-${i}`, kind: "status", tone: "note", date, sort: `${date}|1|${h.at}`, text: `${statusWords(h.from)} → ${statusWords(h.status)}` });
  });
  for (const m of thread) {
    const date = sydneyDate(m.at);
    if (m.direction === "out") {
      const state = m.send_status === "queued" ? "sending…" : m.send_status === "failed" ? "not sent" : "";
      out.push({ key: `msg-${m.id}`, kind: "sent", tone: "out", date, sort: `${date}|1|${m.at}`, message: m,
        text: `Sent to ${m.with || "no one"}`, detail: [plural(m.documents, "document"), trNumber(m.number), state].filter(Boolean).join(" · ") });
      continue;
    }
    const answer = !!m.body_document && answers.has(m.body_document);
    out.push({ key: `msg-${m.id}`, kind: "reply", tone: answer ? "answer" : "in", date, sort: `${date}|1|${m.at}`, message: m,
      text: `Reply from ${m.with || "someone"}${answer ? " · the answer" : ""}`, quote: m.excerpt || undefined,
      detail: m.documents ? plural(m.documents, "attachment") : undefined, documentId: m.body_document, readingId: m.reading || undefined });
  }
  for (const id of d.answer_documents ?? []) {
    if (replyDocs.has(id)) continue; // the reply's own line says so
    out.push({ key: `answer-${id}`, kind: "answer", tone: "answer", date: answeredAt, sort: answeredSort, documentId: id, text: `Answered by ${titleOf(id)}` });
  }
  if ((d.answer ?? "").trim()) {
    out.push({ key: "typed", kind: "typed", tone: "answer", date: answeredAt, sort: answeredSort === "~" ? "~~" : `${answeredSort}~`, text: "Answer" });
  }
  for (const rd of row.readings ?? []) {
    if (replyDocs.has(rd.document_id)) continue; // shown on its reply, with Use as the answer
    const date = rd.filed_on ?? "";
    out.push({ key: `reading-${rd.id}`, kind: "reading", tone: "check", date, sort: date ? `${date}|4` : "~", readingId: rd.id, documentId: rd.document_id,
      text: `Possible answer: ${rd.document}`, detail: rd.basis || undefined });
  }
  return out.sort((a, b) => (a.sort < b.sort ? -1 : a.sort > b.sort ? 1 : 0));
}

/** A notice's or other row's history: raised and each status change. */
export function rowHistory(row: DeliveryRecord): HistoryLine[] {
  const p = row.provenance;
  const at = sydneyDate(p.at);
  const out: HistoryLine[] = [{ key: "raised", kind: "raised", tone: "note", date: at, sort: `${at}|0`, text: "Added to the register" }];
  (p.status_history ?? []).forEach((h, i) => {
    if (h.from === h.status) return;
    const date = sydneyDate(h.at);
    out.push({ key: `status-${i}`, kind: "status", tone: "note", date, sort: `${date}|1|${h.at}`, text: `${statusWords(h.from)} → ${statusWords(h.status)}` });
  });
  return out.sort((a, b) => (a.sort < b.sort ? -1 : a.sort > b.sort ? 1 : 0));
}
