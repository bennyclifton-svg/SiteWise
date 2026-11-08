import { useEffect, useState, type DragEvent, type FormEvent } from "react";
import { FILED_DOCUMENT_TYPE } from "./documentDrag";
import { IconSelectAll, IconUpdate } from "./icons";
import type { ReportBlock, RfpControl, RfpControls } from "./reportApi";
import { blockedKeys, type Write } from "./RfpEditor";
import { Stepper } from "./RfpAllowanceTable";
import type { TransmittalSelection } from "./RfpTransmittal";
import "./rfp-tables.css";

/** Saved drafts from before 8 Nov 2026 still hold this block; it no longer prints. */
export const AGREEMENT_BLOCK = "rfp:terms:agreement";
const AGREEMENT_DOCS = "agreement_docs";

/**
 * Terms as one table (owner direction, 8 November 2026): the numbered terms
 * first, cited by number in negotiation, then the answers they draw on,
 * Item | Response, in the same table. The client agreement is one line of
 * document titles chosen in the repository; the documents themselves travel
 * in the transmittal. A missing agreement is marked on its row and blocks
 * issue; there is no banner.
 */
export function RfpTermsTable({ controls, block, selection, busy, onWrite }: { controls: RfpControls; block?: ReportBlock; selection?: TransmittalSelection; busy: boolean; onWrite: Write }) {
  const items = controls.groups.find(g => g.id === "terms")?.controls ?? [];
  const lines = termLines(block?.text ?? "");
  const blocked = blockedKeys(controls);
  if (!items.length && !lines.length) return null;
  const agreementIDs = (block?.basis?.agreement_ids ?? []) as string[];
  return <div className="report-table-scroll"><table className="report-table rfp-terms-table">
    <colgroup><col className="rfp-terms-no-col" /><col className="rfp-terms-item-col" /><col /></colgroup>
    <thead><tr><th scope="col" className="rfp-terms-no">No.</th><th scope="col" colSpan={2}>Term</th></tr></thead>
    <tbody>
      {lines.map((line, i) => <tr key={i} className="rfp-terms-line"><td className="rfp-terms-no">{i + 1}</td><td colSpan={2}>{line}</td></tr>)}
      {items.length > 0 && <tr className="table-subhead"><th scope="col" colSpan={2}>Item</th><th scope="col">Response</th></tr>}
      {items.map(c => c.key === "field:agreement"
        ? <AgreementRow key={c.key} control={c} saved={agreementIDs} selection={selection} blocked={blocked.has(c.key)} busy={busy} onWrite={onWrite} />
        : <tr key={c.key} data-control={c.key} data-blocked={blocked.has(c.key) || undefined}>
          <th scope="row" colSpan={2}><label htmlFor={termID(c)}>{c.label}</label></th>
          <td><TermValue control={c} busy={busy} onWrite={onWrite} /></td>
        </tr>)}
    </tbody>
  </table></div>;
}

function termID(c: RfpControl) {
  return `rfp-term-${c.key.replace(/[^a-z0-9]+/gi, "-")}`;
}

// The agreement is named by the titles of the documents chosen for it: pick
// them in the repository (one or several), or drag one onto the row.
function AgreementRow({ control, saved, selection, blocked, busy, onWrite }: { control: RfpControl; saved: string[]; selection?: TransmittalSelection; blocked: boolean; busy: boolean; onWrite: Write }) {
  const [over, setOver] = useState(false);
  const filed = (e: DragEvent) => e.dataTransfer.types.includes(FILED_DOCUMENT_TYPE);
  const drop = (e: DragEvent) => {
    if (!filed(e)) return;
    e.preventDefault(); e.stopPropagation(); setOver(false);
    const id = e.dataTransfer.getData(FILED_DOCUMENT_TYPE);
    if (id && !saved.includes(id) && !busy) void onWrite(AGREEMENT_DOCS, [...saved, id]);
  };
  const name = control.value === null || control.value === undefined ? "" : String(control.value);
  return <tr className="rfp-terms-agreement" data-control={control.key} data-blocked={blocked || undefined} data-drop={over || undefined}
    onDragOver={e => { if (filed(e)) { e.preventDefault(); e.stopPropagation(); e.dataTransfer.dropEffect = "link"; setOver(true); } }}
    onDragLeave={() => setOver(false)} onDrop={drop}>
    <th scope="row" colSpan={2}>{control.label}</th>
    <td><div className="rfp-terms-agreement-line">
      <span className={name ? "rfp-terms-agreement-name" : "rfp-terms-empty"}>{name || `${blocked ? "Needed before issue. " : ""}Select it in the document repository.`}</span>
      {selection && <span className="rfp-terms-agreement-actions">
        <button type="button" className="btn btn-small" disabled={busy} title="Show the agreement's documents selected in the repository" onClick={() => selection.onShowDocuments(saved)}><IconSelectAll />Select agreement documents</button>
        <button type="button" className="btn btn-small" disabled={busy || (!selection.selectedIds.length && !saved.length)} title={`Use the ${selection.selectedIds.length} documents selected in the repository`} onClick={() => void onWrite(AGREEMENT_DOCS, [...selection.selectedIds])}><IconUpdate />Update agreement</button>
      </span>}
    </div></td>
  </tr>;
}

function TermValue({ control: c, busy, onWrite }: { control: RfpControl; busy: boolean; onWrite: Write }) {
  const value = c.value === null || c.value === undefined ? "" : String(c.value);
  const save = (v: string) => void onWrite(c.key, v === "" ? null : v);
  if (c.kind === "number") return <span className="rfp-terms-number"><Stepper id={termID(c)} label={c.label} value={value} placeholder="" busy={busy} onCommit={save} /><span className="rfp-terms-unit">days</span></span>;
  if (c.kind === "date") return <TextValue id={termID(c)} type="date" value={value} busy={busy} onCommit={save} />;
  return <TextValue id={termID(c)} value={value} busy={busy} onCommit={save} />;
}

// Typed text commits on blur or Enter, so a half-typed value is never saved.
function TextValue({ id, value, type = "text", busy, onCommit }: { id: string; value: string; type?: string; busy: boolean; onCommit: (v: string) => void }) {
  const [draft, setDraft] = useState(value);
  useEffect(() => setDraft(value), [value]);
  const commit = (e?: FormEvent) => { e?.preventDefault(); if (draft.trim() !== value) onCommit(draft.trim()); };
  return <form className="rfp-terms-inline" onSubmit={commit}>
    <input id={id} className="rfp-terms-input" type={type} value={draft} placeholder="—" disabled={busy} onChange={e => setDraft(e.target.value)} onBlur={() => commit()} />
  </form>;
}

/** The terms wording split into its numbered terms; the PDF numbers the same lines (internal/export). */
export function termLines(text: string): string[] {
  return text.split("\n").map(l => l.trim().replace(/^- /, "").trim()).filter(Boolean);
}
