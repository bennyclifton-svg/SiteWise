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

import { useCallback, useEffect, useRef, useState } from "react";
import { Conflict, FieldRow, NotYetKnown, ResetButton, unanswered, type Setter } from "./ProfileFields";
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
  /** Opens the source requirements filtered to a system or group (no longer shown here). */
  onSources?: (system: string) => void;
  onSignedOut: () => void;
}

// The viewer's "Active only" choice, kept across visits on this browser.
const ACTIVE_KEY = "sitewise.scope.activeOnly";
function readActiveOnly() {
  try {
    return localStorage.getItem(ACTIVE_KEY) === "1";
  } catch {
    return false;
  }
}
function storeActiveOnly(on: boolean) {
  try {
    localStorage.setItem(ACTIVE_KEY, on ? "1" : "0");
  } catch {
    // Storage blocked: the toggle still works for this visit.
  }
}

const PROVIDERS: Record<string, string> = { contractor: "Contractor", owner: "Owner", landlord: "Landlord", others: "Others" };

// Evidence, not a typical default or a blank reading, that the completed
// project will not have the system.
const saysOut = (r: SystemRow) =>
  r.presence.value === "not_included" && !["", "blank", "suggested"].includes(r.presence.band);

// Who settled a system's scope, as one dot colour (owner, 8 Nov 2026):
// green from documents, blue typical for the building, rust set by you.
// Disagreeing documents keep the amber half dot; the count beside the dot
// is the number of source passages.
function evidence(r: SystemRow): { state: string; label: string } {
  if (r.presence.band === "red") return { state: "mixed", label: "Documents disagree" };
  if (r.scope_origin === "user") return { state: "user", label: r.in_scope ? "Set by you" : "Removed by you" };
  if (r.scope_origin === "document" || saysOut(r))
    return { state: "doc", label: saysOut(r) ? "Documents say not included" : "From documents" };
  if (r.scope_origin === "default") return { state: "typical", label: "Typical for this building" };
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

export function ScopeWorks({ projectId, data, tick, selected, onSelect, onScope, onSet, onJump, onSignedOut }: Props) {
  const [busy, setBusy] = useState(false);
  // A tick shows at once; the rebuilt profile then settles it.
  const [pending, setPending] = useState<Record<string, boolean>>({});
  const [open, setOpen] = useState<string | null>(null);
  // Active only hides every system, cell and entry not in scope. A system
  // still to check stays, so a warning never disappears.
  const [activeOnly, setActiveOnly] = useState(readActiveOnly);
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
  const answered = (id: string) => entryInputs(data, id).flags.filter((f) => !f.scope_decision && f.value !== "").length;
  const shownEntries = activeOnly
    ? entries.filter((e) =>
        e.id === OTHER || checksIn(e.id, e.rows) > 0 || (e.id === PLANNING ? answered(e.id) > 0 : e.rows.some(isIn)))
    : entries;
  // Nothing open until a line is clicked; clicking the open line closes it.
  const current = shownEntries.find((e) => e.id === selected);

  return (
    <section className="pf-section sw" aria-labelledby="pf-scope">
      <div className="pf-head sw-head section-band">
        <h2 id="pf-scope">Scope and works</h2>
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
        <div className="sw-keyrow">
          <div className="sw-key" aria-hidden="true">
            <span><i className="sw-cell" data-on="true" />In scope</span>
            <span><i className="sw-cell" />Not in scope</span>
            <span><i className="sw-cell" data-check="true" />To check</span>
            <span className="sw-key-gap"><i className="pf-ev" data-ev="doc" />Documents</span>
            <span><i className="pf-ev" data-ev="typical" />Typical</span>
            <span><i className="pf-ev" data-ev="user" />You</span>
          </div>
          <label className="pf-toggle sw-all" title="Show only the systems in scope">
            <input
              type="checkbox"
              checked={activeOnly}
              onChange={(e) => {
                setActiveOnly(e.target.checked);
                storeActiveOnly(e.target.checked);
              }}
            />{" "}
            Active only
          </label>
        </div>
      </div>
      {/* Presets start an empty scope; once anything is in scope a preset
          would only add systems nobody chose, so they step aside. */}
      {inScope === 0 && (
        <div className="pf-scope-start">
          <p className="pf-scope-hint">Tick the systems the works touch, or start from a preset. Compliance inputs follow the scope.</p>
          {data.presets.length > 0 && (
            <div className="pf-scope-presets" role="group" aria-label="Scope presets">
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
            </div>
          )}
        </div>
      )}
      {workNote && (
        <div className="banner" role="alert">
          <p>{workNote}</p>
          <button type="button" className="btn btn-small" disabled={busy} onClick={() => setReload((n) => n + 1)}>
            Reload saved work
          </button>
        </div>
      )}

      <div className="sw-body sw-matrix">
        <div className="sw-rail" role="group" aria-label="Scope entries">
          {activeOnly && shownEntries.length === 0 && <p className="sw-empty muted">Nothing in scope yet. Turn off Active only to choose systems.</p>}
          {shownEntries.map((e) => {
            const on = e.rows.filter(isIn).length;
            const checks = checksIn(e.id, e.rows);
            // Planning's cells are its conditions; appointment choices are not cells.
            const flags = entryInputs(data, e.id).flags.filter((f) => !f.scope_decision);
            const cells = e.id === PLANNING
              ? flags.map((f) => ({ key: f.key, on: f.value !== "", check: f.band === "red" }))
              : e.id === OTHER
                ? orphans.map((w) => ({ key: w.id, on: true, check: false }))
                : e.rows.map((r) => ({ key: r.leaf, on: isIn(r), check: needsCheck(r) }));
            const shownCells = activeOnly ? cells.filter((c) => c.on || c.check) : cells;
            const active = e.id === current?.id;
            return (
              <div key={e.id} className="sw-matrow" data-open={active || undefined}>
                <button
                  type="button"
                  className="sw-entry"
                  aria-current={active ? "true" : undefined}
                  aria-expanded={active}
                  data-active={on > 0 ? "true" : undefined}
                  onClick={() => {
                    onSelect(active ? "" : e.id);
                    setOpen(null);
                  }}
                >
                  <span className="sw-entry-label">
                    <span className="sw-caret" aria-hidden="true" />
                    {e.label}
                  </span>
                  <span className="sw-cells" aria-hidden="true">
                    {shownCells.map((c) => <i key={c.key} className="sw-cell" data-on={c.on || undefined} data-check={c.check || undefined} />)}
                  </span>
                  {checks > 0 && <span className="sw-check" title={`${checks} to check`}>{checks} to check</span>}
                  <span className="sw-count">
                    {e.id === PLANNING ? `${flags.filter((f) => f.value !== "").length} / ${flags.length}` : e.id === OTHER ? orphans.length : `${on} / ${e.rows.length}`}
                  </span>
                </button>
                {active && current && (
            <div className="sw-pane" role="region" aria-label={current.label}>
              {/* No title line or column heads: the entry row above names it,
                  the rows are system then dot (owner, 8 Nov 2026). */}
              {(() => {
                // Two tables side by side halve the rows; the first reads
                // down, then the second.
                const rows = activeOnly ? current.rows.filter((r) => isIn(r) || needsCheck(r)) : current.rows;
                if (rows.length === 0) return null;
                const half = Math.ceil(rows.length / 2);
                const cols = rows.length > 1 ? [rows.slice(0, half), rows.slice(half)] : [rows];
                return (
                  <div className="sw-cols" data-cols={cols.length}>
                    {cols.map((col, i) => (
                      <div key={i} className="sw-table" role="table" aria-label={`${current.label} systems${cols.length > 1 ? `, column ${i + 1}` : ""}`}>
                        {col.map((r) => (
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
                            onWorkChanged={() => setReload((n) => n + 1)}
                            onWorkError={failWork}
                            onWorks={setWorks}
                          />
                        ))}
                      </div>
                    ))}
                  </div>
                );
              })()}

              {current.id === OTHER && (
                <WorkList projectId={projectId} items={orphans} parts={data.parts} label={(w) => labels.get(w.system_id) ?? w.system_id} busy={busy} setBusy={setBusy} onChanged={() => setReload((n) => n + 1)} onError={failWork} onWorks={setWorks} />
              )}

              <EntryInputs data={data} id={current.id} onSet={onSet} onJump={onJump} />
            </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}

function EntryInputs({ data, id, onSet, onJump }: { data: Profile; id: string; onSet: Setter; onJump: (id: string) => void }) {
  // Inputs this scope does not need stay hidden until asked for.
  const [showAll, setShowAll] = useState(false);
  const { flags: all, inputs } = entryInputs(data, id);
  const flags = all.filter((f) => !f.scope_decision);
  // What each consultant is asked to do here. Answered ones show; the
  // unknown tail folds into one line.
  const choices = all.filter((f) => f.scope_decision);
  const shown = inputs.filter((f) => showAll || f.relevant !== false || f.band === "red");
  const hidden = inputs.length - shown.length;
  const choice = (f: ProfileField) => <FieldRow key={f.key} field={f} onSet={onSet} onJump={onJump} compact />;
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
      {choices.length > 0 && (
        <div className="sw-inputs" role="group" aria-label="Consultant appointment choices">
          <h4>Consultant appointment choices <span className="muted">{choices.filter((f) => !unanswered(f)).length} of {choices.length} answered · also set in each RFP</span></h4>
          <div className="pf-grid sw-grid">
            {choices.filter((f) => !unanswered(f)).map(choice)}
            <NotYetKnown fields={choices.filter(unanswered)} render={choice} />
          </div>
        </div>
      )}
      {inputs.length > 0 && (
        <div className="sw-inputs" role="group" aria-label="Compliance inputs">
          <h4>
            Compliance inputs
            {hidden > 0 && <span className="muted">{hidden} not needed for this scope</span>}
            {(hidden > 0 || showAll) && (
              <button type="button" className="cell-link" onClick={() => setShowAll(!showAll)}>
                {showAll ? "Hide unneeded" : "Show"}
              </button>
            )}
          </h4>
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

// One system: its scope tick and name, then the dot. Works and supplier are
// almost always "new" and "Contractor", so they live in the dot's pop-up and
// only an exception shows, as a short note after the name (owner, 8 Nov 2026).
function SystemLine({ projectId, row: r, items, parts, inScope, open, busy, setBusy, onOpen, onTick, onSet, onJump, onWorkChanged, onWorkError, onWorks }: {
  projectId: string; row: SystemRow; items: WorkRecord[]; parts: Profile["parts"]; inScope: boolean; open: boolean;
  busy: boolean; setBusy: (b: boolean) => void; onOpen: () => void; onTick: (choice: ScopeChoice) => void;
  onSet: Setter; onJump: (id: string) => void;
  onWorkChanged: () => void; onWorkError: (e: unknown) => void; onWorks: (f: (w: WorkRecord[]) => WorkRecord[]) => void;
}) {
  const own = items.length === 1 && !items[0].is_group ? items[0] : undefined;
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

  const exceptions = [
    own && own.action !== "new" ? own.action : "",
    !own && items.length > 0 ? `${items.length} work item${items.length === 1 ? "" : "s"}` : "",
    r.provider.value && r.provider.value !== "contractor" ? `by ${PROVIDERS[r.provider.value] ?? r.provider.value}` : "",
  ].filter(Boolean);

  return (
    <div className="sw-line" data-open={open ? "true" : undefined} data-check={check ? "true" : undefined}>
      <div className="sw-row" role="row">
        <span className="sw-name" role="cell">
          <label>
            <input type="checkbox" checked={inScope} disabled={busy} onChange={() => onTick(inScope ? "out" : "in")} />
            {r.label}
          </label>
          {exceptions.length > 0 && <span className="sw-except">{exceptions.join(" · ")}</span>}
        </span>
        <span role="cell" className="sw-ev">
          <SystemPop row={r} check={check} own={own} items={items} inScope={inScope} busy={busy} open={open} onOpen={onOpen} onAction={setAction} onSet={onSet} onJump={onJump} />
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
      {open && items.length > 0 && (
        <div className="sw-strip">
          <WorkList projectId={projectId} items={items} parts={parts} label={() => r.label} busy={busy} setBusy={setBusy} onChanged={onWorkChanged} onError={onWorkError} onWorks={onWorks} />
        </div>
      )}
    </div>
  );
}

// The dot, its source count and the system's pop-up: where the evidence
// came from (each passage with a link to its document, disagreeing readings
// to pick from, a reset when the choice is yours), then the exceptions:
// works, who supplies it, and the work item's parts and split.
function SystemPop({ row: r, check, own, items, inScope, busy, open: workOpen, onOpen, onAction, onSet, onJump }: {
  row: SystemRow; check: boolean; own: WorkRecord | undefined; items: WorkRecord[]; inScope: boolean; busy: boolean;
  open: boolean; onOpen: () => void; onAction: (w: WorkRecord, action: string) => void; onSet: Setter; onJump: (id: string) => void;
}) {
  const ref = useRef<HTMLDetailsElement>(null);
  const [open, setOpen] = useState(false);
  const close = () => {
    if (ref.current) ref.current.open = false;
  };
  // Clicking outside the pop-up, or Escape, closes it.
  useEffect(() => {
    if (!open) return;
    const onDown = (e: PointerEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) close();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    document.addEventListener("pointerdown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const base = `sys.${r.leaf}`;
  const ev = evidence(r);
  const n = r.presence.sources.length;
  const picks = (r.presence.band === "red" || r.presence.band === "amber") && r.presence.alternatives.length > 0;
  const mine = r.presence.band === "user";
  return (
    <details className="sw-evpop" ref={ref} onToggle={(e) => setOpen(e.currentTarget.open)}>
      <summary
        className="sw-evmark"
        title={`${ev.label}. Sources, works and supplier`}
        aria-label={`${r.label}: ${ev.label}, ${n} source${n === 1 ? "" : "s"}. Sources, works and supplier`}
      >
        <span className="pf-ev" data-ev={ev.state} />
        {n > 0 && <span className="pf-scope-count">{n}</span>}
        {check && <span className="sw-check">⚠</span>}
      </summary>
      <div className="sw-evpanel" role="group" aria-label={`Details for ${r.label}`}>
        <p className="sw-evtitle">{ev.label}</p>
        {n > 0 && (
          <ul>
            {r.presence.sources.map((s, i) => (
              <li key={i}>
                {s.excerpt ? <q>{s.excerpt}</q> : <span className="muted">No excerpt</span>}{" "}
                <button type="button" className="cell-link" onClick={() => onJump(s.document_id)}>
                  Show in register
                </button>
              </li>
            ))}
          </ul>
        )}
        {picks && <Conflict cell={r.presence} onPick={(v) => onSet(`${base}.presence`, { part_id: r.part_id, value: v })} />}
        {mine && (
          <p className="sw-evreset">
            Set by you <ResetButton onClick={() => onSet(`${base}.presence`, { part_id: r.part_id, reset: true })} />
          </p>
        )}

        <div className="sw-evfields">
          <label>
            <span>Works</span>
            {own ? (
              <select
                className="select sw-select"
                aria-label={`Action for ${own.title}`}
                disabled={busy || own.deprecated || !inScope}
                value={own.action}
                onChange={(e) => onAction(own, e.target.value)}
              >
                {WORK_ACTIONS.map((a) => <option key={a} value={a}>{a}</option>)}
              </select>
            ) : (
              <span className="muted">{items.length > 0 ? `${items.length} work items, set in each` : "—"}</span>
            )}
          </label>
          <label>
            <span>Supplied by</span>
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
          </label>
          <Conflict cell={r.provider} onPick={(v) => onSet(`${base}.provider`, { part_id: r.part_id, value: v })} />
          {items.length > 0 && (
            <button
              type="button"
              className="cell-link"
              aria-expanded={workOpen}
              onClick={() => {
                close();
                onOpen();
              }}
            >
              {workOpen ? "Close parts and split" : "Parts and split…"}
            </button>
          )}
        </div>
      </div>
    </details>
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
