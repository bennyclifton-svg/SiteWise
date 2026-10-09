// "Finish this RFP": the choices a consultant RFP still needs, beside the
// draft it changes. Decisions that block issue come first; every other
// control sits in a collapsible group. Each control writes on change and
// shows where its value came from, as Profile rows do.
import { useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { RfpCheck, RfpChoice, RfpControl, RfpControls } from "./reportApi";

type Write = (key: string, value: unknown | null) => Promise<void>;

const sourceNames: Record<string, string> = {
  user: "You", profile: "Profile", package: "Package", derived: "Profile", cost_plan: "Cost plan",
  rule: "Rule", default: "Default", boundary: "Appointment", suggested: "Suggested", assumed: "Assumed",
  unknown: "Not known", conflict: "Sources disagree", none: "Not set",
};
const groupIntro: Record<string, string> = {
  project: "Facts the rules read. Set one here only for this RFP; the profile keeps its own value.",
  stages: "In the base fee, priced as an option, left out, or already done by others.",
  elements: "Elements priced by stage in the R1 grid.",
  allowances: "Quantities every firm prices. Blank client quantities block issue.",
  positions: "I included · O optional, priced on its own · X excluded, done by the party named.",
  issue: "Printed on the RFP and checked before issue.",
};

export function RfpEditor({ controls, busyKey, onWrite }: { controls: RfpControls; busyKey: string; onWrite: Write }) {
  const [openGroup, setOpenGroup] = useState<string>("");
  const [focusKey, setFocusKey] = useState("");
  const panel = useRef<HTMLElement>(null);
  const groupOf = (key: string) => key === "boundary:range" || key === "scale" ? "stages" : controls.groups.find(g => g.controls.some(c => c.key === key))?.id ?? "";
  // A decision opens its group and focuses its control, so the fix is one
  // move away from the item that asks for it.
  function jump(key?: string) {
    if (!key) return;
    setOpenGroup(groupOf(key));
    setFocusKey(key);
  }
  useEffect(() => {
    if (!focusKey || !panel.current) return;
    const target = panel.current.querySelector<HTMLElement>(`[data-control="${CSS.escape(focusKey)}"]`);
    if (target) {
      target.scrollIntoView({ block: "center", behavior: "smooth" });
      target.querySelector<HTMLElement>("input, select, button")?.focus({ preventScroll: true });
    }
    setFocusKey("");
  }, [focusKey, openGroup]);
  const advisory = controls.checks.filter(c => !c.blocking);
  const draftRules = controls.status !== "reviewed";
  return <aside className="rfp-panel" aria-label="Finish this RFP" ref={panel}>
    <header className="rfp-panel-head">
      <h3>Finish this RFP</h3>
      <p className="rfp-status" role="status">{controls.decisions.length ? `${controls.decisions.length} ${controls.decisions.length === 1 ? "item blocks" : "items block"} issue` : draftRules ? "Choices complete. Issue waits for owner review of the rules." : "Ready to issue."}</p>
      {draftRules && <p className="rfp-rules-note">The {controls.title.toLowerCase()} rules are a draft. You can prepare and edit this RFP; issue stays blocked until the owner marks the rules reviewed.</p>}
    </header>
    {controls.decisions.length > 0 && <section className="rfp-decisions" aria-label="Decisions before issue">
      <h4>Decide before issue</h4>
      <ol>{controls.decisions.map(d => <Decision key={d.id} check={d} controls={controls} busyKey={busyKey} onWrite={onWrite} onJump={jump} />)}</ol>
    </section>}
    <Group id="stages" title="Appointment and stages" open={openGroup} setOpen={setOpenGroup} count={controls.groups.find(g => g.id === "stages")?.controls.length}>
      <Boundary controls={controls} busy={busyKey === "boundary:range"} onWrite={onWrite} />
      <Scale controls={controls} busy={busyKey === "scale"} onWrite={onWrite} />
      <Rows controls={controls.groups.find(g => g.id === "stages")?.controls ?? []} busyKey={busyKey} onWrite={onWrite} />
    </Group>
    {controls.groups.filter(g => g.id !== "stages").map(g => <Group key={g.id} id={g.id} title={g.title} open={openGroup} setOpen={setOpenGroup} count={g.controls.length} extra={g.id === "issue" ? <WeightTotal controls={g.controls} /> : undefined}>
      <Rows controls={g.controls} busyKey={busyKey} onWrite={onWrite} />
    </Group>)}
    {advisory.length > 0 && <details className="rfp-group rfp-advisory">
      <summary><span>Other checks</span><span className="rfp-count">{advisory.length}</span></summary>
      <p className="rfp-group-intro">Issue is not blocked by these; review them before you send.</p>
      <ul>{advisory.map(c => <li key={c.id}><strong>{c.label}.</strong> {c.text} {c.control && <button type="button" className="cell-link" onClick={() => jump(c.control)}>Show</button>}</li>)}</ul>
    </details>}
  </aside>;
}

function Group({ id, title, open, setOpen, count, extra, children }: { id: string; title: string; open: string; setOpen: (id: string) => void; count?: number; extra?: ReactNode; children: ReactNode }) {
  const ref = useRef<HTMLDetailsElement>(null);
  useEffect(() => { if (open === id && ref.current) ref.current.open = true; }, [open, id]);
  return <details className="rfp-group" ref={ref} onToggle={e => { if ((e.target as HTMLDetailsElement).open) setOpen(id); }}>
    <summary><span>{title}</span>{extra}{count !== undefined && <span className="rfp-count">{count}</span>}</summary>
    {groupIntro[id] && <p className="rfp-group-intro">{groupIntro[id]}</p>}
    <div className="rfp-rows">{children}</div>
  </details>;
}

function Decision({ check, controls, busyKey, onWrite, onJump }: { check: RfpCheck; controls: RfpControls; busyKey: string; onWrite: Write; onJump: (key?: string) => void }) {
  const control = controls.groups.flatMap(g => g.controls).find(c => c.key === check.control);
  const inline = control && ["choice", "tristate", "count", "text", "date", "number"].includes(control.kind);
  return <li className="rfp-decision">
    <p className="rfp-decision-label">{check.label}</p>
    <p className="rfp-decision-text">{check.text}</p>
    {check.id === "boundary" && <BoundaryConfirm controls={controls} busy={busyKey === "boundary:range"} onWrite={onWrite} />}
    {check.id.startsWith("changed:") && check.control && <div className="rfp-decision-actions">
      <button type="button" className="btn btn-small" disabled={busyKey === check.control} onClick={() => { const c = control; if (c) void onWrite(c.key, c.value); }}>Keep my setting</button>
      <button type="button" className="btn btn-small" disabled={busyKey === check.control} onClick={() => void onWrite(check.control!, null)}>Use the new value</button>
    </div>}
    {check.id === "fields" && <div className="rfp-decision-fields">
      {controls.groups.flatMap(g => g.controls).filter(c => c.key.startsWith("field:") && (check.changes ?? []).includes(c.label)).map(c => <ControlRow key={c.key} control={c} busy={busyKey === c.key} onWrite={onWrite} compact />)}
    </div>}
    {inline && check.id !== "fields" && !check.id.startsWith("changed:") && <ControlRow control={control!} busy={busyKey === control!.key} onWrite={onWrite} compact />}
    {!inline && check.id !== "boundary" && check.id !== "fields" && !check.id.startsWith("changed:") && check.control && <button type="button" className="cell-link" onClick={() => onJump(check.control)}>Show the setting</button>}
  </li>;
}

function stageName(controls: RfpControls, key: string) { return controls.boundary.stages.find(s => s.value === key)?.label ?? key; }

function BoundaryConfirm({ controls, busy, onWrite }: { controls: RfpControls; busy: boolean; onWrite: Write }) {
  const b = controls.boundary;
  return <div className="rfp-decision-actions">
    <button type="button" className="btn btn-small btn-primary" disabled={busy} onClick={() => void onWrite("boundary:range", { start: b.start, end: b.end })}>Confirm: {stageName(controls, b.start)} to {stageName(controls, b.end)}</button>
  </div>;
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
  return <>{controls.map(c => <ControlRow key={c.key} control={c} busy={busyKey === c.key} onWrite={onWrite} />)}</>;
}

function ControlRow({ control: c, busy, onWrite, compact }: { control: RfpControl; busy: boolean; onWrite: Write; compact?: boolean }) {
  const id = `rfp-${c.key.replace(/[^a-z0-9]+/gi, "-")}${compact ? "-d" : ""}`;
  const labelID = `${id}-label`;
  return <div className="rfp-row" data-control={compact ? undefined : c.key} data-kind={c.kind} aria-busy={busy || undefined}>
    {c.kind === "tristate" || c.kind === "position" || c.kind === "parts"
      ? <span className="rfp-label" id={labelID}>{c.label}</span>
      : <label className="rfp-label" htmlFor={id}>{c.label}</label>}
    <div className="rfp-value">
      <Control control={c} id={id} labelID={labelID} busy={busy} onWrite={onWrite} />
      <Source source={c.set ? "user" : c.source} />
      {c.set && <button type="button" className="cell-link" disabled={busy} onClick={() => void onWrite(c.key, null)}>Reset</button>}
      {c.kind === "position" && c.value === "X" && c.party && <span className="rfp-why">Done by {c.party}.</span>}
      {c.why && c.kind !== "position" && <span className="rfp-why">{c.why}</span>}
      {c.why && c.kind === "position" && c.value !== "X" && <span className="rfp-why">{c.why}</span>}
      {c.kind === "count" && c.owner === "consultant" && <span className="rfp-why">The consultant proposes this one in R2.</span>}
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
