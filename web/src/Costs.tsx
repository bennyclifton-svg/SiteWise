import { useEffect, useRef, useState } from "react";
import { ApiError } from "./api";
import { costApi, type CostPlan, type CostItem, type CostContent, type CostSettings, type CostValue, type CostTotals, type BenchmarkSuggestion } from "./costApi";
import { packageApi, type PackageRecord, type ScopeWork, type ScopeRecord } from "./packageApi";
import { CostGrid, editContent, fmt, subtotal } from "./CostGrid";
import "./reports.css";
import "./costs.css";
// Budget is the one planning figure; estimate was retired (2026-10-10).
const metrics = ["budget", "commitment", "claimed_to_date"];
const metricLabel: Record<string, string> = { budget: "budget", commitment: "approved contract", claimed_to_date: "claimed to date" };
const words = (s: string) => s.replaceAll("_", " ");

const content = editContent;
const valueBlank = (): CostValue => ({ amount: null, low: null, high: null, value_state: "unknown", as_of: null, origin: "user", meaning: "allowance", rationale: "" });
export function Costs({ projectId, tick, onSignedOut }: { projectId: string; tick: number; onSignedOut: () => void }) {
  const saving = useRef(false);
  const loadSequence = useRef(0);
  const pendingLoad = useRef<Promise<void> | null>(null);
  const [quantities, setQuantities] = useState(() => {try {return localStorage.getItem("sitewise.costs.quantities") === "true";} catch {return false;}});
  const [plan, setPlan] = useState<CostPlan | null>(null), [totals, setTotals] = useState<CostTotals | null>(null);
  const [packages, setPackages] = useState<PackageRecord[]>([]), [works, setWorks] = useState<ScopeWork[]>([]), [scopes, setScopes] = useState<(ScopeRecord & { packageTitle: string })[] | null>(null);
  const [selected, setSelected] = useState(""), [reload, setReload] = useState(0), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const [line, setLine] = useState<{ id: string; input: CostContent; base: CostPlan } | null>(null);
  const [settings, setSettings] = useState<{ input: CostSettings; base: CostPlan } | null>(null);
  const [metric, setMetric] = useState<{ name: string; input: CostValue; base: CostPlan } | null>(null);
  const [split, setSplit] = useState<{ base: CostPlan; item: CostItem; names: string[]; amounts: string[]; metric: string; residual: string } | null>(null);
  const [calculate, setCalculate] = useState(false);
  const [geography, setGeography] = useState(""), [quality, setQuality] = useState(""), [suggestions, setSuggestions] = useState<BenchmarkSuggestion[]>([]), [benchmarkMessage, setBenchmarkMessage] = useState(""), [acknowledged, setAcknowledged] = useState("");
  const [scope, setScope] = useState(""), [baseline, setBaseline] = useState(false), [history, setHistory] = useState("");
  function fail(e: unknown) { if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; } setError(e instanceof ApiError && e.status === 404 ? "That cost line is no longer available. Reload the cost plan to see the current scope." : e instanceof ApiError && e.status === 409 ? "The cost plan changed or is frozen. Your entries are kept. Reload and reopen the editor against the saved version." : e instanceof ApiError && e.status === 422 ? `Check the entries and assignments. ${e.message}` : "Could not save or load the cost plan. Your entries are kept; try again."); }
  // The ledger reloads after every save; packages and works only when the project changes.
  useEffect(() => {
    if (saving.current) return;
    let stop = false;
    const sequence = ++loadSequence.current;
    const previous = pendingLoad.current;
    // Coalesce refreshes into one follow-up after an in-flight read, so a source
    // event arriving during that read cannot leave the ledger on its old snapshot.
    const promise = previous ? previous.catch(() => null).then(() => {
      if (stop) return null;
      return load();
    }) : load();
    function load() {
      return history ? Promise.all([costApi.read(projectId, history), costApi.totals(projectId, "overall", history)]).then(([plan, totals]) => ({plan, totals})) : costApi.prepare(projectId);
    }
    const tracked = promise.then(result => {
      if (result && !stop && sequence === loadSequence.current) { setPlan(result.plan); setTotals(result.totals); }
    }).catch(e => { if (!stop && sequence === loadSequence.current) fail(e); throw e; });
    pendingLoad.current = tracked;
    void tracked.catch(() => {}).finally(() => { if (pendingLoad.current === tracked) pendingLoad.current = null; });
    return () => { stop = true; };
  }, [projectId, tick, reload, history]);
  useEffect(() => { let stop = false; Promise.all([packageApi.list(projectId), packageApi.works(projectId)]).then(([pk, w]) => { if (!stop) { setPackages(pk.items); setWorks(w.items); setScopes(null); } }).catch(e => { if (!stop) fail(e); }); return () => { stop = true; }; }, [projectId, tick]);
  // Scope lists cost one request per package, so they load only when a line is opened.
  useEffect(() => { if (!selected || scopes) return; let stop = false; Promise.all(packages.map(async x => (await packageApi.scope(projectId, x.id)).items.map(s => ({ ...s, packageTitle: x.title })))).then(rows => { if (!stop) setScopes(rows.flat()); }).catch(e => { if (!stop) fail(e); }); return () => { stop = true; }; }, [selected, scopes, packages, projectId]);
  const item = plan?.items.find(i => i.cost_item_id === selected), editing = !!line || !!settings || !!metric || !!split;
  const inheritedIDs = new Set<string>();
  let ancestor = item?.parent_item_id;
  while (ancestor && !inheritedIDs.has(ancestor)) { inheritedIDs.add(ancestor); ancestor = plan?.items.find(i => i.cost_item_id === ancestor)?.parent_item_id; }
  const references = plan?.links.filter(l => l.cost_item_id === item?.cost_item_id || inheritedIDs.has(l.cost_item_id)) ?? [];
  const frozen = !!plan && plan.status !== "draft" && plan.status !== "unavailable";
  const open = !!item || editing;
  async function save(action: () => Promise<CostPlan>, close = true) {
    if (saving.current) return false;
    saving.current = true; ++loadSequence.current; setBusy(true); setError("");
    try {
      const p = await action();
      setPlan(p); setTotals(p.totals ?? null);
      setReload(n => n + 1);
      if (close) { setLine(null); setSettings(null); setMetric(null); setSplit(null); }
      return true;
    } catch (e) { fail(e); if (e instanceof ApiError && e.status === 404) setReload(n => n + 1); return false; }
    finally { saving.current = false; setBusy(false); }
  }

  async function remove(target: CostItem) {
    if (!plan) return false;
    const ids = new Set([target.cost_item_id]);
    let changed = true;
    while (changed) { changed = false; for (const row of plan.items) { if (ids.has(row.parent_item_id) && !ids.has(row.cost_item_id)) { ids.add(row.cost_item_id); changed = true; } } }
    if (ids.size > 1 && !window.confirm(`Delete ${target.label} and its ${ids.size - 1} child lines?`)) return false;
    const ok = await save(() => costApi.remove(projectId, plan, target.cost_item_id), false);
    if (ok && ids.has(selected)) close();
    return ok;
  }
  const stale = (p: CostPlan) => p.id !== plan?.id || p.version !== plan?.version;
  function close() { setSelected(""); setLine(null); setSettings(null); setMetric(null); setSplit(null); }
  function choose(id: string) { setSelected(id); setLine(null); setSettings(null); setMetric(null); setSplit(null); setScope(""); setSuggestions([]); setBenchmarkMessage(""); setAcknowledged(""); }
  useEffect(() => { if (!open) return; const onKey = (e: KeyboardEvent) => { if (e.key === "Escape" && !(e.target instanceof HTMLInputElement)) close(); }; window.addEventListener("keydown", onKey); return () => window.removeEventListener("keydown", onKey); }, [open]);
  function saveAmount(i: CostItem, metric: "budget" | "commitment", amount: string | null) {
    const current = i.values[metric];
    return save(() => costApi.value(projectId, plan!, i.cost_item_id, metric, { ...valueBlank(), ...current, amount, low: null, high: null, value_state: amount == null ? "unknown" : "known", origin: "user", meaning: current?.meaning || "allowance" }), false);
  }
  const overall = totals?.overall, derived = totals?.buckets?.overall;
  const status = plan?.status === "unavailable" ? "not started" : plan?.status;
  return <section className="costs" aria-label="Costs" aria-busy={busy}>
    <div className="costs-head"><h1>Costs</h1>{plan && <span className="costs-chip">Cost Plan r{plan.revision || 1} · {status}</span>}
      <div className="costs-toolbar">
      <label><input type="checkbox" checked={quantities} onChange={e => {setQuantities(e.target.checked);try{localStorage.setItem("sitewise.costs.quantities",String(e.target.checked));}catch{/* Storage is optional. */}}} /> Quantities</label>
      <button className="btn" disabled={!plan || busy || editing || frozen} onClick={() => { const p = plan!; setSelected(""); setSettings({ base: p, input: { currency: p.currency || "AUD", tax_basis: p.tax_basis || "ex_tax", tax_rate: p.tax_rate ?? null, price_date: p.price_date ?? null, coverage: p.coverage || "", funding_target: p.funding_target ?? null, funding_target_provenance: p.funding_target_provenance || {} } }); }}>Cost settings</button>
        {plan?.id && <button className="btn" disabled={busy || editing || frozen} onClick={() => setBaseline(true)}>Freeze baseline</button>}
        {plan?.baseline_version_id && <button className="btn" disabled={busy || editing} onClick={() => { close(); setHistory(history ? "" : plan?.baseline_version_id!); }}>{history ? "Return to current draft" : "View frozen baseline"}</button>}</div></div>
    {baseline && plan && <div className="report-notice"><p>Freeze this revision and continue in a new editable draft?</p><button className="btn" disabled={busy} onClick={() => { void save(async () => { const p = await costApi.baseline(projectId, plan); setBaseline(false); return p; }); }}>Confirm baseline</button><button className="btn" onClick={() => setBaseline(false)}>Cancel baseline</button></div>}
    {error && <div className="banner" role="alert"><p>{error}</p><button className="btn" disabled={busy} onClick={() => { setError(""); setReload(n => n + 1); }}>Reload cost plan</button></div>}
    {!plan ? <p role="status">Loading cost plan…</p> : <>
      <dl className="costs-totals" aria-label="Cost totals">
        <div><dt>Funding target</dt><dd>{fmt(plan.funding_target)}</dd></div>
        <div><dt>Budget</dt><dd>{subtotal(overall?.budget)}</dd></div>
        <div><dt>Forecast final cost</dt><dd>{subtotal(overall?.forecast)}</dd></div>
        <div><dt>Budget variance</dt><dd>{fmt(derived?.budget_variance)}</dd></div>
        <div><dt>Basis</dt><dd>{plan.currency || "AUD"} · {plan.tax_basis === "inc_tax" ? "including tax" : "excluding tax"} · price date {plan.price_date || "not recorded"}</dd></div>
        <div><dt>Coverage</dt><dd>{plan.coverage || "Not recorded"}</dd></div>
      </dl>
      <CostGrid plan={plan} totals={totals} packages={packages} selected={selected} editable={!frozen && !editing} busy={busy} quantities={quantities} onOpen={choose} onValue={saveAmount}
        onDelete={remove}
        onItem={(c, id, calculate) => save(() => costApi.item(projectId, plan, c, id, calculate), false)} />
    </>}
    {open && <aside className="cost-drawer" aria-label="Cost line">
      <button type="button" className="btn btn-small cost-drawer-close" onClick={close}>Close</button>
      {item && !editing && <section className="report-section"><h2>{item.label}</h2><p>{item.system_id || "No system assignment"} · {item.part_id || "No part assignment"}</p><p>{item.rationale}</p>{item.line_kind === "works" && <><h3>Included work items</h3>{item.scope_work_items?.length ? <ul>{item.scope_work_items.map(w => <li key={w.id}>{w.title}</li>)}</ul> : <p>No included work items.</p>}</>}<div className="report-actions"><button className="btn" disabled={busy || frozen} onClick={() => setLine({ id: item.cost_item_id, input: content(item), base: plan! })}>Edit cost line</button><button className="btn" disabled={busy || frozen || !item.posting || item.excluded} onClick={() => setSplit({ base: plan!, item, names: ["", ""], amounts: ["", ""], metric: "budget", residual: "explicit" })}>Subdivide allowance</button></div>{!item.posting && <p>Grouping line. Amounts are held on its posting children and included once in totals.</p>}{item.posting && metrics.map(m => <div key={m}><p>{metricLabel[m]}: {item.values[m]?.value_state === "known" ? (item.values[m].amount ?? `${item.values[m].low}–${item.values[m].high}`) : "Unknown"}{item.values[m]?.as_of ? ` · as of ${item.values[m].as_of}` : ""}</p><button className="btn btn-small" disabled={busy || frozen || !item.posting} onClick={() => { setCalculate(false); setMetric({ name: m, input: { ...valueBlank(), ...item.values[m] }, base: plan! }); }}>Edit {metricLabel[m]}</button></div>)}<h3>Reviewed benchmarks</h3><p>Look up only matching reviewed benchmarks; missing evidence remains unknown. An applied benchmark sets the budget.</p><form className="package-scope-form" onSubmit={e => { e.preventDefault(); setBusy(true); setAcknowledged(""); void costApi.benchmarks(projectId, item.cost_item_id, geography, quality).then(result => { setSuggestions(result.suggestions); setBenchmarkMessage(result.message); }).catch(fail).finally(() => setBusy(false)); }}><label htmlFor="benchmark-geography">Benchmark geography</label><input id="benchmark-geography" required value={geography} onChange={e => { setGeography(e.target.value); setSuggestions([]); setAcknowledged(""); }} /><label htmlFor="benchmark-quality">Benchmark quality</label><input id="benchmark-quality" required value={quality} onChange={e => { setQuality(e.target.value); setSuggestions([]); setAcknowledged(""); }} /><button className="btn" disabled={busy || frozen}>Find reviewed benchmarks</button></form>{benchmarkMessage && <p role="status">{benchmarkMessage}</p>}{suggestions.map(s => <div key={s.benchmark.id}><p>{s.amount} · {s.benchmark.geography} · {s.benchmark.quality} · {s.benchmark.price_date}</p><p>Includes: {JSON.stringify(s.benchmark.inclusions)}</p><p>Excludes: {JSON.stringify(s.benchmark.exclusions)}</p><p>Sources: {JSON.stringify(s.benchmark.sources)}</p><p>Limitations: {JSON.stringify(s.limitations)}</p><label><input type="checkbox" checked={acknowledged === s.benchmark.id} onChange={e => setAcknowledged(e.target.checked ? s.benchmark.id : "")} /> I reviewed the date, location, quality and scope basis</label><button className="btn" disabled={acknowledged !== s.benchmark.id || busy || frozen} onClick={() => void save(() => costApi.applyBenchmark(projectId, plan!, item.cost_item_id, s, geography, quality))}>Apply benchmark allowance</button></div>)}<h3>Scope references</h3>{!scopes ? <p role="status">Loading scope…</p> : <>{references.map(l => <p key={`${l.cost_item_id}:${l.package_scope_item_id}`}>{l.cost_item_id !== item.cost_item_id ? "Inherited from parent group · " : ""}{scopes.find(s => s.id === l.package_scope_item_id)?.user_text || l.package_scope_item_id}<button className="btn btn-small" disabled={busy || frozen || l.cost_item_id !== item.cost_item_id} onClick={() => void save(() => costApi.link(projectId, plan!, item.cost_item_id, l.package_scope_item_id, true))}>Remove reference</button></p>)}<div className="package-scope-form"><label htmlFor="cost-scope">Package scope reference</label><select id="cost-scope" value={scope} onChange={e => setScope(e.target.value)}><option value="">Choose scope</option>{scopes.map(s => <option key={s.id} value={s.id}>{s.packageTitle} · {s.user_text || s.deliverable || s.item_kind}</option>)}</select><button className="btn" disabled={busy || frozen || !scope} onClick={() => void save(() => costApi.link(projectId, plan!, item.cost_item_id, scope, false))}>Link scope</button></div></>}</section>}
      {line && <section className="report-section"><h2>{line.id ? "Edit cost line" : "Add cost line"}</h2><form className="package-scope-form" onSubmit={e => { e.preventDefault(); void save(() => costApi.item(projectId, line.base, line.input, line.id)); }}><fieldset disabled={busy || stale(line.base)}><label htmlFor="cost-label">Cost label</label><input id="cost-label" required maxLength={500} value={line.input.label} onChange={e => setLine({ ...line, input: { ...line.input, label: e.target.value } })} /><label htmlFor="cost-kind">Line type</label><select id="cost-kind" value={line.input.line_kind} onChange={e => setLine({ ...line, input: { ...line.input, line_kind: e.target.value, category: e.target.value === "project_wide" ? "other" : "", work_item_id: "", system_id: "", package_id: "", package_stage_id: "" } })}>{["works", "fee", "project_wide"].map(k => <option key={k} value={k}>{words(k)}</option>)}</select>
        {line.input.line_kind === "project_wide" && <><label htmlFor="cost-category">Category</label><select id="cost-category" value={line.input.category} onChange={e => setLine({ ...line, input: { ...line.input, category: e.target.value } })}>{["other", "contingency", "escalation", "authority_fees", "exclusions"].map(k => <option key={k} value={k}>{words(k)}</option>)}</select></>}
        {line.input.line_kind === "works" && <><label htmlFor="cost-work">Linked work</label><select id="cost-work" required={!line.input.system_id} value={line.input.work_item_id} onChange={e => setLine({ ...line, input: { ...line.input, work_item_id: e.target.value, system_id: "" } })}><option value="">Choose work</option>{works.map(w => <option key={w.id} value={w.id}>{w.title}</option>)}</select></>}
        {line.input.line_kind !== "project_wide" && <><label htmlFor="cost-package">Cost package</label><select id="cost-package" required={line.input.line_kind === "fee"} value={line.input.package_id} onChange={e => setLine({ ...line, input: { ...line.input, package_id: e.target.value, package_stage_id: "" } })}><option value="">Not assigned</option>{packages.map(p => <option key={p.id} value={p.id}>{p.title}</option>)}</select></>}
        {line.input.line_kind === "fee" && <><label htmlFor="cost-stage">Fee stage</label><select id="cost-stage" value={line.input.package_stage_id} onChange={e => setLine({ ...line, input: { ...line.input, package_stage_id: e.target.value } })}><option value="">Whole package</option>{packages.find(p => p.id === line.input.package_id)?.stages.map(s => <option key={s.id} value={s.id}>{s.label}</option>)}</select></>}
        {(["quantity", "unit", "rate", "rate_basis", "rationale"] as const).map(k => <div key={k}><label htmlFor={`cost-${k}`}>{words(k)}</label><input id={`cost-${k}`} value={line.input[k] ?? ""} onChange={e => setLine({ ...line, input: { ...line.input, [k]: (k === "quantity" || k === "rate") ? e.target.value || null : e.target.value } })} /></div>)}<label><input type="checkbox" checked={line.input.excluded} onChange={e => setLine({ ...line, input: { ...line.input, excluded: e.target.checked } })} /> Excluded from totals</label><button className="btn">Save cost line</button></fieldset>{stale(line.base) && <p role="alert">Saved plan changed. Cancel and reopen this editor; copy any unsaved entries first.</p>}<button className="btn" type="button" onClick={() => setLine(null)}>Cancel cost edit</button></form></section>}
      {settings && <section className="report-section"><h2>Cost settings</h2><form className="package-scope-form" onSubmit={e => { e.preventDefault(); void save(() => costApi.settings(projectId, settings.base, settings.input)); }}><fieldset disabled={busy || stale(settings.base)}>{(["currency", "tax_rate", "price_date", "coverage", "funding_target"] as const).map(k => <div key={k}><label htmlFor={`settings-${k}`}>{words(k)}{k === "tax_rate" ? " (decimal rate, e.g. 0.10)" : ""}</label><input id={`settings-${k}`} type={k === "price_date" ? "date" : "text"} value={settings.input[k] ?? ""} onChange={e => setSettings({ ...settings, input: { ...settings.input, [k]: ["tax_rate", "price_date", "funding_target"].includes(k) ? e.target.value || null : e.target.value, funding_target_provenance: k === "funding_target" ? { source: "User entered in cost settings" } : settings.input.funding_target_provenance } })} /></div>)}<label htmlFor="settings-tax-basis">Tax basis</label><select id="settings-tax-basis" value={settings.input.tax_basis} onChange={e => setSettings({ ...settings, input: { ...settings.input, tax_basis: e.target.value } })}><option value="ex_tax">Excluding tax</option><option value="inc_tax">Including tax</option></select><button className="btn">Save cost settings</button></fieldset><button className="btn" type="button" onClick={() => setSettings(null)}>Cancel settings</button></form></section>}
      {metric && item && <section className="report-section"><h2>Edit {metricLabel[metric.name]}</h2><form className="package-scope-form" onSubmit={e => { e.preventDefault(); void save(() => costApi.value(projectId, metric.base, item.cost_item_id, metric.name, metric.input, calculate)); }}><fieldset disabled={busy || stale(metric.base)}>{metric.name === "budget" && item.quantity != null && item.rate != null && <label><input type="checkbox" checked={calculate} onChange={e => { setCalculate(e.target.checked); setMetric({ ...metric, input: { ...metric.input, amount: null, low: null, high: null, value_state: "known" } }); }} /> Calculate {item.quantity} {item.unit} × {item.rate} from this line</label>}<label htmlFor="cost-state">Value state</label><select id="cost-state" value={metric.input.value_state} onChange={e => setMetric({ ...metric, input: { ...metric.input, value_state: e.target.value, ...(e.target.value === "unknown" ? { amount: null, low: null, high: null } : {}) } })}><option value="unknown">Unknown</option><option value="known">Known</option></select>{(["amount", "low", "high", "as_of", "rationale"] as const).map(k => <div key={k}><label htmlFor={`value-${k}`}>{words(k)}</label><input id={`value-${k}`} disabled={(calculate || metric.input.value_state === "unknown") && ["amount", "low", "high"].includes(k)} required={k === "as_of" && metric.name === "claimed_to_date"} type={k === "as_of" ? "date" : "text"} value={metric.input[k] ?? ""} onChange={e => setMetric({ ...metric, input: { ...metric.input, [k]: k === "rationale" ? e.target.value : e.target.value || null } })} /></div>)}<button className="btn">Save value</button></fieldset><button className="btn" type="button" onClick={() => setMetric(null)}>Cancel value edit</button></form></section>}
      {split && <section className="report-section"><h2>Subdivide allowance</h2><p>The parent becomes a group. Existing scope references remain attached to that group. An explicit residual preserves unallocated value.</p><form className="package-scope-form" onSubmit={e => { e.preventDefault(); void save(() => costApi.subdivide(projectId, split.base, split.item, split.names.map((label, i) => ({ content: { ...content(split.item), ...(split.item.line_kind === "works" ? {package_id:"",package_stage_id:""} : {}), label, code: "", quantity: null, rate: null, unit: "", rate_basis: "" }, amount: split.amounts[i] })), split.metric, split.residual)); }}><fieldset disabled={busy || stale(split.base)}><label htmlFor="split-metric">Allocation metric</label><select id="split-metric" value={split.metric} onChange={e => setSplit({ ...split, metric: e.target.value })}>{metrics.filter(m => m !== "claimed_to_date").map(m => <option key={m} value={m}>{metricLabel[m]}</option>)}</select>{split.names.map((n, i) => <div key={i}><label htmlFor={`allocation-name-${i}`}>Allocation {i + 1} label</label><input id={`allocation-name-${i}`} required value={n} onChange={e => setSplit({ ...split, names: split.names.map((v, j) => j === i ? e.target.value : v) })} /><label htmlFor={`allocation-value-${i}`}>Allocation {i + 1} amount</label><input id={`allocation-value-${i}`} value={split.amounts[i]} onChange={e => setSplit({ ...split, amounts: split.amounts.map((v, j) => j === i ? e.target.value : v) })} /></div>)}<label htmlFor="split-residual">Unallocated value</label><select id="split-residual" value={split.residual} onChange={e => setSplit({ ...split, residual: e.target.value })}><option value="explicit">Keep explicit residual</option><option value="change_total">Change total to entered allocations</option></select><button className="btn">Save subdivision</button></fieldset><button className="btn" type="button" onClick={() => setSplit(null)}>Cancel subdivision</button></form></section>}
    </aside>}
  </section>;
}

export function CostSummary({projectId,tick,onSignedOut}:{projectId:string;tick:number;onSignedOut:()=>void}) {
 const [data,setData]=useState<{plan:CostPlan;totals:CostTotals}|null>(null);
 const [error,setError]=useState(false);
 useEffect(()=>{let stopped=false;setError(false);Promise.all([costApi.read(projectId),costApi.totals(projectId)]).then(([plan,totals])=>{if(!stopped)setData({plan,totals});}).catch(e=>{if(stopped)return;if(e instanceof ApiError&&e.status===401)onSignedOut();else setError(true);});return()=>{stopped=true;};},[projectId,tick]);
 const display=(metric:string)=>{const v=data?.totals.overall[metric];return v?.amount??(v?.low!=null&&v.high!=null?`${v.low}–${v.high}`:"Unknown");};
 return <section className="report-section" aria-label="Cost summary"><h2>Cost summary</h2>{error?<p role="status">Cost summary unavailable. Open Costs to retry.</p>:!data?<p>Loading cost summary…</p>:<><p>{data.plan.currency||"Currency not set"} · Coverage: {data.plan.coverage||"Unknown"}</p><dl><dt>Budget</dt><dd>{display("budget")}</dd><dt>Forecast</dt><dd>{display("forecast")}</dd><dt>Variance</dt><dd>{data.totals.variance??"Unknown"}</dd></dl><p>Saved cost plan totals. Incomplete values remain unknown; edit the shared ledger in Costs.</p></>}</section>;
}
