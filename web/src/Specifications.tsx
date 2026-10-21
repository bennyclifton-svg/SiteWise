import { Pending, ImportStatus } from "./Activity";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { ApiError, type Doc } from "./api";
import { specificationApi, type SpecificationType, type SpecificationApplication, type SpecificationAssertion, type SpecificationValue, type SpecificationView, type SpecificationIssue } from "./specificationApi";
import "./reports.css";
import { externallyMaintained, coverageNames } from "./SpecificationCoverage";

const words = (s: string) => s.replaceAll("_", " ");
const blankType = (): SpecificationType => ({id:"",project_id:"",code:"",title:"",kind:"material",system_id:"",description:"",scheme:"",option_group:"",status:"active",version:0});
const blankApplication = (type: string): SpecificationApplication => ({id:"",project_id:"",type_id:type,location:"",surface:"",action:"investigate",inclusion:"unknown",selection:"undecided",version:0});
export function specificationValue(v: SpecificationValue): string {
  if (v.state !== "known") return words(v.state);
  let value = v.text ?? (v.number !== undefined ? `${v.number}${v.upper !== undefined ? `–${v.upper}` : ""}${v.unit ? ` ${v.unit}` : ""}` : v.boolean === false ? "No" : "Yes");
  if (v.qualifier && v.qualifier !== "exact") value = `${words(v.qualifier)} ${value}`;
  return [value, v.datum, v.basis, v.method].filter(Boolean).join("; ");
}

export function Specifications({projectId, tick, documents, onSignedOut, initialSelected, initialMode, initialSystem, onChanged}: {projectId: string; tick: number; documents: Doc[]; onSignedOut: () => void; initialSelected?:string; initialMode?:string; initialSystem?:string; onChanged?:()=>void}) {
  const [view,setView] = useState<SpecificationView | null>(null);
  const [issues,setIssues] = useState<SpecificationIssue[]>([]);
  const [selected,setSelected] = useState("");
  const [editor,setEditor] = useState<SpecificationType | null>(null);
  const [application,setApplication] = useState<SpecificationApplication | null>(null);
  const [schedule,setSchedule] = useState(false);
  const [busy,setBusy] = useState(false);
  const [error,setError] = useState("");
  const [loadError,setLoadError] = useState("");
  const [reload,setReload] = useState(0);
  const [search,setSearch] = useState("");
  const [message,setMessage] = useState("");
  const [issueTitle,setIssueTitle] = useState("Materials and finishes schedule");
  const [purpose,setPurpose] = useState("preliminary");
  const [linkTarget,setLinkTarget] = useState("");
  const [linkKind,setLinkKind] = useState("finished_with");
  const [linkSequence,setLinkSequence] = useState("");
  const [linkProportion,setLinkProportion] = useState("");
  const [linkBasis,setLinkBasis] = useState("");
  const [importDoc,setImportDoc] = useState("");
  const [importTarget,setImportTarget] = useState("");
  useEffect(()=>{setSelected(initialSelected??"");setSchedule(initialMode==="schedule");setEditor(initialMode==="new"?{...blankType(),system_id:initialSystem??""}:null);setApplication(null);},[initialSelected,initialMode,initialSystem]);
  const importing=view?.imports?.some(i=>i.status==="queued" || i.status==="leased") ?? false;
  useEffect(()=>{if(!importing)return;const timer=window.setInterval(()=>setReload(n=>n+1),3000);return()=>window.clearInterval(timer);},[importing]);
  useEffect(() => {
    let stopped=false;
    Promise.all([specificationApi.get(projectId),specificationApi.issues(projectId)]).then(([v,i]) => {if(!stopped){setView(v);setIssues(i);setLoadError("");}},e=>{if(stopped)return;if(e instanceof ApiError && e.status===401)onSignedOut();else setLoadError("Could not load specifications. Reload to try again; your unsaved entries are kept.");});
    return ()=>{stopped=true;};
  },[projectId,tick,reload,onSignedOut]);
  const item=view?.snapshot.types.find(t=>t.id===selected);
  const reader=useRef<HTMLDivElement>(null);
  useEffect(()=>{
    if(!editor && !item && !schedule)return;
    const heading=reader.current?.querySelector<HTMLElement>("h2");
    if(!heading)return;
    heading.tabIndex=-1;
    heading.focus({preventScroll:true});
    if(window.matchMedia("(max-width: 760px)").matches)heading.scrollIntoView({block:"start"});
  },[!!editor,editor?.id,item?.id,schedule]);
  const apps=view?.snapshot.applications.filter(a=>a.type_id===selected) ?? [];
  const assertions=view?.snapshot.assertions.filter(a=>a.type_id===selected) ?? [];
  const stale=!!editor?.id && item?.version!==editor.version;
  async function run(action: ()=>Promise<unknown>, done: ()=>void = ()=>{}) {
    setBusy(true);setError("");setMessage("");
    try {await action();done();setReload(n=>n+1);onChanged?.();setMessage("Specification saved.");}
    catch(e){if(e instanceof ApiError && e.status===401)onSignedOut();else setError(e instanceof Error?e.message:"Could not save the specification.");if(e instanceof ApiError && e.status===409)setReload(n=>n+1);}
    finally {setBusy(false);}
  }
  function select(id:string){setSelected(id);setSchedule(false);setApplication(null);setError("");setLinkTarget("");}
  function saveType(e:FormEvent){e.preventDefault();if(!editor || stale)return;void run(async()=>{const saved=await specificationApi.saveType(projectId,editor);setSelected(saved.id);},()=>setEditor(null));}
  function saveApplication(e:FormEvent){e.preventDefault();if(!application)return;void run(()=>specificationApi.saveApplication(projectId,application),()=>setApplication(null));}
  const blocked=busy || !!loadError;
  return <section className={`reports${initialMode?" spec-embedded":""}`} aria-label="Specification details" aria-busy={busy}>
    <div className="report-library">
      <h1>Specifications</h1><p className="report-muted">Materials, finishes and equipment, with their locations and decisions.</p>
      <div className="report-actions"><button className="btn btn-primary" disabled={blocked || !view || !!editor} onClick={()=>{setEditor(blankType());setSelected("");setSchedule(false);setApplication(null);}}>Add item</button><button className="btn" disabled={!view || !!editor} aria-pressed={schedule} onClick={()=>{setSchedule(true);setApplication(null);}}>View schedule</button></div>
      <label htmlFor="spec-search">Find an item</label><input id="spec-search" className="input" value={search} onChange={e=>setSearch(e.target.value)} placeholder="Code, item or scheme" />
      {!view ? <p role="status">{loadError || <Pending delayed>Loading specifications…</Pending>}</p> : view.snapshot.types.length===0 ? <p className="report-empty">Start with a material or product. Add its locations and the details you already know.</p> : <ul className="report-list">{view.snapshot.types.filter(t=>`${t.code} ${t.title} ${t.scheme}`.toLowerCase().includes(search.toLowerCase())).map(t=><li key={t.id}><button disabled={busy || !!editor} aria-current={!schedule && selected===t.id?"true":undefined} onClick={()=>select(t.id)}><strong>{t.code ? `${t.code} · ` : ""}{t.title}</strong><br/>{t.scheme || words(t.kind)}{t.status==="retired"?" · retired":""}</button></li>)}</ul>}
      <h2>Issued schedules</h2>{issues.length===0?<p className="report-muted">No schedules issued yet.</p>:<ul className="report-list">{issues.map(i=><li key={i.id}><a href={`${specificationApi.exportURL(projectId,i.id)}?format=pdf`}>Issue {i.number}: {i.title} · PDF</a> <a href={specificationApi.exportURL(projectId,i.id)} aria-label={`Issue ${i.number} CSV`}>CSV</a><p className="report-muted">{words(i.purpose)} · {new Date(i.created_at).toLocaleDateString()}</p></li>)}</ul>}
      <h2>Import project information</h2><p className="report-muted">Read a filed schedule, specification or product sheet. Imported readings need source review. Confirming a reading does not select a product or transfer specification responsibility.</p>
      {documents.length===0?<p className="report-muted">File your documents in the project register, then return here to import them.</p>:<form onSubmit={e=>{e.preventDefault();void run(()=>specificationApi.import(projectId,importDoc,importTarget));}}>
        <label>Document to import<select required value={importDoc} onChange={e=>setImportDoc(e.target.value)}><option value="">Choose a document</option>{documents.map(d=><option value={d.id} key={d.id}>{d.filename}</option>)}</select></label>
        <label>Import into<select value={importTarget} onChange={e=>setImportTarget(e.target.value)}><option value="">New candidate items</option>{view?.snapshot.types.filter(t=>t.status==="active").map(t=><option key={t.id} value={t.id}>{t.code} {t.title}</option>)}</select></label>
        <button className="btn" disabled={blocked || !importDoc || importing}>Read specification information</button>
      </form>}
      {view?.imports?.map(i=><p className="report-muted" role="status" key={i.document_id}>{documents.find(d=>d.id===i.document_id)?.filename ?? "Source document"}: <ImportStatus status={i.status} /> · {i.count} candidate items retained.{i.error && ` ${i.error}`}</p>)}
    </div>
    <div className="report-reader" ref={reader}>
      {(error || loadError) && <div className="banner" role="alert"><p>{error || loadError}</p><button className="btn" disabled={busy} onClick={()=>setReload(n=>n+1)}>Reload specifications</button></div>}
      <p className={busy ? "report-muted" : "sr-only"} role="status">{busy ? <Pending delayed>Saving…</Pending> : message}</p>
      {editor && <form className="package-scope-form" onSubmit={saveType} aria-label="Specification item">
        <h2>{editor.id?"Edit item":"Add a specification item"}</h2>
        {stale && <p className="report-conflict" role="alert">This item changed elsewhere. Cancel this edit and reopen it to use the saved version.</p>}
        <label>Item name<input required maxLength={200} value={editor.title} onChange={e=>setEditor({...editor,title:e.target.value})}/></label>
        <label>Code<input maxLength={80} value={editor.code} onChange={e=>setEditor({...editor,code:e.target.value})}/></label>
        <label>Kind<select value={editor.kind} onChange={e=>setEditor({...editor,kind:e.target.value})}>{["material","product","finish","assembly","equipment","unspecified"].map(v=><option key={v}>{v}</option>)}</select></label>
        <label>Building system<select value={editor.system_id} onChange={e=>setEditor({...editor,system_id:e.target.value})}><option value="">Not assigned yet</option>{view?.sections.map(s=><optgroup key={s.id} label={s.label}><option value={s.id}>{s.label} — general</option>{s.systems.map(leaf=><option key={leaf.id} value={leaf.id}>{leaf.label}</option>)}</optgroup>)}</select></label>
        <label>Description<textarea rows={3} readOnly={!!editor.id&&!!view&&externallyMaintained(view,editor)} maxLength={4000} value={editor.description} onChange={e=>setEditor({...editor,description:e.target.value})}/></label>
        <label>Scheme<input maxLength={120} value={editor.scheme} onChange={e=>setEditor({...editor,scheme:e.target.value})} placeholder="For example, light scheme"/></label>
        <label>Alternative group<input maxLength={120} value={editor.option_group} onChange={e=>setEditor({...editor,option_group:e.target.value})} placeholder="For example, island benchtop options"/></label>
        {editor.id && <label>Status<select value={editor.status} onChange={e=>setEditor({...editor,status:e.target.value})}><option value="active">Active</option><option value="retired">Retired</option></select></label>}
        <div className="report-actions"><button className="btn btn-primary" disabled={blocked || stale}>Save item</button><button type="button" className="btn" disabled={busy} onClick={()=>setEditor(null)}>Cancel</button></div>
      </form>}
      {!editor && !schedule && item && <>
        <header className="report-toolbar"><div><h2>{item.code ? `${item.code} · ` : ""}{item.title}</h2><p className="report-muted">{words(item.kind)}{item.scheme?` · ${item.scheme}`:""}{item.option_group?` · ${item.option_group}`:""}</p></div><button className="btn" disabled={blocked} onClick={()=>setEditor({...item})}>Edit item</button></header>
        {item.description && <p className="report-wording">{item.description}</p>}
        {item.review_status==="proposed" && <p className="report-notice">Imported candidate. Review the item identity, scope and source values before using it as a project selection.</p>}
        <section className="report-section" aria-label="Item applications"><h3>Where it is used</h3>
          {apps.length===0?<p className="report-muted">No location assigned. This item remains available for a later choice.</p>:apps.map(a=><div className="report-block" key={a.id}><strong>{a.location}{a.surface?` · ${a.surface}`:""}</strong><p>{words(a.action)} · {words(a.inclusion)} · {words(a.selection)}</p><button className="btn btn-small" disabled={blocked || !!application} onClick={()=>setApplication({...a})}>Edit {a.location}</button></div>)}
          <div className="report-actions"><button className="btn" disabled={blocked || !!application || item.status==="retired"} onClick={()=>setApplication(blankApplication(item.id))}>Add location</button></div>
          {application && <form className="package-scope-form" onSubmit={saveApplication} aria-label="Item location"><label>Link to Profiler work<select value={application.work_item_id??""} onChange={e=>{const scope=view?.snapshot.scope.find(s=>s.work_item_id===e.target.value);setApplication({...application,work_item_id:scope?.work_item_id??"",part_id:scope?.part_id??"",location:scope?.location??application.location,action:scope?.action??"investigate",inclusion:scope?.inclusion??"unknown"});}}><option value="">Not linked to project scope</option>{view?.snapshot.scope.filter(s=>!s.retired).map(s=><option key={s.work_item_id} value={s.work_item_id}>{s.title} · {s.location} · {s.action} · {s.inclusion}</option>)}</select></label><label>Location<input required readOnly={!!application.work_item_id} maxLength={200} value={application.location} onChange={e=>setApplication({...application,location:e.target.value})} placeholder="For example, ground floor bathroom"/></label><label>Surface<input maxLength={80} value={application.surface} onChange={e=>setApplication({...application,surface:e.target.value})} placeholder="Floor, walls, ceiling…"/></label>
            <p className="report-muted">Work action and inclusion are maintained in Project Profile.{application.work_item_id?` ${words(application.action)} · ${application.inclusion}.`:" This is an application only; project scope remains unconfirmed."}</p>
            <label>Choice<select value={application.selection} onChange={e=>setApplication({...application,selection:e.target.value})}>{["undecided","selected","rejected"].map(v=><option key={v}>{v}</option>)}</select></label>
            <div className="report-actions"><button className="btn btn-primary" disabled={blocked}>Save location</button><button className="btn" type="button" disabled={busy} onClick={()=>setApplication(null)}>Cancel</button></div>
          </form>}
        </section>
        <section className="report-section" aria-label="Specification values"><h3>Details and evidence</h3>
          <p className="report-muted">Values apply to every location unless a location is specified. Source claims remain separate from project requirements and choices.</p>
          {assertions.filter(a=>a.status!=="superseded" && a.status!=="rejected").map(a=><AssertionReview key={a.id} assertion={a} view={view!} documents={documents} busy={blocked} onReview={(status,reason,checked)=>run(()=>specificationApi.review(projectId,a,status,assertions.find(p=>p.status==="accepted" && p.key===a.key && p.meaning===a.meaning && (p.application_id??"")===(a.application_id??""))?.id??"",reason,checked))}/>)}
          {item.status!=="retired" && <AssertionForm key={item.id} view={view!} type={item.id} applications={apps} documents={documents} disabled={blocked} onAdd={a=>run(()=>specificationApi.addAssertion(projectId,a))}/>}
          <details className="delivery-dates"><summary>Decision history</summary>{assertions.filter(a=>a.status==="superseded" || a.status==="rejected").map(a=><p key={a.id}>{view?.attributes.find(d=>d.key===a.key)?.label}: {specificationValue(a.value)} · {a.status}{a.reason?` · ${a.reason}`:""}</p>)}</details>
        </section>
        <section className="report-section"><h3>Related items</h3>{view?.rows.find(r=>r.type_id===item.id)?.relationships.map((r,i)=><p key={i}>{r}</p>)}
          <form className="package-scope-form" onSubmit={e=>{e.preventDefault();void run(()=>specificationApi.relationship(projectId,{from_id:item.id,to_id:linkTarget,kind:linkKind,sequence:linkSequence?Number(linkSequence):undefined,proportion:linkProportion?Number(linkProportion):undefined,basis:linkBasis}),()=>{setLinkTarget("");setLinkSequence("");setLinkProportion("");setLinkBasis("");});}}>
            <label>Relationship<select value={linkKind} onChange={e=>{setLinkKind(e.target.value);setLinkSequence("");setLinkProportion("");}}>{["finished_with","component_of","requires","matches","replaces"].map(v=><option key={v} value={v}>{words(v)}</option>)}</select></label>
            <label>Related item<select required value={linkTarget} onChange={e=>setLinkTarget(e.target.value)}><option value="">Choose an item</option>{view?.snapshot.types.filter(t=>t.id!==item.id && t.status==="active").map(t=><option value={t.id} key={t.id}>{t.code} {t.title}</option>)}</select></label>
            {linkKind==="component_of" && <><label>Layer order<input type="number" min={0} max={1000} value={linkSequence} onChange={e=>setLinkSequence(e.target.value)}/></label><label>Blend percentage<input type="number" min={0} max={100} step="any" value={linkProportion} onChange={e=>setLinkProportion(e.target.value)}/></label>{linkProportion && <label>Proportion basis<input required maxLength={200} value={linkBasis} onChange={e=>setLinkBasis(e.target.value)} placeholder="By count, area or mass"/></label>}</>}
            <button className="btn" disabled={blocked || !linkTarget || item.status==="retired" || (!!view&&externallyMaintained(view,item))}>Link items</button>
          </form>
        </section>
      </>}
      {!editor && schedule && view && <>
        <header className="report-toolbar"><div><h2>Materials and finishes schedule</h2><p className="report-muted">Draft · {view.snapshot.project_name}. Accepted values are shown; unresolved choices stay visible.</p></div><a className="btn" href={specificationApi.exportURL(projectId)}>Export draft CSV</a></header>
        {view.snapshot.coverage.length>0&&<section className="report-section"><h3>Consultant references and coverage</h3>{view.snapshot.coverage.map(c=><div className="report-block" key={c.id}><h4>{view.sections.find(s=>s.id===c.system_id)?.label??c.system_id}{c.type_id?` · ${view.snapshot.types.find(t=>t.id===c.type_id)?.title??"Item"}`:""}</h4><p>{coverageNames[c.mode]} · {c.extent}</p><p>Maintained by: {c.maintainer||"Not assigned"} · Review coordinated by: {c.reviewer||"Not assigned"}</p>{c.exclusions&&<p>Gaps: {c.exclusions}</p>}{c.references.map(r=><p key={r.document_id}><a href={`/api/documents/${r.document_id}/file`} target="_blank" rel="noreferrer">{r.filename}</a> · {r.number} · Revision {r.revision||"not stated"} · {r.location}{r.stale?" · REVISION REVIEW REQUIRED":""}{r.unavailable?" · SOURCE UNAVAILABLE":""}</p>)}</div>)}</section>}
        {view.rows.length===0?<p className="report-empty">Add an item to start this schedule.</p>:view.rows.map((r,i)=><section className="report-block" key={`${r.type_id}/${r.application_id??i}`}><h3>{r.code} {r.title}</h3><p>{r.location}{r.surface?` · ${r.surface}`:""} · {r.selection} · {r.inclusion}</p>{r.authority==="external"&&<p className="report-muted">Maintained externally. These are confirmed source readings; the referenced document remains authoritative.</p>}{r.description && <p className="report-wording">{r.description}</p>}{r.fields.map(f=><p key={`${f.key}/${f.meaning}`}><strong>{f.label}</strong> ({f.meaning}): {f.value}{f.source.unavailable?" · source unavailable":""}{f.source.stale?" · revision review required":""}</p>)}{r.relationships.map((link,n)=><p key={n}>{link}</p>)}{r.pending>0 && <p className="report-muted">{r.pending} proposed {r.pending===1?"value needs":"values need"} review.</p>}</section>)}
 
        <section className="report-section"><h3>Issued schedules</h3>{issues.length===0?<p className="report-muted">No schedules issued yet.</p>:<ul className="report-list">{issues.map(i=><li key={i.id}><a href={`${specificationApi.exportURL(projectId,i.id)}?format=pdf`}>Issue {i.number}: {i.title} · PDF</a> <a href={specificationApi.exportURL(projectId,i.id)} aria-label={`Issue ${i.number} CSV`}>CSV</a><p className="report-muted">{words(i.purpose)} · {new Date(i.created_at).toLocaleDateString()}</p></li>)}</ul>}</section>
        <form className="package-scope-form" onSubmit={e=>{e.preventDefault();void run(()=>specificationApi.issue(projectId,issueTitle,purpose,view.snapshot.revision));}}><h3>Issue this schedule</h3><p>Save an immutable copy of the current values, evidence and unresolved decisions.</p><label>Schedule title<input required maxLength={200} value={issueTitle} onChange={e=>setIssueTitle(e.target.value)}/></label><label>Issue purpose<select value={purpose} onChange={e=>setPurpose(e.target.value)}>{["preliminary","coordination","pricing"].map(v=><option key={v}>{v}</option>)}</select></label><button className="btn btn-primary" disabled={blocked || (view.rows.length===0 && view.snapshot.coverage.length===0)}>Issue schedule</button></form>
      </>}
      {!editor && !item && !schedule && view && <p className="report-empty">Choose an item to review its locations, requirements and selections, or add your first item.</p>}
    </div>
  </section>;
}

function AssertionReview({assertion:a,view,documents,busy,onReview}: {assertion:SpecificationAssertion;view:SpecificationView;documents:Doc[];busy:boolean;onReview:(status:string,reason:string,checked:boolean)=>Promise<void>}) {
  const [reason,setReason]=useState("");const [checked,setChecked]=useState(false);
  const prior=view.snapshot.assertions.find(p=>p.status==="accepted" && p.type_id===a.type_id && p.key===a.key && p.meaning===a.meaning && (p.application_id??"")===(a.application_id??""));
  const doc=documents.find(d=>d.id===a.source.document_id);
  const type=view.snapshot.types.find(t=>t.id===a.type_id);
  const external=!!type&&externallyMaintained(view,type);
  return <article className="report-block"><h4>{view.attributes.find(d=>d.key===a.key)?.label ?? a.key} · {words(a.meaning)}</h4><p className="report-wording">{specificationValue(a.value)}</p><p className="report-muted">{a.application_id?view.snapshot.applications.find(p=>p.id===a.application_id)?.location:"All applications"} · {a.status} · {a.origin==="user"?"User input":"Source statement"}</p>
    {a.source.document_id && <details className="report-citation"><summary>Source: {doc?.filename || "Unavailable document"}, page {a.source.page}</summary>{!a.source.unavailable && <a target="_blank" rel="noreferrer" href={`/api/documents/${a.source.document_id}/file#page=${a.source.page}`}>Open source page</a>}<p>{a.source.excerpt}</p><p>{a.source.issue} · {words(a.source.marker||"unreviewed")}{a.source.stale?" · Revision review required":""}</p></details>}
    {external&&a.origin==="user"&&a.status==="accepted"&&<p className="report-notice">Local decision retained for coordination. It is excluded from issued requirements while this item is maintained externally.</p>}{a.status==="proposed" && <div className="package-scope-form">{prior && <p>Current accepted value: {specificationValue(prior.value)}</p>}<label>Decision reason{prior?" (required to replace the current value)":""}<input maxLength={1000} value={reason} onChange={e=>setReason(e.target.value)}/></label>{a.source.document_id && <label><input type="checkbox" checked={checked} onChange={e=>setChecked(e.target.checked)}/> I checked the source page and its revision markings.</label>}{a.source.marker==="withdrawn" && <p className="report-muted">This source row is withdrawn and cannot become an active choice.</p>}<div className="report-actions"><button className="btn btn-primary" disabled={busy || !!(prior && !reason.trim()) || !!(a.source.document_id && !checked) || a.source.marker==="withdrawn" || a.source.unavailable || a.source.stale || (external && a.origin==="user")} onClick={()=>void onReview("accepted",reason,checked)}>{a.origin==="document"?"Confirm reading":"Accept value"}</button><button className="btn" disabled={busy} onClick={()=>void onReview("rejected",reason,checked)}>Reject value</button></div></div>}
  </article>;
}

function AssertionForm({view,type,applications,documents,disabled,onAdd}: {view:SpecificationView;type:string;applications:SpecificationApplication[];documents:Doc[];disabled:boolean;onAdd:(a:Partial<SpecificationAssertion>)=>Promise<void>}) {
  const [open,setOpen]=useState(false);const [key,setKey]=useState("material");const [meaning,setMeaning]=useState("selection");const [scope,setScope]=useState("");
  const [state,setState]=useState("known");const [value,setValue]=useState("");const [unit,setUnit]=useState("");const [qualifier,setQualifier]=useState("");const [basis,setBasis]=useState("");const [method,setMethod]=useState("");const [datum,setDatum]=useState("");
  const [doc,setDoc]=useState("");const [page,setPage]=useState("1");const [excerpt,setExcerpt]=useState("");const [marker,setMarker]=useState("unreviewed");
  const definition=view.attributes.find(a=>a.key===key)!;
  async function submit(e:FormEvent){e.preventDefault();const val:SpecificationValue={state,basis,method,datum,qualifier};if(state==="known"){if(definition.type==="number"){val.number=Number(value);val.unit=unit||definition.units?.[0];}else val.text=value;}
    await onAdd({type_id:type,application_id:scope,key,meaning,value:val,source:doc?{document_id:doc,page:Number(page),excerpt,marker}:{}});
  }
  return <div className="report-block"><button className="btn" disabled={disabled} aria-expanded={open} onClick={()=>setOpen(!open)}>{open?"Close new value":"Add a detail or requirement"}</button>{open && <form className="package-scope-form" onSubmit={submit} aria-label="Specification detail">
    <label>Attribute<select value={key} onChange={e=>{setKey(e.target.value);setUnit("");setValue("");}}>{Array.from(new Set(view.attributes.map(a=>a.group))).map(g=><optgroup label={g} key={g}>{view.attributes.filter(a=>a.group===g).map(a=><option value={a.key} key={a.key}>{a.label}</option>)}</optgroup>)}</select></label>
    <label>Meaning<select value={meaning} onChange={e=>setMeaning(e.target.value)}>{["selection","requirement","capability","allowance","exclusion","observed"].map(v=><option value={v} key={v}>{words(v)}</option>)}</select></label>
    <label>Applies to<select value={scope} onChange={e=>setScope(e.target.value)}><option value="">All applications</option>{applications.map(a=><option value={a.id} key={a.id}>{a.location}{a.surface?` · ${a.surface}`:""}</option>)}</select></label>
    <label>Value state<select value={state} onChange={e=>setState(e.target.value)}>{["known","unknown","deferred","not_applicable","not_required"].map(v=><option value={v} key={v}>{words(v)}</option>)}</select></label>
    {state==="known" && <><label>Value{definition.type==="number"?<input type="number" required step="any" value={value} onChange={e=>setValue(e.target.value)}/>:<textarea required rows={3} maxLength={4000} value={value} onChange={e=>setValue(e.target.value)}/>}</label>{definition.units && <label>Unit<select value={unit||definition.units[0]} onChange={e=>setUnit(e.target.value)}>{definition.units.map(u=><option key={u}>{u}</option>)}</select></label>}</>}
    <details className="delivery-dates"><summary>Measurement and performance context</summary><label>Qualifier<select value={qualifier} onChange={e=>setQualifier(e.target.value)}>{["","exact","minimum","maximum","approximate","site_measured","tolerance"].map(v=><option value={v} key={v}>{v?words(v):"None"}</option>)}</select></label><label>Basis<input maxLength={500} value={basis} onChange={e=>setBasis(e.target.value)}/></label><label>Test method / conditions<input maxLength={300} value={method} onChange={e=>setMethod(e.target.value)}/></label><label>Datum<input maxLength={120} value={datum} onChange={e=>setDatum(e.target.value)} placeholder="For example, above finished floor"/></label></details>
    <label>Supporting document<select value={doc} onChange={e=>setDoc(e.target.value)}><option value="">Direct user input</option>{documents.map(d=><option value={d.id} key={d.id}>{d.filename}</option>)}</select></label>
    {doc && <><label>Source page<input type="number" required min={1} max={100000} value={page} onChange={e=>setPage(e.target.value)}/></label><label>Source excerpt<textarea required rows={3} maxLength={4000} value={excerpt} onChange={e=>setExcerpt(e.target.value)}/></label><label>Source marking<select value={marker} onChange={e=>setMarker(e.target.value)}>{["unreviewed","normal","changed","withdrawn"].map(v=><option value={v} key={v}>{v}</option>)}</select></label></>}
    <button className="btn btn-primary" disabled={disabled}>Save for review</button>
  </form>}</div>;
}
