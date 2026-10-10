// One work item's detail, shown inside its system's row in Profile: what was
// saved, where it came from, and the corrections a person can make (correct,
// split, retire). Every write carries the version it was read at, so a
// concurrent edit is refused and shown rather than overwritten.

import { useState } from "react";
import { ApiError } from "./api";
import { workApi, type WorkRecord } from "./workApi";
import type { Part } from "./profileApi";
import "./reports.css";

export const WORK_ACTIONS = ["new", "replace", "upgrade", "alter", "repair", "remove", "retain", "investigate"];
const words = (s: string) => s.replaceAll("_", " ");
export const layoutEligible = (system: string, action: string) => system === "interiors.walls-linings" && ["new", "alter", "remove"].includes(action);

// Quantity is no longer entered here; a saved one passes through unchanged.
export function keptQuantity(work: WorkRecord) {
  return work.quantity == null ? { quantity: null } : { quantity: work.quantity, unit: work.unit ?? "" };
}

/** What a failed work write tells the user; null means the session ended. */
export function workError(e: unknown): string | null {
  if (e instanceof ApiError && e.status === 401) return null;
  if (e instanceof ApiError && e.status === 409 && e.message.includes("referenced"))
    return "This work item is still referenced. Review package scope, costs, delivery and child work before retiring it. " + e.message;
  if (e instanceof ApiError && e.status === 409) return "This work item changed. Your correction is still here; reload saved work and compare before saving.";
  if (e instanceof ApiError && e.status === 422) return "The correction is not valid. Check the title and action.";
  return "Could not load or save work. Reload saved work to try again.";
}

/** Saves one row-level change (action or inclusion), keeping every other field as saved. */
export function patchWorkRow(projectId: string, work: WorkRecord, change: { action?: string; inclusion?: string }) {
  const next = { ...work, ...change };
  return workApi.patch(projectId, work.id, {
    layout_change: layoutEligible(work.system_id, next.action) ? work.layout_change || "unknown" : "unknown",
    version: work.version, title: work.title, action: next.action, inclusion: next.inclusion,
    existing_condition_note: work.existing_condition_note ?? "", target: work.target, ...keptQuantity(work),
  });
}

type Editor = { layout: "unknown" | "yes" | "no"; base: WorkRecord; title: string; action: string; inclusion: string; note: string; target: string };

export function WorkItemDetail({ projectId, item, parts, systemLabel, busy, setBusy, onChanged, onError }: {
  projectId: string; item: WorkRecord; parts: Part[]; systemLabel: string;
  busy: boolean; setBusy: (b: boolean) => void;
  /** Reload saved work after a write. */
  onChanged: () => void;
  onError: (e: unknown) => void;
}) {
  const [editor, setEditor] = useState<Editor | null>(null);
  const [split, setSplit] = useState<{ base: WorkRecord; children: { title: string; part_id: string; action: string }[] } | null>(null);
  const [retire, setRetire] = useState(false);
  const stale = !!editor && item.version !== editor.base.version;
  const part = parts.find(p => p.id === item.part_id)?.label ?? "Location unavailable";

  function edit() {
    setEditor({ layout: item.layout_change || "unknown", base: item, title: item.title, action: item.action, inclusion: item.inclusion, note: item.existing_condition_note ?? "", target: item.target.text ?? "" });
  }
  async function save() {
    if (!editor) return;
    setBusy(true);
    try {
      await workApi.patch(projectId, editor.base.id, { layout_change: layoutEligible(editor.base.system_id, editor.action) ? editor.layout : "unknown", version: editor.base.version, title: editor.title, action: editor.action, inclusion: editor.inclusion, existing_condition_note: editor.note, target: { ...editor.base.target, text: editor.target }, ...keptQuantity(editor.base) });
      setEditor(null); onChanged();
    } catch (e) { onError(e); } finally { setBusy(false); }
  }
  async function mutateTree(remove: boolean) {
    setBusy(true);
    try {
      if (remove) await workApi.retire(projectId, item.id, item.version);
      else if (split) await workApi.split(projectId, split.base.id, split.base.version, split.children);
      setSplit(null); setRetire(false); onChanged();
    } catch (e) { onError(e); } finally { setBusy(false); }
  }
  const setChild = (i: number, change: Partial<{ title: string; part_id: string; action: string }>) =>
    split && setSplit({ ...split, children: split.children.map((x, j) => i === j ? { ...x, ...change } : x) });

  return <div className="works-detail">
    <header className="report-toolbar"><div><h4>{item.title}</h4><p>{part} · {systemLabel}</p><p className="report-muted">{words(item.review_status)} · {words(item.origin)} input</p></div>
      {!editor && !split && <div className="report-actions">
        <button type="button" className="btn btn-small" disabled={busy || item.deprecated} onClick={edit}>Correct work item</button>
        <button type="button" className="btn btn-small" disabled={busy || item.is_group} onClick={() => setSplit({ base: item, children: [{ title: "", part_id: item.part_id, action: item.action }, { title: "", part_id: item.part_id, action: item.action }] })}>Split work item</button>
        <button type="button" className="btn btn-small" disabled={busy} onClick={() => setRetire(true)}>Retire work item</button>
      </div>}
    </header>
    {item.deprecated && <p className="report-notice">This system is no longer in the current catalogue. Review its scope above.</p>}
    <section className="report-section" aria-label="Saved work"><h5>Saved work</h5>
      <p>Action: {item.action} · {item.inclusion}{item.is_group ? " · group" : ""}</p>
      <p>Existing condition: {item.existing_condition ? words(item.existing_condition) : "Not recorded"}. Shared site conditions are edited in the project details.</p>
      {item.existing_condition_note && <p className="report-wording">{item.existing_condition_note}</p>}
      {layoutEligible(item.system_id, item.action) && <p>Room or space boundaries change: {item.layout_change === "yes" ? "Yes" : item.layout_change === "no" ? "No" : "Not recorded"}</p>}
      {item.is_group && layoutEligible(item.system_id, item.action) && <p className="report-muted">Layout coordination uses each child’s recorded answer. Children do not inherit this group’s answer.</p>}
      {item.quantity != null && <p>Planning quantity: {item.quantity} {item.unit ?? ""}</p>}
      <p className="report-wording">Target: {item.target.text || "Not recorded"}</p>
      {!!item.target.values?.length && <ul>{item.target.values.map((value, i) => <li key={i}>{value.key}: {String(value.value)} {value.unit}</li>)}</ul>}
      {!!item.target.clause_refs?.length && <ul>{item.target.clause_refs.map(ref => <li key={`${ref.id}:${ref.version}`}>Clause {ref.id}, version {ref.version}</li>)}</ul>}
      <details><summary>Source and latest correction</summary>
        {item.provenance.rationale && <p className="report-wording">{item.provenance.rationale}</p>}
        {item.source_proposal_key && <p>Created from an accepted proposal.</p>}
        <p>Latest editor: {item.provenance.last_edited_by || item.provenance.actor || "Not recorded"}</p>
        {item.provenance.last_edited_at && <p>Edited: {new Date(item.provenance.last_edited_at).toLocaleString()}</p>}
        <WorkSources sources={item.provenance.sources} />
      </details>
    </section>
    {retire && <div className="report-notice"><p>Retire {item.title}? Existing references must be resolved first.</p>
      <button type="button" className="btn btn-small" disabled={busy} onClick={() => void mutateTree(true)}>Confirm retirement</button>
      <button type="button" className="btn btn-small" disabled={busy} onClick={() => setRetire(false)}>Cancel retirement</button></div>}
    {split && <section className="report-section"><h5>Split work item</h5><p>The parent becomes a group. Evidence remains linked; room-layout answers require explicit allocation.</p>
      <form className="package-scope-form" onSubmit={e => { e.preventDefault(); void mutateTree(false); }}><fieldset disabled={busy}>
        {split.children.map((c, i) => <div key={i}>
          <label htmlFor={`child-title-${i}`}>Child {i + 1} title</label><input id={`child-title-${i}`} required maxLength={200} value={c.title} onChange={e => setChild(i, { title: e.target.value })} />
          <label htmlFor={`child-part-${i}`}>Child {i + 1} part</label><select id={`child-part-${i}`} value={c.part_id} onChange={e => setChild(i, { part_id: e.target.value })}>{parts.map(p => <option key={p.id} value={p.id}>{p.label}</option>)}</select>
          <label htmlFor={`child-action-${i}`}>Child {i + 1} action</label><select id={`child-action-${i}`} value={c.action} onChange={e => setChild(i, { action: e.target.value })}>{WORK_ACTIONS.map(a => <option key={a}>{a}</option>)}</select>
        </div>)}
        <button type="button" className="btn btn-small" disabled={split.children.length >= 100} onClick={() => setSplit({ ...split, children: [...split.children, { title: "", part_id: split.base.part_id, action: split.base.action }] })}>Add child</button>
        <button className="btn btn-small">Save work split</button>
        <button type="button" className="btn btn-small" onClick={() => setSplit(null)}>Cancel split</button>
      </fieldset></form></section>}
    {editor && <section className="report-section" aria-label="Work correction"><h5>Correct work item</h5><p className="report-muted">Saving records your planning correction. It does not verify the work or its requirements.</p>
      {stale && <div className="report-notice"><strong>Saved work changed</strong><p>Your unsaved correction is below. Compare it with the saved work above.</p>
        <button type="button" className="btn btn-small" disabled={busy} onClick={() => setEditor({ ...editor, base: item })}>Keep my correction against this version</button></div>}
      <form className="package-scope-form" onSubmit={e => { e.preventDefault(); void save(); }}><fieldset disabled={busy}>
        <label htmlFor="work-title">Work title</label><input id="work-title" required maxLength={200} value={editor.title} onChange={e => setEditor({ ...editor, title: e.target.value })} />
        <label htmlFor="work-action">Action</label><select id="work-action" value={editor.action} onChange={e => setEditor({ ...editor, action: e.target.value })}>{WORK_ACTIONS.map(action => <option key={action} value={action}>{action}</option>)}</select>
        <label htmlFor="work-inclusion">Inclusion</label><select id="work-inclusion" value={editor.inclusion} onChange={e => setEditor({ ...editor, inclusion: e.target.value })}><option value="included">Included</option><option value="excluded">Excluded</option></select>
        {!editor.base.is_group && layoutEligible(editor.base.system_id, editor.action) && <><label htmlFor="work-layout">Does this work change room or space boundaries?</label><select id="work-layout" value={editor.layout} onChange={e => setEditor({ ...editor, layout: e.target.value as Editor["layout"] })}><option value="unknown">Not recorded</option><option value="yes">Yes</option><option value="no">No</option></select><p className="report-muted">Replacing linings without moving boundaries is No. Yes requests coordination; it does not establish a design outcome.</p></>}
        <label htmlFor="work-condition-note">Condition note</label><input id="work-condition-note" maxLength={120} value={editor.note} onChange={e => setEditor({ ...editor, note: e.target.value })} />
        <label htmlFor="work-target">Target wording</label><textarea id="work-target" rows={3} maxLength={1000} value={editor.target} onChange={e => setEditor({ ...editor, target: e.target.value })} />
        <div className="report-actions"><button className="btn btn-small" disabled={busy || stale || !editor.title.trim()}>{busy ? "Saving…" : "Save correction"}</button><button type="button" className="btn btn-small" disabled={busy} onClick={() => setEditor(null)}>Cancel correction</button></div>
      </fieldset></form>
    </section>}
  </div>;
}

export function WorkSources({ sources }: { sources: unknown }) {
  if (!Array.isArray(sources) || !sources.length) return <p>No direct document excerpts recorded on this item.</p>;
  return <ul>{sources.map((source, index) => {
    const s = source && typeof source === "object" ? source as Record<string, unknown> : {};
    return <li key={index}><p>{typeof s.filename === "string" ? s.filename : "Document source"}{typeof s.page === "number" && s.page > 0 ? ` · page ${s.page}` : ""}</p><p className="report-wording">{typeof s.excerpt === "string" ? s.excerpt : "Excerpt not recorded"}</p></li>;
  })}</ul>;
}
