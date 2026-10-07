import { useCallback, useEffect, useState } from "react";
import { ApiError } from "./api";
import { deliveryApi, type DeliveryContent, type DeliveryRecord, type DeliveryDependency } from "./deliveryApi";
import { packageApi, type PackageRecord, type ScopeWork } from "./packageApi";
import "./reports.css";

const statuses: Record<string, string[]> = {
  milestone: ["planned", "achieved", "cancelled"], approval: ["not_submitted", "submitted", "approved", "rejected", "withdrawn"],
  risk: ["open", "mitigating", "closed"], activity: ["not_started", "in_progress", "blocked", "complete", "cancelled"],
  action: ["not_started", "in_progress", "blocked", "complete", "cancelled"], issue: ["open", "in_progress", "resolved", "closed"], decision: ["open", "decided", "superseded"],
};
const words = (s: string) => s.replaceAll("_", " ");
const dateFields = ["baseline_date", "target_date", "forecast_date", "actual_date", "as_of"] as const;
const dateLabels = ["Baseline date", "Target date", "Forecast date", "Actual date", "As of date"];
const blank = (): DeliveryContent => ({ kind: "milestone", title: "", status: "planned", owner_text: "", baseline_date: null, target_date: null, forecast_date: null, actual_date: null, as_of: null, package_id: "", work_item_id: "", stage_id: "", details: {} });
interface Editor { base: DeliveryRecord | null; input: DeliveryContent }
function recordInput(record: DeliveryRecord): DeliveryContent {
  return { kind: record.kind, title: record.title, status: record.status, owner_text: record.owner_text, baseline_date: record.baseline_date, target_date: record.target_date, forecast_date: record.forecast_date, actual_date: record.actual_date, as_of: record.as_of, package_id: record.package_id ?? "", work_item_id: record.work_item_id ?? "", stage_id: record.stage_id ?? "", details: { ...record.details } };
}
function changes(editor: Editor): Partial<DeliveryContent> {
  if (!editor.base) return editor.input;
  const previous = recordInput(editor.base);
  return Object.fromEntries(Object.entries(editor.input).filter(([key, value]) => JSON.stringify(value) !== JSON.stringify(previous[key as keyof DeliveryContent])));
}

export function Delivery({ projectId, tick, onSignedOut }: { projectId: string; tick: number; onSignedOut: () => void }) {
  const [dependencies, setDependencies] = useState<DeliveryDependency[]>([]);
  const [predecessor, setPredecessor] = useState("");
  const [lag, setLag] = useState("0");
  const [items, setItems] = useState<DeliveryRecord[]>([]);
  const [packages, setPackages] = useState<PackageRecord[]>([]);
  const [works, setWorks] = useState<ScopeWork[]>([]);
  const [selected, setSelected] = useState("");
  const [editor, setEditor] = useState<Editor | null>(null);
  const [busy, setBusy] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState("");
  const [loadError, setLoadError] = useState("");
  const [reload, setReload] = useState(0);
  const fail = useCallback((e: unknown) => {
    if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
    setError(e instanceof ApiError && e.status === 409 ? "This record changed. Your edits are kept; reload and compare the saved record before saving." : e instanceof ApiError && e.status === 422 ? "Check the dates, status and assignments. An approval determination cannot precede its submission." : e instanceof ApiError && e.status === 404 ? "This record or one of its assignments is no longer available. Reload saved delivery records and review the selection." : "Could not load or save delivery records. Your entries are kept; try again.");
  }, [onSignedOut]);
  useEffect(() => {
    let stopped = false;
    Promise.all([deliveryApi.list(projectId), packageApi.list(projectId), packageApi.works(projectId), deliveryApi.dependencies(projectId)]).then(([d, p, w, dependencies]) => {
      if (!stopped) { setItems(d.items); setDependencies(dependencies.items); setPackages(p.items); setWorks(w.items); setLoaded(true); setLoadError(""); }
    }).catch(e => { if (!stopped) {
      if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
      setLoadError("Could not load delivery records and their assignments. Reload to try again. Your unsaved entries are kept.");
    } });
    return () => { stopped = true; };
  }, [projectId, tick, reload, onSignedOut]);
  const item = items.find(row => row.id === selected);
  const stale = !!editor?.base && item?.version !== editor.base.version;
  const input = editor?.input;
  const pkg = packages.find(p => p.id === input?.package_id);
  function update(patch: Partial<DeliveryContent>) { if (editor) setEditor({ ...editor, input: { ...editor.input, ...patch } }); }
  function detail(key: string, value: unknown) { if (input) update({ details: { ...input.details, [key]: value } }); }
  function edit(record: DeliveryRecord) {
    setError("");
    setEditor({ base: record, input: recordInput(record) });
  }
  function rebase(record: DeliveryRecord) {
    if (!editor?.base || record.kind !== editor.input.kind) return;
    const changedDetails = Object.fromEntries(Object.entries(editor.input.details).filter(([key, value]) => JSON.stringify(value) !== JSON.stringify(editor.base?.details[key])));
    setEditor({ ...editor, base: record, input: { ...recordInput(record), ...changes(editor), details: { ...record.details, ...changedDetails } } });
  }
  async function save() {
    if (!editor || stale) return;
    setBusy(true); setError("");
    try {
      const saved = editor.base ? await deliveryApi.patch(projectId, editor.base.id, editor.base.version, changes(editor)) : await deliveryApi.create(projectId, editor.input);
      setItems(old => [...old.filter(row => row.id !== saved.id), saved]); setSelected(saved.id); setEditor(null); setReload(n => n + 1);
    } catch (e) { fail(e); if (e instanceof ApiError && e.status === 409) setReload(n => n + 1); }
    finally { setBusy(false); }
  }
  async function changeDependency(remove: boolean, dependency?: DeliveryDependency) {
    if (!item) return; setBusy(true);setError("");
    try {await deliveryApi.dependency(projectId,dependency??{predecessor_id:predecessor,successor_id:item.id,type:"finish_to_start",lag_days:Number(lag)},item.version,remove);setPredecessor("");setReload(n=>n+1);}
    catch(e){if(e instanceof ApiError&&e.status===422)setError(items.reduce((message,row)=>message.replaceAll(row.id,row.title),e.message));else fail(e);} finally{setBusy(false);}
  }
  const packageName = (id: string) => packages.find(p => p.id === id)?.title ?? (id ? "Unavailable package" : "Not assigned");
  const workName = (id: string) => works.find(w => w.id === id)?.title ?? (id ? "Unavailable work item" : "Not assigned");
  const kindName = (record: DeliveryRecord) => record.provenance.proposal?.kind === "hold_point" ? "Hold point" : words(record.kind);
  return <section className="reports" aria-label="Delivery" aria-busy={busy}>
    <div className="report-library"><h1>Delivery</h1><p className="report-muted">Dates, risks and approvals recorded by the project team.</p>
      <div className="report-actions"><button type="button" className="btn" disabled={busy || !!editor || !loaded || !!loadError} onClick={() => { setSelected(""); setEditor({ base: null, input: blank() }); setError(""); }}>Add delivery record</button></div>
      {!loaded ? <p role="status">{loadError ? "Delivery records unavailable." : "Loading delivery records…"}</p> : !items.length ? <p className="report-empty">No delivery records yet. Add a milestone, risk or approval for the project or a package.</p> : <ul className="report-list">{items.map(row => <li key={row.id}><button type="button" disabled={busy || !!editor} aria-current={selected === row.id ? "true" : undefined} onClick={() => { setSelected(row.id); setError(""); }}>{row.title}<br />{kindName(row)} · {words(row.status)}<br />{row.target_date ? `Target ${row.target_date}` : "Target date not recorded"}</button></li>)}</ul>}
    </div>
    <div className="report-reader">
      {(error || loadError) && <div className="banner" role="alert">{loadError && <p>{loadError}</p>}{error && <p>{error}</p>}<button type="button" className="btn btn-small" disabled={busy} onClick={() => setReload(n => n + 1)}>Reload saved delivery records</button></div>}
      {loaded && !item && !editor && <p className="report-empty">Choose a delivery record to review its dates and status.</p>}
      {item && <><header className="report-toolbar"><div><h2>{item.title}</h2><p>{kindName(item)} · {words(item.status)}</p><p className="report-muted">{words(item.review_status)} · {words(item.origin)} input</p></div>{!editor && <button type="button" className="btn" disabled={busy} onClick={() => edit(item)}>Edit delivery record</button>}</header>
        <section className="report-section" aria-label="Saved delivery record"><h3>Saved delivery record</h3>
          <p>Owner: {item.owner_text || item.owner_user_id || "Not recorded"}</p><p>Package: {packageName(item.package_id)}</p><p>Work: {workName(item.work_item_id)}</p>
          {item.stage_id && <p>Stage: {packages.find(p => p.id === item.package_id)?.stages.find(s => s.id === item.stage_id)?.label ?? "Unavailable stage"}</p>}
          {dateFields.map((key, i) => <p key={key}>{dateLabels[i]}: {item[key] || "Not recorded"}</p>)}
          <DeliveryDetails item={item} />
          <h3>Finish-to-start dependencies</h3><p className="report-muted">Explicit sequence and calendar-day lag; dates are not rescheduled automatically.</p>
          {dependencies.filter(d=>d.successor_id===item.id).map(d=><p key={d.predecessor_id}>{items.find(i=>i.id===d.predecessor_id)?.title??"Unavailable predecessor"} · {d.lag_days} day lag <button className="btn btn-small" disabled={busy||!!editor} onClick={()=>void changeDependency(true,d)}>Remove dependency</button></p>)}
          <form className="package-scope-form" onSubmit={e=>{e.preventDefault();void changeDependency(false);}}><fieldset disabled={busy||!!editor}><label htmlFor="dependency-predecessor">Predecessor</label><select id="dependency-predecessor" required value={predecessor} onChange={e=>setPredecessor(e.target.value)}><option value="">Choose a delivery record</option>{items.filter(i=>i.id!==item.id).map(i=><option key={i.id} value={i.id}>{i.title}</option>)}</select><label htmlFor="dependency-lag">Lag in calendar days</label><input id="dependency-lag" type="number" min={-36500} max={36500} required value={lag} onChange={e=>setLag(e.target.value)}/><button className="btn" disabled={!predecessor}>Save dependency</button></fieldset></form>
          <details><summary>Source and latest correction</summary><p>Recorded by: {item.provenance.actor || "Not recorded"}</p>{item.provenance.at && <p>Recorded: {new Date(item.provenance.at).toLocaleString()}</p>}{item.provenance.last_edited_by && <p>Latest editor: {item.provenance.last_edited_by}</p>}{item.provenance.last_edited_at && <p>Last edited: {new Date(item.provenance.last_edited_at).toLocaleString()}</p>}
            {item.provenance.proposal && <p className="report-wording">Accepted proposal: {item.provenance.proposal.label}{item.provenance.proposal.draft ? " (draft knowledge)" : ""}.</p>}
            {item.provenance.sources?.length ? item.provenance.sources.map((s, i) => <p className="report-wording" key={i}>{s.filename || "Document source"}{s.page ? ` · page ${s.page}` : ""}{s.excerpt ? ` — ${s.excerpt}` : ""}</p>) : <p>No direct document excerpts recorded.</p>}
          </details>
        </section></>}
      {editor && input && <section className="report-section" aria-label="Delivery editor"><h3>{editor.base ? "Edit delivery record" : "Add delivery record"}</h3><p className="report-muted">Enter known dates and decisions. Leave unknown dates blank. Saving records the team's position; it does not verify an approval or release a hold point.</p>
        {stale && <div className="report-notice"><strong>Saved delivery record changed</strong><p>Your unsaved entries are below. Compare them with the saved record above.</p>{item && item.kind === input.kind ? <button type="button" className="btn btn-small" disabled={busy} onClick={() => rebase(item)}>Keep my edits against this version</button> : <p>{item ? "The record type changed. Cancel these edits and reopen the saved record to edit its new fields." : "This record is no longer available. Cancel to leave the editor."}</p>}</div>}
        <form className="package-scope-form" onSubmit={e => { e.preventDefault(); void save(); }}><fieldset disabled={busy}>
          {!editor.base && <><label htmlFor="delivery-kind">Record type</label><select id="delivery-kind" value={input.kind} onChange={e => update({ kind: e.target.value, status: statuses[e.target.value][0], details: {} })}>{Object.keys(statuses).map(k=><option key={k} value={k}>{words(k)}</option>)}</select></>}
          <label htmlFor="delivery-title">Record title</label><input id="delivery-title" required maxLength={200} value={input.title} onChange={e => update({ title: e.target.value })} />
          <label htmlFor="delivery-status">Status</label><select id="delivery-status" value={input.status} onChange={e => update({ status: e.target.value })}>{(statuses[input.kind] ?? []).map(s => <option key={s} value={s}>{words(s)}</option>)}</select>
          <label htmlFor="delivery-owner">Owner name or role</label><input id="delivery-owner" maxLength={200} value={input.owner_text} onChange={e => update({ owner_text: e.target.value })} />
          <label htmlFor="delivery-package">Package</label><select id="delivery-package" value={input.package_id} onChange={e => update({ package_id: e.target.value, stage_id: "" })}><option value="">Not assigned</option>{input.package_id && !pkg && <option value={input.package_id}>Unavailable package — select another</option>}{packages.map(p => <option key={p.id} value={p.id}>{p.title}</option>)}</select>
          <label htmlFor="delivery-stage">Package stage</label><select id="delivery-stage" disabled={!input.package_id} value={input.stage_id} onChange={e => update({ stage_id: e.target.value })}><option value="">Not assigned</option>{input.stage_id && !pkg?.stages.some(s => s.id === input.stage_id) && <option value={input.stage_id}>Unavailable stage — select another</option>}{pkg?.stages.map(s => <option key={s.id} value={s.id}>{s.label}{s.novation_phase === "none" ? "" : s.novation_phase === "pre" ? " · before novation" : " · after novation"}</option>)}</select>
          <label htmlFor="delivery-work">Work item</label><select id="delivery-work" value={input.work_item_id} onChange={e => update({ work_item_id: e.target.value })}><option value="">Not assigned</option>{input.work_item_id && !works.some(w => w.id === input.work_item_id) && <option value={input.work_item_id}>Unavailable work item — select another</option>}{works.map(w => <option key={w.id} value={w.id}>{w.title} · {w.action}</option>)}</select>
          <details className="delivery-dates" open><summary>Dates</summary><p className="report-muted">Clear a date to remove it. Approval submission and determination dates are recorded separately below.</p>{dateFields.map((key, i) => <div key={key}><label htmlFor={`delivery-${key}`}>{dateLabels[i]}</label><input id={`delivery-${key}`} type="date" value={input[key] ?? ""} onChange={e => update({ [key]: e.target.value || null })} /></div>)}</details>
          {input.kind === "decision" && <><label htmlFor="delivery-options">Decision options (one per line)</label><textarea id="delivery-options" value={Array.isArray(input.details.options)?input.details.options.join("\n"):""} onChange={e=>detail("options",e.target.value.split("\n").filter(Boolean))}/><label htmlFor="delivery-chosen">Chosen option</label><select id="delivery-chosen" value={String(input.details.chosen??"")} onChange={e=>detail("chosen",e.target.value)}><option value="">Not decided</option>{(Array.isArray(input.details.options)?input.details.options:[]).map((x,i)=><option key={i}>{String(x)}</option>)}</select></>}
          {input.kind === "risk" && <><label htmlFor="delivery-likelihood">Likelihood</label><input id="delivery-likelihood" maxLength={200} value={String(input.details.likelihood ?? "")} onChange={e => detail("likelihood", e.target.value)} /><label htmlFor="delivery-consequence">Consequence</label><textarea id="delivery-consequence" rows={3} maxLength={2000} value={String(input.details.consequence ?? "")} onChange={e => detail("consequence", e.target.value)} /></>}
          {input.kind === "approval" && <><label htmlFor="delivery-authority">Approval authority</label><input id="delivery-authority" maxLength={200} value={String(input.details.authority ?? "")} onChange={e => detail("authority", e.target.value)} /><label htmlFor="delivery-reference">Approval reference</label><input id="delivery-reference" maxLength={200} value={String(input.details.reference ?? "")} onChange={e => detail("reference", e.target.value)} />{["submitted_on", "determined_on"].map(key => <div key={key}><label htmlFor={`delivery-${key}`}>{key === "submitted_on" ? "Submitted on" : "Determined on"}</label><input id={`delivery-${key}`} type="date" value={String(input.details[key] ?? "")} onChange={e => detail(key, e.target.value || null)} /></div>)}</>}
          <div className="report-actions"><button className="btn" disabled={busy || stale || !input.title.trim()}>{busy ? "Saving…" : "Save delivery record"}</button><button type="button" className="btn" disabled={busy} onClick={() => { setEditor(null); setError(""); }}>Cancel edits</button></div>
        </fieldset></form>
      </section>}
    </div>
  </section>;
}

function DeliveryDetails({ item }: { item: DeliveryRecord }) {
  if (item.kind === "risk") return <><p>Likelihood: {String(item.details.likelihood || "Not recorded")}</p><p className="report-wording">Consequence: {String(item.details.consequence || "Not recorded")}</p></>;
  if (item.kind === "approval") return <><p>Authority: {String(item.details.authority || "Not recorded")}</p><p>Reference: {String(item.details.reference || "Not recorded")}</p><p>Submitted on: {String(item.details.submitted_on || "Not recorded")}</p><p>Determined on: {String(item.details.determined_on || "Not recorded")}</p></>;
  if (item.kind === "decision") return <><p>Options: {Array.isArray(item.details.options) ? item.details.options.join("; ") : "Not recorded"}</p><p>Chosen: {String(item.details.chosen || "Not recorded")}</p></>;
  return null;
}
