import { useEffect, useState } from "react";
import { ApiError } from "./api";
import { fmt } from "./CostGrid";
import { clickRow, emptySelection, prune, selectAll } from "./table/selection";
import { revealQuietly, showingLine, useChatTable } from "./chatView";
import { invoiceApi, type Contract, type Ledger, type OpCall, type Variation, type VariationStatus } from "./invoiceApi";

// The Cost Plan's "Variations and claims" tab
// (docs/plans/2026-11-17-variations-progress-claims.md, Task 10). Every
// figure is the server's: the browser never sums money. Every write goes
// through POST /ops, so each has one Undo; approving waits for nothing here
// because pressing Approve is the PM's own click.

const STATUS: Record<VariationStatus, string> = { proposed: "Proposed", submitted: "Submitted", approved: "Approved", rejected: "Rejected", withdrawn: "Withdrawn" };
const money = (m: string | null | undefined) => fmt(m, false);
const plain = (s: string) => s.replace(/[,$\s]/g, "");

export function VariationsTab({ projectId, ledger, reload, onSignedOut }: { projectId: string; ledger: Ledger | null; reload: () => void; onSignedOut: () => void }) {
  const contracts = ledger?.contracts ?? [];
  const [chosen, setChosen] = useState("");
  const contract = contracts.find(c => c.package_id === chosen) ?? contracts[0];
  const [busy, setBusy] = useState(false), [error, setError] = useState("");
  const [last, setLast] = useState<{ change: string; label: string } | null>(null);
  useEffect(() => { if (chosen && !contracts.some(c => c.package_id === chosen)) setChosen(""); }, [contracts, chosen]);

  async function apply(area: "variations" | "claims", ops: OpCall[], label: string) {
    if (busy) return false;
    setBusy(true); setError("");
    try {
      const out = await invoiceApi.ops(projectId, area, ops);
      setLast(out.change_id ? { change: out.change_id, label } : null);
      reload();
      return true;
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) { onSignedOut(); return false; }
      setError(e instanceof ApiError && e.message ? e.message : "Could not save. Try again.");
      return false;
    } finally { setBusy(false); }
  }

  if (!ledger) return <p role="status">Loading…</p>;
  if (!contract) return <section className="inv-empty" aria-label="Variations and claims"><h2>No live contracts yet</h2><p>Variations and progress claims sit on an awarded contract. Award a package in Procurement and it appears here.</p></section>;
  return <section className="variations" aria-label="Variations and claims" aria-busy={busy}>
    {contracts.length > 1 && <div className="costs-toolbar"><label className="var-contract">Contract <select value={contract.package_id} onChange={e => setChosen(e.target.value)}>
      {contracts.map(c => <option key={c.package_id} value={c.package_id}>{c.title}{c.firm ? ` · ${c.firm}` : ""}</option>)}</select></label></div>}
    <ContractHeader contract={contract} />
    {error && <p role="alert" className="inv-error">{error}</p>}
    {last && <p className="reg-notice" role="status">{last.label} <button type="button" className="cell-link" disabled={busy} onClick={() => { const c = last.change; setLast(null); void invoiceApi.undo(projectId, c).then(reload).catch(() => setError("Can't undo: this has changed since.")); }}>Undo</button></p>}
    <VariationTable contract={contract} variations={(ledger.variations ?? []).filter(v => v.package_id === contract.package_id)} busy={busy} apply={apply} />
  </section>;
}

function ContractHeader({ contract }: { contract: Contract }) {
  return <dl className="var-head" aria-label={`${contract.title} figures`}>
    <div><dt>Contract</dt><dd>{contract.title}{contract.firm ? ` · ${contract.firm}` : ""}</dd></div>
    <div><dt>Contract sum</dt><dd className="num">{money(contract.contract_sum)}</dd></div>
    <div><dt>Approved variations</dt><dd className="num">{money(contract.approved_variations)}</dd></div>
    <div><dt>Contract sum to date</dt><dd className="num">{money(contract.contract_sum_to_date)}</dd></div>
  </dl>;
}

type Apply = (area: "variations" | "claims", ops: OpCall[], label: string) => Promise<boolean>;

function VariationTable({ contract, variations, busy, apply }: { contract: Contract; variations: Variation[]; busy: boolean; apply: Apply }) {
  const [selection, setSelection] = useState(emptySelection);
  const [only, setOnly] = useState<ReadonlySet<string> | null>(null), [onlyShown, setOnlyShown] = useState("");
  const [draft, setDraft] = useState({ title: "", claimed: "", line: "" });
  const ids = variations.map(v => v.id);
  useEffect(() => setSelection(s => prune(s, ids)), [ids.join(",")]); // eslint-disable-line react-hooks/exhaustive-deps
  const shown = only ? variations.filter(v => only.has(v.id)) : variations;
  useChatTable("variations", "costs", {
    select: list => { const keep = list.filter(id => ids.includes(id)); setOnly(null); setSelection(selectAll(keep)); if (keep[0]) requestAnimationFrame(() => revealQuietly(document.querySelector(`tr[data-variation="${CSS.escape(keep[0])}"]`))); },
    only: (list, words) => { setOnly(new Set(list.filter(id => ids.includes(id)))); setOnlyShown(words ?? ""); },
    clear: () => setOnly(null),
  }, selection.ids);
  const edit = (v: Variation, fields: Record<string, unknown>, label: string) => apply("variations", [{ op: "variations.edit", fields: { variation: v.id, ...fields } }], label);
  function decide(v: Variation, status: string) {
    if (status === v.status) return;
    if (status === "approved") {
      const amount = window.prompt(`Approve ${v.number} at what amount, ex GST?`, v.estimate ?? v.claimed ?? "");
      if (amount == null || !plain(amount)) return;
      void apply("variations", [{ op: "variations.decide", fields: { variation: v.id, decision: "approve", approved: plain(amount) } }], `Approved ${v.number}.`);
      return;
    }
    if (status === "rejected") { void apply("variations", [{ op: "variations.decide", fields: { variation: v.id, decision: "reject" } }], `Rejected ${v.number}.`); return; }
    void edit(v, { status }, `${v.number} is ${STATUS[status as VariationStatus].toLowerCase()}.`);
  }
  async function add() {
    if (!draft.title.trim()) return;
    const fields: Record<string, unknown> = { package: contract.package_id, title: draft.title.trim() };
    if (plain(draft.claimed)) fields.claimed = plain(draft.claimed);
    if (draft.line) fields.line = draft.line;
    if (await apply("variations", [{ op: "variations.add", fields }], `Recorded "${draft.title.trim()}".`)) setDraft({ title: "", claimed: "", line: "" });
  }
  const cellInput = (v: Variation, field: "claimed" | "estimate" | "time_claimed_days", value: string, label: string) =>
    <input className="field-plain num" aria-label={`${label} for ${v.number}`} inputMode="decimal" key={`${v.id}:${field}:${value}`} defaultValue={field === "time_claimed_days" ? value : value ? money(value) : ""} disabled={busy || v.status === "approved" && field !== "time_claimed_days"}
      onBlur={e => { const next = plain(e.target.value); const was = field === "time_claimed_days" ? value : value ? plain(money(value)) : ""; if (next !== was) void edit(v, { [field]: next }, `Changed ${v.number}.`); }}
      onKeyDown={e => { if (e.key === "Enter") (e.target as HTMLInputElement).blur(); }} />;
  return <>
    <h2 className="var-title">Variations</h2>
    {only && <p className="reg-filter">{showingLine("variations", only.size, onlyShown)} · <button type="button" className="cell-link" onClick={() => setOnly(null)}>Clear</button></p>}
    <div className="inv-table-scroll"><table className="inv-table var-table" onKeyDown={e => { if (e.key === "Escape") { setSelection(emptySelection); setOnly(null); } }}>
      <caption className="sr-only">Variations on {contract.title}</caption>
      <thead><tr><th scope="col">No.</th><th scope="col">Description</th>{contract.lines.length > 1 && <th scope="col">Line</th>}<th scope="col">Status</th><th scope="col" className="num">Claimed</th><th scope="col" className="num">PM's estimate</th><th scope="col" className="num">Approved</th><th scope="col" className="num">Days</th><th scope="col">Documents</th></tr></thead>
      <tbody>
        {shown.map(v => <tr key={v.id} data-variation={v.id} aria-selected={selection.ids.includes(v.id)} onClick={e => { if (!(e.target as HTMLElement).closest("input,select,button")) setSelection(s => clickRow(s, v.id, ids, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })); }}>
          <td>{v.number}</td>
          <td><input className="field-plain" aria-label={`Description of ${v.number}`} key={`${v.id}:t:${v.title}`} defaultValue={v.title} disabled={busy} maxLength={200} onBlur={e => { const t = e.target.value.trim(); if (t && t !== v.title) void edit(v, { title: t }, `Changed ${v.number}.`); }} onKeyDown={e => { if (e.key === "Enter") (e.target as HTMLInputElement).blur(); }} /></td>
          {contract.lines.length > 1 && <td><select className="field-plain" aria-label={`Line for ${v.number}`} value={v.line_id} disabled={busy} onChange={e => void edit(v, { line: e.target.value }, `Moved ${v.number}.`)}>{contract.lines.map(l => <option key={l.id} value={l.id}>{l.label}</option>)}</select></td>}
          <td><select className="field-plain" aria-label={`Status of ${v.number}`} value={v.status} disabled={busy} onChange={e => decide(v, e.target.value)}>{(Object.keys(STATUS) as VariationStatus[]).map(s => <option key={s} value={s}>{STATUS[s]}</option>)}</select></td>
          <td className="num">{cellInput(v, "claimed", v.claimed ?? "", "Claimed")}</td>
          <td className="num">{cellInput(v, "estimate", v.estimate ?? "", "PM's estimate")}{v.forecast_basis === "claimed" && <span className="cost-tag" title="No estimate yet: the forecast uses the claimed amount">claimed</span>}</td>
          <td className="num">{money(v.approved)}</td>
          <td className="num">{cellInput(v, "time_claimed_days", v.time_claimed_days == null ? "" : String(v.time_claimed_days), "Days claimed")}</td>
          <td>{v.documents.length ? v.documents.map(d => d.name || "Document").join(", ") : "—"}</td>
        </tr>)}
        <tr className="cost-ghost">
          <td>New</td>
          <td><input aria-label="New variation description" placeholder="Describe a variation" value={draft.title} disabled={busy} onChange={e => setDraft({ ...draft, title: e.target.value })} onKeyDown={e => { if (e.key === "Enter") void add(); }} /></td>
          {contract.lines.length > 1 && <td><select aria-label="New variation line" value={draft.line} onChange={e => setDraft({ ...draft, line: e.target.value })}><option value="">line</option>{contract.lines.map(l => <option key={l.id} value={l.id}>{l.label}</option>)}</select></td>}
          <td>Submitted</td>
          <td className="num"><input aria-label="New variation claimed amount" inputMode="decimal" placeholder="claimed" value={draft.claimed} disabled={busy} onChange={e => setDraft({ ...draft, claimed: e.target.value })} onKeyDown={e => { if (e.key === "Enter") void add(); }} /></td>
          <td /><td /><td /><td />
        </tr>
      </tbody>
    </table></div>
  </>;
}
