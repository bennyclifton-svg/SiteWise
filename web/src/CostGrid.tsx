import { useRef, useState } from "react";
import type { CostItem, CostPlan, CostTotal, CostTotals, CostValue } from "./costApi";
import type { PackageRecord } from "./packageApi";

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
  { kind: "fee", title: "Fees and charges", bucket: "fees" },
  { kind: "works", title: "Construction", bucket: "works" },
  { kind: "project_wide", title: "Project-wide", bucket: "project_wide" },
] as const;
// Placeholder columns keep the cost report's shape until variations and monthly
// claims are tracked; they never show a figure.
const COLUMNS = ["Budget", "Approved Contract", "Forecast Variations", "Approved Variations", "Forecast Final Cost", "Budget Variance", "Claimed to Date", "This Month", "Remaining"];
const PLACEHOLDER = new Set(["Forecast Variations", "Approved Variations", "This Month"]);
type SortKey = "code" | "category" | "item";

export function CostGrid({ plan, totals, packages, selected, editable, onOpen, onBudget }: {
  plan: CostPlan; totals: CostTotals | null; packages: PackageRecord[]; selected: string; editable: boolean;
  onOpen: (id: string) => void; onBudget: (item: CostItem, amount: string | null) => Promise<boolean>;
}) {
  const [sort, setSort] = useState<{ key: SortKey; dir: 1 | -1 } | null>(null);
  const category = (i: CostItem) => i.line_kind === "project_wide" ? words(i.category) : packages.find(p => p.id === i.package_id)?.title || "Unassigned";
  const sortValue = (i: CostItem) => sort?.key === "code" ? i.code : sort?.key === "category" ? category(i) : i.label;
  const ordered = (list: CostItem[]) => sort ? [...list].sort((a, b) => sort.dir * sortValue(a).localeCompare(sortValue(b), undefined, { numeric: true })) : list;
  const ids = new Set(plan.items.map(i => i.cost_item_id));
  // Children follow their group line, indented; sorting applies within a level.
  const tree = (list: CostItem[], depth: number): { item: CostItem; depth: number }[] =>
    ordered(list).flatMap(i => [{ item: i, depth }, ...tree(plan.items.filter(c => c.parent_item_id === i.cost_item_id), depth + 1)]);
  const header = (key: SortKey, label: string) => <th scope="col" className="cost-sticky" data-col={key} aria-sort={sort?.key === key ? (sort.dir === 1 ? "ascending" : "descending") : undefined}>
    <button type="button" onClick={() => setSort(sort?.key === key ? (sort.dir === 1 ? { key, dir: -1 } : null) : { key, dir: 1 })}>{label}</button></th>;
  return <div className="cost-grid-scroll"><table className="cost-grid">
    <thead><tr>{header("code", "Code")}{header("category", "Category")}{header("item", "Item")}
      {COLUMNS.map(c => PLACEHOLDER.has(c) ? <th key={c} scope="col" className="cost-num cost-placeholder" aria-disabled="true" title="Not tracked yet">{c}</th> : <th key={c} scope="col" className="cost-num">{c}</th>)}</tr></thead>
    {GROUPS.map(g => {
      const top = plan.items.filter(i => i.line_kind === g.kind && (!i.parent_item_id || !ids.has(i.parent_item_id)));
      if (top.length === 0) return null;
      const t = totals?.[g.bucket], d = totals?.buckets?.[g.bucket];
      return <tbody key={g.kind}>
        <tr className="cost-group"><th colSpan={3 + COLUMNS.length} scope="colgroup">{g.title}</th></tr>
        {tree(top, 0).map(({ item: i, depth }) => {
          const f = totals?.items?.[i.cost_item_id];
          return <tr key={i.cost_item_id} className={`cost-row${i.excluded ? " cost-excluded" : ""}${i.posting ? "" : " cost-parent"}`} aria-selected={selected === i.cost_item_id} onClick={() => onOpen(i.cost_item_id)}>
            <td className="cost-sticky" data-col="code">{i.code}</td>
            <td className="cost-sticky" data-col="category">{category(i)}</td>
            <td className="cost-sticky" data-col="item" style={{ paddingLeft: `${8 + depth * 16}px` }}><button type="button" className="cost-open" onClick={e => { e.stopPropagation(); onOpen(i.cost_item_id); }}>{i.label}</button>{i.excluded && <span className="cost-tag">excluded</span>}</td>
            {!i.posting ? <td colSpan={COLUMNS.length} className="cost-note">Group · amounts are on the lines below</td> : <>
              <td className="cost-num"><BudgetCell item={i} editable={editable && !i.excluded} onOpen={() => onOpen(i.cost_item_id)} onSave={a => onBudget(i, a)} /></td>
              <td className="cost-num">{cell(i.values.commitment)}</td>
              <td className="cost-num cost-placeholder">—</td><td className="cost-num cost-placeholder">—</td>
              <td className="cost-num">{cell(f?.forecast)}</td>
              <td className="cost-num">{fmt(f?.budget_variance)}</td>
              <td className="cost-num">{cell(i.values.claimed_to_date)}</td>
              <td className="cost-num cost-placeholder">—</td>
              <td className="cost-num">{fmt(f?.remaining)}</td></>}
          </tr>;
        })}
        <tr className="cost-subtotal"><th colSpan={3} scope="row" className="cost-sticky" data-col="subtotal">{g.title} subtotal</th>
          <td className="cost-num">{subtotal(t?.budget)}</td><td className="cost-num">{subtotal(t?.commitment)}</td>
          <td className="cost-num cost-placeholder">—</td><td className="cost-num cost-placeholder">—</td>
          <td className="cost-num">{subtotal(t?.forecast)}</td><td className="cost-num">{fmt(d?.budget_variance)}</td>
          <td className="cost-num">{subtotal(t?.claimed_to_date)}</td><td className="cost-num cost-placeholder">—</td>
          <td className="cost-num">{fmt(d?.remaining)}</td></tr>
      </tbody>;
    })}
  </table></div>;
}

// Typed budgets accept plain amounts only. A range or an as-of date is set in
// the drawer so a single figure never overwrites it.
function BudgetCell({ item, editable, onOpen, onSave }: { item: CostItem; editable: boolean; onOpen: () => void; onSave: (amount: string | null) => Promise<boolean> }) {
  const v = item.values.budget, range = v?.value_state === "known" && v.amount == null;
  const [draft, setDraft] = useState<string | null>(null), [invalid, setInvalid] = useState(false);
  // One editing session ends once; Enter and the blur that follows must not both save.
  const active = useRef(false), cancelled = useRef(false), saving = useRef(false);
  const label = `Budget for ${item.label}`;
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
  if (draft != null) return <input className="cost-input" aria-label={label} aria-invalid={invalid || undefined} autoFocus inputMode="decimal" value={draft}
    onClick={e => e.stopPropagation()} onChange={e => { setDraft(e.target.value); setInvalid(false); }} onBlur={() => void commit()}
    onKeyDown={e => { if (e.key === "Enter") void commit(); if (e.key === "Escape") { cancelled.current = true; void commit(); } }} />;
  return <button type="button" className="cost-cell" aria-label={label} disabled={!editable && !range} title={range ? "Range budgets are edited in the line drawer" : undefined}
    onClick={e => { e.stopPropagation(); if (range) onOpen(); else if (editable) { active.current = true; setDraft(v?.value_state === "known" ? v.amount ?? "" : ""); } }}>{cell(v)}</button>;
}
