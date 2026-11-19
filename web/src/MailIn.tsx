import { useEffect, useMemo, useState, type ReactNode } from "react";
import { ApiError, request } from "./api";
import { dialogOf, registerHost, showingLine, useChatTable } from "./chatView";
import { ConfirmDialog } from "./ConfirmDialog";
import { HELD_WORDS, NOTICE_WORDS, readMail, readMessage, shortDay, transmittalNumber, type MailList, type MailMessage } from "./mailApi";
import { packageApi } from "./packageApi";
import { invoiceApi, uploadInvoice, type Contract } from "./invoiceApi";
import { sendState } from "./SendDialog";
import { clickRow, emptySelection, selectAll, type Selection } from "./table/selection";
import "./mail.css";

// The repository's head and its Messages list (email intake Task 9;
// correspondence Task 6). The address line shows only when email intake is
// on; the Documents | Messages toggle only once the project has a message
// (the wave's rule 5). A message is a record: nothing here edits or deletes
// one that is filed or sent. Bodies are shown as text, never as HTML.

/**
 * The chip's buttons (agreement 9). "Add to register" posts a notice row;
 * the other two are plan 3's, through its own routes, each started only by
 * the PM's click: Read as progress claim sends a filed attachment's bytes to
 * POST /invoices as the picked contract's progress claim (the one claim
 * fan-out already budgeted); Add variation posts the variation with the
 * message's body document linked.
 */
export type ChipAction = "register" | "claim" | "variation";
export function chipAction(pick: string): ChipAction {
  if (pick === "payment_claim") return "claim";
  if (pick === "variation_claim") return "variation";
  return "register";
}
/** Plan 3: read a filed attachment as the contract's progress claim. */
export async function readAsProgressClaim(projectId: string, documentId: string, filename: string, contractPackage: string): Promise<void> {
  const res = await fetch(`/api/documents/${documentId}/file`, { credentials: "same-origin" });
  if (!res.ok) throw new ApiError(res.status, "The attachment could not be read. Open it and try again.");
  const file = new File([await res.blob()], filename);
  await uploadInvoice(projectId, file, () => undefined, contractPackage).done;
}
/** Plan 3: add a variation on a contract line with the message's body linked. */
export async function addVariation(projectId: string, m: MailMessage, contractPackage: string, line: string): Promise<void> {
  const body = m.parts.find(p => p.n === 0)?.document_id;
  await request("POST", `/projects/${projectId}/variations`, { package: contractPackage, line, title: m.subject || "Variation", link_documents: body ? [body] : [] });
}
export const CHIP_ACTIONS: Record<ChipAction, boolean> = { register: true, claim: true, variation: true };

const CHIP_LABEL: Record<ChipAction, string> = { register: "Add to register", claim: "Read as progress claim", variation: "Add variation" };

interface Props {
  projectId: string;
  /** Moves when a mail event arrives. */
  tick: number;
  documents: ReactNode;
  onOpenDocument: (id: string) => boolean;
}

export function MailRepository({ projectId, tick, documents, onOpenDocument }: Props) {
  const [list, setList] = useState<MailList | null>(null);
  const [tab, setTab] = useState<"documents" | "messages">("documents");
  const [copied, setCopied] = useState(false);
  const [rotate, setRotate] = useState(false);
  const [notice, setNotice] = useState("");
  const load = () => readMail(projectId).then(setList).catch(() => undefined);
  useEffect(() => { void load(); }, [projectId, tick]);
  useEffect(() => registerHost("messages", () => setTab("messages")), []);
  const hasMessages = (list?.messages.length ?? 0) > 0;
  const copy = async () => {
    try { await navigator.clipboard.writeText(list?.address ?? ""); setCopied(true); setTimeout(() => setCopied(false), 1500); } catch { setCopied(false); }
  };
  const doRotate = async () => {
    try { await request("POST", `/projects/${projectId}/mail/address`); setRotate(false); setNotice("The project has a new address."); await load(); }
    catch (e) { setNotice(e instanceof ApiError ? e.message : "The address could not be changed."); setRotate(false); }
  };
  return (
    <div className="mail-repository">
      {(list?.address || hasMessages) && <div className="mail-head">
        {list?.address && <p className="mail-address">
          Email documents to <code>{list.address}</code>{" "}
          <button type="button" className="cell-link" onClick={() => void copy()}>{copied ? "Copied" : "Copy"}</button>{" · "}
          <button type="button" className="cell-link" onClick={() => setRotate(true)}>New address…</button>
          {list.held > 0 && <>{" · "}<button type="button" className="cell-link" onClick={() => setTab("messages")}>{list.held} held</button></>}
        </p>}
        {hasMessages && <div className="mail-toggle" role="tablist" aria-label="Repository">
          <button type="button" role="tab" aria-selected={tab === "documents"} onClick={() => setTab("documents")}>Documents</button>
          <button type="button" role="tab" aria-selected={tab === "messages"} onClick={() => setTab("messages")}>Messages</button>
        </div>}
        {notice && <p className="reg-notice" role="status">{notice}</p>}
      </div>}
      {tab === "messages" && hasMessages ? <Messages projectId={projectId} list={list!} reload={load} onOpenDocument={onOpenDocument} /> : documents}
      {rotate && <ConfirmDialog title="Give the project a new address" message="Replies to messages sent from the old address will not arrive. Mail to the old address stops at once." confirmLabel="New address" busy={false} onConfirm={() => void doRotate()} onCancel={() => setRotate(false)} />}
    </div>
  );
}

/** A thread's rows: its first message, then the replies under it. */
function threads(messages: MailMessage[]): { m: MailMessage; reply: boolean }[] {
  const by = new Map<string, MailMessage[]>();
  for (const m of messages) by.set(m.thread_id, [...(by.get(m.thread_id) ?? []), m]);
  const groups = [...by.values()].map(g => g.sort((a, b) => a.received_at.localeCompare(b.received_at)));
  groups.sort((a, b) => b[b.length - 1].received_at.localeCompare(a[a.length - 1].received_at));
  return groups.flatMap(g => g.map((m, i) => ({ m, reply: i > 0 })));
}

function Messages({ projectId, list, reload, onOpenDocument }: { projectId: string; list: MailList; reload: () => Promise<void>; onOpenDocument: (id: string) => boolean }) {
  const [open, setOpen] = useState<string | null>(null);
  const [selection, setSelection] = useState<Selection>(emptySelection);
  const [only, setOnly] = useState<{ ids: Set<string>; shown?: string } | null>(null);
  const [packages, setPackages] = useState<{ id: string; title: string }[]>([]);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const [discard, setDiscard] = useState<MailMessage | null>(null);
  useEffect(() => {
    packageApi.list(projectId).then(o => setPackages(o.items.filter(p => !p.retired_at).map(p => ({ id: p.id, title: p.title })))).catch(() => setPackages([]));
  }, [projectId]);
  const rows = useMemo(() => threads(list.messages).filter(r => !only || only.ids.has(r.m.id)), [list, only]);
  const ids = rows.map(r => r.m.id);
  useChatTable("messages", "documents", {
    select: s => setSelection(selectAll(s.filter(id => list.messages.some(m => m.id === id)))),
    only: (s, shown) => setOnly({ ids: new Set(s), shown }),
    clear: () => { setOnly(null); setSelection(emptySelection); },
    open: id => setOpen(id),
  }, selection.ids, true);
  const act = async (fn: () => Promise<unknown>, done: string) => {
    setBusy(true); setNotice("");
    try { await fn(); setNotice(done); await reload(); }
    catch (e) { setNotice(e instanceof ApiError ? e.message : "That could not be done. Try again."); }
    finally { setBusy(false); }
  };
  const patch = (m: MailMessage, body: Record<string, unknown>) => request("PATCH", `/projects/${projectId}/mail/${m.id}`, { version: m.version, ...body });
  const addToRegister = (m: MailMessage) => act(() => request("POST", `/projects/${projectId}/delivery`, {
    kind: "notice", title: m.subject || "Notice", status: "open",
    details: { message_id: m.id, notice: NOTICE_WORDS[m.notice ?? ""] ?? "", received_on: m.received_at.slice(0, 10) },
  }), "Added to the register.");
  const [pick, setPick] = useState<{ m: MailMessage; action: "claim" | "variation" } | null>(null);
  const runChip = (m: MailMessage, a: ChipAction = chipAction(m.notice ?? "")) => {
    if (a === "register") return addToRegister(m);
    setPick({ m, action: a });
  };
  return (
    <div className="mail-messages">
      <p className="reg-notice" role="status">{notice}</p>
      {only && <p className="reg-filter">{showingLine("messages", only.ids.size, only.shown)} · <button type="button" className="cell-link" onClick={() => setOnly(null)}>Clear</button></p>}
      <table className="reg-table mail-table" aria-label="Messages. Click to open.">
        <colgroup><col className="mail-dir" /><col className="mail-no" /><col className="mail-date" /><col className="mail-with" /><col className="mail-subject" /><col className="mail-pkg" /></colgroup>
        <thead><tr><th scope="col"><span className="sr-only">In or out</span></th><th scope="col">No.</th><th scope="col">Date</th><th scope="col">With</th><th scope="col">Subject</th><th scope="col">Package</th></tr></thead>
        {rows.map(({ m, reply }) => {
          const chosen = selection.ids.includes(m.id);
          const action = m.notice ? chipAction(m.notice) : null;
          return <tbody key={m.id} className="reg-doc mail-row" data-selected={chosen ? "true" : undefined} data-open={open === m.id ? "true" : undefined} data-reply={reply ? "true" : undefined} data-held={m.status === "held" ? "true" : undefined}>
            <tr className="reg-line" tabIndex={0} onClick={e => { setSelection(clickRow(selection, m.id, ids, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })); setOpen(o => o === m.id ? null : m.id); }}
              onKeyDown={e => { if (e.key === "Enter") setOpen(o => o === m.id ? null : m.id); }}>
              <td className="mail-dir">{m.direction === "out" ? "Out" : "In"}</td>
              <td className="mail-no">{transmittalNumber(m.number)}</td>
              <td className="mail-date">{shortDay(m.received_at)}</td>
              <td className="mail-with" title={m.with}>{m.firm || m.sender_name || m.with}</td>
              <td className="mail-subject" title={m.subject}>
                <span>{m.subject || "(no subject)"}</span>
                {m.replied && <span className="mail-mark">Replied</span>}
                {m.status === "held" && <span className="mail-mark" data-tone="held">Held · {HELD_WORDS[m.reason] ?? m.reason}</span>}
                {sendState(m) && <span className="mail-mark" data-tone={m.send_status === "failed" ? "alert" : undefined}>{sendState(m)}</span>}
                {action && CHIP_ACTIONS[action] && <span className="mail-chip" onClick={e => e.stopPropagation()}>
                  Possible {NOTICE_WORDS[m.notice!] ?? "notice"} · to check{" "}
                  <button type="button" className="btn btn-small" disabled={busy} onClick={() => void runChip(m)}>{CHIP_LABEL[action]}</button>{" "}
                  <button type="button" className="cell-link" disabled={busy} onClick={() => void act(() => patch(m, { pick_state: "dismissed" }), "Dismissed.")}>Not one</button>
                </span>}
              </td>
              <td className="mail-pkg" onClick={e => e.stopPropagation()}>
                {m.status === "filed" ? <select className="select mail-pkg-select" aria-label={`Package of ${m.subject}`} disabled={busy} value={m.package_id ?? ""}
                  onChange={e => void act(() => patch(m, { package_id: e.target.value }), "Saved.")}>
                  <option value="">—</option>
                  {packages.map(p => <option key={p.id} value={p.id}>{p.title}</option>)}
                </select> : null}
              </td>
            </tr>
            {open === m.id && <tr className="mail-open"><td colSpan={6}><MessageBody projectId={projectId} message={m} busy={busy} onOpenDocument={onOpenDocument}
              onAccept={() => void act(() => request("POST", `/projects/${projectId}/mail/${m.id}/accept`), "Accepted. Its files are filing.")}
              onDiscard={() => setDiscard(m)}
              onRetry={() => void act(() => request("POST", `/projects/${projectId}/transmittals`, { retry: m.id }), "Sending again.")}
              onAddToRegister={() => void addToRegister(m)} onAction={a => void runChip(m, a)} /></td></tr>}
          </tbody>;
        })}
      </table>
      {pick && <ContractPicker projectId={projectId} message={pick.m} action={pick.action} onClose={() => setPick(null)}
        onDone={words => { setPick(null); setNotice(words); void reload(); }} />}
      {discard && <ConfirmDialog title="Discard held message" message={`Discard the message from ${discard.sender} and its files? It can't be put back.`} confirmLabel="Discard" busy={busy}
        onConfirm={() => { const m = discard; setDiscard(null); void act(() => request("POST", `/projects/${projectId}/mail/${m.id}/discard`), "Discarded."); }} onCancel={() => setDiscard(null)} />}
    </div>
  );
}

function MessageBody({ projectId, message, busy, onOpenDocument, onAccept, onDiscard, onRetry, onAddToRegister, onAction }: {
  projectId: string; message: MailMessage; busy: boolean; onOpenDocument: (id: string) => boolean;
  onAccept: () => void; onDiscard: () => void; onRetry: () => void; onAddToRegister: () => void; onAction: (a: ChipAction) => void;
}) {
  const [m, setM] = useState<MailMessage>(message);
  useEffect(() => { readMessage(projectId, message.id).then(setM).catch(() => undefined); }, [projectId, message.id, message.version, message.send_status, message.status]);
  const files = m.parts.filter(p => p.n > 0 || m.direction === "out");
  const reply = () => dialogOf("send")?.open({ reply_to: m.id, to: m.direction === "in" ? m.sender : m.to, subject: m.subject.toLowerCase().startsWith("re:") ? m.subject : `Re: ${m.subject}` });
  return (
    <div className="mail-body">
      <p className="mail-meta">{m.direction === "out" ? `To ${m.to}` : `From ${m.sender_name ? `${m.sender_name} <${m.sender}>` : m.sender}`}{m.cc ? ` · Cc ${m.cc}` : ""}</p>
      <pre className="mail-text">{m.body ?? ""}</pre>
      {files.length > 0 && <ul className="mail-files">{files.map(p => <li key={p.n}>
        {p.document_id ? <button type="button" className="cell-link" onClick={() => { if (!onOpenDocument(p.document_id!)) window.open(`/api/projects/${projectId}/mail/${m.id}/parts/${p.n}`); }}>{p.title || p.filename}</button>
          : p.skipped ? <span className="mail-skipped">{p.filename} (not kept)</span>
            : <a href={`/api/projects/${projectId}/mail/${m.id}/parts/${p.n}`}>{p.filename}</a>}
        {p.revision ? <span className="mail-skipped"> · rev {p.revision}</span> : null}
      </li>)}</ul>}
      {m.send_status === "failed" && <p className="send-error">Not sent: {m.send_error}</p>}
      <p className="mail-actions">
        {m.status === "held" ? <>
          <button type="button" className="btn btn-small" disabled={busy || m.reason === "held_full"} onClick={onAccept}>Accept</button>{" "}
          <button type="button" className="btn btn-small" disabled={busy} onClick={onDiscard}>Discard</button>
        </> : <>
          <button type="button" className="btn btn-small" onClick={reply}>Reply</button>{" "}
          {m.send_status === "failed" && <button type="button" className="btn btn-small" disabled={busy} onClick={onRetry}>Retry</button>}{" "}
          {/* The message's own actions, with or without a chip. */}
          {m.direction === "in" && <>
            <button type="button" className="cell-link" disabled={busy} onClick={onAddToRegister}>Add to register</button>{" · "}
            <button type="button" className="cell-link" disabled={busy || !m.parts.some(p => p.n > 0 && p.document_id)} onClick={() => onAction("claim")}>Read as progress claim…</button>{" · "}
            <button type="button" className="cell-link" disabled={busy} onClick={() => onAction("variation")}>Add variation…</button>
          </>}
        </>}
      </p>
    </div>
  );
}

/** Picks the contract (and the line, or the attachment) for plan 3's two
 * actions. Nothing happens until the PM presses the button. */
function ContractPicker({ projectId, message, action, onClose, onDone }: {
  projectId: string; message: MailMessage; action: "claim" | "variation"; onClose: () => void; onDone: (words: string) => void;
}) {
  const [contracts, setContracts] = useState<Contract[] | null>(null);
  const [m, setM] = useState<MailMessage>(message);
  const [pkg, setPkg] = useState("");
  const [line, setLine] = useState("");
  const [file, setFile] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    invoiceApi.ledger(projectId).then(l => setContracts(l.contracts ?? [])).catch(() => setContracts([]));
    readMessage(projectId, message.id).then(setM).catch(() => undefined);
  }, [projectId, message.id]);
  const files = m.parts.filter(p => p.n > 0 && p.document_id);
  const contract = contracts?.find(c => c.package_id === pkg);
  const ready = !!pkg && (action === "claim" ? !!file : !!line || (contract?.lines.length ?? 0) === 1);
  const go = async () => {
    setBusy(true); setError("");
    try {
      if (action === "claim") {
        const part = files.find(p => p.document_id === file)!;
        await readAsProgressClaim(projectId, file, part.filename, pkg);
        onDone("Reading it as a progress claim.");
      } else {
        await addVariation(projectId, m, pkg, line || contract?.lines[0]?.id || "");
        onDone("Variation added.");
      }
    } catch (e) { setError(e instanceof ApiError ? e.message : "That could not be done."); }
    finally { setBusy(false); }
  };
  return (
    <ConfirmDialog tone="plain" title={action === "claim" ? "Read as progress claim" : "Add variation"}
      message={action === "claim" ? "The attachment is read as this contract's progress claim." : "The variation is added to this contract with the message linked."}
      confirmLabel={action === "claim" ? "Read" : "Add"} busy={busy} busyLabel="Working…" confirmDisabled={!ready} onConfirm={() => void go()} onCancel={onClose}>
      <div className="send-dialog">
        {contracts && contracts.length === 0 && <p className="send-error">There is no live contract to put it on.</p>}
        <label htmlFor="pick-contract">Contract</label>
        <select id="pick-contract" className="select" value={pkg} onChange={e => { setPkg(e.target.value); setLine(""); }}>
          <option value="" disabled>Choose…</option>
          {(contracts ?? []).map(c => <option key={c.package_id} value={c.package_id}>{c.title}{c.firm ? ` · ${c.firm}` : ""}</option>)}
        </select>
        {action === "variation" && contract && contract.lines.length > 1 && <>
          <label htmlFor="pick-line">Contract line</label>
          <select id="pick-line" className="select" value={line} onChange={e => setLine(e.target.value)}>
            <option value="" disabled>Choose…</option>
            {contract.lines.map(l => <option key={l.id} value={l.id}>{l.label}</option>)}
          </select>
        </>}
        {action === "claim" && <>
          <label htmlFor="pick-file">Attachment</label>
          <select id="pick-file" className="select" value={file} onChange={e => setFile(e.target.value)}>
            <option value="" disabled>Choose…</option>
            {files.map(p => <option key={p.n} value={p.document_id}>{p.filename}</option>)}
          </select>
        </>}
        {error && <p className="send-error" role="alert">{error}</p>}
      </div>
    </ConfirmDialog>
  );
}
