// Minimum M1 package allocation inside the incumbent workbench. Explicit user
// wording and responsibility assignments stay separate from catalogue approval.
import { useCallback, useEffect, useState } from "react";
import { ApiError } from "./api";
import { packageApi, type PackageRecord, type ScopeRecord, type ScopeWork, type ScopeInput } from "./packageApi";
import "./reports.css";

const roles = ["design", "document", "supply", "install", "test", "certify", "inspect", "maintain_operation", "protect"];
const words = (value: string) => value.replaceAll("_", " ");
const stageLabel = (stage: PackageRecord["stages"][number]) => `${stage.label}${stage.novation_phase === "none" ? "" : ` · ${stage.novation_phase === "pre" ? "before" : "after"} novation`}`;
const blankScope = (): ScopeInput => ({ item_kind: "obligation", user_text: "", inclusion: "included", deliverable: "", interface_ids: [] });

export function Packages({ projectId, tick, onSignedOut }: { projectId: string; tick: number; onSignedOut: () => void }) {
  const [items, setItems] = useState<PackageRecord[]>([]);
  const [works, setWorks] = useState<ScopeWork[]>([]);
  const [selected, setSelected] = useState("");
  const [scope, setScope] = useState<ScopeRecord[]>([]);
  const [title, setTitle] = useState("");
  const [kind, setKind] = useState("services");
  const [worksScope, setWorksScope] = useState("trade");
  const [novation, setNovation] = useState(false);
  const [input, setInput] = useState<ScopeInput>(blankScope);
  const [editing, setEditing] = useState<ScopeRecord | null>(null);
  const [busy, setBusy] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState("");
  const [reload, setReload] = useState(0);
  const fail = useCallback((e: unknown) => {
    if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
    setError(e instanceof ApiError && e.status === 409 ? "This assignment changed or already exists. Your wording has been kept; review the saved scope before retrying." : e instanceof ApiError && e.status === 422 ? "This assignment is not valid for the selected package, work item or stage. Review those choices." : "Could not load or save package scope. Try again.");
  }, [onSignedOut]);
  useEffect(() => {
    let stopped = false;
    Promise.all([packageApi.list(projectId), packageApi.works(projectId)]).then(([p, w]) => { if (!stopped) { setItems(p.items); setWorks(w.items.filter(item => !item.is_group)); setLoaded(true); } }).catch(e => { if (!stopped) { fail(e); setLoaded(true); } });
    return () => { stopped = true; };
  }, [projectId, tick, reload, fail]);
  useEffect(() => {
    if (!selected) return;
    let stopped = false;
    packageApi.scope(projectId, selected).then(result => { if (!stopped) setScope(result.items); }).catch(e => { if (!stopped) fail(e); });
    return () => { stopped = true; };
  }, [projectId, selected, tick, reload, fail]);
  const pkg = items.find(item => item.id === selected);
  const latestScope = editing && scope.find(row => row.id === editing.id);
  function choose(id: string) { setSelected(id); setScope([]); setInput(blankScope()); setEditing(null); setError(""); }
  async function create() {
    setBusy(true); setError("");
    try { const created = await packageApi.create(projectId, title, kind, worksScope, novation); setItems(old => [...old, created]); choose(created.id); setTitle(""); setReload(n => n + 1); }
    catch (e) { fail(e); } finally { setBusy(false); }
  }
  async function saveScope() {
    setBusy(true); setError("");
    try {
      if (editing) await packageApi.patchScope(projectId, selected, editing.id, editing.version, { user_text: input.user_text, deliverable: input.deliverable, inclusion: input.inclusion });
      else await packageApi.addScope(projectId, selected, input);
      setEditing(null); setInput(blankScope()); setReload(n => n + 1);
    } catch (e) { fail(e); } finally { setBusy(false); }
  }
  return <section className="reports" aria-label="Packages" aria-busy={busy}>
    <div className="report-library"><h1>Packages</h1><p className="report-muted">Assign scope and responsibilities before requesting a proposal.</p>
      <form onSubmit={e => { e.preventDefault(); void create(); }}><fieldset disabled={busy}>
        <label htmlFor="package-title">Package title</label><input id="package-title" required maxLength={200} value={title} onChange={e => setTitle(e.target.value)} disabled={busy} />
        <label htmlFor="package-kind">Package kind</label><select id="package-kind" value={kind} onChange={e => setKind(e.target.value)} disabled={busy}><option value="services">Services</option><option value="works">Works</option><option value="supply">Supply</option></select>
        {kind === "works" && <><label htmlFor="package-works-scope">Works scope</label><select id="package-works-scope" value={worksScope} onChange={e => setWorksScope(e.target.value)}><option value="trade">Trade</option><option value="head_contract">Head contract</option></select></>}
        {kind === "services" && <label><input type="checkbox" checked={novation} onChange={e => setNovation(e.target.checked)} /> Separate stages before and after novation</label>}
        <button className="btn" disabled={busy || !!editing || !!input.user_text || !title.trim()}>Create package</button>
      </fieldset></form>
      <h2>Saved packages</h2>{!loaded ? <p role="status">Loading packages…</p> : !items.length ? <p>No packages yet.</p> : <ul className="report-list">{items.map(item => <li key={item.id}><button type="button" aria-current={selected === item.id ? "true" : undefined} disabled={busy || !!editing || !!input.user_text} onClick={() => choose(item.id)}>{item.title} · {words(item.kind)}</button></li>)}</ul>}
    </div>
    <div className="report-reader">
      {error && <div className="banner" role="alert"><p>{error}</p><button type="button" className="btn btn-small" onClick={() => setReload(n => n + 1)}>Reload saved scope</button></div>}
      {!pkg ? <p className="report-empty">Choose a package to review and assign its scope.</p> : <>
        <header className="report-toolbar"><div><h2>{pkg.title}</h2><p className="report-muted">{words(pkg.kind)} · {words(pkg.lifecycle_status)}. Scope is accepted for planning, not a verified appointment.</p></div></header>
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
      </>}
    </div>
  </section>;
}
