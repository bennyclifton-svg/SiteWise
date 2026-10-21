import { Pending } from "./Activity";
// One report, full width, opened from a Procurement row. Sources open beside
// the value; edits stay visibly protected. Assembly and last-completed consent
// remain separate actions.
import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "./api";
import { reportApi, type ReportView, type ReportReference, type ReportBlock, type PageFit } from "./reportApi";
import { submissionApi } from "./submissionApi";
import { ProjectPage, RfpReleaseDates, RfpIssue, RfpSectionEditor, SectionMarkers, controlSection, editableSections } from "./RfpEditor";
import "./reports.css";
import { ConfirmDialog } from "./ConfirmDialog";
import { RfpScopePosition } from "./RfpScopePosition";
import { RfpScopeTable, scopeTable } from "./RfpScopeTable";

function localDate() { const d = new Date(); return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,"0")}-${String(d.getDate()).padStart(2,"0")}`; }
const referenceNames = { E: "Evidence", U: "User input", C: "Calculation", A: "Assumption" };
// Why a draft is stale, shown beside Refresh as one compact line;
// Refresh keeps protected wording.
const staleText = (reason: string) => staleNames[reason] ?? "Saved sources or report rules changed";
const staleNames: Record<string,string> = { not_assembled: "Draft not yet assembled", reading_pending: "Document reading is pending", reading_failed: "Document reading stopped", saved_profile: "Project profile needs updating", package_retired: "This package is retired", packages: "Package or scope changed", works: "Work items changed", delivery: "Delivery records changed", profile_inputs: "Project inputs changed" };

export function Reports({ reportId, tick, onBack, onSignedOut }: { reportId: string; tick: number; onBack: () => void; onSignedOut: () => void }) {
  const [issueDate, setIssueDate] = useState(localDate);
  // null until the user ticks or clears it: a consultant RFP prints the
  // budget by default (owner decision, 13 October 2026); a tender to
  // contractors and the PMP do not.
  const [discloseChoice, setDiscloseBudget] = useState<boolean | null>(null);
  const [acceptStale, setAcceptStale] = useState(false);
  const [staleReason, setStaleReason] = useState("");
  const [issuing, setIssuing] = useState(false);
  const [issueVersion, setIssueVersion] = useState(0);
  useEffect(() => { setIssuing(false); setIssueVersion(0); setIssueDate(localDate()); setDiscloseBudget(null); setAcceptStale(false); setStaleReason(""); }, [reportId]);
  const [view, setView] = useState<ReportView | null>(null);
  const discloseBudget = () => discloseChoice ?? view?.report.kind === "rfp";
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [editor, setEditor] = useState<{ target: string; text: string; version: number } | null>(null);
  const [editConflict, setEditConflict] = useState(false);
  const [latest, setLatest] = useState<ReportView | null>(null);
	const [resetTarget, setResetTarget] = useState<{ id: string; missing: boolean; version: number } | null>(null);
  const [reload, setReload] = useState(0);
  const [busyKey, setBusyKey] = useState("");
  // RFP sections whose controls are open under their heading, and a control
  // to focus once its section has rendered.
  const [openEdits, setOpenEdits] = useState<Set<string>>(new Set());
  const [focusKey, setFocusKey] = useState("");
  const reader = useRef<HTMLDivElement>(null);
  useEffect(() => { setOpenEdits(new Set()); }, [reportId]);
  useEffect(() => {
    if (!focusKey || !reader.current) return;
    const target = reader.current.querySelector<HTMLElement>(`[data-control="${CSS.escape(focusKey)}"]`);
    if (!target) return;
    for (let parent = target.parentElement; parent; parent = parent.parentElement) {
      if (parent instanceof HTMLDetailsElement) parent.open = true;
    }
    target.scrollIntoView({ block: "center", behavior: "smooth" });
    target.querySelector<HTMLElement>("input, select, button")?.focus({ preventScroll: true });
    setFocusKey("");
  }, [focusKey, openEdits, view]);
  // The issue error names the open report's own page limit (consultant RFPs allow four).
  const pageLimit = useRef(2);
  const fail = useCallback((e: unknown) => {
    if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
    if (e instanceof ApiError) { try { const code = JSON.parse(e.message).error; const messages: Record<string,string> = { report_overflow: `This report exceeds ${["", "one", "two", "three", "four"][pageLimit.current] ?? pageLimit.current} readable pages. Shorten the draft or reduce its scope before issuing; no PDF was issued.`, unreviewed_clauses: "Standard clauses or discipline rules still need owner review before this report can be issued.", rfp_open_checks: "Resolve the items marked in red first; no PDF was issued.", invalid_setting: "That value is not allowed for this setting.", report_stale: "Sources changed. Refresh the draft or record why you are issuing the saved state." }; if (messages[code]) { setError(messages[code]); return; } } catch { /* Ordinary API error. */ } }
    setError(e instanceof ApiError && e.status === 409 ? "The saved draft or its source changed. Your wording is still here. Review the latest draft before saving again." : e instanceof ApiError && e.status === 404 ? "This report is no longer available." : "Could not load or save this report. Try again.");
  }, [onSignedOut]);
  useEffect(() => {
    if (!reportId || editor || busy) return;
    let stopped = false;
    // A read that started before a save can land after it; never replace a
    // draft with an older version of itself.
    reportApi.read(reportId).then(async next => {
      if (!stopped && !next.draft && next.issues.length === 0 && !next.stale.includes("package_retired")) {
        await reportApi.ensure(reportId);
        return reportApi.read(reportId);
      }
      return next;
    }).then(next => { if (!stopped) setView(old => old?.draft && next.draft && old.draft.id === next.draft.id && next.draft.version < old.draft.version ? old : next); }).catch(e => { if (!stopped) fail(e); });
    return () => { stopped = true; };
  }, [reportId, tick, reload, editor, busy, fail]);
  async function refresh(lastCompleted = true) {
    setBusy(true); setError("");
    try { await reportApi.refresh(reportId, lastCompleted); setReload(n => n + 1); }
    catch (e) { fail(e); }
    finally { setBusy(false); }
  }
  async function save() {
    if (!editor) return;
    setBusy(true); setError("");
    try { const saved = await reportApi.edit(reportId, editor.target, editor.text, editor.version); setView(current => current ? { ...current, draft: saved } : current); setEditor(null); setReload(n => n + 1); }
    catch (e) { if (e instanceof ApiError && e.status === 409) setEditConflict(true); fail(e); } finally { setBusy(false); }
  }
  async function reviewLatest() {
    setBusy(true);
    try { setLatest(await reportApi.read(reportId)); } catch (e) { fail(e); } finally { setBusy(false); }
  }
	async function resetEdit() {
	  if (!resetTarget) return;
	  setBusy(true); setError("");
	  try { const saved = await reportApi.reset(reportId, resetTarget.id, resetTarget.version); setView(current => current ? { ...current, draft: saved } : current); setReload(n => n + 1); }
	  catch (e) { fail(e); }
	  finally { setBusy(false); setResetTarget(null); }
	}
  // One RFP choice at a time: only the control being saved is disabled, and
  // the returned view replaces the draft so the change shows at once. A
  // project fact (fact:<name>) saves to Profile instead of this RFP.
  async function writeSetting(key: string, value: unknown | null) {
    if (!view?.draft) return;
    setBusyKey(key); setError("");
    try {
      const fact = key.startsWith("fact:") ? key.slice(5) : "";
      const next = fact ? await reportApi.fact(reportId, fact, value === null ? null : String(value), view.draft.version)
        : value === null ? await reportApi.clearSetting(reportId, key, view.draft.version) : await reportApi.setting(reportId, key, value, view.draft.version);
      setView(next);
    } catch (e) { fail(e); } finally { setBusyKey(""); }
  }
  function toggleEdit(section: string) {
    setOpenEdits(old => { const next = new Set(old); if (next.has(section)) next.delete(section); else next.add(section); return next; });
  }
  // A decision opens the section that holds its control and focuses it.
  function jump(key?: string) {
    if (!key || !view?.rfp) return;
    if (key.startsWith("fact:")) {
      const check = view.rfp.decisions.find(c => c.control === key) ?? view.rfp.checks.find(c => c.control === key);
      const row = view.rfp.groups.flatMap(g => g.controls).find(c => (c.key.startsWith("position:") || c.key.startsWith("stage:")) && check?.changes?.includes(c.label));
      if (row) key = row.key;
    }
    const section = controlSection(view.rfp, key);
    if (section) setOpenEdits(old => new Set(old).add(section));
    setFocusKey(key);
  }
  async function confirmDiscipline() {
    const suggestion = view?.rfp_suggestion;
    if (!suggestion || !view) return;
    const projectId = view.report.project_id;
    setBusy(true); setError("");
    try {
      // The package version is read on this one click only, so the report
      // view needs no package list of its own.
      const pkg = (await reportApi.packages(projectId)).items.find(p => p.id === view.report.package_id);
      if (!pkg) { setError("This report's package is no longer available."); return; }
      await reportApi.confirmDiscipline(projectId, pkg.id, suggestion.discipline_id, pkg.version);
      await reportApi.refresh(reportId, true);
      setReload(n => n + 1);
    } catch (e) { fail(e); } finally { setBusy(false); }
  }
  // An RFP's issue form is always open at its end, so it issues the version
  // on screen; other reports issue the version their opened form saw.
  async function issue(version = issueVersion) {
    if (!view?.draft) return;
    setBusy(true); setError("");
    try { await reportApi.issue(reportId, { version, reporting_date: view.rfp ? String(view.rfp.groups.find(g => g.id === "issue")?.controls.find(c => c.key === "field:issued")?.value ?? "") : issueDate, budget_disclosed: discloseBudget(), accept_stale: acceptStale, stale_reason: staleReason }); setIssuing(false); setAcceptStale(false); setStaleReason(""); setReload(n => n + 1); }
    catch (e) { fail(e); } finally { setBusy(false); }
  }
  const latestBlock = latest?.draft?.sections.flatMap(section => section.blocks).find(block => block.id === editor?.target);
  const draft = view?.draft;
  pageLimit.current = view?.page_fit?.limit ?? 2;
  const editable = view?.rfp ? editableSections(view.rfp) : new Set<string>();
  // One saved block as the reader sees it, with protected-wording edits.
  const blockView = (block: ReportBlock) => draft && <article className="report-block" key={block.id}>
            <div className="report-block-heading">{block.label && <h4>{block.label}</h4>}{block.provisional && !view?.rfp && <span>Provisional</span>}{block.edited && <span>Protected wording</span>}</div>
            {block.conflict && <p className="report-conflict">{block.source_missing ? "The source is no longer present. Your wording has been kept for review." : "The source changed. Review your protected wording against its updated source."}</p>}
            {editor?.target === block.id ? <form onSubmit={e => { e.preventDefault(); void save(); }}><label htmlFor="report-wording">Report wording</label><textarea id="report-wording" maxLength={20000} rows={5} value={editor.text} onChange={e => setEditor({ ...editor, text: e.target.value })} disabled={busy} autoFocus /><div className="report-actions"><button className="btn" disabled={busy || editConflict}>Save wording</button><button type="button" className="btn btn-small" disabled={busy} onClick={() => { setEditor(null); setLatest(null); setEditConflict(false); setError(""); }}>Cancel edit</button></div></form> : <>{view?.rfp && scopeTable(block) ? <RfpScopeTable block={block} controls={view.rfp} busy={busy || !!busyKey || !!editor} onWrite={writeSetting} /> : block.id === "rfp:positions" ? <RfpScopePosition block={block} /> : <p className="report-wording">{block.text}</p>}<button type="button" className="btn btn-small" aria-label={`Edit wording: ${block.label}`} disabled={busy || !!editor || block.source_missing} onClick={() => setEditor({ target: block.id, text: block.text, version: draft.version })}>Edit wording</button></>}
            {block.table && !(view?.rfp && scopeTable(block) && editor?.target !== block.id) && <div className="report-table-scroll"><table className="report-table">{block.table.display !== "document" && <caption>Saved source comparison{block.edited ? " — independent of edited wording" : ""}</caption>}<thead><tr>{block.table.columns.map((column,i) => <th key={i} scope="col">{column}</th>)}</tr></thead><tbody>{block.table.rows.map((row,i) => <tr key={i}>{row.map((cell,j) => <td key={j}>{cell}</td>)}</tr>)}</tbody></table></div>}
            <BlockReferences block={block} references={draft.references} hideCalculations={view?.report.kind === "rfp"} />
            {block.edited && !block.source_missing && block.generated_text === undefined && <p>Refresh this draft before restoring its saved source wording.</p>}{block.edited && (block.source_missing || block.generated_text !== undefined) && <button type="button" className="btn btn-small" disabled={busy || !!editor} onClick={() => setResetTarget({ id: block.id, missing: block.source_missing, version: draft.version })}>{block.source_missing ? "Remove orphaned wording" : "Restore saved source wording"}</button>}
          </article>;
  // The issue form: in a toggled notice for most reports, always open in an
  // RFP's closing Issue section, where it waits for the blocking checks.
  const issueForm = (version: number, blocked = false) => draft && view && <form className={view.rfp ? "report-issue-form rfp-issue-form" : "report-notice report-issue-form"} onSubmit={e => { e.preventDefault(); void issue(version); }}>
    {!view.rfp && <h3>Issue saved report</h3>}
    <p>Issuing freezes the wording, sources and PDF. Later changes belong to a new draft.</p>
    {!view.rfp && <><label htmlFor="issue-date">Reporting date</label><input id="issue-date" type="date" required value={issueDate} onChange={e => setIssueDate(e.target.value)} disabled={busy} /></>}
    {view.report.kind !== "pmp" && <label><input type="checkbox" checked={discloseBudget()} onChange={e => setDiscloseBudget(e.target.checked)} disabled={busy} /> Include internal budget amounts in this external report</label>}
    {!!view.stale.length && <><label><input type="checkbox" checked={acceptStale} onChange={e => setAcceptStale(e.target.checked)} disabled={busy} /> Issue the saved state despite changed or incomplete sources</label>{acceptStale && <><label htmlFor="issue-reason">Reason for using saved state</label><textarea id="issue-reason" required maxLength={1000} value={staleReason} onChange={e => setStaleReason(e.target.value)} disabled={busy} /></>}</>}
    {version !== draft.version && <p role="alert">The draft changed while this issue form was open. Cancel and reopen it to review the current version.</p>}
    <button className="btn" disabled={busy || blocked || version !== draft.version || (!!view.stale.length && (!acceptStale || !staleReason.trim()))}>{busy ? <Pending delayed>Issuing…</Pending> : view.rfp ? "Issue report" : "Freeze and issue PDF"}</button>
  </form>;
  return <section className="reports report-view" aria-label="Report" aria-busy={busy}>
    <div className="report-reader" ref={reader}>
      <button type="button" className="btn btn-small report-back" disabled={!!editor} title={editor ? "Save or cancel your wording first" : undefined} onClick={onBack}>← Procurement</button>
      {error && <div className="banner" role="alert"><p>{error}</p>{editConflict && editor ? <button type="button" className="btn btn-small" disabled={busy} onClick={() => void reviewLatest()}>Review latest draft</button> : <button type="button" className="btn btn-small" disabled={busy || !!editor} onClick={() => { setError(""); setReload(n => n + 1); }}>Retry loading</button>}</div>}
      {latest && editor && <section className="report-notice" aria-label="Review concurrent edit"><h3>Latest saved wording</h3><p className="report-wording">{latestBlock?.text ?? "This source is no longer in the saved draft. Your unsaved text remains in the editor below."}</p><p>Compare this with your unsaved wording below before choosing which to keep.</p>{latestBlock && !latestBlock.source_missing && latest.draft && <button type="button" className="btn" disabled={busy} onClick={() => { setView(latest); setEditor({ ...editor, version: latest.draft!.version }); setLatest(null); setEditConflict(false); setError(""); }}>Keep my wording against this version</button>}</section>}
      {!view ? !error && <p role="status"><Pending delayed>Opening report…</Pending></p> : <>
        <header className="report-toolbar"><div><h2>{view.report.title}</h2><p className="report-muted">Draft — not for issue · {draft ? `Version ${draft.version} · ${draft.reporting_date}` : "Not yet assembled"}</p>{view.page_fit && <PageFitNote fit={view.page_fit} />}</div><div className="report-refresh">{!!view.stale.length && draft && <span className="report-stale" role="status" title="Refresh keeps protected wording and flags changed sources.">{view.stale.map((reason, i) => <span key={reason}>{i > 0 && " · "}<span>{staleText(reason)}</span></span>)}</span>}{view.rfp && draft && <a className="btn" href={submissionApi.responseFormURL(view.report.id)} download title="An Excel form with a row for every fee item and a table for each returnable schedule. Proposals returned on it are read row by row.">Response form</a>}<button type="button" className="btn" disabled={busy || !!editor || view.stale.includes("package_retired")} onClick={() => void refresh()}>{busy ? <Pending delayed>Saving…</Pending> : draft ? "Refresh draft" : "Assemble draft"}</button></div></header>
        {view.rfp && <>
          {draft?.sections.find(s => s.id === "issue_dates")?.blocks.filter(b => b.edited || b.source_missing).map(blockView)}
          <RfpReleaseDates controls={view.rfp} busyKey={busy || !!busyKey || !!editor ? "busy" : ""} onWrite={writeSetting} />
          <SectionMarkers controls={view.rfp} section="issue_dates" busyKey={busyKey} onWrite={writeSetting} onJump={jump} />
        </>}
        {view.rfp_suggestion && <div className="report-notice" role="status"><strong>This package looks like {view.rfp_suggestion.title.toLowerCase()}</strong><p>Use the {view.rfp_suggestion.title.toLowerCase()} RFP, built from the reviewed golden set, instead of the generic one? This records the package's discipline; the package title alone never changes the scope.</p><div className="report-actions"><button type="button" className="btn btn-small" disabled={busy || !!editor} onClick={() => void confirmDiscipline()}>Use the {view.rfp_suggestion.title.toLowerCase()} RFP</button></div></div>}
        {!!view.issues?.length && <section className="report-notice" aria-label="Issued reports"><h3>Issued reports</h3><ul>{view.issues.map(item => <li key={item.id}><a href={`/api/reports/${reportId}/versions/${item.id}/file`}>Download issue {item.number} — {item.reporting_date}</a></li>)}</ul><p>Issued PDFs retain their saved sources and wording.</p></section>}
        {!!view.changes_since_issue?.length && <details className="report-notice"><summary>Changes since last issue ({view.changes_since_issue.length})</summary><ul>{view.changes_since_issue.map(change => <li key={change.target_id}><strong>{change.kind}</strong>: {change.after || change.before}</li>)}</ul></details>}
        {draft && <div>
          <div className="rfp-draft">
          {!view.rfp && <div className="report-actions"><button type="button" className="btn" disabled={busy || !!editor} onClick={() => { setIssuing(!issuing); setIssueVersion(draft.version); if (!issuing) { setAcceptStale(false); setStaleReason(""); setDiscloseBudget(false); } }}>{issuing ? "Cancel issue" : "Issue report"}</button></div>}
          {!view.rfp && issuing && issueForm(issueVersion)}
          {!!draft.material_assumptions?.length && <section className="report-notice" aria-label="Material assumptions"><h3>Material assumptions</h3>{draft.material_assumptions.map(ref => <Citation key={ref.citation_id} reference={ref} />)}</section>}
          {!view.rfp && draft.sections.filter(section => section.internal && section.blocks?.length).map(section => <details className="report-notice report-review" key={section.id}><summary><strong>{section.title} ({section.blocks.length})</strong> — kept out of the issued report</summary><ul>{section.blocks.map(block => <li key={block.id}><strong>{block.label}:</strong> {block.text}</li>)}</ul></details>)}
          {draft.sections.filter(section => !section.internal && section.id !== "issue_dates").map(section => view.rfp && section.id === "project"
            ? <section className="report-section" key={section.id}>
              <h3 id={`rs-${section.id}`}>{section.title}</h3>
              <SectionMarkers controls={view.rfp} section="project" busyKey={busyKey} onWrite={writeSetting} onJump={jump} />
              <ProjectPage blocks={section.blocks ?? []} controls={view.rfp} busyKey={busyKey} onWrite={writeSetting} onJump={jump} renderBlock={blockView} />
            </section>
            : <section className="report-section" key={section.id}>
            {view.rfp && editable.has(section.id) ? <div className="report-section-head"><h3 id={`rs-${section.id}`}>{section.title}</h3><button type="button" className="btn btn-small" aria-expanded={openEdits.has(section.id)} aria-controls={`rfp-edit-${section.id}`} aria-label={`${openEdits.has(section.id) ? "Done editing" : "Edit"} ${section.title}`} disabled={!!editor} onClick={() => toggleEdit(section.id)}>{openEdits.has(section.id) ? "Done" : "Edit"}</button></div> : <h3>{section.title}</h3>}
            {view.rfp && <SectionMarkers controls={view.rfp} section={section.id} busyKey={busyKey} onWrite={writeSetting} onJump={jump} />}
            {view.rfp && editable.has(section.id) && openEdits.has(section.id) && <div id={`rfp-edit-${section.id}`}><RfpSectionEditor controls={view.rfp} section={section.id} busyKey={busyKey} onWrite={writeSetting} /></div>}
            {(section.blocks ?? []).map(blockView)}</section>)}
          {view.rfp && <RfpIssue controls={view.rfp} busyKey={busyKey} onWrite={writeSetting} onJump={jump}>{issueForm(draft.version, view.rfp.decisions.length > 0)}</RfpIssue>}
          </div>
        </div>}
      </>}
    </div>
	{resetTarget && <ConfirmDialog title={resetTarget.missing ? "Remove orphaned wording?" : "Restore saved source wording?"} message={resetTarget.missing ? "This source no longer appears in the draft. Removing its protected wording cannot be undone. Copy any text you need before continuing." : "Your protected wording will be removed and replaced with the source wording saved in this draft. Copy any edits you need before continuing."} confirmLabel={resetTarget.missing ? "Remove wording" : "Restore wording"} busy={busy} busyLabel="Saving…" onConfirm={() => void resetEdit()} onCancel={() => setResetTarget(null)} />}
  </section>;
}

function PageFitNote({ fit }: { fit: PageFit }) {
  if (fit.error) return <p className="report-page-fit report-page-fit-over">Page count unavailable: {fit.error}</p>;
  const over = !fit.fits;
  return <p className={over ? "report-page-fit report-page-fit-over" : "report-page-fit"}>Issued length: {fit.pages} {fit.pages === 1 ? "page" : "pages"} of {fit.limit} allowed (target {fit.target === 1.5 ? "1½" : fit.target}){over ? " — shorten before issue" : ""}</p>;
}
function BlockReferences({ block, references, hideCalculations }: { block: ReportBlock; references: ReportReference[]; hideCalculations: boolean }) {
  // Keep calculation provenance in the saved report for future auditing;
  // its internal metadata does not belong in the everyday RFP view.
  const visible = (block.citation_ids ?? []).map(id => ({ id, ref: (references ?? []).find(r => r.citation_id === id) }))
    .filter(({ ref }) => !hideCalculations || ref?.label !== "C");
  if (!visible.length) return null;
  return <div className="report-references">{visible.map(({ id, ref }) => ref ? <Citation key={id} reference={ref} /> : <span key={id}>Source unavailable: {id}</span>)}</div>;
}
function Citation({ reference }: { reference: ReportReference }) {
  return <details className="report-citation" data-label={reference.label}><summary><span>{reference.citation_id}</span> {referenceNames[reference.label]}</summary><p>{reference.basis.text}</p></details>;
}
