import type { TransmittalSelection } from "./RfpTransmittal";
import { Pending } from "./Activity";
// Procurement is one table with one row per package, grouped Consultants and
// Trades (docs/plans/2026-10-10-procurement-table.md). Choosing a discipline
// saves a package; Create makes its RFP or RFT in one request; View opens the
// report full width. Each package shows the firms whose proposals arrived
// (docs/plans/2026-10-16-procurement-to-award-implementation.md): a proposal
// filed anywhere lands here, and one dropped on a row lands in that package.
import { useCallback, useEffect, useState, type DragEvent } from "react";
import { api, ApiError, type Option } from "./api";
import { FILED_DOCUMENT_TYPE } from "./documentDrag";
import { ConfirmDialog } from "./ConfirmDialog";
import { costApi, type CostPlan } from "./costApi";
import { IconBin, IconPrint, IconUpload } from "./icons";
import { packageApi, type CombinePrompt, type PackageOverview, type PackageRecord } from "./packageApi";
import { procurementApi } from "./procurementApi";
import { Prices } from "./Prices";
import { STATUS_WORDS, submissionApi, uploadProposal, type Board, type FirmCard } from "./submissionApi";
import { fmt } from "./CostGrid";
import { reportApi, type SavedReport } from "./reportApi";
import { Reports } from "./Reports";
import { MoreMenu, PageActions } from "./PageActions";
import { InlineCellEditor } from "./table/InlineCellEditor";
import { clickRow, emptySelection, prune } from "./table/selection";
import "./reports.css";

type View = { at: "table" } | { at: "report"; id: string } | { at: "prices"; pkg: string; report?: string };

/** One table row: a saved package, or a profile suggestion not yet accepted. */
interface Row {
  key: string;
  label: string;
  kind: string;
  pkg?: PackageRecord;
  proposalKey?: string;
  report?: SavedReport;
  /** A combined engagement's row, or one of its members shown beneath it
   * (docs/plans/2026-11-07-combined-engagements-slice1.md). */
  group?: boolean;
  member?: boolean;
}

interface Loaded { overview: PackageOverview; reports: SavedReport[]; board: Board }

export function Procurement({ projectId, tick, onSignedOut, onOpenCostPlan, onOpenDocument, transmittal }: { transmittal?: TransmittalSelection; projectId: string; tick: number; onSignedOut: () => void; onOpenCostPlan?: () => void; onOpenDocument?: (documentId: string) => boolean }) {
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
  // Consultant rows selected for Combine (click, Ctrl-click, Shift-click, as in
  // Program and the document register), and the combine being confirmed.
  const [selection, setSelection] = useState(emptySelection);
  // Consultants and Trades are two tabs over the one table (owner, 8
  // November 2026). The open tab is a per-viewer convenience.
  const [tab, setTabState] = useState<ProcurementTab>(() => { try { return localStorage.getItem(TAB_KEY) === "trades" ? "trades" : "consultants"; } catch { return "consultants"; } });
  const setTab = (next: ProcurementTab) => { setTabState(next); setDraft(null); setSelection(emptySelection); try { localStorage.setItem(TAB_KEY, next); } catch { /* Storage is optional. */ } };
  const [combining, setCombining] = useState<{ members: PackageRecord[]; names: string[]; title: string; costLabels: string[]; submissionId?: string } | null>(null);
  // Members unticked on a combine prompt (slice 5): a prompt starts ticked.
  const [unticked, setUnticked] = useState<Record<string, string[]>>({});
  // The unplaced combined proposal being combined for, and the rows ticked.
  const [forProposal, setForProposal] = useState<{ card: FirmCard; ids: string[] } | null>(null);

  // The three reads go out together, so the table waits for one round trip.
  useEffect(() => {
    let stopped = false;
    Promise.all([packageApi.list(projectId), reportApi.list(projectId), submissionApi.board(projectId)])
      .then(([overview, reports, board]) => {
        if (stopped) return;
        setData({ overview, reports, board });
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
    if (e instanceof ApiError && conflict === "group") {
      if (e.status === 409) { setError(e.message.includes("award") ? "Undo the award before changing this combined engagement." : "One of these consultants is already combined. The table has been refreshed."); setReload(n => n + 1); return; }
      if (e.status === 422) { setError(e.message || "Could not change this combined engagement."); setReload(n => n + 1); return; }
      if (e.status === 404) { setError("A row changed. The table has been refreshed."); setReload(n => n + 1); return; }
    }
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
  // Combine shows what moves before it moves: the members, their draft fee
  // lines that nest under the group's line, and the group's name.
  async function reviewCombine(members: PackageRecord[], names: string[], submissionId?: string) {
    setBusyRow("combine"); setError("");
    try {
      const plan = await costApi.read(projectId);
      const ids = new Set(members.map(m => m.id));
      const costLabels = plan.items.filter(i => i.line_kind === "fee" && ids.has(i.package_id) && !i.parent_item_id).map(i => i.label);
      setCombining({ members, names, title: joinNames(names), costLabels, submissionId });
    } catch (e) { fail(e, "group"); } finally { setBusyRow(""); }
  }
  async function combine() {
    if (!combining) return;
    setBusyRow("combine"); setError("");
    try { await procurementApi.combine(projectId, combining.members.map(m => m.id), combining.title, combining.submissionId); setSelection(emptySelection); setForProposal(null); setReload(n => n + 1); }
    catch (e) {
      setSelection(emptySelection);
      if (e instanceof ApiError && e.status === 409 && combining.submissionId && e.message.includes("placed")) { setError("This proposal was placed meanwhile. The table has been refreshed."); setForProposal(null); setReload(n => n + 1); }
      else fail(e, "group");
    } finally { setBusyRow(""); setCombining(null); }
  }
  // A prompt is the PM's to act on: Combine opens the usual confirmation
  // with the ticked rows; Dismiss hides it on this project for good.
  async function dismissPrompt(prompt: CombinePrompt) {
    setBusyRow(prompt.key); setError("");
    try { await procurementApi.dismissPrompt(projectId, prompt.key); setReload(n => n + 1); }
    catch (e) { fail(e); } finally { setBusyRow(""); }
  }
  async function uncombine(group: PackageRecord) {
    setBusyRow(group.id); setError("");
    try { await procurementApi.uncombine(projectId, group.id); setReload(n => n + 1); }
    catch (e) { fail(e, "group"); } finally { setBusyRow(""); }
  }
  async function ungroup(member: PackageRecord) {
    setBusyRow(member.id); setError("");
    try { await procurementApi.ungroup(projectId, member.id); setReload(n => n + 1); }
    catch (e) { fail(e, "group"); } finally { setBusyRow(""); }
  }
  async function rename(group: PackageRecord, title: string): Promise<boolean> {
    const next = title.trim();
    if (!next) return false;
    if (next === group.title) return true;
    try { await procurementApi.rename(projectId, group.id, group.version, next); setReload(n => n + 1); return true; }
    catch (e) {
      if (e instanceof ApiError && e.status === 401) onSignedOut();
      if (e instanceof ApiError && e.status === 409) setReload(n => n + 1);
      return false;
    }
  }
  // A suggested row holds nothing, so it goes without a confirm; Add
  // discipline brings it back.
  async function dismiss(row: Row) {
    setBusyRow(row.key); setError("");
    try { await procurementApi.dismiss(projectId, row.proposalKey!); setReload(n => n + 1); }
    catch (e) { fail(e, "create"); } finally { setBusyRow(""); }
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
    try { await submissionApi.linkDocument(projectId, pkg.id, document); setReload(n => n + 1); }
    catch (err) { fail(err); } finally { setBusyRow(""); }
  }
  async function detach(card: FirmCard) {
    setBusyRow(card.submission_id); setError("");
    try {
      await submissionApi.edit(projectId, card.submission_id, { package_id: "" });
      setReload(n => n + 1);
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        setError("Undo the award before detaching this proposal.");
        setReload(n => n + 1);
      } else if (e instanceof ApiError && e.status === 401) onSignedOut();
      else setError("Could not detach the proposal. Try again.");
    } finally { setBusyRow(""); }
  }
  async function assign(card: FirmCard, pkg: string) {
    if (!pkg) return;
    setBusyRow(card.submission_id); setError("");
    try { await submissionApi.edit(projectId, card.submission_id, { package_id: pkg }); setReload(n => n + 1); }
    catch (err) { fail(err); } finally { setBusyRow(""); }
  }

  if (view.at === "report") return <Reports transmittal={transmittal} key={view.id} reportId={view.id} tick={tick} onBack={back} onSignedOut={onSignedOut} />;
  // An open RFP has its own menu (Reports); prices print from the top right,
  // the table from its own three dots beside New.
  const print = <PageActions surface="procurement"><button type="button" onClick={() => window.print()}>Print procurement</button></PageActions>;
  if (view.at === "prices") return <>{print}<Prices key={view.pkg} projectId={projectId} pkg={view.pkg} reportId={view.report} tick={tick} onBack={back} onSignedOut={onSignedOut} onOpenCostPlan={onOpenCostPlan} onOpenDocument={onOpenDocument} /></>;
  const label = (pkg: PackageRecord) => disciplines.find(d => d.id === pkg.discipline_id)?.label ?? pkg.title;
  const live = data?.overview.items.filter(p => !p.retired_at) ?? [];
  const reportOf = (pkg: PackageRecord) => data?.reports.find(r => r.package_id === pkg.id && r.kind !== "pmp");
  // A group row shows its own name (the PM may rename it); its members are
  // listed beneath it rather than among the other rows.
  const rows: Row[] = data ? [
    ...live.filter(pkg => !pkg.group_id).map((pkg): Row => pkg.role === "group"
      ? { key: pkg.id, label: pkg.title, kind: pkg.kind, pkg, group: true, report: reportOf(pkg) }
      : { key: pkg.id, label: label(pkg), kind: pkg.kind, pkg, report: reportOf(pkg) }),
    ...data.overview.suggestions.map(s => ({ key: s.key, label: s.title, kind: s.kind || "services", proposalKey: s.key })),
    // A head-contract proposal that adds lines to an existing package is
    // handled in that package's scope, so only a new one shows as a row.
    ...(data.overview.proposed_works && !data.overview.proposed_works.package_id ? [{ key: data.overview.proposed_works.key, label: "Main Works", kind: "works", proposalKey: data.overview.proposed_works.key }] : []),
  ] : [];
  const byTitle = (a: Row, b: Row) => a.label.localeCompare(b.label);
  const membersOf = (group: PackageRecord): Row[] => live.filter(p => p.group_id === group.id)
    .map((pkg): Row => ({ key: pkg.id, label: label(pkg), kind: pkg.kind, pkg, report: reportOf(pkg), member: true })).sort(byTitle);
  const consultants = rows.filter(r => r.kind === "services").sort(byTitle).flatMap(r => r.group && r.pkg ? [r, ...membersOf(r.pkg)] : [r]);
  const combinable = (row: Row) => !!row.pkg && row.kind === "services" && !row.group && !row.member;
  const order = consultants.filter(combinable).map(r => r.pkg!.id);
  const tickedRows = consultants.filter(r => combinable(r) && selection.ids.includes(r.pkg!.id));
  // A click on a button, link or field inside the row does its own job.
  const pick = (row: Row, e: { ctrlKey: boolean; metaKey: boolean; shiftKey: boolean; target: EventTarget }) => {
    if ((e.target as HTMLElement).closest("button,a,input,select,textarea,[popover]")) return;
    if (!combinable(row)) { setSelection(emptySelection); return; }
    setSelection(s => clickRow(prune(s, order), row.pkg!.id, order, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey }));
  };
  const trades = rows.filter(r => r.kind !== "services").sort(byTitle);
  const used = new Set(live.map(p => p.discipline_id).filter(Boolean));
  const options = (prefix: string) => disciplines.filter(d => d.id.startsWith(prefix) && !used.has(d.id));
  const shown = tab === "consultants" ? consultants : trades;
  const doc = tab === "consultants" ? "RFP" : "RFT";

  function rowView(row: Row) {
    const busy = busyRow === row.key;
    const doc = row.kind === "services" ? "RFP" : "RFT";
    const dropping = row.pkg && dropRow === row.pkg.id;
    const selectable = combinable(row);
    return <tr key={row.key} className={[row.pkg ? "" : "row-suggested", dropping ? "row-drop" : ""].filter(Boolean).join(" ") || undefined}
      data-proposal-drop={row.pkg ? "true" : undefined} data-combinable={selectable || undefined}
      aria-selected={selectable ? selection.ids.includes(row.pkg!.id) : undefined} onClick={e => pick(row, e)}
      onDragOver={row.pkg ? (e => { if (e.dataTransfer.types.includes("Files") || e.dataTransfer.types.includes(FILED_DOCUMENT_TYPE)) { e.preventDefault(); e.stopPropagation(); e.dataTransfer.dropEffect = e.dataTransfer.types.includes(FILED_DOCUMENT_TYPE) ? "link" : "copy"; setDropRow(row.pkg!.id); } }) : undefined}
      onDragLeave={row.pkg ? (() => setDropRow("")) : undefined}
      onDrop={row.pkg ? (e => void dropOn(row.pkg!, e)) : undefined}>
      {/* Keyboard: Enter selects the row, Space adds or removes it. */}
      <th scope="row" className="discipline-cell" tabIndex={selectable ? 0 : undefined} style={row.member ? { paddingLeft: "2rem" } : undefined}
        onKeyDown={selectable ? (e => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); pick(row, { ctrlKey: e.key === " ", metaKey: false, shiftKey: e.shiftKey, target: e.currentTarget }); } if (e.key === "Escape") setSelection(emptySelection); }) : undefined}>
        {row.group && row.pkg ? <GroupName group={row.pkg} onRename={title => rename(row.pkg!, title)} /> : row.label}
      </th>
      <FirmSlots busy={!!busyRow} onDetach={card => void detach(card)} cards={row.pkg ? data?.board.packages[row.pkg.id] ?? [] : []} droppable={!!row.pkg} uploading={!!row.pkg && uploadingRow === row.pkg.id}
        onOpen={() => row.pkg && setView({ at: "prices", pkg: row.pkg.id, report: row.report?.id })} />
      <td>{row.kind === "supply" ? "—"
        : row.report ? <button type="button" className="table-action" aria-label={`View ${doc} for ${row.label}`} onClick={() => setView({ at: "report", id: row.report!.id })}>View</button>
        : <button type="button" className="table-action" aria-label={`Create ${doc} for ${row.label}`} disabled={busy} onClick={() => void create(row)}>{busy ? <Pending delayed>Creating…</Pending> : "Create"}</button>}</td>
      {/* A group holds its members' proposals, so its prices open here. */}
      <td>{row.pkg && !row.member ? (() => {
        const n = data?.board.packages[row.pkg.id]?.length ?? 0;
        return <button type="button" className="table-action" aria-label={`${n > 1 ? "Compare" : n === 1 ? "Review" : "Prices for"} ${row.label}`} onClick={() => setView({ at: "prices", pkg: row.pkg!.id, report: row.report?.id })}>{n > 1 ? `Compare ${n}` : n === 1 ? "Review" : "Prices"}</button>;
      })() : "—"}</td>
      <td className="row-bin-cell">{row.group && row.pkg ? <button type="button" className="table-action" aria-label={`Uncombine ${row.label}`} disabled={busy} onClick={() => void uncombine(row.pkg!)}>Uncombine</button>
        : row.member && row.pkg ? <button type="button" className="table-action" aria-label={`Remove ${row.label} from group`} disabled={busy} onClick={() => void ungroup(row.pkg!)}>Remove from group</button>
        : (row.pkg || row.proposalKey?.startsWith("default:services:")) &&
        <button type="button" className="reg-del" aria-label={`Remove ${row.label}`} title={`Remove ${row.label}`} disabled={busy}
          onClick={() => void (row.pkg ? reviewRemoval(row.pkg, row.label) : dismiss(row))}><IconBin /></button>}</td>
    </tr>;
  }

  return <section className="procurement" aria-label="Procurement">
    <header className="procurement-head">
      <h1 className="sr-only">Procurement</h1>
      <div className="costs-tabs" role="tablist" aria-label="Procurement views">
        {TABS.map(([id, name]) => {
          const n = id === "consultants" ? consultants.filter(r => !r.member).length : trades.length;
          return <button key={id} type="button" role="tab" id={`procurement-tab-${id}`} aria-controls="procurement-panel" aria-selected={tab === id} tabIndex={tab === id ? 0 : -1} className="costs-tab"
            onKeyDown={e => { if (e.key === "ArrowRight" || e.key === "ArrowLeft") { e.preventDefault(); setTab(id === "consultants" ? "trades" : "consultants"); } }}
            onClick={() => setTab(id)}>{name}{data && n > 0 ? <span className="costs-tab-count">{n}</span> : null}</button>;
        })}
      </div>
      <div className="procurement-actions">
      {tab === "consultants" && tickedRows.length >= 2 && <button type="button" className="btn btn-small" disabled={!!busyRow} onClick={() => void reviewCombine(tickedRows.map(r => r.pkg!), tickedRows.map(r => r.label))}>Combine {tickedRows.length}</button>}
      <button type="button" className="btn btn-small" aria-label="New discipline" disabled={!data || draft !== null} onClick={() => { setDraft(""); setError(""); }}><span aria-hidden="true">+ </span>New</button>
      <MoreMenu label="Procurement actions" items={[{ label: "Print procurement", icon: <IconPrint />, onSelect: () => window.print() }]} />
      </div>
    </header>
    {error && <div className="banner" role="alert"><p>{error}</p><button type="button" className="btn btn-small" onClick={() => { setError(""); setReload(n => n + 1); }}>Reload</button></div>}
    {data && tab === "consultants" && (data.overview.prompts ?? []).map(prompt => {
      // Members still live and combinable; the server ticked them all.
      const members = prompt.package_ids.map(id => live.find(p => p.id === id)).filter((p): p is PackageRecord => !!p && !p.group_id && p.role !== "group");
      const chosen = members.filter(p => !(unticked[prompt.key] ?? []).includes(p.id));
      return members.length >= 2 && <div key={prompt.key} className="procurement-line procurement-prompt" role="note" aria-label="Combine suggestion">
        <p>{prompt.text}</p>
        {members.map(p => <label key={p.id}><input type="checkbox" aria-label={`Include ${label(p)}`} checked={chosen.includes(p)} disabled={!!busyRow}
          onChange={e => { const on = e.target.checked; setUnticked(u => ({ ...u, [prompt.key]: on ? (u[prompt.key] ?? []).filter(x => x !== p.id) : [...(u[prompt.key] ?? []), p.id] })); }} /> {label(p)}</label>)}
        <button type="button" className="btn btn-small" disabled={!!busyRow || chosen.length < 2} onClick={() => void reviewCombine(chosen, chosen.map(label))}>Combine {chosen.length}</button>
        <button type="button" className="table-action" disabled={!!busyRow} onClick={() => void dismissPrompt(prompt)}>Dismiss</button>
      </div>;
    })}
    {data && data.board.unassigned.length > 0 && <div className="procurement-line procurement-unassigned" role="note">
      <p><strong>{data.board.unassigned.length} proposal{data.board.unassigned.length === 1 ? " needs" : "s need"} a package.</strong></p>
      {data.board.unassigned.map(c => <div key={c.submission_id}>
        <label>{c.firm_name || c.filename}{c.combined && " covers several disciplines."}
          <select aria-label={c.combined ? `Place ${c.filename} on one discipline` : `Package for ${c.filename}`} defaultValue="" disabled={busyRow === c.submission_id} onChange={e => void assign(c, e.target.value)}>
            <option value="">{c.combined ? "Place on one discipline" : "Choose package"}</option>
            {live.map(p => <option key={p.id} value={p.id}>{label(p)}</option>)}
          </select></label>
        {/* A combined proposal is never guessed onto rows: the PM ticks them. */}
        {c.combined && (forProposal?.card.submission_id !== c.submission_id
          ? <button type="button" className="table-action" disabled={!!busyRow} onClick={() => setForProposal({ card: c, ids: [] })}>Combine for this proposal</button>
          : <fieldset className="procurement-combine-for"><legend>Tick the disciplines this proposal covers</legend>
            {live.filter(p => p.kind === "services" && p.role !== "group" && !p.group_id).sort((a, b) => label(a).localeCompare(label(b))).map(p => <label key={p.id}>
              <input type="checkbox" aria-label={`${label(p)} for ${c.filename}`} checked={forProposal.ids.includes(p.id)} disabled={!!busyRow}
                onChange={e => { const on = e.target.checked; setForProposal(f => f && ({ ...f, ids: on ? [...f.ids, p.id] : f.ids.filter(x => x !== p.id) })); }} /> {label(p)}</label>)}
            {(() => { const chosen = live.filter(p => forProposal.ids.includes(p.id)); return <button type="button" className="btn btn-small" disabled={!!busyRow || chosen.length < 2} onClick={() => void reviewCombine(chosen, chosen.map(label), c.submission_id)}>Combine {chosen.length}</button>; })()}
            <button type="button" className="table-action" disabled={!!busyRow} onClick={() => setForProposal(null)}>Cancel</button>
          </fieldset>)}
      </div>)}
    </div>}
    {!data ? !error && <p role="status"><Pending delayed>Loading procurement…</Pending></p> : <div className="procurement-scroll" role="tabpanel" id="procurement-panel" aria-labelledby={`procurement-tab-${tab}`}>
      <table className="procurement-table">
        <thead><tr><th scope="col">{tab === "consultants" ? "Consultant" : "Trade"}</th>{[1, 2, 3].map(n => <th key={n} scope="col" title="Drop a proposal on a row to add it to that package">Firm {n}</th>)}<th scope="col">{doc}</th><th scope="col">Prices</th><th scope="col"><span className="sr-only">Remove</span></th></tr></thead>
        {shown.length > 0 && <tbody>{shown.map(rowView)}</tbody>}
        {draft !== null && <tbody><tr className="draft-row"><td colSpan={COLUMNS}>
          <form className="draft-form" onSubmit={e => { e.preventDefault(); void addRow(); }}>
            <select aria-label="Discipline" value={draft} disabled={busyRow === "draft"} onChange={e => setDraft(e.target.value)} autoFocus>
              <option value="">{tab === "consultants" ? "Choose a consultant" : "Choose a trade"}</option>
              {tab === "consultants" ? options("consultant.").map(d => <option key={d.id} value={d.id}>{d.label}</option>)
                : <>{options("trade.").map(d => <option key={d.id} value={d.id}>{d.label}</option>)}
                  {/* A supply is tendered as a trade, with an RFT (owner, 8 November 2026). */}
                  {options("supplier.").length > 0 && <optgroup label="Supply">{options("supplier.").map(d => <option key={d.id} value={d.id}>{d.label}</option>)}</optgroup>}</>}
            </select>
            <button className="btn btn-small" disabled={!draft || busyRow === "draft"}>{busyRow === "draft" ? "Adding…" : "Add row"}</button>
            <button type="button" className="table-action" disabled={busyRow === "draft"} onClick={() => setDraft(null)}>Cancel</button>
          </form>
        </td></tr></tbody>}
        {!shown.length && draft === null && <tbody><tr><td colSpan={COLUMNS} className="procurement-empty">{tab === "consultants" ? "No consultants yet. Add one to start an RFP." : "No trades yet. Add Main Works for the head contract, or a trade let separately, to start an RFT."}</td></tr></tbody>}
      </table>
    </div>}
    {combining && <ConfirmDialog tone="plain" title={`Combine ${combining.members.length} consultants?`}
      message={`${joinNames(combining.names)} become one engagement named “${combining.title}”: one row here and one fee line in the cost plan. ${combining.costLabels.length ? `These fee lines move under it and keep their budgets: ${combining.costLabels.join(", ")}.` : "No fee lines move yet."}${combining.submissionId ? " The proposal is placed on it." : ""} Double-click the name to change it. Uncombine reverses this.`}
      confirmLabel="Combine" busy={busyRow === "combine"} busyLabel="Combining…" onConfirm={() => void combine()} onCancel={() => setCombining(null)} />}
    {removing && <ConfirmDialog title={`Remove ${removing.label}?`} message={removing.costLabels.length ? `Also delete ${removing.costLabels.length} linked cost-plan line(s), including any subdivisions and their amounts: ${removing.costLabels.join(", ")}? Frozen baselines are kept. Any RFP or RFT will show the package as retired.` : "The row leaves the table. Any RFP or RFT made from it will show its package as retired."} confirmLabel={removing.costLabels.length ? "Remove and delete cost lines" : "Remove"} busy={busyRow === removing.pkg.id} busyLabel="Removing…" onConfirm={() => void remove()} onCancel={() => setRemoving(null)} />}
  </section>;
}

/** joinNames reads "A and B" or "A, B and C", the group's default name. */
function joinNames(names: string[]): string {
  return names.length < 2 ? names.join("") : `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}`;
}

/** GroupName is a combined engagement's name: double-click (or Enter, F2)
 * to change it, with the cost plan's inline edit. */
function GroupName({ group, onRename }: { group: PackageRecord; onRename: (title: string) => Promise<boolean> }) {
  const [editing, setEditing] = useState(false), [error, setError] = useState("");
  if (editing) return <InlineCellEditor value={group.title} label={`Name of ${group.title}`} error={error} maxLength={200}
    onCancel={() => { setEditing(false); setError(""); }}
    onCommit={text => { void onRename(text).then(ok => { if (ok) { setEditing(false); setError(""); } else setError("Could not rename. Try again."); }); }} />;
  return <span tabIndex={0} title="Double-click to rename" onDoubleClick={() => setEditing(true)}
    onKeyDown={e => { if (e.key === "F2" || e.key === "Enter") { e.preventDefault(); setEditing(true); } }}>{group.title}</span>;
}

/** Discipline, three firm slots, RFP / RFT, Prices and the bin. */
const COLUMNS = 7;

type ProcurementTab = "consultants" | "trades";
const TABS: [ProcurementTab, string][] = [["consultants", "Consultants"], ["trades", "Trades"]];
const TAB_KEY = "sitewise.procurement.tab";

/** FirmSlots fills Firm 1-3 with the package's firms in the board's order,
 * so every row reads as the place firms are compared (owner decision 5,
 * docs/plans/2026-10-21-source-to-cost-plan-design.md). A fourth firm and
 * later show as "+n more" in Firm 3. */
function FirmSlots({ cards, droppable, uploading, onOpen, onDetach, busy }: { onDetach: (card: FirmCard) => void; busy: boolean; cards: FirmCard[]; droppable: boolean; uploading: boolean; onOpen: () => void }) {
  const extra = cards.length - 3;
  return <>{[0, 1, 2].map(i => {
    const c = cards[i];
    const empty = droppable ? <span className="firms-none">{uploading && i === cards.length ? <Pending>Uploading…</Pending> : <span className="proposal-drop-icon" role="img" aria-label="Drop a proposal" title="Drop a proposal from your computer or document repository"><IconUpload /></span>}</span> : <span className="firms-none">—</span>;
    return <td key={i} className="firm-slot">{c ? <FirmChip card={c} onOpen={onOpen} onDetach={() => onDetach(c)} busy={busy} /> : empty}
      {i === 2 && extra > 0 && <button type="button" className="table-action" onClick={onOpen}>+{extra} more</button>}</td>;
  })}</>;
}

/** FirmChip shows one firm: its name, the base total the server computed,
 * with provenance and review status in its accessible description. */
function FirmChip({ card: c, onOpen, onDetach, busy }: { card: FirmCard; onOpen: () => void; onDetach: () => void; busy: boolean }) {
  const description = [c.filename, `received ${c.received_on}`, STATUS_WORDS[c.status],
    c.needs_review ? "Check proposal" : "",
    c.not_priced ? `${c.not_priced} not priced` : "",
    c.pi_expires_on ? `Professional indemnity${c.pi_limit_cents ? ` $${(c.pi_limit_cents / 100).toLocaleString("en-AU")}` : ""} ${c.pi_expired ? "expired" : "to"} ${c.pi_expires_on}${c.pi_expired ? ": request a current certificate" : ""}` : "",
    c.base_total == null && (!c.known_base || c.known_base === "0.00") && c.stated_total != null
      ? "Firm's stated total, ex GST; the fee schedule is not priced yet" : "",
  ].filter(Boolean).join(" · ");
  return <div className="firm-card"><button type="button" className={`firm-chip${c.lowest ? " lowest" : ""}`} data-status={c.status} onClick={onOpen}
    title={description} aria-description={description}>
    <span className="firm-chip-name">{c.firm_name || c.filename}</span>
    {c.status === "not_proposal" && <span className="firm-chip-status">Named in a document</span>}
    {c.pi_expired && <span className="firm-chip-status warn" title={`Professional indemnity expired ${c.pi_expires_on}: request a current certificate`}>PI expired</span>}
    {/* Shown, not spoken: the description already says the firm is awarded. */}
    {c.status === "awarded" && <span className="firm-awarded" aria-hidden="true" title="Awarded"><svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" focusable="false"><path d="m3.5 8.5 3 3 6-7" /></svg></span>}
    <span className="firm-chip-figure">{c.status === "reading" ? <Pending>Reading proposal…</Pending> : c.base_total != null ? `$${fmt(c.base_total).replace(/\.00$/, "")}` : c.known_base && c.known_base !== "0.00" ? `$${fmt(c.known_base).replace(/\.00$/, "")}+` : c.stated_total != null ? `$${fmt(c.stated_total).replace(/\.00$/, "")}` : "—"}</span>
  </button>
    <button type="button" className="firm-detach" aria-label={`Detach proposal for ${c.firm_name || c.filename}`}
      title={c.status === "awarded" ? "Undo the award before detaching this proposal" : "Detach proposal from this row; keep the filed document"}
      disabled={busy || c.status === "awarded"} onClick={onDetach}>
      <svg viewBox="0 0 16 16" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true" focusable="false"><path d="m4 4 8 8m0-8-8 8" /></svg>
    </button>
  </div>;
}
