import { useEffect, useState } from "react";
import { ApiError, request } from "./api";
import { registerDialog } from "./chatView";
import { ConfirmDialog } from "./ConfirmDialog";

// Share: a link for one person outside the org to one thing — documents, an
// issued report, or an issued RFP or RFT for one invited firm
// (docs/plans/2026-11-17-guest-access.md, Task 8). The PM presses Create
// link; chat only opens this dialog filled in. The link is shown once and
// copied into the PM's own email: SiteWise sends nothing.

/** What Share was opened on. Every id was checked by Go when chat sent it. */
export interface ShareFields {
  kind?: string;
  /** Comma-separated document ids. */
  documents?: string;
  report?: string;
  package?: string;
  firm?: string;
  firm_name?: string;
  /** What the thing is called, for the title. */
  title?: string;
  shared_with?: string;
  days?: string;
}

export interface ShareView {
  id: string;
  kind: string;
  shared_with: string;
  shared_by: string;
  what: string;
  opens: number;
  last_opened_on?: string;
  expires_on: string;
  revoked_at?: string;
  live: boolean;
}

interface SharesBody { shares: ShareView[]; budget_disclosed: boolean; issued: boolean; suggest_to?: string }

const DAY = 86_400_000;

function isoDay(d: Date) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

export function shownDate(iso?: string) {
  if (!iso) return "";
  return new Date(`${iso.slice(0, 10)}T00:00:00`).toLocaleDateString("en-AU", { day: "numeric", month: "short", year: "numeric" });
}

function listQuery(f: ShareFields) {
  const q = new URLSearchParams({ kind: f.kind ?? "" });
  const docs = (f.documents ?? "").split(",").filter(Boolean);
  if (f.kind === "documents" && docs.length === 1) q.set("document", docs[0]);
  if (f.report) q.set("report", f.report);
  if (f.package) q.set("package", f.package);
  if (f.firm) q.set("firm", f.firm);
  return q.toString();
}

function whatOf(f: ShareFields) {
  const docs = (f.documents ?? "").split(",").filter(Boolean);
  if (f.title) return f.title;
  if (f.kind === "documents") return docs.length === 1 ? "1 document" : `${docs.length} documents`;
  if (f.kind === "package") return `the RFP or RFT${f.firm_name ? ` with ${f.firm_name}` : ""}`;
  return "this report";
}

export function ShareDialog({ projectId, fields, onClose }: { projectId: string; fields: ShareFields; onClose: () => void }) {
  const base = `/projects/${projectId}/shares`;
  const life = Number(fields.days) || (fields.kind === "package" ? 21 : 30);
  const today = new Date();
  const [to, setTo] = useState(fields.shared_with ?? "");
  const [until, setUntil] = useState(isoDay(new Date(today.getTime() + life * DAY)));
  const [list, setList] = useState<SharesBody | null>(null);
  const [link, setLink] = useState("");
  const [copied, setCopied] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = async () => {
    try {
      const body = await request<SharesBody>("GET", `${base}?${listQuery(fields)}`);
      setList(body);
      if (!fields.shared_with && body.suggest_to) setTo(t => t || body.suggest_to!);
    } catch { setError("The links could not be read. Close and try again."); }
  };
  useEffect(() => { void load(); }, [projectId, fields]);

  const days = Math.round((new Date(`${until}T00:00:00`).getTime() - new Date(`${isoDay(today)}T00:00:00`).getTime()) / DAY);
  const create = async () => {
    if (link) { onClose(); return; }
    setBusy(true); setError("");
    try {
      const body: Record<string, unknown> = { kind: fields.kind, shared_with: to.trim(), days };
      if (fields.kind === "documents") body.document_ids = (fields.documents ?? "").split(",").filter(Boolean);
      if (fields.kind === "report") body.report_id = fields.report;
      if (fields.kind === "package") { body.package_id = fields.package; body.firm_id = fields.firm; }
      const out = await request<{ share: ShareView; link: string }>("POST", base, body);
      setLink(out.link);
      await load();
    } catch (e) {
      setError(e instanceof ApiError && e.status === 404 ? "This can't be shared: it is not filed, not issued, or not on this project." : e instanceof Error ? e.message : "The link could not be made.");
    } finally { setBusy(false); }
  };
  const revoke = async (s: ShareView) => {
    setBusy(true); setError("");
    try { await request("DELETE", `${base}/${s.id}`); await load(); }
    catch { setError("The link could not be withdrawn. Try again."); }
    finally { setBusy(false); }
  };
  const copy = async () => {
    try { await navigator.clipboard.writeText(link); setCopied(true); } catch { setCopied(false); }
  };

  const unissued = list && !list.issued && fields.kind !== "documents";
  const valid = to.trim().length > 0 && days >= 1 && days <= 180 && !unissued;
  return (
    <ConfirmDialog tone="plain" title={`Share ${whatOf(fields)}`}
      message="Whoever holds the link sees only this, until the date below. Copy the link into your own email."
      confirmLabel={link ? "Done" : "Create link"} busy={busy} busyLabel="Working…" confirmDisabled={!link && !valid}
      onConfirm={() => void create()} onCancel={onClose}>
      <div className="share-dialog">
        {list?.budget_disclosed && <p className="share-warning" role="note">The newest issue shows the budget. Anyone with this link will see it.</p>}
        {unissued && <p className="share-warning" role="note">{fields.kind === "package" ? "Issue the RFT first." : "Issue the report first."}</p>}
        {link ? <div className="share-link">
          <label htmlFor="share-link">Link (shown once)</label>
          <div className="share-link-row">
            <input id="share-link" className="input" readOnly value={link} onFocus={e => e.currentTarget.select()} />
            <button type="button" className="btn btn-small" onClick={() => void copy()}>{copied ? "Copied" : "Copy"}</button>
          </div>
        </div> : <div className="share-fields">
          <label htmlFor="share-to">Shared with</label>
          <input id="share-to" className="input" type="text" maxLength={254} value={to} placeholder="name@example.com" onChange={e => setTo(e.target.value)} />
          <label htmlFor="share-until">Link works until</label>
          <input id="share-until" className="input" type="date" value={until} min={isoDay(new Date(today.getTime() + DAY))} max={isoDay(new Date(today.getTime() + 180 * DAY))} onChange={e => setUntil(e.target.value)} />
        </div>}
        {error && <p className="share-error" role="alert">{error}</p>}
        {!!list?.shares.length && <table className="share-list" aria-label="Who has a link">
          <thead><tr><th scope="col">Shared with</th><th scope="col">Opened</th><th scope="col">Works until</th><th scope="col"><span className="sr-only">Revoke</span></th></tr></thead>
          <tbody>{list.shares.map(s => <tr key={s.id} data-live={s.live ? undefined : "false"}>
            <td>{s.shared_with}<span className="share-by">by {s.shared_by}</span></td>
            <td>{s.opens ? `${s.opens} time${s.opens === 1 ? "" : "s"}, last ${shownDate(s.last_opened_on)}` : "Not opened"}</td>
            <td>{s.revoked_at ? "Withdrawn" : s.live ? shownDate(s.expires_on) : "Ended"}</td>
            <td>{s.live && <button type="button" className="btn btn-small" disabled={busy} aria-label={`Revoke the link shared with ${s.shared_with}`} onClick={() => void revoke(s)}>Revoke</button>}</td>
          </tr>)}</tbody>
        </table>}
      </div>
    </ConfirmDialog>
  );
}

/** ShareHost holds the one Share dialog of a project. Screens and chat open
 * it through the chat view registry under the name "share". */
export function ShareHost({ projectId }: { projectId: string }) {
  const [fields, setFields] = useState<ShareFields | null>(null);
  useEffect(() => registerDialog("share", { open: f => setFields({ ...f }) }), []);
  if (!fields) return null;
  return <ShareDialog projectId={projectId} fields={fields} onClose={() => setFields(null)} />;
}
