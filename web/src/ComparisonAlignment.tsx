import { useState } from "react";
import type { ScheduleItem } from "./submissionApi";
export function ComparisonRowEditor({ item, busy, onSave, onCancel }: {
 item?: ScheduleItem; busy: boolean; onSave: (key: string, label: string, section: string, unit: string) => Promise<void>; onCancel: () => void;
}) {
 const [label, setLabel] = useState(item?.label ?? "");
 const [section, setSection] = useState(item?.kind === "rate" ? "rate" : item?.base === false ? "option" : "base");
 const [unit, setUnit] = useState(item?.unit || "hour");
 return <form className="prices-row-editor" aria-label={item ? "Edit comparison row" : "Add comparison row"} onSubmit={e => { e.preventDefault(); void onSave(item?.key ?? "", label, section, unit); }}>
  <label>Service description<input autoFocus required maxLength={500} value={label} onChange={e => setLabel(e.target.value)} disabled={busy} /></label>
  {!item && <label>Placement<select value={section} onChange={e => setSection(e.target.value)} disabled={busy}><option value="base">Base fee</option><option value="option">Option</option><option value="rate">Unit rate</option></select></label>}
  {section === "rate" && <label>Rate unit<select value={unit} onChange={e => setUnit(e.target.value)} disabled={busy}>{["hour", "meeting", "inspection", "approval", "report", "review", "revision"].map(u => <option key={u} value={u}>Per {u}</option>)}</select></label>}
  <button className="btn btn-small" disabled={busy || !label.trim()}>Save row</button><button type="button" className="table-action" disabled={busy} onClick={onCancel}>Cancel</button>
 </form>;
}
