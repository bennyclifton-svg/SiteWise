// The stage picker on a Procurement package (program plan 7.6): one
// checkbox per program stage. Ticking creates the package stage, unticking
// retires it through the existing path; labels come from the program, and
// Procurement never adds or renames a stage. When the program's stages have
// changed since the package was built, a notice lists the differences and
// offers Update or Keep. RfpStages is the same notice on an RFP, with the
// RFP's stage rows. No path here calls Jev.
import { useCallback, useEffect, useState } from "react";
import { ApiError } from "./api";
import { changeWords, stageSyncApi, type StageSync, type StageSyncChoice } from "./stageSyncApi";
import "./packageStages.css";

function failText(e: unknown): string {
  if (e instanceof ApiError && e.status === 409) return "The program's stages changed again. Showing the latest differences.";
  if (e instanceof ApiError && e.status === 422) return "This stage still has costs, scope or open delivery records. Resolve those first.";
  return "Could not save. Try again.";
}

/** The Update or Keep notice for one document. */
function StageSyncNotice({ sync, what, busy, onChoose }: { sync: StageSync; what: string; busy: boolean; onChoose: (c: StageSyncChoice) => void }) {
  if (!sync.offer) return null;
  return <div className="report-notice stage-sync-notice" role="status" aria-label={`Stage changes for ${what}`}>
    <strong>The program's stages changed since this {what} was built.</strong>
    <ul>{sync.changes.map(c => <li key={c.kind + c.program_stage_id}>{changeWords(c)}</li>)}{sync.reordered && <li>Stages reordered</li>}</ul>
    {sync.proposals_flagged > 0 && <p>{sync.proposals_flagged} proposal{sync.proposals_flagged === 1 ? " was" : "s were"} priced on the earlier stages. Proposals are never changed.</p>}
    <div className="report-actions">
      <button type="button" className="btn btn-small" disabled={busy} onClick={() => onChoose("update")}>Update</button>
      <button type="button" className="btn btn-small" disabled={busy} onClick={() => onChoose("keep")}>Keep</button>
    </div>
  </div>;
}

export function PackageStages({ projectId, pkgId, label, tick, onSignedOut }: { projectId: string; pkgId: string; label: string; tick: number; onSignedOut: () => void }) {
  const [sync, setSync] = useState<StageSync | null>(null);
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [reload, setReload] = useState(0);

  const fail = useCallback((e: unknown) => {
    if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
    setError(failText(e));
    setReload(n => n + 1);
  }, [onSignedOut]);

  useEffect(() => {
    let stopped = false;
    stageSyncApi.pkg(projectId, pkgId).then(s => { if (!stopped) setSync(s); }).catch(e => { if (!stopped && e instanceof ApiError && e.status === 401) onSignedOut(); });
    return () => { stopped = true; };
  }, [projectId, pkgId, tick, reload, onSignedOut]);

  async function run(work: () => Promise<unknown>) {
    setBusy(true); setError("");
    try { await work(); setReload(n => n + 1); } catch (e) { fail(e); } finally { setBusy(false); }
  }
  // The picker needs the stage plan; the first time it opens on a project
  // with no program, Procurement creates it (the surface that comes first).
  async function toggle() {
    const next = !open;
    setOpen(next);
    if (next && sync && !sync.program_exists) await run(() => stageSyncApi.prepare(projectId));
  }
  const choose = (choice: StageSyncChoice) => sync && run(async () => setSync(await stageSyncApi.syncPkg(projectId, pkgId, choice, sync.revision)));

  if (!sync) return null;
  const ticked = sync.stages.filter(s => s.ticked).length;
  return <div className="package-stages">
    <button type="button" className="table-action" aria-expanded={open} aria-label={`Stages for ${label}`} onClick={() => void toggle()}>
      Stages{sync.program_exists ? ` ${ticked} of ${sync.stages.length}` : ""}{sync.offer ? " · changed" : ""}
    </button>
    {error && <p className="stage-error" role="alert">{error}</p>}
    <StageSyncNotice sync={sync} what="package" busy={busy} onChoose={c => void choose(c)} />
    {open && sync.program_exists && <fieldset className="stage-picker" disabled={busy}>
      <legend className="sr-only">Program stages for {label}</legend>
      {sync.stages.map(s => <label key={s.program_stage_id}>
        <input type="checkbox" checked={s.ticked} onChange={() => void run(() => s.ticked && s.package_stage_id
          ? stageSyncApi.untick(projectId, pkgId, s.package_stage_id, s.package_stage_version ?? 1)
          : stageSyncApi.tick(projectId, pkgId, s.program_stage_id))} />
        {s.label}
      </label>)}
      {sync.kept.filter(k => k.removed_from_program).map(k => <label key={k.id} className="stage-kept"><input type="checkbox" checked disabled />{k.label} <span>removed from program</span></label>)}
    </fieldset>}
    {open && !sync.program_exists && <p className="stage-error" role="status">Preparing the stage plan…</p>}
  </div>;
}

/** Update or Keep on an RFP, and the RFP's stage rows. */
export function RfpStages({ reportId, tick, onChanged, onSignedOut }: { reportId: string; tick: number; onChanged: () => void; onSignedOut: () => void }) {
  const [sync, setSync] = useState<StageSync | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [reload, setReload] = useState(0);
  useEffect(() => {
    let stopped = false;
    stageSyncApi.report(reportId).then(s => { if (!stopped) setSync(s); }).catch(e => { if (!stopped && e instanceof ApiError && e.status === 401) onSignedOut(); });
    return () => { stopped = true; };
  }, [reportId, tick, reload, onSignedOut]);
  async function choose(choice: StageSyncChoice) {
    if (!sync) return;
    setBusy(true); setError("");
    try { setSync(await stageSyncApi.syncReport(reportId, choice, sync.revision)); onChanged(); }
    catch (e) { if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; } setError(failText(e)); setReload(n => n + 1); }
    finally { setBusy(false); }
  }
  if (!sync || !sync.program_exists || (!sync.rows.length && !sync.offer)) return null;
  return <section className="rfp-stages" aria-label="Program stages">
    {error && <p className="stage-error" role="alert">{error}</p>}
    <StageSyncNotice sync={sync} what={sync.document === "issued_rfp" ? "issued RFP" : "RFP draft"} busy={busy} onChoose={c => void choose(c)} />
    {sync.rows.length > 0 && <p className="report-muted">Program stages: <span className="rfp-stage-rows">{sync.rows.map(r => <span key={r.program_stage_id} className={r.removed_from_program ? "stage-kept" : undefined}>{r.label}{r.removed_from_program ? " (removed from program)" : ""}</span>)}</span></p>}
  </section>;
}
