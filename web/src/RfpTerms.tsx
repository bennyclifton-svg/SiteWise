import { Fragment, useEffect, useRef, useState, type DragEvent, type KeyboardEvent, type MouseEvent, type ReactNode } from "react";
import { FILED_DOCUMENT_TYPE } from "./documentDrag";
import { IconBin, IconRename, IconSelectAll, IconUndo, IconUpdate } from "./icons";
import { InlineText } from "./InlineText";
import type { ReportBlock, RfpControl, RfpControls } from "./reportApi";
import { blockedKeys, type SettingChange, type Write } from "./RfpEditor";
import { Stepper } from "./RfpAllowanceTable";
import type { TransmittalSelection } from "./RfpTransmittal";
import { RowMenu, type MenuItem, type RowMenuAt } from "./table/RowMenu";
import { clickRow, emptySelection, prune, type Selection } from "./table/selection";
import { termParts, withSlots, type TermPart } from "./termWording";
import "./rfp-tables.css";

/** Saved drafts from before 8 Nov 2026 still hold this block; it no longer prints. */
export const AGREEMENT_BLOCK = "rfp:terms:agreement";
const AGREEMENT_DOCS = "agreement_docs";
const AGREEMENT = "field:agreement";
const CUSTOM = "custom:terms";

/** One term as internal/rfp resolves it (rfp.Row). */
type Term = { key: string; label?: string; text: string; template?: string; edited?: boolean; custom?: boolean; held?: boolean };
type Own = { id: string; text: string; value: "I" };

const newTermID = () => "u" + Date.now().toString(36);
/** Wording as saved: one line, no braces of its own. */
const cleanText = (text: string) => text.replace(/\s+/g, " ").trim().replace(/[{}]/g, "");
/** A combined engagement labels a term some members carry; the label leads the printed text. */
const wording = (t: Term) => t.label && t.text.startsWith(`${t.label}: `) ? t.text.slice(t.label.length + 2) : t.text;
const rowID = (t: Term) => t.label ? `${t.key}@${t.label}` : t.key;
const slotsOf = (t: Term) => [...(t.template ?? "").matchAll(/\{(field:[a-z_0-9]+)\}/g)].map(m => m[1]);

/**
 * Terms as one table, No. | Term (owner direction, 10 November 2026),
 * ranked by importance with the client agreement first. Each answer (the
 * agreement, insurance, days) is edited inside the term that prints it.
 * Rows select, edit and delete as the stage table's lines do: a click
 * selects, Ctrl and Shift extend, a double-click (or Enter or F2) edits the
 * wording in place, Delete or the bin removes. "Add term…" adds one. A term
 * with an optional blank shows unnumbered until filled; only then it prints.
 */
export function RfpTermsTable({ controls, block, selection: repo, busy, onWrite }: { controls: RfpControls; block?: ReportBlock; selection?: TransmittalSelection; busy: boolean; onWrite: Write }) {
  const [edit, setEdit] = useState<{ id: string; text: string } | null>(null);
  const [ghost, setGhost] = useState("");
  const [selection, setSelection] = useState<Selection>(emptySelection);
  const [menu, setMenu] = useState<(RowMenuAt & { ids: string[] }) | null>(null);
  const [over, setOver] = useState(false);
  const scroller = useRef<HTMLDivElement>(null);
  const rows = (block?.basis?.rows ?? []) as Term[];
  const fields = new Map((controls.groups.find(g => g.id === "terms")?.controls ?? []).map(c => [c.key, c]));
  const blocked = blockedKeys(controls);
  const saved = (block?.basis?.agreement_ids ?? []) as string[];
  const order = rows.map(rowID);
  const orderKey = order.join("|");
  useEffect(() => setSelection(s => prune(s, orderKey ? orderKey.split("|") : [])), [orderKey]);
  const selected = new Set(selection.ids);
  // A click anywhere but a selected row, the bin or the menu clears the selection.
  useEffect(() => {
    if (!selection.ids.length) return;
    const away = (e: PointerEvent) => {
      const target = e.target as HTMLElement;
      if (target.closest(".reg-menu, .rfp-scope-bin") || (scroller.current?.contains(target) && target.closest("tr.rfp-term"))) return;
      setSelection(emptySelection);
    };
    document.addEventListener("pointerdown", away, true);
    return () => document.removeEventListener("pointerdown", away, true);
  }, [selection.ids.length]);

  // A required field no term holds (its term deleted or reworded without
  // it) stays reachable below the terms while it blocks issue.
  const placed = new Set(rows.flatMap(slotsOf));
  const loose = [...fields.values()].filter(c => !placed.has(c.key) && blocked.has(c.key));
  if (!rows.length && !loose.length) return null;

  const own: Own[] = rows.filter(t => t.custom).map(t => ({ id: t.key, text: t.text, value: "I" }));
  const byID = new Map(rows.map(t => [rowID(t), t]));
  const write = (changes: SettingChange[]) => {
    if (!changes.length) return;
    if (onWrite.many) void onWrite.many(changes);
    else void (async () => { for (const c of changes) await onWrite(c.key, c.value); })();
  };
  const save = (t: Term, raw: string) => {
    setEdit(null);
    const text = cleanText(raw);
    if (!text || text === wording(t)) return;
    if (t.custom) return void onWrite(CUSTOM, own.map(o => o.id === t.key ? { ...o, text } : o));
    // A field whose value still reads in the new wording stays a field.
    void onWrite(`wording:terms.${t.key}`, withSlots(text, termParts(t.template ?? "", wording(t)) ?? []));
  };
  const add = () => {
    const text = cleanText(ghost);
    if (!text) return;
    setGhost("");
    void onWrite(CUSTOM, [...own, { id: newTermID(), text, value: "I" }]);
  };
  // Deleting is final, as for scope lines: a standard term leaves this RFP
  // and the user's own term is removed, all in one write.
  const remove = (ids: string[]) => {
    const changes: SettingChange[] = [];
    const gone = new Set<string>();
    for (const id of ids) {
      const t = byID.get(id);
      if (!t) continue;
      if (t.custom) { gone.add(t.key); continue; }
      changes.push({ key: `term:${t.key}`, value: "D" });
      if (t.edited) changes.push({ key: `wording:terms.${t.key}`, value: null });
    }
    if (gone.size) {
      const kept = own.filter(o => !gone.has(o.id));
      changes.push({ key: CUSTOM, value: kept.length ? kept : null });
    }
    setSelection(emptySelection);
    write(changes);
  };
  const menuItems = (ids: string[]): MenuItem[] => {
    const one = ids.length === 1 ? byID.get(ids[0]) : undefined;
    const out: MenuItem[] = [];
    if (one) out.push({ label: "Edit wording", icon: <IconRename />, disabled: busy, onSelect: () => setEdit({ id: ids[0], text: wording(one) }) });
    if (one?.edited) out.push({ label: "Reset to standard wording", icon: <IconUndo />, disabled: busy, onSelect: () => void onWrite(`wording:terms.${one.key}`, null) });
    out.push("sep");
    out.push({ label: ids.length === 1 ? "Delete term" : `Delete ${ids.length} terms`, icon: <IconBin />, danger: true, disabled: busy, onSelect: () => remove(ids) });
    return out;
  };
  // A click selects the row; Ctrl adds or removes it, Shift selects a range.
  const rowClick = (id: string) => (e: MouseEvent<HTMLTableRowElement>) => {
    if ((e.target as HTMLElement).closest("input,button,textarea,select,label,a")) return;
    if (e.shiftKey) { e.preventDefault(); window.getSelection()?.removeAllRanges(); }
    setSelection(s => clickRow(s, id, order, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey }));
    e.currentTarget.focus({ preventScroll: true });
  };
  // Right-click acts on the selection when the row is in it; otherwise the
  // row becomes the selection.
  const rowMenu = (id: string) => (e: MouseEvent<HTMLTableRowElement>) => {
    if ((e.target as HTMLElement).closest("textarea,input")) return;
    e.preventDefault();
    const ids = selected.has(id) ? [...selection.ids] : [id];
    if (!selected.has(id)) setSelection(clickRow(emptySelection, id, order));
    setMenu({ x: e.clientX, y: e.clientY, trigger: e.currentTarget, ids });
  };
  const keys = (e: KeyboardEvent) => {
    if ((e.target as HTMLElement).closest("textarea,input")) return;
    if ((e.key === "Delete" || e.key === "Backspace") && selection.ids.length && !busy) { e.preventDefault(); remove([...selection.ids]); }
    if (e.key === "Escape" && selection.ids.length) setSelection(emptySelection);
    // Enter or F2 edits the one selected row, as in the Cost Plan.
    if ((e.key === "Enter" || e.key === "F2") && selection.ids.length === 1 && !busy && !edit) {
      const t = byID.get(selection.ids[0]);
      if (t) { e.preventDefault(); setEdit({ id: selection.ids[0], text: wording(t) }); }
    }
  };

  // The agreement is named by the titles of the documents chosen for it:
  // pick them in the repository, or drag one onto its row.
  const filed = (e: DragEvent) => e.dataTransfer.types.includes(FILED_DOCUMENT_TYPE);
  const dropAgreement = {
    onDragOver: (e: DragEvent) => { if (filed(e)) { e.preventDefault(); e.stopPropagation(); e.dataTransfer.dropEffect = "link"; setOver(true); } },
    onDragLeave: () => setOver(false),
    onDrop: (e: DragEvent) => {
      if (!filed(e)) return;
      e.preventDefault(); e.stopPropagation(); setOver(false);
      const id = e.dataTransfer.getData(FILED_DOCUMENT_TYPE);
      if (id && !saved.includes(id) && !busy) void onWrite(AGREEMENT_DOCS, [...saved, id]);
    },
  };
  const agreementActions = repo && <span className="rfp-terms-agreement-actions">
    {saved.length > 0 && <button type="button" className="btn btn-small" disabled={busy} title="Select the agreement's documents in the repository" onClick={() => repo.onShowDocuments(saved)}><IconSelectAll />Show in repository</button>}
    <button type="button" className="btn btn-small" disabled={busy || (!repo.selectedIds.length && !saved.length)} title={repo.selectedIds.length ? `Use the ${repo.selectedIds.length} documents selected in the repository` : "Select the agreement's documents in the repository first"} onClick={() => void onWrite(AGREEMENT_DOCS, [...repo.selectedIds])}><IconUpdate />Choose agreement</button>
  </span>;
  const field = (slot: string, value: string, key: number) => <TermField key={key} slot={slot} value={value} control={fields.get(slot)} busy={busy} onWrite={onWrite} />;

  let number = 0;
  return <div className="report-table-scroll" ref={scroller} onKeyDown={keys}><table className="report-table rfp-terms-table">
    <colgroup><col className="rfp-terms-no-col" /><col /></colgroup>
    <thead><tr><th scope="col" className="rfp-terms-no">No.</th><th scope="col"><span className="rfp-scope-head">Term
      {/* The bin shows while rows are selected, as in the stage table. */}
      {selection.ids.length > 0 && <button type="button" className="rfp-scope-bin" disabled={busy} title={`Delete ${selection.ids.length} selected`}
        aria-label={`Delete ${selection.ids.length} selected ${selection.ids.length === 1 ? "term" : "terms"}`} onClick={() => remove([...selection.ids])}>
        <IconBin /><span aria-hidden="true">{selection.ids.length}</span></button>}</span></th></tr></thead>
    <tbody>
      {rows.map(t => {
        const id = rowID(t);
        const shown = wording(t);
        const parts = t.template ? termParts(t.template, shown) : null;
        const slots = slotsOf(t);
        const agreement = slots.includes(AGREEMENT);
        const editing = edit?.id === id;
        return <tr key={id} className={`rfp-term rfp-line${agreement ? " rfp-terms-agreement" : ""}`} data-term={t.key} data-held={t.held || undefined}
          data-edited={t.edited || undefined} data-blocked={slots.some(s => blocked.has(s)) || undefined} data-drop={(agreement && over) || undefined}
          data-selected={selected.has(id) || undefined} aria-selected={selected.has(id)} tabIndex={-1}
          title={t.held ? "Prints once its blank is filled" : undefined}
          onClick={rowClick(id)} onContextMenu={rowMenu(id)} {...(agreement ? dropAgreement : {})}>
          <td className="rfp-terms-no">{t.held ? "" : ++number}</td>
          <td className="rfp-term-text rfp-line-text"><div className="rfp-term-line">
            {t.label && <span className="rfp-term-owners">{t.label}: </span>}
            <InlineText compact text={shown} label={`Term: ${shown}`} editing={editing} value={editing ? edit.text : shown} maxLength={600}
              display={parts ? parts.map((p: TermPart, i) => "slot" in p ? field(p.slot, p.value, i) : <Fragment key={i}>{p.text}</Fragment>) : undefined}
              disabled={busy || (!!edit && !editing)} busy={busy}
              onStart={() => { setSelection(clickRow(emptySelection, id, order)); setEdit({ id, text: shown }); }}
              onChange={text => setEdit({ id, text })} onSave={() => save(t, edit?.text ?? "")} onCancel={() => setEdit(null)}
              actions={agreement ? agreementActions : undefined} />
          </div></td>
        </tr>;
      })}
      {loose.map(c => <tr key={c.key} className="rfp-terms-loose" data-blocked="true" {...(c.key === AGREEMENT ? dropAgreement : {})}>
        <td className="rfp-terms-no" />
        <td className="rfp-term-text"><div className="rfp-term-line"><span>{c.label}: {field(c.key, "", 0)}</span>{c.key === AGREEMENT && agreementActions}</div></td>
      </tr>)}
      {/* The ghost line: a quiet "Add term…" that becomes a term on Enter or
          when clicked away from, as the stage table's "Add item…". */}
      <tr className="rfp-terms-add"><td className="rfp-terms-no" />
        <td><textarea className="rfp-ghost" rows={1} aria-label="New term" placeholder="Add term…" maxLength={600} disabled={busy}
          value={ghost} onChange={e => { setGhost(e.target.value.replace(/\n/g, " ")); grow(e.target); }}
          onKeyDown={e => { if (e.key === "Enter") { e.preventDefault(); add(); } if (e.key === "Escape") setGhost(""); }}
          onBlur={add} /></td>
      </tr>
    </tbody>
  </table>
  {menu && <RowMenu at={menu} items={menuItems(menu.ids)} label={menu.ids.length > 1 ? `Actions for ${menu.ids.length} terms` : "Term actions"}
    head={menu.ids.length > 1 ? `${menu.ids.length} terms selected` : undefined} onClose={restore => { if (restore) menu.trigger?.focus({ preventScroll: true }); setMenu(null); }} />}
  </div>;
}

/** The ghost line grows with its text: no inner scroll bar. */
function grow(el: HTMLTextAreaElement) {
  el.style.height = "auto";
  el.style.height = `${el.scrollHeight + el.offsetHeight - el.clientHeight}px`;
}

// Where a value came from, in the app's origin colours (trace.css).
const origin = (source: string) => ({ user: "user", document: "doc" } as Record<string, string>)[source] ?? "typical";

/**
 * One field inside a term, edited where it reads: days on a stepper, a date,
 * or text that grows with its value. The agreement is the titles of its
 * documents, chosen with the row's buttons. A slot with no terms control
 * prints its value as text.
 */
function TermField({ slot, value, control: c, busy, onWrite }: { slot: string; value: string; control?: RfpControl; busy: boolean; onWrite: Write }): ReactNode {
  const current = c?.value === null || c?.value === undefined ? "" : String(c.value);
  if (slot === AGREEMENT) return <span className="rfp-term-doc" data-origin={current ? origin(c?.source ?? "") : undefined}>{current || "___"}</span>;
  if (!c || c.locked) return <span className="rfp-term-value">{value}</span>;
  const save = (v: string) => void onWrite(c.key, v === "" ? null : v);
  const tone = current ? origin(c.source) : undefined;
  if (c.kind === "number") return <span className="rfp-term-number" data-origin={tone}><Stepper label={c.label} name={c.label} value={current} placeholder="___" busy={busy} onCommit={save} /></span>;
  return <TextField label={c.label} type={c.kind === "date" ? "date" : "text"} value={current} tone={tone} busy={busy} onCommit={save} />;
}

// Typed text commits on blur or Enter, so a half-typed value is never saved.
function TextField({ label, type, value, tone, busy, onCommit }: { label: string; type: string; value: string; tone?: string; busy: boolean; onCommit: (v: string) => void }) {
  const [draft, setDraft] = useState(value);
  useEffect(() => setDraft(value), [value]);
  const commit = () => { if (draft.trim() !== value) onCommit(draft.trim()); };
  return <span className={`rfp-term-field${type === "date" ? " rfp-term-date" : ""}`} data-value={draft || "___"} data-origin={tone}>
    <input type={type} aria-label={label} value={draft} placeholder="___" size={1} disabled={busy}
      onChange={e => setDraft(e.target.value)} onBlur={commit}
      onKeyDown={e => {
        if (e.key === "Enter") { e.preventDefault(); e.currentTarget.blur(); }
        if (e.key === "Escape") { setDraft(value); }
      }} />
  </span>;
}
