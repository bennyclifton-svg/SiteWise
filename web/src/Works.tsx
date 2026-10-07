import { useCallback, useEffect, useState } from "react";
import { ApiError } from "./api";
import { workApi, type WorkRecord } from "./workApi";
import type { Profile } from "./profileApi";
import "./reports.css";

const actions = ["new", "replace", "upgrade", "alter", "repair", "remove", "retain", "investigate"];
const words = (s: string) => s.replaceAll("_", " ");
const layoutEligible = (system: string, action: string) => system === "interiors.walls-linings" && ["new", "alter", "remove"].includes(action);
type Editor = { layout: "unknown" | "yes" | "no"; base: WorkRecord; title: string; action: string; inclusion: string; note: string; target: string; quantity: string; unit: string };

export function Works({ projectId, tick, onSignedOut }: { projectId: string; tick: number; onSignedOut: () => void }) {
  const [items, setItems] = useState<WorkRecord[]>([]);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [selected, setSelected] = useState("");
  const [editor, setEditor] = useState<Editor | null>(null);
  const [split, setSplit] = useState<{base: WorkRecord; children: {title: string; part_id: string; action: string}[]} | null>(null);
  const [retire, setRetire] = useState(false);
  const [busy, setBusy] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState("");
  const [reload, setReload] = useState(0);
  const fail = useCallback((e: unknown) => {
    if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
    setError(e instanceof ApiError && e.status === 409 ? "This work item changed. Your correction is still here; reload saved work and compare before saving." : e instanceof ApiError && e.status === 422 ? "The correction is not valid. Check the title, action, quantity and unit." : "Could not load or save work. Reload saved work to try again.");
  }, [onSignedOut]);
  useEffect(() => {
    let stopped = false;
    Promise.all([workApi.list(projectId), workApi.profile(projectId)]).then(([work, nextProfile]) => {
      if (!stopped) { setItems(work.items); setProfile(nextProfile); setLoaded(true); }
    }).catch(e => { if (!stopped) { fail(e); setLoaded(true); } });
    return () => { stopped = true; };
  }, [projectId, tick, reload, fail]);
  const item = items.find(row => row.id === selected);
  const stale = !!editor && item?.version !== editor.base.version;
  const part = (work: WorkRecord) => profile?.parts.find(p => p.id === work.part_id)?.label ?? "Location unavailable";
  const system = (work: WorkRecord) => profile?.systems.flatMap(group => group.rows).find(row => row.leaf === work.system_id)?.label ?? work.system_id;
  function edit(work: WorkRecord) {
    setError(""); setEditor({ layout: work.layout_change || "unknown", base: work, title: work.title, action: work.action, inclusion: work.inclusion, note: work.existing_condition_note ?? "", target: work.target.text ?? "", quantity: work.quantity ?? "", unit: work.unit ?? "" });
  }
  async function save() {
    if (!editor) return;
    setBusy(true); setError("");
    const before = items;
    setItems(old => old.map(w => w.id === editor.base.id ? {...w, title:editor.title, action:editor.action, inclusion:editor.inclusion} : w));
    try {
      await workApi.patch(projectId, editor.base.id, { layout_change: layoutEligible(editor.base.system_id, editor.action) ? editor.layout : "unknown", version: editor.base.version, title: editor.title, action: editor.action, inclusion: editor.inclusion, existing_condition_note: editor.note, target: { ...editor.base.target, text: editor.target }, quantity: editor.quantity.trim() || null, ...(editor.quantity.trim() ? { unit: editor.unit.trim() } : {}) });
      setEditor(null); setReload(n => n + 1);
    } catch (e) { setItems(before); fail(e); } finally { setBusy(false); }
  }
  async function mutateTree(remove: boolean) {
    if (!item) return; setBusy(true);setError("");
    try { if(remove) {await workApi.retire(projectId,item.id,item.version);setSelected("");} else if(split) await workApi.split(projectId,split.base.id,split.base.version,split.children); setSplit(null);setRetire(false);setReload(n=>n+1); }
    catch(e) {if(e instanceof ApiError && e.status===409 && e.message.includes('referenced')) setError("This work item is still referenced. Review package scope, costs, delivery and child work before retiring it. " + e.message); else fail(e);} finally{setBusy(false);}
  }
  return <section className="reports" aria-label="Works" aria-busy={busy}>
    <div className="report-library"><h1>Works</h1><p className="report-muted">Review the action and target before assigning package responsibilities.</p>
      {!loaded ? <p role="status">Loading work…</p> : !items.length ? <p>No work items yet. Review the project profile in Filing and include the systems this project touches.</p> : <ul className="report-list">{items.map(work => <li key={work.id}><button type="button" disabled={busy || !!editor || !!split} aria-current={work.id === selected ? "true" : undefined} onClick={() => { setSelected(work.id); setError(""); }}>{work.title}<br />{part(work)} · {work.action} · {work.inclusion}</button></li>)}</ul>}
    </div>
    <div className="report-reader">
      {error && <div className="banner" role="alert"><p>{error}</p><button type="button" className="btn btn-small" disabled={busy} onClick={() => setReload(n => n + 1)}>Reload saved work</button></div>}
      {!item && !editor ? <p className="report-empty">Choose a work item to review its action and source.</p> : <>
        {item && <><header className="report-toolbar"><div><h2>{item.title}</h2><p>{part(item)} · {system(item)}</p><p className="report-muted">{words(item.review_status)} · {words(item.origin)} input</p></div>{!editor && !split && <div className="report-actions"><button type="button" className="btn" disabled={busy || item.deprecated} onClick={() => edit(item)}>Correct work item</button><button className="btn" disabled={busy||item.is_group} onClick={()=>setSplit({base:item,children:[{title:"",part_id:item.part_id,action:item.action},{title:"",part_id:item.part_id,action:item.action}]})}>Split work item</button><button className="btn" disabled={busy} onClick={()=>setRetire(true)}>Retire work item</button></div>}</header>
          {item.deprecated && <p className="report-notice">This system is no longer in the current catalogue. Review its scope in the project profile.</p>}
          <section className="report-section" aria-label="Saved work"><h3>Saved work</h3><p>Action: {item.action} · {item.inclusion}{item.is_group ? " · group" : ""}</p><p>Existing condition: {item.existing_condition ? words(item.existing_condition) : "Not recorded"}. Shared site conditions are edited in the project profile.</p>{item.existing_condition_note && <p className="report-wording">{item.existing_condition_note}</p>}{layoutEligible(item.system_id, item.action) && <p>Room or space boundaries change: {item.layout_change === "yes" ? "Yes" : item.layout_change === "no" ? "No" : "Not recorded"}</p>}{item.is_group && layoutEligible(item.system_id, item.action) && <p className="report-muted">Layout coordination uses each child’s recorded answer. Children do not inherit this group’s answer.</p>}<p>Quantity: {item.quantity == null ? "Not recorded" : `${item.quantity} ${item.unit ?? ""}`}</p><p className="report-wording">Target: {item.target.text || "Not recorded"}</p>
            {!!item.target.values?.length && <ul>{item.target.values.map((value, i) => <li key={i}>{value.key}: {String(value.value)} {value.unit}</li>)}</ul>}
            {!!item.target.clause_refs?.length && <ul>{item.target.clause_refs.map(ref => <li key={`${ref.id}:${ref.version}`}>Clause {ref.id}, version {ref.version}</li>)}</ul>}
            <details><summary>Source and latest correction</summary>{item.provenance.rationale && <p className="report-wording">{item.provenance.rationale}</p>}{item.source_proposal_key && <p>Created from an accepted proposal.</p>}<p>Latest editor: {item.provenance.last_edited_by || item.provenance.actor || "Not recorded"}</p>{item.provenance.last_edited_at && <p>Edited: {new Date(item.provenance.last_edited_at).toLocaleString()}</p>}<WorkSources sources={item.provenance.sources} /></details>
          </section></>}
        {retire && item && <div className="report-notice"><p>Retire {item.title}? Existing references must be resolved first.</p><button className="btn" disabled={busy} onClick={()=>void mutateTree(true)}>Confirm retirement</button><button className="btn" disabled={busy} onClick={()=>setRetire(false)}>Cancel retirement</button></div>}
        {split && <section className="report-section"><h3>Split work item</h3><p>The parent becomes a group. Evidence remains linked; quantities and room-layout answers require explicit allocation.</p><form className="package-scope-form" onSubmit={e=>{e.preventDefault();void mutateTree(false);}}><fieldset disabled={busy}>{split.children.map((c,i)=><div key={i}><label htmlFor={`child-title-${i}`}>Child {i+1} title</label><input id={`child-title-${i}`} required maxLength={200} value={c.title} onChange={e=>setSplit({...split,children:split.children.map((x,j)=>i===j?{...x,title:e.target.value}:x)})}/><label htmlFor={`child-part-${i}`}>Child {i+1} part</label><select id={`child-part-${i}`} value={c.part_id} onChange={e=>setSplit({...split,children:split.children.map((x,j)=>i===j?{...x,part_id:e.target.value}:x)})}>{profile?.parts.map(p=><option key={p.id} value={p.id}>{p.label}</option>)}</select><label htmlFor={`child-action-${i}`}>Child {i+1} action</label><select id={`child-action-${i}`} value={c.action} onChange={e=>setSplit({...split,children:split.children.map((x,j)=>i===j?{...x,action:e.target.value}:x)})}>{actions.map(a=><option key={a}>{a}</option>)}</select></div>)}<button type="button" className="btn" disabled={split.children.length>=100} onClick={()=>setSplit({...split,children:[...split.children,{title:"",part_id:split.base.part_id,action:split.base.action}]})}>Add child</button><button className="btn">Save work split</button><button type="button" className="btn" onClick={()=>setSplit(null)}>Cancel split</button></fieldset></form></section>}
        {editor && <section className="report-section" aria-label="Work correction"><h3>Correct work item</h3><p className="report-muted">Saving records your planning correction. It does not verify the work or its requirements.</p>
          {stale && <div className="report-notice"><strong>Saved work changed</strong><p>Your unsaved correction is below. Compare it with the saved work above.</p>{item ? <button type="button" className="btn btn-small" disabled={busy} onClick={() => setEditor({ ...editor, base: item })}>Keep my correction against this version</button> : <p>This work item is no longer available. Cancel to leave the editor.</p>}</div>}
          <form className="package-scope-form" onSubmit={e => { e.preventDefault(); void save(); }}><fieldset disabled={busy}>
            <label htmlFor="work-title">Work title</label><input id="work-title" required maxLength={200} value={editor.title} onChange={e => setEditor({ ...editor, title: e.target.value })} />
            <label htmlFor="work-action">Action</label><select id="work-action" value={editor.action} onChange={e => setEditor({ ...editor, action: e.target.value })}>{actions.map(action => <option key={action} value={action}>{action}</option>)}</select>
            <label htmlFor="work-inclusion">Inclusion</label><select id="work-inclusion" value={editor.inclusion} onChange={e => setEditor({ ...editor, inclusion: e.target.value })}><option value="included">Included</option><option value="excluded">Excluded</option></select>
            {!editor.base.is_group && layoutEligible(editor.base.system_id, editor.action) && <><label htmlFor="work-layout">Does this work change room or space boundaries?</label><select id="work-layout" value={editor.layout} onChange={e => setEditor({...editor, layout: e.target.value as Editor["layout"]})}><option value="unknown">Not recorded</option><option value="yes">Yes</option><option value="no">No</option></select><p className="report-muted">Replacing linings without moving boundaries is No. Yes requests coordination; it does not establish a design outcome.</p></>}
            <label htmlFor="work-condition-note">Condition note</label><input id="work-condition-note" maxLength={120} value={editor.note} onChange={e => setEditor({ ...editor, note: e.target.value })} />
            <label htmlFor="work-target">Target wording</label><textarea id="work-target" rows={3} maxLength={1000} value={editor.target} onChange={e => setEditor({ ...editor, target: e.target.value })} />
            <label htmlFor="work-quantity">Planning quantity</label><input id="work-quantity" inputMode="decimal" pattern="[0-9]{1,18}(\.[0-9]{1,9})?" value={editor.quantity} onChange={e => setEditor({ ...editor, quantity: e.target.value })} /><p className="report-muted">Leave blank to remove the quantity and unit.</p>
            <label htmlFor="work-unit">Unit</label><input id="work-unit" maxLength={32} required={!!editor.quantity.trim()} disabled={!editor.quantity.trim()} value={editor.unit} onChange={e => setEditor({ ...editor, unit: e.target.value })} />
            <div className="report-actions"><button className="btn" disabled={busy || stale || !editor.title.trim()}>{busy ? "Saving…" : "Save correction"}</button><button type="button" className="btn" disabled={busy} onClick={() => { setEditor(null); setError(""); }}>Cancel correction</button></div>
          </fieldset></form>
        </section>}
      </>}
    </div>
  </section>;
}

export function WorkSources({ sources }: { sources: unknown }) {
  if (!Array.isArray(sources) || !sources.length) return <p>No direct document excerpts recorded on this item.</p>;
  return <ul>{sources.map((source, index) => {
    const s = source && typeof source === "object" ? source as Record<string, unknown> : {};
    return <li key={index}><p>{typeof s.filename === "string" ? s.filename : "Document source"}{typeof s.page === "number" && s.page > 0 ? ` · page ${s.page}` : ""}</p><p className="report-wording">{typeof s.excerpt === "string" ? s.excerpt : "Excerpt not recorded"}</p></li>;
  })}</ul>;
}
