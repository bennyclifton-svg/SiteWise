import { Pending } from "./Activity";
import { useEffect, useRef, useState, type DragEvent } from "react";
import { ApiError } from "./api";
import { costApi, type CostPlan, type CostItem, type CostSettings, type CostValue, type CostTotals } from "./costApi";
import { packageApi, type PackageRecord } from "./packageApi";
import { CostGrid, editContent } from "./CostGrid";
import { InvoicesTab, VariationsTab } from "./Invoices";
import { invoiceApi, type InvoiceFilter, type Ledger } from "./invoiceApi";
import { IconDownload } from "./icons";
import { profileApi } from "./profileApi";
import "./invoices.css";
import "./reports.css";
import "./costs.css";
// Budget is the one planning figure; estimate was retired (2026-10-10).
const metrics = ["budget", "commitment", "claimed_to_date"];
const metricLabel: Record<string, string> = { budget: "budget", commitment: "approved contract", claimed_to_date: "claimed to date" };
const words = (s: string) => s.replaceAll("_", " ");

const content = editContent;
const valueBlank = (): CostValue => ({ amount: null, low: null, high: null, value_state: "unknown", as_of: null, origin: "user", meaning: "allowance", rationale: "" });
type CostTab = "plan" | "invoices" | "variations";
const NO_FILTER: InvoiceFilter = { supplier: "", month: "", line: "", status: "" };

function CostDownloads({ projectId, tab, filter }: { projectId: string; tab: CostTab; filter: InvoiceFilter }) {
  const [open, setOpen] = useState(false);
  const box = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const close = (e: MouseEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      e.stopPropagation();
      setOpen(false);
    };
    document.addEventListener("mousedown", close);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", close);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);
  const scoped = tab === "invoices" ? filter : NO_FILTER;
  // Close after the click's default action. Unmounting a download link in the
  // same turn drops the navigation before the browser starts the file.
  const closeAfter = () => { window.setTimeout(() => setOpen(false), 0); };
  return (
    <div className="costs-toolbar costs-export" ref={box} onKeyDown={e => { if (e.key === "Escape") { e.stopPropagation(); setOpen(false); } }}>
      <button type="button" className="btn costs-download" aria-label="Download" title="Download" aria-haspopup="menu" aria-expanded={open} aria-controls="cost-download-menu" onClick={() => setOpen(v => !v)}>
        <IconDownload />
      </button>
      {open && (
        <div id="cost-download-menu" className="costs-download-menu" role="menu" aria-label="Download">
          <a role="menuitem" href={invoiceApi.exportURL(projectId, scoped)} download onClick={closeAfter}>Export to Excel</a>
          <a role="menuitem" href={invoiceApi.exportURL(projectId, NO_FILTER, "csv", "cost_plan")} download onClick={closeAfter}>Cost plan CSV</a>
          <a role="menuitem" href={invoiceApi.exportURL(projectId, scoped, "csv", "invoices")} download onClick={closeAfter}>Invoices CSV</a>
          <button type="button" role="menuitem" onClick={() => { setOpen(false); window.print(); }}>Print</button>
        </div>
      )}
    </div>
  );
}

export function Costs({ projectId, tick, invoiceTick = 0, onSignedOut }: { projectId: string; tick: number; invoiceTick?: number; onSignedOut: () => void }) {
  const [tab, setTab] = useState<CostTab>("plan");
  const [ledger, setLedger] = useState<Ledger | null>(null), [ledgerReload, setLedgerReload] = useState(0), [ledgerError, setLedgerError] = useState(false);
  const [filter, setFilter] = useState<InvoiceFilter>(NO_FILTER), [invoice, setInvoice] = useState("");
  const [incoming, setIncoming] = useState<File[] | null>(null);
  // One ledger feeds both tabs, so a change in one shows in the other.
  useEffect(() => {
    let stop = false;
    const t = setTimeout(() => {
      invoiceApi.ledger(projectId).then(l => { if (!stop) { setLedger(l); setLedgerError(false); } }).catch(e => { if (stop) return; if (e instanceof ApiError && e.status === 401) onSignedOut(); else setLedgerError(true); });
    }, ledger ? 120 : 0);
    return () => { stop = true; clearTimeout(t); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, tick, invoiceTick, ledgerReload]);
  function showInvoices(line: string) { setFilter({ ...NO_FILTER, line }); setInvoice(""); setTab("invoices"); }
  // Files dropped anywhere on the cost plan are invoices for the ledger.
  const dropFiles = {
    onDragEnter: (e: DragEvent) => { if (e.dataTransfer.types.includes("Files")) { e.preventDefault(); e.stopPropagation(); } },
    onDragOver: (e: DragEvent) => { if (e.dataTransfer.types.includes("Files")) { e.preventDefault(); e.stopPropagation(); e.dataTransfer.dropEffect = "copy"; } },
    onDragLeave: (e: DragEvent) => { e.stopPropagation(); },
    onDrop: (e: DragEvent) => { if (!e.dataTransfer.types.includes("Files")) return; e.preventDefault(); e.stopPropagation(); setIncoming(Array.from(e.dataTransfer.files)); setTab("invoices"); },
  };
  const saving = useRef(false);
  const loadSequence = useRef(0);
  const pendingLoad = useRef<Promise<void> | null>(null);
  const [quantities, setQuantities] = useState(() => {try {return localStorage.getItem("sitewise.costs.quantities") === "true";} catch {return false;}});
  const [plan, setPlan] = useState<CostPlan | null>(null), [totals, setTotals] = useState<CostTotals | null>(null);
  const [packages, setPackages] = useState<PackageRecord[]>([]);
  const [selected, setSelected] = useState(""), [reload, setReload] = useState(0), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const [settings, setSettings] = useState<{ input: CostSettings; base: CostPlan } | null>(null);
  const [split, setSplit] = useState<{ base: CostPlan; item: CostItem; names: string[]; amounts: string[]; metric: string; residual: string } | null>(null);
  const [baseline, setBaseline] = useState(false), [history, setHistory] = useState("");
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
  useEffect(() => { let stop = false; packageApi.list(projectId).then(pk => { if (!stop) setPackages(pk.items); }).catch(e => { if (!stop) fail(e); }); return () => { stop = true; }; }, [projectId, tick]);
  const item = plan?.items.find(i => i.cost_item_id === selected), editing = !!settings || !!split;
  const frozen = !!plan && plan.status !== "draft" && plan.status !== "unavailable";
  const open = !!item || editing;
  async function save(action: () => Promise<CostPlan>, close = true) {
    if (saving.current) return false;
    saving.current = true; ++loadSequence.current; setBusy(true); setError("");
    try {
      const p = await action();
      setPlan(p); setTotals(p.totals ?? null);
      setReload(n => n + 1);
      if (close) { setSettings(null); setSplit(null); }
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
  function close() { setSelected(""); setSettings(null); setSplit(null); }
  function choose(id: string) { setSelected(id); setSettings(null); setSplit(null); }
  useEffect(() => { if (!open) return; const onKey = (e: KeyboardEvent) => { if (e.key === "Escape" && !(e.target instanceof HTMLInputElement)) close(); }; window.addEventListener("keydown", onKey); return () => window.removeEventListener("keydown", onKey); }, [open]);
  function saveAmount(i: CostItem, metric: "budget" | "commitment", amount: string | null) {
    const current = i.values[metric];
    return save(() => costApi.value(projectId, plan!, i.cost_item_id, metric, { ...valueBlank(), ...current, amount, low: null, high: null, value_state: amount == null ? "unknown" : "known", origin: "user", meaning: current?.meaning || "allowance" }), false);
  }
  // The construction budget lives on the profile row; saving it there and
  // preparing again re-spreads the construction lines from the new figure.
  async function saveBudget(amount: string | null) {
    if (saving.current) return false;
    saving.current = true; ++loadSequence.current; setBusy(true); setError("");
    try {
      await profileApi.set(projectId, "proj.budget_stated", { value: amount == null ? null : `$${amount}` });
      const r = await costApi.prepare(projectId);
      setPlan(r.plan); setTotals(r.totals);
      return true;
    } catch (e) { fail(e); return false; }
    finally { saving.current = false; setBusy(false); }
  }
  const status = plan?.status === "unavailable" ? "not started" : plan?.status;
  return <section className="costs" aria-label="Costs" aria-busy={busy} {...(tab === "invoices" ? {} : dropFiles)}>
    <div className="costs-head"><h1>Cost Plan</h1><span role="status">{busy && <Pending delayed>Saving…</Pending>}</span>{plan && <span className="costs-chip">Cost Plan r{plan.revision || 1} · {status}</span>}
      <CostDownloads projectId={projectId} tab={tab} filter={filter} />
      <div className="costs-toolbar" hidden={tab !== "plan"}>
      <label><input type="checkbox" checked={quantities} onChange={e => {setQuantities(e.target.checked);try{localStorage.setItem("sitewise.costs.quantities",String(e.target.checked));}catch{/* Storage is optional. */}}} /> Quantities</label>
      <button className="btn" disabled={!plan || busy || editing || frozen} onClick={() => { const p = plan!; setSelected(""); setSettings({ base: p, input: { currency: p.currency || "AUD", tax_basis: p.tax_basis || "ex_tax", tax_rate: p.tax_rate ?? null, price_date: p.price_date ?? null, coverage: p.coverage || "", funding_target: p.funding_target ?? null, funding_target_provenance: p.funding_target_provenance || {} } }); }}>Cost settings</button>
        {plan?.id && <button className="btn" disabled={busy || editing || frozen} onClick={() => setBaseline(true)}>Freeze baseline</button>}
        {plan?.baseline_version_id && <button className="btn" disabled={busy || editing} onClick={() => { close(); setHistory(history ? "" : plan?.baseline_version_id!); }}>{history ? "Return to current draft" : "View frozen baseline"}</button>}</div></div>
    {baseline && plan && <div className="report-notice"><p>Freeze this revision and continue in a new editable draft?</p><button className="btn" disabled={busy} onClick={() => { void save(async () => { const p = await costApi.baseline(projectId, plan); setBaseline(false); return p; }); }}>Confirm baseline</button><button className="btn" onClick={() => setBaseline(false)}>Cancel baseline</button></div>}
    {error && <div className="banner" role="alert"><p>{error}</p><button className="btn" disabled={busy} onClick={() => { setError(""); setReload(n => n + 1); }}>Reload cost plan</button></div>}
    <p className="print-only">{ledger?.project ?? ""} · {tab === "invoices" ? "Invoices" : tab === "variations" ? "Variations" : "Cost plan"} · printed {new Date().toLocaleString("en-AU", { timeZone: "Australia/Sydney", dateStyle: "long", timeStyle: "short" })} · filters: {tab === "invoices" && (filter.supplier || filter.month || filter.line || filter.status) ? [filter.supplier && `supplier ${ledger?.packages.find(p => p.id === filter.supplier)?.title ?? ""}`, filter.month && `month ${filter.month}`, filter.line && `line ${filter.line === "unallocated" ? "Unallocated" : ledger?.lines.find(l => l.id === filter.line)?.label ?? ""}`, filter.status && `status ${filter.status.replace("_", " ")}`].filter(Boolean).join("; ") : "none"}</p>
    <div className="costs-tabs" role="tablist" aria-label="Cost plan views">
      {([["plan", "Cost plan"], ["invoices", "Invoices"], ["variations", "Variations"]] as [CostTab, string][]).map(([id, label]) =>
        <button key={id} type="button" role="tab" id={`cost-tab-${id}`} aria-controls={`cost-panel-${id}`} aria-selected={tab === id} tabIndex={tab === id ? 0 : -1} className="costs-tab"
          onKeyDown={e => { const order: CostTab[] = ["plan", "invoices", "variations"]; const i = order.indexOf(tab); if (e.key === "ArrowRight") setTab(order[(i + 1) % 3]); if (e.key === "ArrowLeft") setTab(order[(i + 2) % 3]); }}
          onClick={() => setTab(id)}>{label}{id === "invoices" && ledger ? <span className="costs-tab-count">{ledger.invoices.length}</span> : null}</button>)}
    </div>
    {ledgerError && <div className="banner" role="alert"><p>Could not load the invoice ledger. The cost plan is unaffected.</p><button className="btn" onClick={() => setLedgerReload(n => n + 1)}>Reload invoices</button></div>}
    <div role="tabpanel" id="cost-panel-invoices" aria-labelledby="cost-tab-invoices" hidden={tab !== "invoices"}>
      {tab === "invoices" && <InvoicesTab projectId={projectId} ledger={ledger} filter={filter} onFilter={setFilter} selected={invoice} onSelect={setInvoice} onLedger={setLedger} onSignedOut={onSignedOut} reload={() => setLedgerReload(n => n + 1)} incoming={incoming} onIncoming={() => setIncoming(null)} />}
    </div>
    <div role="tabpanel" id="cost-panel-variations" aria-labelledby="cost-tab-variations" hidden={tab !== "variations"}><VariationsTab /></div>
    <div role="tabpanel" id="cost-panel-plan" aria-labelledby="cost-tab-plan" hidden={tab !== "plan"}>
    {!plan ? !error && <p role="status"><Pending delayed>Loading cost plan…</Pending></p> : <CostGrid plan={plan} totals={totals} packages={packages} editable={!frozen && !editing} busy={busy} quantities={quantities} onOpen={choose} onValue={saveAmount}
        onSubdivide={item => { choose(item.cost_item_id); setSplit({ base: plan, item, names: ["", ""], amounts: ["", ""], metric: "budget", residual: "explicit" }); }}
        onDetailedValue={(item, name, value) => save(() => costApi.value(projectId, plan, item.cost_item_id, name, value), false)}
        onReorder={ids => save(() => costApi.reorder(projectId, plan, ids), false)}
        onDelete={remove} ledger={history ? null : ledger} onInvoices={showInvoices}
        onItem={(c, id, calculate) => save(() => costApi.item(projectId, plan, c, id, calculate), false)} onBudget={history ? undefined : saveBudget} />}
    </div>
    {open && tab === "plan" && <aside className="cost-drawer" aria-label="Cost line">
      <button type="button" className="btn btn-small cost-drawer-close" onClick={close}>Close</button>
      {item && !editing && <section className="report-section"><h2>{item.label}</h2><p>{item.rationale}</p>{item.line_kind === "works" && <><h3>Included work items</h3>{item.scope_work_items?.length ? <ul>{item.scope_work_items.map(w => <li key={w.id}>{w.title}</li>)}</ul> : <p>No included work items.</p>}</>}{!item.posting && <p>Grouping line. Amounts are held on its posting children and included once in totals.</p>}</section>}
      {settings && <section className="report-section"><h2>Cost settings</h2><form className="package-scope-form" onSubmit={e => { e.preventDefault(); void save(() => costApi.settings(projectId, settings.base, settings.input)); }}><fieldset disabled={busy || stale(settings.base)}>{(["currency", "tax_rate", "price_date", "coverage", "funding_target"] as const).map(k => <div key={k}><label htmlFor={`settings-${k}`}>{words(k)}{k === "tax_rate" ? " (decimal rate, e.g. 0.10)" : ""}</label><input id={`settings-${k}`} type={k === "price_date" ? "date" : "text"} value={settings.input[k] ?? ""} onChange={e => setSettings({ ...settings, input: { ...settings.input, [k]: ["tax_rate", "price_date", "funding_target"].includes(k) ? e.target.value || null : e.target.value, funding_target_provenance: k === "funding_target" ? { source: "User entered in cost settings" } : settings.input.funding_target_provenance } })} /></div>)}<label htmlFor="settings-tax-basis">Tax basis</label><select id="settings-tax-basis" value={settings.input.tax_basis} onChange={e => setSettings({ ...settings, input: { ...settings.input, tax_basis: e.target.value } })}><option value="ex_tax">Excluding tax</option><option value="inc_tax">Including tax</option></select><button className="btn">Save cost settings</button></fieldset><button className="btn" type="button" onClick={() => setSettings(null)}>Cancel settings</button></form></section>}
      {split && <section className="report-section"><h2>Subdivide allowance</h2><p>The parent becomes a group. Existing scope references remain attached to that group. An explicit residual preserves unallocated value.</p><form className="package-scope-form" onSubmit={e => { e.preventDefault(); void save(() => costApi.subdivide(projectId, split.base, split.item, split.names.map((label, i) => ({ content: { ...content(split.item), ...(split.item.line_kind === "works" ? {package_id:"",package_stage_id:""} : {}), label, code: "", quantity: null, rate: null, unit: "", rate_basis: "" }, amount: split.amounts[i] })), split.metric, split.residual)); }}><fieldset disabled={busy || stale(split.base)}><label htmlFor="split-metric">Allocation metric</label><select id="split-metric" value={split.metric} onChange={e => setSplit({ ...split, metric: e.target.value })}>{metrics.filter(m => m !== "claimed_to_date").map(m => <option key={m} value={m}>{metricLabel[m]}</option>)}</select>{split.names.map((n, i) => <div key={i}><label htmlFor={`allocation-name-${i}`}>Allocation {i + 1} label</label><input id={`allocation-name-${i}`} required value={n} onChange={e => setSplit({ ...split, names: split.names.map((v, j) => j === i ? e.target.value : v) })} /><label htmlFor={`allocation-value-${i}`}>Allocation {i + 1} amount</label><input id={`allocation-value-${i}`} value={split.amounts[i]} onChange={e => setSplit({ ...split, amounts: split.amounts.map((v, j) => j === i ? e.target.value : v) })} /></div>)}<label htmlFor="split-residual">Unallocated value</label><select id="split-residual" value={split.residual} onChange={e => setSplit({ ...split, residual: e.target.value })}><option value="explicit">Keep explicit residual</option><option value="change_total">Change total to entered allocations</option></select><button className="btn">Save subdivision</button></fieldset><button className="btn" type="button" onClick={() => setSplit(null)}>Cancel subdivision</button></form></section>}
    </aside>}
  </section>;
}

