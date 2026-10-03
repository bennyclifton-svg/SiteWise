// Scope of works: which building systems the works touch. The checklist and
// the compliance rows follow it. Defaults come from the building class and
// work type, documents can add systems, and the user's choice is final.

import { useState } from "react";
import type { Profile, ScopeChoice, SystemRow } from "./profileApi";

interface Props {
  data: Profile;
  onSet: (systems: Record<string, ScopeChoice>) => Promise<void>;
}

const ORIGINS: Record<string, string> = {
  default: "Typical",
  document: "From documents",
  user: "You",
};

export function ScopePicker({ data, onSet }: Props) {
  const [busy, setBusy] = useState(false);
  // A tick shows at once; the rebuilt profile then settles it.
  const [pending, setPending] = useState<Record<string, boolean>>({});
  const isIn = (r: SystemRow) => pending[r.leaf] ?? r.in_scope;
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
          const groupId = `scope-${g.id}`;
          return (
            <details className="pf-scope-group" key={g.id} data-active={on > 0 ? "true" : undefined}>
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
                <span className="muted">
                  {on} of {g.rows.length}
                </span>
              </summary>
              <ul aria-labelledby={groupId}>
                {g.rows.map((r) => (
                  <li key={r.leaf}>
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
                  </li>
                ))}
              </ul>
            </details>
          );
        })}
      </div>
    </section>
  );
}
