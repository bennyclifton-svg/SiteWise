// The project profile: what the filed documents say the project is, drafted
// in the background by Jev and code, polished here by the user. Every value
// says how it was decided; the user's word is final.

import { useCallback, useEffect, useState, type FormEvent } from "react";
import { ProfileSources } from "./ProfileSources";
import { ApiError } from "./api";
import { IconCheck, IconConfirmed, IconNotChecked, IconYou } from "./icons";
import { profileApi, type Cell, type Profile as ProfileData, type ProfileField, type SystemRow } from "./profileApi";

interface Props {
  projectId: string;
  /** Changes when a profile event arrives; the panel refetches. */
  tick: number;
  onJump: (documentId: string) => void;
  onSignedOut: () => void;
}

const GROUP_LABELS: Record<string, string> = {
  classification: "Classification",
  site: "Site",
  services: "Services",
  fire: "Fire",
};

const ASSERTIONS: Record<string, string> = {
  stated: "Stated",
  required: "Required",
  allowance: "Allowance",
};

const REASONS: Record<string, string> = {
  unreviewed: "Awaiting owner review of the rule and table",
  pending: "Pending: table not verified",
  missing_evidence: "Needs stated inputs",
  conflict: "Inputs disagree",
  mixed_scope: "Inputs belong to different parts",
  no_row: "No table row for these inputs",
  not_built: "Not built yet",
};

export function Profile({ projectId, tick, onJump, onSignedOut }: Props) {
  const [sourceSystem, setSourceSystem] = useState("");
  const [sourcesOpen, setSourcesOpen] = useState(false);
  const [data, setData] = useState<ProfileData | null>(null);
  const [error, setError] = useState("");
  const [showAll, setShowAll] = useState(false);
  const [requesting, setRequesting] = useState(false);
  const [requestNote, setRequestNote] = useState("");

  const fail = useCallback(
    (e: unknown) => {
      if (e instanceof ApiError && e.status === 401) return onSignedOut();
      setError(e instanceof Error ? e.message : "Couldn't load the profile.");
    },
    [onSignedOut],
  );

  useEffect(() => {
    let live = true;
    profileApi.get(projectId).then((p) => live && (setData(p), setError("")), (e) => live && fail(e));
    return () => {
      live = false;
    };
  }, [projectId, tick, fail]);

  // Completion can follow the profile event, or an SSE event can be missed.
  // Poll only while work is outstanding so the last "updating" state clears.
  const pending = (data?.pending_documents ?? 0) > 0;
  useEffect(() => {
    if (!pending) return;
    let live = true;
    let timer: ReturnType<typeof setTimeout>;
    const refresh = async () => {
      try {
        const p = await profileApi.get(projectId);
        if (live) setData(p);
      } catch (e) {
        if (live) fail(e);
      }
      if (live) timer = setTimeout(refresh, 2000);
    };
    timer = setTimeout(refresh, 2000);
    return () => { live = false; clearTimeout(timer); };
  }, [projectId, pending, fail]);

  const set = useCallback(
    async (key: string, body: { part_id?: string; value?: string | null; note?: string; reset?: boolean }) => {
      try {
        setData(await profileApi.set(projectId, key, body));
        setError("");
      } catch (e) {
        fail(e);
      }
    },
    [projectId, fail],
  );

  const update = useCallback(async () => {
    setRequesting(true);
    try {
      const p = await profileApi.read(projectId);
      setData(p);
      setError("");
      setRequestNote(p.pending_documents > 0 || p.failed_documents > 0 ? "" : "No new reading queued. Check source coverage and unresolved passages below.");
    } catch (e) {
      fail(e);
    } finally {
      setRequesting(false);
    }
  }, [projectId, fail]);

  if (!data) {
    return <p className="muted">{error || "Reading the profile…"}</p>;
  }
  const builtAt = data.built_at ? new Date(data.built_at) : null;

  return (
    <div className="profile">
      <div className="profile-actions">
        <button type="button" className="btn btn-primary btn-small" onClick={update} disabled={requesting}>
          {requesting ? "Requesting profile update…" : data.failed_documents > 0 ? "Retry project profile" : data.active_documents > 0 ? "Updating project profile…" : data.pending_documents > 0 ? "Profile update queued" : "Update project profile"}
        </button>
        <span className="profile-actions-note">
          {data.failed_documents > 0
            ? `${data.failed_documents} document${data.failed_documents === 1 ? "" : "s"} could not be processed${data.active_documents > 0 ? ` · ${data.active_documents} still processing` : ""}`
            : data.unread_documents > 0
            ? `${data.unread_documents} document${data.unread_documents === 1 ? "" : "s"} not read yet`
            : data.pending_documents > 0
              ? data.active_documents > 0 ? "Documents are being processed; fields appear as reading finishes" : "Waiting for document processing to start or resume"
              : requestNote || ((data.coverage ?? []).length === 0 ? "Upload documents to start" : data.coverage?.some(d => !d.current) ? "Update needed to check complete source coverage" : "Reading finished — review source coverage below")}
        </span>
      </div>
      {data.failed_documents > 0 && (
        <p className="banner" role="alert">
          {data.payment_required
            ? "Profile reading stopped: Jev returned Payment Required (402). Check your TypeSafe account’s billing or credits, then retry. Your documents and prepared text are saved."
            : data.active_documents > 0 ? "Some document processing stopped; other documents are still being processed. Your files are saved. Retry the project profile to try the failed documents again." : "Document processing stopped. Your files are saved. Retry the project profile to try again."}
        </p>
      )}
      <p className="profile-status">
        {builtAt ? `Updated ${builtAt.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}` : "Not built yet"}
        {data.pending_documents > 0 && ` · ${data.pending_documents} document${data.pending_documents === 1 ? "" : "s"} pending`}
        {!data.thresholds.applied
          ? " · Readings are stored but not shown until the thresholds are approved"
          : data.thresholds.provisional && " · Provisional thresholds"}
      </p>
      {error && (
        <p className="banner" role="alert">
          {error}
        </p>
      )}

      <ProfileSources projectId={projectId} data={data} system={sourceSystem} onSystem={setSourceSystem} open={sourcesOpen} onOpen={setSourcesOpen} />

      <PartsBar data={data} projectId={projectId} onChanged={() => profileApi.get(projectId).then(setData, fail)} onError={fail} />

      <section className="pf-section" aria-labelledby="pf-project">
        <h2 id="pf-project">Project</h2>
        <div className="pf-grid">
          {data.header.map((f) => (
            <FieldRow key={f.key} field={f} onSet={set} onJump={onJump} />
          ))}
        </div>
        <details className="pf-more">
          <summary>Approvals and contract</summary>
          <div className="pf-grid">
            {data.facts.map((f) => (
              <FieldRow key={f.key} field={f} onSet={set} onJump={onJump} />
            ))}
          </div>
        </details>
      </section>

      <section className="pf-section" aria-labelledby="pf-systems">
        <div className="pf-head">
          <h2 id="pf-systems">Systems</h2>
          <label className="pf-toggle">
            <input type="checkbox" checked={showAll} onChange={(e) => setShowAll(e.target.checked)} /> Show all systems
          </label>
        </div>
        {data.systems.map((g) => {
          const rows = g.rows.filter((r) => showAll || r.shown_by_default);
          if (rows.length === 0) return null;
          return (
            <div className="pf-sysgroup" key={g.id}>
              <h3>{g.label}</h3>
              {rows.map((r) => (
                <SystemLine key={r.leaf} row={r} onSet={set} onJump={onJump} onSources={() => { setSourceSystem(r.leaf); setSourcesOpen(true); document.getElementById("profile-sources")?.scrollIntoView({ block: "start" }); }} />
              ))}
            </div>
          );
        })}
        {!showAll && data.systems.every((g) => g.rows.every((r) => !r.shown_by_default)) && (
          <p className="muted">
            Nothing read yet. Choose a building type and work type above for typical systems, or show all systems.
          </p>
        )}
      </section>

      <section className="pf-section" aria-labelledby="pf-compliance">
        <h2 id="pf-compliance">Compliance</h2>
        {data.compliance.map((g) =>
          g.rows.length === 0 ? null : (
            <div key={g.group}>
              <h3>{GROUP_LABELS[g.group] ?? g.group}</h3>
              <div className="pf-grid">
                {g.rows.map((f) => (
                  <FieldRow key={f.key + f.part_id} field={f} onSet={set} onJump={onJump} parts={data.parts} />
                ))}
              </div>
            </div>
          ),
        )}
      </section>
    </div>
  );
}

type Setter = (key: string, body: { part_id?: string; value?: string | null; note?: string; reset?: boolean }) => void;

function Mark({ cell }: { cell: Cell }) {
  switch (cell.band) {
    case "user":
      return (
        <span className="pf-mark" data-band="user" title="Set by you">
          <IconYou />
          <span className="sr-only">Set by you</span>
        </span>
      );
    case "green":
      return (
        <span className="pf-mark" data-band="green" title="From the documents">
          <IconConfirmed />
          <span className="sr-only">From the documents</span>
        </span>
      );
    case "amber":
      return (
        <span className="pf-mark" data-band="amber" title={cell.tenders === "consistent" ? "Check: the tenders agree" : "Check: read by Jev"}>
          <IconCheck />
          <span className="sr-only">Check</span>
        </span>
      );
    case "red":
      return (
        <span className="pf-mark" data-band="red" title={cell.tenders === "differ" ? "Tenderers differ" : "Sources differ"}>
          <IconCheck />
          <span className="pf-mark-text">{cell.tenders === "differ" ? "Tenderers differ" : "Sources differ"}</span>
        </span>
      );
    case "suggested":
      return (
        <span className="pf-mark" data-band="suggested" title="Typical for this building type; not read from a document">
          <span className="pf-mark-text">Typical</span>
        </span>
      );
    case "unchecked":
      return (
        <span className="pf-mark" data-band="unchecked" title="Not checked: Jev didn't answer in time">
          <IconNotChecked />
          <span className="sr-only">Not checked</span>
        </span>
      );
    case "blank":
      return cell.sources.length > 0 ? (
        <span className="pf-mark" data-band="blank" title="Read, but not sure enough to fill in">
          <span className="pf-mark-text">Not sure</span>
        </span>
      ) : null;
    default:
      return null;
  }
}

function Sources({ cell, onJump }: { cell: Cell; onJump: (id: string) => void }) {
  if (cell.sources.length === 0) return null;
  return (
    <details className="pf-sources">
      <summary aria-label={`${cell.sources.length} source${cell.sources.length === 1 ? "" : "s"}`}>
        {cell.sources.length}
      </summary>
      <ul>
        {cell.sources.map((s, i) => (
          <li key={i}>
            {s.excerpt ? <q>{s.excerpt}</q> : <span className="muted">No excerpt</span>}{" "}
            <button type="button" className="cell-link" onClick={() => onJump(s.document_id)}>
              Show in register
            </button>
          </li>
        ))}
      </ul>
    </details>
  );
}

function Conflict({ cell, onPick }: { cell: Cell; onPick: (v: string) => void }) {
  if (cell.band !== "red" || cell.alternatives.length === 0) return null;
  return (
    <span className="pf-alts">
      {cell.alternatives.map((a) => (
        <button type="button" className="btn btn-small" key={a.value} onClick={() => onPick(a.value)}>
          Use {a.value} ({a.sources.length})
        </button>
      ))}
    </span>
  );
}

function FieldRow({
  field,
  onSet,
  onJump,
  parts,
}: {
  field: ProfileField;
  onSet: Setter;
  onJump: (id: string) => void;
  parts?: ProfileData["parts"];
}) {
  const [draft, setDraft] = useState(field.value);
  useEffect(() => setDraft(field.value), [field.value]);
  const id = `pf-${field.key}-${field.part_id}`;
  const part = parts && parts.length > 1 ? parts.find((p) => p.id === field.part_id) : undefined;
  const label = part && part.kind !== "whole" ? `${field.label} · ${part.label}` : field.label;
  const write = (value: string) =>
    value === "" ? onSet(field.key, { part_id: field.part_id, reset: true }) : onSet(field.key, { part_id: field.part_id, value });
  const derived = field.derived;

  let control;
  if (derived) {
    control = (
      <span className="pf-derived">
        {field.value ? `${field.value}${field.unit ? ` ${field.unit}` : ""}` : REASONS[derived.reason ?? ""] ?? "Unknown"}
      </span>
    );
  } else if (field.kind === "choice" && field.options && field.options.length > 0) {
    control = (
      <select id={id} className="select pf-control" value={field.value} onChange={(e) => write(e.target.value)}>
        <option value="">—</option>
        {field.options.map((o) => (
          <option key={o.id} value={o.id}>
            {o.label}
          </option>
        ))}
      </select>
    );
  } else {
    const commit = (e?: FormEvent) => {
      e?.preventDefault();
      if (draft.trim() !== field.value) write(draft.trim());
    };
    control = (
      <form onSubmit={commit} className="pf-inline">
        <input
          id={id}
          className="input pf-control"
          inputMode={field.kind === "number" ? "decimal" : undefined}
          value={draft}
          placeholder={field.kind === "multi" ? "e.g. 2, 6" : undefined}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={() => commit()}
        />
        {field.unit && <span className="muted">{field.unit === "m2" ? "m²" : field.unit}</span>}
      </form>
    );
  }

  return (
    <div className="pf-row" data-band={field.band || undefined}>
      <label htmlFor={derived ? undefined : id} className="pf-label">
        {label}
      </label>
      <div className="pf-value">
        {control}
        {field.assertion && ASSERTIONS[field.assertion] && (
          <span className="pf-assert" data-assertion={field.assertion}>
            {ASSERTIONS[field.assertion]}
          </span>
        )}
        <Mark cell={field} />
        <Conflict cell={field} onPick={(v) => onSet(field.key, { part_id: field.part_id, value: v })} />
        <Sources cell={field} onJump={onJump} />
        {field.band === "user" && (
          <button type="button" className="cell-link" onClick={() => onSet(field.key, { part_id: field.part_id, reset: true })}>
            Reset
          </button>
        )}
        {!field.value && !derived && field.band !== "user" && field.stated_in && field.stated_in.length > 0 && (
          <span className="pf-hint">Usually in: {field.stated_in.join(", ")}</span>
        )}
      </div>
    </div>
  );
}

const PRESENCE: { value: string; label: string }[] = [
  { value: "included", label: "Included" },
  { value: "not_included", label: "Not included" },
  { value: "", label: "Unknown" },
];

function SystemLine({ row, onSet, onJump, onSources }: { row: SystemRow; onSet: Setter; onJump: (id: string) => void; onSources: () => void }) {
  const base = `sys.${row.leaf}`;
  const [note, setNote] = useState(row.note.value);
  useEffect(() => setNote(row.note.value), [row.note.value]);
  const current = row.presence.value;
  return (
    <div className="pf-sys" data-band={row.presence.band || undefined}>
      <span className="pf-sys-label" id={`${base}-label`}>
        {row.label}
      </span>
      <div className="pf-seg" role="radiogroup" aria-labelledby={`${base}-label`}>
        {PRESENCE.map((p) => (
          <label key={p.value} data-on={current === p.value ? "true" : undefined}>
            <input
              type="radio"
              name={`${base}-presence`}
              checked={current === p.value}
              onChange={() => onSet(`${base}.presence`, { part_id: row.part_id, value: p.value || null })}
            />
            {p.label}
          </label>
        ))}
      </div>
      <select
        className="select pf-provider"
        aria-label={`Who provides ${row.label}`}
        value={row.provider.value}
        onChange={(e) =>
          e.target.value
            ? onSet(`${base}.provider`, { part_id: row.part_id, value: e.target.value })
            : onSet(`${base}.provider`, { part_id: row.part_id, reset: true })
        }
      >
        <option value="">Provider —</option>
        <option value="contractor">Contractor</option>
        <option value="owner">Owner</option>
        <option value="others">Others</option>
      </select>
      <span className="pf-sys-marks">
        <Mark cell={row.presence} />
        <Mark cell={row.provider.band === "red" ? row.provider : { ...row.provider, band: "" }} />
        <Conflict cell={row.presence} onPick={(v) => onSet(`${base}.presence`, { part_id: row.part_id, value: v })} />
        <Conflict cell={row.provider} onPick={(v) => onSet(`${base}.provider`, { part_id: row.part_id, value: v })} />
        <Sources cell={row.presence} onJump={onJump} />
          <button type="button" className="cell-link" onClick={onSources} aria-label={`View requirements for ${row.label}`}>Requirements</button>
        {row.presence.band === "user" && (
          <button type="button" className="cell-link" onClick={() => onSet(`${base}.presence`, { part_id: row.part_id, reset: true })}>
            Reset
          </button>
        )}
      </span>
      <input
        className="input pf-note"
        aria-label={`Note for ${row.label}`}
        maxLength={120}
        value={note}
        placeholder={row.presence.note ?? ""}
        onChange={(e) => setNote(e.target.value)}
        onBlur={() => note !== row.note.value && onSet(`${base}.note`, { part_id: row.part_id, value: note })}
      />
    </div>
  );
}

const KINDS = ["building", "part", "storey", "compartment", "tenancy", "outbuilding"];

function PartsBar({
  data,
  projectId,
  onChanged,
  onError,
}: {
  data: ProfileData;
  projectId: string;
  onChanged: () => void;
  onError: (e: unknown) => void;
}) {
  const [adding, setAdding] = useState(false);
  const [label, setLabel] = useState("");
  const [kind, setKind] = useState("building");
  async function add(e: FormEvent) {
    e.preventDefault();
    if (!label.trim()) return;
    try {
      await profileApi.addPart(projectId, label.trim(), kind);
      setLabel("");
      setAdding(false);
      onChanged();
    } catch (err) {
      onError(err);
    }
  }
  return (
    <div className="pf-parts" aria-label="Parts">
      {data.parts.map((p) => (
        <span className="pf-part" key={p.id} data-kind={p.kind}>
          {p.label}
        </span>
      ))}
      {adding ? (
        <form onSubmit={add} className="pf-inline">
          <input className="input" aria-label="Part label" value={label} onChange={(e) => setLabel(e.target.value)} autoFocus />
          <select className="select" aria-label="Part kind" value={kind} onChange={(e) => setKind(e.target.value)}>
            {KINDS.map((k) => (
              <option key={k}>{k}</option>
            ))}
          </select>
          <button type="submit" className="btn btn-small">
            Add
          </button>
          <button type="button" className="cell-link" onClick={() => setAdding(false)}>
            Cancel
          </button>
        </form>
      ) : (
        <button type="button" className="cell-link" onClick={() => setAdding(true)}>
          + Part
        </button>
      )}
    </div>
  );
}
