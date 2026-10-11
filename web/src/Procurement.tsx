// Procurement is one table with one row per package, grouped Consultants and
// Trades (docs/plans/2026-10-10-procurement-table.md). Choosing a discipline
// saves a package; Create makes its RFP or RFT in one request; View opens the
// report full width. Firms and comparison are placeholders until tenderers
// are designed.
import { useCallback, useEffect, useState } from "react";
import { api, ApiError, type Option } from "./api";
import { ConfirmDialog } from "./ConfirmDialog";
import { costApi, type CostPlan } from "./costApi";
import { IconShield } from "./icons";
import { PackageScope } from "./PackageScope";
import { packageApi, type PackageOverview, type PackageRecord, type ScopeWork } from "./packageApi";
import { procurementApi } from "./procurementApi";
import { reportApi, type SavedReport } from "./reportApi";
import { Reports } from "./Reports";
import "./reports.css";

const words = (value: string) => value.replaceAll("_", " ");
const routeLabels: Record<string, string> = { traditional: "Traditional (Lump Sum)", design_construct: "Design & Construct", eci: "Early Contractor Involvement", managing_contractor: "Managing Contractor", alliance: "Alliance", ppp: "PPP" };
const gapWords: Record<string, string> = { install: "nobody carries out the works", design: "nobody designs it", maintain_operation_or_protect: "nobody protects it", inspect_or_test: "nobody inspects or tests it" };

type View = { at: "table" } | { at: "report"; id: string } | { at: "scope"; pkg: PackageRecord };

/** One table row: a saved package, or a profile suggestion not yet accepted. */
interface Row {
  key: string;
  label: string;
  kind: string;
  novation: boolean;
  pkg?: PackageRecord;
  proposalKey?: string;
  report?: SavedReport;
}

interface Loaded { overview: PackageOverview; reports: SavedReport[]; works: ScopeWork[] }

export function Procurement({ projectId, tick, onSignedOut, onOpenProfile }: { projectId: string; tick: number; onSignedOut: () => void; onOpenProfile: () => void }) {
  const [view, setView] = useState<View>({ at: "table" });
  const [data, setData] = useState<Loaded | null>(null);
  const [disciplines, setDisciplines] = useState<Option[]>([]);
  const [reload, setReload] = useState(0);
  const [error, setError] = useState("");
  const [busyRow, setBusyRow] = useState("");
  const [draft, setDraft] = useState<string | null>(null);
  const [removing, setRemoving] = useState<{ pkg: PackageRecord; label: string; costPlan?: CostPlan; costLabels: string[] } | null>(null);

  // The three reads go out together, so the table waits for one round trip.
  useEffect(() => {
    let stopped = false;
    Promise.all([packageApi.list(projectId), reportApi.list(projectId), packageApi.works(projectId)])
      .then(([overview, reports, works]) => { if (!stopped) setData({ overview, reports, works: works.items }); })
      .catch(e => {
        if (stopped) return;
        if (e instanceof ApiError && e.status === 401) onSignedOut();
        else setError("Could not load Procurement. Try again.");
      });
    return () => { stopped = true; };
  }, [projectId, tick, reload, onSignedOut]);
  // The discipline list is fixed for the session; fetch it once.
  useEffect(() => {
    let stopped = false;
    api.catalog().then(c => { if (!stopped) setDisciplines(c.disciplines); }).catch(() => { /* Rows fall back to package titles. */ });
    return () => { stopped = true; };
  }, []);

  const fail = useCallback((e: unknown, conflict?: string) => {
    if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
    if (e instanceof ApiError && e.status === 404 && conflict === "create") { setError("This suggestion changed. The table has been refreshed."); setReload(n => n + 1); return; }
    if (e instanceof ApiError && e.status === 409 && conflict === "row") { setError("That discipline already has a row."); setReload(n => n + 1); return; }
    if (e instanceof ApiError && conflict === "remove") {
      if (e.status === 409) { setError("The package or cost plan changed. The table has been refreshed; review removal again."); setReload(n => n + 1); return; }
      if (e.status === 422) { setError("This package still has scope, costs or open delivery records. Resolve those assignments before removing it."); return; }
    }
    setError("Could not save. Try again.");
  }, [onSignedOut]);

  async function create(row: Row) {
    setBusyRow(row.key); setError("");
    try {
      const { report } = await procurementApi.create(projectId, row.pkg ? { package_id: row.pkg.id } : { proposal_key: row.proposalKey! });
      setView({ at: "report", id: report.id });
    } catch (e) { fail(e, "create"); } finally { setBusyRow(""); }
  }
  async function addRow() {
    if (!draft) return;
    setBusyRow("draft"); setError("");
    try { await procurementApi.addRow(projectId, draft); setDraft(null); setReload(n => n + 1); }
    catch (e) { if (e instanceof ApiError && e.status === 409) setDraft(null); fail(e, "row"); } finally { setBusyRow(""); }
  }
  async function remove() {
    if (!removing) return;
    setBusyRow(removing.pkg.id); setError("");
    try { await procurementApi.retire(projectId, removing.pkg.id, removing.pkg.version, removing.costPlan); setReload(n => n + 1); }
    catch (e) { fail(e, "remove"); } finally { setBusyRow(""); setRemoving(null); }
  }
  async function reviewRemoval(pkg: PackageRecord, label: string) {
    setBusyRow(pkg.id); setError("");
    try {
      const plan = await costApi.read(projectId);
      const ids = new Set(plan.items.filter(i => i.package_id === pkg.id).map(i => i.cost_item_id));
      for (let changed = true; changed;) {
        changed = false;
        for (const item of plan.items) {
          if (ids.has(item.parent_item_id) && !ids.has(item.cost_item_id)) {
            ids.add(item.cost_item_id); changed = true;
          }
        }
      }
      setRemoving({pkg, label, costPlan: ids.size ? plan : undefined, costLabels: plan.items.filter(i => ids.has(i.cost_item_id)).map(i => i.label)});
    } catch (e) { fail(e, "remove"); } finally { setBusyRow(""); }
  }
  async function createPmp() {
    setBusyRow("pmp"); setError("");
    try { const report = await reportApi.create(projectId, "", "pmp"); setView({ at: "report", id: report.id }); }
    catch (e) { fail(e); } finally { setBusyRow(""); }
  }
  const back = () => { setView({ at: "table" }); setReload(n => n + 1); };

  if (view.at === "report") return <Reports key={view.id} reportId={view.id} tick={tick} onBack={back} onSignedOut={onSignedOut} />;
  if (view.at === "scope") return <section className="reports report-view" aria-label="Package scope">
    <div className="scope-back"><button type="button" className="btn btn-small report-back" onClick={back}>← Procurement</button></div>
    <PackageScope projectId={projectId} pkg={view.pkg} tick={tick} onSignedOut={onSignedOut} />
  </section>;

  const label = (pkg: PackageRecord) => disciplines.find(d => d.id === pkg.discipline_id)?.label ?? pkg.title;
  const live = data?.overview.items.filter(p => !p.retired_at) ?? [];
  const rows: Row[] = data ? [
    ...live.map(pkg => ({ key: pkg.id, label: label(pkg), kind: pkg.kind, novation: pkg.novation, pkg, report: data.reports.find(r => r.package_id === pkg.id && r.kind !== "pmp") })),
    ...data.overview.suggestions.map(s => ({ key: s.key, label: s.title, kind: s.kind || "services", novation: s.novation, proposalKey: s.key })),
    // A head-contract proposal that adds lines to an existing package is
    // handled in that package's scope, so only a new one shows as a row.
    ...(data.overview.proposed_works && !data.overview.proposed_works.package_id ? [{ key: data.overview.proposed_works.key, label: "Main Works", kind: "works", novation: false, proposalKey: data.overview.proposed_works.key }] : []),
  ] : [];
  const byTitle = (a: Row, b: Row) => a.label.localeCompare(b.label);
  const consultants = rows.filter(r => r.kind === "services").sort(byTitle);
  const trades = rows.filter(r => r.kind !== "services").sort(byTitle);
  const used = new Set(live.map(p => p.discipline_id).filter(Boolean));
  const options = (prefix: string) => disciplines.filter(d => d.id.startsWith(prefix) && !used.has(d.id));
  const pmp = data?.reports.find(r => r.kind === "pmp");

  const route = data?.overview.route;
  const workTitle = (id: string) => data?.works.find(w => w.id === id)?.title ?? "Work item";
  const gaps = data?.overview.gaps.filter(g => g.state === "gap" || g.state === "overlap") ?? [];
  const waiting = data?.overview.gaps.filter(g => g.state === "not_yet_accepted").length ?? 0;
  const gapItems = new Set(gaps.map(g => g.work_item_id)).size;

  function rowView(row: Row) {
    const busy = busyRow === row.key;
    const doc = row.kind === "services" ? "RFP" : "RFT";
    return <tr key={row.key} className={row.pkg ? undefined : "row-suggested"}>
      <th scope="row" className="discipline-cell">
        {row.novation && <span className="novation-mark" title="Novated consultant"><IconShield /><span className="sr-only">Novated: </span></span>}
        {row.label}{!row.pkg && <span className="suggested-tag">suggested</span>}
      </th>
      {[1, 2, 3].map(n => <td key={n} className="firm-cell"><input disabled placeholder="Add firm" aria-label={`Firm ${n} for ${row.label}`} title="Firms come later" /></td>)}
      <td>{row.kind === "supply" ? "—"
        : row.report ? <button type="button" className="table-action" aria-label={`View ${doc} for ${row.label}`} onClick={() => setView({ at: "report", id: row.report!.id })}>View</button>
        : <button type="button" className="table-action" aria-label={`Create ${doc} for ${row.label}`} disabled={busy} onClick={() => void create(row)}>{busy ? "Creating…" : "Create"}</button>}</td>
      <td><button type="button" className="table-action" disabled title="Tender comparison comes later">Compare</button></td>
      <td className="row-menu-cell">{row.pkg && <details className="row-menu">
        <summary aria-label={`More for ${row.label}`}>⋯</summary>
        <div className="row-menu-items">
          <button type="button" onClick={e => { e.currentTarget.closest("details")?.removeAttribute("open"); setView({ at: "scope", pkg: row.pkg! }); }}>Scope</button>
          <button type="button" disabled={busy} onClick={e => { e.currentTarget.closest("details")?.removeAttribute("open"); void reviewRemoval(row.pkg!, row.label); }}>Remove</button>
        </div>
      </details>}</td>
    </tr>;
  }
  const group = (title: string, list: Row[]) => list.length > 0 && <tbody>
    <tr className="group-row"><th colSpan={7} scope="rowgroup">{title}</th></tr>
    {list.map(rowView)}
  </tbody>;

  return <section className="procurement" aria-label="Procurement">
    <header className="procurement-head">
      <h1>Procurement</h1>
      <button type="button" className="btn btn-small" disabled={!data || draft !== null} onClick={() => { setDraft(""); setError(""); }}><span aria-hidden="true">+ </span>Add discipline</button>
    </header>
    {error && <div className="banner" role="alert"><p>{error}</p><button type="button" className="btn btn-small" onClick={() => { setError(""); setReload(n => n + 1); }}>Reload</button></div>}
    {route && <div className="procurement-line" role="note">{route.value === ""
      ? <p><strong>Procurement route not answered.</strong> Packages assume Design &amp; Construct: the head contract designs, consultants are novated.</p>
      : route.confirm ? <p><strong>{routeLabels[route.value] ?? words(route.value)}</strong> is proposed as Design &amp; Construct. Confirm who designs.</p>
      : <p><strong>{routeLabels[route.value] ?? words(route.value)}.</strong> {route.builder_designs ? "The head contract designs; consultants are novated." : "Consultants design; the head contract constructs."}</p>}
      <button type="button" className="btn btn-small" onClick={onOpenProfile}>{route.value === "" ? "Answer on the profile" : "Change on the profile"}</button></div>}
    {data && <p className="procurement-line">{!gaps.length
      ? <>Every accepted work item is covered.{waiting ? ` ${waiting} work item${waiting === 1 ? " is" : "s are"} not yet accepted in Profile under Scope and works.` : ""}</>
      : <>{gapItems} work item{gapItems === 1 ? " has" : "s have"} a gap: {gaps.map(g => `${workTitle(g.work_item_id)} (${g.state === "overlap" ? `two packages hold ${words(g.role ?? "")}` : gapWords[g.role ?? ""] ?? words(g.role ?? "")})`).join(", ")}</>}</p>}
    {!data ? !error && <p role="status">Loading procurement…</p> : <div className="procurement-scroll">
      <table className="procurement-table">
        <thead><tr><th scope="col">Discipline</th><th scope="col">Firm 1</th><th scope="col">Firm 2</th><th scope="col">Firm 3</th><th scope="col">RFP / RFT</th><th scope="col">Comparison</th><th scope="col"><span className="sr-only">Actions</span></th></tr></thead>
        {group("Consultants", consultants)}
        {group("Trades", trades)}
        {draft !== null && <tbody><tr className="draft-row"><td colSpan={7}>
          <form className="draft-form" onSubmit={e => { e.preventDefault(); void addRow(); }}>
            <select aria-label="Discipline" value={draft} disabled={busyRow === "draft"} onChange={e => setDraft(e.target.value)} autoFocus>
              <option value="">Choose a discipline</option>
              <optgroup label="Consultants">{options("consultant.").map(d => <option key={d.id} value={d.id}>{d.label}</option>)}</optgroup>
              <optgroup label="Trades">{options("trade.").map(d => <option key={d.id} value={d.id}>{d.label}</option>)}</optgroup>
            </select>
            <button className="btn btn-small" disabled={!draft || busyRow === "draft"}>{busyRow === "draft" ? "Adding…" : "Add row"}</button>
            <button type="button" className="table-action" disabled={busyRow === "draft"} onClick={() => setDraft(null)}>Cancel</button>
          </form>
        </td></tr></tbody>}
        {!rows.length && draft === null && <tbody><tr><td colSpan={7} className="procurement-empty">No disciplines yet. Add one to start an RFP or RFT.</td></tr></tbody>}
      </table>
    </div>}
    {data && <p className="procurement-line procurement-pmp">Project management plan:{" "}
      {pmp ? <button type="button" className="table-action" aria-label="View project management plan" onClick={() => setView({ at: "report", id: pmp.id })}>View</button>
        : <button type="button" className="table-action" aria-label="Create project management plan" disabled={busyRow === "pmp"} onClick={() => void createPmp()}>{busyRow === "pmp" ? "Creating…" : "Create"}</button>}</p>}
    {removing && <ConfirmDialog title={`Remove ${removing.label}?`} message={removing.costLabels.length ? `Also delete ${removing.costLabels.length} linked cost-plan line(s), including any subdivisions and their amounts: ${removing.costLabels.join(", ")}? Frozen baselines are kept. Any RFP or RFT will show the package as retired.` : "The row leaves the table. Any RFP or RFT made from it will show its package as retired."} confirmLabel={removing.costLabels.length ? "Remove and delete cost lines" : "Remove"} busy={busyRow === removing.pkg.id} busyLabel="Removing…" onConfirm={() => void remove()} onCancel={() => setRemoving(null)} />}
  </section>;
}
