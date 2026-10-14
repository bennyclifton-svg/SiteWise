// The project profile: what the filed documents say the project is, drafted
// in the background by Jev and code, polished here by the user. Every value
// says how it was decided; the user's word is final.
//
// The page reads top down: who and where the project is and how far reading
// got; the Project card (the building, the site and its approvals, how it is
// delivered); then scope and works, system by system in build order, each
// with the risks and compliance inputs that belong to it.

import { useCallback, useEffect, useState, type FormEvent } from "react";
import { ProfileSources } from "./ProfileSources";
import { FieldRow, type Setter } from "./ProfileFields";
import { PLANNING, ScopeWorks, entryInputs } from "./ScopeWorks";
import { ApiError } from "./api";
import { profileApi, type Profile as ProfileData, type ProfileField, type ScopeChoice, type Site } from "./profileApi";

interface Props {
  projectId: string;
  view?: "all" | "summary" | "systems";
  /** Changes when a profile event arrives; the panel refetches. */
  tick: number;
  /** Changes when work, packages or reports change elsewhere; saved work reloads. */
  workTick?: number;
  onJump: (documentId: string) => void;
  onSignedOut: () => void;
  /** Filters the register to documents the profile does not read. */
  onShowNotRead: () => void;
}

const KIND_LABELS: Record<string, string> = {
  drawing: "drawings",
  photo: "photos",
  correspondence: "correspondence",
  unknown: "unclassified documents",
};

type Column = "building" | "site" | "delivery";

const COLUMN_LABELS: Record<Column, string> = {
  building: "The building",
  site: "Site and planning",
  delivery: "Delivery",
};

// The Project card's columns, in reading order. Pairs that look alike but
// follow different definitions (building class and NCC class, storeys and
// NCC rise in storeys, GFA and NCC floor area) sit together.
const COLUMNS: Record<Column, string[]> = {
  building: [
    "hdr.building_class", "hdr.subclass", "det.ncc_class", "hdr.work_type",
    "hdr.scale.storeys", "hdr.scale.total_storeys", "hdr.scale.levels", "det.rise_in_storeys",
    "hdr.scale.gfa_sqm", "hdr.scale.total_gfa_sqm", "det.floor_area", "det.effective_height",
    "hdr.scale.units", "hdr.scale.dwellings", "hdr.scale.residential_units",
    "hdr.scale.bedrooms", "hdr.scale.bathrooms", "hdr.cond.quality_level",
    "hdr.scale.car_parks", "hdr.scale.garage_spaces",
    "det.existing_building", "det.existing_building_year", "det.ncc_edition",
  ],
  site: [
    "site.address", "site.lot", "det.state", "proj.lga", "proj.zone", "hdr.scale.site_sqm",
    "det.heritage_status", "hdr.cond.planning", "fact.consent_number", "fact.consent_date",
    "fact.consent_lapse_date", "hdr.cond.access_constraints", "hdr.cond.environmental_sensitivity",
  ],
  delivery: [
    "proj.client", "proj.budget_stated", "hdr.cond.procurement_route", "fact.contract_form", "fact.contract_basis",
    "fact.defects_liability_period", "fact.design_life", "proj.program", "proj.sustainability",
    "proj.flag.sustainability_target", "proj.regulated_design", "proj.flag.contract_admin",
    "hdr.cond.stakeholder_complexity", "hdr.cond.operational_constraints",
  ],
};

// Shown even when empty: what every project needs stated.
const ESSENTIAL = new Set([
  "hdr.building_class", "hdr.subclass", "det.ncc_class", "hdr.work_type", "hdr.scale.storeys",
  "hdr.scale.gfa_sqm", "site.address", "site.lot", "det.state", "proj.lga", "proj.zone",
  "hdr.cond.planning", "proj.client", "hdr.cond.procurement_route",
]);

// Scale fields that count the same thing. The first with a value is shown;
// another is shown only when it says something different.
const SAME_MEASURE = [
  ["hdr.scale.storeys", "hdr.scale.total_storeys", "hdr.scale.levels"],
  ["hdr.scale.gfa_sqm", "hdr.scale.total_gfa_sqm"],
  ["hdr.scale.units", "hdr.scale.dwellings", "hdr.scale.residential_units"],
  ["hdr.scale.car_parks", "hdr.scale.garage_spaces"],
];

function columnOf(key: string): Column {
  for (const c of Object.keys(COLUMNS) as Column[]) if (COLUMNS[c].includes(key)) return c;
  if (key.startsWith("site.") || key.startsWith("hdr.scale.site")) return "site";
  if (key.startsWith("hdr.scale.") || key.startsWith("det.")) return "building";
  return "delivery";
}

const rank = (key: string) => {
  const c = COLUMNS[columnOf(key)].indexOf(key);
  return c < 0 ? Number.MAX_SAFE_INTEGER : c;
};

/** Splits card fields into columns: what shows, and what waits behind "more". */
function arrangeCard(fields: ProfileField[]) {
  const dropped = new Set<ProfileField>();
  for (const group of SAME_MEASURE) {
    const members = group.map((k) => fields.find((f) => f.key === k)).filter((f): f is ProfileField => !!f);
    const first = members.find((f) => f.value !== "") ?? members[0];
    for (const f of members) if (f !== first && (f.value === "" || f.value === first.value) && f.band !== "red") dropped.add(f);
  }
  const out: Record<Column, { shown: ProfileField[]; more: ProfileField[] }> = {
    building: { shown: [], more: [] },
    site: { shown: [], more: [] },
    delivery: { shown: [], more: [] },
  };
  const sorted = fields.filter((f) => !dropped.has(f)).sort((a, b) => rank(a.key) - rank(b.key));
  for (const f of sorted) {
    const needed = ESSENTIAL.has(f.key) || f.value !== "" || f.band === "red";
    out[columnOf(f.key)][needed ? "shown" : "more"].push(f);
  }
  return out;
}

export function Profile({ projectId, tick, workTick = 0, onJump, onSignedOut, onShowNotRead, view = "all" }: Props) {
  const [sourceSystem, setSourceSystem] = useState("");
  const [sourcesOpen, setSourcesOpen] = useState(false);
  const [data, setData] = useState<ProfileData | null>(null);
  const [error, setError] = useState("");
  const [requesting, setRequesting] = useState(false);
  const [requestNote, setRequestNote] = useState("");
  const [entry, setEntry] = useState("");

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

  const setScope = useCallback(
    async (systems: Record<string, ScopeChoice>) => {
      try {
        setData(await profileApi.setScope(projectId, systems));
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
      setRequestNote(p.pending_documents > 0 || p.failed_documents > 0 ? "" : "No new reading queued. Check source coverage and unresolved passages.");
    } catch (e) {
      fail(e);
    } finally {
      setRequesting(false);
    }
  }, [projectId, fail]);

  const site = useSiteFields(projectId, tick, data?.site_details ?? [], fail);

  if (!data) {
    return <p className="muted">{error || "Reading the profile…"}</p>;
  }

  const openSources = (system: string) => {
    setSourceSystem(system);
    setSourcesOpen(true);
    requestAnimationFrame(() => document.getElementById("profile-sources")?.scrollIntoView({ block: "start" }));
  };

  // The card holds the header, project facts and details, the site record,
  // and every input placed in the Project (layout.yaml home "project").
  const projectInputs = data.compliance.flatMap((g) => g.rows).filter((f) => f.home === "project");
  const cardFields = [
    ...data.header,
    ...data.facts,
    ...data.details.filter((f) => !f.key.startsWith("proj.flag.") || f.home === "project"),
    ...site.fields,
    ...projectInputs,
  ];
  const card = arrangeCard(cardFields);
  const setField: Setter = (key, body) => (key.startsWith("site.") ? site.set(key, body) : set(key, body));

  // What needs the user's eye: conflicting values, and systems whose
  // documents disagree or say a ticked system is not included.
  const cardChecks = cardFields.filter((f) => f.band === "red").length;
  const entries = [PLANNING, ...data.systems.map((g) => g.id)];
  const entryChecks = (id: string) => {
    const rows = data.systems.find((g) => g.id === id)?.rows ?? [];
    const { flags, inputs } = entryInputs(data, id);
    return rows.filter((r) => r.presence.band === "red" || (r.in_scope && r.presence.value === "not_included" && !["", "blank", "suggested"].includes(r.presence.band))).length
      + [...flags, ...inputs].filter((f) => f.band === "red").length;
  };
  const firstCheck = entries.find((id) => entryChecks(id) > 0);
  const checks = cardChecks + entries.reduce((n, id) => n + entryChecks(id), 0);
  const selected = entry || firstCheck || "site";
  const goToCheck = () => {
    const red = document.querySelector<HTMLElement>(".pf-card .pf-row[data-band=red]");
    if (red && view !== "systems") {
      red.scrollIntoView({ block: "center" });
      red.querySelector<HTMLElement>("select, input, button")?.focus();
    } else if (firstCheck) {
      setEntry(firstCheck);
      requestAnimationFrame(() => document.getElementById("pf-scope")?.scrollIntoView({ block: "start" }));
    }
  };

  const builtAt = data.built_at ? new Date(data.built_at) : null;
  const coverage = data.coverage ?? [];
  const unresolved = coverage.reduce((n, d) => n + d.needs_mapping, 0);
  const identity = [site.fields.find((f) => f.key === "site.address")?.value, data.details.find((f) => f.key === "proj.client")?.value, data.details.find((f) => f.key === "proj.lga")?.value].filter(Boolean);
  const note = data.failed_documents > 0
    ? `${data.failed_documents} document${data.failed_documents === 1 ? "" : "s"} could not be processed${data.active_documents > 0 ? ` · ${data.active_documents} still processing` : ""}`
    : data.unread_documents > 0
      ? `${data.unread_documents} document${data.unread_documents === 1 ? "" : "s"} not read yet`
      : data.pending_documents > 0
        ? data.active_documents > 0 ? "Documents are being processed; fields appear as reading finishes" : "Waiting for document processing to start or resume"
        : requestNote || (coverage.length === 0 ? "Upload documents to start" : coverage.some((d) => !d.current) ? "Update needed to check complete source coverage" : "");

  return (
    <div className="profile">
      <div className="pf-header">
        <p className="pf-identity">{identity.length > 0 ? identity.join(" · ") : <span className="muted">Address, client and council not stated yet</span>}</p>
        <button type="button" className="btn btn-primary btn-small" onClick={update} disabled={requesting}>
          {requesting ? "Requesting profile update…" : data.failed_documents > 0 ? "Retry project profile" : data.active_documents > 0 ? "Updating project profile…" : data.pending_documents > 0 ? "Profile update queued" : "Update project profile"}
        </button>
        <p className="profile-reading">
          {data.read_documents + data.skipped_documents > 0 && (
            <span>
              Reading {data.read_documents} of {data.read_documents + data.skipped_documents} document
              {data.read_documents + data.skipped_documents === 1 ? "" : "s"}
              {data.skipped_documents > 0 && (
                <>
                  {" "}({data.skipped_documents} not read
                  {data.skipped_kind ? `, mostly ${KIND_LABELS[data.skipped_kind] ?? data.skipped_kind.replace(/_/g, " ")}` : ""}{" "}
                  <button type="button" className="cell-link" onClick={onShowNotRead}>
                    Show
                  </button>
                  )
                </>
              )}
            </span>
          )}
          <span>
            {unresolved > 0 && `${unresolved} passages need review `}
            <button type="button" className="cell-link" aria-expanded={sourcesOpen} onClick={() => setSourcesOpen(!sourcesOpen)}>
              Source requirements
            </button>
          </span>
          <span>
            {builtAt ? `Updated ${builtAt.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}` : "Not built yet"}
            {data.pending_documents > 0 && ` · ${data.pending_documents} document${data.pending_documents === 1 ? "" : "s"} pending`}
          </span>
          {!data.thresholds.applied
            ? <span>Readings are stored but not shown until the thresholds are approved</span>
            : data.thresholds.provisional && <span>Provisional thresholds</span>}
          {note && <span className="profile-actions-note">{note}</span>}
          {checks > 0 && (
            <button type="button" className="pf-checks" onClick={goToCheck}>
              ⚠ {checks} to check
            </button>
          )}
        </p>
      </div>
      {data.failed_documents > 0 && (
        <p className="banner" role="alert">
          {data.payment_required
            ? "Profile reading stopped: Jev returned Payment Required (402). Check your TypeSafe account’s billing or credits, then retry. Your documents and prepared text are saved."
            : data.active_documents > 0 ? "Some document processing stopped; other documents are still being processed. Your files are saved. Retry the project profile to try the failed documents again." : "Document processing stopped. Your files are saved. Retry the project profile to try again."}
        </p>
      )}
      {error && (
        <p className="banner" role="alert">
          {error}
        </p>
      )}

      {sourcesOpen && <ProfileSources projectId={projectId} data={data} system={sourceSystem} onSystem={setSourceSystem} onClose={() => setSourcesOpen(false)} />}

      <section className="pf-section" aria-labelledby="pf-project" hidden={view === "systems"}>
        <div className="pf-head">
          <h2 id="pf-project">Project</h2>
          <PartsBar data={data} projectId={projectId} onChanged={() => profileApi.get(projectId).then(setData, fail)} onError={fail} />
        </div>
        <div className="pf-card">
          {(Object.keys(COLUMN_LABELS) as Column[]).map((c) => (
            <CardColumn key={c} label={COLUMN_LABELS[c]} shown={card[c].shown} more={card[c].more} parts={data.parts} onSet={setField} onJump={onJump} />
          ))}
        </div>
      </section>

      <div hidden={view === "summary"}>
        <ScopeWorks
          projectId={projectId}
          data={data}
          tick={tick + workTick}
          selected={selected}
          onSelect={setEntry}
          onScope={setScope}
          onSet={set}
          onJump={onJump}
          onSources={openSources}
          onSignedOut={onSignedOut}
        />
      </div>
    </div>
  );
}

function CardColumn({ label, shown, more, parts, onSet, onJump }: { label: string; shown: ProfileField[]; more: ProfileField[]; parts: ProfileData["parts"]; onSet: Setter; onJump: (id: string) => void }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="pf-col" role="group" aria-label={label}>
      <h3>{label}</h3>
      {shown.map((f) => <FieldRow key={f.key + f.part_id} field={f} onSet={onSet} onJump={onJump} parts={parts} compact />)}
      {open && more.map((f) => <FieldRow key={f.key + f.part_id} field={f} onSet={onSet} onJump={onJump} parts={parts} compact />)}
      {more.length > 0 && (
        <button type="button" className="cell-link pf-col-more" aria-expanded={open} onClick={() => setOpen(!open)}>
          {open ? "Fewer" : `+ ${more.length} more`}
        </button>
      )}
    </div>
  );
}

// Address and lot live on the site record, which outlasts the project. What
// the user typed there wins; until then the documents' reading is shown, with
// its sources, and typing over it or resetting only touches the site record.
function useSiteFields(projectId: string, tick: number, readings: ProfileField[], onError: (e: unknown) => void) {
  const [site, setSite] = useState<Site | null>(null);
  useEffect(() => {
    let live = true;
    profileApi.site(projectId).then((s) => live && setSite(s), (e) => live && onError(e));
    return () => {
      live = false;
    };
  }, [projectId, tick, onError]);
  if (!site) return { fields: [] as ProfileField[], set: (() => undefined) as Setter };
  const save = async (field: "address" | "lot", value: string) => {
    try {
      setSite(await profileApi.setSite(site.id, site.version, { [field]: value }));
    } catch (e) {
      // A concurrent edit: show the saved site rather than overwrite it.
      if (e instanceof ApiError && e.status === 409) setSite(await profileApi.site(projectId));
      else onError(e);
    }
  };
  const set: Setter = (key, body) => void save(key === "site.address" ? "address" : "lot", body.reset || body.value == null ? "" : body.value);
  const fields = readings.map((reading) => {
    const typed = reading.key === "site.address" ? site.address : site.lot;
    return typed ? { ...reading, value: typed, band: "user" as const, sources: [], alternatives: [], assertion: "" } : reading;
  });
  return { fields, set };
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
      <span className="muted">Parts</span>
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
