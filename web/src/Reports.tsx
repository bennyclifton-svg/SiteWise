// Operate in the existing paper workbench: saved reports at left, a readable
// draft at right. Sources open beside the value; edits stay visibly protected.
// Creation, assembly and last-completed consent remain separate actions.
import { useCallback, useEffect, useState } from "react";
import { ApiError } from "./api";
import { reportApi, type SavedReport, type ReportPackage, type ReportView, type ReportReference, type ReportBlock } from "./reportApi";
import "./reports.css";
import { ConfirmDialog } from "./ConfirmDialog";

function localDate() { const d = new Date(); return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,"0")}-${String(d.getDate()).padStart(2,"0")}`; }
const referenceNames = { E: "Evidence", U: "User input", C: "Calculation", A: "Assumption" };
const staleNames: Record<string,string> = { not_assembled: "Draft not yet assembled", reading_pending: "Document reading is pending", reading_failed: "Document reading stopped", saved_profile: "Project profile needs updating", package_retired: "This package is retired", packages: "Package or scope changed", works: "Work items changed", delivery: "Delivery records changed", profile_inputs: "Project inputs changed" };

export function Reports({ projectId, tick, onSignedOut }: { projectId: string; tick: number; onSignedOut: () => void }) {
  const [items, setItems] = useState<SavedReport[]>([]);
  const [packages, setPackages] = useState<ReportPackage[]>([]);
  const [kind, setKind] = useState("rfp");
  const [issueDate, setIssueDate] = useState(localDate);
  const [discloseBudget, setDiscloseBudget] = useState(false);
  const [acceptStale, setAcceptStale] = useState(false);
  const [staleReason, setStaleReason] = useState("");
  const [issuing, setIssuing] = useState(false);
  const [issueVersion, setIssueVersion] = useState(0);
  const [packageId, setPackageId] = useState("");
  const [selected, setSelected] = useState("");
  useEffect(() => { setIssuing(false); setIssueVersion(0); setIssueDate(localDate()); setDiscloseBudget(false); setAcceptStale(false); setStaleReason(""); }, [selected]);
  const [view, setView] = useState<ReportView | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [readingChoice, setReadingChoice] = useState(false);
  const [editor, setEditor] = useState<{ target: string; text: string; version: number } | null>(null);
  const [editConflict, setEditConflict] = useState(false);
  const [latest, setLatest] = useState<ReportView | null>(null);
	const [resetTarget, setResetTarget] = useState<{ id: string; missing: boolean; version: number } | null>(null);
  const [reload, setReload] = useState(0);
  const fail = useCallback((e: unknown) => {
    if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
    if (e instanceof ApiError) { try { const code = JSON.parse(e.message).error; const messages: Record<string,string> = { report_overflow: "This report exceeds two readable pages. Shorten the draft or reduce its scope before issuing; no PDF was issued.", unreviewed_clauses: "Standard clauses still need owner review before this report can be issued.", report_stale: "Sources changed. Refresh the draft or record why you are issuing the saved state." }; if (messages[code]) { setError(messages[code]); return; } } catch { /* Ordinary API error. */ } }
    setError(e instanceof ApiError && e.status === 409 ? "The saved draft or its source changed. Your wording is still here. Review the latest draft before saving again." : e instanceof ApiError && e.status === 404 ? "This report is no longer available." : "Could not load or save this report. Try again.");
  }, [onSignedOut]);
  useEffect(() => {
    let stopped = false;
    Promise.all([reportApi.list(projectId), reportApi.packages(projectId)]).then(([reports, result]) => {
      if (stopped) return;
      setItems(reports);
      const available = result.items.filter(p => !p.retired_at);
      setPackages(available);
      setPackageId(old => available.some(p => p.id === old) ? old : available[0]?.id ?? "");
      setLoading(false);
    }).catch(e => { if (!stopped) { fail(e); setLoading(false); } });
    return () => { stopped = true; };
  }, [projectId, tick, reload, fail]);
  useEffect(() => {
    if (!selected || editor || busy) return;
    let stopped = false;
    reportApi.read(selected).then(next => { if (!stopped) setView(next); }).catch(e => { if (!stopped) fail(e); });
    return () => { stopped = true; };
  }, [selected, tick, reload, editor, busy, fail]);
  async function create() {
    setBusy(true); setError("");
    try { const report = await reportApi.create(projectId, packageId, kind); setItems(old => [...old, report]); setView(null); setSelected(report.id); }
    catch (e) { fail(e); } finally { setBusy(false); }
  }
  async function refresh(lastCompleted = false) {
    setBusy(true); setError(""); setReadingChoice(false);
    try { await reportApi.refresh(selected, lastCompleted); setReload(n => n + 1); }
    catch (e) {
      let reading = false;
      if (e instanceof ApiError) { try { reading = JSON.parse(e.message).error === "reading_incomplete"; } catch { /* Plain-text errors are handled below. */ } }
      if (reading) setReadingChoice(true); else fail(e);
    } finally { setBusy(false); }
  }
  async function save() {
    if (!editor) return;
    setBusy(true); setError("");
    try { const saved = await reportApi.edit(selected, editor.target, editor.text, editor.version); setView(current => current ? { ...current, draft: saved } : current); setEditor(null); setReload(n => n + 1); }
    catch (e) { if (e instanceof ApiError && e.status === 409) setEditConflict(true); fail(e); } finally { setBusy(false); }
  }
  async function reviewLatest() {
    setBusy(true);
    try { setLatest(await reportApi.read(selected)); } catch (e) { fail(e); } finally { setBusy(false); }
  }
	async function resetEdit() {
	  if (!resetTarget) return;
	  setBusy(true); setError("");
	  try { const saved = await reportApi.reset(selected, resetTarget.id, resetTarget.version); setView(current => current ? { ...current, draft: saved } : current); setReload(n => n + 1); }
	  catch (e) { fail(e); }
	  finally { setBusy(false); setResetTarget(null); }
	}
  async function issue() {
    if (!view?.draft) return;
    setBusy(true); setError("");
    try { await reportApi.issue(selected, { version: issueVersion, reporting_date: issueDate, budget_disclosed: discloseBudget, accept_stale: acceptStale, stale_reason: staleReason }); setIssuing(false); setReload(n => n + 1); }
    catch (e) { fail(e); } finally { setBusy(false); }
  }
  const availablePackages = packages.filter(p => p.kind === (kind === "rft" ? "works" : "services"));
  useEffect(() => { setPackageId(old => availablePackages.some(p => p.id === old) ? old : availablePackages[0]?.id ?? ""); }, [kind, packages]);
  const latestBlock = latest?.draft?.sections.flatMap(section => section.blocks).find(block => block.id === editor?.target);
  const draft = view?.draft;
  return <section className="reports" aria-label="Reports" aria-busy={busy || loading}>
    <div className="report-library">
      <h1>Reports</h1>
      <p className="report-muted">Build reports from the saved project, scope, cost and delivery records.</p>
      <form onSubmit={e => { e.preventDefault(); void create(); }}>
        <label htmlFor="report-kind">Report type</label>
        <select id="report-kind" value={kind} disabled={busy || !!editor} onChange={e => setKind(e.target.value)}><option value="rfp">RFP — services appointment</option><option value="rft">RFT — works tender</option><option value="pmp">PMP — project management plan</option></select>
        {kind !== "pmp" && <><label htmlFor="report-package">{kind === "rfp" ? "Services" : "Works"} package</label>
        <select id="report-package" value={packageId} onChange={e => setPackageId(e.target.value)} disabled={busy || !!editor || !availablePackages.length}>
          {!availablePackages.length && <option value="">No matching packages</option>}
          {availablePackages.map(p => <option key={p.id} value={p.id}>{p.title}</option>)}
        </select></>}
        <button className="btn" disabled={busy || !!editor || (kind !== "pmp" && !packageId)}>Create {kind.toUpperCase()} draft</button>
      </form>
      {!loading && kind !== "pmp" && !availablePackages.length && <p className="report-muted">Create a {kind === "rfp" ? "services" : "works"} package before drafting this report.</p>}
      <h2>Saved reports</h2>
      {loading ? <p role="status">Loading reports…</p> : !items.length ? <p>No reports yet.</p> : <ul className="report-list">{items.map(report => <li key={report.id}><button type="button" aria-current={selected === report.id ? "true" : undefined} disabled={busy || !!editor} onClick={() => { setSelected(report.id); setView(null); setError(""); setReadingChoice(false); }}>{report.title}</button></li>)}</ul>}
    </div>
    <div className="report-reader">
      {error && <div className="banner" role="alert"><p>{error}</p>{editConflict && editor ? <button type="button" className="btn btn-small" disabled={busy} onClick={() => void reviewLatest()}>Review latest draft</button> : <button type="button" className="btn btn-small" disabled={busy || !!editor} onClick={() => { setError(""); setReload(n => n + 1); }}>Retry loading</button>}</div>}
      {latest && editor && <section className="report-notice" aria-label="Review concurrent edit"><h3>Latest saved wording</h3><p className="report-wording">{latestBlock?.text ?? "This source is no longer in the saved draft. Your unsaved text remains in the editor below."}</p><p>Compare this with your unsaved wording below before choosing which to keep.</p>{latestBlock && !latestBlock.source_missing && latest.draft && <button type="button" className="btn" disabled={busy} onClick={() => { setView(latest); setEditor({ ...editor, version: latest.draft!.version }); setLatest(null); setEditConflict(false); setError(""); }}>Keep my wording against this version</button>}</section>}
      {!selected ? <p className="report-empty">Choose a saved report, or create a draft.</p> : !view ? <p role="status">Opening report…</p> : <>
        <header className="report-toolbar"><div><h2>{view.report.title}</h2><p className="report-muted">Draft — not for issue · {draft ? `Version ${draft.version} · ${draft.reporting_date}` : "Not yet assembled"}</p></div><button type="button" className="btn" disabled={busy || !!editor || view.stale.includes("package_retired")} onClick={() => void refresh()}>{busy ? "Saving…" : draft ? "Refresh draft" : "Assemble draft"}</button></header>
        {!!view.stale.length && <div className="report-notice" role="status"><strong>Review needed</strong><ul>{view.stale.map(reason => <li key={reason}>{staleNames[reason] ?? "Saved sources or report rules changed"}</li>)}</ul><p>Refreshing keeps protected wording and flags changed sources.</p></div>}
        {readingChoice && <div className="report-notice" role="alert"><strong>Document reading is incomplete</strong><p>Wait for the project profile to finish updating, or draft from the last completed state. Missing and failed reading will remain visible.</p><div className="report-actions"><button type="button" className="btn" disabled={busy} onClick={() => void refresh(true)}>Use last completed state</button><button type="button" className="btn btn-small" onClick={() => setReadingChoice(false)}>Wait</button></div></div>}
        {!!view.issues?.length && <section className="report-notice" aria-label="Issued reports"><h3>Issued reports</h3><ul>{view.issues.map(item => <li key={item.id}><a href={`/api/reports/${selected}/versions/${item.id}/file`}>Download issue {item.number} — {item.reporting_date}</a></li>)}</ul><p>Issued PDFs retain their saved sources and wording.</p></section>}
        {!!view.changes_since_issue?.length && <details className="report-notice"><summary>Changes since last issue ({view.changes_since_issue.length})</summary><ul>{view.changes_since_issue.map(change => <li key={change.target_id}><strong>{change.kind}</strong>: {change.after || change.before}</li>)}</ul></details>}
        {draft && <>
          <div className="report-actions"><button type="button" className="btn" disabled={busy || !!editor} onClick={() => { setIssuing(!issuing); setIssueVersion(draft.version); if (!issuing) { setAcceptStale(false); setStaleReason(""); setDiscloseBudget(false); } }}>{issuing ? "Cancel issue" : "Issue report"}</button></div>
          {issuing && <form className="report-notice report-issue-form" onSubmit={e => { e.preventDefault(); void issue(); }}><h3>Issue saved report</h3><p>Issuing freezes the wording, sources and PDF. Later changes belong to a new draft.</p><label htmlFor="issue-date">Reporting date</label><input id="issue-date" type="date" required value={issueDate} onChange={e => setIssueDate(e.target.value)} disabled={busy} />{view.report.kind !== "pmp" && <label><input type="checkbox" checked={discloseBudget} onChange={e => setDiscloseBudget(e.target.checked)} disabled={busy} /> Include internal budget amounts in this external report</label>}{!!view.stale.length && <><label><input type="checkbox" checked={acceptStale} onChange={e => setAcceptStale(e.target.checked)} disabled={busy} /> Issue the saved state despite changed or incomplete sources</label>{acceptStale && <><label htmlFor="issue-reason">Reason for using saved state</label><textarea id="issue-reason" required maxLength={1000} value={staleReason} onChange={e => setStaleReason(e.target.value)} disabled={busy} /></>}</>}{issueVersion !== draft.version && <p role="alert">The draft changed while this issue form was open. Cancel and reopen it to review the current version.</p>}<button className="btn" disabled={busy || issueVersion !== draft.version || (!!view.stale.length && (!acceptStale || !staleReason.trim()))}>Freeze and issue PDF</button></form>}
          {!!draft.material_assumptions?.length && <section className="report-notice" aria-label="Material assumptions"><h3>Material assumptions</h3>{draft.material_assumptions.map(ref => <Citation key={ref.citation_id} reference={ref} />)}</section>}
          {draft.sections.map(section => <section className="report-section" key={section.id}><h3>{section.title}</h3>{(section.blocks ?? []).map(block => <article className="report-block" key={block.id}>
            <div className="report-block-heading"><h4>{block.label}</h4>{block.provisional && <span>Provisional</span>}{block.edited && <span>Protected wording</span>}</div>
            {block.conflict && <p className="report-conflict">{block.source_missing ? "The source is no longer present. Your wording has been kept for review." : "The source changed. Review your protected wording against its updated source."}</p>}
            {editor?.target === block.id ? <form onSubmit={e => { e.preventDefault(); void save(); }}><label htmlFor="report-wording">Report wording</label><textarea id="report-wording" maxLength={20000} rows={5} value={editor.text} onChange={e => setEditor({ ...editor, text: e.target.value })} disabled={busy} autoFocus /><div className="report-actions"><button className="btn" disabled={busy || editConflict}>Save wording</button><button type="button" className="btn btn-small" disabled={busy} onClick={() => { setEditor(null); setLatest(null); setEditConflict(false); setError(""); }}>Cancel edit</button></div></form> : <><p className="report-wording">{block.text}</p><button type="button" className="btn btn-small" aria-label={`Edit wording: ${block.label}`} disabled={busy || !!editor || block.source_missing} onClick={() => setEditor({ target: block.id, text: block.text, version: draft.version })}>Edit wording</button></>}
            {block.table && <div className="report-table-scroll"><table className="report-table"><caption>Saved source comparison{block.edited ? " — independent of edited wording" : ""}</caption><thead><tr>{block.table.columns.map((column,i) => <th key={i} scope="col">{column}</th>)}</tr></thead><tbody>{block.table.rows.map((row,i) => <tr key={i}>{row.map((cell,j) => <td key={j}>{cell}</td>)}</tr>)}</tbody></table></div>}
            <BlockReferences block={block} references={draft.references} />
			{block.edited && !block.source_missing && block.generated_text === undefined && <p>Refresh this draft before restoring its saved source wording.</p>}{block.edited && (block.source_missing || block.generated_text !== undefined) && <button type="button" className="btn btn-small" disabled={busy || !!editor} onClick={() => setResetTarget({ id: block.id, missing: block.source_missing, version: draft.version })}>{block.source_missing ? "Remove orphaned wording" : "Restore saved source wording"}</button>}
          </article>)}</section>)}
        </>}
      </>}
    </div>
	{resetTarget && <ConfirmDialog title={resetTarget.missing ? "Remove orphaned wording?" : "Restore saved source wording?"} message={resetTarget.missing ? "This source no longer appears in the draft. Removing its protected wording cannot be undone. Copy any text you need before continuing." : "Your protected wording will be removed and replaced with the source wording saved in this draft. Copy any edits you need before continuing."} confirmLabel={resetTarget.missing ? "Remove wording" : "Restore wording"} busy={busy} busyLabel="Saving…" onConfirm={() => void resetEdit()} onCancel={() => setResetTarget(null)} />}
  </section>;
}

function BlockReferences({ block, references }: { block: ReportBlock; references: ReportReference[] }) {
  return <div className="report-references">{(block.citation_ids ?? []).map(id => { const ref = (references ?? []).find(r => r.citation_id === id); return ref ? <Citation key={id} reference={ref} /> : <span key={id}>Source unavailable: {id}</span>; })}</div>;
}
function Citation({ reference }: { reference: ReportReference }) {
  return <details className="report-citation" data-label={reference.label}><summary><span>{reference.citation_id}</span> {referenceNames[reference.label]}</summary><p>{reference.basis.text}</p></details>;
}
