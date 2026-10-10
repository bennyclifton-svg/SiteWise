// A consultant RFP is one report edited in place. Page one is the project,
// always open, each value edited where it prints; the other sections open
// their own controls under their heading. A check that blocks issue sits
// beside what it concerns, and the RFP closes with its Issue section.
// Project facts save to Profile, the one home every RFP reads (owner
// decision D1); the rest belong to this RFP. Each control writes on change
// and shows where its value came from.
import { useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { ReportBlock, RfpChoice, RfpControl, RfpControls } from "./reportApi";

export type Write = (key: string, value: unknown | null) => Promise<void>;

const sourceNames: Record<string, string> = {
  user: "You", profile: "Profile", package: "Package", derived: "Profile", cost_plan: "Cost plan",
  rule: "Rule", default: "Default", boundary: "Appointment", suggested: "Suggested", assumed: "Assumed",
  unknown: "Not known", conflict: "Sources disagree", none: "Not set",
};
const groupIntro: Record<string, string> = {
  stages: "In the base fee, priced as an option, left out, or already done by others.",
  elements: "Elements priced by stage in the R1 grid.",
  allowances: "Quantities every firm prices. Blank client quantities block issue.",
  positions: "I included · O optional, priced on its own · X excluded, done by the party named.",
  terms: "Printed in section 6. Terms can differ for each appointment.",
  weights: "How proposals are scored. The weights must total 100%.",
};

/** The report section a control is edited under ("" when none holds it). */
export function controlSection(controls: RfpControls, key: string): string {
  if (key === "boundary:range") return "role";
  if (key === "scale") return "stages";
  return controls.groups.find(g => g.controls.some(c => c.key === key))?.section ?? "";
}

/** Sections opened by an Edit toggle. Page one is always open; Issue is the screen's own. */
export function editableSections(controls: RfpControls): Set<string> {
  const out = new Set(controls.groups.filter(g => g.controls.length > 0).map(g => g.section));
  out.add("stages");
  out.add("role");
  out.delete("project");
  out.delete("issue");
  return out;
}

/** Control keys a blocking check names, so their values show red where they print. */
export function blockedKeys(controls: RfpControls): Set<string> {
  const all = controls.groups.flatMap(g => g.controls);
  const out = new Set<string>();
  for (const d of controls.decisions) {
    if (d.control) out.add(d.control);
    for (const c of all) if (d.changes?.includes(c.label)) out.add(c.key);
  }
  return out;
}

/** Blocking checks listed under a section's heading; "" lists those no section holds. */
export function SectionMarkers({ controls, section, onJump }: { controls: RfpControls; section: string; onJump: (key?: string) => void }) {
  const here = controls.decisions.filter(d => (d.control ? controlSection(controls, d.control) : "") === section);
  if (!here.length) return null;
  return <ul className="rfp-markers" aria-label="Blocks issue">
    {here.map(d => <li key={d.id}>
      <strong>{d.label}.</strong> {d.text}
      {d.control && <button type="button" className="cell-link" onClick={() => onJump(d.control)}>Show</button>}
    </li>)}
  </ul>;
}

export function RfpSectionEditor({ controls, section, busyKey, onWrite }: { controls: RfpControls; section: string; busyKey: string; onWrite: Write }) {
  const groups = controls.groups.filter(g => g.section === section && g.controls.length > 0);
  return <div className="rfp-edit" role="group" aria-label={`Edit ${section}`}>
    {section === "role" && <div className="rfp-rows"><Boundary controls={controls} busy={busyKey === "boundary:range"} onWrite={onWrite} /></div>}
    {section === "stages" && <div className="rfp-rows"><Scale controls={controls} busy={busyKey === "scale"} onWrite={onWrite} /></div>}
    {groups.map(g => <div key={g.id} className="rfp-edit-group">
      {(groups.length > 1 || section === "stages" || section === "role") && <h5 className="rfp-edit-title">{g.title}{g.id === "weights" && <WeightTotal controls={g.controls} />}</h5>}
      {groups.length === 1 && g.id === "weights" && <p className="rfp-group-intro"><WeightTotal controls={g.controls} /></p>}
      {groupIntro[g.id] && <p className="rfp-group-intro">{groupIntro[g.id]}</p>}
      <Rows controls={g.controls} busyKey={busyKey} onWrite={onWrite} />
    </div>)}
  </div>;
}

type GridCellData = { key: string; label: string; text: string; control?: string; empty?: boolean };
type GridColumnData = { title: string; cells: GridCellData[] };

/**
 * Page one: the project, as it prints, laid out like the Profile card
 * (.pf-card columns). Each value opens its control in place; a change saves
 * to Profile and every RFP that reads it.
 */
export function ProjectPage({ blocks, controls, busyKey, onWrite, onJump, renderBlock }: { blocks: ReportBlock[]; controls: RfpControls; busyKey: string; onWrite: Write; onJump: (key?: string) => void; renderBlock: (b: ReportBlock) => ReactNode }) {
  const byKey = new Map(controls.groups.flatMap(g => g.controls).map(c => [c.key, c]));
  const project = controls.groups.find(g => g.id === "project")?.controls ?? [];
  const blocked = blockedKeys(controls);
  const block = (id: string) => blocks.find(b => b.id === id);
  const header = block("rfp:header");
  const intro = block("rfp:project:intro");
  const budget = block("rfp:budget");
  const columns = (block("rfp:project:grid")?.basis?.grid ?? []) as GridColumnData[];
  const conditions = project.filter(c => c.key.startsWith("fact:flag."));
  const information = byKey.get("field:information");
  const informationText = block("rfp:project:information")?.text.replace(/^Information provided: /, "") || String(information?.value ?? "") || "___";
  return <div className="rfp-page-one">
    {header && <button type="button" className="rfp-strip" title="Change the issue details in the Issue section" onClick={() => onJump("field:ref")}>{header.text}</button>}
    {intro && renderBlock(intro)}
    <div className="pf-card rfp-grid">
      {columns.map(col => <GridColumn key={col.title} column={col} byKey={byKey} busyKey={busyKey} blocked={blocked} onWrite={onWrite} />)}
    </div>
    {conditions.length > 0 && <div className="rfp-conditions" role="group" aria-label="Project conditions">
      <h4>Project conditions</h4>
      <div className="rfp-conditions-grid">{conditions.map(c => <ControlRow key={c.key} control={c} busy={busyKey === c.key} blocked={blocked.has(c.key)} onWrite={onWrite} />)}</div>
    </div>}
    {information && <div className="rfp-information"><GridValue label="Information provided" text={informationText} control={information} busy={busyKey === information.key} blocked={blocked.has(information.key)} onWrite={onWrite} /></div>}
    {budget && renderBlock(budget)}
  </div>;
}

/** One Profile column: what prints, then the unset rows behind "more", as on the Profile card. */
function GridColumn({ column, byKey, busyKey, blocked, onWrite }: { column: GridColumnData; byKey: Map<string, RfpControl>; busyKey: string; blocked: Set<string>; onWrite: Write }) {
  const [open, setOpen] = useState(false);
  const cell = (c: GridCellData) => <GridValue key={c.key} label={c.label} text={c.empty ? "—" : c.text} control={c.control ? byKey.get(c.control) : undefined} busy={!!c.control && busyKey === c.control} blocked={!!c.control && blocked.has(c.control)} onWrite={onWrite} />;
  const more = column.cells.filter(c => c.empty);
  return <div className="pf-col" role="group" aria-label={column.title}>
    <h4>{column.title}</h4>
    {column.cells.filter(c => !c.empty).map(cell)}
    {open && more.map(cell)}
    {more.length > 0 && <button type="button" className="cell-link pf-col-more" aria-expanded={open} onClick={() => setOpen(!open)}>{open ? "Fewer" : `+ ${more.length} more`}</button>}
  </div>;
}

/** A printed value that opens its control in place; closes once it saves or loses focus. */
function GridValue({ label, text, control, busy, blocked, onWrite }: { label: string; text: string; control?: RfpControl; busy: boolean; blocked: boolean; onWrite: Write }) {
  const [open, setOpen] = useState(false);
  const box = useRef<HTMLDivElement>(null);
  const editable = !!control && !control.locked;
  const id = `rfp-cell-${(control?.key ?? label).replace(/[^a-z0-9]+/gi, "-")}`;
  useEffect(() => { if (open) box.current?.querySelector<HTMLElement>("input, select")?.focus(); }, [open]);
  const write: Write = async (key, value) => { await onWrite(key, value); setOpen(false); };
  return <div className="pf-row rfp-cell-row" data-control={control?.key} data-blocked={blocked || undefined}>
    {open ? <label className="pf-label" htmlFor={id} id={`${id}-label`}>{label}</label> : <span className="pf-label">{label}</span>}
    <div className="pf-value" ref={box} onKeyDown={e => { if (e.key === "Escape") setOpen(false); }} onBlur={e => { if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setOpen(false); }}>
      {open && control
        ? <Control control={control} id={id} labelID={`${id}-label`} busy={busy} onWrite={write} />
        : editable
          ? <button type="button" className="rfp-cell" aria-label={`${label}: ${text}. Change`} disabled={busy} onClick={() => setOpen(true)}>{text}</button>
          : <span className="rfp-cell-text">{text}</span>}
      <Source source={control ? (control.set ? "user" : control.source) : "profile"} />
      {control?.locked && control.help && <span className="rfp-why">{control.help}</span>}
    </div>
  </div>;
}

/**
 * The RFP's closing section, on screen only: what still blocks issue, the
 * issue details, the advisory checks and the issue form itself.
 */
export function RfpIssue({ controls, busyKey, onWrite, onJump, children }: { controls: RfpControls; busyKey: string; onWrite: Write; onJump: (key?: string) => void; children: ReactNode }) {
  const n = controls.decisions.length;
  const draftRules = controls.status !== "reviewed";
  const issue = controls.groups.find(g => g.id === "issue")?.controls ?? [];
  const blocked = blockedKeys(controls);
  const advisory = controls.checks.filter(c => !c.blocking);
  return <section className="report-section rfp-issue" aria-labelledby="rs-issue">
    <div className="report-section-head">
      <h3 id="rs-issue">Issue</h3>
      <p className="rfp-status" role="status">{n ? `${n} ${n === 1 ? "item blocks" : "items block"} issue` : draftRules ? "Choices complete" : "Ready to issue"}
        {n > 0 && controls.decisions[0].control && <> · <button type="button" className="cell-link" onClick={() => onJump(controls.decisions[0].control)}>Show first</button></>}</p>
    </div>
    {draftRules && <p className="rfp-rules-note">The {controls.title.toLowerCase()} rules are a draft. You can prepare and edit this RFP; issue stays blocked until the owner marks the rules reviewed.</p>}
    <SectionMarkers controls={controls} section="issue" onJump={onJump} />
    <SectionMarkers controls={controls} section="" onJump={onJump} />
    <div className="rfp-issue-grid" role="group" aria-label="Issue details">
      {issue.map(c => <ControlRow key={c.key} control={c} busy={busyKey === c.key} blocked={blocked.has(c.key)} onWrite={onWrite} />)}
    </div>
    {advisory.length > 0 && <details className="rfp-advisory">
      <summary><strong>Other checks</strong> <span className="rfp-count">{advisory.length}</span></summary>
      <p className="rfp-group-intro">Issue is not blocked by these; review them before you send.</p>
      <ul>{advisory.map(c => <li key={c.id}><strong>{c.label}.</strong> {c.text} {c.control && <button type="button" className="cell-link" onClick={() => onJump(c.control)}>Show</button>}</li>)}</ul>
    </details>}
    {children}
  </section>;
}


function Boundary({ controls, busy, onWrite }: { controls: RfpControls; busy: boolean; onWrite: Write }) {
  const b = controls.boundary;
  const [start, setStart] = useState(b.start);
  const [end, setEnd] = useState(b.end);
  useEffect(() => { setStart(b.start); setEnd(b.end); }, [b.start, b.end]);
  const order = (key: string) => b.stages.findIndex(s => s.value === key);
  const changed = start !== b.start || end !== b.end;
  const valid = order(start) <= order(end);
  return <div className="rfp-row rfp-boundary" data-control="boundary:range">
    <span className="rfp-label">This appointment buys</span>
    <div className="rfp-value">
      <label className="sr-only" htmlFor="rfp-boundary-start">First stage</label>
      <select id="rfp-boundary-start" className="select rfp-control" value={start} disabled={busy} onChange={e => setStart(e.target.value)}>{b.stages.map(s => <option key={s.value} value={s.value}>{s.label}</option>)}</select>
      <span className="muted">to</span>
      <label className="sr-only" htmlFor="rfp-boundary-end">Last stage</label>
      <select id="rfp-boundary-end" className="select rfp-control" value={end} disabled={busy} onChange={e => setEnd(e.target.value)}>{b.stages.map(s => <option key={s.value} value={s.value}>{s.label}</option>)}</select>
      {(changed || !b.confirmed) && <button type="button" className="btn btn-small btn-primary" disabled={busy || !valid} onClick={() => void onWrite("boundary:range", { start, end })}>{b.confirmed ? "Save" : "Confirm"}</button>}
      <Source source={b.confirmed ? "user" : "suggested"} />
      {b.set && !changed && <button type="button" className="cell-link" disabled={busy} onClick={() => void onWrite("boundary:range", null)}>Reset</button>}
      {!valid && <span className="rfp-why rfp-invalid">The first stage must come before the last.</span>}
      <span className="rfp-why">Stages after the last one that apply are priced as options.</span>
    </div>
  </div>;
}

function Scale({ controls, busy, onWrite }: { controls: RfpControls; busy: boolean; onWrite: Write }) {
  const s = controls.scale;
  return <div className="rfp-row" data-control="scale">
    <span className="rfp-label" id="rfp-scale-label">Service level</span>
    <div className="rfp-value">
      <Segment name="scale" labelledBy="rfp-scale-label" value={s.value} disabled={busy} options={[{ value: "simple", label: "Simple" }, { value: "complex", label: "Complex" }]} onChange={v => void onWrite("scale", v)} />
      <Source source={s.set ? "user" : s.source} />
      {s.set && <button type="button" className="cell-link" disabled={busy} onClick={() => void onWrite("scale", null)}>Reset</button>}
      <span className="rfp-why">{s.why}. Sets renders, model detail and similar service levels; technical scope follows the project's facts.</span>
    </div>
  </div>;
}

function Rows({ controls, busyKey, onWrite }: { controls: RfpControl[]; busyKey: string; onWrite: Write }) {
  if (!controls.length) return <p className="rfp-group-intro">Nothing to set for this RFP.</p>;
  return <div className="rfp-rows">{controls.map(c => <ControlRow key={c.key} control={c} busy={busyKey === c.key} onWrite={onWrite} />)}</div>;
}

function ControlRow({ control: c, busy, blocked, onWrite }: { control: RfpControl; busy: boolean; blocked?: boolean; onWrite: Write }) {
  const id = `rfp-${c.key.replace(/[^a-z0-9]+/gi, "-")}`;
  const labelID = `${id}-label`;
  return <div className="rfp-row" data-control={c.key} data-kind={c.kind} data-blocked={blocked || undefined} aria-busy={busy || undefined}>
    {c.kind === "tristate" || c.kind === "position" || c.kind === "parts"
      ? <span className="rfp-label" id={labelID}>{c.label}</span>
      : <label className="rfp-label" htmlFor={id}>{c.label}</label>}
    <div className="rfp-value">
      <Control control={c} id={id} labelID={labelID} busy={busy || (!!c.locked && c.kind !== "stage")} onWrite={onWrite} />
      <Source source={c.set ? "user" : c.source} />
      {c.set && <button type="button" className="cell-link" disabled={busy} onClick={() => void onWrite(c.key, null)}>Reset</button>}
      {c.kind === "position" && c.value === "X" && c.party && <span className="rfp-why">Done by {c.party}.</span>}
      {c.why && c.kind !== "position" && <span className="rfp-why">{c.why}</span>}
      {c.why && c.kind === "position" && c.value !== "X" && <span className="rfp-why">{c.why}</span>}
      {c.kind === "count" && c.owner === "consultant" && <span className="rfp-why">The consultant proposes this one in R2.</span>}
      {c.help && <span className="rfp-why">{c.help}</span>}
    </div>
  </div>;
}

function Control({ control: c, id, labelID, busy, onWrite }: { control: RfpControl; id: string; labelID: string; busy: boolean; onWrite: Write }) {
  const value = c.value === null || c.value === undefined ? "" : String(c.value);
  switch (c.kind) {
    case "choice":
      return <select id={id} className="select rfp-control" value={value} disabled={busy} onChange={e => e.target.value ? void onWrite(c.key, e.target.value) : void onWrite(c.key, "unknown")}>
        <option value="">Not established</option>
        {(c.options ?? []).map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
      </select>;
    case "stage":
    case "element":
      return <select id={id} className="select rfp-control" value={value} disabled={busy || c.locked} onChange={e => void onWrite(c.key, e.target.value)}>
        {value === "ask" && <option value="ask" disabled>To confirm</option>}
        {(c.options ?? []).map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
      </select>;
    case "tristate":
      return <Segment name={id} labelledBy={labelID} value={value || (c.source === "conflict" ? "" : "unknown")} disabled={busy} options={c.options ?? []} onChange={v => void onWrite(c.key, v)} />;
    case "position":
      return <Segment name={id} labelledBy={labelID} value={value} disabled={busy} options={(c.options ?? []).map(o => ({ value: o.value, label: o.value, title: `${c.label}: ${o.label.toLowerCase()}` }))} onChange={v => void onWrite(c.key, v)} />;
    case "parts":
      return <Parts control={c} labelID={labelID} busy={busy} onWrite={onWrite} />;
    case "weight":
      return <Typed id={id} value={value} busy={busy} inputMode="numeric" width="narrow" pattern="[0-9]*" onCommit={v => void onWrite(c.key, Number(v))} />;
    case "count":
      return <Typed id={id} value={value} busy={busy} inputMode="numeric" width="narrow" pattern="[0-9]*" onCommit={v => v === "" ? (c.set ? void onWrite(c.key, null) : undefined) : void onWrite(c.key, v)} />;
    case "date":
      return <Typed id={id} value={value} busy={busy} type="date" onCommit={v => void onWrite(c.key, v === "" ? null : v)} />;
    case "number":
    case "money":
      return <Typed id={id} value={value} busy={busy} inputMode="decimal" onCommit={v => void onWrite(c.key, v === "" ? null : v)} />;
    default:
      return <Typed id={id} value={value} busy={busy} onCommit={v => void onWrite(c.key, v === "" ? null : v)} />;
  }
}

// Typed text commits on blur or Enter, so a half-typed value is never saved.
function Typed({ id, value, busy, type = "text", inputMode, pattern, width, onCommit }: { id: string; value: string; busy: boolean; type?: string; inputMode?: "numeric" | "decimal"; pattern?: string; width?: "narrow"; onCommit: (v: string) => void }) {
  const [draft, setDraft] = useState(value);
  useEffect(() => setDraft(value), [value]);
  const commit = (e?: FormEvent) => { e?.preventDefault(); if (draft.trim() !== value) onCommit(draft.trim()); };
  return <form className="rfp-inline" onSubmit={commit}>
    <input id={id} className={`input rfp-control${width === "narrow" ? " rfp-narrow" : ""}`} type={type} inputMode={inputMode} pattern={pattern} value={draft} disabled={busy} onChange={e => setDraft(e.target.value)} onBlur={() => commit()} />
  </form>;
}

// The choice shows at once while it saves; the saved view then confirms it,
// or a failed save puts the previous choice back.
function Segment({ name, labelledBy, value, disabled, options, onChange }: { name: string; labelledBy: string; value: string; disabled: boolean; options: (RfpChoice & { title?: string })[]; onChange: (v: string) => void }) {
  const [pending, setPending] = useState<string | null>(null);
  useEffect(() => setPending(null), [value]);
  useEffect(() => { if (!disabled) setPending(null); }, [disabled]);
  const shown = pending ?? value;
  return <div className="rfp-segment" role="radiogroup" aria-labelledby={labelledBy}>
    {options.map(o => <label key={o.value} className="rfp-segment-option" title={o.title}>
      <input type="radio" name={name} value={o.value} checked={shown === o.value} disabled={disabled} onChange={() => { setPending(o.value); onChange(o.value); }} aria-label={o.title} />
      <span>{o.label}</span>
    </label>)}
  </div>;
}

function Parts({ control: c, labelID, busy, onWrite }: { control: RfpControl; labelID: string; busy: boolean; onWrite: Write }) {
  const selected = Array.isArray(c.value) ? c.value as string[] : [];
  const toggle = (id: string, on: boolean) => {
    const next = on ? [...selected, id] : selected.filter(s => s !== id);
    if (next.length) void onWrite(c.key, next);
  };
  return <fieldset className="rfp-parts" aria-labelledby={labelID}>
    {(c.options ?? []).map(o => <label key={o.value}><input type="checkbox" checked={selected.includes(o.value)} disabled={busy || (selected.length === 1 && selected.includes(o.value))} onChange={e => toggle(o.value, e.target.checked)} /> {o.label}</label>)}
  </fieldset>;
}

function Source({ source }: { source: string }) {
  return <span className="rfp-source" data-source={source}>{sourceNames[source] ?? source}</span>;
}

function WeightTotal({ controls }: { controls: RfpControl[] }) {
  const total = controls.filter(c => c.kind === "weight").reduce((sum, c) => sum + Number(c.value ?? 0), 0);
  return <span className={total === 100 ? "rfp-count" : "rfp-count rfp-count-alert"}>Weights {total}%</span>;
}
