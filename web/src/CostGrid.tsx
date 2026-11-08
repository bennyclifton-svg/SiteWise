import { clickRow, emptySelection, prune, selectAll } from "./table/selection";
import { InlineCellEditor } from "./table/InlineCellEditor";
import { RowHandle, startRowDrag } from "./table/RowHandle";
import { IconBin, IconEye, IconEyeOff, IconOpen, IconSubdivide } from "./icons";
import { createContext, use, useEffect, useId, useRef, useState, type PointerEvent as ReactPointerEvent, type ReactNode } from "react";
import type { CostContent, CostItem, CostPlan, CostTotal, CostTotals, CostValue, PreliminaryEstimate } from "./costApi";
import type { PackageRecord } from "./packageApi";
import type { Ledger, LedgerLine } from "./invoiceApi";

// Money arrives as exact decimal strings. Group digits without parsing to a
// float so the grid shows what the ledger holds.
// Without cents the amount rounds half away from zero to whole dollars, still
// on the string (BigInt), so display never drifts from the stored figure.
export const fmt = (m: string | null | undefined, cents = true) => {
  if (m == null) return "—";
  const negative = m.startsWith("-");
  let [whole, part = "00"] = m.replace("-", "").split(".");
  part = part.padEnd(2, "0");
  if (!cents && part[0] >= "5") whole = (BigInt(whole || "0") + 1n).toString();
  const shown = `${whole.replace(/\B(?=(\d{3})+(?!\d))/g, ",")}${cents ? `.${part}` : ""}`;
  return `${negative && /[1-9]/.test(shown) ? "−" : ""}${shown}`;
};
// The Cost Plan's Decimals switch. Everything that formats money inside the
// grid reads it through money(), which other screens' fmt never touches.
export const CentsContext = createContext(true);
const money = (m: string | null | undefined) => fmt(m, use(CentsContext));
const cell = (v?: CostValue) => v?.value_state !== "known" ? "—" : v.amount != null ? money(v.amount) : `${money(v.low)}–${money(v.high)}`;
// A subtotal with no known line is unknown, not a zero. With some lines
// unknown it shows what is known; the count is left off (owner direction,
// 7 Nov 2026: it cluttered every column).
const subtotal = (t?: CostTotal) => !t || t.lines === 0 || t.unknown === t.lines ? "—" : t.amount != null ? money(t.amount) : t.unknown > 0 ? money(t.known_subtotal) : `${money(t.low)}–${money(t.high)}`;
const words = (s: string) => s.replaceAll("_", " ");

const GROUPS = [
  { kind: "project_wide", title: "Fees and charges", bucket: "authority", category: "authority_fees" },
  { kind: "fee", title: "Consultants", bucket: "consultants", category: "" },
  { kind: "works", title: "Construction", bucket: "construction", category: "" },
  { kind: "project_wide", title: "Contingency", bucket: "contingency", category: "contingency" },
] as const;
// Placeholder columns keep the cost report's shape until variations and monthly
// claims are tracked; they never show a figure. Claimed to Date and Remaining
// come from the invoice ledger (owner direction, 7 Nov 2026: one claimed
// figure, from uploaded invoices; remaining is approved contract less it).
const COLUMNS = ["Budget", "Approved Contract", "Forecast Variations", "Approved Variations", "Forecast Final Cost", "Claimed to Date", "This Month", "Remaining"];
const LEDGER_COLUMNS = new Set(["Claimed to Date", "Remaining"]);
const PLACEHOLDER = new Set(["Forecast Variations", "Approved Variations", "This Month"]);
type SortKey = "category" | "item";

// Claimed to Date is invoiced to date from the ledger, computed on the server:
// the same figure the Invoices tab sums, so the two views cannot disagree.
function ClaimedCell({ line, onInvoices }: { line?: LedgerLine; onInvoices?: (id: string) => void }) {
  if (!line) return <td className="cost-num cost-ledger">—</td>;
  const over = line.over_budget ? " cost-over" : "";
  return <td className={`cost-num cost-ledger${over}`}>{onInvoices && line.invoices > 0 ? <button type="button" className="cost-open" title={`Show the ${line.invoices} invoice${line.invoices === 1 ? "" : "s"} on ${line.label}`} onClick={e => { e.stopPropagation(); onInvoices(line.id || "unallocated"); }}>{money(line.invoiced)}</button> : money(line.invoiced)}{line.over_budget && <span className="cost-tag">over budget</span>}</td>;
}

export function CostGrid({ projectId, reveal, plan, totals, packages, editable, busy, onOpen, onSubdivide, onDetailedValue, onReorder, onValue, onDelete, onItem, quantities, ledger, onInvoices }: {
  projectId: string; reveal?: { id: string };
  plan: CostPlan; totals: CostTotals | null; packages: PackageRecord[]; editable: boolean;
  ledger?: Ledger | null; onInvoices?: (lineId: string) => void;
  quantities: boolean; busy: boolean;
  onSubdivide: (item: CostItem) => void;
  onDetailedValue: (item: CostItem, metric: string, value: CostValue) => Promise<boolean>;
  onReorder: (ids: string[]) => Promise<boolean>;
  onDelete: (item: CostItem) => Promise<boolean>;
  onItem: (content: CostContent, id?: string, calculate?: boolean, values?: Record<string, CostValue>) => Promise<boolean>;
  onOpen: (id: string) => void; onValue: (item: CostItem, metric: "budget" | "commitment", amount: string | null) => Promise<boolean>;
}) {
  const [sort, setSort] = useState<{ key: SortKey; dir: 1 | -1 } | null>(null);
  const [selection, setSelection] = useState(emptySelection);
  const [drop, setDrop] = useState<{ id: string; after: boolean } | null>(null);
  const dragging = useRef(false);
  const scroll = useRef<HTMLDivElement>(null);
  const pendingReveal = useRef("");
  const storageKey = `sitewise.costs.collapsed.${projectId}`;
  const [collapsed, setCollapsed] = useState<Set<string>>(() => {
    try { const value: unknown = JSON.parse(localStorage.getItem(storageKey) ?? "[]"); return new Set(Array.isArray(value) ? value.filter((id): id is string => typeof id === "string") : []); }
    catch { return new Set(); }
  });
  useEffect(() => { try { localStorage.setItem(storageKey, JSON.stringify([...collapsed])); } catch { /* Optional view preference. */ } }, [collapsed, storageKey]);
  const items = plan.items;
  useEffect(() => setSelection(s => prune(s, items.map(i => i.cost_item_id))), [items]);
  const category = (i: CostItem) => i.resolved_category || (i.line_kind === "project_wide" ? words(i.category) : packages.find(p => p.id === i.package_id)?.title || "Unassigned");
  const sortValue = (i: CostItem) => sort?.key === "category" ? category(i) : i.label;
  // A line typed into the ghost row has no code; it stays below the coded
  // lines, where it was entered.
  const ordered = (list: CostItem[]) => sort ? [...list].sort((a, b) => sort.dir * sortValue(a).localeCompare(sortValue(b), undefined, { numeric: true })) : [...list].sort((a,b) => (a.display_order ?? Number.MAX_SAFE_INTEGER) - (b.display_order ?? Number.MAX_SAFE_INTEGER) || (Number(["other", "exclusions"].includes(a.category)) - Number(["other", "exclusions"].includes(b.category))) || (Number(!a.code) - Number(!b.code)) || a.code.localeCompare(b.code, undefined, {numeric:true}) || a.label.localeCompare(b.label));
  const ids = new Set(items.map(i => i.cost_item_id));
  const children = new Map<string, CostItem[]>();
  for (const item of items) {
    const siblings = children.get(item.parent_item_id) ?? [];
    siblings.push(item); children.set(item.parent_item_id, siblings);
  }
  const groupIds = items.filter(i => !i.posting && children.has(i.cost_item_id)).map(i => i.cost_item_id);
  useEffect(() => {
    if (!reveal || !items.some(i => i.cost_item_id === reveal.id)) return;
    const byId = new Map(items.map(i => [i.cost_item_id, i]));
    const ancestors = new Set<string>();
    for (let id = byId.get(reveal.id)?.parent_item_id; id && !ancestors.has(id); id = byId.get(id)?.parent_item_id) ancestors.add(id);
    setCollapsed(current => new Set([...current].filter(id => !ancestors.has(id))));
    setSelection(clickRow(emptySelection, reveal.id, [reveal.id]));
    pendingReveal.current = reveal.id;
  }, [reveal]);
  useEffect(() => {
    if (!pendingReveal.current) return;
    const row = Array.from(scroll.current?.querySelectorAll<HTMLTableRowElement>("tr[data-id]") ?? []).find(r => r.dataset.id === pendingReveal.current);
    if (row) {
      row.scrollIntoView({ block: "nearest", inline: "nearest" });
      // Start at the activity's identity columns after following an invoice.
      if (scroll.current) scroll.current.scrollLeft = 0;
      row.focus({ preventScroll: true }); pendingReveal.current = "";
    }
  });
  function toggleGroup(id: string) {
    setCollapsed(current => { const next = new Set(current); if (next.has(id)) next.delete(id); else next.add(id); return next; });
    setSelection(emptySelection);
  }
  const ledgerLine = new Map((ledger?.summary.lines ?? []).map(l => [l.id, l]));
  const width = 4 + COLUMNS.length + (quantities ? 3 : 0);
  // Display positions are not identities: links continue to use cost_item_id.
  let rowNumber = 0;
  // Children follow their group line; sorting applies within a level. Item
  // text is not indented (owner direction, 7 Nov 2026): a group line shows its
  // chevron in a slot every row keeps, so all item text starts at one edge.
  const tree = (list: CostItem[], depth: number): { item: CostItem; depth: number }[] =>
    ordered(list).flatMap(i => [{ item: i, depth }, ...(collapsed.has(i.cost_item_id) ? [] : tree(children.get(i.cost_item_id) ?? [], depth + 1))]);
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
  const header = (key: SortKey, label: string, lead?: ReactNode) => <th scope="col" className="cost-sticky" data-col={key} aria-sort={sort?.key === key ? (sort.dir === 1 ? "ascending" : "descending") : undefined}>
    {lead}<button type="button" onClick={() => setSort(sort?.key === key ? (sort.dir === 1 ? { key, dir: -1 } : null) : { key, dir: 1 })}>{label}</button></th>;
  // One chevron in the Item head opens or closes every group line at once.
  const allOpen = !groupIds.some(id => collapsed.has(id));
  const masterToggle = groupIds.length > 0 && <button type="button" className="cost-disclosure-toggle cost-master-toggle" aria-label={allOpen ? "Collapse all" : "Expand all"} aria-expanded={allOpen} onClick={() => { setCollapsed(allOpen ? new Set(groupIds) : new Set()); setSelection(emptySelection); }}><Chevron open={allOpen} /></button>;
  return <><div ref={scroll} className="cost-grid-scroll"><table className="cost-grid" tabIndex={0} onKeyDown={e => {
    if ((e.target as HTMLElement).closest("input,select,textarea,button,[popover]")) return;
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "a") { e.preventDefault(); setSelection(selectAll(visible)); }
    if (e.key === "Escape") setSelection(emptySelection);
  }}>
    <thead><tr><th scope="col" className="cost-sticky" data-col="code">No.</th>{header("category", "Category")}{header("item", "Item", masterToggle)}
      {quantities && ["Qty", "Unit", "Rate"].map(c => <th key={c} scope="col">{c}</th>)}
      {COLUMNS.map(c => PLACEHOLDER.has(c) ? <th key={c} scope="col" className="cost-num cost-placeholder" aria-disabled="true" title="Not tracked yet">{c}</th> : LEDGER_COLUMNS.has(c) ? <th key={c} scope="col" className="cost-num cost-ledger" title={c === "Remaining" ? "Approved contract less invoices to date, ex GST" : "From the uploaded invoices, ex GST"}>{c}</th> : <th key={c} scope="col" className="cost-num">{c}</th>)}<th scope="col" className="cost-actions"><span className="sr-only">Actions</span></th></tr></thead>
    {GROUPS.map(g => {
      const top = items.filter(i => totals?.section_items[i.cost_item_id] === g.bucket && (!i.parent_item_id || !ids.has(i.parent_item_id)));
      const t = totals?.sections[g.bucket];
      return <tbody key={g.bucket}>
        <tr className="cost-group"><th colSpan={width} scope="colgroup"><span>{g.title}</span></th></tr>
        {tree(top, 0).map(({ item: i, depth }) => {
          const f = totals?.items?.[i.cost_item_id];
          const group = !i.posting && children.has(i.cost_item_id);
          const expanded = group && !collapsed.has(i.cost_item_id);
          const summary = totals?.rollups?.[i.cost_item_id];
          return <tr key={i.cost_item_id} tabIndex={-1} className={`cost-row${i.excluded ? " cost-excluded" : ""}${i.posting ? "" : " cost-parent"}`} onClickCapture={e => { if ((e.ctrlKey || e.metaKey || e.shiftKey) && !(e.target as HTMLElement).closest("input,select,textarea,[popover]")) { e.stopPropagation(); setSelection(s => clickRow(s, i.cost_item_id, visible, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })); } }} data-id={i.cost_item_id} data-drop={drop?.id === i.cost_item_id ? (drop.after ? "after" : "before") : undefined} aria-selected={selection.ids.includes(i.cost_item_id)} onClick={e => { if (!dragging.current) setSelection(s => clickRow(s, i.cost_item_id, visible, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })); }}>
            <td className="cost-sticky" data-col="code">{editable && <RowHandle disabled={busy || !!sort} onPointerDown={e => handleDown(e, i)} />}{String(++rowNumber).padStart(2, "0")}</td>
            <td className="cost-sticky" data-col="category">{editable ? <CategoryCell item={i} plan={plan} packages={packages} label={category(i)} onSave={c => onItem(c, i.cost_item_id)} /> : category(i)}</td>
            <td className="cost-sticky" data-col="item" data-depth={depth}>{group && <button type="button" className="cost-disclosure-toggle" aria-label={`${expanded ? "Collapse" : "Expand"} ${i.label}`} aria-expanded={expanded} onClick={e => { e.stopPropagation(); toggleGroup(i.cost_item_id); }}><Chevron open={expanded} /></button>}{editable ? <TextCell label={i.label} value={i.label} selected={selection.ids.length === 1 && selection.ids.includes(i.cost_item_id)} onSave={label => label.trim() ? onItem(editContent(i, {label}), i.cost_item_id) : Promise.resolve(false)} /> : i.label}{i.excluded && <span className="cost-tag">excluded</span>}</td>
            {quantities && <QuantityCells key={i.cost_item_id} item={i} editable={editable && i.posting} onSave={c => onItem(c, i.cost_item_id, true)} />}
            {!i.posting ? <>
              <td className="cost-num">{i.values.budget ? <MoneyCell onDetailed={v => onDetailedValue(i, "budget", v)} metric="budget" item={i} editable={editable} onSave={a => onValue(i, "budget", a)} /> : expanded ? null : subtotal(summary?.values.budget)}</td>
              <td className="cost-num">{expanded ? null : subtotal(summary?.values.commitment)}</td>
              <td className={`cost-num${expanded ? "" : " cost-placeholder"}`}>{expanded ? null : "—"}</td><td className={`cost-num${expanded ? "" : " cost-placeholder"}`}>{expanded ? null : "—"}</td>
              <td className="cost-num">{expanded ? null : subtotal(summary?.values.forecast)}</td>
              {expanded ? <td className="cost-ledger" /> : <ClaimedCell line={ledger?.groups?.[i.cost_item_id]} onInvoices={onInvoices} />}
              <td className={`cost-num${expanded ? "" : " cost-placeholder"}`}>{expanded ? null : "—"}</td>
              <td className="cost-num cost-ledger">{expanded ? null : money(ledger?.groups?.[i.cost_item_id]?.contract_remaining)}</td>
            </> : <>
              <td className="cost-num"><MoneyCell onDetailed={v => onDetailedValue(i, "budget", v)} metric="budget" item={i} editable={editable} onSave={a => onValue(i, "budget", a)} /></td>
              <td className="cost-num"><MoneyCell onDetailed={v => onDetailedValue(i, "commitment", v)} metric="commitment" item={i} editable={editable} onSave={a => onValue(i, "commitment", a)} /></td>
              <td className="cost-num cost-placeholder">—</td><td className="cost-num cost-placeholder">—</td>
              <td className="cost-num">{cell(f?.forecast)}</td>
              <ClaimedCell line={ledgerLine.get(i.cost_item_id)} onInvoices={onInvoices} />
              <td className="cost-num cost-placeholder">—</td>
              <td className="cost-num cost-ledger">{money(ledgerLine.get(i.cost_item_id)?.contract_remaining)}</td></>}
            <td className="cost-actions" onClick={e => e.stopPropagation()}><RowActions item={i} editable={editable} busy={busy} onOpen={() => onOpen(i.cost_item_id)} onSubdivide={() => onSubdivide(i)} onToggle={() => void onItem(editContent(i, { excluded: !i.excluded }), i.cost_item_id)} onDelete={() => void onDelete(i)} /></td>
          </tr>;
        })}
        {editable && <GhostRow key={g.bucket} group={g} plan={plan} packages={packages} quantities={!!quantities} onSave={(c, values) => onItem(c, undefined, false, values)} />}
        <tr className="cost-subtotal"><th colSpan={3} scope="row" className="cost-sticky" data-col="subtotal">{g.title} subtotal</th>{quantities && <td colSpan={3} />}
          <td className="cost-num">{subtotal(t?.budget)}</td><td className="cost-num">{subtotal(t?.commitment)}</td>
          <td className="cost-num cost-placeholder">—</td><td className="cost-num cost-placeholder">—</td>
          <td className="cost-num">{subtotal(t?.forecast)}</td>
          <td className="cost-num cost-ledger">{ledger ? money(ledger.summary.sections[g.bucket] ?? "0.00") : "—"}</td><td className="cost-num cost-placeholder">—</td>
          <td className="cost-num cost-ledger">{money(ledger?.summary.sections_contract_remaining?.[g.bucket])}</td><td /></tr>
      </tbody>;
    })}
    <tbody className="cost-totals">
      <tr className="cost-subtotal cost-total"><th colSpan={3} scope="row" className="cost-sticky" data-col="subtotal">Total</th>{quantities && <td colSpan={3} />}
        <td className="cost-num">{subtotal(totals?.overall.budget)}</td><td className="cost-num">{subtotal(totals?.overall.commitment)}</td>
        <td className="cost-num cost-placeholder">—</td><td className="cost-num cost-placeholder">—</td>
        <td className="cost-num">{subtotal(totals?.overall.forecast)}</td>
        <td className="cost-num cost-ledger">{ledger ? money(ledger.summary.invoiced) : "—"}</td><td className="cost-num cost-placeholder">—</td>
        <td className="cost-num cost-ledger">{money(ledger?.summary.contract_remaining)}</td><td /></tr>
      {plan.preliminary_estimate && <tr className="cost-row cost-budget-row">
        <th colSpan={3} scope="row" className="cost-sticky" data-col="subtotal">Builder's preliminary estimate</th>{quantities && <td colSpan={3} />}
        <td className="cost-num">{plan.preliminary_estimate.low === plan.preliminary_estimate.high ? money(plan.preliminary_estimate.low) : `${money(plan.preliminary_estimate.low)} – ${money(plan.preliminary_estimate.high)}`}</td>
        <td colSpan={COLUMNS.length - 1} className="cost-note cost-budget-note"><span>{estimateNote(plan.preliminary_estimate)}</span></td><td /></tr>}
    </tbody>
  </table></div></>;
}

// estimateNote says what the builder's figure is: advice read from a
// document, shown for comparison, never spread or awarded.
function estimateNote(e: PreliminaryEstimate) {
  return [`Read from a source document: “${e.text}”`, e.gst === "inc" ? "GST inclusive" : e.gst === "ex" ? "Ex GST" : "GST not stated",
    "Preliminary advice for comparison: not a budget or a tender price"].join(" · ");
}

// Excel-style movement between editable cells (those carrying data-metric;
// subtotal rows, range cells and calculated columns carry none).
// Up/Down keep the column; Tab, Shift+Tab and Left/Right at the edge of the
// text go along the row, wrapping to the next line as Excel does.
function editableCells(from: HTMLElement, col?: string): HTMLElement[] {
  const table = from.closest("table");
  if (!table) return [];
  return Array.from(table.querySelectorAll<HTMLElement>(col ? `[data-metric="${col}"]` : "[data-metric]")).filter(el => !(el as HTMLButtonElement).disabled);
}
function cellAfter(from: HTMLElement, step: 1 | -1, col?: string): HTMLElement | null {
  const cells = editableCells(from, col);
  const at = cells.indexOf(from);
  return at < 0 ? null : cells[at + step] ?? null;
}
/** The cell a key asks for: an element, null at the grid's edge, or undefined
 * when the key is not a move (or Tab has nowhere to go and should leave the grid). */
function cellFor(e: React.KeyboardEvent<HTMLInputElement>): HTMLElement | null | undefined {
  const el = e.currentTarget;
  if (e.key === "Tab") return cellAfter(el, e.shiftKey ? -1 : 1) ?? undefined;
  if (e.shiftKey || e.ctrlKey || e.metaKey || e.altKey) return undefined; // text selection keys stay with the field
  if (e.key === "ArrowDown" || e.key === "ArrowUp") return cellAfter(el, e.key === "ArrowDown" ? 1 : -1, el.dataset.metric);
  // A fully selected figure counts as being at both edges, so one press moves on.
  if (e.key === "ArrowRight" && el.selectionEnd === el.value.length) return cellAfter(el, 1);
  if (e.key === "ArrowLeft" && el.selectionStart === 0) return cellAfter(el, -1);
  return undefined;
}
function enterCell(el: HTMLElement) {
  if (el instanceof HTMLButtonElement) el.click(); else el.focus();
}

// Single amounts edit directly; ranges retain both bounds in the cell editor.
function MoneyCell({ item, metric, editable, onSave, onDetailed }: { onDetailed: (v: CostValue) => Promise<boolean>; metric: "budget" | "commitment"; item: CostItem; editable: boolean; onSave: (amount: string | null) => Promise<boolean> }) {
  const v = item.values[metric], range = v?.value_state === "known" && v.amount == null;
  const [draft, setDraft] = useState<string | null>(null), [invalid, setInvalid] = useState(false);
  // One editing session ends once; Enter and the blur that follows must not both save.
  const active = useRef(false), cancelled = useRef(false), saving = useRef(false);
  const label = `${metric === "budget" ? "Budget" : "Approved Contract"} for ${item.label}`;
  /** Resolves true once the cell has closed (saved, unchanged or cancelled). */
  async function commit(): Promise<boolean> {
    if (draft == null || !active.current || saving.current) return false;
    if (cancelled.current) { cancelled.current = false; active.current = false; setDraft(null); setInvalid(false); return true; }
    const raw = draft.replace(/[,\s$]/g, "");
    if (raw !== "" && !/^-?\d+(\.\d{1,2})?$/.test(raw)) { setInvalid(true); return false; }
    const amount = raw === "" ? null : raw;
    if (amount === (v?.value_state === "known" ? v.amount : null)) { active.current = false; setDraft(null); return true; }
    saving.current = true;
    const ok = await onSave(amount);
    saving.current = false;
    if (ok) { active.current = false; setDraft(null); setInvalid(false); }
    return ok;
  }
  if (range) return <DetailedValueCell item={item} metric={metric} editable={editable} onSave={onDetailed} />;
  if (draft != null) return <input className="cost-input" data-metric={metric} aria-label={label} aria-invalid={invalid || undefined} autoFocus inputMode="decimal" value={draft}
    onFocus={e => e.currentTarget.select()} onClick={e => e.stopPropagation()} onChange={e => { setDraft(e.target.value); setInvalid(false); }} onBlur={() => void commit()}
    onKeyDown={e => {
      if (e.key === "Enter") void commit();
      if (e.key === "Escape") { cancelled.current = true; void commit(); }
      const next = cellFor(e);
      if (next !== undefined) {
        e.preventDefault();
        // Save first; move only once this cell has closed, so a refused value stays put.
        void commit().then(closed => { if (closed && next?.isConnected) enterCell(next); });
      }
    }} />;
  // A share of the construction budget reads as a suggestion until edited.
  const spread = metric === "budget" && v?.origin === "assumption" && !!v.provenance?.spread;
  return <button type="button" className={`cost-cell${spread ? " cost-spread" : ""}`} data-metric={metric} aria-label={label} disabled={!editable} title={spread ? v!.rationale : undefined}
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
 const picker = useRef<HTMLSelectElement>(null);
 // The click that opened the cell opens the list too, so choosing is one step.
 useEffect(() => { if (editing) try { (picker.current as (HTMLSelectElement & { showPicker?: () => void }) | null)?.showPicker?.(); } catch { /* Needs a user gesture; focus still works. */ } }, [editing]);
 const isPackage = item.line_kind !== "project_wide";
 const options = isPackage ? packages.filter(p => !p.retired_at && p.kind === (item.line_kind === "fee" ? "services" : "works")).map(p => ({id:p.id,label:p.title})) : categories.map(id => ({id,label:words(id)}));
 return editing ? <select ref={picker} className="field-plain cost-category-select" autoFocus disabled={saving} aria-label={`Category for ${item.label}`} value={isPackage ? item.package_id : item.category} onClick={e => e.stopPropagation()} onBlur={() => {if(!saving)setEditing(false);}} onKeyDown={e=>{if(e.key==="Escape")setEditing(false);}} onChange={e => {
  const v=e.target.value;setSaving(true);
  void onSave(editContent(item, isPackage ? {package_id:v,package_stage_id:""} : {category:v})).then(ok=>{setSaving(false);if(ok)setEditing(false);});
 }}><option value="">{item.line_kind === "works" ? "Use automatic assignment" : "Choose category"}</option>{isPackage && item.package_id && !options.some(o=>o.id===item.package_id) && <option value={item.package_id} disabled>{label}</option>}{options.map(o=><option key={o.id} value={o.id}>{o.label}</option>)}</select> : <button type="button" className="cost-open" title={label} aria-label={`Category for ${item.label}`} onClick={e=>{e.stopPropagation();setEditing(true);}}>{label}</button>;
}
function QuantityCells({item, editable, onSave}: {item:CostItem;editable:boolean;onSave:(c:CostContent)=>Promise<boolean>}) {
 const [draft,setDraft]=useState({quantity:item.quantity ?? "",unit:item.unit ?? "",rate:item.rate ?? ""});const saving=useRef(false);const saved=useRef(JSON.stringify(draft));
 async function commit(){if(saving.current || JSON.stringify(draft)===saved.current || !draft.quantity || !draft.unit || !draft.rate)return;if(draft.quantity===item.quantity && draft.unit===item.unit && draft.rate===item.rate)return;saving.current=true;const ok=await onSave(editContent(item,{quantity:draft.quantity,unit:draft.unit,rate:draft.rate,rate_basis:"User supplied"}));if(ok)saved.current=JSON.stringify(draft);saving.current=false;}
 return <>{(["quantity","unit","rate"] as const).map(k=><td key={k} className="cost-num">{editable?<input className="cost-input" data-metric={k} aria-label={`${k} for ${item.label}`} value={draft[k]} onClick={e=>e.stopPropagation()} onChange={e=>setDraft({...draft,[k]:e.target.value})} onBlur={()=>void commit()} onKeyDown={e=>{if(e.key==="Enter")void commit();if(e.key==="Escape")setDraft({quantity:item.quantity??"",unit:item.unit??"",rate:item.rate??""});const next=cellFor(e);if(next){e.preventDefault();enterCell(next);}}} onFocus={e=>e.currentTarget.select()} />:item[k]??"—"}</td>)}</>;
}
// The ghost line at the foot of each section: a blank line showing only
// "Add item…" (its pickers appear once clicked into) that
// becomes a real line once it has an item (owner direction, 8 Nov 2026,
// replacing "+ Add line"). Item, budget and approved contract save together in
// one write; nothing is created until the line is complete.
type Ghost={category:string;package_id:string;system_id:string;label:string;budget:string;commitment:string};
const GHOST_PLACEHOLDER={fee:"consultant",works:"trade"} as const;
const parseMoney=(s:string)=>{const raw=s.replace(/[,\s$]/g,"");return raw===""?null:/^-?\d+(\.\d{1,2})?$/.test(raw)?raw:undefined;};
function GhostRow({group:g,plan,packages,quantities,onSave}:{group:typeof GROUPS[number];plan:CostPlan;packages:PackageRecord[];quantities:boolean;onSave:(c:CostContent,values:Record<string,CostValue>)=>Promise<boolean>}){
 const kind=g.kind;
 const blank:Ghost={category:g.category,package_id:"",system_id:"",label:"",budget:"",commitment:""};
 const [draft,setDraft]=useState<Ghost>(blank),[active,setActive]=useState(false),[tried,setTried]=useState(false);
 const saving=useRef(false),item=useRef<HTMLInputElement>(null);
 const options=kind==="fee"?packages.filter(p=>p.kind==="services").map(p=>({id:p.id,label:p.title})):kind==="works"?packages.filter(p=>p.kind==="works"&&!p.retired_at).map(p=>({id:p.id,label:p.title})):categories.map(id=>({id,label:words(id)}));
 const dirty=draft.label!==""||draft.budget!==""||draft.commitment!==""||draft.package_id!==""||draft.system_id!==""||draft.category!==g.category;
 // A consultant line names its package and a construction line its physical
 // scope; either stands in for a blank item, as the seeded lines do.
 const label=draft.label.trim()||(kind==="fee"?options.find(o=>o.id===draft.package_id)?.label:kind==="works"?plan.systems?.find(x=>x.id===draft.system_id)?.label:"")||"";
 const budget=parseMoney(draft.budget),commitment=parseMoney(draft.commitment);
 const missing={label:!label,package:kind==="fee"&&!draft.package_id,scope:kind==="works"&&!draft.system_id,budget:budget===undefined,commitment:commitment===undefined};
 const set=(k:keyof Ghost)=>(e:{target:{value:string}})=>setDraft(d=>({...d,[k]:e.target.value}));
 async function commit(refocus:boolean){
  if(!dirty||saving.current)return;
  setTried(true);
  if(Object.values(missing).some(Boolean))return;
  const values:Record<string,CostValue>={};
  for(const [m,a] of [["budget",budget],["commitment",commitment]] as const)if(a)values[m]={amount:a,low:null,high:null,value_state:"known",as_of:null,origin:"user",meaning:"allowance",rationale:""};
  saving.current=true;
  const ok=await onSave({parent_item_id:"",code:"",label,line_kind:kind,category:kind==="project_wide"?draft.category:"",work_item_id:"",system_id:kind==="works"?draft.system_id:"",package_id:kind==="project_wide"?"":draft.package_id,package_stage_id:"",posting:true,quantity:null,unit:"",rate:null,rate_basis:"",excluded:false,origin:"user",meaning:"allowance",rationale:""},values);
  saving.current=false;
  if(ok){setDraft(blank);setTried(false);if(refocus)item.current?.focus();}
 }
 const keys=(e:React.KeyboardEvent)=>{if(e.key==="Enter"){e.preventDefault();void commit(true);}if(e.key==="Escape"){setDraft(blank);setTried(false);(e.target as HTMLElement).blur();}};
 const bad=(k:keyof typeof missing)=>tried&&missing[k]||undefined;
 const pick=kind==="project_wide"?draft.category:draft.package_id;
 const ghostPick=kind==="project_wide"?!dirty:!draft.package_id;
 return <tr className={`cost-ghost${dirty?" cost-ghost-dirty":""}`} onKeyDown={keys} onClick={e=>e.stopPropagation()}
  onFocus={()=>setActive(true)} onBlur={e=>{if(!e.currentTarget.contains(e.relatedTarget as Node)){setActive(false);void commit(false);}}}>
  <td className="cost-sticky" data-col="code" />
  <td className="cost-sticky" data-col="category">{(active||dirty)&&<select className={ghostPick?"cost-ghost-hint":undefined} aria-label={`New ${g.title} category`} aria-invalid={bad("package")} value={pick} onChange={set(kind==="project_wide"?"category":"package_id")}>
   {kind!=="project_wide"&&<option value="">{GHOST_PLACEHOLDER[kind]}</option>}{options.map(o=><option key={o.id} value={o.id}>{o.label}</option>)}</select>}</td>
  <td className="cost-sticky" data-col="item"><span className="cost-ghost-item">{kind==="works"&&(active||dirty)&&<select className={draft.system_id?undefined:"cost-ghost-hint"} aria-label="New Construction physical scope" aria-invalid={bad("scope")} value={draft.system_id} onChange={set("system_id")}><option value="">Choose physical scope</option>{plan.systems?.map(x=><option key={x.id} value={x.id}>{x.label}</option>)}</select>}
   <input ref={item} aria-label={`New ${g.title} item`} aria-invalid={bad("label")} placeholder="Add item…" maxLength={500} value={draft.label} onChange={set("label")}/></span></td>
  {quantities&&<><td/><td/><td/></>}
  <td className="cost-num"><input inputMode="decimal" aria-label={`New ${g.title} budget`} aria-invalid={bad("budget")} placeholder="—" value={draft.budget} onChange={set("budget")}/></td>
  <td className="cost-num"><input inputMode="decimal" aria-label={`New ${g.title} approved contract`} aria-invalid={bad("commitment")} placeholder="—" value={draft.commitment} onChange={set("commitment")}/></td>
  <td className="cost-num cost-placeholder">—</td><td className="cost-num cost-placeholder">—</td><td className="cost-num">—</td>
  <td className="cost-num cost-ledger">—</td><td className="cost-num cost-placeholder">—</td><td className="cost-num cost-ledger">—</td><td className="cost-actions"/>
 </tr>;
}

function RowActions({ item, editable, busy, onOpen, onSubdivide, onToggle, onDelete }: { item: CostItem; editable: boolean; busy: boolean; onOpen: () => void; onSubdivide: () => void; onToggle: () => void; onDelete: () => void }) {
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
      <button type="button" onClick={() => act(onOpen)}><span className="more-menu-icon"><IconOpen /></span>View line details</button>
      {editable && <><button type="button" disabled={busy || !item.posting || item.excluded} onClick={() => act(onSubdivide)}><span className="more-menu-icon"><IconSubdivide /></span>Subdivide allowance</button><button type="button" disabled={busy} onClick={() => act(onToggle)}><span className="more-menu-icon">{item.excluded ? <IconEye /> : <IconEyeOff />}</span>{item.excluded ? "Include in totals" : "Exclude from totals"}</button>
        <button type="button" className="cost-menu-delete" disabled={busy} aria-label={`Delete ${item.label}`} onClick={() => act(onDelete)}><span className="more-menu-icon"><IconBin /></span>Delete line</button></>}
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

/** Disclosure chevron: down when open, right when closed. */
function Chevron({ open }: { open: boolean }) {
  return <svg width="20" height="20" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d={open ? "m4 6 4 4 4-4" : "m6 4 4 4-4 4"} /></svg>;
}
