import { Fragment, useCallback, useEffect, useState } from "react";
import { ApiError } from "./api";
import { proposalApi, type ProposalRecord, type ProposalAssignment } from "./proposalApi";
import { packageApi, type PackageRecord } from "./packageApi";
import { workApi, type WorkRecord } from "./workApi";
import type { Profile } from "./profileApi";
import { WorkSources } from "./WorkItem";
import { ConfirmDialog } from "./ConfirmDialog";
import "./reports.css";
import "./implications.css";

// Users see "implications"; the API and database keep the name "proposal".
const words = (s: string) => s.replaceAll("_", " ");
const kindLabel: Record<string, string> = { work_item: "Work", investigation: "Investigation", discipline: "Consultant", obligation: "Obligation", approval: "Approval", hold_point: "Hold point" };
const stateLabel: Record<string, string> = { open: "Open", addressed_by_evidence: "Addressed by evidence", reopened: "Reopened", accepted: "Accepted", dismissed: "Dismissed" };
const decided = (p: ProposalRecord) => p.state === "accepted" || p.state === "dismissed";
const emptyAssignment = (): ProposalAssignment => ({ package_id: "", work_item_id: "", stage_id: "", role: "" });
type Review = { base: ProposalRecord; mode: "accept" | "dismiss"; assignment: ProposalAssignment; rationale: string };
type Group = { id: string; label: string; rows: ProposalRecord[] };

const signalName = (id: string) => { const s = words(id.replace(/^sig\./, "").replaceAll("-", " ")); return s.charAt(0).toUpperCase() + s.slice(1); };
function signalState(state: string) {
  if (state === "true" || state === "stated_true") return "stated in the documents";
  if (state === "false" || state === "stated_false") return "documents say it is not present";
  if (state === "unknown" || state === "stated_unknown") return "not stated in the documents";
  if (state === "set") return "recorded";
  return words(state);
}

// The system an implication belongs to: its own target, else the work that raised it.
const systemOf = (p: ProposalRecord) => p.target_system_id || p.reason.triggers?.[0]?.system || "";
const partOf = (p: ProposalRecord) => p.target_part_id || p.reason.triggers?.[0]?.part || "";

// Rows follow the profile's system groups and order, the same list Profile uses,
// so the page reads in building order and follows any change to that order.
function groupImplications(items: ProposalRecord[], profile: Profile | null): Group[] {
  const groups: Group[] = (profile?.systems ?? []).map(g => ({ id: g.id, label: g.label, rows: [] }));
  const where = new Map<string, [number, number]>();
  profile?.systems.forEach((g, gi) => g.rows.forEach((r, ri) => { if (!where.has(r.leaf)) where.set(r.leaf, [gi, ri]); }));
  const project: Group = { id: "project", label: "Whole project", rows: [] };
  const other: Group = { id: "other", label: "Other", rows: [] };
  const locate = (system: string): [number, number] | undefined => {
    const at = where.get(system);
    if (at) return at;
    const gi = groups.findIndex(g => system.split(".")[0] === g.id);
    return gi < 0 ? undefined : [gi, Number.MAX_SAFE_INTEGER];
  };
  for (const p of items) {
    const system = systemOf(p);
    const at = system ? locate(system) : undefined;
    (!system ? project : at ? groups[at[0]] : other).rows.push(p);
  }
  // Stable sort keeps the server's ranking within one system.
  const rank = (p: ProposalRecord) => locate(systemOf(p))?.[1] ?? Number.MAX_SAFE_INTEGER;
  for (const g of groups) g.rows.sort((a, b) => rank(a) - rank(b));
  return [project, ...groups, other].filter(g => g.rows.length);
}

export function Implications({ projectId, tick, onSignedOut }: { projectId: string; tick: number; onSignedOut: () => void }) {
  const [items, setItems] = useState<ProposalRecord[]>([]);
  const [packages, setPackages] = useState<PackageRecord[]>([]);
  const [works, setWorks] = useState<WorkRecord[]>([]);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [showDecided, setShowDecided] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [loadError, setLoadError] = useState("");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [open, setOpen] = useState<Set<string>>(new Set());
  const [review, setReview] = useState<Review | null>(null);
  const [undo, setUndo] = useState<ProposalRecord | null>(null);
  const [busy, setBusy] = useState(false);
  const [reload, setReload] = useState(0);
  const [recent, setRecent] = useState("");
  const signedOut = useCallback((e: unknown) => {
    if (e instanceof ApiError && e.status === 401) { onSignedOut(); return true; }
    return false;
  }, [onSignedOut]);
  useEffect(() => {
    let stopped = false;
    Promise.all([proposalApi.list(projectId, true), packageApi.list(projectId), workApi.list(projectId), workApi.profile(projectId)]).then(([result, pkgs, work, prof]) => {
      if (!stopped) { setItems(result.items); setPackages(pkgs.items); setWorks(work.items); setProfile(prof); setLoaded(true); setLoadError(""); }
    }).catch(e => { if (!stopped && !signedOut(e)) setLoadError("Could not load implications. Reload to try again."); });
    return () => { stopped = true; };
  }, [projectId, tick, reload, signedOut]);

  const partLabel = (id: string) => id ? profile?.parts.find(p => p.id === id)?.label ?? "Location unavailable" : "Whole project";
  // Several implications share a title and differ only in the system they reach.
  const systemLabel = (id: string) => profile?.systems.flatMap(g => g.rows).find(r => r.leaf === id)?.label ?? "";
  const workTitle = (id: string, system: string) => works.find(w => w.id === id)?.title || system;
  const canAccept = (p: ProposalRecord) => ["discipline", "obligation", "approval", "hold_point"].includes(p.kind) || ["work_item", "investigation"].includes(p.kind) && !!p.target_system_id && !!p.target_part_id;
  const current = review ? items.find(p => p.key === review.base.key) : undefined;
  const stale = !!review && (!current || current.inputs_fingerprint !== review.base.inputs_fingerprint || current.decision?.version !== review.base.decision?.version);
  const pkg = packages.find(p => p.id === review?.assignment.package_id);
  const work = works.find(w => w.id === review?.assignment.work_item_id);
  const needsPackage = review?.base.kind === "obligation";
  const delivery = review?.base.kind === "approval" || review?.base.kind === "hold_point";
  const roles = pkg?.kind === "supply" ? ["supply"] : work?.action === "retain" ? ["maintain_operation", "protect"] : ["design", "document", "supply", "install", "test", "certify", "inspect", "maintain_operation", "protect"];
  const needsAction = review?.base.kind === "work_item" && !review.base.action;
  const invalidAssignment = !!review && (needsAction && !review.assignment.action || needsPackage && !pkg || !!review.assignment.package_id && !pkg || !!review.assignment.work_item_id && !work || !!review.assignment.stage_id && !pkg?.stages.some(s => s.id === review.assignment.stage_id) || !!review.assignment.role && !roles.includes(review.assignment.role) || needsPackage && work?.action === "retain" && (pkg?.kind !== "works" || !["maintain_operation", "protect"].includes(review.assignment.role)));

  const decidedCount = items.filter(decided).length;
  // A row just decided, or being decided or undone, stays visible so its
  // outcome and Undo remain in place.
  const visible = items.filter(p => showDecided || !decided(p) || p.key === recent || p.key === review?.base.key || p.key === undo?.key);
  const toggle = (key: string) => setOpen(prev => { const next = new Set(prev); if (next.has(key)) next.delete(key); else next.add(key); return next; });
  function begin(p: ProposalRecord, mode: Review["mode"]) {
    setReview({ base: p, mode, assignment: emptyAssignment(), rationale: "" }); setError(""); setMessage("");
    setOpen(prev => new Set(prev).add(p.key));
  }
  async function decide() {
    if (!review || stale || loadError) return;
    setBusy(true); setError("");
    try {
      if (review.mode === "dismiss") await proposalApi.dismiss(projectId, review.base, review.rationale);
      else await proposalApi.accept(projectId, review.base, needsPackage || delivery || needsAction ? review.assignment : {});
      setMessage(review.mode === "accept" ? `Accepted for planning: ${review.base.label}. The created record is in its project view.` : `Dismissed: ${review.base.label}.`);
      setRecent(review.base.key); setReview(null); setReload(n => n + 1);
    } catch (e) {
      if (!signedOut(e)) setError(e instanceof ApiError && e.status === 409 ? "The implication or assignment changed. Reload and review before trying again." : e instanceof ApiError && e.status === 422 ? "This implication cannot be accepted with these choices. Check its target and assignment." : "Could not save the decision. Your choices are still here; reload and try again.");
    } finally { setBusy(false); }
  }
  async function undoDecision() {
    if (!undo?.decision) return;
    setBusy(true); setError("");
    try { await proposalApi.undo(projectId, undo.key, undo.decision.version); setUndo(null); setMessage("Decision undone."); setReload(n => n + 1); }
    catch (e) { if (!signedOut(e)) { setUndo(null); setError(e instanceof ApiError && e.status === 409 ? "Undo was refused: the created record has been edited or used, or the decision changed. Reload to see the saved state." : "Could not undo the decision. Reload and try again."); } }
    finally { setBusy(false); }
  }

  function decisionForm(r: Review) {
    return <section className="implication-decision" aria-label="Implication decision"><h3>{r.mode === "accept" ? "Accept for planning" : "Dismiss implication"}</h3>
      {stale && <div className="report-notice"><strong>Saved implication changed</strong><p>Your choices remain below. Cancel, check the reason again and begin again.</p></div>}
      <form className="package-scope-form" onSubmit={e => { e.preventDefault(); void decide(); }}><fieldset disabled={busy}>
        {r.mode === "dismiss" ? <><label htmlFor="implication-rationale">Reason for dismissal (optional)</label><textarea id="implication-rationale" rows={2} maxLength={200} value={r.rationale} onChange={e => setReview({ ...r, rationale: e.target.value })} /></> : <>
          <p>{r.base.kind === "discipline" ? "Creates a planned services package. It does not appoint a consultant." : needsPackage ? "Adds this obligation to the package you choose." : delivery ? "Creates an uncompleted delivery requirement. It does not grant approval or release a hold point." : "Creates an included work item accepted for planning. It does not verify compliance."}</p>
          {needsAction && <><label htmlFor="implication-action">Action for the new work item</label><select id="implication-action" required value={r.assignment.action ?? ""} onChange={e => setReview({ ...r, assignment: { ...r.assignment, action: e.target.value } })}><option value="">Choose the intended work</option>{["new", "replace", "upgrade", "alter", "repair", "remove", "retain"].map(action => <option key={action} value={action}>{words(action)}</option>)}</select></>}
          {(needsPackage || delivery) && <>
            <label htmlFor="implication-package">Package{needsPackage ? " (required)" : " (optional)"}</label><select id="implication-package" required={needsPackage} value={r.assignment.package_id} onChange={e => setReview({ ...r, assignment: { ...r.assignment, package_id: e.target.value, stage_id: "", role: "" } })}><option value="">{needsPackage ? "Choose a package" : "No package assignment"}</option>{packages.map(p => <option key={p.id} value={p.id}>{p.title} · {p.kind}</option>)}</select>
            <label htmlFor="implication-work">Work item (optional)</label><select id="implication-work" value={r.assignment.work_item_id} onChange={e => setReview({ ...r, assignment: { ...r.assignment, work_item_id: e.target.value, role: "" } })}><option value="">No work assignment</option>{works.filter(w => w.inclusion === "included").map(w => <option key={w.id} value={w.id}>{w.title} · {w.action}</option>)}</select>
            {needsPackage && <><label htmlFor="implication-role">Responsibility role{work?.action === "retain" ? " (required for retained work)" : " (optional)"}</label><select id="implication-role" value={r.assignment.role} onChange={e => setReview({ ...r, assignment: { ...r.assignment, role: e.target.value } })}><option value="">No role assigned</option>{roles.map(x => <option key={x} value={x}>{words(x)}</option>)}</select>{work?.action === "retain" && <p>Retained work needs a works package and a maintain operation or protect role.</p>}</>}
            {pkg && <><label htmlFor="implication-stage">Stage (optional)</label><select id="implication-stage" value={r.assignment.stage_id} onChange={e => setReview({ ...r, assignment: { ...r.assignment, stage_id: e.target.value } })}><option value="">No stage assignment</option>{pkg.stages.map(s => <option key={s.id} value={s.id}>{s.label}{s.novation_phase !== "none" ? ` · ${words(s.novation_phase)}` : ""}</option>)}</select></>}
          </>}
        </>}
        <div className="report-actions"><button className="btn" disabled={busy || stale || !!loadError || r.mode === "accept" && invalidAssignment}>{busy ? "Saving…" : r.mode === "accept" ? "Accept for planning" : "Save dismissal"}</button><button type="button" className="btn" disabled={busy} onClick={() => { setReview(null); setError(""); }}>Cancel</button></div>
      </fieldset></form>
    </section>;
  }

  function detail(p: ProposalRecord) {
    const triggers = p.reason.triggers ?? [];
    return <div className="implication-detail">
      {p.draft && <p className="implication-note">Draft knowledge: accepting keeps this provisional and does not verify it.</p>}
      {p.unaccepted_triggers && <p className="implication-note">Some of the work that raised this is not yet accepted.</p>}
      {p.inputs_changed && <p className="implication-note">Inputs changed since the saved decision; the previous decision is kept.</p>}
      <p><strong>Why:</strong> {triggers.length ? <>raised by {triggers.map((t, i) => <Fragment key={t.work_item_id}>{i > 0 && "; "}{workTitle(t.work_item_id, t.system)} ({words(t.action)}, {partLabel(t.part)})</Fragment>)}.</> : "raised by the project profile."}{p.action && <> Proposed action: {words(p.action)}.</>}</p>
      {!!p.reason.determinants?.length && <p><strong>Values used:</strong> {p.reason.determinants.map(d => `${d.key} ${d.value} (${words(d.origin)})`).join("; ")}</p>}
      {!!p.reason.signals?.length && <ul className="implication-signals">{p.reason.signals.map(s => <li key={s.id}>{signalName(s.id)}: {signalState(s.state)}{Array.isArray(s.sources) && s.sources.length > 0 && <details><summary>Evidence</summary><WorkSources sources={s.sources} /></details>}</li>)}</ul>}
      {triggers.map(t => <details key={t.work_item_id}><summary>Work evidence: {workTitle(t.work_item_id, t.system)}</summary><WorkSources sources={works.find(w => w.id === t.work_item_id)?.provenance.sources} /></details>)}
      <p className="implication-ref">Knowledge record {p.reason.record}{p.reason.interface ? ` · interface ${p.reason.interface}` : ""}{p.reason.rules?.length ? ` · rule references (not verified here): ${p.reason.rules.join(", ")}` : ""}</p>
      {p.decision && <section className="implication-decision" aria-label="Saved decision"><p><strong>{stateLabel[p.decision.decision] ?? words(p.decision.decision)}</strong> by {p.decision.actor}, {new Date(p.decision.decided_at).toLocaleString()}{p.decision.created_record_type ? `. Created ${words(p.decision.created_record_type)} for planning.` : "."}</p>{p.decision.rationale && <p className="report-wording">{p.decision.rationale}</p>}</section>}
      {!canAccept(p) && !p.decision && <p className="implication-note">This does not yet name a complete target, so it cannot be accepted.</p>}
      {review?.base.key === p.key && decisionForm(review)}
    </div>;
  }

  function actions(p: ProposalRecord) {
    const locked = busy || !!loadError || !!review;
    if (p.decision) return <button type="button" className="btn btn-small" disabled={locked} aria-label={`Undo decision on ${p.label}`} onClick={() => setUndo(p)}>Undo</button>;
    return <><button type="button" className="btn btn-small" disabled={locked || !canAccept(p)} aria-label={`Accept ${p.label}`} onClick={() => begin(p, "accept")}>Accept</button><button type="button" className="btn btn-small" disabled={locked} aria-label={`Dismiss ${p.label}`} onClick={() => begin(p, "dismiss")}>Dismiss</button></>;
  }

  return <section className="reports implications" aria-label="Implications" aria-busy={busy}>
    <header className="implications-head"><div><h1>Implications</h1><p className="report-muted">What the works raise: investigations, consultants, obligations, approvals and hold points, in building order. Accepting adds the item to planning.</p></div>
      <div className="report-actions">{decidedCount > 0 && <button type="button" className="btn btn-small" aria-pressed={showDecided} disabled={busy} onClick={() => setShowDecided(!showDecided)}>{showDecided ? "Hide decided" : `Show decided (${decidedCount})`}</button>}<button type="button" className="btn btn-small" disabled={busy} onClick={() => setReload(n => n + 1)}>Reload implications</button></div></header>
    {(loadError || error) && <div className="banner" role="alert"><p>{loadError || error}</p></div>}
    {message && <p role="status">{message}</p>}
    {!loaded ? <p role="status">{loadError ? "Implications unavailable." : "Loading implications…"}</p>
      : !items.length ? <p>No implications yet. This does not establish that all obligations have been identified.</p>
      : !visible.length ? <p>No open implications. {decidedCount} decided.</p>
      : <table className="implications-table">
        <colgroup><col /><col className="col-kind" /><col className="col-where" /><col className="col-status" /><col className="col-actions" /></colgroup>
        <thead><tr><th scope="col">Implication</th><th scope="col">Kind</th><th scope="col">Location</th><th scope="col">Status</th><th scope="col"><span className="implications-sr">Decision</span></th></tr></thead>
        {groupImplications(visible, profile).map(g => <tbody key={g.id} aria-label={g.label}>
          <tr className="implications-group"><th scope="rowgroup" colSpan={5}>{g.label}</th></tr>
          {g.rows.map(p => { const expanded = open.has(p.key); return <Fragment key={p.key}>
            <tr className={`implication-row${decided(p) ? " is-decided" : ""}`} data-key={p.key}>
              <td><button type="button" className="implication-toggle" aria-expanded={expanded} onClick={() => toggle(p.key)}><span aria-hidden="true">{expanded ? "▾" : "▸"}</span>{p.label}</button>{systemLabel(systemOf(p)) && <span className="implication-system">{systemLabel(systemOf(p))}</span>}</td>
              <td>{kindLabel[p.kind] ?? words(p.kind)}</td>
              <td>{partLabel(partOf(p))}</td>
              <td>{p.critical && <span className="implication-badge is-critical">Critical</span>}{stateLabel[p.state] ?? words(p.state)}{p.draft && <span className="implication-badge">Draft</span>}</td>
              <td className="implication-actions"><div>{actions(p)}</div></td>
            </tr>
            {expanded && <tr className="implication-expanded"><td colSpan={5}>{detail(p)}</td></tr>}
          </Fragment>; })}
        </tbody>)}
      </table>}
    {undo && <ConfirmDialog title="Undo this decision?" message={undo.decision?.decision === "accepted" ? "This retires the created record only if it is untouched and unreferenced. Undo is refused if someone edited or used it." : "This reopens the implication for review. The previous decision remains in its history."} confirmLabel="Undo decision" busyLabel="Undoing…" busy={busy} onConfirm={() => void undoDecision()} onCancel={() => setUndo(null)} />}
  </section>;
}
