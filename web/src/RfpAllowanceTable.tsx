import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import type { ReportBlock, RfpControls } from "./reportApi";
import { blockedKeys, type Write } from "./RfpEditor";
import "./rfp-tables.css";

type Allowance = { key: string; label: string; detail?: string; value: string; owner: string; source: string };

/** A draft saved before the two-pair table (8 Nov 2026) prints its saved table until refreshed. */
export function allowanceTable(block: ReportBlock): boolean {
  const rows = block.basis?.allowances as Allowance[] | undefined;
  return !block.source_missing && !!block.table && block.basis?.layout === "allowance_pairs" && !!rows?.every(a => a.detail !== undefined);
}

/**
 * Allowances in two pairs of columns, Item | Qty | Item | Qty, read down the
 * left pair then the right, as the printed RFP lays them out (owner
 * direction, 8 November 2026). Each item carries what it covers beneath its
 * name; each quantity starts at its default and is nudged or typed in place.
 * A cleared quantity is marked on its cell and blocks issue.
 */
export function RfpAllowanceTable({ block, controls, busy, onWrite }: { block: ReportBlock; controls: RfpControls; busy: boolean; onWrite: Write }) {
  const all = block.basis?.allowances as Allowance[];
  const blocked = blockedKeys(controls);
  const byKey = new Map(controls.groups.flatMap(g => g.controls).map(c => [c.key, c]));
  const half = Math.ceil(all.length / 2);
  const pair = (a: Allowance | undefined) => {
    if (!a) return <><td className="rfp-al-item" /><td className="rfp-al-qty" /></>;
    const key = `allowance:${a.key}`;
    const control = byKey.get(key);
    const value = control ? String(control.value ?? "") : a.value;
    return <>
      <td className="rfp-al-item" data-control={key} data-blocked={blocked.has(key) || undefined}>
        <span className="rfp-al-label">{a.label}</span>{a.detail && <span className="rfp-al-detail">{a.detail}</span>}
      </td>
      <td className="rfp-al-qty" data-blocked={blocked.has(key) || undefined}>{control
        ? <Stepper label={a.label} value={value} placeholder={a.owner === "consultant" ? "R2" : ""} busy={busy} onCommit={v => void onWrite(key, v === "" ? null : v)} />
        : value || (a.owner === "consultant" ? "Proposed in R2" : "___")}</td>
    </>;
  };
  return <div className="report-table-scroll"><table className="report-table rfp-allowance-pairs">
    <colgroup><col /><col className="rfp-al-qty-col" /><col /><col className="rfp-al-qty-col" /></colgroup>
    <thead><tr><th scope="col">Item</th><th scope="col" className="rfp-al-qty">Qty</th><th scope="col">Item</th><th scope="col" className="rfp-al-qty">Qty</th></tr></thead>
    <tbody>{Array.from({ length: half }, (_, i) => <tr key={i}>{pair(all[i])}{pair(all[i + half])}</tr>)}</tbody>
  </table></div>;
}

const MAX = 999; // the server accepts one to three digits

// Arrow clicks change the number at once and save after a short pause, so
// five quick clicks are one write, not five.
// Terms reuse it for day counts; an id ties it to a visible label there.
export function Stepper({ id, label, value, placeholder, busy, onCommit }: { id?: string; label: string; value: string; placeholder: string; busy: boolean; onCommit: (v: string) => void }) {
  const [draft, setDraft] = useState(value);
  const timer = useRef<number | undefined>(undefined);
  useEffect(() => setDraft(value), [value]);
  useEffect(() => () => window.clearTimeout(timer.current), []);
  const commit = (v: string) => { window.clearTimeout(timer.current); if (v !== value) onCommit(v); };
  const step = (by: number) => {
    const n = Math.min(MAX, Math.max(0, (Number(draft) || 0) + by));
    setDraft(String(n));
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => commit(String(n)), 500);
  };
  const keys = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "ArrowUp" || e.key === "ArrowDown") { e.preventDefault(); step(e.key === "ArrowUp" ? 1 : -1); }
    if (e.key === "Enter") { e.preventDefault(); commit(draft); }
  };
  return <span className="rfp-al-stepper">
    <button type="button" className="rfp-al-step" aria-label={`Decrease ${label}`} disabled={busy || (Number(draft) || 0) <= 0} onClick={() => step(-1)}><Chevron down /></button>
    <input id={id} className="rfp-al-value" inputMode="numeric" pattern="[0-9]*" maxLength={3} aria-label={id ? undefined : `${label} quantity`} placeholder={placeholder} value={draft} disabled={busy}
      onChange={e => setDraft(e.target.value.replace(/[^0-9]/g, "").slice(0, 3))} onBlur={() => commit(draft)} onKeyDown={keys} onFocus={e => e.currentTarget.select()} />
    <button type="button" className="rfp-al-step" aria-label={`Increase ${label}`} disabled={busy || (Number(draft) || 0) >= MAX} onClick={() => step(1)}><Chevron /></button>
  </span>;
}

function Chevron({ down = false }: { down?: boolean }) {
  return <svg viewBox="0 0 10 10" width="10" height="10" aria-hidden="true" focusable="false">
    <path d={down ? "M2 3.5 5 6.5 8 3.5" : "M2 6.5 5 3.5 8 6.5"} fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
  </svg>;
}
