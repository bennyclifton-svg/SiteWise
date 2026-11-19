// The register: what is owed on the project, in one table
// (docs/plans/2026-11-17-rfis-review-comments.md, Task 5). Every kind
// shares six columns; what is particular to a kind sits in the strip under
// the table. Numbers, due dates, days late and With are Go's. Every write
// goes through POST /ops (area delivery), so each action has one Undo. No
// screen read or write here calls Jev or the model.

import { Fragment, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Pending } from "./Activity";
import { ApiError, request } from "./api";
import { dialogOf, useChatTable, showingLine, revealQuietly } from "./chatView";
import { deliveryApi, GATES, rfiNumber, sydneyToday, type ConditionDetails, type DeliveryRecord, type OpCall, type RegisterView } from "./deliveryApi";
import { IconMore } from "./icons";
import type { WorkbenchView } from "./LeftNav";
import { MoreMenu } from "./PageActions";
import { ProjectReportButton, useProjectReportDialog } from "./ProjectReportButton";
import { Reports } from "./Reports";
import { InlineCellEditor, type CommitKey } from "./table/InlineCellEditor";
import { RowMenu, type MenuItem, type RowMenuAt } from "./table/RowMenu";
import { clickRow, emptySelection, prune, selectAll, type Selection } from "./table/selection";
import "./register.css";

const KIND: Record<string, [string, string]> = {
  rfi: ["RFI", "RFIs"], notice: ["Notice", "Notices"], action: ["Action", "Actions"], risk: ["Risk", "Risks"], issue: ["Issue", "Issues"],
  decision: ["Decision", "Decisions"], approval: ["Approval", "Approvals"], milestone: ["Hold point", "Hold points"],
  condition: ["Condition", "Conditions"],
};
const STATUSES: Record<string, string[]> = {
  rfi: ["draft", "open", "answered", "closed", "withdrawn"], notice: ["open", "closed", "withdrawn"],
  action: ["not_started", "in_progress", "blocked", "complete", "cancelled"], risk: ["open", "mitigating", "closed"],
  issue: ["open", "in_progress", "resolved", "closed"], decision: ["open", "decided", "superseded"],
  approval: ["not_submitted", "submitted", "approved", "rejected", "withdrawn"], milestone: ["planned", "achieved", "cancelled"],
  condition: ["open", "evidence_filed", "satisfied", "not_applicable"],
};
const CLOSED = new Set(["closed", "withdrawn", "complete", "cancelled", "resolved", "decided", "superseded", "approved", "rejected", "achieved", "satisfied", "not_applicable"]);
const words = (s: string) => s.replaceAll("_", " ");
// Consent conditions (docs/plans/2026-11-17-consent-conditions.md, Task 9):
// a kind in this one table, grouped by gate, in the shared columns.
const GATE_WORDS: Record<string, string> = Object.fromEntries(GATES);
const GATE_RANK: Record<string, number> = Object.fromEntries(GATES.map(([g], i) => [g, i]));
const cond = (r: DeliveryRecord) => r.details as ConditionDetails;
/** A condition's status in words: what is needed next. */
function conditionStatus(r: DeliveryRecord): string {
  const d = cond(r);
  if (r.status === "open" && (r.readings ?? []).length > 0) return "Possible evidence · Link";
  if (r.status === "open") return d.evidence_needed ? "Open · evidence needed" : "Open";
  return ({ evidence_filed: "Evidence filed", satisfied: "Satisfied", not_applicable: "Not applicable" } as Record<string, string>)[r.status] ?? words(r.status);
}
/** An operation's refusal in its own words. */
function opMessage(raw: string): string {
  try { const m = (JSON.parse(raw) as { message?: string }).message; if (m) return m; } catch { /* plain text */ }
  return raw;
}
/** Conditions in gate order, then by their printed number. */
const conditionOrder = (a: DeliveryRecord, b: DeliveryRecord) =>
  (GATE_RANK[cond(a).stage ?? ""] ?? 99) - (GATE_RANK[cond(b).stage ?? ""] ?? 99) || (cond(a).number ?? "").localeCompare(cond(b).number ?? "", undefined, { numeric: true });
type Field = "title" | "with" | "due" | "system" | "status";
interface Edit { id: string; field: Field; many?: string[]; error?: string }

export function Delivery({ projectId, tick, readerTick = 0, raise, onSignedOut, onShowView, onOpenDocument, documents = [] }: {
  projectId: string; tick: number; readerTick?: number; raise: { documentId: string; title: string; n: number } | null;
  onSignedOut: () => void; onShowView: (view: WorkbenchView) => void; onOpenDocument: (id: string) => void;
  /** Filed documents, for Read conditions and Link evidence (statutory instruments first). */
  documents?: { id: string; title: string; kind: string }[];
}) {
  // The project report opens here in the shared reader, from the toolbar or
  // from chat (monthly plan, Task 7).
  const [report, setReport] = useState<string | null>(null);
  useProjectReportDialog(setReport);
  const [view, setView] = useState<RegisterView | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [reload, setReload] = useState(0);
  const [kind, setKind] = useState("");
  const [system, setSystem] = useState("");
  const [showClosed, setShowClosed] = useState(false);
  const [only, setOnly] = useState<{ ids: Set<string>; shown: string } | null>(null);
  const [selection, setSelection] = useState<Selection>(emptySelection);
  const [edit, setEdit] = useState<Edit | null>(null);
  const [menu, setMenu] = useState<(RowMenuAt & { ids: string[] }) | null>(null);
  const [ghost, setGhost] = useState("");
  const [days, setDays] = useState<string | null>(null);
  const [readFrom, setReadFrom] = useState<string | null>(null);
  const [last, setLast] = useState<{ change: string; label: string } | null>(null);
  const tableRef = useRef<HTMLTableElement>(null);
  const raised = useRef(0);

  useEffect(() => {
    let stopped = false;
    deliveryApi.view(projectId, system).then(v => { if (!stopped) { setView(v); setError(""); } }).catch(e => {
      if (stopped) return;
      if (e instanceof ApiError && e.status === 401) onSignedOut();
      else setError("The register could not be read. Reload to try again.");
    });
    return () => { stopped = true; };
  }, [projectId, system, tick, reload, onSignedOut]);

  const all = view?.items ?? [];
  const rows = useMemo(() => {
    const shown = all.filter(r => (!kind || r.kind === kind) && (showClosed || !CLOSED.has(r.status)) && (!only || only.ids.has(r.id)));
    // Conditions group by gate, in the order a project reaches them.
    return kind === "condition" ? [...shown].sort(conditionOrder) : shown;
  }, [all, kind, showClosed, only]);
  const order = rows.map(r => r.id);
  const orderKey = order.join("|");
  useEffect(() => setSelection(s => prune(s, orderKey ? orderKey.split("|") : [])), [orderKey]);
  const selected = new Set(selection.ids);
  const rowOf = (id: string) => all.find(r => r.id === id);
  const one = selection.ids.length === 1 ? rowOf(selection.ids[0]) : undefined;
  const systemLabel = (id?: string) => view?.systems.find(s => s.id === id)?.label ?? "";
  const refocus = () => requestAnimationFrame(() => tableRef.current?.focus({ preventScroll: true }));

  // One batch, one Undo. A 409 means the row changed under the PM: the
  // register is read again and the PM tries once more.
  const run = useCallback(async (ops: OpCall[], label: string) => {
    setBusy(true); setError("");
    try {
      const out = await deliveryApi.ops(projectId, ops);
      setLast(out.change_id ? { change: out.change_id, label } : null);
      setReload(n => n + 1);
      return true;
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) onSignedOut();
      else if (e instanceof ApiError && e.status === 409) setError("This row changed while you were editing. The register was read again; try once more.");
      else setError(e instanceof ApiError && e.message ? e.message : "Not saved. Try again.");
      setReload(n => n + 1);
      return false;
    } finally { setBusy(false); }
  }, [projectId, onSignedOut]);
  const editOps = (ids: string[], fields: Record<string, unknown>) =>
    ids.flatMap(id => { const r = rowOf(id); return r ? [{ op: "delivery.edit", fields: { item: r.id, version: r.version, ...fields } }] : []; });

  // "Raise RFI from this document": a draft RFI with the document as its source.
  useEffect(() => {
    if (!raise || raise.n === raised.current) return;
    raised.current = raise.n;
    void run([{ op: "delivery.add", fields: { kind: "rfi", title: raise.title.slice(0, 200), raised_on: sydneyToday(), source_refs: [{ document_id: raise.documentId }] } }], "Raised an RFI from the document.");
  }, [raise, run]);

  useChatTable("register", "register", {
    select: ids => { setOnly(null); setSelection(selectAll(ids.filter(id => all.some(r => r.id === id)))); },
    only: (ids, shown) => { setShowClosed(true); setKind(""); setOnly({ ids: new Set(ids), shown: shown ?? "" }); },
    clear: () => setOnly(null),
    open: id => { setSelection(clickRow(emptySelection, id, all.map(r => r.id))); revealQuietly(tableRef.current?.querySelector(`tr[data-id="${CSS.escape(id)}"]`)); },
  }, selection.ids, !!view);

  const commit = async (e: Edit, text: string, key: CommitKey) => {
    const value = text.trim();
    const ids = e.many ?? [e.id];
    const r = rowOf(e.id);
    const gate = r?.kind === "condition" && e.field === "due";
    if (e.field === "due" && !gate && value && !/^\d{4}-\d{2}-\d{2}$/.test(value)) { setEdit({ ...e, error: "A date such as 2026-12-04" }); return; }
    if (e.field === "title" && !value) { setEdit({ ...e, error: "An item needs words" }); return; }
    const fields: Record<string, unknown> = e.field === "title" ? { title: value } : e.field === "with" ? { owner_text: value }
      : gate ? { gate: value } : e.field === "due" ? { target_date: value || null } : e.field === "system" ? { system: value || null } : { status: value };
    const same = ids.length === 1 && r && (e.field === "title" ? r.title === value : e.field === "with" ? r.owner_text === value : gate ? (cond(r).stage ?? "") === value : e.field === "due" ? (r.target_date ?? "") === value : e.field === "system" ? (r.system_id ?? "") === value : r.status === value);
    setEdit(null);
    if (!same) await run(editOps(ids, fields), "Saved.");
    if (key !== "blur") refocus();
  };

  const statusOps = (ids: string[], status: string) => editOps(ids.filter(id => STATUSES[rowOf(id)?.kind ?? ""]?.includes(status)), { status });
  const menuItems = (ids: string[]): MenuItem[] => {
    const rs = ids.flatMap(id => rowOf(id) ?? []);
    const owed = rs.every(r => r.kind === "rfi" || r.kind === "notice");
    const conditions = rs.every(r => r.kind === "condition");
    const out: MenuItem[] = [];
    if (rs.length === 1) out.push({ label: "Edit item", icon: null, disabled: busy, onSelect: () => setEdit({ id: rs[0].id, field: "title" }) });
    // Send… (correspondence plan): an RFI goes out with its question and
    // documents, and its reply threads back to it; a condition with its notice.
    if (rs.length === 1 && rs[0].kind === "rfi") {
      const r = rs[0], d = r.details as { question?: string; answer_documents?: string[] };
      const docs = (r.provenance.sources ?? []).flatMap(s => s.document_id ? [s.document_id] : []);
      out.push({ label: "Send…", icon: null, onSelect: () => dialogOf("send")?.open({ rfi: r.id, subject: `RFI-${String(r.number ?? 0).padStart(3, "0")} ${r.title}`, note: d.question ?? "", documents: docs.join(",") }) });
    }
    if (rs.length === 1 && rs[0].kind === "condition") {
      const c = cond(rs[0]);
      out.push({ label: "Send…", icon: null, onSelect: () => dialogOf("send")?.open({ subject: `Condition ${c.number ?? ""} ${c.heading ?? ""}`.trim(), note: c.text ?? "", documents: c.source?.document_id ?? "" }) });
    }
    out.push({ label: rs.length === 1 ? "Set system" : `Set system for ${rs.length} rows`, icon: null, disabled: busy, onSelect: () => setEdit({ id: rs[0].id, field: "system", many: rs.length > 1 ? ids : undefined }) });
    if (owed) {
      out.push("sep");
      if (rs.some(r => !CLOSED.has(r.status))) {
        out.push({ label: rs.length === 1 ? "Close" : `Close ${rs.length} rows`, icon: null, disabled: busy, onSelect: () => void run(statusOps(ids, "closed"), "Closed.") });
        out.push({ label: rs.length === 1 ? "Withdraw" : `Withdraw ${rs.length} rows`, icon: null, disabled: busy, onSelect: () => void run(statusOps(ids, "withdrawn"), "Withdrawn.") });
      }
      if (rs.some(r => CLOSED.has(r.status) || r.status === "answered")) out.push({ label: "Reopen", icon: null, disabled: busy, onSelect: () => void run(statusOps(ids, "open"), "Reopened.") });
    } else if (conditions) {
      // Satisfied is the PM's own click; evidence never satisfies a condition.
      out.push("sep");
      if (rs.length === 1) out.push({ label: "Set status", icon: null, disabled: busy, onSelect: () => setEdit({ id: rs[0].id, field: "status" }) });
      out.push({ label: rs.length === 1 ? "Satisfied" : `Mark ${rs.length} satisfied`, icon: null, disabled: busy, onSelect: () => void run(statusOps(ids, "satisfied"), "Marked satisfied.") });
      out.push({ label: rs.length === 1 ? "Not applicable" : `Mark ${rs.length} not applicable`, icon: null, disabled: busy, onSelect: () => void run(statusOps(ids, "not_applicable"), "Marked not applicable.") });
      if (rs.length === 1) out.push({ label: "Add to RFP", icon: null, disabled: busy || !rs[0].package_id || !!cond(rs[0]).rfp, onSelect: () => void addToRFP(rs[0]) });
    } else if (rs.length === 1) out.push({ label: "Set status", icon: null, disabled: busy, onSelect: () => setEdit({ id: rs[0].id, field: "status" }) });
    out.push("sep", { label: rs.length === 1 ? "Remove from register" : `Remove ${rs.length} rows`, icon: null, danger: true, disabled: busy, onSelect: () => void run(editOps(ids, { retired: true }), "Removed.") });
    return out;
  };
  const openMenu = (id: string, x: number, y: number, trigger: HTMLElement | null) => {
    const ids = selected.has(id) ? [...selection.ids] : [id];
    if (!selected.has(id)) setSelection(clickRow(emptySelection, id, order));
    setMenu({ x, y, trigger, ids });
  };
  const keys = (e: React.KeyboardEvent) => {
    if ((e.target as HTMLElement).closest("input,select,textarea")) return;
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "a") { e.preventDefault(); setSelection(selectAll(order)); }
    else if (e.key === "Escape" && selection.ids.length) setSelection(emptySelection);
    else if ((e.key === "ArrowDown" || e.key === "ArrowUp") && order.length) {
      e.preventDefault();
      const at = order.indexOf(selection.anchor ?? "");
      const next = order[Math.max(0, Math.min(order.length - 1, at + (e.key === "ArrowDown" ? 1 : -1)))] ?? order[0];
      setSelection(clickRow(emptySelection, next, order));
    } else if ((e.key === "Enter" || e.key === "F2") && selection.ids.length === 1 && !busy && !edit) { e.preventDefault(); setEdit({ id: selection.ids[0], field: "title" }); }
    else if ((e.key === "ContextMenu" || (e.shiftKey && e.key === "F10")) && selection.ids.length) {
      e.preventDefault();
      const row = tableRef.current?.querySelector<HTMLElement>(`tr[data-id="${CSS.escape(selection.ids[0])}"]`);
      const b = row?.getBoundingClientRect();
      openMenu(selection.ids[0], b ? b.left + 24 : 0, b ? b.bottom : 0, row ?? null);
    }
  };
  // Add to RFP writes the RFP draft's setting, which has no Undo yet.
  const addToRFP = async (r: DeliveryRecord) => {
    setBusy(true); setError("");
    try {
      await request("POST", `/projects/${projectId}/ops`, { area: "reports", ops: [{ op: "reports.add_scope_line", fields: { item: r.id } }] });
      setLast(null); setReload(n => n + 1);
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) onSignedOut();
      else setError(e instanceof ApiError && e.message ? opMessage(e.message) : "Not added. Try again.");
    } finally { setBusy(false); }
  };
  // Read conditions: Go checks the notice and queues the background read.
  const readNotice = async (documentId: string) => {
    setBusy(true); setError(""); setReadFrom(null);
    try { await deliveryApi.readNotice(projectId, documentId); setReload(n => n + 1); }
    catch (e) {
      if (e instanceof ApiError && e.status === 401) onSignedOut();
      else setError(e instanceof ApiError && e.message ? e.message : "The conditions could not be read. Try again.");
    } finally { setBusy(false); }
  };
  const addRFI = () => {
    const title = ghost.trim();
    if (!title) return;
    setGhost("");
    void run([{ op: "delivery.add", fields: { kind: "rfi", title, raised_on: sydneyToday() } }], "Added an RFI.");
  };

  const cell = (r: DeliveryRecord, field: Field, shown: React.ReactNode, value: string, extra: { band?: string; title?: string } = {}) => {
    const editing = edit?.id === r.id && edit.field === field;
    const options = field === "system" ? [{ id: "", label: "Not set" }, ...(view?.systems ?? [])] : field === "status" ? (STATUSES[r.kind] ?? []).map(s => ({ id: s, label: words(s) }))
      : field === "due" && r.kind === "condition" ? [{ id: "", label: "Not known" }, ...GATES.map(([id, label]) => ({ id, label }))] : undefined;
    const start = () => { if (busy) return; setSelection(clickRow(emptySelection, r.id, order)); setEdit({ id: r.id, field }); };
    return <td data-field={field} data-band={extra.band} title={extra.title} onDoubleClick={start}>
      {editing ? <InlineCellEditor value={value} label={`${r.title} ${field}`} error={edit.error ?? ""} busy={busy} maxLength={200} options={options}
        onCommit={(text, key) => void commit(edit, text, key)} onCancel={() => { setEdit(null); refocus(); }} />
        : <span className="register-value">{shown}</span>}
    </td>;
  };
  const due = (r: DeliveryRecord) => {
    if (!r.target_date) return "—";
    return r.late_days ? <>{r.target_date} <span className="register-late">· {r.late_days} {r.late_days === 1 ? "day" : "days"} late</span></> : r.target_date;
  };
  const typicalDue = (r: DeliveryRecord) => r.kind === "rfi" && !!view?.response_typical && !r.details.due_typed && !!r.target_date;
  const calendarWords = view?.calendar === "weekdays" ? "Monday to Friday; there is no Program" : "the Program's working days";

  const also = view?.also_open;
  const filters = [kind && (KIND[kind]?.[1] ?? kind), system && `on ${systemLabel(system)}`].filter(Boolean).join(" ");
  if (report) return <Reports key={report} reportId={report} tick={tick + readerTick} backTo="Register" surface="register" onBack={() => { setReport(null); setReload(n => n + 1); }} onSignedOut={onSignedOut} />;
  return <section className="register-screen" aria-label="Register" aria-busy={busy}>
    <div className="register-head section-band">
      <h2>Register</h2>
      {view && view.kinds.length > 0 && <select aria-label="Kind" value={kind} onChange={e => setKind(e.target.value)}>
        <option value="">All kinds</option>
        {view.kinds.map(k => <option key={k.kind} value={k.kind}>{KIND[k.kind]?.[1] ?? k.kind}{k.to_check ? ` · ${k.to_check} to check` : ""}</option>)}
      </select>}
      {view && <select aria-label="System" value={system} onChange={e => setSystem(e.target.value)}>
        <option value="">All systems</option>
        {view.systems.map(s => <option key={s.id} value={s.id}>{s.label}</option>)}
      </select>}
      <label className="register-toggle"><input type="checkbox" checked={showClosed} onChange={e => setShowClosed(e.target.checked)} /> Show closed</label>
      <ProjectReportButton projectId={projectId} onOpen={setReport} onSignedOut={onSignedOut} />
      <MoreMenu label="Register actions" items={[{ label: "Read conditions…", icon: null, disabled: busy || documents.length === 0, onSelect: () => setReadFrom(documents[0]?.id ?? "") }, { label: `RFI response time: ${view?.response_days ?? 5} working days${view?.response_typical ? " (typical)" : ""}…`, icon: null, onSelect: () => setDays(String(view?.response_days ?? 5)) }]} />
    </div>
    {days !== null && <form className="register-days" onSubmit={e => { e.preventDefault(); const n = Number(days); if (n >= 1 && n <= 60) { setDays(null); void run([{ op: "delivery.settings", fields: { rfi_response_days: n } }], `RFI response time set to ${n} working days.`); } }}>
      <label>RFI response time <input type="number" min={1} max={60} value={days} onChange={e => setDays(e.target.value)} autoFocus aria-label="RFI response time in working days" /> working days, on {calendarWords}.</label>
      <button className="btn btn-small" disabled={busy}>Save</button> <button type="button" className="cell-link" onClick={() => setDays(null)}>Cancel</button>
    </form>}
    {readFrom !== null && <form className="register-days" onSubmit={e => { e.preventDefault(); if (readFrom) void readNotice(readFrom); }}>
      <label>Read the consent conditions of <select aria-label="Notice to read" value={readFrom} onChange={e => setReadFrom(e.target.value)}>
        {documents.map(d => <option key={d.id} value={d.id}>{d.title}</option>)}</select></label>
      <button className="btn btn-small" disabled={busy || !readFrom}>Read</button> <button type="button" className="cell-link" onClick={() => setReadFrom(null)}>Cancel</button>
    </form>}
    {error && <div className="banner" role="alert"><p>{error}</p></div>}
    {view && (kind === "condition" || (!kind && all.some(r => r.kind === "condition")) || (view.consents ?? []).some(c => c.status !== "done")) && <ConsentHead view={view} />}
    {last && <p className="reg-notice" role="status">{last.label} <button type="button" className="cell-link" disabled={busy} onClick={() => { const c = last.change; setLast(null); void deliveryApi.undo(projectId, c).then(() => setReload(n => n + 1)).catch(() => setError("Can't undo: this has changed since.")); }}>Undo</button></p>}
    {filters && !only && <p className="reg-filter">Showing {filters} · <button type="button" className="cell-link" onClick={() => { setKind(""); setSystem(""); }}>Clear</button></p>}
    {only && <p className="reg-filter">{showingLine("register", only.ids.size, only.shown)} · <button type="button" className="cell-link" onClick={() => setOnly(null)}>Clear</button></p>}
    {also && <p className="register-also">Also open on {also.label}: <button type="button" className="cell-link" onClick={() => onShowView("specifications")}>{also.spec_unknowns} specification {also.spec_unknowns === 1 ? "row" : "rows"} not settled</button>
      {" · "}<button type="button" className="cell-link" onClick={() => onShowView("procurement")}>{also.exclusions} tender {also.exclusions === 1 ? "exclusion" : "exclusions"}</button>
      {also.main_works_exclusions > 0 && <>{" · "}<button type="button" className="cell-link" onClick={() => onShowView("procurement")}>Main Works lists {also.main_works_exclusions}, not sorted by system</button></>}</p>}
    {!view ? !error && <p role="status"><Pending delayed>Loading the register…</Pending></p> : <div className="register-scroll">
      <table ref={tableRef} className="register-table" tabIndex={0} onKeyDown={keys} data-testid="register-table"
        aria-label="Register. Click to select, Shift or Ctrl to select more, F2 to edit, right-click for actions.">
        <thead><tr><th scope="col">No.</th><th scope="col">Item</th><th scope="col">With</th>
          <th scope="col" title={`RFI response time ${view.response_days} working days${view.response_typical ? " (typical)" : ""}, on ${calendarWords}`}>Due</th>
          <th scope="col">Status</th><th scope="col">System</th><th scope="col"><span className="sr-only">Actions</span></th></tr></thead>
        <tbody>
          {all.length === 0 && <tr className="register-empty"><td colSpan={7}>Nothing is recorded as owed yet.</td></tr>}
          {rows.map((r, i) => <Fragment key={r.id}>{kind === "condition" && (i === 0 || (cond(rows[i - 1]).stage ?? "") !== (cond(r).stage ?? "")) &&
            <tr className="register-group"><th colSpan={7} scope="rowgroup">{GATE_WORDS[cond(r).stage ?? ""] ?? "Gate not known"}</th></tr>}<tr data-id={r.id} aria-selected={selected.has(r.id)} tabIndex={-1}
            onClick={e => { if ((e.target as HTMLElement).closest("input,select,button")) return; setSelection(s => clickRow(s, r.id, order, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })); }}
            onContextMenu={e => { if ((e.target as HTMLElement).closest("input")) return; e.preventDefault(); openMenu(r.id, e.clientX, e.clientY, e.currentTarget); }}>
            <th scope="row" className="register-no">{r.kind === "condition" ? <span title={`Condition ${cond(r).number ?? ""}`}>{cond(r).number}</span> : rfiNumber(r.number) || <span title={KIND[r.kind]?.[0]}>{KIND[r.kind]?.[0] ?? r.kind}</span>}</th>
            {cell(r, "title", r.title, r.title)}
            {cell(r, "with", r.with || "—", r.owner_text)}
            {r.kind === "condition"
              ? cell(r, "due", GATE_WORDS[cond(r).stage ?? ""] ?? "Gate not known", cond(r).stage ?? "", { band: r.review_status === "proposed" ? "amber" : undefined })
              : cell(r, "due", due(r), r.target_date ?? "", { band: typicalDue(r) ? "typical" : undefined, title: typicalDue(r) ? `${view.response_days} working days after it was raised (typical)` : undefined })}
            {r.kind === "condition"
              ? cell(r, "status", <>{conditionStatus(r)}{r.review_status === "proposed" ? <span className="register-check"> · to check</span> : null}</>, r.status, { band: r.review_status === "proposed" || r.readings?.length ? "amber" : undefined })
              : cell(r, "status", <>{words(r.status)}{r.readings?.length ? <span className="register-check"> · {r.readings.length} to check</span> : null}</>, r.status, { band: r.readings?.length ? "amber" : undefined })}
            {cell(r, "system", systemLabel(r.system_id) || "—", r.system_id ?? "")}
            <td className="register-more"><button type="button" className="reg-more-btn" aria-label={`Actions for ${r.title}`} aria-haspopup="menu"
              onClick={e => { e.stopPropagation(); const b = e.currentTarget.getBoundingClientRect(); openMenu(r.id, b.left, b.bottom, e.currentTarget); }}><IconMore /></button></td>
          </tr></Fragment>)}
          <tr className="register-ghost"><td colSpan={7}>
            <input value={ghost} disabled={busy} maxLength={200} placeholder="Raise an RFI…" aria-label="Raise an RFI"
              onChange={e => setGhost(e.target.value)} onBlur={addRFI}
              onKeyDown={e => { e.stopPropagation(); if (e.key === "Enter") { e.preventDefault(); addRFI(); } else if (e.key === "Escape") setGhost(""); }} />
          </td></tr>
        </tbody>
      </table>
    </div>}
    {one && one.kind === "condition" && <ConditionDetail row={one} busy={busy} documents={documents} onOpenDocument={onOpenDocument}
      onLink={doc => void run(editOps([one.id], { link_evidence: doc }), "Linked the evidence.")}
      onAccept={id => void run(editOps([one.id], { accept_reading: id }), "Linked the evidence.")}
      onDismiss={id => void run([{ op: "delivery.edit", fields: { item: one.id, dismiss_readings: [id] } }], "Dismissed.")}
      onSatisfied={() => void run(statusOps([one.id], "satisfied"), "Marked satisfied.")}
      onAddToRFP={() => void addToRFP(one)} />}
    {one && one.kind !== "condition" && <Detail row={one} busy={busy} onOpenDocument={onOpenDocument}
      onSave={fields => void run(editOps([one.id], fields), "Saved.")}
      onAccept={id => void run(editOps([one.id], { accept_reading: id }), "Accepted the answer.")}
      onDismiss={id => void run([{ op: "delivery.edit", fields: { item: one.id, dismiss_readings: [id] } }], "Dismissed.")} />}
    {menu && <RowMenu at={menu} items={menuItems(menu.ids)} label="Register actions" head={menu.ids.length > 1 ? `${menu.ids.length} rows` : rowOf(menu.ids[0])?.title}
      onClose={restore => { setMenu(null); if (restore) menu.trigger?.focus({ preventScroll: true }); }} />}
  </section>;
}

// Detail is what is particular to the selected row's kind: an RFI's
// question, answer, documents and possible answers.
function Detail({ row, busy, onSave, onAccept, onDismiss, onOpenDocument }: {
  row: DeliveryRecord; busy: boolean; onSave: (fields: Record<string, unknown>) => void;
  onAccept: (reading: string) => void; onDismiss: (reading: string) => void; onOpenDocument: (id: string) => void;
}) {
  const d = row.details as { question?: string; answer?: string; ref?: string; raised_on?: string | null; answer_documents?: string[] };
  const text = (key: "question" | "answer" | "ref", label: string, area = true) =>
    <label className="register-field">{label}{area
      ? <textarea key={`${row.id}-${row.version}-${key}`} defaultValue={d[key] ?? ""} maxLength={8000} rows={3} disabled={busy} onBlur={e => { if (e.target.value !== (d[key] ?? "")) onSave({ [key]: e.target.value }); }} />
      : <input key={`${row.id}-${row.version}-${key}`} defaultValue={d[key] ?? ""} maxLength={100} disabled={busy} onBlur={e => { if (e.target.value !== (d[key] ?? "")) onSave({ [key]: e.target.value }); }} />}</label>;
  const history = row.provenance.status_history ?? [];
  return <section className="register-detail" aria-label="Selected row">
    <h3>{rfiNumber(row.number) ? `${rfiNumber(row.number)} · ` : ""}{row.title}</h3>
    {row.kind === "rfi" && <>
      {text("question", "Question")}
      <div className="register-pair">{text("ref", "Ref", false)}
        <label className="register-field">Raised<input type="date" key={`${row.id}-${row.version}-raised`} defaultValue={d.raised_on ?? ""} disabled={busy}
          onBlur={e => { if (e.target.value !== (d.raised_on ?? "")) onSave({ raised_on: e.target.value || null }); }} /></label></div>
      {text("answer", "Answer")}
      {(d.answer_documents ?? []).length > 0 && <p>Answered by: {(d.answer_documents ?? []).map(id => <button key={id} type="button" className="cell-link" onClick={() => onOpenDocument(id)}>Open</button>)}</p>}
    </>}
    {(row.readings ?? []).map(rd => <div key={rd.id} className="register-reading" data-band="amber">
      <span>Possible answer: {rd.document}{rd.basis ? ` (${rd.basis})` : ""}</span>
      <button type="button" className="cell-link" onClick={() => onOpenDocument(rd.document_id)}>Open</button>
      <button type="button" className="btn btn-small" disabled={busy} onClick={() => onAccept(rd.id)}>Accept</button>
      <button type="button" className="cell-link" disabled={busy} onClick={() => onDismiss(rd.id)}>Dismiss</button>
    </div>)}
    {(row.provenance.sources ?? []).length > 0 && <p>Sources: {(row.provenance.sources ?? []).map((s, i) => <button key={i} type="button" className="cell-link" onClick={() => s.document_id && onOpenDocument(s.document_id)}>{s.filename || "Document"}{s.page ? ` p. ${s.page}` : ""}</button>)}</p>}
    {history.length > 0 && <p className="register-history">{history.map(h => `${words(h.from)} → ${words(h.status)} ${new Date(h.at).toLocaleDateString()}`).join(" · ")}</p>}
  </section>;
}

// ConsentHead is above the conditions: the Profile's consent number, date
// and lapse date, and each notice read, with what it found or why it
// stopped.
function ConsentHead({ view }: { view: RegisterView }) {
  const c = view.consent ?? {};
  return <div className="register-consent" role="status">
    {(c.number || c.date || c.lapse_date) && <p>Consent {c.number || "—"}{c.date ? ` · ${c.date}` : ""}{c.lapse_date ? ` · lapses ${c.lapse_date}` : ""}</p>}
    {(view.consents ?? []).map(r => <p key={r.document_id}>{r.title}: {
      r.status === "queued" ? "Queued" : r.status === "reading" ? "Reading the conditions…"
        : r.status === "failed" ? (r.error || "The read stopped.")
          : r.notice ? <>Read {r.rows} of {r.notice.conditions}{r.notice.gaps.length > 0 ? <> · Conditions {r.notice.gaps.join(", ")} were not found; check the notice</> : null}</> : "Read"}</p>)}
  </div>;
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
  return <section className="register-detail" aria-label="Selected condition">
    <h3>{row.title}</h3>
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
  </section>;
}
