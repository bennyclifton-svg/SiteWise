import { Pending } from "./Activity";
// The one-page comparison report: price, non-price, recommendation. It
// prints on A4 landscape. Ratings and the recommendation are the PM's; the
// passages are what Jev picked from each proposal; nothing scores a firm.
import { useCallback, useEffect, useState } from "react";
import { ApiError } from "./api";
import { fmt } from "./CostGrid";
import { submissionApi, type ReportData, type PriceTable } from "./submissionApi";

const money = (m: string | null | undefined) => (m == null ? "—" : `$${fmt(m)}`);
const RATING: Record<string, string> = { strong: "Strong", acceptable: "Acceptable", concern: "Concern" };

export function ProcurementReport({ projectId, table: initial, onBack, onSignedOut }: { projectId: string; table: PriceTable; onBack: () => void; onSignedOut: () => void }) {
  const pkg = initial.package_id;
  const [data, setData] = useState<ReportData | null>(null);
  const [text, setText] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const fail = useCallback((e: unknown) => { if (e instanceof ApiError && e.status === 401) onSignedOut(); else setError("Could not save. Try again."); }, [onSignedOut]);
  useEffect(() => {
    submissionApi.report(projectId, pkg).then(d => { setData(d); setText(d.report.recommendation || d.report.draft); }).catch(fail);
  }, [projectId, pkg, fail]);
  async function run(action: () => Promise<ReportData>) {
    setBusy(true); setError("");
    try { const d = await action(); setData(d); } catch (e) { fail(e); } finally { setBusy(false); }
  }
  if (!data) return <section className="prices"><button type="button" className="btn btn-small" onClick={onBack}>← Price table</button>{error ? <p role="alert">{error}</p> : <p role="status"><Pending delayed>Loading report…</Pending></p>}</section>;
  const { table, report } = data;
  const cols = table.columns;
  const quoted = !table.builder && cols.some(c => c.quote);
  const base = table.items.filter(i => i.base && i.kind !== "rate" && i.kind !== "criterion" && (!quoted || i.key === "comparison:appointment"));
  const name = (i: number) => cols[i].firm?.name ?? cols[i].submission.filename;
  const departures = report.criteria.find(c => c.key === "criterion:r6");
  const criteria = report.criteria.filter(c => c.key !== "criterion:r6");
  const signed = report.signed_off_by && report.recommendation === text;

  return <section className="prices procurement-report" aria-label="Comparison report">
    <header className="prices-head no-print">
      <button type="button" className="btn btn-small" onClick={onBack}>← Price table</button>
      <div className="prices-actions"><button type="button" className="btn btn-small" onClick={() => window.print()}>Print</button></div>
    </header>
    {error && <div className="banner" role="alert"><p>{error}</p></div>}
    <div className="report-page">
      <header className="report-page-head">
        <h1>{table.title}: comparison of {cols.length} proposal{cols.length === 1 ? "" : "s"}</h1>
        <p>{cols.map((c, i) => `${name(i)} (received ${c.submission.received_on})`).join(" · ")} · prepared {new Date().toLocaleDateString("en-AU", { day: "numeric", month: "long", year: "numeric" })}</p>
      </header>
      <h2>Price <span>ex GST</span></h2>
      <table className="report-grid">
        <thead><tr><th scope="col">Item</th>{cols.map((c, i) => <th scope="col" key={c.submission.id} className={c.totals.lowest ? "lowest" : undefined}>{name(i)}</th>)}</tr></thead>
        <tbody>
          {base.map(it => <tr key={it.key}><th scope="row">{it.label}</th>{cols.map(c => { const cell = c.cells[it.key]; return <td key={c.submission.id}>{!cell || cell.state === "" ? "Not read" : cell.state === "priced" ? money(cell.amount) : cell.state === "included" ? "Included" : cell.state === "excluded" ? "Excluded" : "Not priced"}{cell?.basis && <small className="prices-basis">{cell.basis}</small>}</td>; })}</tr>)}
          <tr className="report-total"><th scope="row">{quoted ? "Confirmed comparison fee" : "Base fee total"}</th>{cols.map(c => <td key={c.submission.id} className={c.totals.lowest ? "lowest" : undefined}>{c.totals.base_total != null ? money(c.totals.base_total) : c.totals.known_base && c.totals.known_base !== "0.00" ? `${money(c.totals.known_base)} known · incomplete` : "Unknown"}{c.totals.not_priced > 0 && <small> · {c.totals.not_priced} not priced</small>}</td>)}</tr>
          {!quoted && <tr><th scope="row">Options priced</th>{cols.map(c => <td key={c.submission.id}>{money(c.totals.options_total)}</td>)}</tr>}
        </tbody>
      </table>
      <h2>Non-price <span>passages as the firms wrote them; ratings are the PM's</span></h2>
      <table className="report-grid report-nonprice">
        <thead><tr><th scope="col">Criterion</th>{cols.map((c, i) => <th scope="col" key={c.submission.id}>{name(i)}</th>)}</tr></thead>
        <tbody>
          {criteria.map(cr => <tr key={cr.key}><th scope="row">{cr.label}</th>{cols.map(c => {
            const resp = report.columns.find(r => r.submission_id === c.submission.id)?.responses[cr.key];
            return <td key={c.submission.id}>
              {resp?.passages.length ? resp.passages.slice(0, 1).map(p => <p key={p.ordinal} className="report-passage"><strong>{p.heading}</strong>{p.page ? ` (p${p.page})` : ""}: {p.excerpt.replace(p.heading, "").trim().slice(0, 220)}{p.excerpt.length > 220 ? "…" : ""}</p>) : <p className="report-passage muted">No section answers this.</p>}
              <select aria-label={`Rating for ${cr.label}, ${c.firm?.name ?? ""}`} value={resp?.rating ?? ""} disabled={busy} onChange={e => void run(() => submissionApi.rate(projectId, pkg, c.submission.id, cr.key, e.target.value))}>
                <option value="">Not rated</option><option value="strong">Strong</option><option value="acceptable">Acceptable</option><option value="concern">Concern</option>
              </select>
              {resp?.rating && <span className="print-only">{RATING[resp.rating]}</span>}
            </td>;
          })}</tr>)}
          {departures && <tr><th scope="row">Departures, exclusions and assumptions</th>{cols.map(c => {
            const resp = report.columns.find(r => r.submission_id === c.submission.id)?.responses[departures.key];
            return <td key={c.submission.id}>{resp?.passages.length ? resp.passages.slice(0, 3).map(p => <p key={p.ordinal} className="report-passage"><strong>{p.heading}</strong>: {p.excerpt.replace(p.heading, "").trim().slice(0, 160)}…</p>) : <p className="report-passage muted">None found.</p>}</td>;
          })}</tr>}
        </tbody>
      </table>
      <h2>Recommendation</h2>
      <textarea className="no-print report-recommendation" rows={6} value={text} onChange={e => setText(e.target.value)} disabled={busy} aria-label="Recommendation" />
      <p className="print-only report-recommendation-print">{text}</p>
      <p className="report-signoff">{report.recommendation ? `Drafted from the price table, edited by ${report.edited_by || "the PM"}` : "Drafted from the price table; not yet edited."}{report.signed_off_by && report.signed_off_at ? ` · Signed off by ${report.signed_off_by} on ${new Date(report.signed_off_at).toLocaleDateString("en-AU")}` : ""}</p>
      <div className="no-print prices-actions">
        <button type="button" className="btn btn-small" disabled={busy || text === report.recommendation} onClick={() => void run(() => submissionApi.recommend(projectId, pkg, text, false))}>Save</button>
        <button type="button" className="btn btn-small btn-primary" disabled={busy || !text.trim() || text.includes("[state the firm") || !!signed} onClick={() => void run(() => submissionApi.recommend(projectId, pkg, text, true))}>{signed ? "Signed off" : "Save and sign off"}</button>
      </div>
    </div>
  </section>;
}
