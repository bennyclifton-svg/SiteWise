// The register in the right-hand panel (owner, 20 November 2026; first
// built as a middle view in docs/plans/2026-11-17-rfis-review-comments.md,
// Task 5): what is owed on the project, in a narrow table of four columns.
// A row opens as a sheet in the middle panel (RegisterSheet.tsx). Numbers,
// due dates, days late and With are Go's. Every write goes through POST /ops
// (area delivery), so each action has one Undo. No screen read or write
// here calls Jev or the model.

import { Fragment, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Pending } from "./Activity";
import { useChatTable, showingLine, revealQuietly } from "./chatView";
import { GATES, rfiNumber, sydneyToday, type DeliveryRecord, type RegisterView } from "./deliveryApi";
import { IconMore } from "./icons";
import type { WorkbenchView } from "./LeftNav";
import { MoreMenu } from "./PageActions";
import { ProjectReportButton } from "./ProjectReportButton";
import { editOps, type RegisterData } from "./registerData";
import { CLOSED, conditionOrder, conditionStatus, cond, GATE_WORDS, KIND, STATUSES, statusWords, words } from "./registerKinds";
import { InlineCellEditor, type CommitKey } from "./table/InlineCellEditor";
import { RowMenu, type MenuItem, type RowMenuAt } from "./table/RowMenu";
import { clickRow, emptySelection, prune, selectAll, type Selection } from "./table/selection";
import "./register.css";

type Field = "title" | "with" | "due" | "system" | "status";
interface Edit { id: string; field: Field; many?: string[]; error?: string }

/** The register's row menu items that do not need the list: shared with the sheet's three dots. */
export function statusItems(rows: DeliveryRecord[], busy: boolean, run: (status: string, label: string) => void): MenuItem[] {
  const owed = rows.every(r => r.kind === "rfi" || r.kind === "notice");
  const out: MenuItem[] = [];
  if (!owed) return out;
  const n = rows.length;
  if (rows.some(r => !CLOSED.has(r.status))) {
    out.push({ label: n === 1 ? "Close" : `Close ${n} rows`, icon: null, disabled: busy, onSelect: () => run("closed", "Closed.") });
    out.push({ label: n === 1 ? "Withdraw" : `Withdraw ${n} rows`, icon: null, disabled: busy, onSelect: () => run("withdrawn", "Withdrawn.") });
  }
  if (rows.some(r => CLOSED.has(r.status) || r.status === "answered")) out.push({ label: "Reopen", icon: null, disabled: busy, onSelect: () => run("open", "Reopened.") });
  return out;
}

/** The status chip's words: "Open · with MBM", or a condition's next step. */
export function statusChip(r: DeliveryRecord): string {
  if (r.kind === "condition") return conditionStatus(r);
  const status = statusWords(r.status);
  if (!r.with) return status;
  return `${status} · with ${r.with === "You" ? "you" : r.with}`;
}

export function RegisterPanel({ data, shown, switcher, projectId, raise, openId, onOpen, onOpenReport, onSignedOut, onShowView, documents = [] }: {
  data: RegisterData;
  /** The panel is on screen: chat moves its table only then. */
  shown: boolean;
  /** The Documents | Register switch, at the head of the band. */
  switcher: ReactNode;
  projectId: string;
  raise: { documentId: string; title: string; n: number } | null;
  /** The row whose sheet is open in the middle panel. */
  openId: string | null;
  onOpen: (id: string) => void;
  onOpenReport: (reportId: string) => void;
  onSignedOut: () => void; onShowView: (view: WorkbenchView) => void;
  /** Filed documents, for Read conditions (statutory instruments first). */
  documents?: { id: string; title: string; kind: string }[];
}) {
  const { view, busy, error, run, last } = data;
  const [kind, setKind] = useState("");
  const [showClosed, setShowClosed] = useState(false);
  const [only, setOnly] = useState<{ ids: Set<string>; shown: string } | null>(null);
  const [selection, setSelection] = useState<Selection>(emptySelection);
  const [edit, setEdit] = useState<Edit | null>(null);
  const [menu, setMenu] = useState<(RowMenuAt & { ids: string[] }) | null>(null);
  const [ghost, setGhost] = useState("");
  const [days, setDays] = useState<string | null>(null);
  const [readFrom, setReadFrom] = useState<string | null>(null);
  const tableRef = useRef<HTMLTableElement>(null);
  const raised = useRef(0);
  const system = data.system;

  const all = view?.items ?? [];
  const rows = useMemo(() => {
    const listed = all.filter(r => (!kind || r.kind === kind) && (!system || r.system_id === system) && (showClosed || !CLOSED.has(r.status)) && (!only || only.ids.has(r.id)));
    // Conditions group by gate, in the order a project reaches them.
    return kind === "condition" ? [...listed].sort(conditionOrder) : listed;
  }, [all, kind, system, showClosed, only]);
  const order = rows.map(r => r.id);
  const orderKey = order.join("|");
  useEffect(() => setSelection(s => prune(s, orderKey ? orderKey.split("|") : [])), [orderKey]);
  const selected = new Set(selection.ids);
  const rowOf = data.rowOf;
  const systemLabel = (id?: string) => view?.systems.find(s => s.id === id)?.label ?? "";
  const refocus = () => requestAnimationFrame(() => tableRef.current?.focus({ preventScroll: true }));
  const editRows = (ids: string[], fields: Record<string, unknown>) => editOps(ids.map(rowOf), fields);

  // "Raise RFI from this document": a draft RFI with the document as its source.
  useEffect(() => {
    if (!raise || raise.n === raised.current) return;
    raised.current = raise.n;
    void run([{ op: "delivery.add", fields: { kind: "rfi", title: raise.title.slice(0, 200), raised_on: sydneyToday(), source_refs: [{ document_id: raise.documentId }] } }], "Raised an RFI from the document.");
  }, [raise, run]);

  const open = (id: string) => { setSelection(clickRow(emptySelection, id, all.map(r => r.id))); onOpen(id); };
  useChatTable("register", "documents", {
    select: ids => { setOnly(null); setSelection(selectAll(ids.filter(id => all.some(r => r.id === id)))); },
    only: (ids, words) => { setShowClosed(true); setKind(""); data.setSystem(""); setOnly({ ids: new Set(ids), shown: words ?? "" }); },
    clear: () => setOnly(null),
    open: id => { open(id); revealQuietly(tableRef.current?.querySelector(`tr[data-id="${CSS.escape(id)}"]`)); },
  }, selection.ids, !!view && shown);

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
    if (!same) await run(editRows(ids, fields), "Saved.");
    if (key !== "blur") refocus();
  };

  const statusOps = (ids: string[], status: string) => editRows(ids.filter(id => STATUSES[rowOf(id)?.kind ?? ""]?.includes(status)), { status });
  const menuItems = (ids: string[]): MenuItem[] => {
    const rs = ids.flatMap(id => rowOf(id) ?? []);
    const conditions = rs.every(r => r.kind === "condition");
    const out: MenuItem[] = [];
    if (rs.length === 1) {
      out.push({ label: "Open", icon: null, onSelect: () => onOpen(rs[0].id) });
      out.push({ label: "Edit item", icon: null, disabled: busy, onSelect: () => setEdit({ id: rs[0].id, field: "title" }) });
    }
    // An RFI is sent from its sheet, with its To, Cc and transmittal.
    if (rs.length === 1 && rs[0].kind === "rfi") out.push({ label: "Send…", icon: null, onSelect: () => onOpen(rs[0].id) });
    out.push({ label: rs.length === 1 ? "Set system" : `Set system for ${rs.length} rows`, icon: null, disabled: busy, onSelect: () => setEdit({ id: rs[0].id, field: "system", many: rs.length > 1 ? ids : undefined }) });
    const status = statusItems(rs, busy, (s, label) => void run(statusOps(ids, s), label));
    if (status.length) out.push("sep", ...status);
    else if (conditions) {
      // Satisfied is the PM's own click; evidence never satisfies a condition.
      out.push("sep");
      if (rs.length === 1) out.push({ label: "Set status", icon: null, disabled: busy, onSelect: () => setEdit({ id: rs[0].id, field: "status" }) });
      out.push({ label: rs.length === 1 ? "Satisfied" : `Mark ${rs.length} satisfied`, icon: null, disabled: busy, onSelect: () => void run(statusOps(ids, "satisfied"), "Marked satisfied.") });
      out.push({ label: rs.length === 1 ? "Not applicable" : `Mark ${rs.length} not applicable`, icon: null, disabled: busy, onSelect: () => void run(statusOps(ids, "not_applicable"), "Marked not applicable.") });
    } else if (rs.length === 1 && rs[0].kind !== "rfi" && rs[0].kind !== "notice") out.push({ label: "Set status", icon: null, disabled: busy, onSelect: () => setEdit({ id: rs[0].id, field: "status" }) });
    out.push("sep", { label: rs.length === 1 ? "Remove from register" : `Remove ${rs.length} rows`, icon: null, danger: true, disabled: busy, onSelect: () => void run(editRows(ids, { retired: true }), "Removed.") });
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
    } else if (e.key === "Enter" && selection.ids.length === 1 && !edit) { e.preventDefault(); onOpen(selection.ids[0]); }
    else if (e.key === "F2" && selection.ids.length === 1 && !busy && !edit) { e.preventDefault(); setEdit({ id: selection.ids[0], field: "title" }); }
    else if ((e.key === "ContextMenu" || (e.shiftKey && e.key === "F10")) && selection.ids.length) {
      e.preventDefault();
      const row = tableRef.current?.querySelector<HTMLElement>(`tr[data-id="${CSS.escape(selection.ids[0])}"]`);
      const b = row?.getBoundingClientRect();
      openMenu(selection.ids[0], b ? b.left + 24 : 0, b ? b.bottom : 0, row ?? null);
    }
  };
  const readNotice = async (documentId: string) => { setReadFrom(null); await data.readNotice(documentId); };
  const addRFI = () => {
    const title = ghost.trim();
    if (!title) return;
    setGhost("");
    void run([{ op: "delivery.add", fields: { kind: "rfi", title, raised_on: sydneyToday() } }], "Added an RFI.");
  };

  const editor = (r: DeliveryRecord, field: Field, value: string) => {
    const options = field === "system" ? [{ id: "", label: "Not set" }, ...(view?.systems ?? [])] : field === "status" ? (STATUSES[r.kind] ?? []).map(s => ({ id: s, label: words(s) }))
      : field === "due" && r.kind === "condition" ? [{ id: "", label: "Not known" }, ...GATES.map(([id, label]) => ({ id, label }))] : undefined;
    return <InlineCellEditor value={value} label={`${r.title} ${field}`} error={edit?.error ?? ""} busy={busy} maxLength={200} options={options}
      onCommit={(text, key) => edit && void commit(edit, text, key)} onCancel={() => { setEdit(null); refocus(); }} />;
  };
  const start = (r: DeliveryRecord, field: Field) => { if (busy) return; setSelection(clickRow(emptySelection, r.id, order)); setEdit({ id: r.id, field }); };
  // The Item cell carries the status chip and hosts the status and system
  // editors too, since neither is a column in the narrow table.
  const itemCell = (r: DeliveryRecord) => {
    const editing = edit?.id === r.id && (edit.field === "title" || edit.field === "status" || edit.field === "system") ? edit.field : null;
    const value = editing === "title" ? r.title : editing === "status" ? r.status : r.system_id ?? "";
    const check = r.kind === "condition" ? r.review_status === "proposed" : (r.readings?.length ?? 0) > 0;
    return <td data-field="title" onDoubleClick={() => start(r, "title")}>
      {editing ? editor(r, editing, value) : <span className="register-item">
        <span className="register-value" title={r.title}>{r.title}</span>
        <span className="register-chip register-status" data-band={check ? "amber" : undefined}>{r.kind === "condition" ? conditionStatus(r) : statusWords(r.status)}
          {r.kind === "condition" && r.review_status === "proposed" ? <span className="register-check"> · to check</span> : null}
          {r.kind !== "condition" && r.readings?.length ? <span className="register-check"> · {r.readings.length} to check</span> : null}</span>
      </span>}
    </td>;
  };
  const cell = (r: DeliveryRecord, field: "with" | "due", shown: ReactNode, value: string, extra: { band?: string; title?: string } = {}) => {
    const editing = edit?.id === r.id && edit.field === field;
    return <td data-field={field} data-band={extra.band} title={extra.title} onDoubleClick={() => start(r, field)}>
      {editing ? editor(r, field, value) : <span className="register-value">{shown}</span>}
    </td>;
  };
  const due = (r: DeliveryRecord) => {
    if (!r.target_date) return "—";
    return r.late_days ? <>{r.target_date} <span className="register-late">· {r.late_days} {r.late_days === 1 ? "day" : "days"} late</span></> : r.target_date;
  };
  const typicalDue = (r: DeliveryRecord) => r.kind === "rfi" && !!view?.response_typical && !r.details.due_typed && !!r.target_date;
  const calendarWords = view?.calendar === "weekdays" ? "Monday to Friday; there is no Program" : "the Program's working days";

  const also = system ? data.alsoOpen : undefined;
  const filters = [kind && (KIND[kind]?.[1] ?? kind), system && `on ${systemLabel(system)}`].filter(Boolean).join(" ");
  // The band sits above the card, as the Documents side's does; the card
  // scrolls under it.
  return <>
    <div className="panel-band">
      {switcher}
      <span className="panel-band-tools">
        <ProjectReportButton projectId={projectId} onOpen={onOpenReport} onSignedOut={onSignedOut} />
        <MoreMenu label="Register actions" items={[{ label: "Read conditions…", icon: null, disabled: busy || documents.length === 0, onSelect: () => setReadFrom(documents[0]?.id ?? "") }, { label: `RFI response time: ${view?.response_days ?? 5} working days${view?.response_typical ? " (typical)" : ""}…`, icon: null, onSelect: () => setDays(String(view?.response_days ?? 5)) }]} />
      </span>
    </div>
    <section className="register-screen register-panel" aria-label="Register" aria-busy={busy}>
    <div className="register-filters">
      {view && view.kinds.length > 0 && <select aria-label="Kind" value={kind} onChange={e => setKind(e.target.value)}>
        <option value="">All kinds</option>
        {view.kinds.map(k => <option key={k.kind} value={k.kind}>{KIND[k.kind]?.[1] ?? k.kind}{k.to_check ? ` · ${k.to_check} to check` : ""}</option>)}
      </select>}
      {view && <select aria-label="System" value={system} onChange={e => data.setSystem(e.target.value)}>
        <option value="">All systems</option>
        {view.systems.map(s => <option key={s.id} value={s.id}>{s.label}</option>)}
      </select>}
      <label className="register-toggle"><input type="checkbox" checked={showClosed} onChange={e => setShowClosed(e.target.checked)} /> Show closed</label>
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
    {error && (!openId || last?.from !== "sheet") && <div className="banner" role="alert"><p>{error}</p></div>}
    {view && (kind === "condition" || (!kind && all.some(r => r.kind === "condition")) || (view.consents ?? []).some(c => c.status !== "done")) && <ConsentHead view={view} />}
    {last && last.from === "list" && <p className="reg-notice" role="status">{last.label} <button type="button" className="cell-link" disabled={busy} onClick={data.undo}>Undo</button></p>}
    {filters && !only && <p className="reg-filter">Showing {filters} · <button type="button" className="cell-link" onClick={() => { setKind(""); data.setSystem(""); }}>Clear</button></p>}
    {only && <p className="reg-filter">{showingLine("register", only.ids.size, only.shown)} · <button type="button" className="cell-link" onClick={() => setOnly(null)}>Clear</button></p>}
    {also && <p className="register-also">Also open on {also.label}: <button type="button" className="cell-link" onClick={() => onShowView("specifications")}>{also.spec_unknowns} specification {also.spec_unknowns === 1 ? "row" : "rows"} not settled</button>
      {" · "}<button type="button" className="cell-link" onClick={() => onShowView("procurement")}>{also.exclusions} tender {also.exclusions === 1 ? "exclusion" : "exclusions"}</button>
      {also.main_works_exclusions > 0 && <>{" · "}<button type="button" className="cell-link" onClick={() => onShowView("procurement")}>Main Works lists {also.main_works_exclusions}, not sorted by system</button></>}</p>}
    {!view ? !error && <p role="status"><Pending delayed>Loading the register…</Pending></p> : <div className="register-scroll">
      <table ref={tableRef} className="register-table" tabIndex={0} onKeyDown={keys} data-testid="register-table"
        aria-label="Register. Click to open, Shift or Ctrl to select more, F2 to edit, right-click for actions.">
        <colgroup><col className="register-col-no" /><col /><col className="register-col-with" /><col className="register-col-due" /><col className="register-col-more" /></colgroup>
        <thead><tr><th scope="col">No.</th><th scope="col">Item</th><th scope="col">With</th>
          <th scope="col" title={`RFI response time ${view.response_days} working days${view.response_typical ? " (typical)" : ""}, on ${calendarWords}`}>Due</th>
          <th scope="col"><span className="sr-only">Actions</span></th></tr></thead>
        <tbody>
          {all.length === 0 && <tr className="register-empty"><td colSpan={5}>Nothing is recorded as owed yet.</td></tr>}
          {rows.map((r, i) => <Fragment key={r.id}>{kind === "condition" && (i === 0 || (cond(rows[i - 1]).stage ?? "") !== (cond(r).stage ?? "")) &&
            <tr className="register-group"><th colSpan={5} scope="rowgroup">{GATE_WORDS[cond(r).stage ?? ""] ?? "Gate not known"}</th></tr>}<tr data-id={r.id} aria-selected={selected.has(r.id)} data-open={openId === r.id ? "true" : undefined} tabIndex={-1}
            title={systemLabel(r.system_id) ? `System: ${systemLabel(r.system_id)}` : undefined}
            onClick={e => {
              if ((e.target as HTMLElement).closest("input,select,button")) return;
              const more = e.ctrlKey || e.metaKey || e.shiftKey;
              setSelection(s => clickRow(s, r.id, order, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey }));
              if (!more && !(edit?.id === r.id)) onOpen(r.id);
            }}
            onContextMenu={e => { if ((e.target as HTMLElement).closest("input")) return; e.preventDefault(); openMenu(r.id, e.clientX, e.clientY, e.currentTarget); }}>
            <th scope="row" className="register-no">{r.kind === "condition" ? <span title={`Condition ${cond(r).number ?? ""}`}>{cond(r).number}</span> : rfiNumber(r.number) || <span title={KIND[r.kind]?.[0]}>{KIND[r.kind]?.[0] ?? r.kind}</span>}</th>
            {itemCell(r)}
            {cell(r, "with", r.with || "—", r.owner_text)}
            {r.kind === "condition"
              ? cell(r, "due", GATE_WORDS[cond(r).stage ?? ""] ?? "Gate not known", cond(r).stage ?? "", { band: r.review_status === "proposed" ? "amber" : undefined })
              : cell(r, "due", due(r), r.target_date ?? "", { band: typicalDue(r) ? "typical" : undefined, title: typicalDue(r) ? `${view.response_days} working days after it was raised (typical)` : undefined })}
            <td className="register-more"><button type="button" className="reg-more-btn" aria-label={`Actions for ${r.title}`} aria-haspopup="menu"
              onClick={e => { e.stopPropagation(); const b = e.currentTarget.getBoundingClientRect(); openMenu(r.id, b.left, b.bottom, e.currentTarget); }}><IconMore /></button></td>
          </tr></Fragment>)}
          <tr className="register-ghost"><td colSpan={5}>
            <input value={ghost} disabled={busy} maxLength={200} placeholder="Raise an RFI…" aria-label="Raise an RFI"
              onChange={e => setGhost(e.target.value)} onBlur={addRFI}
              onKeyDown={e => { e.stopPropagation(); if (e.key === "Enter") { e.preventDefault(); addRFI(); } else if (e.key === "Escape") setGhost(""); }} />
          </td></tr>
        </tbody>
      </table>
    </div>}
    {menu && <RowMenu at={menu} items={menuItems(menu.ids)} label="Register actions" head={menu.ids.length > 1 ? `${menu.ids.length} rows` : rowOf(menu.ids[0])?.title}
      onClose={restore => { setMenu(null); if (restore) menu.trigger?.focus({ preventScroll: true }); }} />}
    </section>
  </>;
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
