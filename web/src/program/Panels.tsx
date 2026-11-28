// The Program view's update button and readings panels (plan
// docs/plans/2026-10-18-program.md 7.4, 7.7, 10.2 to 10.4). Each panel only
// shows what the server sent and hands a choice back to Program.tsx, which
// sends it as one write through `apply`: no panel changes a value itself and
// none calls Jev.

import { useState, type ReactNode } from "react";
import { GroundButton } from "../GroundButton";
import type { ProgramActivity, ProgramHoldPoint, ProgramImport, ProgramProposedLink } from "../programApi";
import { fmtDate } from "./layout";
import { fileLinks, notReadWords, proposedText, type ReadingGroup } from "./readings";

/**
 * The page's primary action, like Update project profile on the Profile
 * (owner, 2026-11-03). A change elsewhere never moves the program's dates:
 * it only lights this button with a count. Clicking it is the user committing
 * to every update, so it applies them at once with nothing to answer (owner,
 * 12 November 2026); one undo reverses it.
 */
export function UpdateProgramButton({ pending, busy, onUpdate }: { pending: number; busy: boolean; onUpdate: () => void }) {
  // The ground-line button every view uses for its one primary action (PanelHead).
  return (
    <GroundButton
      primary
      icon="refresh"
      className={pending > 0 ? "btn-update pg-update is-pending" : "btn-update pg-update"}
      disabled={busy}
      busy={busy}
      title={pending > 0
        ? `${pending} update${pending === 1 ? "" : "s"} from the Profile and Procurement. Click to apply; undo reverses it.`
        : "Rebuild the program from the Profile and Procurement."}
      onClick={onUpdate}
    >
      Update program{pending > 0 && <span className="pg-update-count">{pending}</span>}
    </GroundButton>
  );
}

/** A stage to add a file row under; "" adds it as a stage. */
export interface StageChoice {
  id: string;
  label: string;
}

/**
 * Proposed readings by target with Accept and Dismiss, each file's state,
 * its rows that matched nothing, each added in one click, and the links its
 * predecessor column makes between matched rows (plan 10.2); an awarded
 * proposal's readings show their own line (10.3).
 */
export function ReadingsPanel({
  imports,
  groups,
  labelOf,
  stages,
  busy,
  onAccept,
  onDismiss,
  onAddRow,
  onAddLinks,
  onClose,
}: {
  imports: readonly ProgramImport[];
  groups: readonly ReadingGroup[];
  labelOf: (id: string) => string;
  stages: readonly StageChoice[];
  busy: boolean;
  onAccept: (ids: string[]) => void;
  onDismiss: (ids: string[]) => void;
  onAddRow: (importId: string, row: string, parentId: string) => void;
  onAddLinks: (links: ProgramProposedLink[]) => void;
  onClose: () => void;
}) {
  const [parent, setParent] = useState<Record<string, string>>({});
  const live = groups.filter((g) => g.targetId);
  const proposals = live.filter((g) => g.source === "proposal_r7");
  return (
    <section className="pg-panel" aria-label="Readings">
      <header className="pg-panel-head">
        <h2>Readings</h2>
        <span className="pg-panel-why">Proposals from documents: nothing changes until you accept.</span>
        <button type="button" className="btn btn-small" onClick={onClose}>
          Close
        </button>
      </header>
      {imports.map((imp) => {
        const mine = live.filter((g) => g.importId === imp.id);
        const under = parent[imp.id] ?? stages[0]?.id ?? "";
        const notMatched = (imp.not_matched ?? []).map(labelOf).filter(Boolean);
        const grey = (imp.grey ?? []).map(labelOf).filter(Boolean);
        const links = fileLinks(imp, new Set(mine.map((g) => g.targetId!)), (id) => labelOf(id) !== "");
        const addable = links.filter((l) => !l.onAccept);
        return (
          <div key={imp.id} className="pg-import" data-import={imp.id}>
            <h3>
              <a href={`/api/documents/${imp.document_id}/file?view=1`} target="_blank" rel="noreferrer">
                {imp.filename}
              </a>
              <span className={`pg-import-state is-${imp.status}`}>{imp.text}</span>
            </h3>
            {imp.status === "not_read" && <p className="pg-panel-empty">{notReadWords(imp.not_read)}</p>}
            {mine.length > 0 && <ReadingList groups={mine} labelOf={labelOf} busy={busy} onAccept={onAccept} onDismiss={onDismiss} />}
            {links.length > 0 && (
              <div className="pg-file-rows pg-file-links">
                <div className="pg-file-rows-head">
                  <span>
                    {links.length} link{links.length === 1 ? "" : "s"} from the file's predecessor column
                  </span>
                  {addable.length > 1 && (
                    <button type="button" className="btn btn-small" disabled={busy} onClick={() => onAddLinks(addable.map((l) => l.link))}>
                      Add all {addable.length} links
                    </button>
                  )}
                </div>
                <ul>
                  {links.map((l) => {
                    const pred = labelOf(l.link.pred_id);
                    const succ = labelOf(l.link.succ_id);
                    return (
                      <li key={l.key} data-pred={l.link.pred_id} data-succ={l.link.succ_id}>
                        <span className="pg-file-label">
                          {pred} → {succ}
                        </span>
                        <span className="pg-file-dates">{l.text}</span>
                        {l.onAccept ? (
                          <span className="pg-reading-kept">Added when you accept the readings</span>
                        ) : (
                          <button type="button" className="btn btn-small" disabled={busy} aria-label={`Add link ${pred} to ${succ}`} onClick={() => onAddLinks([l.link])}>
                            Add
                          </button>
                        )}
                      </li>
                    );
                  })}
                </ul>
              </div>
            )}
            {grey.length > 0 && <p className="pg-import-grey">Not read yet, retried in the background: {grey.join(", ")}.</p>}
            {imp.status === "read" && notMatched.length > 0 && (
              <details className="pg-import-unmatched">
                <summary>{notMatched.length} program row{notMatched.length === 1 ? "" : "s"} matched nothing in the file</summary>
                <p>{notMatched.join(", ")}</p>
              </details>
            )}
            {imp.unmatched > 0 && (
              <div className="pg-file-rows">
                <div className="pg-file-rows-head">
                  <span>
                    {imp.unmatched} file row{imp.unmatched === 1 ? "" : "s"} matched no activity
                  </span>
                  <label>
                    <span>Add under</span>
                    <select className="input" value={under} onChange={(e) => setParent((m) => ({ ...m, [imp.id]: e.target.value }))}>
                      {stages.map((s) => (
                        <option key={s.id} value={s.id}>
                          {s.label}
                        </option>
                      ))}
                      <option value="">As a new stage</option>
                    </select>
                  </label>
                </div>
                <ul>
                  {(imp.unmatched_rows ?? []).map((r) => (
                    <li key={r.row}>
                      <span className="pg-file-label">{r.label}</span>
                      <span className="pg-file-dates">{[r.start ? fmtDate(r.start) : "", r.finish ? fmtDate(r.finish) : "", r.duration ?? ""].filter(Boolean).join(" – ")}</span>
                      <button type="button" className="btn btn-small" disabled={busy} aria-label={`Add ${r.label}`} onClick={() => onAddRow(imp.id, r.row, under)}>
                        Add
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        );
      })}
      {proposals.length > 0 && (
        <div className="pg-import">
          <h3>From awarded proposals</h3>
          <ReadingList groups={proposals} labelOf={labelOf} busy={busy} onAccept={onAccept} onDismiss={onDismiss} />
        </div>
      )}
    </section>
  );
}

function ReadingList({
  groups,
  labelOf,
  busy,
  onAccept,
  onDismiss,
}: {
  groups: readonly ReadingGroup[];
  labelOf: (id: string) => string;
  busy: boolean;
  onAccept: (ids: string[]) => void;
  onDismiss: (ids: string[]) => void;
}) {
  return (
    <ul className="pg-readings">
      {groups.map((g) => {
        const label = labelOf(g.targetId!);
        return (
          <li key={g.key} data-target={g.targetId!}>
            <span className="pg-reading-target">{label}</span>
            <span className="pg-reading-value">{g.text || proposedText(g, fmtDate)}</span>
            {!g.applies && <span className="pg-reading-kept">Your value is kept if you accept</span>}
            <button type="button" className="btn btn-small" disabled={busy} aria-label={`Accept reading for ${label}`} onClick={() => onAccept([g.ids[0]])}>
              Accept
            </button>
            <button type="button" className="btn btn-small" disabled={busy} aria-label={`Dismiss reading for ${label}`} onClick={() => onDismiss([g.ids[0]])}>
              Dismiss
            </button>
          </li>
        );
      })}
    </ul>
  );
}

/** "Evidence for Approval documentation has no stage. Add the stage back or dismiss." (plan 7.7) */
export function EvidenceNotice({ label, busy, onDismiss }: { label: string; busy: boolean; onDismiss: () => void }) {
  return (
    <div className="pg-note pg-banner" role="status">
      <span>Evidence for {label} has no stage. Add the stage back or dismiss.</span>
      <button type="button" className="pg-link-button" disabled={busy} onClick={onDismiss}>
        Dismiss
      </button>
    </div>
  );
}

/**
 * Hold points with their release evidence (plan 10.4): "Evidence found:
 * <title>, page <n>" linking to that page, the document's filed date, and
 * the known limit. Only the user's click releases one, with the evidence
 * or without it.
 */
export function HoldPointsPanel({
  holds,
  info,
  busy,
  onRelease,
}: {
  holds: readonly ProgramActivity[];
  info: ReadonlyMap<string, ProgramHoldPoint>;
  busy: boolean;
  onRelease: (id: string, readingId: string) => void;
}) {
  const pending = holds.filter((h) => !info.get(h.id)?.released && (info.get(h.id)?.evidence ?? []).length > 0).length;
  const released = holds.filter((h) => info.get(h.id)?.released).length;
  return (
    <details className="pg-panel pg-holds" open={pending > 0 || undefined}>
      <summary>
        Hold points: {holds.length}
        {released > 0 && `, ${released} released`}
        {pending > 0 && `, ${pending} with evidence to check`}
      </summary>
      <ul>
        {holds.map((h) => {
          const hp = info.get(h.id);
          const rel = hp?.released ?? null;
          return (
            <li key={h.id} data-hold={h.id}>
              <span className="pg-hold-label">{h.label}</span>
              {rel ? (
                <span className="pg-hold-released">
                  Released
                  {rel.document_id && (
                    <>
                      {" on "}
                      <DocLink id={rel.document_id} page={rel.page ?? 0}>
                        {rel.document_title || "the document"}
                        {rel.page ? `, page ${rel.page}` : ""}
                      </DocLink>
                      {rel.document_date && `, ${rel.date_label || "document dated"} ${fmtDate(rel.document_date)}`}
                    </>
                  )}
                </span>
              ) : (
                <>
                  {(hp?.evidence ?? []).map((e) => (
                    <span key={e.reading_id} className="pg-hold-evidence" title={hp?.limit}>
                      <DocLink id={e.document_id} page={e.page}>
                        {e.text}
                      </DocLink>
                      {e.document_date && <span className="pg-hold-date">{`${e.date_label || "document dated"} ${fmtDate(e.document_date)}`}</span>}
                      <button type="button" className="btn btn-small" disabled={busy} onClick={() => onRelease(h.id, e.reading_id)}>
                        Mark released
                      </button>
                    </span>
                  ))}
                  {(hp?.evidence ?? []).length === 0 && (
                    <button type="button" className="btn btn-small" disabled={busy} aria-label={`Mark ${h.label} released`} onClick={() => onRelease(h.id, "")}>
                      Mark released
                    </button>
                  )}
                </>
              )}
            </li>
          );
        })}
      </ul>
    </details>
  );
}

// The document link convention of the Profile sources: the viewer at the page.
function DocLink({ id, page, children }: { id: string; page: number; children: ReactNode }) {
  return (
    <a href={`/api/documents/${id}/file?view=1${page > 0 ? `#page=${page}` : ""}`} target="_blank" rel="noreferrer">
      {children}
    </a>
  );
}
