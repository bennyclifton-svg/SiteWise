import { Pending } from "./Activity";
// Procurement is one table with one row per package, grouped Consultants and
// Trades (docs/plans/2026-10-10-procurement-table.md). Choosing a discipline
// saves a package; Create makes its RFP or RFT in one request; View opens the
// report full width. Each package shows the firms whose proposals arrived
// (docs/plans/2026-10-16-procurement-to-award-implementation.md): a proposal
// filed anywhere lands here, and one dropped on a row lands in that package.
import { useCallback, useEffect, useState, type DragEvent } from "react";
import { api, ApiError, type Option, type Doc } from "./api";
import { FILED_DOCUMENT_TYPE } from "./documentDrag";
import { ConfirmDialog } from "./ConfirmDialog";
import { costApi, type CostPlan } from "./costApi";
import { PackageScope } from "./PackageScope";
import { PackageStages } from "./PackageStages";
import { IconBin } from "./icons";
import { packageApi, type PackageOverview, type PackageRecord, type ScopeWork } from "./packageApi";
import { procurementApi } from "./procurementApi";
import { Prices } from "./Prices";
import { STATUS_WORDS, submissionApi, uploadProposal, type Board, type FirmCard } from "./submissionApi";
import { fmt } from "./CostGrid";
import { reportApi, type SavedReport } from "./reportApi";
import { Reports } from "./Reports";
import "./reports.css";

const words = (value: string) => value.replaceAll("_", " ");
const routeLabels: Record<string, string> = { traditional: "Traditional (Lump Sum)", design_construct: "Design & Construct", eci: "Early Contractor Involvement", managing_contractor: "Managing Contractor", alliance: "Alliance", ppp: "PPP" };
const gapWords: Record<string, string> = { install: "nobody carries out the works", design: "nobody designs it", maintain_operation_or_protect: "nobody protects it", inspect_or_test: "nobody inspects or tests it" };

type View = { at: "table" } | { at: "report"; id: string } | { at: "scope"; pkg: PackageRecord } | { at: "prices"; pkg: string; report?: string };

/** One table row: a saved package, or a profile suggestion not yet accepted. */
interface Row {
  key: string;
  label: string;
  kind: string;
  pkg?: PackageRecord;
  proposalKey?: string;
  report?: SavedReport;
}

interface Loaded { overview: PackageOverview; reports: SavedReport[]; works: ScopeWork[]; board: Board }

export function Procurement({ projectId, tick, onSignedOut, onOpenProfile, onOpenCostPlan }: { projectId: string; tick: number; onSignedOut: () => void; onOpenProfile: () => void; onOpenCostPlan?: () => void }) {
  const [view, setView] = useState<View>({ at: "table" });
  const [data, setData] = useState<Loaded | null>(null);
  const [disciplines, setDisciplines] = useState<Option[]>([]);
  const [reload, setReload] = useState(0);
  const [error, setError] = useState("");
  const [busyRow, setBusyRow] = useState("");
  const [uploadingRow, setUploadingRow] = useState("");
  const [draft, setDraft] = useState<string | null>(null);
  const [removing, setRemoving] = useState<{ pkg: PackageRecord; label: string; costPlan?: CostPlan; costLabels: string[] } | null>(null);
  const [dropRow, setDropRow] = useState("");
  const [picker, setPicker] = useState<PackageRecord | null>(null);
  const [filed, setFiled] = useState<Doc[]>([]);
  const [chosen, setChosen] = useState("");
  const [loadingFiles, setLoadingFiles] = useState(false);

  // The three reads go out together, so the table waits for one round trip.
  useEffect(() => {
    let stopped = false;
    Promise.all([packageApi.list(projectId), reportApi.list(projectId), packageApi.works(projectId), submissionApi.board(projectId)])
      .then(([overview, reports, works, board]) => {
        if (stopped) return;
        setData({ overview, reports, works: works.items, board });
        // Proposed consultants are saved as rows (owner decision 3,
        // docs/plans/2026-10-21-source-to-cost-plan-design.md); the reads
        // above stay one round trip, and only a project with new proposals
        // pays a second.
        if (overview.suggestions.some(s => s.kind === "services")) {
          procurementApi.adopt(projectId).then(r => { if (!stopped && r.added > 0) setReload(n => n + 1); }).catch(() => { /* Rows stay as proposed. */ });
        }
      })
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
  // A suggested row holds nothing, so it goes without a confirm; Add
  // discipline brings it back.
  async function dismiss(row: Row) {
    setBusyRow(row.key); setError("");
    try { await procurementApi.dismiss(projectId, row.proposalKey!); setReload(n => n + 1); }
    catch (e) { fail(e, "create"); } finally { setBusyRow(""); }
  }
  async function createPmp() {
    setBusyRow("pmp"); setError("");
    try { const report = await reportApi.create(projectId, "", "pmp"); setView({ at: "report", id: report.id }); }
    catch (e) { fail(e); } finally { setBusyRow(""); }
  }
  const back = () => { setView({ at: "table" }); setReload(n => n + 1); };
  // A proposal dropped on a row is filed like any upload, and its package
  // is the row's: the firm column appears once it is read.
  async function dropOn(pkg: PackageRecord, e: DragEvent) {
    e.preventDefault(); e.stopPropagation(); setDropRow("");
    const document = e.dataTransfer.getData(FILED_DOCUMENT_TYPE);
    if (document) { await linkDocument(pkg, document); return; }
    const files = Array.from(e.dataTransfer.files);
    if (!files.length) return;
    setBusyRow(pkg.id); setUploadingRow(pkg.id); setError("");
    try { for (const f of files) await uploadProposal(projectId, pkg.id, f); setReload(n => n + 1); }
    catch (err) { fail(err); } finally { setBusyRow(""); setUploadingRow(""); }
  }
  async function linkDocument(pkg: PackageRecord, document: string) {
    setBusyRow(pkg.id); setError("");
    try { await submissionApi.linkDocument(projectId, pkg.id, document); setPicker(null); setReload(n => n + 1); }
    catch (err) { fail(err); } finally { setBusyRow(""); }
  }
  async function chooseFiled(pkg: PackageRecord) {
    setPicker(pkg); setChosen(""); setFiled([]); setLoadingFiles(true); setError("");
    try { const list = await api.documents(projectId); setFiled(list.documents.filter(d => d.status === "filed")); }
    catch (err) { fail(err); } finally { setLoadingFiles(false); }
  }
  async function assign(card: FirmCard, pkg: string) {
    if (!pkg) return;
    setBusyRow(card.submission_id); setError("");
    try { await submissionApi.edit(projectId, card.submission_id, { package_id: pkg }); setReload(n => n + 1); }
    catch (err) { fail(err); } finally { setBusyRow(""); }
  }

  if (view.at === "report") return <Reports key={view.id} reportId={view.id} tick={tick} onBack={back} onSignedOut={onSignedOut} />;
  if (view.at === "prices") return <Prices key={view.pkg} projectId={projectId} pkg={view.pkg} reportId={view.report} tick={tick} onBack={back} onSignedOut={onSignedOut} onOpenCostPlan={onOpenCostPlan} />;
  if (view.at === "scope") return <section className="reports report-view" aria-label="Package scope">
    <div className="scope-back"><button type="button" className="btn btn-small report-back" onClick={back}>← Procurement</button></div>
    <PackageScope projectId={projectId} pkg={view.pkg} tick={tick} onSignedOut={onSignedOut} />
  </section>;

  const label = (pkg: PackageRecord) => disciplines.find(d => d.id === pkg.discipline_id)?.label ?? pkg.title;
  const live = data?.overview.items.filter(p => !p.retired_at) ?? [];
  const rows: Row[] = data ? [
    ...live.map(pkg => ({ key: pkg.id, label: label(pkg), kind: pkg.kind, pkg, report: data.reports.find(r => r.package_id === pkg.id && r.kind !== "pmp") })),
    ...data.overview.suggestions.map(s => ({ key: s.key, label: s.title, kind: s.kind || "services", proposalKey: s.key })),
    // A head-contract proposal that adds lines to an existing package is
    // handled in that package's scope, so only a new one shows as a row.
    ...(data.overview.proposed_works && !data.overview.proposed_works.package_id ? [{ key: data.overview.proposed_works.key, label: "Main Works", kind: "works", proposalKey: data.overview.proposed_works.key }] : []),
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
    const dropping = row.pkg && dropRow === row.pkg.id;
    return <tr key={row.key} className={[row.pkg ? "" : "row-suggested", dropping ? "row-drop" : ""].filter(Boolean).join(" ") || undefined}
      data-proposal-drop={row.pkg ? "true" : undefined}
      onDragOver={row.pkg ? (e => { if (e.dataTransfer.types.includes("Files") || e.dataTransfer.types.includes(FILED_DOCUMENT_TYPE)) { e.preventDefault(); e.stopPropagation(); e.dataTransfer.dropEffect = e.dataTransfer.types.includes(FILED_DOCUMENT_TYPE) ? "link" : "copy"; setDropRow(row.pkg!.id); } }) : undefined}
      onDragLeave={row.pkg ? (() => setDropRow("")) : undefined}
      onDrop={row.pkg ? (e => void dropOn(row.pkg!, e)) : undefined}>
      <th scope="row" className="discipline-cell">
        {row.label}
        {row.pkg && <button type="button" className="table-action scope-link" aria-label={`Scope for ${row.label}`} onClick={() => setView({ at: "scope", pkg: row.pkg! })}>Scope</button>}
        {row.pkg && <PackageStages projectId={projectId} pkgId={row.pkg.id} label={row.label} tick={tick} onSignedOut={onSignedOut} />}
      </th>
      <FirmSlots cards={row.pkg ? data?.board.packages[row.pkg.id] ?? [] : []} droppable={!!row.pkg} uploading={!!row.pkg && uploadingRow === row.pkg.id}
        onChoose={() => row.pkg && void chooseFiled(row.pkg)}
        onOpen={() => row.pkg && setView({ at: "prices", pkg: row.pkg.id, report: row.report?.id })} />
      <td>{row.kind === "supply" ? "—"
        : row.report ? <button type="button" className="table-action" aria-label={`View ${doc} for ${row.label}`} onClick={() => setView({ at: "report", id: row.report!.id })}>View</button>
        : <button type="button" className="table-action" aria-label={`Create ${doc} for ${row.label}`} disabled={busy} onClick={() => void create(row)}>{busy ? <Pending delayed>Creating…</Pending> : "Create"}</button>}</td>
      <td>{row.pkg ? (() => {
        const n = data?.board.packages[row.pkg.id]?.length ?? 0;
        return <button type="button" className="table-action" aria-label={`${n > 1 ? "Compare" : n === 1 ? "Review" : "Prices for"} ${row.label}`} onClick={() => setView({ at: "prices", pkg: row.pkg!.id, report: row.report?.id })}>{n > 1 ? `Compare ${n}` : n === 1 ? "Review" : "Prices"}</button>;
      })() : "—"}</td>
      <td className="row-bin-cell">{(row.pkg || row.proposalKey?.startsWith("default:services:")) &&
        <button type="button" className="reg-del" aria-label={`Remove ${row.label}`} title={`Remove ${row.label}`} disabled={busy}
          onClick={() => void (row.pkg ? reviewRemoval(row.pkg, row.label) : dismiss(row))}><IconBin /></button>}</td>
    </tr>;
  }
  const group = (title: string, list: Row[]) => list.length > 0 && <tbody>
    <tr className="group-row"><th colSpan={COLUMNS} scope="rowgroup">{title}</th></tr>
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
    {picker && <form className="procurement-line procurement-unassigned" aria-label="Link filed proposal" onSubmit={e => { e.preventDefault(); if (chosen) void linkDocument(picker, chosen); }}>
      <label>Filed document for {label(picker)}<select autoFocus value={chosen} onChange={e => setChosen(e.target.value)} disabled={loadingFiles || !!busyRow}>
        <option value="">{loadingFiles ? "Loading documents…" : "Choose a filed document"}</option>
        {filed.map(d => <option key={d.id} value={d.id}>{d.fields.find(f => f.field === "title")?.value || d.filename} · {d.filename}</option>)}
      </select></label>
      <p>Links the existing document. If it is already a proposal, moves it to this package with its saved firm and fees.</p>
      <button className="btn btn-small" disabled={!chosen || !!busyRow}>Link proposal</button>
      <button type="button" className="btn btn-small" disabled={!!busyRow} onClick={() => setPicker(null)}>Cancel</button>
    </form>}
    {data && data.board.unassigned.length > 0 && <div className="procurement-line procurement-unassigned" role="note">
      <p><strong>{data.board.unassigned.length} proposal{data.board.unassigned.length === 1 ? " needs" : "s need"} a package.</strong></p>
      {data.board.unassigned.map(c => <label key={c.submission_id}>{c.firm_name || c.filename}
        <select aria-label={`Package for ${c.filename}`} defaultValue="" disabled={busyRow === c.submission_id} onChange={e => void assign(c, e.target.value)}>
          <option value="">Choose package</option>
          {live.map(p => <option key={p.id} value={p.id}>{label(p)}</option>)}
        </select></label>)}
    </div>}
    {!data ? !error && <p role="status"><Pending delayed>Loading procurement…</Pending></p> : <div className="procurement-scroll">
      <table className="procurement-table">
        <thead><tr><th scope="col">Discipline</th>{[1, 2, 3].map(n => <th key={n} scope="col" title="Drop a proposal on a row to add it to that package">Firm {n}</th>)}<th scope="col">RFP / RFT</th><th scope="col">Prices</th><th scope="col"><span className="sr-only">Remove</span></th></tr></thead>
        {group("Consultants", consultants)}
        {group("Trades", trades)}
        {draft !== null && <tbody><tr className="draft-row"><td colSpan={COLUMNS}>
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
        {!rows.length && draft === null && <tbody><tr><td colSpan={COLUMNS} className="procurement-empty">No disciplines yet. Add one to start an RFP or RFT.</td></tr></tbody>}
      </table>
    </div>}
    {data && <p className="procurement-line procurement-pmp">Project management plan:{" "}
      {pmp ? <button type="button" className="table-action" aria-label="View project management plan" onClick={() => setView({ at: "report", id: pmp.id })}>View</button>
        : <button type="button" className="table-action" aria-label="Create project management plan" disabled={busyRow === "pmp"} onClick={() => void createPmp()}>{busyRow === "pmp" ? <Pending delayed>Creating…</Pending> : "Create"}</button>}</p>}
    {removing && <ConfirmDialog title={`Remove ${removing.label}?`} message={removing.costLabels.length ? `Also delete ${removing.costLabels.length} linked cost-plan line(s), including any subdivisions and their amounts: ${removing.costLabels.join(", ")}? Frozen baselines are kept. Any RFP or RFT will show the package as retired.` : "The row leaves the table. Any RFP or RFT made from it will show its package as retired."} confirmLabel={removing.costLabels.length ? "Remove and delete cost lines" : "Remove"} busy={busyRow === removing.pkg.id} busyLabel="Removing…" onConfirm={() => void remove()} onCancel={() => setRemoving(null)} />}
  </section>;
}

/** Discipline, three firm slots, RFP / RFT, Prices and the bin. */
const COLUMNS = 7;

/** FirmSlots fills Firm 1-3 with the package's firms in the board's order,
 * so every row reads as the place firms are compared (owner decision 5,
 * docs/plans/2026-10-21-source-to-cost-plan-design.md). A fourth firm and
 * later show as "+n more" in Firm 3. */
function FirmSlots({ cards, droppable, uploading, onOpen, onChoose }: { cards: FirmCard[]; droppable: boolean; uploading: boolean; onOpen: () => void; onChoose: () => void }) {
  const extra = cards.length - 3;
  return <>{[0, 1, 2].map(i => {
    const c = cards[i];
    const empty = droppable ? <span className="firms-none">{uploading && i === cards.length ? <Pending>Uploading…</Pending> : <>Drop a proposal<br /><button type="button" className="table-action" onClick={onChoose}>Choose filed proposal</button></>}</span> : <span className="firms-none">—</span>;
    return <td key={i} className="firm-slot">{c ? <FirmChip card={c} onOpen={onOpen} /> : empty}
      {i === 2 && extra > 0 && <button type="button" className="table-action" onClick={onOpen}>+{extra} more</button>}</td>;
  })}</>;
}

/** FirmChip shows one firm: its name, the base total the server computed,
 * and its status in words. */
function FirmChip({ card: c, onOpen }: { card: FirmCard; onOpen: () => void }) {
  return <button type="button" className={`firm-chip${c.lowest ? " lowest" : ""}`} data-status={c.status} onClick={onOpen}
    title={`${c.filename} · received ${c.received_on}${c.not_priced ? ` · ${c.not_priced} not priced` : ""}`}>
    <span className="firm-chip-name">{c.firm_name || c.filename}</span>
    <span className="firm-chip-figure">{c.status === "reading" ? <Pending>Reading proposal…</Pending> : c.base_total != null ? `$${fmt(c.base_total).replace(/\.00$/, "")}` : c.known_base && c.known_base !== "0.00" ? `$${fmt(c.known_base).replace(/\.00$/, "")}+` : c.stated_total != null ? <>{`$${fmt(c.stated_total).replace(/\.00$/, "")}`} <small title="The firm's own total, ex GST; the fee schedule is not priced yet">stated</small></> : "—"}</span>
    {c.status !== "received" && c.status !== "reading" && <span className="firm-chip-status">{STATUS_WORDS[c.status]}</span>}
    {c.status === "received" && c.needs_review && <span className="firm-chip-status warn">Check</span>}
  </button>;
}
