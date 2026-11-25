// A register row opened as one sheet in the middle panel (owner,
// 20 November 2026). An RFI's sheet holds who it goes to, its question, its
// transmittal and its history, and its one Send goes through the same
// POST /transmittals and checks as the Send dialog. Every edit goes through
// POST /ops (area delivery) with Undo. Nothing here calls Jev or the model;
// every date shown is a record's own.

import { useEffect, useRef, useState, type ReactNode } from "react";
import { ApiError, request } from "./api";
import { statusChip, statusItems } from "./Delivery";
import { ClaimDetail } from "./ClaimSheet";
import { numberLabel, rfiNumber, type DeliveryRecord, type RFIDetails } from "./deliveryApi";
import type { MailMessage } from "./mailApi";
import { BackButton, MoreMenu, PagePrimaryAction, type MenuItem } from "./PageActions";
import { RfpTransmittal, type TransmittalSelection } from "./RfpTransmittal";
import { editOps, type RegisterData } from "./registerData";
import { rfiHistory, rowHistory, type HistoryLine } from "./registerHistory";
import { cond, GATE_WORDS, KIND, RECEIVED, STATUSES } from "./registerKinds";
import { FirmAddresses, split } from "./SendDialog";
import "./register.css";

/** What chat filled in when it opened an RFI to send (dialog send with rfi). */
export interface SheetFill { n: number; to?: string; cc?: string; subject?: string; warning?: string }

interface Props {
  data: RegisterData;
  id: string;
  projectId: string;
  backTo: string;
  onClose: () => void;
  documents: { id: string; title: string }[];
  transmittal?: TransmittalSelection;
  onOpenDocument: (id: string) => void;
  /** Opens a message of the thread in the repository's Messages. */
  onOpenMessage: (id: string) => void;
  /** A message was queued: the repository's Messages reads again. */
  onSent: () => void;
  fill?: SheetFill;
}

const plural = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;
const sourceIds = (r: DeliveryRecord) => [...new Set((r.provenance.sources ?? []).flatMap(s => s.document_id ? [s.document_id] : []))];

export function RegisterSheet(props: Props) {
  const { data, id, backTo, onClose } = props;
  const row = data.rowOf(id);
  const back = <PagePrimaryAction surface="register"><BackButton to={backTo} onClick={onClose} title={`Back to ${backTo} (Esc)`} here={row ? rowLabel(row) : undefined} /></PagePrimaryAction>;
  useEffect(() => {
    // Escape in the right-hand panel belongs to its tables (clearing a selection).
    const key = (e: KeyboardEvent) => { if (e.key === "Escape" && !(e.target as HTMLElement).closest("input,textarea,select,[role=menu],[role=dialog],.project-side-panel")) onClose(); };
    document.addEventListener("keydown", key);
    return () => document.removeEventListener("keydown", key);
  }, [onClose]);
  if (!data.view) return <section className="register-sheet" aria-label="Register row">{back}<p role="status">Opening…</p></section>;
  if (!row) return <section className="register-sheet" aria-label="Register row">{back}<p className="reg-empty">This row is no longer on the register.</p></section>;
  return <section className="register-sheet" aria-label={`${rowLabel(row)} ${row.title}`} aria-busy={data.busy}>
    {back}
    {row.kind === "rfi" ? <RfiSheet {...props} row={row} /> : <OtherSheet {...props} row={row} />}
  </section>;
}

/** One band of the sheet, headed as an RFP's sections are. */
export function Band({ title, children, className = "" }: { title: string; children: ReactNode; className?: string }) {
  return <section className={`report-section register-band ${className}`.trim()}>
    <div className="report-section-head section-band"><h3>{title}</h3></div>
    {children}
  </section>;
}

/** "RFI-003", "Condition 14", or the kind's name. */
export function rowLabel(r: DeliveryRecord): string {
  if (r.kind === "condition") return `Condition ${cond(r).number ?? ""}`.trim();
  return numberLabel(r.kind, r.number) || KIND[r.kind]?.[0] || r.kind;
}

// Head is the sheet's one header row: number and title, the status chip
// that says who holds it, the due chip, then the three dots and Send.
function Head({ row, data, children, extra = [] }: { row: DeliveryRecord; data: RegisterData; children?: ReactNode; extra?: MenuItem[] }) {
  const [renaming, setRenaming] = useState(false);
  const run = (fields: Record<string, unknown>, label: string) => data.run(editOps([row], fields), label, "sheet");
  const status = statusItems([row], data.busy, (s, label) => void run({ status: s }, label)).flatMap(m => typeof m === "object" && "label" in m ? [{ label: m.label, icon: null, disabled: m.disabled, onSelect: m.onSelect }] : []);
  const items: MenuItem[] = [{ label: "Rename", icon: null, disabled: data.busy, onSelect: () => setRenaming(true) }, ...status, ...extra,
    { label: "Remove from register", icon: null, disabled: data.busy, onSelect: () => void run({ retired: true }, "Removed.") }];
  const due = row.kind === "condition" ? GATE_WORDS[cond(row).stage ?? ""] ?? "Gate not known" : row.target_date ? `Due ${row.target_date}` : "";
  return <header className="register-sheet-head"><div className="register-sheet-title">
    <h2>{renaming
      ? <input className="input register-rename" defaultValue={row.title} maxLength={200} autoFocus aria-label="Item"
        onKeyDown={e => { if (e.key === "Enter") e.currentTarget.blur(); if (e.key === "Escape") { e.currentTarget.value = row.title; e.currentTarget.blur(); } }}
        onBlur={e => { const v = e.target.value.trim(); setRenaming(false); if (v && v !== row.title) void run({ title: v }, "Renamed."); }} />
      : <>{rowLabel(row)} · {row.title}</>}</h2>
    <span className="register-chip register-status" data-band={(row.readings?.length ?? 0) > 0 || row.review_status === "proposed" ? "amber" : undefined}>{statusChip(row)}</span>
    {due && <span className="register-chip" data-band={row.late_days ? "late" : undefined}>{due}{row.late_days ? ` · ${plural(row.late_days, "day")} late` : ""}</span>}
    </div><span className="register-sheet-actions">
      <MoreMenu label={`${rowLabel(row)} actions`} items={items} />
      {children}
    </span>
  </header>;
}

function Notice({ data }: { data: RegisterData }) {
  return <>
    {data.error && data.last?.from !== "list" && <div className="banner" role="alert"><p>{data.error}</p></div>}
    {data.last && data.last.from === "sheet" && <p className="reg-notice" role="status">{data.last.label} <button type="button" className="cell-link" disabled={data.busy} onClick={data.undo}>Undo</button></p>}
  </>;
}

function RfiSheet({ row, data, projectId, documents, transmittal, onOpenDocument, onOpenMessage, onSent, fill }: Props & { row: DeliveryRecord }) {
  const d = row.details as RFIDetails;
  const number = rfiNumber(row.number);
  const defaultSubject = `${number} ${row.title}`.trim();
  const saved = { to: (d.to ?? []).join(", "), cc: (d.cc ?? []).join(", "), subject: d.subject ?? "", question: d.question ?? "" };
  const [to, setTo] = useState(saved.to);
  const [cc, setCc] = useState(saved.cc);
  const [subject, setSubject] = useState(saved.subject);
  const [question, setQuestion] = useState(saved.question);
  const [warning, setWarning] = useState("");
  const [sending, setSending] = useState(false);
  const [sendNote, setSendNote] = useState<{ error?: string; done?: string } | null>(null);
  const pending = useRef<Promise<unknown> | null>(null);
  // A read after any write brings the row's saved values in, except into a
  // field the PM has changed since the last read: the save of one field
  // must not wipe what is being typed in the next.
  const was = useRef({ id: row.id, ...saved });
  useEffect(() => {
    const before = was.current, fresh = before.id !== row.id;
    const take = (now: string, old: string, set: (fn: (v: string) => string) => void) => set(v => (fresh || v.trim() === old.trim() ? now : v));
    take(saved.to, before.to, setTo); take(saved.cc, before.cc, setCc); take(saved.subject, before.subject, setSubject); take(saved.question, before.question, setQuestion);
    was.current = { id: row.id, ...saved };
  }, [row.id, row.version]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => { setSendNote(null); setWarning(""); }, [row.id]);
  // Chat opened this RFI to send it: its fields go in, unsaved, for the PM to check.
  useEffect(() => {
    if (!fill) return;
    if (fill.to !== undefined) setTo(fill.to);
    if (fill.cc !== undefined) setCc(fill.cc);
    if (fill.subject !== undefined) setSubject(fill.subject);
    setWarning(fill.warning ?? "");
  }, [fill]);

  // The sheet saves one field at a time, as each loses focus, so two saves
  // can be in flight together (To, then Cc). Each goes after the one before,
  // and without a version: Go puts the field into the row's current details
  // (liftRFIFields) and takes the version it holds, so the second save
  // neither fails on the first's new version nor undoes it.
  const save = (fields: Record<string, unknown>, label = "Saved.") => {
    const p = Promise.resolve(pending.current).catch(() => undefined)
      .then(() => data.run([{ op: "delivery.edit", fields: { item: row.id, ...fields } }], label, "sheet"));
    pending.current = p;
    return p;
  };
  const dirty = () => {
    const out: Record<string, unknown> = {};
    if (to.trim() !== saved.to) out.to = split(to);
    if (cc.trim() !== saved.cc) out.cc = split(cc);
    if (subject.trim() !== saved.subject && subject.trim() !== defaultSubject) out.subject = subject.trim();
    if (question !== saved.question) out.question = question;
    return out;
  };
  const saveIfChanged = (key: "to" | "cc" | "subject" | "question") => {
    const f = dirty();
    if (key in f) void save({ [key]: f[key] });
  };

  const recipients = split(to), copies = split(cc), docs = sourceIds(row);
  const sendOff = data.view?.send_off ?? "";
  const what = `Send to ${plural(recipients.length + copies.length, "address", "addresses")} with ${plural(docs.length, "document")}`;
  const why = sendOff || (recipients.length === 0 ? "Add who it goes to under To." : "");
  // Send: what is on the sheet, through the one send route. A field the PM
  // changed is saved first, so the RFI keeps what went.
  const send = async () => {
    setSending(true); setSendNote(null);
    try {
      await pending.current;
      const f = dirty();
      if (Object.keys(f).length) await save(f);
      const m = await request<MailMessage>("POST", `/projects/${projectId}/transmittals`, {
        to: recipients, cc: copies, subject: (subject.trim() || defaultSubject), note: question, document_ids: docs, rfi: row.id,
      });
      setWarning("");
      const tr = m.number ? ` as TR-${String(m.number).padStart(3, "0")}` : "";
      setSendNote({ done: `Sent${tr}. It is in Messages.` });
      onSent();
      data.reload();
    } catch (e) {
      setSendNote({ error: e instanceof ApiError && e.message ? e.message : "The message could not be sent. Try again." });
    } finally { setSending(false); }
  };

  const titleOf = (doc: string) => documents.find(x => x.id === doc)?.title ?? "a document";
  const lines = rfiHistory(row, titleOf);
  const accept = (reading: string) => void data.run(editOps([row], { accept_reading: reading }), "Accepted the answer.", "sheet");
  const dismiss = (reading: string) => void data.run([{ op: "delivery.edit", fields: { item: row.id, dismiss_readings: [reading] } }], "Dismissed.", "sheet");
  // A reply with no possible answer to accept: the same edit the PM could
  // make by hand, its document an answer and the RFI answered on the
  // reply's own date.
  const useReply = (l: HistoryLine) => {
    if (l.readingId) return accept(l.readingId);
    if (!l.documentId) return;
    const fields: Record<string, unknown> = { details: { answer_documents: [...(d.answer_documents ?? []), l.documentId] }, status: "answered" };
    if (!row.actual_date && l.date) fields.actual_date = l.date;
    void data.run(editOps([row], fields), "Accepted the answer.", "sheet");
  };
  const raisedTypical = data.view?.response_typical ? " (typical)" : "";
  const theirRef = !!(d.ref || d.raised_by);

  return <>
    <Head row={row} data={data}>
      <button type="button" className="btn btn-primary btn-small" disabled={!!why || sending || data.busy} aria-busy={sending}
        title={why || what} aria-describedby="rfi-send-what" onClick={() => void send()}>{sending ? "Sending…" : "Send"}</button>
    </Head>
    <p id="rfi-send-what" className="register-send-what" role={sendNote?.error ? "alert" : "status"} data-tone={sendNote?.error || sendOff ? "alert" : undefined}>
      {sendNote?.error ?? sendNote?.done ?? (why || what)}</p>
    {warning && <p className="send-error" role="note">{warning}</p>}
    <Notice data={data} />
    <Band title="Particulars"><div className="pf-grid register-grid">
      <label className="pf-row"><span className="pf-label">To</span><span className="pf-value">
        <input className="input" list="rfi-firms" value={to} placeholder="name@example.com, or a firm" onChange={e => setTo(e.target.value)} onBlur={() => saveIfChanged("to")} aria-label="To" /></span></label>
      <label className="pf-row"><span className="pf-label">Cc</span><span className="pf-value">
        <input className="input" list="rfi-firms" value={cc} placeholder="You are copied in" onChange={e => setCc(e.target.value)} onBlur={() => saveIfChanged("cc")} aria-label="Cc" /></span></label>
      <label className="pf-row"><span className="pf-label">Subject</span><span className="pf-value">
        <input className="input" maxLength={300} value={subject} placeholder={defaultSubject} onChange={e => setSubject(e.target.value)} onBlur={() => saveIfChanged("subject")} aria-label="Subject" /></span></label>
      <label className="pf-row"><span className="pf-label">Raised</span><span className="pf-value">
        <input type="date" className="input register-date" key={`${row.id}-${row.version}-raised`} defaultValue={d.raised_on ?? ""} disabled={data.busy} aria-label="Raised"
          onBlur={e => { if (e.target.value !== (d.raised_on ?? "")) void save({ raised_on: e.target.value || null }); }} />
        <span className="register-muted">response time {data.view?.response_days ?? 5} working days{raisedTypical}</span></span></label>
      {/* The sender's own number matters only on an RFI someone else raised. */}
      {theirRef && <label className="pf-row"><span className="pf-label">Their ref</span><span className="pf-value">
        <input className="input" key={`${row.id}-${row.version}-ref`} defaultValue={d.ref ?? ""} maxLength={100} disabled={data.busy} aria-label="Their ref"
          onBlur={e => { if (e.target.value !== (d.ref ?? "")) void save({ ref: e.target.value }); }} /></span></label>}
      <FirmAddresses id="rfi-firms" />
    </div></Band>
    <Band title="Question">
      <textarea className="register-body" aria-label="Question" value={question} maxLength={8000} rows={4} onChange={e => setQuestion(e.target.value)} onBlur={() => saveIfChanged("question")} /></Band>
    {/* An RFI's documents are its transmittal, chosen as an RFP's are: select
        in the repository, then Update Transmittal (owner, 20 November 2026). */}
    <section className="report-section register-transmittal">
      <div className="report-section-head section-band"><h3>Transmittal</h3></div>
      <RfpTransmittal busy={data.busy} selection={transmittal} onWrite={async (_key, ids) => { await save({ source_refs: (ids as string[]).map(x => ({ document_id: x })) }, "Updated the transmittal."); }}
        block={{ id: "rfi:transmittal", label: "Transmittal", text: "", provisional: false, edited: false, conflict: false, source_missing: false, citation_ids: [],
          table: { columns: ["Doc number", "Title", "Rev", "Date", "Discipline"], rows: row.transmittal ?? [] },
          basis: { document_ids: docs } }} />
    </section>
    <section className="report-section register-history-band">
      <div className="report-section-head section-band"><h3>Responses and history</h3></div>
      <Timeline lines={lines} render={l => <>
        {l.kind === "sent" && l.message && <button type="button" className="cell-link" onClick={() => onOpenMessage(l.message!.id)}>Open what was sent</button>}
        {l.kind === "reply" && l.message && <>
          <button type="button" className="cell-link" onClick={() => onOpenMessage(l.message!.id)}>Open the reply</button>
          {l.tone !== "answer" && l.documentId && <button type="button" className="btn btn-small" disabled={data.busy} onClick={() => useReply(l)}>Use as the answer</button>}
        </>}
        {l.kind === "answer" && l.documentId && <button type="button" className="cell-link" onClick={() => onOpenDocument(l.documentId!)}>Open</button>}
        {l.kind === "reading" && l.readingId && <>
          <button type="button" className="cell-link" onClick={() => l.documentId && onOpenDocument(l.documentId)}>Open</button>
          <button type="button" className="btn btn-small" disabled={data.busy} onClick={() => accept(l.readingId!)}>Accept</button>
          <button type="button" className="cell-link" disabled={data.busy} onClick={() => dismiss(l.readingId!)}>Dismiss</button>
        </>}
        {l.kind === "typed" && <TypedAnswer text={d.answer ?? ""} busy={data.busy} onSave={text => void save({ answer: text }, "Saved the answer.")} />}
      </>} />
      {!(d.answer ?? "").trim() && <AddAnswer busy={data.busy} onSave={text => void save({ answer: text }, "Saved the answer.")} />}
    </section>
  </>;
}

// Timeline draws history lines: the date in a fixed column, a mark on a
// thin line, and the words with their actions.
function Timeline({ lines, render }: { lines: HistoryLine[]; render: (l: HistoryLine) => ReactNode }) {
  return <ol className="register-timeline" aria-label="History">
    {lines.map(l => <li key={l.key} data-tone={l.tone} data-kind={l.kind}>
      <span className="register-when">{l.date || "—"}</span>
      <span className="register-mark" aria-hidden="true" />
      <div className="register-what">
        <p className="register-line">{l.text}{l.detail ? <span className="register-muted"> · {l.detail}</span> : null}</p>
        {l.quote && <blockquote className="register-quote">{l.quote}</blockquote>}
        <div className="register-line-actions">{render(l)}</div>
      </div>
    </li>)}
  </ol>;
}

function TypedAnswer({ text, busy, onSave }: { text: string; busy: boolean; onSave: (text: string) => void }) {
  const [editing, setEditing] = useState(false);
  if (!editing) return <><blockquote className="register-quote register-typed">{text}</blockquote>
    <button type="button" className="cell-link" disabled={busy} onClick={() => setEditing(true)}>Edit</button></>;
  return <AnswerForm initial={text} busy={busy} onCancel={() => setEditing(false)} onSave={t => { setEditing(false); onSave(t); }} />;
}

function AddAnswer({ busy, onSave }: { busy: boolean; onSave: (text: string) => void }) {
  const [open, setOpen] = useState(false);
  if (!open) return <p className="register-add-answer"><button type="button" className="cell-link" disabled={busy} onClick={() => setOpen(true)}>Add an answer</button></p>;
  return <AnswerForm initial="" busy={busy} onCancel={() => setOpen(false)} onSave={t => { setOpen(false); onSave(t); }} />;
}

function AnswerForm({ initial, busy, onSave, onCancel }: { initial: string; busy: boolean; onSave: (text: string) => void; onCancel: () => void }) {
  const [text, setText] = useState(initial);
  return <form className="register-answer-form" onSubmit={e => { e.preventDefault(); onSave(text); }}>
    <label className="register-field">Answer<textarea aria-label="Answer" value={text} maxLength={8000} rows={3} autoFocus onChange={e => setText(e.target.value)} /></label>
    <p><button className="btn btn-small" disabled={busy || text === initial}>Save</button> <button type="button" className="cell-link" onClick={onCancel}>Cancel</button></p>
  </form>;
}

// OtherSheet is a notice, a consent condition or one of the older kinds:
// today's detail, in the sheet, with its history.
function OtherSheet({ row, data, documents, onOpenDocument, onOpenMessage }: Props & { row: DeliveryRecord }) {
  const extra: MenuItem[] = row.kind !== "rfi" && row.kind !== "notice" && row.kind !== "condition" && !RECEIVED.has(row.kind)
    ? (STATUSES[row.kind] ?? []).filter(s => s !== row.status).map(s => ({ label: `Set ${s.replaceAll("_", " ")}`, icon: null, disabled: data.busy, onSelect: () => void data.run(editOps([row], { status: s }), "Saved.", "sheet") }))
    : row.kind === "condition" ? [{ label: "Not applicable", icon: null, disabled: data.busy, onSelect: () => void data.run(editOps([row], { status: "not_applicable" }), "Marked not applicable.", "sheet") }] : [];
  const notice = row.details as { notice?: string; received_on?: string | null; message_id?: string };
  return <>
    <Head row={row} data={data} extra={extra} />
    <Notice data={data} />
    {RECEIVED.has(row.kind) ? <ClaimDetail row={row} data={data} onOpenMessage={onOpenMessage} />
      : row.kind === "condition"
      ? <ConditionDetail row={row} busy={data.busy} documents={documents} onOpenDocument={onOpenDocument}
        onLink={doc => void data.run(editOps([row], { link_evidence: doc }), "Linked the evidence.", "sheet")}
        onAccept={id => void data.run(editOps([row], { accept_reading: id }), "Linked the evidence.", "sheet")}
        onDismiss={id => void data.run([{ op: "delivery.edit", fields: { item: row.id, dismiss_readings: [id] } }], "Dismissed.", "sheet")}
        onSatisfied={() => void data.run(editOps([row], { status: "satisfied" }), "Marked satisfied.", "sheet")}
        onAddToRFP={() => void data.addToRFP(row)} />
      : <Band title="Particulars"><div className="pf-grid register-grid">
        <div className="pf-row"><span className="pf-label">With</span><span className="pf-value">{row.with || "—"}</span></div>
        <div className="pf-row"><span className="pf-label">Due</span><span className="pf-value">{row.target_date ?? "—"}</span></div>
        {row.kind === "notice" && notice.notice && <div className="pf-row"><span className="pf-label">Notice</span><span className="pf-value">{notice.notice}</span></div>}
        {row.kind === "notice" && notice.received_on && <div className="pf-row"><span className="pf-label">Received</span><span className="pf-value">{notice.received_on}</span></div>}
        {row.kind === "notice" && notice.message_id && <div className="pf-row"><span className="pf-label">Message</span><span className="pf-value">
          <button type="button" className="cell-link" onClick={() => onOpenMessage(notice.message_id!)}>Open the message</button></span></div>}
        {(row.provenance.sources ?? []).length > 0 && <div className="pf-row"><span className="pf-label">Sources</span><span className="pf-value">
          {(row.provenance.sources ?? []).map((s, i) => <button key={i} type="button" className="cell-link" onClick={() => s.document_id && onOpenDocument(s.document_id)}>{s.filename || "Document"}{s.page ? ` p. ${s.page}` : ""}</button>)}</span></div>}
      </div></Band>}
    <section className="report-section register-history-band">
      <div className="report-section-head section-band"><h3>History</h3></div>
      <Timeline lines={rowHistory(row)} render={() => null} />
    </section>
  </>;
}

// ConditionDetail is what is particular to a consent condition: the
// council's words, the page it is on, how each value was decided, its
// evidence and what can be done next.
function ConditionDetail({ row, busy, documents, onOpenDocument, onLink, onAccept, onDismiss, onSatisfied, onAddToRFP }: {
  row: DeliveryRecord; busy: boolean; documents: { id: string; title: string }[]; onOpenDocument: (id: string) => void;
  onLink: (doc: string) => void; onAccept: (reading: string) => void; onDismiss: (reading: string) => void; onSatisfied: () => void; onAddToRFP: () => void;
}) {
  const d = cond(row);
  const [link, setLink] = useState("");
  const fields = row.provenance.reading?.fields ?? {};
  const how = (field: string, label: string) => {
    const f = fields[field];
    const who = row.review_status !== "proposed" ? "you" : !f ? "not read" : f.band === "rule" ? "the notice's heading" : f.band === "amber" ? "SiteWise read it" : "not decided";
    return <span className="register-chip" data-band={f?.band === "amber" ? "amber" : undefined}>{label}: {who}</span>;
  };
  const title = (id: string) => documents.find(x => x.id === id)?.title ?? "Document deleted";
  return <Band title="Condition"><section className="register-detail" aria-label="Selected condition">
    {row.provenance.flag && <p className="register-flag" data-band="amber">{row.provenance.flag}</p>}
    {d.stage_heading && <p className="register-history">{d.stage_heading}</p>}
    <p className="register-council">{d.text || "No words were read for this condition."}</p>
    {d.source && <p><button type="button" className="cell-link" onClick={() => onOpenDocument(d.source!.document_id)}>Open page {d.source.page}</button>{d.consent_ref ? ` · ${d.consent_ref}` : ""}</p>}
    {row.review_status === "proposed" && <p className="register-chips">To check: {how("timing", "Gate")} {how("party", "Who acts")} {how("system", "System")} {how("evidence", "Evidence needed")}</p>}
    {(d.evidence ?? []).length > 0 && <p>Evidence: {(d.evidence ?? []).map(e => <button key={e.document_id} type="button" className="cell-link" onClick={() => onOpenDocument(e.document_id)}>{title(e.document_id)}</button>)}</p>}
    {(row.readings ?? []).map(rd => <div key={rd.id} className="register-reading" data-band="amber">
      <span>Possible evidence: {rd.document}{rd.basis ? ` (${rd.basis})` : ""}</span>
      <button type="button" className="cell-link" onClick={() => onOpenDocument(rd.document_id)}>Open</button>
      <button type="button" className="btn btn-small" disabled={busy} onClick={() => onAccept(rd.id)}>Link</button>
      <button type="button" className="cell-link" disabled={busy} onClick={() => onDismiss(rd.id)}>Dismiss</button>
    </div>)}
    <div className="register-pair">
      <label className="register-field">Link evidence<select value={link} disabled={busy} onChange={e => setLink(e.target.value)} aria-label="Evidence document">
        <option value="">Choose a filed document…</option>{documents.map(x => <option key={x.id} value={x.id}>{x.title}</option>)}</select></label>
      <button type="button" className="btn btn-small" disabled={busy || !link} onClick={() => { onLink(link); setLink(""); }}>Link</button>
    </div>
    <p>{row.status !== "satisfied" && <button type="button" className="btn btn-small" disabled={busy} onClick={onSatisfied}>Satisfied</button>}{" "}
      <button type="button" className="cell-link" disabled={busy || !row.package_id || !!d.rfp} title={!row.package_id ? "Give it to a package first" : d.rfp ? "Already on the RFP" : undefined} onClick={onAddToRFP}>{d.rfp ? "On the RFP" : "Add to RFP"}</button></p>
  </section></Band>;
}
