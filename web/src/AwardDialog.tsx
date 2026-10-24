import { Pending } from "./Activity";
// Award in two steps. Step 1: choose what goes into the cost plan: each
// priced item is its own line by default, extras are left out, and rows can
// be combined under one name. Step 2: confirm the parent and children the
// server previews. Nothing is written until the PM confirms, and every
// figure, the running total included, is computed by the server. When the
// server offers the letter's stages (they add to the confirmed fee), the PM
// can write them as separate lines instead of the one total.
import { useEffect, useMemo, useRef, useState } from "react";
import { ApiError } from "./api";
import { fmt } from "./CostGrid";
import { submissionApi, type Award, type AwardRequest, type PriceTable } from "./submissionApi";

const money = (m: string | null | undefined) => (m == null ? "—" : `$${fmt(m)}`);

interface Row { id: string; key: string; priceId: string; quoteId: string; label: string; amount: string | null; extra: boolean }
type Mode = "stages" | "total";
interface Line { label: string; rows: string[] }

export function AwardDialog({ projectId, table, submissionId, onClose, onAwarded, onSignedOut }: {
  projectId: string; table: PriceTable; submissionId: string; onClose: () => void; onAwarded: (t: PriceTable) => void; onSignedOut: () => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  const col = table.columns.find(c => c.submission.id === submissionId)!;
  const stages = table.builder ? [] : col.award_stages ?? [];
  const [mode, setMode] = useState<Mode>(stages.length ? "stages" : "total");
  const rowsFor = (m: Mode): Row[] => {
    if (m === "stages") return stages.map(f => ({ id: `q:${f.id}`, key: "", priceId: "", quoteId: f.id, label: f.label, amount: f.amount, extra: false }));
    const out: Row[] = [];
    for (const it of table.items) {
      if ((col.quote || col.cells["comparison:appointment"]) && !table.builder && it.key !== "comparison:appointment") continue;
      const c = col.cells[it.key];
      // The comparison fee lands in the cost plan as the firm's professional
      // fee; "Confirmed comparison fee" is a procurement word, not a cost line.
      const label = it.key === "comparison:appointment" ? "Professional fee" : it.label;
      if (it.kind !== "rate" && it.kind !== "criterion" && c?.state === "priced" && c.amount != null) out.push({ id: `k:${it.key}`, key: it.key, priceId: "", quoteId: "", label, amount: c.amount, extra: false });
    }
    if (!table.builder && !col.quote && !col.cells["comparison:appointment"]) for (const o of col.others) if (o.amount != null) out.push({ id: `p:${o.id}`, key: "", priceId: o.id, quoteId: "", label: o.label || "Offered item", amount: o.amount, extra: true });
    return out;
  };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const rows = useMemo(() => rowsFor(mode), [table, col, mode]);
  const defaultLines = (rs: Row[]): Line[] => rs.filter(r => !r.extra).map(r => ({ label: r.label, rows: [r.id] }));
  const [lines, setLines] = useState<Line[]>(() => defaultLines(rows));
  const [picked, setPicked] = useState<Set<string>>(new Set());
  const [combineName, setCombineName] = useState("");
  const [preview, setPreview] = useState<Award | null>(null);
  const [step, setStep] = useState<1 | 2>(1);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => { const d = ref.current; if (d && !d.open) d.showModal(); return () => d?.close(); }, []);

  const byId = useMemo(() => new Map(rows.map(r => [r.id, r])), [rows]);
  const lineOf = (id: string) => lines.findIndex(l => l.rows.includes(id));
  const request = (): AwardRequest => ({
    submission_id: submissionId, total: "",
    lines: lines.filter(l => l.rows.length).map(l => ({ label: l.label.trim(), item_keys: l.rows.map(id => byId.get(id)!.key).filter(Boolean), price_ids: l.rows.map(id => byId.get(id)!.priceId).filter(Boolean), quote_ids: l.rows.map(id => byId.get(id)!.quoteId).filter(Boolean) })),
  });

  // The running total is the server's, refreshed as lines change.
  useEffect(() => {
    let stopped = false;
    const req = request();
    if (!req.lines.length || req.lines.some(l => !l.label)) { setPreview(null); return; }
    const t = setTimeout(() => {
      submissionApi.previewAward(projectId, table.package_id, req).then(p => { if (!stopped) { setPreview(p); setError(""); } })
        .catch(e => { if (stopped) return; if (e instanceof ApiError && e.status === 401) onSignedOut(); else setError("These lines cannot be awarded as they are. Check each line has a name and priced items."); });
    }, 120);
    return () => { stopped = true; clearTimeout(t); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lines]);

  function switchMode(m: Mode) {
    if (m === mode) return;
    setMode(m); setLines(defaultLines(rowsFor(m))); setPicked(new Set()); setCombineName(""); setPreview(null);
  }
  // By stage every stage must be written, so a combined stage goes back to
  // its own line instead of out of the award.
  function separate(id: string) {
    const i = lineOf(id);
    const rest = lines.map((l, n) => n === i ? { ...l, rows: l.rows.filter(r => r !== id) } : l).filter(l => l.rows.length);
    setLines([...rest, { label: byId.get(id)!.label, rows: [id] }]);
  }
  function toggleInclude(id: string) {
    const i = lineOf(id);
    if (i >= 0) setLines(lines.map((l, n) => n === i ? { ...l, rows: l.rows.filter(r => r !== id) } : l).filter(l => l.rows.length));
    else setLines([...lines, { label: byId.get(id)!.label, rows: [id] }]);
  }
  function combine() {
    const ids = rows.map(r => r.id).filter(id => picked.has(id));
    if (ids.length < 2 || !combineName.trim()) return;
    const kept = lines.map(l => ({ ...l, rows: l.rows.filter(r => !picked.has(r)) })).filter(l => l.rows.length);
    const first = Math.min(...ids.map(lineOf).filter(i => i >= 0), kept.length);
    kept.splice(Math.min(first, kept.length), 0, { label: combineName.trim(), rows: ids });
    setLines(kept); setPicked(new Set()); setCombineName("");
  }
  function rename(i: number, label: string) { setLines(lines.map((l, n) => n === i ? { ...l, label } : l)); }

  async function confirm() {
    if (!preview) return;
    setBusy(true); setError("");
    try { onAwarded(await submissionApi.award(projectId, table.package_id, { ...request(), total: preview.total ?? "" })); }
    catch (e) { if (e instanceof ApiError && e.status === 401) onSignedOut(); else setError(e instanceof ApiError && e.status === 409 ? "This package's cost line is already split or awarded. Open the cost plan to check it." : "Could not award. Try again."); }
    finally { setBusy(false); }
  }

  const firm = col.firm?.name ?? col.submission.filename;
  return <dialog ref={ref} className="award" aria-labelledby="award-title" onCancel={e => { e.preventDefault(); if (!busy) onClose(); }}>
    <h2 id="award-title">Award {table.title} to {firm}</h2>
    <p className="award-step">Step {step} of 2 · {step === 1 ? "Choose what goes into the cost plan" : "Confirm the cost plan lines"}</p>
    {error && <p className="award-error" role="alert">{error}</p>}
    {step === 1 ? <>
      {stages.length > 0 && <fieldset className="award-mode">
        <legend>Write to the cost plan as</legend>
        <label><input type="radio" name="award-mode" checked={mode === "stages"} onChange={() => switchMode("stages")} /> By stage: {stages.length} lines from the letter</label>
        <label><input type="radio" name="award-mode" checked={mode === "total"} onChange={() => switchMode("total")} /> One total</label>
        {mode === "stages" && <p className="award-muted">These stages add up to the confirmed fee of {money(col.cells["comparison:appointment"]?.amount)}. The fee total line in the letter is not added again.</p>}
      </fieldset>}
      <table className="award-table">
        <thead><tr><th scope="col"><span className="sr-only">Select to combine</span></th><th scope="col">Item the firm priced</th><th scope="col" className="num">Amount ex GST</th><th scope="col">Cost plan line</th></tr></thead>
        <tbody>{rows.map(r => {
          const i = lineOf(r.id);
          const line = i >= 0 ? lines[i] : null;
          const first = line && line.rows[0] === r.id;
          return <tr key={r.id} className={line ? undefined : "award-out"}>
            <td><input type="checkbox" aria-label={`Select ${r.label} to combine`} checked={picked.has(r.id)} disabled={!line} onChange={() => { const s = new Set(picked); if (s.has(r.id)) s.delete(r.id); else s.add(r.id); setPicked(s); }} /></td>
            <td>{r.label}{r.extra && <span className="award-tag">awaiting alignment</span>}</td>
            <td className="num">{money(r.amount)}</td>
            <td>{mode === "stages" && line ? (first
              ? <span className="award-line"><input aria-label={`Line name for ${r.label}`} value={line.label} maxLength={200} onChange={e => rename(i, e.target.value)} />{line.rows.length > 1 && <><span className="award-tag">{line.rows.length} stages</span><button type="button" className="table-action" onClick={() => separate(r.id)}>Separate</button></>}</span>
              : <span className="award-in">in “{line.label}” <button type="button" className="table-action" onClick={() => separate(r.id)}>Separate</button></span>)
              : !line ? <button type="button" className="table-action" onClick={() => toggleInclude(r.id)}>Include</button>
              : first ? <span className="award-line"><input aria-label={`Line name for ${r.label}`} value={line.label} maxLength={200} onChange={e => rename(i, e.target.value)} />{line.rows.length > 1 && <span className="award-tag">{line.rows.length} items</span>}<button type="button" className="table-action" onClick={() => toggleInclude(r.id)}>Leave out</button></span>
              : <span className="award-in">in “{line.label}” <button type="button" className="table-action" onClick={() => toggleInclude(r.id)}>Leave out</button></span>}</td>
          </tr>;
        })}</tbody>
      </table>
      <div className="award-combine">
        <input aria-label="Name for combined line" placeholder="Name for the combined line, e.g. Design development and documentation" value={combineName} onChange={e => setCombineName(e.target.value)} />
        <button type="button" className="btn btn-small" disabled={picked.size < 2 || !combineName.trim()} onClick={combine}>Combine {picked.size || ""} selected</button>
      </div>
      <p className="award-total">To be written: <strong>{preview ? money(preview.total) : "—"}</strong> ex GST in {lines.length} line{lines.length === 1 ? "" : "s"}.</p>
      <div className="award-actions">
        <button type="button" className="btn" onClick={onClose}>Cancel</button>
        <button type="button" className="btn btn-primary" disabled={!preview} onClick={() => setStep(2)}>Next: review</button>
      </div>
    </> : preview && <>
      <div className="award-preview">
        <p><strong>{preview.parent_label}</strong> · contract {money(preview.total)} · budget {preview.budget != null ? `${money(preview.budget)} (unchanged)` : "not set, so no variance is shown"}</p>
        <ul>{preview.lines.map(l => <li key={l.label}><span>{l.label}</span><span className="num">{money(l.amount)}</span></li>)}</ul>
        <p className="award-muted">{table.columns.length > 1 ? `The other ${table.columns.length - 1} proposal${table.columns.length === 2 ? " is" : "s are"} marked unsuccessful and kept unchanged. ` : ""}The package is appointed to {firm}; invoices from its ABN will find these lines. You can undo this until an invoice is allocated to one of them.</p>
      </div>
      <div className="award-actions">
        <button type="button" className="btn" disabled={busy} onClick={() => setStep(1)}>Back</button>
        <button type="button" className="btn btn-primary" disabled={busy} onClick={() => void confirm()}>{busy ? <Pending delayed>Awarding…</Pending> : "Award and write to cost plan"}</button>
      </div>
    </>}
  </dialog>;
}
