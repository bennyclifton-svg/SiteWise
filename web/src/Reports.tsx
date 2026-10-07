// Operate in the existing paper workbench: saved reports at left, a readable
// draft at right. Sources open beside the value; edits stay visibly protected.
// Creation, assembly and last-completed consent remain separate actions.
import { useCallback, useEffect, useState } from "react";
import { ApiError } from "./api";
import { reportApi, type SavedReport, type ReportPackage, type ReportView, type ReportReference, type ReportBlock } from "./reportApi";
import "./reports.css";
import { ConfirmDialog } from "./ConfirmDialog";

const referenceNames = { E: "Evidence", U: "User input", C: "Calculation", A: "Assumption" };
const staleNames: Record<string,string> = { not_assembled: "Draft not yet assembled", reading_pending: "Document reading is pending", reading_failed: "Document reading stopped", saved_profile: "Project profile needs updating", package_retired: "This package is retired", packages: "Package or scope changed", works: "Work items changed", delivery: "Delivery records changed", profile_inputs: "Project inputs changed" };

export function Reports({ projectId, tick, onSignedOut }: { projectId: string; tick: number; onSignedOut: () => void }) {
  const [items, setItems] = useState<SavedReport[]>([]);
  const [packages, setPackages] = useState<ReportPackage[]>([]);
  const [packageId, setPackageId] = useState("");
  const [selected, setSelected] = useState("");
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
    setError(e instanceof ApiError && e.status === 409 ? "The saved draft or its source changed. Your wording is still here. Review the latest draft before saving again." : e instanceof ApiError && e.status === 404 ? "This report is no longer available." : "Could not load or save this report. Try again.");
  }, [onSignedOut]);
  useEffect(() => {
    let stopped = false;
    Promise.all([reportApi.list(projectId), reportApi.packages(projectId)]).then(([reports, result]) => {
      if (stopped) return;
      setItems(reports);
      const available = result.items.filter(p => p.kind === "services" && !p.retired_at);
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
    try { const report = await reportApi.create(projectId, packageId); setItems(old => [...old, report]); setView(null); setSelected(report.id); }
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
  const latestBlock = latest?.draft?.sections.flatMap(section => section.blocks).find(block => block.id === editor?.target);
  const draft = view?.draft;
  return <section className="reports" aria-label="Reports" aria-busy={busy || loading}>
    <div className="report-library">
      <h1>Reports</h1>
      <p className="report-muted">Build an RFP from the saved project and package scope.</p>
      <form onSubmit={e => { e.preventDefault(); void create(); }}>
        <label htmlFor="report-package">Services package</label>
        <select id="report-package" value={packageId} onChange={e => setPackageId(e.target.value)} disabled={busy || !!editor || !packages.length}>
          {!packages.length && <option value="">No services packages</option>}
          {packages.map(p => <option key={p.id} value={p.id}>{p.title}</option>)}
        </select>
        <button className="btn" disabled={busy || !!editor || !packageId}>Create RFP draft</button>
      </form>
      {!loading && !packages.length && <p className="report-muted">A services package is needed before an RFP can be drafted.</p>}
      <h2>Saved reports</h2>
      {loading ? <p role="status">Loading reports…</p> : !items.length ? <p>No reports yet.</p> : <ul className="report-list">{items.map(report => <li key={report.id}><button type="button" aria-current={selected === report.id ? "true" : undefined} disabled={busy || !!editor} onClick={() => { setSelected(report.id); setView(null); setError(""); setReadingChoice(false); }}>{report.title}</button></li>)}</ul>}
    </div>
    <div className="report-reader">
      {error && <div className="banner" role="alert"><p>{error}</p>{editConflict && editor ? <button type="button" className="btn btn-small" disabled={busy} onClick={() => void reviewLatest()}>Review latest draft</button> : <button type="button" className="btn btn-small" disabled={busy || !!editor} onClick={() => { setError(""); setReload(n => n + 1); }}>Retry loading</button>}</div>}
      {latest && editor && <section className="report-notice" aria-label="Review concurrent edit"><h3>Latest saved wording</h3><p className="report-wording">{latestBlock?.text ?? "This source is no longer in the saved draft. Your unsaved text remains in the editor below."}</p><p>Compare this with your unsaved wording below before choosing which to keep.</p>{latestBlock && !latestBlock.source_missing && latest.draft && <button type="button" className="btn" disabled={busy} onClick={() => { setView(latest); setEditor({ ...editor, version: latest.draft!.version }); setLatest(null); setEditConflict(false); setError(""); }}>Keep my wording against this version</button>}</section>}
      {!selected ? <p className="report-empty">Choose a saved report, or create an RFP draft.</p> : !view ? <p role="status">Opening report…</p> : <>
        <header className="report-toolbar"><div><h2>{view.report.title}</h2><p className="report-muted">Draft — not for issue · {draft ? `Version ${draft.version} · ${draft.reporting_date}` : "Not yet assembled"}</p></div><button type="button" className="btn" disabled={busy || !!editor || view.stale.includes("package_retired")} onClick={() => void refresh()}>{busy ? "Saving…" : draft ? "Refresh draft" : "Assemble draft"}</button></header>
        {!!view.stale.length && <div className="report-notice" role="status"><strong>Review needed</strong><ul>{view.stale.map(reason => <li key={reason}>{staleNames[reason] ?? "Saved sources or report rules changed"}</li>)}</ul><p>Refreshing keeps protected wording and flags changed sources.</p></div>}
        {readingChoice && <div className="report-notice" role="alert"><strong>Document reading is incomplete</strong><p>Wait for the project profile to finish updating, or draft from the last completed state. Missing and failed reading will remain visible.</p><div className="report-actions"><button type="button" className="btn" disabled={busy} onClick={() => void refresh(true)}>Use last completed state</button><button type="button" className="btn btn-small" onClick={() => setReadingChoice(false)}>Wait</button></div></div>}
        {draft && <>
          {!!draft.material_assumptions.length && <section className="report-notice" aria-label="Material assumptions"><h3>Material assumptions</h3>{draft.material_assumptions.map(ref => <Citation key={ref.citation_id} reference={ref} />)}</section>}
          {draft.sections.map(section => <section className="report-section" key={section.id}><h3>{section.title}</h3>{section.blocks.map(block => <article className="report-block" key={block.id}>
            <div className="report-block-heading"><h4>{block.label}</h4>{block.provisional && <span>Provisional</span>}{block.edited && <span>Protected wording</span>}</div>
            {block.conflict && <p className="report-conflict">{block.source_missing ? "The source is no longer present. Your wording has been kept for review." : "The source changed. Review your protected wording against its updated source."}</p>}
            {editor?.target === block.id ? <form onSubmit={e => { e.preventDefault(); void save(); }}><label htmlFor="report-wording">Report wording</label><textarea id="report-wording" maxLength={20000} rows={5} value={editor.text} onChange={e => setEditor({ ...editor, text: e.target.value })} disabled={busy} autoFocus /><div className="report-actions"><button className="btn" disabled={busy || editConflict}>Save wording</button><button type="button" className="btn btn-small" disabled={busy} onClick={() => { setEditor(null); setLatest(null); setEditConflict(false); setError(""); }}>Cancel edit</button></div></form> : <><p className="report-wording">{block.text}</p><button type="button" className="btn btn-small" aria-label={`Edit wording: ${block.label}`} disabled={busy || !!editor || block.source_missing} onClick={() => setEditor({ target: block.id, text: block.text, version: draft.version })}>Edit wording</button></>}
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
  return <div className="report-references">{(block.citation_ids ?? []).map(id => { const ref = references.find(r => r.citation_id === id); return ref ? <Citation key={id} reference={ref} /> : <span key={id}>Source unavailable: {id}</span>; })}</div>;
}
function Citation({ reference }: { reference: ReportReference }) {
  return <details className="report-citation" data-label={reference.label}><summary><span>{reference.citation_id}</span> {referenceNames[reference.label]}</summary><p>{reference.basis.text}</p></details>;
}
