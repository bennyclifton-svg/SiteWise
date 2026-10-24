// Scope and works in one place. A rail lists Planning and approvals, then
// every top-level system in the order a building is built
// (knowledge/profile/layout.yaml). The chosen entry shows its systems, one
// row each: the scope tick, what the works do, who provides it and the
// document evidence. Below them sit the site risks and compliance inputs that
// belong to that entry.
//
// The tick is the one scope decision: it writes the system's own work item
// for the whole project (its coarse item), so it is also that item's
// inclusion. Other work items (another part, a split, one added by hand)
// carry their own Included box. Defaults come from the building class and
// work type, documents can add systems, and the user's choice is final: a
// document that says a ticked system is not included is flagged, never
// allowed to untick it.

import { useCallback, useEffect, useState } from "react";
import { Conflict, FieldRow, Mark, ResetButton, Sources, type Setter } from "./ProfileFields";
import { WORK_ACTIONS, WorkItemDetail, patchWorkRow, workError } from "./WorkItem";
import { workApi, type WorkRecord } from "./workApi";
import type { Profile, ProfileField, ScopeChoice, SystemRow } from "./profileApi";

export const PLANNING = "planning";
const OTHER = "other";

interface Props {
  projectId: string;
  data: Profile;
  /** Changes when the project changes elsewhere; saved work reloads. */
  tick: number;
  selected: string;
  onSelect: (id: string) => void;
  onScope: (systems: Record<string, ScopeChoice>) => Promise<void>;
  onSet: Setter;
  onJump: (id: string) => void;
  /** Opens the source requirements filtered to a system or group. */
  onSources: (system: string) => void;
  onSignedOut: () => void;
}

const PROVIDERS: Record<string, string> = { contractor: "Contractor", owner: "Owner", landlord: "Landlord", others: "Others" };

const ORIGINS: Record<string, string> = {
  default: "Typical",
  document: "From documents",
  user: "You",
};

// Evidence, not a typical default or a blank reading, that the completed
// project will not have the system.
const saysOut = (r: SystemRow) =>
  r.presence.value === "not_included" && !["", "blank", "suggested"].includes(r.presence.band);

// What the documents say about a system, as one colour: included, not
// included, disagreeing, or nothing read (a typical default or silence).
function evidence(r: SystemRow): { state: string; label: string } {
  if (r.presence.band === "red") return { state: "mixed", label: "Documents disagree" };
  if (saysOut(r)) return { state: "out", label: "Documents say not included" };
  if (r.presence.value === "included" && !["", "blank", "suggested"].includes(r.presence.band))
    return { state: "in", label: r.presence.band === "user" ? "You say included" : "Documents say included" };
  return { state: "none", label: "No document evidence" };
}

/** Inputs that belong to an entry: flags and compliance rows placed there. */
export function entryInputs(data: Profile, id: string) {
  return {
    // A flag the layout does not place still shows, under planning.
    flags: data.details.filter((f) => f.key.startsWith("proj.flag.") && (f.home || PLANNING) === id),
    inputs: data.compliance.flatMap((g) => g.rows).filter((f) => f.home === id),
  };
}

export function ScopeWorks({ projectId, data, tick, selected, onSelect, onScope, onSet, onJump, onSources, onSignedOut }: Props) {
  const [busy, setBusy] = useState(false);
  // A tick shows at once; the rebuilt profile then settles it.
  const [pending, setPending] = useState<Record<string, boolean>>({});
  const [open, setOpen] = useState<string | null>(null);
  const [showAll, setShowAll] = useState(false);
  const [works, setWorks] = useState<WorkRecord[]>([]);
  const [workNote, setWorkNote] = useState("");
  const [reload, setReload] = useState(0);

  const failWork = useCallback((e: unknown) => {
    const message = workError(e);
    if (message === null) onSignedOut();
    else setWorkNote(message);
  }, [onSignedOut]);

  // Saved work follows the profile: a scope tick creates or excludes items.
  useEffect(() => {
    let live = true;
    workApi.list(projectId).then((w) => live && (setWorks(w.items), setWorkNote("")), (e) => live && failWork(e));
    return () => {
      live = false;
    };
  }, [projectId, tick, data, reload, failWork]);

  const isIn = (r: SystemRow) => pending[r.leaf] ?? r.in_scope;
  const needsCheck = (r: SystemRow) => r.presence.band === "red" || (isIn(r) && saysOut(r));
  const run = async (systems: Record<string, ScopeChoice>) => {
    const keys = Object.keys(systems);
    if (keys.length === 0) return;
    setPending((p) => {
      const next = { ...p };
      for (const k of keys) if (systems[k] !== null) next[k] = systems[k] === "in";
      return next;
    });
    setBusy(true);
    try {
      await onScope(systems);
    } finally {
      setBusy(false);
      setPending((p) => {
        const next = { ...p };
        for (const k of keys) delete next[k];
        return next;
      });
    }
  };

  const all = data.systems.flatMap((g) => g.rows);
  const inScope = all.filter(isIn).length;
  const yours = all.filter((r) => r.scope_origin === "user");
  const leaves = new Set(all.map((r) => r.leaf));
  const byLeaf = new Map<string, WorkRecord[]>();
  for (const w of works) byLeaf.set(w.system_id, [...(byLeaf.get(w.system_id) ?? []), w]);
  const orphans = works.filter((w) => !leaves.has(w.system_id));
  const labels = new Map(all.map((r) => [r.leaf, r.label]));

  const entries = [
    { id: PLANNING, label: "Planning and approvals", rows: [] as SystemRow[] },
    ...data.systems,
    ...(orphans.length ? [{ id: OTHER, label: "Other work", rows: [] as SystemRow[] }] : []),
  ];
  const checksIn = (id: string, rows: SystemRow[]) => {
    const { flags, inputs } = entryInputs(data, id);
    return rows.filter(needsCheck).length + [...flags, ...inputs].filter((f) => f.band === "red").length;
  };
  const current = entries.find((e) => e.id === selected) ?? entries[0];

  return (
    <section className="pf-section sw" aria-labelledby="pf-scope">
      <div className="pf-head sw-head">
        <h2 id="pf-scope">Scope and works</h2>
        <span className="muted">
          {inScope} system{inScope === 1 ? "" : "s"} in scope
        </span>
        {data.presets.map((p) => (
          <button
            key={p.id}
            type="button"
            className="btn btn-small"
            disabled={busy}
            onClick={() => run(Object.fromEntries(p.systems.map((s) => [s, "in" as const])))}
          >
            {p.label}
          </button>
        ))}
        {yours.length > 0 && (
          <button
            type="button"
            className="cell-link"
            disabled={busy}
            onClick={() => run(Object.fromEntries(yours.map((r) => [r.leaf, null])))}
            title="Hand every system back to the defaults and documents"
          >
            Reset to defaults
          </button>
        )}
        <label className="pf-toggle sw-all">
          <input type="checkbox" checked={showAll} onChange={(e) => setShowAll(e.target.checked)} /> Show inputs this scope does not need
        </label>
      </div>
      {inScope === 0 && <p className="pf-scope-hint">Tick the systems the works touch, or apply a preset. Compliance inputs follow the scope.</p>}
      {workNote && (
        <div className="banner" role="alert">
          <p>{workNote}</p>
          <button type="button" className="btn btn-small" disabled={busy} onClick={() => setReload((n) => n + 1)}>
            Reload saved work
          </button>
        </div>
      )}

      <div className="sw-body">
        <nav className="sw-rail" aria-label="Scope entries">
          {entries.map((e, i) => {
            const on = e.rows.filter(isIn).length;
            const checks = checksIn(e.id, e.rows);
            const { flags } = entryInputs(data, e.id);
            return (
              <button
                key={e.id}
                type="button"
                className="sw-entry"
                aria-current={e.id === current.id ? "true" : undefined}
                data-active={on > 0 ? "true" : undefined}
                onClick={() => {
                  onSelect(e.id);
                  setOpen(null);
                }}
              >
                <span className="sw-num" aria-hidden="true">{e.id === OTHER ? "·" : i}</span>
                <span className="sw-entry-label">{e.label}</span>
                {checks > 0 && <span className="sw-check" title={`${checks} to check`}>⚠ {checks}</span>}
                <span className="sw-count">
                  {e.id === PLANNING ? `${flags.filter((f) => f.value !== "").length}/${flags.length}` : e.id === OTHER ? orphans.length : `${on}/${e.rows.length}`}
                </span>
              </button>
            );
          })}
        </nav>

        <div className="sw-pane" role="region" aria-label={current.label}>
          <div className="sw-pane-head">
            {current.rows.length > 0 && (() => {
              const on = current.rows.filter(isIn).length;
              return (
                <label className="sw-group-tick">
                  <input
                    type="checkbox"
                    aria-label={`All of ${current.label}`}
                    checked={on > 0 && on === current.rows.length}
                    ref={(el) => {
                      if (el) el.indeterminate = on > 0 && on < current.rows.length;
                    }}
                    disabled={busy}
                    onChange={() => {
                      const every = on === current.rows.length;
                      run(Object.fromEntries(current.rows.filter((r) => isIn(r) === every).map((r) => [r.leaf, every ? "out" : "in"])));
                    }}
                  />
                </label>
              );
            })()}
            <h3>{current.label}</h3>
            {current.rows.length > 0 && <span className="muted">{current.rows.filter(isIn).length} of {current.rows.length} in scope</span>}
            {current.id !== PLANNING && current.id !== OTHER && (
              <button type="button" className="cell-link sw-req" onClick={() => onSources(current.id)} aria-label={`View requirements for ${current.label}`}>
                Requirements
              </button>
            )}
          </div>

          {current.rows.length > 0 && (
            <div className="sw-table" role="table" aria-label={`${current.label} systems`}>
              <div className="sw-row sw-th" role="row">
                <span role="columnheader">System</span>
                <span role="columnheader">Works</span>
                <span role="columnheader">By</span>
                <span role="columnheader">Evidence</span>
                <span role="columnheader"><span className="sr-only">Details</span></span>
              </div>
              {current.rows.map((r) => (
                <SystemLine
                  key={r.leaf}
                  projectId={projectId}
                  row={r}
                  items={byLeaf.get(r.leaf) ?? []}
                  parts={data.parts}
                  inScope={isIn(r)}
                  open={open === r.leaf}
                  busy={busy}
                  setBusy={setBusy}
                  onOpen={() => setOpen(open === r.leaf ? null : r.leaf)}
                  onTick={(choice) => run({ [r.leaf]: choice })}
                  onSet={onSet}
                  onJump={onJump}
                  onSources={() => onSources(r.leaf)}
                  onWorkChanged={() => setReload((n) => n + 1)}
                  onWorkError={failWork}
                  onWorks={setWorks}
                />
              ))}
            </div>
          )}

          {current.id === OTHER && (
            <WorkList projectId={projectId} items={orphans} parts={data.parts} label={(w) => labels.get(w.system_id) ?? w.system_id} busy={busy} setBusy={setBusy} onChanged={() => setReload((n) => n + 1)} onError={failWork} onWorks={setWorks} />
          )}

          <EntryInputs data={data} id={current.id} showAll={showAll} onSet={onSet} onJump={onJump} />
        </div>
      </div>
    </section>
  );
}

function EntryInputs({ data, id, showAll, onSet, onJump }: { data: Profile; id: string; showAll: boolean; onSet: Setter; onJump: (id: string) => void }) {
  const { flags, inputs } = entryInputs(data, id);
  const shown = inputs.filter((f) => showAll || f.relevant !== false || f.band === "red");
  const hidden = inputs.length - shown.length;
  return (
    <>
      {flags.length > 0 && (
        <div className="sw-inputs" role="group" aria-label={id === PLANNING ? "Planning and approvals" : "Site risks and features"}>
          <h4>{id === PLANNING ? "Approvals and stakeholders" : "Site risks and features"} <span className="muted">{flags.filter((f) => f.value !== "").length} of {flags.length} answered</span></h4>
          <div className="pf-grid sw-grid">
            {flags.map((f) => <FieldRow key={f.key} field={f} onSet={onSet} onJump={onJump} compact />)}
          </div>
        </div>
      )}
      {inputs.length > 0 && (
        <div className="sw-inputs" role="group" aria-label="Compliance inputs">
          <h4>Compliance inputs {hidden > 0 && <span className="muted">{hidden} not needed for this scope</span>}</h4>
          {shown.length > 0 && (
            <div className="pf-grid">
              {shown.map((f: ProfileField) => <FieldRow key={f.key + f.part_id} field={f} onSet={onSet} onJump={onJump} parts={data.parts} />)}
            </div>
          )}
        </div>
      )}
    </>
  );
}

function SystemLine({ projectId, row: r, items, parts, inScope, open, busy, setBusy, onOpen, onTick, onSet, onJump, onSources, onWorkChanged, onWorkError, onWorks }: {
  projectId: string; row: SystemRow; items: WorkRecord[]; parts: Profile["parts"]; inScope: boolean; open: boolean;
  busy: boolean; setBusy: (b: boolean) => void; onOpen: () => void; onTick: (choice: ScopeChoice) => void;
  onSet: Setter; onJump: (id: string) => void; onSources: () => void;
  onWorkChanged: () => void; onWorkError: (e: unknown) => void; onWorks: (f: (w: WorkRecord[]) => WorkRecord[]) => void;
}) {
  const base = `sys.${r.leaf}`;
  const ev = evidence(r);
  const own = items.length === 1 && !items[0].is_group ? items[0] : undefined;
  const [note, setNote] = useState(r.note.value);
  useEffect(() => setNote(r.note.value), [r.note.value]);
  const check = r.presence.band === "red" || (inScope && saysOut(r));

  const setAction = async (work: WorkRecord, action: string) => {
    setBusy(true);
    onWorks((old) => old.map((w) => (w.id === work.id ? { ...w, action } : w)));
    try {
      await patchWorkRow(projectId, work, { action });
      onWorkChanged();
    } catch (e) {
      onWorks((old) => old.map((w) => (w.id === work.id ? work : w)));
      onWorkError(e);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="sw-line" data-open={open ? "true" : undefined} data-check={check ? "true" : undefined}>
      <div className="sw-row" role="row">
        <span className="sw-name" role="cell">
          <label>
            <input type="checkbox" checked={inScope} disabled={busy} onChange={() => onTick(inScope ? "out" : "in")} />
            {r.label}
          </label>
          {r.scope_origin && (
            <span className="pf-scope-origin" data-origin={r.scope_origin}>
              {r.in_scope || r.scope_origin !== "user" ? ORIGINS[r.scope_origin] : "Removed by you"}
            </span>
          )}
        </span>
        <span role="cell">
          {own ? (
            <select className="select sw-select" aria-label={`Action for ${own.title}`} disabled={busy || own.deprecated || !inScope} value={own.action} onChange={(e) => void setAction(own, e.target.value)}>
              {WORK_ACTIONS.map((a) => <option key={a} value={a}>{a}</option>)}
            </select>
          ) : items.length > 1 ? (
            <button type="button" className="cell-link" onClick={onOpen}>{items.length} work items</button>
          ) : (
            <span className="muted">—</span>
          )}
        </span>
        <span role="cell">
          {saysOut(r) && !inScope ? (
            <span className="pf-scope-out">Not included</span>
          ) : (
            <select
              className="select sw-select"
              aria-label={`Who provides ${r.label}`}
              data-default={r.provider.value ? undefined : "true"}
              disabled={!inScope}
              value={r.provider.value}
              onChange={(e) =>
                e.target.value
                  ? onSet(`${base}.provider`, { part_id: r.part_id, value: e.target.value })
                  : onSet(`${base}.provider`, { part_id: r.part_id, reset: true })
              }
            >
              {/* Most works are the contractor's; shown, not stored, until a document or the user says. */}
              <option value="">{inScope ? "Contractor (default)" : "—"}</option>
              {Object.entries(PROVIDERS).map(([id, label]) => <option key={id} value={id}>{label}</option>)}
            </select>
          )}
        </span>
        <span role="cell" className="sw-ev">
          <span className="pf-ev" data-ev={ev.state} title={ev.label} />
          <span className="sr-only">{ev.label}</span>
          {r.presence.sources.length > 0 && <span className="pf-scope-count">{r.presence.sources.length}</span>}
          {check && <span className="sw-check" title={r.presence.band === "red" ? "Documents disagree" : "Documents say not included"}>⚠</span>}
        </span>
        <span role="cell">
          <button type="button" className="sw-more" aria-expanded={open} aria-label={`Details for ${r.label}`} onClick={onOpen}>
            {open ? "▾" : "▸"}
          </button>
        </span>
      </div>
      {saysOut(r) && inScope && (
        <p className="pf-scope-check">
          Documents say not included{" "}
          <button type="button" className="cell-link" disabled={busy} onClick={() => onTick("out")}>
            Remove from scope
          </button>
        </p>
      )}
      {r.scope_note && <p className="pf-scope-note">{r.scope_note}</p>}
      {open && (
        <div className="sw-strip">
          <div className="sw-strip-line">
            <input
              className="input pf-note"
              aria-label={`Note for ${r.label}`}
              maxLength={120}
              value={note}
              placeholder={r.presence.note ?? "Note"}
              onChange={(e) => setNote(e.target.value)}
              onBlur={() => note !== r.note.value && onSet(`${base}.note`, { part_id: r.part_id, value: note })}
            />
            <span className="pf-sys-marks">
              {r.presence.band !== "suggested" && <Mark cell={r.presence} />}
              <Mark cell={r.provider.band === "red" ? r.provider : { ...r.provider, band: "" }} />
              <Conflict cell={r.presence} onPick={(v) => onSet(`${base}.presence`, { part_id: r.part_id, value: v })} />
              <Conflict cell={r.provider} onPick={(v) => onSet(`${base}.provider`, { part_id: r.part_id, value: v })} />
              <Sources cell={r.presence} onJump={onJump} />
              <button type="button" className="cell-link" onClick={onSources} aria-label={`View requirements for ${r.label}`}>Requirements</button>
              {r.presence.band === "user" && <ResetButton onClick={() => onSet(`${base}.presence`, { part_id: r.part_id, reset: true })} />}
            </span>
          </div>
          {items.length > 0 && (
            <WorkList projectId={projectId} items={items} parts={parts} label={() => r.label} busy={busy} setBusy={setBusy} onChanged={onWorkChanged} onError={onWorkError} onWorks={onWorks} />
          )}
        </div>
      )}
    </div>
  );
}

// Work items under one system. The item a scope tick owns has no Included
// box of its own: the tick above is its inclusion.
function WorkList({ projectId, items, parts, label, busy, setBusy, onChanged, onError, onWorks }: {
  projectId: string; items: WorkRecord[]; parts: Profile["parts"]; label: (w: WorkRecord) => string;
  busy: boolean; setBusy: (b: boolean) => void; onChanged: () => void; onError: (e: unknown) => void;
  onWorks: (f: (w: WorkRecord[]) => WorkRecord[]) => void;
}) {
  const [selected, setSelected] = useState("");
  const part = (w: WorkRecord) => parts.find((p) => p.id === w.part_id)?.label ?? "Location unavailable";
  const setRow = async (work: WorkRecord, change: { action?: string; inclusion?: string }) => {
    setBusy(true);
    onWorks((old) => old.map((w) => (w.id === work.id ? { ...w, ...change } : w)));
    try {
      await patchWorkRow(projectId, work, change);
      onChanged();
    } catch (e) {
      onWorks((old) => old.map((w) => (w.id === work.id ? work : w)));
      onError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <ul className="sw-works" aria-label="Work items">
      {items.map((w) => {
        const editable = !busy && !w.deprecated;
        return (
          <li key={w.id}>
            <div className="sw-work">
              <button type="button" className="works-open" aria-expanded={w.id === selected} onClick={() => setSelected(w.id === selected ? "" : w.id)}>
                {w.title}
                <span>{part(w)}{w.is_group ? " · group" : ""}</span>
              </button>
              {!(items.length === 1 && !w.is_group) && (
                <select className="select sw-select" aria-label={`Action for ${w.title}`} disabled={!editable} value={w.action} onChange={(e) => void setRow(w, { action: e.target.value })}>
                  {WORK_ACTIONS.map((a) => <option key={a} value={a}>{a}</option>)}
                </select>
              )}
              {!w.coarse_key && (
                <label className="works-included">
                  <input type="checkbox" disabled={!editable} checked={w.inclusion === "included"} onChange={(e) => void setRow(w, { inclusion: e.target.checked ? "included" : "excluded" })} /> Included
                </label>
              )}
            </div>
            {w.id === selected && (
              <WorkItemDetail projectId={projectId} item={w} parts={parts} systemLabel={label(w)} busy={busy} setBusy={setBusy} onChanged={onChanged} onError={onError} />
            )}
          </li>
        );
      })}
    </ul>
  );
}
