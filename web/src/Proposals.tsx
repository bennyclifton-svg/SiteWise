import { useCallback, useEffect, useState } from "react";
import { ApiError } from "./api";
import { proposalApi, type ProposalRecord, type ProposalAssignment } from "./proposalApi";
import { packageApi, type PackageRecord } from "./packageApi";
import { workApi, type WorkRecord } from "./workApi";
import { WorkSources } from "./Works";
import { ConfirmDialog } from "./ConfirmDialog";
import "./reports.css";

const words = (s: string) => s.replaceAll("_", " ");
const emptyAssignment = (): ProposalAssignment => ({ package_id: "", work_item_id: "", stage_id: "", role: "" });
type Review = { base: ProposalRecord; mode: "accept" | "dismiss"; assignment: ProposalAssignment; rationale: string };

export function Proposals({ projectId, tick, onSignedOut }: { projectId: string; tick: number; onSignedOut: () => void }) {
  const [items, setItems] = useState<ProposalRecord[]>([]);
  const [packages, setPackages] = useState<PackageRecord[]>([]);
  const [works, setWorks] = useState<WorkRecord[]>([]);
  const [parts, setParts] = useState<{ id: string; label: string }[]>([]);
  const [total, setTotal] = useState(0);
  const [all, setAll] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [loadError, setLoadError] = useState("");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [selected, setSelected] = useState("");
  const [review, setReview] = useState<Review | null>(null);
  const [undo, setUndo] = useState<ProposalRecord | null>(null);
  const [busy, setBusy] = useState(false);
  const [reload, setReload] = useState(0);
  const signedOut = useCallback((e: unknown) => {
    if (e instanceof ApiError && e.status === 401) { onSignedOut(); return true; }
    return false;
  }, [onSignedOut]);
  useEffect(() => {
    let stopped = false;
    Promise.all([proposalApi.list(projectId, all), packageApi.list(projectId), workApi.list(projectId), workApi.profile(projectId)]).then(([result, pkgs, work, profile]) => {
      if (!stopped) { setItems(result.items); setTotal(result.total); setPackages(pkgs.items); setWorks(work.items); setParts(profile.parts); setLoaded(true); setLoadError(""); }
    }).catch(e => { if (!stopped && !signedOut(e)) setLoadError("Could not load saved proposals. Reload to try again."); });
    return () => { stopped = true; };
  }, [projectId, tick, all, reload, signedOut]);
  const item = items.find(p => p.key === selected);
  const canAccept = !!item && (["discipline", "obligation", "approval", "hold_point"].includes(item.kind) || ["work_item", "investigation"].includes(item.kind) && !!item.target_system_id && !!item.target_part_id && (item.kind === "investigation" || !!item.action));
  const stale = !!review && (!item || item.inputs_fingerprint !== review.base.inputs_fingerprint || item.decision?.version !== review.base.decision?.version);
  const pkg = packages.find(p => p.id === review?.assignment.package_id);
  const work = works.find(w => w.id === review?.assignment.work_item_id);
  const needsPackage = review?.base.kind === "obligation";
  const delivery = review?.base.kind === "approval" || review?.base.kind === "hold_point";
  const roles = pkg?.kind === "supply" ? ["supply"] : work?.action === "retain" ? ["maintain_operation", "protect"] : ["design", "document", "supply", "install", "test", "certify", "inspect", "maintain_operation", "protect"];
  const invalidAssignment = !!review && (needsPackage && !pkg || !!review.assignment.package_id && !pkg || !!review.assignment.work_item_id && !work || !!review.assignment.stage_id && !pkg?.stages.some(s => s.id === review.assignment.stage_id) || !!review.assignment.role && !roles.includes(review.assignment.role) || needsPackage && work?.action === "retain" && (pkg?.kind !== "works" || !["maintain_operation", "protect"].includes(review.assignment.role)));
  function begin(mode: Review["mode"]) {
    if (!item) return;
    setReview({ base: item, mode, assignment: emptyAssignment(), rationale: "" }); setError(""); setMessage("");
  }
  async function decide() {
    if (!review || stale || loadError) return;
    setBusy(true); setError("");
    try {
      if (review.mode === "dismiss") await proposalApi.dismiss(projectId, review.base, review.rationale);
      else await proposalApi.accept(projectId, review.base, needsPackage || delivery ? review.assignment : {});
      setMessage(review.mode === "accept" ? "Accepted for planning. The created record is available in its project view." : "Proposal dismissed. Your decision has been saved.");
      setReview(null); setReload(n => n + 1);
    } catch (e) {
      if (!signedOut(e)) setError(e instanceof ApiError && e.status === 409 ? "The proposal or assignment changed. Reload saved proposals and review before trying again." : e instanceof ApiError && e.status === 422 ? "This proposal cannot be accepted with these choices. Check its target and assignment." : "Could not save the decision. Your choices are still here; reload and try again.");
    } finally { setBusy(false); }
  }
  async function undoDecision() {
    if (!undo?.decision) return;
    setBusy(true); setError("");
    try { await proposalApi.undo(projectId, undo.key, undo.decision.version); setUndo(null); setMessage("Decision undone."); setReload(n => n + 1); }
    catch (e) { if (!signedOut(e)) { setUndo(null); setError(e instanceof ApiError && e.status === 409 ? "Undo was refused: the created record has been edited or used, or the decision changed. Reload to see the saved state." : "Could not undo the decision. Reload and try again."); } }
    finally { setBusy(false); }
  }
  return <section className="reports proposal-review" aria-label="Proposals" aria-busy={busy}>
    <div className="report-library"><h1>Proposals</h1><p className="report-muted">Review the reason before adding work, a package, an obligation or a delivery requirement.</p>
      <button type="button" className="btn btn-small" disabled={busy} onClick={() => setReload(n => n + 1)}>Reload saved proposals</button>
      {!loaded ? <p role="status">{loadError ? "Saved proposals unavailable." : "Loading proposals…"}</p> : !items.length ? <p>No saved proposals. This does not establish that all obligations have been identified.</p> : <>
        <p>Showing {items.length} of {total}. Critical proposals stay visible.</p>
        <ul className="report-list">{items.map(p => <li key={p.key}><button type="button" disabled={busy || !!review} aria-current={p.key === selected ? "true" : undefined} onClick={() => { setSelected(p.key); setError(""); setMessage(""); }}>{p.label}<br />{p.critical ? "Critical · " : ""}{words(p.kind)} · {words(p.state)}{p.draft ? " · draft knowledge" : ""}</button></li>)}</ul>
      </>}
      {loaded && (all || total > items.length) && <button type="button" className="btn btn-small" disabled={busy || !!review} onClick={() => setAll(!all)}>{all ? "Show priority proposals" : "Show all proposals"}</button>}
    </div>
    <div className="report-reader">
      {(loadError || error) && <div className="banner" role="alert"><p>{loadError || error}</p></div>}
      {message && <p role="status">{message}</p>}
      {!item && !review ? <p className="report-empty">Choose a proposal to inspect its reason and make a planning decision.</p> : <>
        {item && <><header className="report-toolbar"><div><h2>{item.label}</h2><p>{words(item.kind)} · {words(item.state)}{item.critical ? " · Critical" : ""}</p></div></header>
          {item.draft && <p className="report-notice">Draft knowledge. Acceptance keeps this proposal provisional; it does not approve or verify the knowledge.</p>}
          {item.unaccepted_triggers && <p className="report-notice">Some triggering work is still proposed. Review that work before relying on this proposal.</p>}
          {item.inputs_changed && <p className="report-notice">Inputs changed since the saved decision. Review the current reason below; the previous decision is retained.</p>}
          <section className="report-section" aria-label="Proposal reason"><h3>Why this is proposed</h3>
            <p>Knowledge record: {item.reason.record}{item.reason.interface ? ` · interface ${item.reason.interface}` : ""}</p>
            {item.action && <p>Proposed action: {words(item.action)}</p>}
            <ul>{(item.reason.triggers ?? []).map(t => <li key={t.work_item_id}>{works.find(w => w.id === t.work_item_id)?.title || t.system} · {t.action} · {parts.find(p => p.id === t.part)?.label || "Location unavailable"}<details><summary>Work evidence</summary><WorkSources sources={works.find(w => w.id === t.work_item_id)?.provenance.sources} /></details></li>)}</ul>
            {!!item.reason.determinants?.length && <><h4>Values used</h4><ul>{item.reason.determinants.map((d, i) => <li key={i}>{d.key}: {d.value} ({words(d.origin)})</li>)}</ul></>}
            {!!item.reason.rules?.length && <p>Rule references from the knowledge record (not verified here): {item.reason.rules.join(", ")}</p>}
            {!!item.reason.signals?.length && <details><summary>Recorded signals</summary>{item.reason.signals.map(s => <div key={s.id}><p>{s.id}: {words(s.state)}</p><WorkSources sources={s.sources} /></div>)}</details>}
          </section>
          {item.decision && <section className="report-section" aria-label="Saved decision"><h3>Saved decision</h3><p>{words(item.decision.decision)} · {new Date(item.decision.decided_at).toLocaleString()}</p><p>Recorded by: {item.decision.actor}</p>{item.decision.rationale && <p className="report-wording">{item.decision.rationale}</p>}{item.decision.created_record_type && <p>Created {words(item.decision.created_record_type)} for planning.</p>}
            {!review && <button type="button" className="btn" disabled={busy || !!loadError} onClick={() => setUndo(item)}>Undo decision</button>}
          </section>}
          {!canAccept && !item.decision && <p className="report-notice">This proposal does not yet identify a complete acceptance target. Review its knowledge record before accepting it.</p>}
          {!review && !item.decision && <div className="report-actions"><button type="button" className="btn" disabled={busy || !!loadError || !canAccept} onClick={() => begin("accept")}>Review acceptance</button><button type="button" className="btn" disabled={busy || !!loadError} onClick={() => begin("dismiss")}>Dismiss proposal</button></div>}
        </>}
        {review && <section className="report-section" aria-label="Proposal decision"><h3>{review.mode === "accept" ? "Accept for planning" : "Dismiss proposal"}</h3>
          {stale && <div className="report-notice"><strong>Saved proposal changed</strong><p>Your choices remain below. Cancel this review, inspect the saved reason and begin again.</p></div>}
          <form className="package-scope-form" onSubmit={e => { e.preventDefault(); void decide(); }}><fieldset disabled={busy}>
            {review.mode === "dismiss" ? <><label htmlFor="proposal-rationale">Reason for dismissal (optional)</label><textarea id="proposal-rationale" rows={3} maxLength={200} value={review.rationale} onChange={e => setReview({ ...review, rationale: e.target.value })} /></> : <>
              <p>{review.base.kind === "discipline" ? "Creates a planned services package. It does not appoint a consultant." : needsPackage ? "Adds this obligation to the package you choose." : delivery ? "Creates an uncompleted delivery requirement. It does not grant approval or release a hold point." : "Creates an included work item accepted for planning. It does not verify compliance."}</p>
              {(needsPackage || delivery) && <>
                <label htmlFor="proposal-package">Package{needsPackage ? " (required)" : " (optional)"}</label><select id="proposal-package" required={needsPackage} value={review.assignment.package_id} onChange={e => setReview({ ...review, assignment: { ...review.assignment, package_id: e.target.value, stage_id: "", role: "" } })}><option value="">{needsPackage ? "Choose a package" : "No package assignment"}</option>{packages.map(p => <option key={p.id} value={p.id}>{p.title} · {p.kind}</option>)}</select>
                <label htmlFor="proposal-work">Work item (optional)</label><select id="proposal-work" value={review.assignment.work_item_id} onChange={e => setReview({ ...review, assignment: { ...review.assignment, work_item_id: e.target.value, role: "" } })}><option value="">No work assignment</option>{works.filter(w => w.inclusion === "included").map(w => <option key={w.id} value={w.id}>{w.title} · {w.action}</option>)}</select>
                {needsPackage && <><label htmlFor="proposal-role">Responsibility role{work?.action === "retain" ? " (required for retained work)" : " (optional)"}</label><select id="proposal-role" value={review.assignment.role} onChange={e => setReview({ ...review, assignment: { ...review.assignment, role: e.target.value } })}><option value="">No role assigned</option>{roles.map(r => <option key={r} value={r}>{words(r)}</option>)}</select>{work?.action === "retain" && <p>Retained work needs a works package and a maintain operation or protect role.</p>}</>}
                {pkg && <><label htmlFor="proposal-stage">Stage (optional)</label><select id="proposal-stage" value={review.assignment.stage_id} onChange={e => setReview({ ...review, assignment: { ...review.assignment, stage_id: e.target.value } })}><option value="">No stage assignment</option>{pkg.stages.map(s => <option key={s.id} value={s.id}>{s.label}{s.novation_phase !== "none" ? ` · ${words(s.novation_phase)}` : ""}</option>)}</select></>}
              </>}
            </>}
            <div className="report-actions"><button className="btn" disabled={busy || stale || !!loadError || review.mode === "accept" && invalidAssignment}>{busy ? "Saving…" : review.mode === "accept" ? "Accept for planning" : "Save dismissal"}</button><button type="button" className="btn" disabled={busy} onClick={() => { setReview(null); setError(""); }}>Cancel review</button></div>
          </fieldset></form>
        </section>}
      </>}
    </div>
    {undo && <ConfirmDialog title="Undo proposal decision?" message={undo.decision?.decision === "accepted" ? "This retires the created record only if it is untouched and unreferenced. Undo is refused if someone edited or used it." : "This reopens the proposal for review. The previous decision remains in its history."} confirmLabel="Undo decision" busyLabel="Undoing…" busy={busy} onConfirm={() => void undoDecision()} onCancel={() => setUndo(null)} />}
  </section>;
}
