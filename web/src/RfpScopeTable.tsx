import { Fragment, useEffect, useState } from "react";
import type { ReportBlock, RfpControls } from "./reportApi";
import { ControlRow, type Write } from "./RfpEditor";

type Service = { key: string; control?: string; label: string; value: string; group?: string; detail?: string; assumption?: string; party?: string; source: string; inactive?: boolean; notes?: {label:string;text:string}[] };
type Stage = { key: string; name: string; state: string; text: string; assumption?: string; design_responsibility?: string; fee_basis?: string };
type Props = { block: ReportBlock; controls: RfpControls; busy: boolean; onWrite: Write };

export function scopeTable(block: ReportBlock): boolean {
  return !block.source_missing && !!block.table
    && ["hydraulic_services", "hydraulic_stages", "scope_services", "scope_stages"].includes(String(block.basis?.layout));
}

export function RfpScopeTable({ block, controls, busy, onWrite }: Props) {
  const [pending, setPending] = useState<{ key: string; value: string } | null>(null);
  useEffect(() => { if (!busy) setPending(null); }, [busy]);
  if (["hydraulic_stages", "scope_stages"].includes(String(block.basis?.layout))) return <StageFees block={block} controls={controls} busy={busy} onWrite={onWrite} />;
  const services = block.basis?.positions as Service[];
  const positions = controls.groups.flatMap(g => g.controls);
  const controlKey = (s: Service) => s.control ?? `position:${s.key}`;
  const writeChoice = (s: Service, value: string) => onWrite(controlKey(s), !controlKey(s).startsWith("position:") ? ({ I: "include", O: "option", X: "exclude", ask: "ask" }[value]) : value);
  return <div className="hydraulic-scope">
    {block.edited && <p className="report-wording">{block.text}</p>}
    <p className="rfp-scope-note">Choose one position for each service. Unknown means the scope decision is still open; resolve it before issue.</p>
    <div className="report-table-scroll"><table className="report-table hydraulic-service-table">
      <thead><tr><th scope="col">Scope of services</th><th scope="col">Included</th><th scope="col">Optional price</th><th scope="col">Excluded</th><th scope="col">Unknown</th></tr></thead>
      <tbody>{services.map((s, i) => <Fragment key={controlKey(s)}> 
        {s.group && s.group !== services[i - 1]?.group && <tr className="hydraulic-group"><th colSpan={5} scope="colgroup">{s.group}</th></tr>}
        <tr data-control={controlKey(s)}>
          <th scope="row"><span>{s.label}</span>{s.detail && <p className="hydraulic-detail">{s.detail}</p>}
            {s.notes?.map((n,i) => <p className="hydraulic-detail" key={i}>{n.label && `${n.label}: `}{n.text}</p>)}
            {s.inactive && <p className="hydraulic-detail">Not required by the current brief. Select a position to add it to this RFP.</p>}
            {s.assumption && <p className="hydraulic-detail">Pricing basis: {s.assumption}</p>}
            {s.value === "ask" && <p className="hydraulic-unresolved">Scope not decided. Choose a position before issue.</p>}
            {s.value === "X" && controlKey(s).startsWith("position:") && !s.inactive && <Responsible service={s} busy={busy} onWrite={onWrite} />}
            {positions.find(p => p.key === controlKey(s))?.set && <button type="button" className="cell-link" disabled={busy} onClick={() => void onWrite(controlKey(s), null)}>Reset selection</button>}
            <ScopeChecks controls={controls} controlKey={controlKey(s)} label={positions.find(p => p.key === controlKey(s))?.label ?? s.label} busy={busy} onWrite={onWrite} />
            {!controlKey(s).startsWith("stage:") && <ScopeInputs controls={controls} target={controlKey(s)} busy={busy} onWrite={onWrite} />}
          </th>
          {[['I', 'Included'], ['O', 'Optional price'], ['X', 'Excluded'], ['ask', 'Unknown']].map(([value, label]) => <td key={value} className="hydraulic-choice">
            <label><input type="radio" name={`scope-${controlKey(s)}`} aria-label={`${s.label}: ${label}`} checked={(pending?.key === controlKey(s) ? pending.value : s.value) === value} disabled={busy} onChange={() => { setPending({ key: controlKey(s), value }); void writeChoice(s, value); }} /><span className="hydraulic-mobile-label">{label}</span></label>
          </td>)}
        </tr>
      </Fragment>)}<tr><td colSpan={5}><ScopeInputs controls={controls} target="scope_basis" available={[...services.map(controlKey), ...controls.boundary.stages.map(s => `stage:${s.value}`)]} busy={busy} onWrite={onWrite} /></td></tr></tbody>
    </table></div>
    <p className="rfp-scope-note">Tell us in R6 if you propose a departure from these selections.</p>
  </div>;
}

function Responsible({ service, busy, onWrite }: { service: Service; busy: boolean; onWrite: Write }) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState(service.party ?? "");
  useEffect(() => { setValue(service.party ?? ""); }, [service.party]);
  return editing ? <form className="hydraulic-owner" onSubmit={e => { e.preventDefault(); if (value.trim()) void onWrite(`responsible:${service.key}`, value.trim()).then(() => setEditing(false)); }}>
    <label>Responsible party<input value={value} onChange={e => setValue(e.target.value)} disabled={busy} maxLength={500} required /></label>
    <button className="btn btn-small" disabled={busy || !value.trim()}>Save party</button>
    <button type="button" className="cell-link" disabled={busy} onClick={() => setEditing(false)}>Cancel</button>
  </form> : <p className="hydraulic-detail">By {service.party || "party not named"}. <button type="button" className="cell-link" disabled={busy} onClick={() => setEditing(true)}>Change party</button></p>;
}

function StageFees({ block, controls, busy, onWrite }: Props) {
  const [pending, setPending] = useState<{ key: string; included: boolean } | null>(null);
  useEffect(() => { if (!busy) setPending(null); }, [busy]);
  const stages = block.basis?.stages as Stage[];
  const table = block.table!;
  const pricingRows = (block.basis?.pricing_rows ?? {}) as Record<string,string[][]>;
  const stageControls = controls.groups.find(g => g.id === "stages")?.controls ?? [];
  return <>
    {controls.decisions.filter(c => c.control === "boundary:range").map(c => <p key={c.id} data-control="boundary:range" className="hydraulic-unresolved">{c.label}. {c.text}</p>)}
    <p className="report-wording">Checked stages form the base fee. Unchecked options are priced separately. {block.text}</p>
    <div className="report-table-scroll"><table className="report-table hydraulic-stage-table">
      <thead><tr>{table.columns.map(c => <th scope="col" key={c}>{c}</th>)}</tr></thead>
      <tbody>{stages.map(s => <Fragment key={s.key}><tr data-control={`stage:${s.key}`}>
        <th scope="row"><label className="hydraulic-stage-check"><input type="checkbox" checked={pending?.key === s.key ? pending.included : s.state === "include"} aria-label={`Include ${s.name} in base fee`} disabled={busy || stageControls.find(c => c.key === `stage:${s.key}`)?.locked} onChange={e => { setPending({ key: s.key, included: e.target.checked }); void onWrite(`stage:${s.key}`, e.target.checked ? "include" : "option"); }} />{s.name}</label>
          {s.state !== "include" && <select aria-label={`${s.name}: unchecked stage`} value={s.state} disabled={busy || stageControls.find(c => c.key === `stage:${s.key}`)?.locked} onChange={e => void onWrite(`stage:${s.key}`, e.target.value)}>
            <option value="ask">Unknown</option><option value="option">Optional price</option><option value="exclude">Not in this RFP</option><option value="done">Done by others</option>
          </select>}
          {stageControls.find(c => c.key === `stage:${s.key}`)?.set && <button type="button" className="cell-link" disabled={busy} onClick={() => void onWrite(`stage:${s.key}`, null)}>Reset stage</button>}
        </th>
        <td><ScopeChecks controls={controls} controlKey={`stage:${s.key}`} label={s.name} busy={busy} onWrite={onWrite} />{s.text}{s.design_responsibility && ` ${s.design_responsibility}`}{s.assumption && ` ${s.assumption}`} {s.fee_basis === "time_to_ceiling" && "Fee ceiling: hours × R3 rate."}{s.fee_basis === "per_unit" && "Fee: rate per unit × quantity."}<ScopeInputs controls={controls} target={`stage:${s.key}`} busy={busy} onWrite={onWrite} /></td>
        {['Fee', 'Weeks', 'Hours'].map(c => <td key={c} className="hydraulic-fee-blank" aria-label={`${c}: ${s.state === "exclude" || s.state === "done" ? "not applicable" : "consultant to complete"}`}>{s.state === "exclude" || s.state === "done" ? "—" : c === "Fee" && pricingRows[s.key]?.length ? "By element below" : ""}</td>)}
      </tr>{(pricingRows[s.key] ?? []).map((row,i) => <tr key={i} className="rfp-element-fee">{row.map((cell,j) => j === 0 ? <th scope="row" key={j}>{cell}</th> : <td key={j}>{cell}</td>)}</tr>)}</Fragment>)}
      {table.rows.slice(Number(block.basis?.stage_rows ?? stages.length)).map((row, i) => <tr key={`total-${i}`}>{row.map((cell, j) => j === 0 ? <th scope="row" key={j}>{cell}</th> : <td key={j}>{cell}</td>)}</tr>)}
      </tbody>
    </table></div>
  </>;
}


// Scope choices do not rewrite project facts. Conflicting evidence can still
// be corrected beside the affected service, without a second questionnaire.
function ScopeChecks({ controls, controlKey, label, busy, onWrite }: { controls: RfpControls; controlKey: string; label: string; busy: boolean; onWrite: Write }) {
  const checks = controls.decisions.filter(c => c.control === controlKey || c.changes?.includes(label)
    || (controlKey === "position:rainwater" && c.id === "legacy:flag.hyd_rainwater")
    || (controlKey === "position:alternative_sources" && c.id === "legacy:flag.hyd_alternative_sources"));
  const facts = controls.groups.flatMap(g => g.controls);
  return <>{checks.filter(c => c.label.includes("sources disagree") || c.id.startsWith("legacy:") || (c.control === controlKey && !c.id.startsWith("scope:"))).map(c => {
    const fact = facts.find(f => f.key === c.control);
    return <div className="hydraulic-unresolved" key={c.id} data-check={c.id}>
      <p>{c.label}. {c.id.startsWith("legacy:") ? c.text : "Resolve the conflicting project information before issue."}</p>
      {fact && !fact.locked && <div role="group" aria-label={`Resolve project information for ${label}`}>
        <span>Is this system or service required by the project? </span>
        {[['yes', 'Yes'], ['no', 'No']].map(([value, text]) => <button type="button" className="btn btn-small" key={value} disabled={busy} onClick={() => void onWrite(fact.key, value)}>{text}</button>)}
      </div>}
    </div>;
  })}</>;
}

function ScopeInputs({ controls, target, available, busy, onWrite }: { controls: RfpControls; target: string; available?: string[]; busy: boolean; onWrite: Write }) {
  const inputs = (controls.groups.find(g => g.id === "appointment")?.controls ?? []).filter(c => c.target === target || (target === "scope_basis" && !available?.includes(c.target ?? "scope_basis")));
  if (!inputs.length) return null;
  return <details className="rfp-scope-inputs"><summary>{target === "scope_basis" ? "Appointment and service-level inputs" : "Scope basis"}</summary><p className="hydraulic-detail">These inputs describe the project. The scope selection above sets what this consultant is asked to price.</p>{inputs.map(c => <ControlRow key={c.key} control={c} busy={busy} showSource={false} blocked={controls.decisions.some(d => d.control === c.key)} onWrite={onWrite} />)}</details>;
}
