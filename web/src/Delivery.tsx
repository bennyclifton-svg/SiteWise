// The register: what is owed on the project, as one list in the middle
// panel's Correspondence view (owner, 25 November 2026; before that a narrow
// card in the right-hand panel, and first a middle view in
// docs/plans/2026-11-17-rfis-review-comments.md, Task 5). Its columns can be
// sorted with a click and moved with a drag; the order is kept for this
// viewer only. A row drills in to its sheet (RegisterSheet.tsx). Numbers,
// due dates, days late and With are Go's. Every write goes through POST /ops
// (area delivery), so each action has one Undo. No screen read or write
// here calls Jev or the model.

import { Fragment, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Pending } from "./Activity";
import { useChatTable, showingLine, revealQuietly } from "./chatView";
import { GATES, numberLabel, sydneyToday, type ClaimDetails, type Completion, type DeliveryRecord, type RegisterView } from "./deliveryApi";
import { IconMore } from "./icons";
import type { WorkbenchView } from "./LeftNav";
import { MoreMenu } from "./PageActions";
import { editOps, type RegisterData } from "./registerData";
import { CLOSED, conditionOrder, conditionStatus, cond, GATE_WORDS, KIND, OWED, RECEIVED, resultWords, STATUSES, statusWords, words } from "./registerKinds";
import { InlineCellEditor, type CommitKey } from "./table/InlineCellEditor";
import { RowMenu, type MenuItem, type RowMenuAt } from "./table/RowMenu";
import { clickRow, emptySelection, prune, selectAll, type Selection } from "./table/selection";
import "./register.css";

type Field = "title" | "with" | "due" | "system" | "status";
interface Edit { id: string; field: Field; many?: string[]; error?: string }

/** The register row menu items that do not need the list: shared with the sheet's three dots. */
export function statusItems(rows: DeliveryRecord[], busy: boolean, run: (status: string, label: string) => void): MenuItem[] {
  const owed = rows.every(r => OWED.has(r.kind));
  const out: MenuItem[] = [];
  if (!owed) return out;
  const n = rows.length;
  if (rows.some(r => !CLOSED.has(r.status))) {
    out.push({ label: n === 1 ? "Close" : `Close ${n} rows`, icon: null, disabled: busy, onSelect: () => run("closed", "Closed.") });
    out.push({ label: n === 1 ? "Withdraw" : `Withdraw ${n} rows`, icon: null, disabled: busy, onSelect: () => run("withdrawn", "Withdrawn.") });
  }
  if (rows.some(r => CLOSED.has(r.status) || r.status === "answered" || r.status === "assessed")) out.push({ label: "Reopen", icon: null, disabled: busy, onSelect: () => run("open", "Reopened.") });
  return out;
}

/** The status chip's words: "Open · with MBM", or a condition's next step. */
export function statusChip(r: DeliveryRecord): string {
  if (r.kind === "condition") return conditionStatus(r);
  const d = r.details as ClaimDetails;
  const status = r.kind === "eot_claim" && d.result ? `${statusWords(r.status)} · ${resultWords(d.result, d.days_claimed)}` : statusWords(r.status);
  if (!r.with) return status;
  return `${status} · with ${r.with === "You" ? "you" : r.with}`;
}

/** Whether the host raised the item (an EOT claim, notice or letter it lodged). */
export const raised = (r: DeliveryRecord) => RECEIVED.has(r.kind) && (r.details as ClaimDetails).direction === "out";

/** Who raised the row: "You" for the host's own RFI or raised item, else the sender. */
export function raisedBy(r: DeliveryRecord): string {
  const d = r.details as { raised_by?: string };
  if (r.kind === "rfi") return d.raised_by || "You";
  if (RECEIVED.has(r.kind)) return raised(r) ? "You" : d.raised_by || "";
  if (r.kind === "notice") return d.raised_by || "";
  return "";
}

/** The day the row was issued: an RFI's raised date, the day an item was received or lodged. */
export function issuedOn(r: DeliveryRecord): string {
  const d = r.details as { raised_on?: string | null; received_on?: string | null };
  return d.raised_on ?? d.received_on ?? "";
}

// The list's columns. The order is the viewer's own, kept in the browser.
type Col = "type" | "no" | "subject" | "from" | "with" | "issued" | "due" | "status";
const COLUMN: Record<Col, string> = { type: "Type", no: "No.", subject: "Subject", from: "Raised by", with: "With", issued: "Issued", due: "Due", status: "Status" };
const DEFAULT_ORDER: Col[] = ["type", "no", "subject", "from", "with", "issued", "due", "status"];
const ORDER_KEY = "sitewise.register.columns.v1";
function readOrder(): Col[] {
  try {
    const v: unknown = JSON.parse(localStorage.getItem(ORDER_KEY) ?? "null");
    if (Array.isArray(v) && v.length === DEFAULT_ORDER.length && DEFAULT_ORDER.every(c => v.includes(c))) return v as Col[];
  } catch { /* private window or a damaged value: the default order */ }
  return DEFAULT_ORDER;
}
function storeOrder(order: Col[]) {
  try { localStorage.setItem(ORDER_KEY, JSON.stringify(order)); } catch { /* the order lasts this visit */ }
}
const GATE_RANK: Record<string, number> = Object.fromEntries(GATES.map(([g], i) => [g, i]));
const byText = (a: string, b: string) => (a === "") === (b === "") ? a.localeCompare(b, undefined, { numeric: true, sensitivity: "base" }) : a === "" ? 1 : -1;
/** Each column's sort, blanks last either way. */
function compare(col: Col, a: DeliveryRecord, b: DeliveryRecord): number {
  switch (col) {
    case "type": return byText(KIND[a.kind]?.[0] ?? a.kind, KIND[b.kind]?.[0] ?? b.kind);
    case "no": return a.kind === "condition" && b.kind === "condition" ? byText(cond(a).number ?? "", cond(b).number ?? "")
      : byText(KIND[a.kind]?.[0] ?? a.kind, KIND[b.kind]?.[0] ?? b.kind) || (a.number ?? 1e9) - (b.number ?? 1e9);
    case "subject": return byText(a.title, b.title);
    case "from": return byText(raisedBy(a), raisedBy(b));
    case "with": return byText(a.with ?? "", b.with ?? "");
    case "issued": return byText(issuedOn(a), issuedOn(b));
    case "due": return a.kind === "condition" || b.kind === "condition"
      ? (GATE_RANK[cond(a).stage ?? ""] ?? 99) - (GATE_RANK[cond(b).stage ?? ""] ?? 99) : byText(a.target_date ?? "", b.target_date ?? "");
    case "status": return byText(statusChip(a), statusChip(b));
  }
}

export function RegisterList({ data, shown, head, raise, onOpen, onShowView, documents = [] }: {
  data: RegisterData;
  /** The list is on screen: chat moves its table only then. */
  shown: boolean;
  /** The view's heading (PanelHead), given the list's three dots. */
  head: (more: ReactNode) => ReactNode;
  raise: { documentId: string; title: string; n: number } | null;
  /** Drills in to the row's sheet. */
  onOpen: (id: string) => void;
  onShowView: (view: WorkbenchView) => void;
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
  const [terms, setTerms] = useState<Terms | null>(null);
  const [ghostKind, setGhostKind] = useState("rfi");
  const [ghostDir, setGhostDir] = useState<"in" | "out">("in");
  const [readFrom, setReadFrom] = useState<string | null>(null);
  const [columns, setColumns] = useState<Col[]>(readOrder);
  const [sort, setSort] = useState<{ col: Col; dir: 1 | -1 } | null>(null);
  const [dragging, setDragging] = useState<Col | null>(null);
  const tableRef = useRef<HTMLTableElement>(null);
  const raisedRef = useRef(0);
  const system = data.system;

  const all = view?.items ?? [];
  const rows = useMemo(() => {
    const listed = all.filter(r => (!kind || r.kind === kind) && (!system || r.system_id === system) && (showClosed || !CLOSED.has(r.status)) && (!only || only.ids.has(r.id)));
    if (sort) return [...listed].sort((a, b) => sort.dir * compare(sort.col, a, b));
    // Conditions group by gate, in the order a project reaches them.
    const conditions = kind === "condition" || (listed.length > 0 && listed.every(r => r.kind === "condition"));
    return conditions ? [...listed].sort(conditionOrder) : listed;
  }, [all, kind, system, showClosed, only, sort]);
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
    if (!raise || raise.n === raisedRef.current) return;
    raisedRef.current = raise.n;
    void run([{ op: "delivery.add", fields: { kind: "rfi", title: raise.title.slice(0, 200), raised_on: sydneyToday(), source_refs: [{ document_id: raise.documentId }] } }], "Raised an RFI from the document.");
  }, [raise, run]);

  const open = (id: string) => { setSelection(clickRow(emptySelection, id, all.map(r => r.id))); onOpen(id); };
  useChatTable("register", "correspondence", {
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
    if ((e.target as HTMLElement).closest("input,select,textarea,button")) return;
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
  // The grey row raises an RFI, or records an EOT claim, a notice of delay
  // or a letter as received today, or as raised by the host today; the
  // sheet takes the rest.
  const addRow = () => {
    const title = ghost.trim();
    if (!title) return;
    setGhost("");
    if (ghostKind === "rfi") { void run([{ op: "delivery.add", fields: { kind: "rfi", title, raised_on: sydneyToday() } }], "Added an RFI."); return; }
    const what = ghostKind === "eot_claim" ? "EOT claim" : (KIND[ghostKind]?.[0] ?? "item").toLowerCase();
    void run([{ op: "delivery.add", fields: { kind: ghostKind, title, received_on: sydneyToday(), ...(ghostDir === "out" ? { direction: "out" } : {}) } }],
      ghostDir === "out" ? `Raised the ${what}.` : `Recorded the ${what}.`);
  };
  const openTerms = () => setTerms({ rfi: view && !view.response_typical ? String(view.response_days) : "", eot: view && !view.eot_typical ? String(view.eot_days) : "", agreed: view?.completion?.agreed ?? "" });
  const saveTerms = (t: Terms) => {
    const fields: Record<string, unknown> = {};
    const rfi = t.rfi.trim() ? Number(t.rfi) : null, eot = t.eot.trim() ? Number(t.eot) : null, agreed = t.agreed || null;
    if (rfi !== (view?.response_typical ? null : view?.response_days)) fields.rfi_response_days = rfi;
    if (eot !== (view?.eot_typical ? null : view?.eot_days)) fields.eot_response_days = eot;
    if (agreed !== (view?.completion?.agreed ?? null)) fields.agreed_completion = agreed;
    setTerms(null);
    if (Object.keys(fields).length) void run([{ op: "delivery.settings", fields }], "Saved the project's terms.");
  };

  const editor = (r: DeliveryRecord, field: Field, value: string) => {
    const options = field === "system" ? [{ id: "", label: "Not set" }, ...(view?.systems ?? [])] : field === "status" ? (STATUSES[r.kind] ?? []).map(s => ({ id: s, label: words(s) }))
      : field === "due" && r.kind === "condition" ? [{ id: "", label: "Not known" }, ...GATES.map(([id, label]) => ({ id, label }))] : undefined;
    return <InlineCellEditor value={value} label={`${r.title} ${field}`} error={edit?.error ?? ""} busy={busy} maxLength={200} options={options}
      onCommit={(text, key) => edit && void commit(edit, text, key)} onCancel={() => { setEdit(null); refocus(); }} />;
  };
  const start = (r: DeliveryRecord, field: Field) => { if (busy) return; setSelection(clickRow(emptySelection, r.id, order)); setEdit({ id: r.id, field }); };
  const due = (r: DeliveryRecord) => {
    if (!r.target_date) return "—";
    const late = r.late_days ? <span className="register-late"> · {r.late_days} {r.late_days === 1 ? "day" : "days"} late</span> : null;
    return <>{r.target_date}{late}</>;
  };
  const typicalDue = (r: DeliveryRecord) => r.due_basis === "default";
  const dueTitle = (r: DeliveryRecord) => !r.due_basis || r.due_basis === "typed" ? undefined
    : r.kind === "eot_claim" ? `${view?.eot_days} days after it was ${raised(r) ? "lodged" : "received"} (${r.due_basis === "default" ? "SiteWise's default; check your contract" : "the project's period"})`
      : `${view?.response_days} working days after it was ${r.kind === "rfi" ? "raised" : raised(r) ? "sent" : "received"} (${r.due_basis === "default" ? "typical" : "the project's period"})`;
  const calendarWords = view?.calendar === "weekdays" ? "Monday to Friday; there is no Program" : "the Program's working days";

  // One cell per column. The Subject cell also hosts the system editor,
  // since the system is not a column.
  const cell = (r: DeliveryRecord, col: Col): ReactNode => {
    const editing = edit?.id === r.id ? edit.field : null;
    switch (col) {
      case "type": return <td key={col} className="register-type">{KIND[r.kind]?.[0] ?? r.kind}</td>;
      case "no": return <th key={col} scope="row" className="register-no">{r.kind === "condition" ? cond(r).number : numberLabel(r.kind, r.number) || "—"}</th>;
      case "subject": {
        const ed = editing === "title" || editing === "system" ? editing : null;
        return <td key={col} data-field="title">
          {ed ? editor(r, ed, ed === "title" ? r.title : r.system_id ?? "") : <span className="register-value" title={r.title}>{r.title}</span>}</td>;
      }
      case "from": return <td key={col} data-field="from"><span className="register-value" title={raisedBy(r)}>{raisedBy(r) || "—"}</span></td>;
      case "with": return <td key={col} data-field="with" data-edit onDoubleClick={() => start(r, "with")}>
        {editing === "with" ? editor(r, "with", r.owner_text) : <span className="register-value" title={r.with}>{r.with || "—"}</span>}</td>;
      case "issued": return <td key={col} data-field="issued" className="register-date-cell">{issuedOn(r) || "—"}</td>;
      case "due": {
        const gate = r.kind === "condition";
        return <td key={col} data-field="due" className="register-date-cell" data-band={gate ? (r.review_status === "proposed" ? "amber" : undefined) : typicalDue(r) ? "typical" : undefined}
          title={gate ? undefined : dueTitle(r)} data-edit onDoubleClick={() => start(r, "due")}>
          {editing === "due" ? editor(r, "due", gate ? cond(r).stage ?? "" : r.target_date ?? "")
            : <span className="register-value">{gate ? GATE_WORDS[cond(r).stage ?? ""] ?? "Gate not known" : due(r)}</span>}</td>;
      }
      case "status": {
        const check = r.kind === "condition" ? r.review_status === "proposed" : (r.readings?.length ?? 0) > 0;
        const d = r.details as ClaimDetails;
        const words = r.kind === "condition" ? conditionStatus(r) : r.kind === "eot_claim" && d.result ? `${statusWords(r.status)} · ${resultWords(d.result, d.days_claimed)}` : statusWords(r.status);
        const editable = r.kind !== "rfi" && r.kind !== "notice";
        return <td key={col} data-field="status" data-band={check ? "amber" : undefined} data-edit={editable || undefined} onDoubleClick={() => editable && start(r, "status")}>
          {editing === "status" ? editor(r, "status", r.status) : <span className="register-value">{words}
            {r.kind === "condition" && r.review_status === "proposed" ? <span className="register-check"> · to check</span> : null}
            {r.kind !== "condition" && r.readings?.length ? <span className="register-check"> · {r.readings.length} to check</span> : null}</span>}</td>;
      }
    }
  };
  const sortBy = (col: Col) => setSort(s => s?.col === col ? { col, dir: s.dir === 1 ? -1 : 1 } : { col, dir: 1 });
  const moveColumn = (from: Col, to: Col) => {
    if (from === to) return;
    const next = columns.filter(c => c !== from);
    next.splice(next.indexOf(to) + (columns.indexOf(from) < columns.indexOf(to) ? 1 : 0), 0, from);
    setColumns(next); storeOrder(next);
  };
  const span = columns.length + 1;

  const also = system ? data.alsoOpen : undefined;
  const filters = [kind && (KIND[kind]?.[1] ?? kind), system && `on ${systemLabel(system)}`].filter(Boolean).join(" ");
  return <section className="register-screen register-list" aria-label="Register" aria-busy={busy}>
    {/* The Kind and System filters are gone for now (owner, 25 Nov 2026);
        Show closed is a switch in the three dots. */}
    {head(<MoreMenu label="Register actions" items={[{ label: "Read conditions…", icon: null, disabled: busy || documents.length === 0, onSelect: () => setReadFrom(documents[0]?.id ?? "") },
      { label: "Project terms…", icon: null, onSelect: openTerms },
      { label: "Show closed", icon: null, checked: showClosed, onSelect: () => setShowClosed(!showClosed) },
      ...(columns.join() !== DEFAULT_ORDER.join() ? [{ label: "Reset columns", icon: null, onSelect: () => { setColumns(DEFAULT_ORDER); storeOrder(DEFAULT_ORDER); } }] : [])]} />)}
    {terms !== null && <TermsForm terms={terms} calendarWords={calendarWords} busy={busy} onChange={setTerms} onSave={saveTerms} onCancel={() => setTerms(null)} />}
    {readFrom !== null && <form className="register-days" onSubmit={e => { e.preventDefault(); if (readFrom) void readNotice(readFrom); }}>
      <label>Read the consent conditions of <select aria-label="Notice to read" value={readFrom} onChange={e => setReadFrom(e.target.value)}>
        {documents.map(d => <option key={d.id} value={d.id}>{d.title}</option>)}</select></label>
      <button className="btn btn-small" disabled={busy || !readFrom}>Read</button> <button type="button" className="cell-link" onClick={() => setReadFrom(null)}>Cancel</button>
    </form>}
    {error && last?.from !== "sheet" && <div className="banner" role="alert"><p>{error}</p></div>}
    {view && (kind === "condition" || (!kind && all.some(r => r.kind === "condition")) || (view.consents ?? []).some(c => c.status !== "done")) && <ConsentHead view={view} />}
    {last && last.from === "list" && <p className="reg-notice" role="status">{last.label} <button type="button" className="cell-link" disabled={busy} onClick={data.undo}>Undo</button></p>}
    {filters && !only && <p className="reg-filter">Showing {filters} · <button type="button" className="cell-link" onClick={() => { setKind(""); data.setSystem(""); }}>Clear</button></p>}
    {only && <p className="reg-filter">{showingLine("register", only.ids.size, only.shown)} · <button type="button" className="cell-link" onClick={() => setOnly(null)}>Clear</button></p>}
    {view && (all.some(r => r.kind === "eot_claim") || view.completion?.agreed) && <CompletionLine c={view.completion} onSet={openTerms} />}
    {also && <p className="register-also">Also open on {also.label}: <button type="button" className="cell-link" onClick={() => onShowView("specifications")}>{also.spec_unknowns} specification {also.spec_unknowns === 1 ? "row" : "rows"} not settled</button>
      {" · "}<button type="button" className="cell-link" onClick={() => onShowView("procurement")}>{also.exclusions} tender {also.exclusions === 1 ? "exclusion" : "exclusions"}</button>
      {also.main_works_exclusions > 0 && <>{" · "}<button type="button" className="cell-link" onClick={() => onShowView("procurement")}>Main Works lists {also.main_works_exclusions}, not sorted by system</button></>}</p>}
    {!view ? !error && <p role="status"><Pending delayed>Loading the register…</Pending></p> : <div className="register-scroll">
      <table ref={tableRef} className="register-table" tabIndex={0} onKeyDown={keys} data-testid="register-table"
        aria-label="Register. Click to open, Shift or Ctrl to select more, F2 to edit, right-click for actions. Click a heading to sort; drag it to move the column.">
        <colgroup>{columns.map(c => <col key={c} className={`register-col-${c}`} />)}<col className="register-col-more" /></colgroup>
        <thead><tr>{columns.map(c => <th key={c} scope="col" aria-sort={sort?.col === c ? (sort.dir === 1 ? "ascending" : "descending") : undefined}
          draggable data-dragging={dragging === c ? "true" : undefined}
          onDragStart={e => { setDragging(c); e.dataTransfer.effectAllowed = "move"; e.dataTransfer.setData("text/x-sitewise-column", c); }}
          onDragOver={e => { if (dragging) { e.preventDefault(); e.dataTransfer.dropEffect = "move"; } }}
          onDrop={e => { e.preventDefault(); if (dragging) moveColumn(dragging, c); setDragging(null); }}
          onDragEnd={() => setDragging(null)}
          title={c === "due" ? `RFI response time ${view.response_days} working days${view.response_typical ? " (typical)" : ""}, on ${calendarWords}` : undefined}>
          <button type="button" className="register-sort" onClick={() => sortBy(c)}>{COLUMN[c]}<span className="register-sort-mark" aria-hidden="true">{sort?.col === c ? (sort.dir === 1 ? "▲" : "▼") : ""}</span></button></th>)}
          <th scope="col"><span className="sr-only">Actions</span></th></tr></thead>
        <tbody>
          {all.length === 0 && <tr className="register-empty"><td colSpan={span}>Nothing is recorded yet.</td></tr>}
          {rows.map((r, i) => <Fragment key={r.id}>{r.kind === "condition" && rows.every(x => x.kind === "condition") && !sort && (i === 0 || (cond(rows[i - 1]).stage ?? "") !== (cond(r).stage ?? "")) &&
            <tr className="register-group"><th colSpan={span} scope="rowgroup">{GATE_WORDS[cond(r).stage ?? ""] ?? "Gate not known"}</th></tr>}<tr data-id={r.id} aria-selected={selected.has(r.id)} tabIndex={-1}
            data-raised={raised(r) ? "true" : undefined}
            title={systemLabel(r.system_id) ? `System: ${systemLabel(r.system_id)}` : undefined}
            onClick={e => {
              if ((e.target as HTMLElement).closest("input,select,button")) return;
              const more = e.ctrlKey || e.metaKey || e.shiftKey;
              setSelection(s => clickRow(s, r.id, order, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey }));
              // With, Due and Status edit on a double click; a click there
              // only selects. Anywhere else, a click drills in.
              if (!more && !(edit?.id === r.id) && !(e.target as HTMLElement).closest("td[data-edit]")) onOpen(r.id);
            }}
            onContextMenu={e => { if ((e.target as HTMLElement).closest("input")) return; e.preventDefault(); openMenu(r.id, e.clientX, e.clientY, e.currentTarget); }}>
            {columns.map(c => cell(r, c))}
            <td className="register-more"><button type="button" className="reg-more-btn" aria-label={`Actions for ${r.title}`} aria-haspopup="menu"
              onClick={e => { e.stopPropagation(); const b = e.currentTarget.getBoundingClientRect(); openMenu(r.id, b.left, b.bottom, e.currentTarget); }}><IconMore /></button></td>
          </tr></Fragment>)}
          <tr className="register-ghost"><td colSpan={span}><span className="register-ghost-row">
            <select className="register-ghost-kind" value={ghostKind} disabled={busy} aria-label="Add as" onChange={e => setGhostKind(e.target.value)}>
              <option value="rfi">RFI</option><option value="eot_claim">EOT claim</option><option value="delay_notice">Notice of delay</option><option value="letter">Letter</option>
            </select>
            {ghostKind !== "rfi" && <select className="register-ghost-kind" value={ghostDir} disabled={busy} aria-label="Direction" onChange={e => setGhostDir(e.target.value as "in" | "out")}>
              <option value="in">Received</option><option value="out">Raised by us</option>
            </select>}
            <input value={ghost} disabled={busy} maxLength={200} placeholder={ghostWords(ghostKind, ghostDir)} aria-label={ghostWords(ghostKind, ghostDir).replace("…", "")}
              onChange={e => setGhost(e.target.value)} onBlur={addRow}
              onKeyDown={e => { e.stopPropagation(); if (e.key === "Enter") { e.preventDefault(); addRow(); } else if (e.key === "Escape") setGhost(""); }} />
          </span></td></tr>
        </tbody>
      </table>
    </div>}
    {menu && <RowMenu at={menu} items={menuItems(menu.ids)} label="Register actions" head={menu.ids.length > 1 ? `${menu.ids.length} rows` : rowOf(menu.ids[0])?.title}
      onClose={restore => { setMenu(null); if (restore) menu.trigger?.focus({ preventScroll: true }); }} />}
  </section>;
}

type Terms = { rfi: string; eot: string; agreed: string };

const GHOST_IN: Record<string, string> = { rfi: "Raise an RFI…", eot_claim: "Record an EOT claim received today…", delay_notice: "Record a notice of delay received today…", letter: "Record a letter received today…" };
const GHOST_OUT: Record<string, string> = { eot_claim: "Raise an EOT claim…", delay_notice: "Give a notice of delay…", letter: "Write a letter…" };
const ghostWords = (kind: string, dir: "in" | "out") => (dir === "out" && GHOST_OUT[kind]) || GHOST_IN[kind];

/** "Monday 21 December 2026" from YYYY-MM-DD, read as a calendar date. */
export function longDate(iso?: string | null): string {
  if (!iso) return "";
  const [y, m, d] = iso.split("-").map(Number);
  return new Date(Date.UTC(y, m - 1, d)).toLocaleDateString("en-AU", { weekday: "long", day: "numeric", month: "long", year: "numeric", timeZone: "UTC" });
}

// CompletionLine is the agreed completion date and what the EOTs granted
// have done to it, every figure Go's. With no agreed date it says so.
function CompletionLine({ c, onSet }: { c?: Completion; onSet: () => void }) {
  const granted = c?.days_granted ?? 0;
  const days = `${granted} ${granted === 1 ? "day" : "days"} granted`;
  return <p className="register-completion" role="status">
    {c?.agreed
      ? <>Agreed completion {longDate(c.agreed)} · {days}{granted > 0 ? <> · now <strong>{longDate(c.adjusted)}</strong></> : null}</>
      : <>No agreed completion date is recorded, so nothing has moved · {days} · <button type="button" className="cell-link" onClick={onSet}>Set the date</button></>}
  </p>;
}

// TermsForm is the project's optional terms. Each may be left blank: a
// blank period is SiteWise's default, and a blank date is unknown.
function TermsForm({ terms, calendarWords, busy, onChange, onSave, onCancel }: {
  terms: Terms; calendarWords: string; busy: boolean;
  onChange: (t: Terms) => void; onSave: (t: Terms) => void; onCancel: () => void;
}) {
  const rfi = terms.rfi.trim() ? Number(terms.rfi) : null, eot = terms.eot.trim() ? Number(terms.eot) : null;
  const bad = (rfi !== null && !(Number.isInteger(rfi) && rfi >= 1 && rfi <= 60)) || (eot !== null && !(Number.isInteger(eot) && eot >= 1 && eot <= 120));
  return <form className="register-days register-terms" onSubmit={e => { e.preventDefault(); if (!bad) onSave(terms); }}>
    <p className="register-muted">Optional. A blank period is SiteWise's default, not your contract's.</p>
    <label>RFIs and letters <input type="number" min={1} max={60} value={terms.rfi} placeholder="5" autoFocus
      onChange={e => onChange({ ...terms, rfi: e.target.value })} aria-label="RFI and letter response time in working days" /> working days, on {calendarWords}</label>
    <label>EOT claims <input type="number" min={1} max={120} value={terms.eot} placeholder="28"
      onChange={e => onChange({ ...terms, eot: e.target.value })} aria-label="EOT claim response time in calendar days" /> days from receipt</label>
    <label>Agreed completion <input type="date" className="register-date" value={terms.agreed} onChange={e => onChange({ ...terms, agreed: e.target.value })} aria-label="Agreed completion date" /></label>
    <p><button className="btn btn-small" disabled={busy || bad}>Save</button> <button type="button" className="cell-link" onClick={onCancel}>Cancel</button></p>
  </form>;
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
