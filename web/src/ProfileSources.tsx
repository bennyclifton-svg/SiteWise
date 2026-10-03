import { useEffect, useState } from "react";
import { profileApi, type Profile, type SourceRecords } from "./profileApi";

export function ProfileSources({ projectId, data, system, onSystem, open, onOpen }: {
  projectId: string; data: Profile; system: string; onSystem: (value: string) => void;
  open: boolean; onOpen: (value: boolean) => void;
}) {
  const [outcome, setOutcome] = useState("");
  const [offset, setOffset] = useState(0);
  const [result, setResult] = useState<SourceRecords | null>(null);
  const [error, setError] = useState("");
  const [retry, setRetry] = useState(0);
  const coverage = data.coverage ?? [];
  const total = coverage.reduce((n, d) => n + d.units, 0);
  const read = coverage.reduce((n, d) => n + d.labelled, 0);
  const unresolved = coverage.reduce((n, d) => n + d.needs_mapping, 0);
  useEffect(() => { setOffset(0); }, [projectId, system, outcome]);
  useEffect(() => {
    if (!open) return;
    let live = true;
    setResult(null); setError("");
    profileApi.sources(projectId, system, outcome, offset).then(
      r => { if (live) setResult(r); },
      e => { if (live) setError(e instanceof Error ? e.message : "Could not load source requirements."); },
    );
    return () => { live = false; };
  }, [projectId, system, outcome, offset, open, retry, data.built_at, data.pending_documents]);
  const labels = new Map(data.systems.flatMap(g => [[g.id, g.label], ...g.rows.map(r => [r.leaf, r.label])]) as [string, string][]);
  return <section className="pf-section pf-sources" id="profile-sources" aria-labelledby="pf-source-heading">
    <h2 id="pf-source-heading"><button type="button" className="cell-link" aria-expanded={open} onClick={() => onOpen(!open)}>Source requirements · {open ? "Hide" : "View"}</button></h2>
    <p className="muted">{read} of {total} passages classified{unresolved > 0 ? ` · ${unresolved} need mapping or review` : ""}. Classification does not confirm accuracy.</p>
    {open && <>
      <details className="pf-more"><summary>Document coverage</summary>
        <ul className="pf-coverage">{coverage.map(d => <li key={d.document_id}>
          <strong>{d.filename}</strong><br />
          {!d.current ? "Update the profile to prepare complete source records." : `${d.pages > 0 ? `${d.pages} pages extracted · ` : ""}${d.labelled} of ${d.units} passages classified`}
          {d.empty_pages.length > 0 && <span className="pf-source-warning"> · Check pages {d.empty_pages.join(", ")}: no extractable text</span>}
        </li>)}</ul>
      </details>
      <div className="pf-source-filters">
        <label>System<select className="select" value={system} onChange={e => onSystem(e.target.value)}>
          <option value="">All systems and project facts</option>
          {data.systems.map(g => <optgroup key={g.id} label={g.label}>
            <option value={g.id}>All {g.label.toLowerCase()}</option>
            {g.rows.map(r => <option key={r.leaf} value={r.leaf}>{r.label}</option>)}
          </optgroup>)}
        </select></label>
        <label>Show<select className="select" value={outcome} onChange={e => setOutcome(e.target.value)}>
          <option value="">All source passages</option><option value="mapped">Mapped</option>
          <option value="needs_mapping">Needs mapping or review</option><option value="background">Background and references</option>
        </select></label>
      </div>
      {error ? <p role="alert">{error} <button className="cell-link" onClick={() => setRetry(n => n + 1)}>Retry</button></p> : !result ? <p role="status">Loading source requirements…</p> : <>
        {result.records.length === 0 && <p className="muted">No source passages match this view.</p>}
        <ol className="pf-source-list" start={offset + 1}>{result.records.map(r => <li key={r.id}>
          <div className="pf-source-meta"><a href={`/api/documents/${r.document_id}/file?view=1${r.page ? `#page=${r.page}` : ""}`} target="_blank" rel="noreferrer">{r.filename}{r.location ? ` · ${r.location}` : ""}</a>
            <span>{r.outcome === "pending" ? "Awaiting reading" : r.outcome === "needs_mapping" ? "Needs mapping or review" : categoryLabel(r.category)}</span>
          </div>
          {r.section && <h3>{r.section}</h3>}
          {r.context && <p className="muted pf-source-context">{r.context}</p>}
          <blockquote>{r.text}</blockquote>
          <p className="pf-source-meta">{r.systems.filter(id => id.includes(".")).map(id => labels.get(id) ?? id).join(" · ") || (r.keys.length ? "Project facts" : "No specific system assigned")}
            {r.provider && r.provider !== "not_stated" ? ` · ${categoryLabel(r.provider)}` : ""}
            {r.scope && r.scope !== "not_stated" ? ` · ${categoryLabel(r.scope)}` : ""}
          </p>
          {r.unresolved.length > 0 && <p className="pf-source-warning">Some answers are uncertain or unavailable. The original text is preserved above.</p>}
        </li>)}</ol>
        <nav className="pf-source-pages" aria-label="Source requirement pages">
          <button className="btn btn-small" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - 50))}>Previous</button>
          <span>Page {Math.floor(offset / 50) + 1}</span>
          <button className="btn btn-small" disabled={!result.more} onClick={() => setOffset(offset + 50)}>Next</button>
        </nav>
      </>}
    </>}
  </section>;
}

function categoryLabel(value: string) { return value.replaceAll("_", " ").replace(/^./, s => s.toUpperCase()); }
