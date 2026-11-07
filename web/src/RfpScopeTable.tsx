import { Fragment, useEffect, useState } from "react";
import type { ReportBlock, RfpControls } from "./reportApi";
import { ControlRow, type Write } from "./RfpEditor";

type Service = { key: string; control?: string; label: string; value: string; group?: string; stage?: string; detail?: string; assumption?: string; party?: string; source: string; inactive?: boolean; notes?: {label:string;text:string}[] };
type Stage = { key: string; name: string; summary?: string; state: string; source?: string; text: string; assumption?: string; design_responsibility?: string; fee_basis?: string };
type Props = { block: ReportBlock; controls: RfpControls; busy: boolean; onWrite: Write };

// The letters match the printed key above the issued table.
const positionChoices = [["I", "I", "Included"], ["O", "O", "Optional price"], ["X", "X", "Excluded"], ["ask", "?", "Unknown"]] as const;

export function scopeTable(block: ReportBlock): boolean {
  return !block.source_missing && !!block.table
    && ["hydraulic_services", "hydraulic_stages", "scope_services", "scope_stages", "stage_scope"].includes(String(block.basis?.layout));
}

export function RfpScopeTable({ block, controls, busy, onWrite }: Props) {
  const [pending, setPending] = useState<{ key: string; value: string } | null>(null);
  useEffect(() => { if (!busy) setPending(null); }, [busy]);
  if (block.basis?.layout === "stage_scope") return <StageScope block={block} controls={controls} busy={busy} onWrite={onWrite} />;
  if (["hydraulic_stages", "scope_stages"].includes(String(block.basis?.layout))) return <StageFees block={block} controls={controls} busy={busy} onWrite={onWrite} />;
  const services = block.basis?.positions as Service[];
  const positions = controls.groups.flatMap(g => g.controls);
  const controlKey = (s: Service) => s.control ?? `position:${s.key}`;
  const writeChoice = (s: Service, value: string) => onWrite(controlKey(s), !controlKey(s).startsWith("position:") ? ({ I: "include", O: "option", X: "exclude", ask: "ask" }[value]) : value);
  return <div className="hydraulic-scope">
    {block.edited && <p className="report-wording">{block.text}</p>}
    <p className="rfp-scope-note">One click sets each service: I included · O optional, priced on its own · X excluded · ? unknown, to resolve before issue.</p>
    <div className="report-table-scroll"><table className="report-table hydraulic-service-table">
      <thead><tr><th scope="col">Scope of services</th><th scope="col">Position</th></tr></thead>
      <tbody>{services.map((s, i) => <Fragment key={controlKey(s)}> 
        {s.group && s.group !== services[i - 1]?.group && <tr className="hydraulic-group"><th colSpan={2} scope="colgroup">{s.group}</th></tr>}
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
          <td className="rfp-scope-position"><div className="rfp-position-choice" role="radiogroup" aria-label={`${s.label}: position`}>
            {positionChoices.map(([value, short, label]) => <label key={value} title={label}>
              <input type="radio" name={`scope-${controlKey(s)}`} value={value} aria-label={`${s.label}: ${label}`} checked={(pending?.key === controlKey(s) ? pending.value : s.value) === value} disabled={busy} onChange={() => { setPending({ key: controlKey(s), value }); void writeChoice(s, value); }} />
              <span aria-hidden="true">{short}</span>
            </label>)}
          </div></td>
        </tr>
      </Fragment>)}<tr><td colSpan={2}><ScopeInputs controls={controls} target="scope_basis" available={[...services.map(controlKey), ...controls.boundary.stages.map(s => `stage:${s.value}`)]} busy={busy} onWrite={onWrite} /></td></tr></tbody>
    </table></div>
    <p className="rfp-scope-note">Tell us in R6 if you propose a departure from these selections.</p>
  </div>;
}

type Line = { key: string; text: string; value: string; source: string; custom?: boolean };
type ScopeStage = Stage & { lines?: Line[]; by_element?: boolean };
type Element = { key: string; code: string; name: string; state: string };

const upperFirst = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

/** One In and one Excluded tick box; ticking one clears the other. */
function InOut({ label, value, disabled, onChoose }: { label: string; value: "I" | "X"; disabled: boolean; onChoose: (v: "I" | "X") => void }) {
  return <>
    <td className="rfp-inout"><label className="rfp-tick"><input type="checkbox" aria-label={`${label}: included`} checked={value === "I"} disabled={disabled} onChange={() => { if (value !== "I") onChoose("I"); }} /></label></td>
    <td className="rfp-inout"><label className="rfp-tick"><input type="checkbox" aria-label={`${label}: excluded`} checked={value === "X"} disabled={disabled} onChange={() => { if (value !== "X") onChoose("X"); }} /></label></td>
  </>;
}

/** "8 items · 1 excluded": what a collapsed stage holds. */
function itemCount(values: string[]): string {
  if (!values.length) return "No items";
  const out = values.filter(v => v === "X").length;
  return `${values.length} ${values.length === 1 ? "item" : "items"}${out ? ` · ${out} excluded` : ""}`;
}

const newLineID = () => "u" + Date.now().toString(36);

/**
 * Scope by stage (owner direction, 3 November 2026): each stage collapsed to
 * its name and item count; opened, every scope line and service with an In
 * and an Excluded tick box, and the user's own lines to add, change or
 * remove. One fee per stage. Every line starts included.
 */
function StageScope({ block, controls, busy, onWrite }: Props) {
  const [open, setOpen] = useState<Set<string>>(new Set());
  const [showAll, setShowAll] = useState(false);
  const [draft, setDraft] = useState<{ stage: string; id: string; text: string } | null>(null);
  // The tick shows at once while it saves; the saved draft replaces it.
  const [pending, setPending] = useState<Record<string, "I" | "X">>({});
  useEffect(() => { if (!busy) setPending({}); }, [busy]);
  const tick = (key: string, value: "I" | "X") => pending[key] ?? value;
  const choose = (key: string, value: "I" | "X", write: () => Promise<void>) => { setPending(p => ({ ...p, [key]: value })); void write(); };
  const stages = (block.basis?.stages ?? []) as ScopeStage[];
  const services = (block.basis?.services ?? []) as Service[];
  const elements = (block.basis?.elements ?? []) as Element[];
  const table = block.table!;
  const stageControls = controls.groups.find(g => g.id === "stages")?.controls ?? [];
  const applies = (s: ScopeStage) => !(s.state === "exclude" && s.source === "rule");
  const shown = stages.filter(s => showAll || applies(s));
  const hidden = stages.length - stages.filter(applies).length;
  const held = (key: string) => services.filter(s => (s.stage ?? "") === key && !s.inactive);
  const toggle = (key: string) => setOpen(old => { const next = new Set(old); if (next.has(key)) next.delete(key); else next.add(key); return next; });
  const own = (s: ScopeStage) => (s.lines ?? []).filter(l => l.custom).map(l => ({ id: l.key, text: l.text, value: l.value }));
  const saveOwn = (s: ScopeStage, lines: { id: string; text: string; value: string }[]) => onWrite(`custom:${s.key}`, lines.length ? lines : null);
  const commit = (s: ScopeStage) => {
    if (!draft) return;
    const text = draft.text.replace(/\s+/g, " ").trim().replace(/[{}]/g, "");
    if (!text) { setDraft(null); return; }
    const lines = own(s);
    const at = lines.findIndex(l => l.id === draft.id);
    if (at >= 0) lines[at] = { ...lines[at], text }; else lines.push({ id: draft.id, text, value: "I" });
    void saveOwn(s, lines).then(() => setDraft(null));
  };
  const editor = (s: ScopeStage) => draft?.stage === s.key && <form className="rfp-own-line" onSubmit={e => { e.preventDefault(); commit(s); }}>
    <label className="sr-only" htmlFor={`own-${s.key}`}>New scope line for {s.name}</label>
    <input id={`own-${s.key}`} autoFocus maxLength={300} value={draft.text} placeholder="Describe the scope line" disabled={busy}
      onChange={e => setDraft({ ...draft, text: e.target.value })} onKeyDown={e => { if (e.key === "Escape") setDraft(null); }} />
    <button className="btn btn-small" disabled={busy || !draft.text.trim()}>Save line</button>
    <button type="button" className="btn btn-small" disabled={busy} onClick={() => setDraft(null)}>Cancel</button>
  </form>;
  // The stage name opens and closes the stage (owner direction, 7 November
  // 2026: no button box); a small caret shows which way.
  const head = (key: string, name: string, isOpen: boolean) => <th scope="row" className="rfp-stage-name">
    <button type="button" className="rfp-stage-toggle" aria-expanded={isOpen} aria-controls={`rfp-stage-${key || "all"}`} onClick={() => toggle(key)}>
      <span className="rfp-stage-caret" aria-hidden="true" /><span>{name}</span>
    </button>
  </th>;
  const serviceRows = (key: string, stageOut: boolean) => held(key).map(s => <tr key={s.key} className="rfp-stage-line" data-control={s.control ?? `position:${s.key}`}>
    <td />
    <td>{s.label}{s.detail && <p className="hydraulic-detail">{s.detail}</p>}
      {s.notes?.map((n, i) => <p className="hydraulic-detail" key={i}>{n.label && `${n.label}: `}{n.text}</p>)}
      <ScopeChecks controls={controls} controlKey={s.control ?? `position:${s.key}`} label={s.label} busy={busy} onWrite={onWrite} />
      <ScopeInputs controls={controls} target={s.control ?? `position:${s.key}`} busy={busy} onWrite={onWrite} /></td>
    <InOut label={s.label} value={stageOut ? "X" : tick(s.key, s.value === "X" ? "X" : "I")} disabled={busy || stageOut} onChoose={v => choose(s.key, v, () => onWrite(s.control ?? `position:${s.key}`, v))} />
    <td />
  </tr>);
  return <>
    {controls.decisions.filter(c => c.control === "boundary:range").map(c => <p key={c.id} data-control="boundary:range" className="hydraulic-unresolved">{c.label}. {c.text}</p>)}
    {block.edited && <p className="report-wording">{block.text}</p>}
    <div className="report-table-scroll"><table className="report-table rfp-stage-scope">
      <thead><tr><th scope="col">Stage</th><th scope="col">Scope</th><th scope="col" className="rfp-inout">In</th><th scope="col" className="rfp-inout">Excluded</th><th scope="col" className="rfp-fee">Fee $</th></tr></thead>
      {shown.map(s => {
        const control = stageControls.find(c => c.key === `stage:${s.key}`);
        const out = tick(`stage:${s.key}`, s.state === "exclude" ? "X" : "I") === "X";
        const isOpen = open.has(s.key);
        const lines = s.lines ?? [];
        const own_elements = s.by_element ? elements : [];
        const values = [...lines.map(l => out ? "X" : l.value), ...own_elements.map(e => out || e.state === "exclude" ? "X" : "I"), ...held(s.key).map(p => out ? "X" : p.value)];
        const note = [s.design_responsibility, s.assumption].filter(Boolean).join(" ");
        return <tbody key={s.key} id={`rfp-stage-${s.key}`} data-stage-state={s.state}>
          <tr className="rfp-stage-row" data-control={`stage:${s.key}`}>
            {head(s.key, s.name, isOpen)}
            <td className="rfp-stage-summary" onClick={e => { if (e.target === e.currentTarget || (e.target as HTMLElement).classList.contains("rfp-stage-count")) toggle(s.key); }}>
              <span className="rfp-stage-count">{out ? "Excluded" : itemCount(values)}</span>
              {isOpen && note && <p className="hydraulic-detail">{note}</p>}
              <ScopeChecks controls={controls} controlKey={`stage:${s.key}`} label={s.name} busy={busy} onWrite={onWrite} />
              {isOpen && <ScopeInputs controls={controls} target={`stage:${s.key}`} busy={busy} onWrite={onWrite} />}</td>
            <InOut label={s.name} value={out ? "X" : "I"} disabled={busy || !!control?.locked} onChoose={v => choose(`stage:${s.key}`, v, () => onWrite(`stage:${s.key}`, v === "I" ? "include" : "exclude"))} />
            <td className="rfp-fee hydraulic-fee-blank">{out ? "—" : ""}</td>
          </tr>
          {isOpen && <>
            {[...lines.filter(l => !l.custom), null, ...lines.filter(l => l.custom)].map(l => l === null ? <Fragment key="services">
              {own_elements.map(e => <tr key={`element-${e.key}`} className="rfp-stage-line" data-control={`element:${e.key}`}>
                <td /><td>{e.code} {e.name}</td>
                <InOut label={`${e.code} ${e.name}`} value={out ? "X" : tick(`element:${e.key}`, e.state === "exclude" ? "X" : "I")} disabled={busy || out}
                  onChoose={v => choose(`element:${e.key}`, v, () => onWrite(`element:${e.key}`, v === "I" ? "include" : "exclude"))} />
                <td />
              </tr>)}
              {serviceRows(s.key, out)}</Fragment> : <tr key={l.key} className="rfp-stage-line" data-line={l.key}>
              <td />
              <td>{draft?.stage === s.key && draft.id === l.key ? editor(s) : <>{upperFirst(l.text)}
                {l.custom && <span className="rfp-own-actions">
                  <button type="button" className="btn btn-small" disabled={busy || !!draft} aria-label={`Change wording: ${l.text}`} onClick={() => setDraft({ stage: s.key, id: l.key, text: l.text })}>✎ Change</button>
                  <button type="button" className="btn btn-small" disabled={busy || !!draft} aria-label={`Remove line: ${l.text}`} onClick={() => void saveOwn(s, own(s).filter(o => o.id !== l.key))}>Remove</button>
                </span>}</>}</td>
              <InOut label={upperFirst(l.text)} value={out ? "X" : tick(`${s.key}.${l.key}`, l.value === "X" ? "X" : "I")} disabled={busy || out}
                onChoose={v => choose(`${s.key}.${l.key}`, v, () => l.custom ? saveOwn(s, own(s).map(o => o.id === l.key ? { ...o, value: v } : o)) : onWrite(`line:${s.key}.${l.key}`, v))} />
              <td />
            </tr>)}
            <tr className="rfp-stage-add"><td /><td colSpan={4}>
              {draft?.stage === s.key && !lines.some(l => l.key === draft.id) ? editor(s)
                : <button type="button" className="btn btn-small rfp-add-line" disabled={busy || !!draft || out} onClick={() => setDraft({ stage: s.key, id: newLineID(), text: "" })}>+ Add a line</button>}
            </td></tr>
          </>}
        </tbody>;
      })}
      <tbody id="rfp-stage-all">
        <tr className="rfp-stage-row">{head("", "Across all stages", open.has(""))}
          <td><span className="rfp-stage-count">{itemCount(held("").map(p => p.value))}</span></td><td /><td /><td className="rfp-fee hydraulic-fee-blank">In stage fees</td></tr>
        {open.has("") && serviceRows("", false)}
      </tbody>
      <tbody>{table.rows.slice(Number(block.basis?.stage_rows ?? 0)).map((row, i) => <tr key={`total-${i}`} className="rfp-stage-total"><th scope="row">{row[0]}</th><td>{row[1]}</td><td /><td /><td className="rfp-fee" /></tr>)}</tbody>
    </table></div>
    {hidden > 0 && <button type="button" className="btn btn-small rfp-stage-more" aria-expanded={showAll} onClick={() => setShowAll(!showAll)}>{showAll ? "Hide stages that do not apply" : `Show ${hidden} ${hidden === 1 ? "stage" : "stages"} that do not apply to this project`}</button>}
    {/* Inputs whose row is not shown (a service the brief does not need) stay reachable here. */}
    <ScopeInputs controls={controls} target="scope_basis" available={[...services.filter(s => !s.inactive).map(s => s.control ?? `position:${s.key}`), ...stages.filter(applies).map(s => `stage:${s.key}`)]} busy={busy} onWrite={onWrite} />
  </>;
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
        {table.columns.slice(2).map(h => h.startsWith("Fee") ? "Fee" : h).map(c => <td key={c} className="hydraulic-fee-blank" aria-label={`${c}: ${s.state === "exclude" || s.state === "done" ? "not applicable" : "consultant to complete"}`}>{s.state === "exclude" || s.state === "done" ? "—" : c === "Fee" && pricingRows[s.key]?.length ? "By element below" : ""}</td>)}
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
