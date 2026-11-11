// The strip at the foot of the profile (docs/design/2026-11-10-run-time-feedback.md).
// At rest it is one line: what is read, Source requirements, when it was
// updated, and the one next step. While documents are read it shows each
// document's passes, the latest passage read, and the facts documents seldom
// settle so the wait is the time to state them. When an update finishes it
// says what the update did and where it matters next. Every number is one
// the server counted.
import { useState, type ReactNode } from "react";
import type { Profile as ProfileData, ProfileField } from "./profileApi";
import type { Next, Run, RunDoc } from "./profileRun";
import type { Receipt } from "./profileDiff";
import type { Setter } from "./ProfileFields";

export type ChipKind = "filled" | "changed" | "added" | "removed" | "held" | "check";
export type ElsewhereView = "procurement" | "costs" | "program";

/** The passage last read, for the strip's "latest find" line. */
export interface LatestFind { what: string; excerpt: string; where: string }

interface Props {
  data: ProfileData;
  run: Run;
  /** Changed since the user last cleared the marks. */
  marks: Receipt;
  marksTotal: number;
  /** The update finished here and its summary is still open. */
  receiptOpen: boolean;
  next: Next | null;
  checks: number;
  requestNote: string;
  sourcesOpen: boolean;
  latestFind: LatestFind | null;
  below: number;
  /** Key facts every RFP asks for that are still unknown. */
  rfpUnknown: ProfileField[];
  /** Unknown facts documents seldom settle: offered while reading. */
  whileFields: ProfileField[];
  docName: (doc: RunDoc) => string;
  kindLabel: (kind: string) => string;
  onSet: Setter;
  onCheck: () => void;
  onShowNotRead: () => void;
  onSources: () => void;
  onBelow: () => void;
  onChip: (kind: ChipKind) => void;
  onMarkSeen: () => void;
  onCloseReceipt: () => void;
  onOpenView?: (view: ElsewhereView) => void;
  onFocusField: (field: ProfileField) => void;
}

export function ProfileStrip(p: Props) {
  const { data, run } = p;
  const running = run.phase === "running" || data.pending_documents > 0;
  const mode = running ? "run" : run.phase === "done" && p.receiptOpen ? "done" : "rest";
  return (
    <div className="pf-strip" data-mode={mode}>
      {mode === "run" ? <RunView {...p} /> : mode === "done" ? <DoneView {...p} /> : <RestView {...p} />}
    </div>
  );
}

/** The documents-read status, worded as the activity contract has it. */
export function statusText(d: ProfileData): string {
  const total = d.read_documents, completed = d.completed_documents ?? 0;
  const queued = Math.max(0, d.pending_documents - d.active_documents);
  const state = completed === total ? "Reading complete"
    : d.failed_documents > 0 ? "Reading needs attention"
    : d.pending_documents > 0 ? (d.active_documents > 0 ? "Reading documents" : "Reading queued")
    : "Ready to read";
  return `${state} · ${d.active_documents > 0 ? `${d.active_documents} processing · ` : ""}${queued > 0 ? `${queued} queued · ` : ""}${completed} of ${total} document${total === 1 ? "" : "s"} read`;
}

function Progress({ data, quiet }: { data: ProfileData; quiet?: boolean }) {
  const total = data.read_documents, completed = data.completed_documents ?? 0;
  if (total === 0) return null;
  const done = completed === total;
  return (
    <div className="profile-progress" data-quiet={quiet && done ? "true" : undefined}>
      <svg className="profile-reading-meter" viewBox="0 0 24 24" role="progressbar" aria-label="Documents read for profile" aria-valuemin={0} aria-valuemax={total} aria-valuenow={completed}>
        <circle className="profile-reading-track" cx="12" cy="12" r="9" />
        {done
          ? <path className="profile-reading-check" d="m7.5 12 3 3 6-6" />
          : <circle className="profile-reading-fill" cx="12" cy="12" r="9" pathLength="100" strokeDasharray={`${Math.min(100, Math.max(0, (completed / total) * 100))} 100`} />}
      </svg>
      {/* At rest a finished reading says itself through the tick; the words stay for screen readers. */}
      <p role="status" aria-atomic="true" className={quiet && done ? "sr-only" : undefined}>{statusText(data)}</p>
    </div>
  );
}

/** Which documents the profile reads, with the way to the ones it does not, and Source requirements. */
function Included(p: Props) {
  const { data } = p;
  const total = data.read_documents + data.skipped_documents;
  const unresolved = (data.coverage ?? []).reduce((n, d) => n + d.needs_mapping, 0);
  return (
    <>
      {total > 0 && (
        <span>
          Included for profile: {data.read_documents} of {total} document{total === 1 ? "" : "s"}
          {data.skipped_documents > 0 && (
            <>
              {" "}({data.skipped_documents} excluded{" "}
              <button type="button" className="cell-link" title={data.skipped_kind ? `Mostly ${p.kindLabel(data.skipped_kind)}` : undefined} onClick={p.onShowNotRead}>Show</button>)
            </>
          )}
        </span>
      )}
      <span>
        <button type="button" className="cell-link" aria-expanded={p.sourcesOpen} title={unresolved > 0 ? `${unresolved} passages need review` : undefined} onClick={p.onSources}>
          Source requirements
        </button>
      </span>
    </>
  );
}

function RestView(p: Props) {
  const { data } = p;
  const builtAt = data.built_at ? new Date(data.built_at) : null;
  const thresholds = data.thresholds.applied && data.thresholds.provisional ? "Provisional thresholds" : undefined;
  return (
    <div className="pf-strip-line">
      <Progress data={data} quiet />
      <p className="profile-reading">
        <Included {...p} />
        <span title={thresholds}>{builtAt ? `Updated ${builtAt.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}` : "Not built yet"}</span>
        {!data.thresholds.applied && <span>Readings are stored but not shown until the thresholds are approved</span>}
        {p.marksTotal > 0 && (
          <span className="pf-strip-marks">
            <span className="rev-mark" aria-hidden="true" />
            {p.marksTotal} change{p.marksTotal === 1 ? "" : "s"} marked{" "}
            <button type="button" className="cell-link" onClick={p.onMarkSeen}>Clear</button>
          </span>
        )}
      </p>
      <NextStep {...p} />
    </div>
  );
}

/** The one next step, the first that applies; nothing when the page is finished. */
function NextStep(p: Props) {
  const { data } = p;
  const parts: ReactNode[] = [];
  if (p.checks > 0) parts.push(<button key="check" type="button" className="pf-checks" onClick={p.onCheck}>⚠ {p.checks} to check</button>);
  const total = data.read_documents + data.skipped_documents;
  let note: ReactNode = null;
  if (p.requestNote) note = <span className="profile-actions-note">{p.requestNote}</span>;
  else if (data.unread_documents > 0) note = <span className="profile-actions-note">{data.unread_documents} document{data.unread_documents === 1 ? "" : "s"} ready to read. Select Update to start.</span>;
  else if (total === 0) note = <span className="profile-actions-note">Upload documents to start</span>;
  else if ((data.coverage ?? []).some((d) => !d.current)) note = <span className="profile-actions-note">Update needed to check complete source coverage</span>;
  else if (p.rfpUnknown.length > 0) {
    const names = p.rfpUnknown.map((f) => f.label);
    note = (
      <span className="profile-actions-note" title={`Every RFP's page one asks for ${names.length === 1 ? "it" : "them"}`}>
        Not yet known: {names.join(", ")}{" "}
        <button type="button" className="cell-link" onClick={() => p.onFocusField(p.rfpUnknown[0])}>Add</button>
      </span>
    );
  }
  if (note) parts.push(<span key="note">{note}</span>);
  if (parts.length === 0) return null;
  return <div className="pf-strip-next">{parts}</div>;
}

function RunView(p: Props) {
  const { data, run } = p;
  const active = run.docs.find((d) => d.state === "reading");
  return (
    <div className="pf-strip-run">
      {run.docs.length > 0 && (
        <ul className="run-docs" aria-label="Documents in this update">
          {run.docs.map((d) => <RunSheet key={d.id} doc={d} name={p.docName(d)} />)}
        </ul>
      )}
      <div className="run-live">
        <Progress data={data} />
        <p className="profile-reading"><Included {...p} /></p>
        {active && (
          <p className="run-now">
            <b>{p.docName(active)}</b> · {active.pass === "label" ? "label pass" : "evidence pass"} ·{" "}
            {active.pass === "label" ? active.labelled : active.evidence} of {active.units} passages
          </p>
        )}
        {!active && run.preparing > 0 && <p className="run-now">Preparing text for {run.preparing} document{run.preparing === 1 ? "" : "s"}</p>}
        {p.latestFind && (
          <p className="run-find" aria-live="off">
            <span className="run-find-what">{p.latestFind.what}</span> · <q>{p.latestFind.excerpt}</q> · <span className="run-find-where">{p.latestFind.where}</span>
          </p>
        )}
        {p.below > 0 && (
          <button type="button" className="cell-link run-below" onClick={p.onBelow}>
            {p.below} change{p.below === 1 ? "" : "s"} below ↓
          </button>
        )}
        {data.failed_documents > 0 && <p className="run-note">{data.failed_documents} document{data.failed_documents === 1 ? "" : "s"} could not be processed</p>}
      </div>
      {p.whileFields.length > 0 && (
        <div className="run-while" role="group" aria-label="While this runs">
          <p className="run-while-head">While this runs</p>
          {p.whileFields.map((f) => <WhileField key={f.key} field={f} onSet={p.onSet} />)}
        </div>
      )}
    </div>
  );
}

/** One document: a sheet hatched as its passages are labelled, inked as they are read for evidence. */
function RunSheet({ doc, name }: { doc: RunDoc; name: string }) {
  // The fill is clipped from the top, so it rises without laying anything out again.
  const rise = (n: number) => `inset(${100 - (doc.units > 0 ? Math.min(100, (n / doc.units) * 100) : 0)}% 0 0 0)`;
  const said = doc.state === "done" ? "read" : doc.state === "failed" ? `${doc.pass} pass stopped`
    : doc.state === "queued" ? `${doc.pass} pass queued` : `${doc.pass} pass, ${doc.pass === "label" ? doc.labelled : doc.evidence} of ${doc.units} passages`;
  return (
    <li className="run-doc" data-state={doc.state} title={`${name}: ${said}`}>
      <span className="run-sheet" aria-hidden="true">
        <span className="run-hatch" style={{ clipPath: rise(doc.labelled) }} />
        <span className="run-ink" style={{ clipPath: rise(doc.evidence) }} />
      </span>
      <span className="run-doc-name">{name}</span>
      <span className="run-ticks" aria-hidden="true">{Array.from({ length: doc.ticks }, (_, i) => <i key={i} />)}</span>
      <span className="sr-only">{said}</span>
    </li>
  );
}

/** A fact stated by the user while documents are read; nothing read later moves it. */
function WhileField({ field, onSet }: { field: ProfileField; onSet: Setter }) {
  const [draft, setDraft] = useState("");
  const id = `run-${field.key}`;
  const commit = () => {
    const v = draft.trim();
    if (v) onSet(field.key, { part_id: field.part_id, value: v });
  };
  return (
    <label className="run-while-field" htmlFor={id}>
      <span>{field.label}</span>
      {field.kind === "choice" && field.options?.length ? (
        <select id={id} className="select" value="" onChange={(e) => e.target.value && onSet(field.key, { part_id: field.part_id, value: e.target.value })}>
          <option value="">—</option>
          {field.options.map((o) => <option key={o.id} value={o.id}>{o.label}</option>)}
        </select>
      ) : (
        <input id={id} className="input" value={draft} placeholder={field.typical || undefined} onChange={(e) => setDraft(e.target.value)} onBlur={commit}
          onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); commit(); } }} />
      )}
    </label>
  );
}

const ELSEWHERE: { key: keyof Next; view: ElsewhereView; title: string; words: (n: number) => string }[] = [
  { key: "procurement", view: "procurement", title: "Procurement", words: (n) => `${n} discipline${n === 1 ? "" : "s"} proposed, not packages yet` },
  { key: "costs", view: "costs", title: "Cost Plan", words: (n) => `${n} line${n === 1 ? "" : "s"} proposed, not in the plan yet` },
  { key: "program", view: "program", title: "Program", words: (n) => `${n} change${n === 1 ? "" : "s"} waiting for review` },
  { key: "drafts", view: "procurement", title: "RFP drafts", words: (n) => `${n} built before this update` },
];

function DoneView(p: Props) {
  const { run, marks } = p;
  const read = run.docs.filter((d) => d.state === "done");
  const passages = read.reduce((n, d) => n + d.units, 0);
  const elapsed = run.started && run.finishedAt > run.startedAt ? duration(run.finishedAt - run.startedAt) : "";
  const chips: { kind: ChipKind; n: number; words: string }[] = [
    { kind: "filled", n: marks.filled, words: "filled" },
    { kind: "changed", n: marks.changed, words: "changed" },
    { kind: "added", n: marks.added, words: `system${marks.added === 1 ? "" : "s"} added` },
    { kind: "removed", n: marks.removed, words: "removed" },
    { kind: "held", n: marks.held, words: "of yours read again, kept" },
    { kind: "check", n: marks.check, words: "to check" },
  ];
  const elsewhere = p.next ? ELSEWHERE.filter((e) => (p.next as Next)[e.key] > 0) : [];
  return (
    <div className="pf-strip-done">
      <div className="done-head">
        <Progress data={p.data} quiet />
        <p className="done-line">
          <b>Update finished.</b>{" "}
          {read.length > 0
            ? <>Read {read.length} document{read.length === 1 ? "" : "s"}, {passages.toLocaleString()} passages{elapsed && <>, in {elapsed}</>}.</>
            : "Nothing new was read."}
        </p>
        <span className="done-actions">
          {p.marksTotal > 0 && <button type="button" className="cell-link" onClick={p.onMarkSeen}>Mark all seen</button>}
          <button type="button" className="done-close" aria-label="Close the update summary" title="Close" onClick={p.onCloseReceipt}>×</button>
        </span>
      </div>
      {chips.some((c) => c.n > 0) && (
        <div className="done-chips" role="group" aria-label="What this update changed">
          {chips.filter((c) => c.n > 0).map((c) => (
            <button key={c.kind} type="button" className="done-chip" data-kind={c.kind} onClick={() => p.onChip(c.kind)}>
              <b>{c.n}</b> {c.words}
            </button>
          ))}
        </div>
      )}
      {elsewhere.length > 0 && (
        <div className="done-elsewhere">
          <span className="done-elsewhere-head">What this changes elsewhere</span>
          {elsewhere.map((e) => (
            <button key={e.key} type="button" className="done-where" onClick={() => p.onOpenView?.(e.view)}>
              <b>{e.title}</b> {e.words((p.next as Next)[e.key])}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function duration(ms: number): string {
  const s = Math.max(1, Math.round(ms / 1000));
  return s < 60 ? `${s} s` : `${Math.floor(s / 60)} min ${s % 60} s`;
}
