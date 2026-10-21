import { Fragment, useState } from "react";
import { fmt } from "./CostGrid";
import { FLAG_WORDS, submissionApi, type Column } from "./submissionApi";

const money = (v: string | null | undefined) => v == null ? "Unresolved" : `$${fmt(v)}`;

export function BuilderQuotedTotals({cols, busy, onUse}: {cols: Column[]; busy: boolean; onUse:(sub:string,amount:string,note:string)=>void}) {
  return <>
    <tr className="prices-total builder-quoted"><th scope="row">Total <span>excluding GST</span></th>
      {cols.map(c => <td key={c.submission.id}>
        <strong className="prices-figure">{money(c.cells["stage:lump_sum"]?.decided_by === "user" ? c.totals.base_total : c.builder?.ex_gst)}</strong>
        <details className="builder-source"><summary>Review total{c.builder?.issues.length || !c.submission.document_id ? " · check needed" : ""}</summary>
        {!c.submission.document_id && <span className="prices-note warn">Saved reading; source file is no longer attached.</span>}
        <span className="prices-note">{c.builder?.basis}</span>
        {c.builder?.issues.map((issue,i) => <p className="prices-note warn" key={i}>{issue}</p>)}
          {c.builder?.sources.map((s,i)=><p key={i}>{s}</p>)}
          {!c.builder?.sources.length && <p>No unambiguous total found in the saved text.</p>}
          {c.submission.flags.filter(f=>FLAG_WORDS[f]&&!f.startsWith("gst_")).map(f=><p key={f}>{FLAG_WORDS[f]}</p>)}
          <p>Section figures are part of the quote, not extra amounts. Confirm the total before awarding.</p>
        {c.builder?.ex_gst && c.cells["stage:lump_sum"]?.decided_by !== "user" && <button className="btn btn-small" disabled={busy || !["received","needs_review"].includes(c.submission.status)} onClick={()=>onUse(c.submission.id,c.builder!.ex_gst!,`Confirmed from quote: ${c.builder!.sources.join("; ")}`.slice(0,500))}>Confirm total for award</button>}
        {c.cells["stage:lump_sum"]?.decided_by === "user" && <p>Your confirmed comparison amount is shown. Source quote: {money(c.builder?.ex_gst)} ex GST.</p>}
        <form onSubmit={e=>{e.preventDefault();const data=new FormData(e.currentTarget);onUse(c.submission.id,String(data.get("amount")),"Total corrected by user");}}>
          <label>Construction amount ex GST<input name="amount" type="number" min="0" step="0.01" required defaultValue={c.cells["stage:lump_sum"]?.amount ?? c.builder?.ex_gst ?? ""} /></label>
          <button className="btn btn-small" disabled={busy || !["received","needs_review"].includes(c.submission.status)}>Save corrected total</button>
        </form>
        </details>
      </td>)}<td />
    </tr>
  </>;
}

export function BuilderWorkSections({cols,projectId}: {cols:Column[];projectId:string}) {
  const [open,setOpen]=useState<Set<string>>(new Set());
  const [notes,setNotes]=useState(false);
  const sections=cols.find(c=>c.builder)?.builder?.sections ?? [];
  const labels:Record<string,string>={site:"Site and groundworks",structure:"Structure and roof",envelope:"Windows, doors, cladding",services:"Building services",interiors:"Interiors and finishes",external:"External works"};
  return <>
    {sections.filter(s=>notes || !["terms","other"].includes(s.key)).map(section=><Fragment key={section.key}>
      <tr className="builder-section">
        <th scope="row">{labels[section.key] || section.label}</th>
        {cols.map(c=>{const s=c.builder?.sections.find(s=>s.key===section.key);const n=s?.details.length ?? 0;return <td key={c.submission.id}><span>{s?.summary || "Unclear"}</span>{n>0 && <button className="builder-excerpts" aria-expanded={open.has(section.key)} aria-label={`${open.has(section.key)?"Hide":"Show"} ${section.label} excerpts, ${c.firm?.name ?? c.submission.filename}`} onClick={()=>setOpen(prev=>{const next=new Set(prev);if(next.has(section.key))next.delete(section.key);else next.add(section.key);return next;})}>{open.has(section.key)?"Hide":"Excerpts"} ({n})</button>}</td>;})}<td />
      </tr>
      {open.has(section.key) && <tr className="builder-section-detail"><th scope="row">{section.label} detail</th>
        {cols.map(c=><td key={c.submission.id}><ul>{c.builder?.sections.find(s=>s.key===section.key)?.details.map((d,i)=><li key={i}>
          <span className="prices-note">{d.relationship}</span><p>{d.text}</p>
          {c.submission.document_id ? <a href={submissionApi.fileURL(projectId,c.submission.id,d.section_page)} target="_blank" rel="noreferrer">Source section starts p. {d.section_page || "?"}</a> : <span className="prices-note">Source file no longer attached</span>}
        </li>)}</ul></td>)}<td />
      </tr>}
    </Fragment>)}
    <tr><td colSpan={cols.length+2}><button className="builder-excerpts builder-notes" aria-expanded={notes} onClick={()=>setNotes(!notes)}>{notes?"Hide":"Show"} conditions and other quote detail</button></td></tr>
  </>;
}
