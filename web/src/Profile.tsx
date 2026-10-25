import { Pending } from "./Activity";
// The project profile: what the filed documents say the project is, drafted
// in the background by Jev and code, polished here by the user. Every value
// says how it was decided; the user's word is final.
//
// Reading feedback lives in the bottom ribbon. The page starts with the Project card (who and
// where, class, work, then subclass, scale and complexity, approvals and
// delivery); then scope and works, system by system in build order, each
// with the risks and compliance inputs that belong to it.

import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { ProfileSources } from "./ProfileSources";
import { ChoiceGroup, FieldRow, type Setter } from "./ProfileFields";
import { PLANNING, ScopeWorks, entryInputs } from "./ScopeWorks";
import { ApiError } from "./api";
import { profileApi, type Profile as ProfileData, type ProfileField, type ScopeChoice, type Site } from "./profileApi";

interface Props {
  projectId: string;
  /** Changes when a profile event arrives; the panel refetches. */
  tick: number;
  /** Changes when work, packages or reports change elsewhere; saved work reloads. */
  workTick?: number;
  onJump: (documentId: string) => void;
  onSignedOut: () => void;
  /** Filters the register to documents the profile does not read. */
  onShowNotRead: () => void;
  /** Address line for the project banner. Empty when the site address is blank. */
  onIdentity?: (line: string) => void;
  feedbackTarget?: HTMLDivElement | null;
}

const KIND_LABELS: Record<string, string> = {
  drawing: "drawings",
  photo: "photos",
  correspondence: "correspondence",
  unknown: "unclassified documents",
};

// The Project card reads top down, as a brief does: who and where, the class
// of building, the work, then that kind of building's subclass, size and
// conditions. Approvals and delivery follow. A key not placed here falls to
// delivery so a new field is never hidden.
type Tier = "identity" | "place" | "class" | "work" | "budget" | "subclass" | "scale" | "complexity" | "approvals" | "delivery";

const TIERS: Record<Tier, string[]> = {
  identity: ["site.address", "proj.client", "det.state"],
  place: ["site.lot", "proj.lga", "proj.zone"],
  class: ["hdr.building_class"],
  work: ["hdr.work_type"],
  budget: ["proj.budget_stated"],
  subclass: ["hdr.subclass", "det.ncc_class"],
  // Pairs that look alike but follow different definitions sit together.
  scale: [
    "hdr.scale.site_sqm", "hdr.scale.storeys", "hdr.scale.total_storeys", "hdr.scale.levels", "det.rise_in_storeys",
    "hdr.scale.gfa_sqm", "hdr.scale.total_gfa_sqm", "det.floor_area", "det.effective_height",
    "hdr.scale.units", "hdr.scale.dwellings", "hdr.scale.residential_units",
    "hdr.scale.bedrooms", "hdr.scale.bathrooms", "hdr.scale.car_parks", "hdr.scale.garage_spaces",
    "det.existing_building", "det.existing_building_year",
  ],
  complexity: [
    "hdr.cond.planning", "hdr.cond.procurement_route", "hdr.cond.quality_level",
    "hdr.cond.access_constraints", "hdr.cond.operational_constraints",
    "hdr.cond.stakeholder_complexity", "hdr.cond.environmental_sensitivity",
    "det.flood_hazard", "det.bushfire_prone_land", "det.heritage_status",
  ],
  approvals: ["fact.consent_number", "fact.consent_date", "fact.consent_lapse_date", "det.ncc_edition"],
  delivery: [
    "fact.contract_form", "fact.contract_basis", "fact.defects_liability_period", "fact.design_life",
    "proj.program", "proj.sustainability", "proj.flag.sustainability_target", "proj.regulated_design", "proj.flag.contract_admin",
  ],
};

function tierOf(key: string): Tier {
  for (const t of Object.keys(TIERS) as Tier[]) if (TIERS[t].includes(key)) return t;
  if (key.startsWith("hdr.scale.")) return "scale";
  if (key.startsWith("hdr.cond.")) return "complexity";
  return "delivery";
}

const rank = (key: string) => {
  const i = TIERS[tierOf(key)].indexOf(key);
  return i < 0 ? Number.MAX_SAFE_INTEGER : i;
};

// Shown even when empty. Scale shows every measure the building type asks
// for and complexity every condition, as a brief form would.
const ESSENTIAL = new Set(["det.ncc_class"]);
const ALWAYS: Tier[] = ["identity", "place", "class", "work", "budget", "subclass", "complexity"];

// Scale fields that count the same thing. The first with a value is shown;
// another is shown only when it says something different.
const SAME_MEASURE = [
  ["hdr.scale.storeys", "hdr.scale.total_storeys", "hdr.scale.levels"],
  ["hdr.scale.gfa_sqm", "hdr.scale.total_gfa_sqm"],
  ["hdr.scale.units", "hdr.scale.dwellings", "hdr.scale.residential_units"],
  ["hdr.scale.car_parks", "hdr.scale.garage_spaces"],
];

type Arranged = Record<Tier, { shown: ProfileField[]; more: ProfileField[] }>;

/** Splits card fields into tiers: what shows, and what waits behind "more". */
function arrangeCard(fields: ProfileField[]): Arranged {
  const dropped = new Set<ProfileField>();
  for (const group of SAME_MEASURE) {
    const members = group.map((k) => fields.find((f) => f.key === k)).filter((f): f is ProfileField => !!f);
    const first = members.find((f) => f.value !== "") ?? members[0];
    for (const f of members) if (f !== first && (f.value === "" || f.value === first.value) && f.band !== "red") dropped.add(f);
  }
  const out = {} as Arranged;
  for (const t of Object.keys(TIERS) as Tier[]) out[t] = { shown: [], more: [] };
  const sorted = fields.filter((f) => !dropped.has(f)).sort((a, b) => rank(a.key) - rank(b.key));
  for (const f of sorted) {
    const t = tierOf(f.key);
    const needed = ALWAYS.includes(t) || f.key.startsWith("hdr.scale.") || ESSENTIAL.has(f.key) || f.value !== "" || f.band === "red";
    out[t][needed ? "shown" : "more"].push(f);
  }
  return out;
}

export function Profile({ projectId, tick, workTick = 0, onJump, onSignedOut, onShowNotRead, onIdentity, feedbackTarget }: Props) {
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
  const addressLine = site.fields.find((f) => f.key === "site.address")?.value ?? "";
  useEffect(() => {
    onIdentity?.(addressLine);
  }, [addressLine, onIdentity]);

  if (!data) {
    return <ProfileFeedback target={feedbackTarget}><p className="muted" role={error ? "alert" : "status"}>{error || <Pending delayed>Loading the profile…</Pending>}</p></ProfileFeedback>;
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
    const red = document.querySelector<HTMLElement>(".pf-card .pf-row[data-band=red], .pf-card .pf-choice[data-band=red]");
    if (red) {
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
  const completed = data.completed_documents ?? 0;
  const total = data.read_documents;
  const note = data.failed_documents > 0
    ? `${data.failed_documents} document${data.failed_documents === 1 ? "" : "s"} could not be processed${data.active_documents > 0 ? ` · ${data.active_documents} still processing` : ""}`
    : data.unread_documents > 0
      ? `${data.unread_documents} document${data.unread_documents === 1 ? "" : "s"} ready to read. Select Update project profile to start.`
      : data.pending_documents > 0
        ? data.active_documents > 0 ? "Documents are being processed; fields appear as reading finishes" : "Waiting for document processing to start or resume"
        : requestNote || (total + data.skipped_documents === 0 ? "Upload documents to start" : coverage.some((d) => !d.current) ? "Update needed to check complete source coverage" : "");

  return (
    <div className="profile">
      <ProfileFeedback target={feedbackTarget}>
      <div className="pf-header">
        <p className="profile-reading">
          {data.read_documents + data.skipped_documents > 0 && (
            <span>
              Included for profile: {data.read_documents} of {data.read_documents + data.skipped_documents} document
              {data.read_documents + data.skipped_documents === 1 ? "" : "s"}
              {data.skipped_documents > 0 && (
                <>
                  {" "}({data.skipped_documents} excluded
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
        {total > 0 && (
          <div className="profile-progress">
            <svg className="profile-reading-meter" viewBox="0 0 24 24" role="progressbar" aria-label="Documents read for profile" aria-valuemin={0} aria-valuemax={total} aria-valuenow={completed}>
              <circle className="profile-reading-track" cx="12" cy="12" r="9" />
              {completed === total
                ? <path className="profile-reading-check" d="m7.5 12 3 3 6-6" />
                : <circle className="profile-reading-fill" cx="12" cy="12" r="9" pathLength="100" strokeDasharray={`${Math.min(100, Math.max(0, completed / total * 100))} 100`} />}
            </svg>
            <p role="status" aria-atomic="true">{completed === total ? "Reading complete" : data.failed_documents > 0 ? "Reading needs attention" : data.pending_documents > 0 ? data.active_documents > 0 ? "Reading documents" : "Reading queued" : "Ready to read"} · {data.active_documents > 0 && `${data.active_documents} processing · `}{Math.max(0, data.pending_documents - data.active_documents) > 0 && `${Math.max(0, data.pending_documents - data.active_documents)} queued · `}{completed} of {total} document{total === 1 ? "" : "s"} read</p>
          </div>
        )}
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
      </ProfileFeedback>

      {sourcesOpen && <ProfileSources projectId={projectId} data={data} system={sourceSystem} onSystem={setSourceSystem} onClose={() => setSourcesOpen(false)} />}

      <div className="profile-toolbar">
        <button type="button" className="btn btn-primary" onClick={update} disabled={requesting}>
          {requesting ? <Pending delayed>Requesting profile update…</Pending> : data.failed_documents > 0 ? "Retry project profile" : data.unread_documents > 0 ? "Update project profile" : data.active_documents > 0 ? <Pending delayed>Updating project profile…</Pending> : data.pending_documents > 0 ? <Pending queued>Profile update queued</Pending> : "Update project profile"}
        </button>
      </div>
      <section className="pf-section" aria-labelledby="pf-project">
        <div className="pf-head">
          <h2 id="pf-project">Project</h2>
          <div className="pf-head-actions">
            <PartsBar data={data} projectId={projectId} onChanged={() => profileApi.get(projectId).then(setData, fail)} onError={fail} />
          </div>
        </div>
        <ProjectCard card={card} parts={data.parts} onSet={setField} onJump={onJump} />
      </section>

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
  );
}

function ProfileFeedback({ target, children }: { target?: HTMLDivElement | null; children: ReactNode }) {
  const scroll = useRef<HTMLDivElement>(null);
  const following = useRef(true);
  useEffect(() => {
    if (following.current && scroll.current) scroll.current.scrollTop = scroll.current.scrollHeight;
  }, [children, target]);
  if (!target) return null;
  return createPortal(
    <div className="ask-bar profile-feedback" role="region" aria-label="Profile updates" tabIndex={0}
      ref={scroll} onScroll={event => {
        const element = event.currentTarget;
        following.current = element.scrollHeight - element.clientHeight - element.scrollTop < 8;
      }}>
      {children}
    </div>, target,
  );
}

function ProjectCard({ card, parts, onSet, onJump }: { card: Arranged; parts: ProfileData["parts"]; onSet: Setter; onJump: (id: string) => void }) {
  const row = (f: ProfileField) => <FieldRow key={f.key + f.part_id} field={f} onSet={onSet} onJump={onJump} parts={parts} compact stacked />;
  const one = (t: Tier, key: string) => card[t].shown.find((f) => f.key === key);
  const chips = (f: ProfileField | undefined, title: string) =>
    f && (f.options?.length ? <ChoiceGroup field={f} title={title} variant="chips" onSet={onSet} onJump={onJump} /> : row(f));
  const subclass = one("subclass", "hdr.subclass");
  // Before a category is chosen the subclass options span every category;
  // a dropdown keeps that long list out of the way until then.
  const listSubclass = !!one("class", "hdr.building_class")?.value && (subclass?.options?.length ?? 0) > 0 && (subclass?.options?.length ?? 0) <= 14;
  return (
    <div className="pf-card pf-tiers">
      <div className="pf-identity-grid" role="group" aria-label="Who and where">
        {card.identity.shown.map(row)}
      </div>
      <div className="pf-place-grid" role="group" aria-label="Lot and council">
        {card.place.shown.map(row)}
      </div>
      {chips(one("class", "hdr.building_class"), "Class")}
      <div className="pf-work-line">
        {chips(one("work", "hdr.work_type"), "Work type")}
        <div className="pf-budget">{card.budget.shown.map(row)}</div>
      </div>
      <div className="pf-cards">
        <Panel title="Subclass">
          {subclass && (listSubclass ? <ChoiceGroup field={subclass} title={subclass.label} variant="list" onSet={onSet} onJump={onJump} /> : row(subclass))}
          {card.subclass.shown.filter((f) => f !== subclass).map(row)}
        </Panel>
        <Panel title="Scale" more={card.scale.more} render={row}>
          {card.scale.shown.length > 0 ? card.scale.shown.map(row) : <p className="muted">Choose a subclass to see its measures.</p>}
        </Panel>
        <Panel title="Complexity" className="pf-panel-wide">
          {card.complexity.shown.map(row)}
        </Panel>
      </div>
      <div className="pf-strips">
        <Panel title="Approvals" more={card.approvals.more} render={row} flat>
          {card.approvals.shown.map(row)}
        </Panel>
        <Panel title="Delivery" more={card.delivery.more} render={row} flat>
          {card.delivery.shown.map(row)}
        </Panel>
      </div>
    </div>
  );
}

function Panel({ title, className = "", more = [], render, flat, children }: { title: string; className?: string; more?: ProfileField[]; render?: (f: ProfileField) => ReactNode; flat?: boolean; children: ReactNode }) {
  const [open, setOpen] = useState(false);
  return (
    <div className={`pf-panel${flat ? " pf-panel-flat" : ""}${className ? ` ${className}` : ""}`} role="group" aria-label={title}>
      <h3 className="pf-rule">{title}</h3>
      <div className="pf-panel-body">
        {children}
        {open && render && more.map(render)}
      </div>
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
