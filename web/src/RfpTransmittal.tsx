import { IconSelectAll, IconUpdate } from "./icons";
import type { ReportBlock } from "./reportApi";
import type { Write } from "./RfpEditor";
import "./rfp-tables.css";

export interface TransmittalSelection {
  selectedIds: readonly string[];
  onSelectDocuments: (ids: readonly string[]) => void;
  /** Shows the register with exactly these documents selected. */
  onShowDocuments: (ids: readonly string[]) => void;
}

/**
 * The transmittal: its two buttons sit in the section's blue band and the
 * table's column heads directly under it, one header block (owner
 * direction, 8 November 2026). Any note follows the table.
 */
export function RfpTransmittal({ block, selection, busy, onWrite }: { block?: ReportBlock; selection?: TransmittalSelection; busy: boolean; onWrite: Write }) {
  const columns = block?.table?.columns ?? ["Doc number", "Title", "Rev", "Date", "Discipline"];
  const rows = block?.table?.rows ?? [];
  // The saved schedule: pick it up again, add or drop documents, then update.
  const saved = (block?.basis?.document_ids ?? []) as string[];
  return <section className="rfp-transmittal" aria-label="Transmittal">
    <div className="rfp-transmittal-heading">{selection && <div className="rfp-transmittal-actions">
      <button type="button" className="btn btn-small" disabled={busy} title="Select this transmittal's saved documents in the repository" onClick={() => selection.onShowDocuments(saved)}><IconSelectAll />Select transmittal documents</button>
      <button type="button" className="btn btn-small" disabled={busy} title={`${selection.selectedIds.length} selected in the repository`} onClick={() => void onWrite("transmittal", [...selection.selectedIds])}><IconUpdate />Update Transmittal</button>
    </div>}</div>
    <div className="report-table-scroll"><table className="report-table rfp-transmittal-table">
      <thead><tr>{columns.map(c => <th key={c} scope="col">{c}</th>)}</tr></thead>
      <tbody>{rows.length ? rows.map((row, i) => <tr key={i}>{row.map((cell, j) => <td key={j}>{cell || "—"}</td>)}</tr>) : <tr><td colSpan={5}>No documents in this transmittal.</td></tr>}</tbody>
    </table></div>
    {block?.text && <p className="report-wording">{block.text.replace(/^Information provided: /, "")}</p>}
  </section>;
}
