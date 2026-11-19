import { useEffect, useState } from "react";
import { ApiError, request } from "./api";
import { registerDialog } from "./chatView";
import { ConfirmDialog } from "./ConfirmDialog";
import { transmittalNumber, type MailMessage } from "./mailApi";

// Send… and Reply: the one way mail leaves SiteWise
// (docs/plans/2026-11-17-correspondence-transmittals.md, Task 6; the wave
// index, agreement 7). The PM presses Send; chat only opens this filled in.
// A numbered transmittal sheet goes with the documents.

/** What Send was opened with. Every id was checked by Go when chat sent it. */
export interface SendFields {
  /** Comma-separated document ids. */
  documents?: string;
  to?: string;
  cc?: string;
  subject?: string;
  note?: string;
  /** The message this replies to. */
  reply_to?: string;
  /** An issued version of the project report, sent as issued. */
  issue?: string;
  /** The register's RFI this message sends. */
  rfi?: string;
  /** Set by Go when chat filled this in after reading outside text. */
  warning?: string;
}

export interface SendDocument { id: string; name: string; size?: number }

const MAX_BYTES = 10 << 20;

function sizeWords(bytes: number) {
  if (bytes >= 1 << 20) return `${(bytes / (1 << 20)).toFixed(1)} MB`;
  return `${Math.max(1, Math.round(bytes / 1024))} KB`;
}

const split = (s?: string) => (s ?? "").split(",").map(x => x.trim()).filter(Boolean);

export function SendDialog({ projectId, projectName, fields, documentOf, onSent, onClose }: {
  projectId: string;
  projectName: string;
  fields: SendFields;
  documentOf: (id: string) => SendDocument | undefined;
  onSent: (m: MailMessage) => void;
  onClose: () => void;
}) {
  const [to, setTo] = useState(fields.to ?? "");
  const [cc, setCc] = useState(fields.cc ?? "");
  const [subject, setSubject] = useState(fields.subject ?? `${projectName} transmittal`);
  const [note, setNote] = useState(fields.note ?? "");
  const [docs, setDocs] = useState<string[]>(split(fields.documents));
  const [firms, setFirms] = useState<{ name: string; email: string }[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    request<{ firms?: { name: string; email: string }[] } | { name: string; email: string }[]>("GET", "/firms")
      .then(body => setFirms((Array.isArray(body) ? body : body.firms ?? []).filter(f => f.email)))
      .catch(() => setFirms([]));
  }, []);

  const known = docs.map(id => documentOf(id) ?? { id, name: "Document" });
  const total = known.reduce((n, d) => n + (d.size ?? 0), 0);
  const recipients = split(to);
  const tooBig = total > MAX_BYTES;
  const send = async () => {
    setBusy(true); setError("");
    try {
      const body: Record<string, unknown> = { to: recipients, cc: split(cc), subject: subject.trim(), note, document_ids: docs };
      if (fields.reply_to) body.in_reply_to = fields.reply_to;
      if (fields.issue) body.issue = fields.issue;
      if (fields.rfi) body.rfi = fields.rfi;
      const m = await request<MailMessage>("POST", `/projects/${projectId}/transmittals`, body);
      onSent(m);
      onClose();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "The message could not be sent.");
    } finally { setBusy(false); }
  };
  return (
    <ConfirmDialog tone="plain" title={fields.reply_to ? "Reply" : "Send"}
      message="A numbered transmittal sheet goes with the documents. The message is kept in Messages and can't be edited or deleted once sent."
      confirmLabel="Send" busy={busy} busyLabel="Sending…" confirmDisabled={recipients.length === 0 || tooBig}
      onConfirm={() => void send()} onCancel={onClose}>
      <div className="send-dialog">
        {fields.warning && <p className="send-error" role="note">{fields.warning}</p>}
        <label htmlFor="send-to">To</label>
        <input id="send-to" className="input" list="send-firms" value={to} placeholder="name@example.com, another@example.com" onChange={e => setTo(e.target.value)} />
        <label htmlFor="send-cc">Cc</label>
        <input id="send-cc" className="input" list="send-firms" value={cc} placeholder="You are copied in" onChange={e => setCc(e.target.value)} />
        <datalist id="send-firms">{firms.map(f => <option key={f.email} value={`${f.name} <${f.email}>`} />)}</datalist>
        <label htmlFor="send-subject">Subject</label>
        <input id="send-subject" className="input" maxLength={300} value={subject} onChange={e => setSubject(e.target.value)} />
        <label htmlFor="send-note">Note</label>
        <textarea id="send-note" className="input" rows={4} maxLength={10000} value={note} onChange={e => setNote(e.target.value)} />
        {fields.issue && <p className="send-docs-head">The issued report goes as it was issued.</p>}
        {known.length > 0 && <>
          <p className="send-docs-head">{known.length} document{known.length === 1 ? "" : "s"}{total ? ` · ${sizeWords(total)}` : ""}</p>
          <ul className="send-docs">{known.map(d => <li key={d.id}>{d.name}
            <button type="button" className="cell-link" aria-label={`Leave out ${d.name}`} onClick={() => setDocs(ds => ds.filter(x => x !== d.id))}>Leave out</button></li>)}</ul>
        </>}
        {tooBig && <p className="send-error" role="alert">These come to {sizeWords(total)}, more than the 10 MB a message can carry. Send fewer at a time.</p>}
        {error && <p className="send-error" role="alert">{error}</p>}
      </div>
    </ConfirmDialog>
  );
}

/** SendHost holds the one Send dialog of a project. Screens and chat open it
 * through the chat view registry under the name "send". */
export function SendHost({ projectId, projectName, documentOf, onSent }: {
  projectId: string;
  projectName: string;
  documentOf: (id: string) => SendDocument | undefined;
  onSent: (m: MailMessage) => void;
}) {
  const [fields, setFields] = useState<SendFields | null>(null);
  useEffect(() => registerDialog("send", { open: f => setFields({ ...f }) }), []);
  if (!fields) return null;
  return <SendDialog projectId={projectId} projectName={projectName} fields={fields} documentOf={documentOf} onSent={onSent} onClose={() => setFields(null)} />;
}

/** Words for a sent message's state, as the list shows it. */
export function sendState(m: MailMessage) {
  if (m.send_status === "queued") return `${transmittalNumber(m.number)} sending…`;
  if (m.send_status === "failed") return "Not sent";
  return "";
}
