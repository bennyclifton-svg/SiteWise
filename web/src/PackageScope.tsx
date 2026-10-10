// One package's scope editor, opened from a Procurement row's menu. Existing
// head contracts can also accept newly uncovered work here.
import { useCallback, useEffect, useState } from "react";
import { ApiError } from "./api";
import { packageApi, type PackageRecord, type ScopeRecord, type ScopeWork, type ScopeInput, type ProposedWorks } from "./packageApi";
import "./reports.css";

const roles = ["design", "document", "supply", "install", "test", "certify", "inspect", "maintain_operation", "protect"];
const words = (value: string) => value.replaceAll("_", " ");
const stageLabel = (stage: PackageRecord["stages"][number]) => `${stage.label}${stage.novation_phase === "none" ? "" : ` · ${stage.novation_phase === "pre" ? "before" : "after"} novation`}`;
const blankScope = (): ScopeInput => ({ item_kind: "obligation", user_text: "", inclusion: "included", deliverable: "", interface_ids: [] });

export function PackageScope({ projectId, pkg, tick, onSignedOut }: { projectId: string; pkg: PackageRecord; tick: number; onSignedOut: () => void }) {
  const [works, setWorks] = useState<ScopeWork[]>([]);
  const [scope, setScope] = useState<ScopeRecord[]>([]);
  const [input, setInput] = useState<ScopeInput>(blankScope);
  const [editing, setEditing] = useState<ScopeRecord | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [reload, setReload] = useState(0);
  const [proposal, setProposal] = useState<ProposedWorks | null>(null);
  const fail = useCallback((e: unknown) => {
    if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
    setError(e instanceof ApiError && e.status === 409 ? "This assignment changed or already exists. Your wording has been kept; review the saved scope before retrying." : e instanceof ApiError && e.status === 422 ? "This assignment is not valid for the selected package, work item or stage. Review those choices." : "Could not load or save package scope. Try again.");
  }, [onSignedOut]);
  useEffect(() => {
    let stopped = false;
    // Scope, work labels and fresh proposals load in parallel.
    Promise.all([packageApi.works(projectId), packageApi.scope(projectId, pkg.id), packageApi.list(projectId)]).then(([w, s, overview]) => { if (!stopped) { setWorks(w.items.filter(item => !item.is_group)); setScope(s.items); setProposal(overview.proposed_works?.package_id === pkg.id ? overview.proposed_works : null); } }).catch(e => { if (!stopped) fail(e); });
    return () => { stopped = true; };
  }, [projectId, pkg.id, tick, reload, fail]);
  const latestScope = editing && scope.find(row => row.id === editing.id);
  async function acceptScope() {
    if (!proposal) return;
    setBusy(true); setError("");
    try { await packageApi.accept(projectId, proposal.key); setReload(n => n + 1); }
    catch (e) { fail(e); } finally { setBusy(false); }
  }
  async function saveScope() {
    setBusy(true); setError("");
    try {
      if (editing) await packageApi.patchScope(projectId, pkg.id, editing.id, editing.version, { user_text: input.user_text, deliverable: input.deliverable, inclusion: input.inclusion });
      else await packageApi.addScope(projectId, pkg.id, input);
      setEditing(null); setInput(blankScope()); setReload(n => n + 1);
    } catch (e) { fail(e); } finally { setBusy(false); }
  }
  return <div className="report-reader" aria-busy={busy}>
    {error && <div className="banner" role="alert"><p>{error}</p><button type="button" className="btn btn-small" onClick={() => setReload(n => n + 1)}>Reload saved scope</button></div>}
    <header className="report-toolbar"><div><h2>{pkg.title}</h2><p className="report-muted">{words(pkg.kind)} · {words(pkg.lifecycle_status)}. Scope is accepted for planning, not a verified appointment.</p></div></header>
    {proposal && <section className="report-section" aria-label="Suggested scope"><h3>Suggested scope</h3><ul>{proposal.lines.map(line => <li key={`${line.work_item_id}:${line.role}`}>{line.text}</li>)}</ul><button type="button" className="btn btn-small" disabled={busy} onClick={() => void acceptScope()}>Accept suggested scope</button></section>}
    <section className="report-section"><h3>Assigned scope</h3>{!scope.length ? <p>No scope assigned yet.</p> : scope.map(row => <article key={row.id} className="report-block"><h4>{words(row.item_kind)} · {row.inclusion}</h4><p className="report-wording">{row.user_text || `Catalogue clause ${row.clause_id}, version ${row.clause_version}`}</p>{row.provisional && <p>Provisional wording</p>}{row.work_item_id && <p>Work: {works.find(work => work.id === row.work_item_id)?.title ?? "Work item unavailable"} · {words(row.role ?? "")}</p>}{row.stage_id && <p>Stage: {pkg.stages.filter(stage => stage.id === row.stage_id).map(stageLabel)[0] ?? "Stage unavailable"}</p>}{row.deliverable && <p>Deliverable: {row.deliverable}</p>}{row.user_text && <button type="button" className="btn btn-small" disabled={busy || !!editing || !!input.user_text} onClick={() => { setEditing(row); setInput({ ...blankScope(), item_kind: row.item_kind, user_text: row.user_text!, deliverable: row.deliverable ?? "", inclusion: row.inclusion }); }}>Edit scope wording</button>}</article>)}</section>
    <section className="report-section"><h3>{editing ? "Edit scope wording" : "Add scope"}</h3><form className="package-scope-form" onSubmit={e => { e.preventDefault(); void saveScope(); }}><fieldset disabled={busy}>
      {!editing && <><label htmlFor="scope-kind">Scope type</label><select id="scope-kind" value={input.item_kind} onChange={e => setInput({ ...input, item_kind: e.target.value, work_item_id: undefined, role: undefined })}><option value="obligation">Obligation</option><option value="responsibility">Work responsibility</option></select>
      {input.item_kind === "responsibility" && <><label htmlFor="scope-work">Work item</label><select id="scope-work" required value={input.work_item_id ?? ""} onChange={e => setInput({ ...input, work_item_id: e.target.value })}><option value="">Choose a work item</option>{works.map(work => <option key={work.id} value={work.id}>{work.title} · {work.action} · {work.inclusion}</option>)}</select><label htmlFor="scope-role">Responsibility</label><select id="scope-role" required value={input.role ?? ""} onChange={e => setInput({ ...input, role: e.target.value })}><option value="">Choose a responsibility</option>{roles.filter(role => pkg.kind !== "supply" || role === "supply").map(role => <option key={role} value={role}>{words(role)}</option>)}</select></>}
      <label htmlFor="scope-stage">Stage</label><select id="scope-stage" value={input.stage_id ?? ""} onChange={e => setInput({ ...input, stage_id: e.target.value || undefined })}><option value="">No stage specified</option>{pkg.stages.map(stage => <option key={stage.id} value={stage.id}>{stage.label}{stage.novation_phase === "none" ? "" : ` · ${stage.novation_phase === "pre" ? "before" : "after"} novation`}</option>)}</select></>}
      {editing && latestScope && editing.version !== latestScope.version && <div className="report-notice"><strong>Saved scope changed</strong><p>{latestScope.user_text || `Catalogue clause ${latestScope.clause_id}`}</p><p>Compare the latest wording above with your unsaved wording below.</p><button type="button" className="btn btn-small" disabled={busy} onClick={() => setEditing(latestScope)}>Keep my wording against this version</button></div>}
      <label htmlFor="scope-wording">Scope wording</label><textarea id="scope-wording" required rows={4} maxLength={10000} value={input.user_text} onChange={e => setInput({ ...input, user_text: e.target.value })} />
      <label htmlFor="scope-deliverable">Deliverable</label><input id="scope-deliverable" maxLength={2000} value={input.deliverable} onChange={e => setInput({ ...input, deliverable: e.target.value })} />
      <label htmlFor="scope-inclusion">Inclusion</label><select id="scope-inclusion" value={input.inclusion} onChange={e => setInput({ ...input, inclusion: e.target.value })}><option value="included">Included</option><option value="excluded">Excluded</option></select>
      <div className="report-actions"><button className="btn" disabled={busy || !input.user_text.trim()}>{editing ? "Save scope wording" : "Add scope item"}</button><button type="button" className="btn btn-small" disabled={busy} onClick={() => { setEditing(null); setInput(blankScope()); setError(""); }}>Cancel scope edit</button></div>
    </fieldset></form></section>
  </div>;
}
