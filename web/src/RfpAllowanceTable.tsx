import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import type { ReportBlock, RfpControls } from "./reportApi";
import { blockedKeys, type Write } from "./RfpEditor";

type Allowance = { key: string; label: string; detail?: string; value: string; owner: string; source: string };

/** A draft saved before the Qty column (8 Nov 2026) prints its old two-column table until refreshed. */
export function allowanceTable(block: ReportBlock): boolean {
  const rows = block.basis?.allowances as Allowance[] | undefined;
  return !block.source_missing && !!block.table && block.table.columns.length === 3 && !!rows?.every(a => a.detail !== undefined);
}

/**
 * Allowances as one table: item, what it covers, and a quantity the PM can
 * nudge up or down. Every row starts at its default so the RFP issues with
 * a number in each line; a cleared quantity is marked in place and blocks issue.
 */
export function RfpAllowanceTable({ block, controls, busy, onWrite }: { block: ReportBlock; controls: RfpControls; busy: boolean; onWrite: Write }) {
  const rows = block.basis?.allowances as Allowance[];
  const blocked = blockedKeys(controls);
  const byKey = new Map(controls.groups.flatMap(g => g.controls).map(c => [c.key, c]));
  return <div className="report-table-scroll"><table className="report-table rfp-allowance-table">
    <thead><tr><th scope="col" className="rfp-al-item">Item</th><th scope="col">Allowance</th><th scope="col" className="rfp-al-qty">Qty</th></tr></thead>
    <tbody>{rows.map(a => {
      const key = `allowance:${a.key}`;
      const control = byKey.get(key);
      const value = control ? String(control.value ?? "") : a.value;
      return <tr key={a.key} data-control={key} data-blocked={blocked.has(key) || undefined}>
        <th scope="row">{a.label}</th>
        <td>{a.detail}</td>
        <td className="rfp-al-qty">{control
          ? <Stepper label={a.label} value={value} placeholder={a.owner === "consultant" ? "R2" : ""} busy={busy} onCommit={v => void onWrite(key, v === "" ? null : v)} />
          : value || (a.owner === "consultant" ? "Proposed in R2" : "___")}</td>
      </tr>;
    })}</tbody>
  </table></div>;
}

const MAX = 999; // the server accepts one to three digits

// Arrow clicks change the number at once and save after a short pause, so
// five quick clicks are one write, not five.
function Stepper({ label, value, placeholder, busy, onCommit }: { label: string; value: string; placeholder: string; busy: boolean; onCommit: (v: string) => void }) {
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
  return <span className="rfp-stepper">
    <button type="button" className="rfp-step" aria-label={`Decrease ${label}`} disabled={busy || (Number(draft) || 0) <= 0} onClick={() => step(-1)}>▾</button>
    <input className="input rfp-step-value" inputMode="numeric" pattern="[0-9]*" maxLength={3} aria-label={`${label} quantity`} placeholder={placeholder} value={draft} disabled={busy}
      onChange={e => setDraft(e.target.value.replace(/[^0-9]/g, "").slice(0, 3))} onBlur={() => commit(draft)} onKeyDown={keys} />
    <button type="button" className="rfp-step" aria-label={`Increase ${label}`} disabled={busy || (Number(draft) || 0) >= MAX} onClick={() => step(1)}>▴</button>
  </span>;
}
