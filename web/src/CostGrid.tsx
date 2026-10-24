import { IconBin } from "./icons";
import { clickRow, emptySelection, prune, selectAll } from "./table/selection";
import { InlineCellEditor } from "./table/InlineCellEditor";
import { RowHandle, startRowDrag } from "./table/RowHandle";
import { useEffect, useId, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";
import type { BudgetCheck, ConstructionBudget, CostContent, CostItem, CostPlan, CostTotal, CostTotals, CostValue } from "./costApi";
import type { PackageRecord } from "./packageApi";
import type { Ledger, LedgerLine } from "./invoiceApi";

// Money arrives as exact decimal strings. Group digits without parsing to a
// float so the grid shows what the ledger holds.
export const fmt = (m: string | null | undefined) => {
  if (m == null) return "—";
  const negative = m.startsWith("-");
  const [whole, cents = "00"] = m.replace("-", "").split(".");
  return `${negative ? "−" : ""}${whole.replace(/\B(?=(\d{3})+(?!\d))/g, ",")}.${cents.padEnd(2, "0")}`;
};
export const cell = (v?: CostValue) => v?.value_state !== "known" ? "—" : v.amount != null ? fmt(v.amount) : `${fmt(v.low)}–${fmt(v.high)}`;
// A subtotal with no known line is unknown, not a zero with a footnote.
export const subtotal = (t?: CostTotal) => !t || t.lines === 0 || t.unknown === t.lines ? "—" : t.amount != null ? fmt(t.amount) : t.unknown > 0 ? `${fmt(t.known_subtotal)} + ${t.unknown} unknown` : `${fmt(t.low)}–${fmt(t.high)}`;
const words = (s: string) => s.replaceAll("_", " ");

const GROUPS = [
  { kind: "project_wide", title: "Fees and charges", bucket: "authority", category: "authority_fees" },
  { kind: "fee", title: "Consultants", bucket: "consultants", category: "" },
  { kind: "works", title: "Construction", bucket: "construction", category: "" },
  { kind: "project_wide", title: "Contingency", bucket: "contingency", category: "contingency" },
] as const;
// Placeholder columns keep the cost report's shape until variations and monthly
// claims are tracked; they never show a figure.
const COLUMNS = ["Budget", "Approved Contract", "Forecast Variations", "Approved Variations", "Forecast Final Cost", "Budget Variance", "Claimed to Date", "This Month", "Remaining"];
const PLACEHOLDER = new Set(["Forecast Variations", "Approved Variations", "This Month"]);
type SortKey = "category" | "item";

// Ledger columns come from the invoice ledger, computed on the server: the
// same figures the Invoices tab sums, so the two views cannot disagree.
const LEDGER = ["Invoiced to Date", "Budget Remaining", "% Spent"];
function LedgerCells({ line, onInvoices }: { line?: LedgerLine; onInvoices?: (id: string) => void }) {
  if (!line) return <><td className="cost-num cost-ledger">—</td><td className="cost-num cost-ledger">—</td><td className="cost-num cost-ledger">—</td></>;
  const over = line.over_budget ? " cost-over" : "";
  return <><td className={`cost-num cost-ledger${over}`}>{onInvoices && line.invoices > 0 ? <button type="button" className="cost-open" title={`Show the ${line.invoices} invoice${line.invoices === 1 ? "" : "s"} on ${line.label}`} onClick={e => { e.stopPropagation(); onInvoices(line.id || "unallocated"); }}>{fmt(line.invoiced)}</button> : fmt(line.invoiced)}</td>
    <td className={`cost-num cost-ledger${over}`}>{fmt(line.remaining)}{line.over_budget && <span className="cost-tag">over budget</span>}</td>
    <td className={`cost-num cost-ledger${over}`}>{line.percent_spent ? `${line.percent_spent}%` : "—"}</td></>;
}

export function CostGrid({ plan, totals, packages, editable, busy, onOpen, onSubdivide, onDetailedValue, onReorder, onValue, onDelete, onItem, quantities, ledger, onInvoices, onBudget }: {
  plan: CostPlan; totals: CostTotals | null; packages: PackageRecord[]; editable: boolean;
  ledger?: Ledger | null; onInvoices?: (lineId: string) => void;
  /** Saves the construction budget to the profile; the plan re-spreads from it. */
  onBudget?: (amount: string | null) => Promise<boolean>;
  quantities: boolean; busy: boolean;
  onSubdivide: (item: CostItem) => void;
  onDetailedValue: (item: CostItem, metric: string, value: CostValue) => Promise<boolean>;
  onReorder: (ids: string[]) => Promise<boolean>;
  onDelete: (item: CostItem) => Promise<boolean>;
  onItem: (content: CostContent, id?: string, calculate?: boolean) => Promise<boolean>;
  onOpen: (id: string) => void; onValue: (item: CostItem, metric: "budget" | "commitment", amount: string | null) => Promise<boolean>;
}) {
  const [sort, setSort] = useState<{ key: SortKey; dir: 1 | -1 } | null>(null);
  const [selection, setSelection] = useState(emptySelection);
  const [drop, setDrop] = useState<{ id: string; after: boolean } | null>(null);
  const dragging = useRef(false);
  const items = plan.items;
  useEffect(() => setSelection(s => prune(s, items.map(i => i.cost_item_id))), [items]);
  const category = (i: CostItem) => i.resolved_category || (i.line_kind === "project_wide" ? words(i.category) : packages.find(p => p.id === i.package_id)?.title || "Unassigned");
  const sortValue = (i: CostItem) => sort?.key === "category" ? category(i) : i.label;
  const ordered = (list: CostItem[]) => sort ? [...list].sort((a, b) => sort.dir * sortValue(a).localeCompare(sortValue(b), undefined, { numeric: true })) : [...list].sort((a,b) => (a.display_order ?? Number.MAX_SAFE_INTEGER) - (b.display_order ?? Number.MAX_SAFE_INTEGER) || (Number(["other", "exclusions"].includes(a.category)) - Number(["other", "exclusions"].includes(b.category))) || a.code.localeCompare(b.code, undefined, {numeric:true}) || a.label.localeCompare(b.label));
  const ids = new Set(items.map(i => i.cost_item_id));
  const ledgerLine = new Map((ledger?.summary.lines ?? []).map(l => [l.id, l]));
  const width = 4 + COLUMNS.length + LEDGER.length + (quantities ? 3 : 0);
  // Display positions are not identities: links continue to use cost_item_id.
  let rowNumber = 0;
  // Children follow their group line, indented; sorting applies within a level.
  const tree = (list: CostItem[], depth: number): { item: CostItem; depth: number }[] =>
    ordered(list).flatMap(i => [{ item: i, depth }, ...tree(items.filter(c => c.parent_item_id === i.cost_item_id), depth + 1)]);
  const visible = GROUPS.flatMap(g => tree(items.filter(i => totals?.section_items[i.cost_item_id] === g.bucket && (!i.parent_item_id || !ids.has(i.parent_item_id))), 0)).map(r => r.item.cost_item_id);
  function handleDown(e: ReactPointerEvent<HTMLElement>, item: CostItem) {
    if (!editable || busy || sort) return;
    const moving = selection.ids.includes(item.cost_item_id) ? visible.filter(id => selection.ids.includes(id)) : [item.cost_item_id];
    const siblings = items.filter(i => i.parent_item_id === item.parent_item_id && totals?.section_items[i.cost_item_id] === totals?.section_items[item.cost_item_id]);
    if (moving.some(id => !siblings.some(i => i.cost_item_id === id))) return;
    if (!selection.ids.includes(item.cost_item_id)) setSelection(clickRow(emptySelection, item.cost_item_id, visible));
    let target: { id: string; after: boolean } | null = null;
    startRowDrag(e, {
      onStart: () => { dragging.current = true; },
      onMove: ev => {
        const row = document.elementFromPoint(ev.clientX, ev.clientY)?.closest<HTMLElement>(".cost-row[data-id]");
        const id = row?.dataset.id;
        target = row && id && !moving.includes(id) && siblings.some(i => i.cost_item_id === id) ? { id, after: ev.clientY > row.getBoundingClientRect().top + row.offsetHeight / 2 } : null;
        setDrop(target);
      },
      onDrop: () => {
        if (!target) return;
        const order = visible.filter(id => siblings.some(i => i.cost_item_id === id) && !moving.includes(id));
        order.splice(order.indexOf(target.id) + Number(target.after), 0, ...moving);
        void onReorder(order);
      },
      onStop: () => { setDrop(null); window.setTimeout(() => { dragging.current = false; }, 0); },
    });
  }
  const header = (key: SortKey, label: string) => <th scope="col" className="cost-sticky" data-col={key} aria-sort={sort?.key === key ? (sort.dir === 1 ? "ascending" : "descending") : undefined}>
    <button type="button" onClick={() => setSort(sort?.key === key ? (sort.dir === 1 ? { key, dir: -1 } : null) : { key, dir: 1 })}>{label}</button></th>;
  return <div className="cost-grid-scroll"><table className="cost-grid" tabIndex={0} onKeyDown={e => {
    if ((e.target as HTMLElement).closest("input,select,textarea,button,[popover]")) return;
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "a") { e.preventDefault(); setSelection(selectAll(visible)); }
    if (e.key === "Escape") setSelection(emptySelection);
  }}>
    <thead><tr><th scope="col" className="cost-sticky" data-col="code">No.</th>{header("category", "Category")}{header("item", "Item")}
      {quantities && ["Qty", "Unit", "Rate"].map(c => <th key={c} scope="col">{c}</th>)}
      {COLUMNS.map(c => PLACEHOLDER.has(c) ? <th key={c} scope="col" className="cost-num cost-placeholder" aria-disabled="true" title="Not tracked yet">{c}</th> : <th key={c} scope="col" className="cost-num">{c}</th>)}
      {LEDGER.map(c => <th key={c} scope="col" className="cost-num cost-ledger" title="From the invoice ledger, ex GST">{c}</th>)}<th scope="col" className="cost-actions">Actions</th></tr></thead>
    {GROUPS.map(g => {
      const top = items.filter(i => totals?.section_items[i.cost_item_id] === g.bucket && (!i.parent_item_id || !ids.has(i.parent_item_id)));
      const t = totals?.sections[g.bucket], d = totals?.section_figures[g.bucket];
      return <tbody key={g.bucket}>
        <tr className="cost-group"><th colSpan={width} scope="colgroup"><span>{g.title}</span></th></tr>
        {tree(top, 0).map(({ item: i, depth }) => {
          const f = totals?.items?.[i.cost_item_id];
          return <tr key={i.cost_item_id} className={`cost-row${i.excluded ? " cost-excluded" : ""}${i.posting ? "" : " cost-parent"}`} onClickCapture={e => { if ((e.ctrlKey || e.metaKey || e.shiftKey) && !(e.target as HTMLElement).closest("input,select,textarea,[popover]")) { e.stopPropagation(); setSelection(s => clickRow(s, i.cost_item_id, visible, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })); } }} data-id={i.cost_item_id} data-drop={drop?.id === i.cost_item_id ? (drop.after ? "after" : "before") : undefined} aria-selected={selection.ids.includes(i.cost_item_id)} onClick={e => { if (!dragging.current) setSelection(s => clickRow(s, i.cost_item_id, visible, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })); }}>
            <td className="cost-sticky" data-col="code">{editable && <RowHandle disabled={busy || !!sort} onPointerDown={e => handleDown(e, i)} />}{String(++rowNumber).padStart(2, "0")}</td>
            <td className="cost-sticky" data-col="category">{editable ? <CategoryCell item={i} plan={plan} packages={packages} label={category(i)} onSave={c => onItem(c, i.cost_item_id)} /> : category(i)}</td>
            <td className="cost-sticky" data-col="item" style={{ paddingLeft: `${8 + depth * 16}px` }}>{editable ? <TextCell label={i.label} value={i.label} selected={selection.ids.length === 1 && selection.ids.includes(i.cost_item_id)} onSave={label => label.trim() ? onItem(editContent(i, {label}), i.cost_item_id) : Promise.resolve(false)} /> : i.label}{i.excluded && <span className="cost-tag">excluded</span>}</td>
            {quantities && <QuantityCells key={i.cost_item_id} item={i} editable={editable && i.posting} onSave={c => onItem(c, i.cost_item_id, true)} />}
            {!i.posting && i.values.budget ? <>
              <td className="cost-num"><MoneyCell onDetailed={v => onDetailedValue(i, "budget", v)} metric="budget" item={i} editable={editable} onSave={a => onValue(i, "budget", a)} /></td>
              <td className="cost-num" title="The sum of the awarded lines below">{cell(f?.forecast)}</td>
              <td className="cost-num cost-placeholder">—</td><td className="cost-num cost-placeholder">—</td>
              <td className="cost-num">{cell(f?.forecast)}</td>
              <td className="cost-num">{fmt(f?.budget_variance)}</td>
              <td className="cost-num cost-placeholder">—</td>
              <td className="cost-num cost-placeholder">—</td>
              <td className="cost-num cost-placeholder">—</td>
              <LedgerCells line={ledgerLine.get(i.cost_item_id)} onInvoices={onInvoices} /></>
            : !i.posting ? <td colSpan={COLUMNS.length + LEDGER.length} className="cost-note">Group · amounts are on the lines below</td> : <>
              <td className="cost-num"><MoneyCell onDetailed={v => onDetailedValue(i, "budget", v)} metric="budget" item={i} editable={editable} onSave={a => onValue(i, "budget", a)} /></td>
              <td className="cost-num"><MoneyCell onDetailed={v => onDetailedValue(i, "commitment", v)} metric="commitment" item={i} editable={editable} onSave={a => onValue(i, "commitment", a)} /></td>
              <td className="cost-num cost-placeholder">—</td><td className="cost-num cost-placeholder">—</td>
              <td className="cost-num">{cell(f?.forecast)}</td>
              <td className="cost-num">{fmt(f?.budget_variance)}</td>
              <td className="cost-num"><DetailedValueCell item={i} metric="claimed_to_date" editable={editable} onSave={v => onDetailedValue(i, "claimed_to_date", v)} /></td>
              <td className="cost-num cost-placeholder">—</td>
              <td className="cost-num">{fmt(f?.remaining)}</td>
              <LedgerCells line={ledgerLine.get(i.cost_item_id)} onInvoices={onInvoices} /></>}
            <td className="cost-actions" onClick={e => e.stopPropagation()}><RowActions item={i} editable={editable} busy={busy} onOpen={() => onOpen(i.cost_item_id)} onSubdivide={() => onSubdivide(i)} onToggle={() => void onItem(editContent(i, { excluded: !i.excluded }), i.cost_item_id)} />{editable && <button type="button" className="cost-delete" disabled={busy} aria-label={`Delete ${i.label}`} title={`Delete ${i.label}`} onClick={() => void onDelete(i)}><IconBin /></button>}</td>
          </tr>;
        })}
        {editable && <AddLineRow key={g.bucket} group={g} plan={plan} packages={packages} columns={width} onSave={onItem} />}
        <tr className="cost-subtotal"><th colSpan={3} scope="row" className="cost-sticky" data-col="subtotal">{g.title} subtotal</th>{quantities && <td colSpan={3} />}
          <td className="cost-num">{subtotal(t?.budget)}</td><td className="cost-num">{subtotal(t?.commitment)}</td>
          <td className="cost-num cost-placeholder">—</td><td className="cost-num cost-placeholder">—</td>
          <td className="cost-num">{subtotal(t?.forecast)}</td><td className="cost-num">{fmt(d?.budget_variance)}</td>
          <td className="cost-num">{subtotal(t?.claimed_to_date)}</td><td className="cost-num cost-placeholder">—</td>
          <td className="cost-num">{fmt(d?.remaining)}</td>
          <td className="cost-num cost-ledger">{ledger ? fmt(ledger.summary.sections[g.bucket] ?? "0.00") : "—"}</td><td className="cost-ledger" /><td className="cost-ledger" /><td /></tr>
      </tbody>;
    })}
    <tbody className="cost-totals">
      <tr className="cost-subtotal cost-total"><th colSpan={3} scope="row" className="cost-sticky" data-col="subtotal">Total</th>{quantities && <td colSpan={3} />}
        <td className="cost-num">{subtotal(totals?.overall.budget)}</td><td className="cost-num">{subtotal(totals?.overall.commitment)}</td>
        <td className="cost-num cost-placeholder">—</td><td className="cost-num cost-placeholder">—</td>
        <td className="cost-num">{subtotal(totals?.overall.forecast)}</td><td className="cost-num">{fmt(totals?.buckets?.overall?.budget_variance)}</td>
        <td className="cost-num">{subtotal(totals?.overall.claimed_to_date)}</td><td className="cost-num cost-placeholder">—</td>
        <td className="cost-num">{fmt(totals?.buckets?.overall?.remaining)}</td>
        <td className="cost-ledger" /><td className="cost-ledger" /><td className="cost-ledger" /><td /></tr>
      {(plan.construction_budget || (editable && onBudget)) && <tr className="cost-row cost-budget-row">
        <th colSpan={3} scope="row" className="cost-sticky" data-col="subtotal">Construction budget</th>{quantities && <td colSpan={3} />}
        <td className="cost-num"><BudgetCell budget={plan.construction_budget} editable={editable && !!onBudget} onSave={a => onBudget ? onBudget(a) : Promise.resolve(false)} /></td>
        <td colSpan={COLUMNS.length - 1 + LEDGER.length} className="cost-note cost-budget-note"><span>{budgetNote(plan.construction_budget, totals?.construction_budget)}</span></td><td /></tr>}
    </tbody>
  </table></div>;
}

// budgetNote says where the construction budget came from, how it is spread
// and how much of it the construction lines hold (figures from the server).
function budgetNote(b?: ConstructionBudget, check?: BudgetCheck) {
  if (!b) return "Not set. Type the construction budget, or upload a brief that states it: the construction lines are spread from it.";
  if (!b.known) return `“${b.stated.text}” gives no single amount. Type the budget to spread it.`;
  const parts: string[] = [];
  if (check) {
    parts.push(`Allocated ${fmt(check.allocated)}${check.unpriced ? ` + ${check.unpriced} unpriced` : ""}`);
    if (check.difference != null) parts.push(check.difference.startsWith("-") ? `Over budget by ${fmt(check.difference.slice(1))}` : check.difference === "0.00" ? "Fully allocated" : `To allocate ${fmt(check.difference)}`);
  }
  parts.push(b.split ? `${b.split} split, ${b.split_confidence} confidence (draft)` : b.no_split ?? "");
  parts.push(b.origin === "document" ? `Read from a source document: “${b.stated.text}”` : "Typed");
  parts.push(b.stated.gst === "inc" ? "Converted from GST inclusive" : b.stated.gst === "ex" ? "Ex GST" : "GST not stated; taken as ex GST");
  return parts.filter(Boolean).join(" · ");
}

// BudgetCell edits the construction budget in place. It saves to the profile
// row, so the Profile and the cost plan always show the same figure.
function BudgetCell({ budget, editable, onSave }: { budget?: ConstructionBudget; editable: boolean; onSave: (amount: string | null) => Promise<boolean> }) {
  const [draft, setDraft] = useState<string | null>(null), [invalid, setInvalid] = useState(false);
  const saving = useRef(false);
  const current = budget?.known ? budget.stated.amount : null;
  async function commit() {
    if (draft == null || saving.current) return;
    const raw = draft.replace(/[,\s$]/g, "");
    if (raw !== "" && !/^\d+(\.\d{1,2})?$/.test(raw)) { setInvalid(true); return; }
    const amount = raw === "" ? null : raw;
    if (amount === current) { setDraft(null); return; }
    saving.current = true;
    const ok = await onSave(amount);
    saving.current = false;
    if (ok) { setDraft(null); setInvalid(false); }
  }
  if (draft != null) return <input className="cost-input" aria-label="Construction budget" aria-invalid={invalid || undefined} autoFocus inputMode="decimal" value={draft}
    onChange={e => { setDraft(e.target.value); setInvalid(false); }} onBlur={() => void commit()}
    onKeyDown={e => { if (e.key === "Enter") void commit(); if (e.key === "Escape") { setDraft(null); setInvalid(false); } }} />;
  return <button type="button" className="cost-cell" aria-label="Construction budget" disabled={!editable} onClick={() => setDraft(current ?? "")}>{current ? fmt(current) : "—"}</button>;
}

// Single amounts edit directly; ranges retain both bounds in the cell editor.
function MoneyCell({ item, metric, editable, onSave, onDetailed }: { onDetailed: (v: CostValue) => Promise<boolean>; metric: "budget" | "commitment"; item: CostItem; editable: boolean; onSave: (amount: string | null) => Promise<boolean> }) {
  const v = item.values[metric], range = v?.value_state === "known" && v.amount == null;
  const [draft, setDraft] = useState<string | null>(null), [invalid, setInvalid] = useState(false);
  // One editing session ends once; Enter and the blur that follows must not both save.
  const active = useRef(false), cancelled = useRef(false), saving = useRef(false);
  const label = `${metric === "budget" ? "Budget" : "Approved Contract"} for ${item.label}`;
  async function commit() {
    if (draft == null || !active.current || saving.current) return;
    if (cancelled.current) { cancelled.current = false; active.current = false; setDraft(null); setInvalid(false); return; }
    const raw = draft.replace(/[,\s$]/g, "");
    if (raw !== "" && !/^-?\d+(\.\d{1,2})?$/.test(raw)) { setInvalid(true); return; }
    const amount = raw === "" ? null : raw;
    if (amount === (v?.value_state === "known" ? v.amount : null)) { active.current = false; setDraft(null); return; }
    saving.current = true;
    const ok = await onSave(amount);
    saving.current = false;
    if (ok) { active.current = false; setDraft(null); setInvalid(false); }
  }
  if (range) return <DetailedValueCell item={item} metric={metric} editable={editable} onSave={onDetailed} />;
  if (draft != null) return <input className="cost-input" aria-label={label} aria-invalid={invalid || undefined} autoFocus inputMode="decimal" value={draft}
    onClick={e => e.stopPropagation()} onChange={e => { setDraft(e.target.value); setInvalid(false); }} onBlur={() => void commit()}
    onKeyDown={e => { if (e.key === "Enter") void commit(); if (e.key === "Escape") { cancelled.current = true; void commit(); } }} />;
  // A share of the construction budget reads as a suggestion until edited.
  const spread = metric === "budget" && v?.origin === "assumption" && !!v.provenance?.spread;
  return <button type="button" className={`cost-cell${spread ? " cost-spread" : ""}`} aria-label={label} disabled={!editable} title={spread ? v!.rationale : undefined}
    onClick={e => { e.stopPropagation(); if (editable) { active.current = true; setDraft(v?.value_state === "known" ? v.amount ?? "" : ""); } }}>{cell(v)}</button>;
}

export function editContent(i: CostItem, changes: Partial<CostContent> = {}): CostContent {
 const {parent_item_id, code, label, line_kind, category, work_item_id, package_id, package_stage_id, posting, quantity, unit, rate, rate_basis, excluded, origin, meaning, rationale} = i;
 return {parent_item_id, code, label, line_kind, category, work_item_id, package_id, package_stage_id, posting, quantity, unit, rate, rate_basis, excluded, origin: origin === "calculation" ? "assumption" : origin, meaning, rationale, system_id: work_item_id ? "" : i.system_id, ...changes};
}
function TextCell({label, value, selected, onSave}: {label: string; value: string; selected: boolean; onSave: (v: string) => Promise<boolean>}) {
 const [editing, setEditing] = useState(false), [error, setError] = useState("");
 return editing ? <InlineCellEditor value={value} label={label} error={error} onCancel={() => { setEditing(false); setError(""); }} onCommit={text => { void onSave(text).then(ok => { if (ok) { setEditing(false); setError(""); } else setError("Could not save item. Try again."); }); }} /> : <span tabIndex={0} title={value} onDoubleClick={() => setEditing(true)} onKeyDown={e => { if (e.key === "F2" || e.key === "Enter") { e.preventDefault(); setEditing(true); } }} onClick={e => { if (selected && !e.ctrlKey && !e.metaKey && !e.shiftKey) { e.stopPropagation(); setEditing(true); } }}>{value || "—"}</span>;
}
const categories = ["other", "contingency", "escalation", "authority_fees", "exclusions"];
function CategoryCell({item, packages, label, onSave}: {item: CostItem; plan: CostPlan; packages: PackageRecord[]; label: string; onSave: (c: CostContent) => Promise<boolean>}) {
 const [editing, setEditing] = useState(false), [saving, setSaving] = useState(false);
 const isPackage = item.line_kind !== "project_wide";
 const options = isPackage ? packages.filter(p => !p.retired_at && p.kind === (item.line_kind === "fee" ? "services" : "works")).map(p => ({id:p.id,label:p.title})) : categories.map(id => ({id,label:words(id)}));
 return editing ? <select autoFocus disabled={saving} aria-label={`Category for ${item.label}`} value={isPackage ? item.package_id : item.category} onClick={e => e.stopPropagation()} onBlur={() => {if(!saving)setEditing(false);}} onKeyDown={e=>{if(e.key==="Escape")setEditing(false);}} onChange={e => {
  const v=e.target.value;setSaving(true);
  void onSave(editContent(item, isPackage ? {package_id:v,package_stage_id:""} : {category:v})).then(ok=>{setSaving(false);if(ok)setEditing(false);});
 }}><option value="">{item.line_kind === "works" ? "Use automatic assignment" : "Choose category"}</option>{isPackage && item.package_id && !options.some(o=>o.id===item.package_id) && <option value={item.package_id} disabled>{label}</option>}{options.map(o=><option key={o.id} value={o.id}>{o.label}</option>)}</select> : <button type="button" className="cost-open" title={label} aria-label={`Category for ${item.label}`} onClick={e=>{e.stopPropagation();setEditing(true);}}>{label}</button>;
}
function QuantityCells({item, editable, onSave}: {item:CostItem;editable:boolean;onSave:(c:CostContent)=>Promise<boolean>}) {
 const [draft,setDraft]=useState({quantity:item.quantity ?? "",unit:item.unit ?? "",rate:item.rate ?? ""});const saving=useRef(false);const saved=useRef(JSON.stringify(draft));
 async function commit(){if(saving.current || JSON.stringify(draft)===saved.current || !draft.quantity || !draft.unit || !draft.rate)return;if(draft.quantity===item.quantity && draft.unit===item.unit && draft.rate===item.rate)return;saving.current=true;const ok=await onSave(editContent(item,{quantity:draft.quantity,unit:draft.unit,rate:draft.rate,rate_basis:"User supplied"}));if(ok)saved.current=JSON.stringify(draft);saving.current=false;}
 return <>{(["quantity","unit","rate"] as const).map(k=><td key={k} className="cost-num">{editable?<input className="cost-input" aria-label={`${k} for ${item.label}`} value={draft[k]} onClick={e=>e.stopPropagation()} onChange={e=>setDraft({...draft,[k]:e.target.value})} onBlur={()=>void commit()} onKeyDown={e=>{if(e.key==="Enter")void commit();if(e.key==="Escape")setDraft({quantity:item.quantity??"",unit:item.unit??"",rate:item.rate??""});}} />:item[k]??"—"}</td>)}</>;
}
function AddLineRow({group:g,plan,packages,columns,onSave}:{group:typeof GROUPS[number];plan:CostPlan;packages:PackageRecord[];columns:number;onSave:(c:CostContent)=>Promise<boolean>}){
 const kind=g.kind;
 const [draft,setDraft]=useState<CostContent|null>(null);const saving=useRef(false);
 const options=kind==="fee"?packages.filter(p=>p.kind==="services").map(p=>({id:p.id,label:p.title})):kind==="works"?packages.filter(p=>p.kind==="works"&&!p.retired_at).map(p=>({id:p.id,label:p.title})):categories.map(id=>({id,label:words(id)}));
 const group=g.title;
 async function commit(){if(!draft || saving.current || !draft.label.trim() || (kind==="fee"&&!draft.package_id) || (kind==="works"&&!draft.system_id))return;saving.current=true;const ok=await onSave(draft);saving.current=false;if(ok)setDraft(null);}
 return <tr className="cost-add">{!draft?<td colSpan={columns}><button type="button" className="cost-open" onClick={()=>setDraft({parent_item_id:"",code:"",label:"",line_kind:kind,category:g.category,work_item_id:"",system_id:"",package_id:"",package_stage_id:"",posting:true,quantity:null,unit:"",rate:null,rate_basis:"",excluded:false,origin:"user",meaning:"allowance",rationale:""})}>+ Add line to {group}</button></td>:<><td/><td data-col="category"><select autoFocus={kind==="fee"} aria-label={`New ${group} category`} value={kind!=="project_wide"?draft.package_id:draft.category} onChange={e=>setDraft({...draft,[kind!=="project_wide"?"package_id":"category"]:e.target.value})}><option value="">{kind==="works"?"Use automatic assignment":"Choose category"}</option>{options.map(o=><option key={o.id} value={o.id}>{o.label}</option>)}</select></td><td>{kind==="works" && <select aria-label="New Construction physical scope" value={draft.system_id} onChange={e=>setDraft({...draft,system_id:e.target.value})}><option value="">Choose physical scope</option>{plan.systems?.map(sys=><option key={sys.id} value={sys.id}>{sys.label}</option>)}</select>}<input autoFocus={kind!=="fee"} aria-label={`New ${group} item`} value={draft.label} maxLength={500} onChange={e=>setDraft({...draft,label:e.target.value})} onBlur={()=>void commit()} onKeyDown={e=>{if(e.key==="Enter")void commit();if(e.key==="Escape")setDraft(null);}}/></td><td colSpan={columns-3}><button type="button" className="btn btn-small" onClick={()=>void commit()}>Save line</button><button type="button" className="btn btn-small" onClick={()=>setDraft(null)}>Cancel</button></td></>}</tr>;
}

function RowActions({ item, editable, busy, onOpen, onSubdivide, onToggle }: { item: CostItem; editable: boolean; busy: boolean; onOpen: () => void; onSubdivide: () => void; onToggle: () => void }) {
  const id = useId(), menu = useRef<HTMLDivElement>(null), trigger = useRef<HTMLButtonElement>(null);
  const [position, setPosition] = useState({ top: 0, left: 0 });
  const [open, setOpen] = useState(false);
  useEffect(() => {
    if (!open) return;
    const close = () => menu.current?.hidePopover();
    window.addEventListener("resize", close);
    window.addEventListener("scroll", close, true);
    return () => { window.removeEventListener("resize", close); window.removeEventListener("scroll", close, true); };
  }, [open]);
  const act = (action: () => void) => { menu.current?.hidePopover(); action(); };
  return <><button ref={trigger} type="button" className="cost-more" aria-label={`Actions for ${item.label}`} title={`Actions for ${item.label}`} popoverTarget={id} onClick={() => {
    const rect = trigger.current!.getBoundingClientRect();
    setPosition({ top: Math.max(8, Math.min(rect.bottom + 4, window.innerHeight - 150)), left: Math.max(8, Math.min(rect.right - 210, window.innerWidth - 218)) });
  }}><svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true"><circle cx="4" cy="10" r="1.5" /><circle cx="10" cy="10" r="1.5" /><circle cx="16" cy="10" r="1.5" /></svg></button>
    <div ref={menu} id={id} popover="auto" onToggle={e => setOpen(e.newState === "open")} className="cost-row-menu" style={position} aria-label={`Actions for ${item.label}`}>
      <button type="button" onClick={() => act(onOpen)}>View line details</button>
      {editable && <><button type="button" disabled={busy || !item.posting || item.excluded} onClick={() => act(onSubdivide)}>Subdivide allowance</button><button type="button" disabled={busy} onClick={() => act(onToggle)}>{item.excluded ? "Include in totals" : "Exclude from totals"}</button></>}
    </div></>;
}

function DetailedValueCell({ item, metric, editable, onSave }: { item: CostItem; metric: string; editable: boolean; onSave: (value: CostValue) => Promise<boolean> }) {
  const current = item.values[metric];
  const [draft, setDraft] = useState<CostValue | null>(null), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const label = `${metric === "claimed_to_date" ? "Claimed to Date" : metric === "budget" ? "Budget" : "Approved Contract"} for ${item.label}`;
  if (!draft) return <button type="button" className="cost-cell" disabled={!editable} aria-label={label} onClick={e => { e.stopPropagation(); setDraft(current ? { ...current } : { amount: null, low: null, high: null, value_state: "unknown", as_of: null, origin: "user", meaning: "allowance", rationale: "" }); }}>{cell(current)}</button>;
  const range = current?.value_state === "known" && current.amount == null;
  return <form className="cost-value-editor" onClick={e => e.stopPropagation()} onKeyDown={e => { e.stopPropagation(); if (e.key === "Escape" && !busy) { setDraft(null); setError(""); } }} onSubmit={e => {
    e.preventDefault(); if (busy) return;
    const fields = range ? ["low", "high"] as const : ["amount"] as const;
    if (fields.some(k => draft[k] && !/^-?\d+(\.\d{1,2})?$/.test(draft[k]!)) || (range && !!draft.low !== !!draft.high)) { setError("Enter valid amounts."); return; }
    setBusy(true);
    void onSave({ ...draft, origin: "user", value_state: fields.some(k => draft[k] != null) ? "known" : "unknown" }).then(ok => { setBusy(false); if (ok) { setDraft(null); setError(""); } else setError("Could not save. Try again."); });
  }}><fieldset disabled={busy}>
    {(range ? ["low", "high"] as const : ["amount"] as const).map((k, index) => <input key={k} autoFocus={index === 0} aria-label={`${label} ${k}`} inputMode="decimal" value={draft[k] ?? ""} onChange={e => { setError(""); setDraft({ ...draft, [k]: e.target.value || null }); }} />)}
    {metric === "claimed_to_date" && <input required type="date" aria-label={`Claims date for ${item.label}`} value={draft.as_of ?? ""} onChange={e => setDraft({ ...draft, as_of: e.target.value })} />}
    <button type="submit" className="btn btn-small">Save</button><button type="button" className="btn btn-small" onClick={() => { setDraft(null); setError(""); }}>Cancel</button>
  </fieldset>{error && <span role="alert">{error}</span>}</form>;
}
