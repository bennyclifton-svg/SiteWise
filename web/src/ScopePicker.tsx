// Scope of works: which building systems the works touch, and for each one
// in scope who provides it and what the documents say. The tick is the one
// decision; compliance follows it. Defaults come from the building class and
// work type, documents can add systems, and the user's choice is final: a
// document that says a ticked system is not included is flagged, never
// allowed to untick it.

import { useState, type ReactNode } from "react";
import type { Profile, ScopeChoice, SystemRow } from "./profileApi";

interface Props {
  data: Profile;
  onSet: (systems: Record<string, ScopeChoice>) => Promise<void>;
  /** Provider, evidence and note for a system, shown when its row opens. */
  detail: (row: SystemRow) => ReactNode;
}

const PROVIDERS: Record<string, string> = { contractor: "Contractor", owner: "Owner", others: "Others" };

// What the documents say about a system, as one colour: included, not
// included, disagreeing, or nothing read (a typical default or silence).
function evidence(r: SystemRow): { state: string; label: string } {
  if (r.presence.band === "red") return { state: "mixed", label: "Documents disagree" };
  if (saysOut(r)) return { state: "out", label: "Documents say not included" };
  if (r.presence.value === "included" && !["", "blank", "suggested"].includes(r.presence.band))
    return { state: "in", label: r.presence.band === "user" ? "You say included" : "Documents say included" };
  return { state: "none", label: "No document evidence" };
}

// Evidence, not a typical default or a blank reading, that the completed
// project will not have the system.
const saysOut = (r: SystemRow) =>
  r.presence.value === "not_included" && !["", "blank", "suggested"].includes(r.presence.band);

const ORIGINS: Record<string, string> = {
  default: "Typical",
  document: "From documents",
  user: "You",
};

export function ScopePicker({ data, onSet, detail }: Props) {
  const [busy, setBusy] = useState(false);
  // Groups open themselves when something needs checking, until the user
  // opens or closes them.
  const [opened, setOpened] = useState<Record<string, boolean>>({});
  const [expanded, setExpanded] = useState<string | null>(null);
  // A tick shows at once; the rebuilt profile then settles it.
  const [pending, setPending] = useState<Record<string, boolean>>({});
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
      await onSet(systems);
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

  return (
    <section className="pf-section" aria-labelledby="pf-scope">
      <div className="pf-head pf-scope-head">
        <h2 id="pf-scope">Scope of works</h2>
        <span className="muted">
          {inScope} system{inScope === 1 ? "" : "s"}
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
      </div>
      <p className="pf-scope-hint">
        {inScope === 0
          ? "Tick the systems the works touch, or apply a preset. The checklist and compliance follow the scope."
          : "The checklist and compliance show what these systems need."}
      </p>
      <div className="pf-scope-groups">
        {data.systems.map((g) => {
          const on = g.rows.filter(isIn).length;
          const checks = g.rows.filter(needsCheck).length;
          const groupId = `scope-${g.id}`;
          return (
            <details
              className="pf-scope-group"
              key={g.id}
              data-active={on > 0 ? "true" : undefined}
              open={opened[g.id] ?? checks > 0}
              onToggle={(e) => {
                const open = e.currentTarget.open;
                if (open !== (opened[g.id] ?? checks > 0)) setOpened((o) => ({ ...o, [g.id]: open }));
              }}
            >
              <summary>
                <input
                  type="checkbox"
                  aria-label={`All of ${g.label}`}
                  checked={on > 0 && on === g.rows.length}
                  ref={(el) => {
                    if (el) el.indeterminate = on > 0 && on < g.rows.length;
                  }}
                  disabled={busy}
                  onClick={(e) => e.stopPropagation()}
                  onChange={() => {
                    const every = on === g.rows.length;
                    run(Object.fromEntries(g.rows.filter((r) => isIn(r) === every).map((r) => [r.leaf, every ? "out" : "in"])));
                  }}
                />
                <span id={groupId} className="pf-scope-label">
                  {g.label}
                </span>
                {checks > 0 && <span className="pf-scope-check">{checks} to check</span>}
                <span className="muted">
                  {on} of {g.rows.length}
                </span>
              </summary>
              <ul aria-labelledby={groupId}>
                {g.rows.map((r) => {
                  const ev = evidence(r);
                  const provider = PROVIDERS[r.provider.value];
                  const open = expanded === r.leaf;
                  return (
                    <li key={r.leaf} data-open={open ? "true" : undefined}>
                      <label>
                        <input
                          type="checkbox"
                          checked={isIn(r)}
                          onChange={() => run({ [r.leaf]: isIn(r) ? "out" : "in" })}
                        />
                        {r.label}
                      </label>
                      {r.scope_origin && (
                        <span className="pf-scope-origin" data-origin={r.scope_origin}>
                          {r.in_scope || r.scope_origin !== "user" ? ORIGINS[r.scope_origin] : "Removed by you"}
                        </span>
                      )}
                      {r.scope_note && <span className="pf-scope-note">{r.scope_note}</span>}
                      {saysOut(r) && isIn(r) && (
                        <span className="pf-scope-check">
                          Documents say not included{" "}
                          <button type="button" className="cell-link" disabled={busy} onClick={() => run({ [r.leaf]: "out" })}>
                            Remove from scope
                          </button>
                        </span>
                      )}
                      <button
                        type="button"
                        className="pf-scope-more"
                        aria-expanded={open}
                        aria-label={`Details for ${r.label}`}
                        onClick={() => setExpanded(open ? null : r.leaf)}
                      >
                        <span className="pf-ev" data-ev={ev.state} title={ev.label} />
                        <span className="sr-only">{ev.label}</span>
                        {saysOut(r) && !isIn(r) ? (
                          <span className="pf-scope-out">Not included</span>
                        ) : provider ? (
                          <span className="pf-scope-provider" data-provider={r.provider.value}>{provider}</span>
                        ) : (
                          isIn(r) && (
                            // Most works are the contractor's; shown, not stored, until a document or the user says.
                            <span className="pf-scope-provider" data-default="true" title="Default: no document names who provides this">
                              Contractor
                            </span>
                          )
                        )}
                        {r.presence.sources.length > 0 && <span className="pf-scope-count">{r.presence.sources.length}</span>}
                        <span aria-hidden="true" className="pf-scope-caret">{open ? "▾" : "▸"}</span>
                      </button>
                      {open && detail(r)}
                    </li>
                  );
                })}
              </ul>
            </details>
          );
        })}
      </div>
    </section>
  );
}
