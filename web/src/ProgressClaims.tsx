import { useRef, useState, type DragEvent } from "react";
import { ApiError } from "./api";
import { fmt } from "./CostGrid";
import { showingLine, useChatTable } from "./chatView";
import { emptySelection, selectAll } from "./table/selection";
import { invoiceApi, uploadInvoice, type Claim, type Contract, type Invoice, type Ledger } from "./invoiceApi";
import "./payment-schedule.css";

// The Cost Plan's Progress claims tab
// (docs/plans/2026-11-26-payment-schedules.md, section 6.2): one table of
// payment schedules, grouped by live contract. A contract's group is where
// its claim is dropped. Every figure is the server's; a row opens its
// schedule as a sheet.

/** A schedule's money always shows its cents. */
export const cents = (m: string | null | undefined) => m == null ? "" : fmt(m, true);
/** Nil prints as a dash, as a schedule does. */
export const nilDash = (m: string | null | undefined) => m == null ? "" : /[1-9]/.test(m) ? fmt(m, true) : "–";
export const day = (iso: string) => iso ? new Date(`${iso}T00:00:00`).toLocaleDateString("en-AU", { day: "numeric", month: "short", year: "numeric" }) : "";
export const ALERT: Record<string, string> = { overdue: "Past due", soon: "Due soon", check: "To check" };

/** Where a claim's schedule stands, in the list's words. */
export function claimStatus(c: Claim): string {
  if (c.issued.length) {
    // Sent from SiteWise sets the day by rule; Served is the PM's own word.
    if (c.schedule_served_on && c.schedule_served_confirmed) return "Served";
    if (c.schedule_send?.status === "sent") return "Sent";
    if (c.schedule_send?.status === "queued") return "Sending";
    return c.schedule_served_on ? "Served" : "Issued";
  }
  return c.unassessed === 0 && c.missing_reasons === 0 && !c.retention_open ? "Ready to issue" : "Draft";
}

/** What a schedule withholds reads as a deduction; a return reads as an addition. */
function retention(m: string | null): string {
  if (m == null) return "";
  if (!/[1-9]/.test(m)) return "–";
  return m.startsWith("-") ? `+${fmt(m.slice(1), true)}` : `−${fmt(m, true)}`;
}

interface Props {
  projectId: string;
  ledger: Ledger | null;
  reload: () => void;
  onSignedOut: () => void;
  onOpen: (claim: string) => void;
}

export function ProgressClaimsTab({ projectId, ledger, reload, onSignedOut, onOpen }: Props) {
  const [note, setNote] = useState(""), [error, setError] = useState(""), [busy, setBusy] = useState(false);
  const [selection, setSelection] = useState(emptySelection);
  const [only, setOnly] = useState<ReadonlySet<string> | null>(null), [onlyShown, setOnlyShown] = useState("");
  const contracts = (ledger?.contracts ?? []).filter(c => c.claims);
  const claims = ledger?.claims ?? [];
  const ids = claims.map(c => c.invoice_id);
  useChatTable("claims", "costs", {
    select: list => { setOnly(null); setSelection(selectAll(list.filter(id => ids.includes(id)))); },
    only: (list, words) => { setOnly(new Set(list.filter(id => ids.includes(id)))); setOnlyShown(words ?? ""); },
    clear: () => setOnly(null),
    open: id => { if (ids.includes(id)) onOpen(id); },
  }, selection.ids, !!ledger);

  // A file dropped on a contract is that contract's claim, read in about a second.
  function drop(contract: Contract, files: File[]) {
    for (const f of files) {
      setNote(`Reading ${f.name}…`);
      uploadInvoice(projectId, f, () => {}, contract.package_id).done
        .then(r => { setNote(r.created ? `${f.name} is being read.` : `${f.name} is already filed.`); reload(); })
        .catch(e => { if (e instanceof ApiError && e.status === 401) onSignedOut(); else setNote(`${f.name}: ${e instanceof Error ? e.message : "upload failed"}`); });
    }
  }
  // Create schedule: every line at its claimed amount to date, then the sheet.
  async function create(c: Claim) {
    if (busy) return;
    setBusy(true); setError("");
    try {
      await invoiceApi.ops(projectId, "claims", [{ op: "claims.accept_claimed", fields: { claim: c.invoice_id } }]);
      reload();
      onOpen(c.invoice_id);
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
      setError(e instanceof ApiError && e.message ? e.message : "Could not create the schedule. Try again.");
    } finally { setBusy(false); }
  }

  // Place an invoice: for one claim on its contract, or not for a claim. One press each.
  async function place(inv: Invoice, action: "settles" | "not_claim", claim = "") {
    if (busy) return;
    setBusy(true); setError("");
    try {
      await invoiceApi.edit(projectId, inv.id, {}, action, claim);
      reload();
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
      setError(e instanceof ApiError && e.message ? e.message : "Could not place the invoice. Try again.");
    } finally { setBusy(false); }
  }
  const toPlace = (ledger?.invoices ?? []).filter(i => i.held);

  if (!ledger) return <p role="status">Loading…</p>;
  if (contracts.length === 0) return <section className="inv-empty" aria-label="Progress claims"><h2>No live contracts yet</h2><p>Progress claims sit on an awarded contract, a builder's or a consultant's. Award a package in Procurement and it appears here.</p></section>;
  return <section className="pc" aria-label="Progress claims" aria-busy={busy}>
    {error && <p role="alert" className="inv-error">{error}</p>}
    {note && <p className="reg-notice" role="status">{note}</p>}
    {toPlace.length > 0 && <div className="pc-place" role="region" aria-label="Invoices to place">
      <p className="pc-place-head">{toPlace.length === 1 ? "1 invoice to place" : `${toPlace.length} invoices to place`}. Not counted until placed.</p>
      <ul>{toPlace.map(inv => {
        const open = claims.filter(c => c.package_id === inv.supplier_id && c.awaiting_invoice);
        return <li key={inv.id}>
          <span>Invoice {inv.number || inv.filename}{inv.supplier ? ` from ${inv.supplier}` : ""}{inv.total != null ? `, $${cents(inv.total)}` : ""}:</span>
          {open.map(c => <button key={c.invoice_id} type="button" className="btn btn-small" disabled={busy} onClick={() => void place(inv, "settles", c.invoice_id)}>For claim {c.display_number}</button>)}
          <button type="button" className="btn btn-small" disabled={busy} onClick={() => void place(inv, "not_claim")}>Not for a claim</button>
        </li>;
      })}</ul>
    </div>}
    {only && <p className="reg-filter">{showingLine("claims", only.size, onlyShown)} · <button type="button" className="cell-link" onClick={() => setOnly(null)}>Clear</button></p>}
    <div className="inv-table-scroll"><table className="inv-table pc-table">
      <caption className="sr-only">Payment schedules by contract</caption>
      <thead><tr><th scope="col">No.</th><th scope="col">Period</th><th scope="col">Received</th><th scope="col" className="num">Claimed</th><th scope="col" className="num">Scheduled</th><th scope="col" className="num">Retention</th><th scope="col" className="num">Payable inc GST</th><th scope="col">Invoice</th><th scope="col">Schedule due</th><th scope="col">Status</th></tr></thead>
      {contracts.map(c => <ContractGroup key={c.package_id} contract={c} claims={claims.filter(k => k.package_id === c.package_id && (!only || only.has(k.invoice_id)))}
        selected={selection.ids} busy={busy} onOpen={onOpen} onCreate={create} onDrop={files => drop(c, files)} />)}
    </table></div>
  </section>;
}

function ContractGroup({ contract, claims, selected, busy, onOpen, onCreate, onDrop }: { contract: Contract; claims: Claim[]; selected: readonly string[]; busy: boolean; onOpen: (id: string) => void; onCreate: (c: Claim) => void; onDrop: (files: File[]) => void }) {
  const [over, setOver] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  const drag = {
    onDragOver: (e: DragEvent) => { if (!e.dataTransfer.types.includes("Files")) return; e.preventDefault(); e.stopPropagation(); setOver(true); },
    onDragLeave: (e: DragEvent) => { e.stopPropagation(); setOver(false); },
    onDrop: (e: DragEvent) => { if (!e.dataTransfer.types.includes("Files")) return; e.preventDefault(); e.stopPropagation(); setOver(false); onDrop(Array.from(e.dataTransfer.files)); },
  };
  const fig = (label: string, value: string) => <span className="pc-fig">{label} <b>{value}</b></span>;
  return <tbody className="pc-group" data-over={over || undefined} aria-label={`${contract.title} claims`} {...drag}>
    <tr className="pc-contract"><th colSpan={10} scope="colgroup">
      <span className="pc-contract-name">{contract.title}</span>{contract.firm && <span className="pc-contract-firm">{contract.firm}</span>}{contract.services && <span className="pc-contract-firm">Consultant · its claims are its tax invoices</span>}
      <button type="button" className="pc-add" onClick={() => input.current?.click()}>Add a claim</button>
      <input ref={input} type="file" hidden data-testid="claim-input" accept=".pdf,.docx,.xlsx,.md,.markdown,application/pdf" onChange={e => { if (e.target.files?.length) onDrop(Array.from(e.target.files)); e.target.value = ""; }} />
    </th></tr>
    <tr className="pc-figs"><td colSpan={10} aria-label={`${contract.title} figures`}>
      {fig("Contract sum to date", contract.contract_sum_to_date == null ? "unknown" : cents(contract.contract_sum_to_date))}
      {contract.adjusted != null && contract.adjusted !== contract.contract_sum_to_date && fig("Adjusted", cents(contract.adjusted))}
      {fig("Certified to date", cents(contract.certified_to_date))}
      {fig("Retention held", /[1-9]/.test(contract.retention_held ?? "") ? cents(contract.retention_held) : "nil")}
      {fig("Paid to date", cents(contract.paid_to_date))}
    </td></tr>
    {claims.length === 0 ? <tr className="pc-hint"><td colSpan={10}>No claims yet. Drop the contractor's claim on this contract, or add one.</td></tr> : claims.map(c => {
      const untouched = c.issued.length === 0 && c.rows.length > 0 && c.unassessed === c.rows.length;
      return <tr key={c.invoice_id} className="pc-claim" data-claim={c.invoice_id} data-alert={c.alert || undefined} aria-selected={selected.includes(c.invoice_id)}
        onClick={e => { if (!(e.target as HTMLElement).closest("button,a,input")) onOpen(c.invoice_id); }}>
        <td><button type="button" className="inv-open" aria-label={`Open schedule ${c.display_number}`} onClick={() => onOpen(c.invoice_id)}>{c.display_number}</button></td>
        <td>{c.period}</td>
        <td>{day(c.served_on)}{!c.served_confirmed && <span className="cost-tag">confirm</span>}</td>
        <td className="num">{nilDash(c.claimed)}</td>
        <td className="num">{nilDash(c.scheduled)}</td>
        <td className="num">{retention(c.retention)}</td>
        <td className="num">{nilDash(c.payable)}</td>
        <td>{c.invoice_no ? <>{c.invoice_no}{c.tax_invoice && <span className="pc-muted"> · own</span>}</> : c.issued.length ? <span className="pc-muted">Awaiting invoice</span> : ""}</td>
        <td>{c.issued.length ? "" : <>{c.schedule_due.date ? day(c.schedule_due.date) : "Unknown"}{c.alert ? <span className="cost-tag" data-alert={c.alert}>{ALERT[c.alert]}</span> : c.schedule_due.to_check && <span className="cost-tag" data-alert="check">To check</span>}</>}</td>
        <td>{claimStatus(c)}{c.late && " · claimed amount counts"}{untouched && <button type="button" className="cell-link pc-create" disabled={busy} onClick={() => onCreate(c)}>Create schedule</button>}</td>
      </tr>;
    })}
  </tbody>;
}
