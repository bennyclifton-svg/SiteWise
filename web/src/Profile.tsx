import { Pending } from "./Activity";
import { IconDownload, IconPlus } from "./icons";
import { MoreMenu, PageHeaderAction, UpdateButton } from "./PageActions";
// The project profile: what the filed documents say the project is, drafted
// in the background by Jev and code, polished here by the user. Every value
// says how it was decided; the user's word is final.
//
// Reading feedback lives in the strip at the foot of the view
// (ProfileStrip). The page starts with the Project card (who and
// where, class, work, then subclass, scale and complexity, approvals and
// delivery); then scope and works, system by system in build order, each
// with the risks and compliance inputs that belong to it.
//
// An update shows what the reading does, when it does it
// (docs/design/2026-11-10-run-time-feedback.md): a passage that speaks to a
// field glints it, a committed pass inks the values it changed, and each
// change keeps a revision mark until the user clears it.

import { useCallback, useEffect, useLayoutEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { ProfileSources } from "./ProfileSources";
import { Accommodation } from "./Accommodation";
import { FieldRow, NotYetKnown, unanswered, type Setter } from "./ProfileFields";
import { CountsRow, COUNT_KEYS } from "./ProfileCounts";
import { ProfileStrip, type ChipKind, type ElsewhereView, type LatestFind } from "./ProfileStrip";
import { PLANNING, ScopeWorks, entryInputs, type ScopeMarks } from "./ScopeWorks";
import { ApiError } from "./api";
import { profileApi, type Profile as ProfileData, type ProfileField, type ScopeChoice, type Site } from "./profileApi";
import { diffSnapshots, profileFields, receiptOf, snapshotOf, type Change, type Snapshot } from "./profileDiff";
import { inkInOrder, inView, motionAllowed, play, viewBox, type InkItem, type Motion } from "./ink";
import { findTargets, runApi, useProfileRun, type Find, type Next, type RunDoc } from "./profileRun";

/** What the nav needs to know about the profile's update. */
export interface ProfileRunState {
  reading: boolean;
  /** Changes marked since the user last cleared them. */
  changes: number;
  /** Counts elsewhere, fetched when an update finishes here. */
  next: Next | null;
  /** Documents read and to read in the update, for the tab title. */
  read: number;
  total: number;
}

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
  /** The register's name for a document (its number), falling back to the filename. */
  documentLabel?: (id: string) => string;
  onRunState?: (state: ProfileRunState) => void;
  onOpenView?: (view: ElsewhereView) => void;
}

const KIND_LABELS: Record<string, string> = {
  drawing: "drawings",
  photo: "photos",
  correspondence: "correspondence",
  unknown: "unclassified documents",
};

/** What Update will read, said before it is pressed. */
function updateTitle(p: ProfileData): string | undefined {
  if (p.unread_documents === 0) return undefined;
  const kind = p.skipped_kind ? `, mostly ${KIND_LABELS[p.skipped_kind] ?? p.skipped_kind.replace(/_/g, " ")}` : "";
  const others = p.skipped_documents > 0
    ? ` ${p.skipped_documents} other${p.skipped_documents === 1 ? " is" : "s are"} filed but not read for the profile${kind}.`
    : "";
  return `Reads ${p.unread_documents} new document${p.unread_documents === 1 ? "" : "s"} for the profile.${others}`;
}

// What the user has seen, per project, in this browser (owner, 10 Nov 2026):
// the marks are what changed since. Storage can be refused (a private
// window); the marks then last for the visit.
const seenKey = (projectId: string) => `sitewise.profile.seen.v1.${projectId}`;
function readSeen(projectId: string): Snapshot | null {
  try {
    const raw = localStorage.getItem(seenKey(projectId));
    return raw ? (JSON.parse(raw) as Snapshot) : null;
  } catch {
    return null;
  }
}
function writeSeen(projectId: string, snap: Snapshot) {
  try {
    localStorage.setItem(seenKey(projectId), JSON.stringify(snap));
  } catch {
    // Refused storage: the marks stay for this visit only.
  }
}

/** A new source on an unchanged value is shown while it happens, not kept as a mark. */
const markable = (c: Change) => c.kind === "scope" || c.how !== "sources";

/** The revision mark's words: what it was, what it is, or what needs a decision. */
function markText(c: Change, value: (key: string, v: string) => string): string {
  if (c.kind === "scope") {
    return c.how === "in" ? "Added to scope by an update" : c.how === "out" ? "Taken out of scope by an update" : "A document says this ticked system is not included";
  }
  const passages = `${c.added} passage${c.added === 1 ? "" : "s"}`;
  switch (c.how) {
    case "filled": return `Read in an update: ${value(c.key, c.now)}`;
    case "changed": return `Changed in an update: was ${value(c.key, c.was)}, now ${value(c.key, c.now)}`;
    case "cleared": return `Cleared in an update: was ${value(c.key, c.was)}`;
    case "conflict": return "The documents now disagree on this value";
    case "held": return `Read again in an update (${passages}); your value stands`;
    default: return `${passages} more read for this value`;
  }
}

function chipOf(c: Change): ChipKind {
  if (c.kind === "scope") return c.how === "in" ? "added" : c.how === "out" ? "removed" : "check";
  return c.how === "filled" ? "filled" : c.how === "conflict" ? "check" : c.how === "held" ? "held" : "changed";
}

/** The elements that show each change, and the one motion each plays. */
function inkItems(root: HTMLElement, changes: Change[]): InkItem[] {
  const items: InkItem[] = [];
  const entries = new Set<string>();
  for (const c of changes) {
    if (c.kind === "scope") {
      const motion: Motion = c.how === "in" ? "ink-on" : c.how === "out" ? "ink-off" : "ink-check";
      root.querySelectorAll(`.sw-cell[data-leaf="${CSS.escape(c.leaf)}"]`).forEach((el) => items.push({ el, motion }));
      entries.add(c.entry);
      continue;
    }
    const key = CSS.escape(c.key);
    // A text value to check writes in like any other; its rose says the rest.
    const value: Motion | null = c.how === "held" ? "ink-held" : c.how === "sources" ? null : "ink-write";
    root.querySelectorAll(`.pf-row[data-key="${key}"][data-part="${CSS.escape(c.part)}"]`).forEach((row) => {
      if (value === "ink-held") items.push({ el: row, motion: value });
      else if (value) items.push({ el: row.querySelector(".pf-value") ?? row, motion: value });
      const dot = row.querySelector(".pf-sources summary");
      if (dot && c.added > 0) items.push({ el: dot, motion: "ink-pop" });
    });
    root.querySelectorAll(`.pf-count[data-key="${key}"]`).forEach((el) => items.push({ el, motion: value ?? "ink-pop" }));
    if (value && value !== "ink-held") root.querySelectorAll(`.pf-t[data-key="${key}"]`).forEach((el) => items.push({ el, motion: value }));
  }
  for (const e of entries) {
    const count = root.querySelector(`.sw-matrow[data-entry="${CSS.escape(e)}"] .sw-count`);
    if (count) items.push({ el: count, motion: "ink-pop" });
  }
  return items;
}

/** A passage's opening words for the strip's latest-find line. */
function excerptOf(text: string): string {
  const t = text.replace(/\s+/g, " ").trim();
  return t.length > 90 ? `${t.slice(0, 88).trimEnd()}…` : t;
}

// The Project card reads top down, as a brief does: who and where, the class
// of building, the work, then that kind of building's subclass, size and
// conditions. Approvals and delivery follow. A key not placed here falls to
// delivery so a new field is never hidden.
// Unit mix, one count per type, smallest first (docs/plans/2026-11-08-unit-mix.md).
const UNIT_MIX = ["hdr.scale.units_studio", "hdr.scale.units_1_bed", "hdr.scale.units_2_bed", "hdr.scale.units_3_bed", "hdr.scale.units_4_bed"];
const UNIT_MIX_WORDS: Record<string, string> = {
  "hdr.scale.units_studio": "studio", "hdr.scale.units_1_bed": "one-bed", "hdr.scale.units_2_bed": "two-bed",
  "hdr.scale.units_3_bed": "three-bed", "hdr.scale.units_4_bed": "four-plus-bed",
};

type Tier = "identity" | "place" | "class" | "work" | "budget" | "subclass" | "scale" | "complexity" | "approvals" | "delivery";

const TIERS: Record<Tier, string[]> = {
  identity: ["site.address", "proj.client", "proj.landlord", "det.state"],
  place: ["site.lot", "proj.lga", "proj.zone"],
  class: ["hdr.building_class"],
  work: ["hdr.work_type"],
  budget: ["proj.budget_stated", "proj.budget_aspiration", "proj.contributions_stated"],
  subclass: ["hdr.subclass", "det.ncc_class"],
  // Pairs that look alike but follow different definitions sit together.
  scale: [
    "hdr.scale.site_sqm", "hdr.scale.storeys", "hdr.scale.total_storeys", "hdr.scale.levels", "det.rise_in_storeys",
    "hdr.scale.existing_gfa_sqm", "hdr.scale.added_gfa_sqm", "hdr.scale.gfa_sqm", "hdr.scale.total_gfa_sqm", "det.floor_area", "det.effective_height",
    "hdr.scale.units", "hdr.scale.dwellings", "hdr.scale.residential_units", ...UNIT_MIX,
    "hdr.scale.bedrooms", "hdr.scale.bathrooms", "hdr.scale.car_parks", "hdr.scale.garage_spaces",
    "proj.works_levels", "det.occupant_load", "det.existing_building", "det.existing_building_year",
  ],
  complexity: [
    "hdr.cond.planning", "hdr.cond.procurement_route", "hdr.cond.quality_level",
    "hdr.cond.access_constraints", "proj.work_hours", "hdr.cond.operational_constraints",
    "hdr.cond.stakeholder_complexity", "hdr.cond.environmental_sensitivity",
    "det.flood_hazard", "det.bushfire_prone_land", "det.heritage_status",
  ],
  approvals: ["fact.consent_number", "fact.consent_date", "fact.consent_lapse_date", "det.ncc_edition"],
  delivery: [
    "fact.contract_form", "fact.contract_basis", "fact.defects_liability_period", "fact.design_life",
    "proj.program", "proj.target_da_lodgement", "proj.target_construction_start", "proj.target_completion",
    "proj.start_conditions", "proj.tenderers",
    "proj.sustainability", "proj.flag.sustainability_target", "proj.regulated_design", "proj.flag.contract_admin",
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

export function Profile({ projectId, tick, workTick = 0, onJump, onSignedOut, onShowNotRead, onIdentity, feedbackTarget, documentLabel, onRunState, onOpenView }: Props) {
  const [sourceSystem, setSourceSystem] = useState("");
  const [sourcesOpen, setSourcesOpen] = useState(false);
  const [data, setData] = useState<ProfileData | null>(null);
  const [error, setError] = useState("");
  const [requesting, setRequesting] = useState(false);
  const [requestNote, setRequestNote] = useState("");
  const [entry, setEntry] = useState("");
  const [addingPart, setAddingPart] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const dataRef = useRef(data);
  dataRef.current = data;

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

  // The update in progress. Progress is polled each second while anything is
  // pending (it replaces the old 2 s profile poll); the profile is fetched
  // when its revision moves, or when the profile still says pending after
  // the work is done (a missed completion event).
  const pending = (data?.pending_documents ?? 0) > 0;
  const refresh = useCallback(() => {
    profileApi.get(projectId).then((p) => { setData(p); setError(""); }, fail);
  }, [projectId, fail]);
  const [pendingFinds, setPendingFinds] = useState<Record<string, number>>({});
  const [latestFind, setLatestFind] = useState<LatestFind | null>(null);
  const glintAt = useRef<Record<string, number>>({});
  const runDocs = useRef<RunDoc[]>([]);
  const docLabel = useCallback((id: string) => documentLabel?.(id) || runDocs.current.find((d) => d.id === id)?.filename || "document", [documentLabel]);
  // A passage just read: glint what it spoke to and count it, until its pass is committed.
  const onFinds = useCallback((finds: Find[]) => {
    const p = dataRef.current, el = root.current;
    if (!p || !el) return;
    const allowed = motionAllowed() && el.offsetParent !== null;
    const box = viewBox(el);
    const now = performance.now();
    const glint = (target: Element | null | undefined, key: string) => {
      if (!allowed || !target || !inView(target, box) || now - (glintAt.current[key] ?? 0) < 600) return;
      glintAt.current[key] = now;
      play(target, "ink-glint");
    };
    const add: Record<string, number> = {};
    let latest: LatestFind | null = null;
    for (const f of finds) {
      const t = findTargets(f);
      for (const k of t.fields) {
        add[k] = (add[k] ?? 0) + 1;
        el.querySelectorAll(`.pf-row[data-key="${CSS.escape(k)}"], .pf-count[data-key="${CSS.escape(k)}"]`).forEach((row) => glint(row, k));
      }
      for (const leaf of t.leaves) glint(el.querySelector(`.sw-cell[data-leaf="${CSS.escape(leaf)}"]`), leaf);
      // Which systems are being read is state, not decoration: the ring shows even without motion.
      for (const e of t.entries) play(el.querySelector(`.sw-matrow[data-entry="${CSS.escape(e)}"]`), "ink-reading");
      if (allowed) play(document.querySelector(`#doc-${CSS.escape(f.document_id)} .reg-line`), "ink-glint");
      const what = t.fields.map((k) => profileFields(p).find((x) => x.key === k)?.label).find(Boolean)
        ?? t.leaves.map((l) => p.systems.flatMap((g) => g.rows).find((r) => r.leaf === l)?.label).find(Boolean);
      if (what) latest = { what, excerpt: excerptOf(f.excerpt), where: `${docLabel(f.document_id)}${f.page ? ` p. ${f.page}` : ""}` };
    }
    if (Object.keys(add).length > 0) setPendingFinds((old) => {
      const merged = { ...old };
      for (const [k, n] of Object.entries(add)) merged[k] = (merged[k] ?? 0) + n;
      return merged;
    });
    if (latest) setLatestFind(latest);
  }, [docLabel]);
  const { run, start, dismiss } = useProfileRun({ projectId, pending, revision: data?.revision ?? 0, onRevision: refresh, onFinds, onSignedOut });
  runDocs.current = run.docs;

  // What changed since the user last looked, remembered in this browser; and
  // between two revisions, the changes to ink as they arrive.
  const [marks, setMarks] = useState<Change[]>([]);
  const [receiptOpen, setReceiptOpen] = useState(false);
  const [next, setNext] = useState<Next | null>(null);
  const previous = useRef<{ project: string; snap: Snapshot } | null>(null);
  const tokenText = useRef<Record<string, string>>({});
  useLayoutEffect(() => {
    if (!data || data.project_id !== projectId) return;
    const now = snapshotOf(data);
    const before = previous.current?.project === projectId ? previous.current.snap : null;
    previous.current = { project: projectId, snap: now };
    let seen = readSeen(projectId);
    if (!seen) {
      seen = now;
      writeSeen(projectId, now);
    }
    setMarks(diffSnapshots(seen, now).filter(markable));
    const el = root.current;
    if (el && before && now.revision > before.revision) {
      // A pass was committed: its passages now count as sources.
      setPendingFinds({});
      const changes = diffSnapshots(before, now);
      if (changes.length > 0 && el.offsetParent !== null && motionAllowed()) {
        // A rewritten word is struck through as the new one writes in.
        for (const c of changes) {
          const old = c.kind === "field" && c.how === "changed" ? tokenText.current[c.key] : "";
          const token = old ? el.querySelector(`.pf-t[data-key="${CSS.escape((c as { key: string }).key)}"]`) : null;
          if (token && old !== token.textContent) {
            token.setAttribute("data-ink-was", old);
            window.setTimeout(() => token.removeAttribute("data-ink-was"), 1200);
          }
        }
        inkInOrder(inkItems(el, changes), viewBox(el));
      }
    }
    el?.querySelectorAll<HTMLElement>(".pf-t[data-key]").forEach((t) => { tokenText.current[t.dataset.key ?? ""] = t.textContent ?? ""; });
  }, [data, projectId]);

  // An update that finished here opens its summary and asks what it changes
  // elsewhere. Text being prepared after an upload is not an update: it ends quietly.
  const followed = useRef(false);
  useEffect(() => {
    if (run.phase === "running") followed.current = true;
    if (run.phase !== "done" || !followed.current) return;
    followed.current = false;
    setLatestFind(null);
    if (!run.started && run.docs.length === 0) return;
    setReceiptOpen(true);
    runApi.next(projectId).then(setNext, () => setNext(null));
  }, [run.phase, run.started, run.docs.length, projectId]);
  useEffect(() => {
    setReceiptOpen(false);
    setNext(null);
    setLatestFind(null);
    setPendingFinds({});
  }, [projectId]);

  const markSeen = useCallback(() => {
    const p = dataRef.current;
    if (p) writeSeen(projectId, snapshotOf(p));
    setMarks([]);
  }, [projectId]);
  const closeReceipt = useCallback(() => {
    setReceiptOpen(false);
    dismiss();
  }, [dismiss]);

  // Changed rows below the visible part of the page, offered as one jump.
  const [below, setBelow] = useState(0);
  useEffect(() => {
    const el = root.current, scroller = el?.closest(".cockpit-scroll");
    if (!el || !scroller) return;
    const count = () => {
      const box = viewBox(el);
      setBelow(Array.from(el.querySelectorAll("[data-changed='true']")).filter((n) => n.getBoundingClientRect().top > box.bottom).length);
    };
    count();
    scroller.addEventListener("scroll", count, { passive: true });
    return () => scroller.removeEventListener("scroll", count);
  }, [marks]);

  useEffect(() => {
    const done = run.docs.filter((d) => d.state === "done").length;
    // Reading means documents being read for the profile, not text being prepared.
    const reading = run.phase === "running" && run.docs.some((d) => d.state === "queued" || d.state === "reading");
    onRunState?.({ reading, changes: marks.length, next, read: done, total: run.docs.length });
  }, [run.phase, run.docs, marks.length, next, onRunState]);

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

  // Several values at once (a bulk change from the scope rows). The writes
  // are independent rows, so they go together; one read then settles the
  // profile, since each write's reply may predate the others.
  const setMany = useCallback(
    async (writes: { key: string; body: { part_id?: string; value?: string | null; reset?: boolean } }[]) => {
      if (writes.length === 0) return;
      const results = await Promise.allSettled(writes.map((w) => profileApi.set(projectId, w.key, w.body)));
      const failed = results.find((r): r is PromiseRejectedResult => r.status === "rejected");
      try {
        setData(await profileApi.get(projectId));
        if (failed) fail(failed.reason);
        else setError("");
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
      const queued = p.pending_documents > 0;
      if (queued) {
        // This update's marks start from here.
        writeSeen(projectId, snapshotOf(p));
        setMarks([]);
        setNext(null);
        setReceiptOpen(false);
        start();
      }
      setData(p);
      setError("");
      setRequestNote(queued || p.failed_documents > 0 ? "" : "No new reading queued. Check source coverage and unresolved passages.");
    } catch (e) {
      fail(e);
    } finally {
      setRequesting(false);
    }
  }, [projectId, fail, start]);

  const site = useSiteFields(projectId, tick, data?.site_details ?? [], fail);
  const addressLine = site.fields.find((f) => f.key === "site.address")?.value ?? "";
  // The banner reads client, lot, area and address, as a title block does.
  const fact = (key: string) => {
    const f = [...(data?.facts ?? []), ...(data?.details ?? []), ...(data?.header ?? []), ...site.fields].find((x) => x.key === key && x.value !== "");
    return f ? f.options?.find((o) => o.id === f.value)?.label ?? f.value : "";
  };
  const area = [fact("proj.lga"), fact("det.state")].filter(Boolean).join(", ");
  const identity = data ? [fact("proj.client"), fact("site.lot"), area, addressLine || "Site address not stated"].filter(Boolean).join("  ·  ") : addressLine;
  useEffect(() => {
    onIdentity?.(identity);
  }, [identity, onIdentity]);

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
  // Scope entries start collapsed; "Go" on a check opens the first one.
  const selected = entry;
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

  // The update's marks, by field and by scope cell.
  const valueText = (key: string, v: string) => {
    if (!v) return "not known";
    const f = profileFields(data).find((x) => x.key === key);
    const label = f?.options?.find((o) => o.id === v)?.label ?? v;
    return f?.unit ? `${label} ${f.unit === "m2" ? "m²" : f.unit}` : label;
  };
  const fieldMarks = new Map<string, string>();
  const scopeMarks: ScopeMarks = { cells: {}, entries: {} };
  for (const c of marks) {
    if (c.kind === "field") fieldMarks.set(c.id, markText(c, valueText));
    else {
      scopeMarks.cells[c.leaf] = markText(c, valueText);
      scopeMarks.entries[c.entry] = (scopeMarks.entries[c.entry] ?? 0) + 1;
    }
  }
  const markOf = (f: ProfileField) => fieldMarks.get(`${f.key}|${f.part_id}`);
  const wholePart = data.parts.find((p) => p.kind === "whole")?.id ?? "";
  const pendingOf = (f: ProfileField) => pendingFinds[f.key] ?? 0;
  const focusField = (f: ProfileField) => {
    const control = document.getElementById(`pf-${f.key}-${f.part_id}`);
    control?.scrollIntoView({ block: "center", behavior: motionAllowed() ? "smooth" : "auto" });
    control?.focus({ preventScroll: true });
  };
  // A receipt chip brings its items into view and marks them for a moment.
  const onChip = (kind: ChipKind) => {
    const el = root.current;
    if (!el) return;
    const targets = marks.filter((c) => chipOf(c) === kind).flatMap((c) => c.kind === "field"
      ? Array.from(el.querySelectorAll(`.pf-row[data-key="${CSS.escape(c.key)}"], .pf-count[data-key="${CSS.escape(c.key)}"]`))
      : Array.from(el.querySelectorAll(`.sw-cell[data-leaf="${CSS.escape(c.leaf)}"]`)));
    if (targets.length === 0) return;
    targets[0].scrollIntoView({ block: "center", behavior: motionAllowed() ? "smooth" : "auto" });
    targets.forEach((t, i) => play(t, "ink-flash", motionAllowed() ? 250 + i * 30 : 0));
  };
  const onBelow = () => {
    const el = root.current;
    if (!el) return;
    const box = viewBox(el);
    Array.from(el.querySelectorAll("[data-changed='true']")).find((n) => n.getBoundingClientRect().top > box.bottom)
      ?.scrollIntoView({ block: "center", behavior: motionAllowed() ? "smooth" : "auto" });
  };
  // Facts every RFP's page one asks for, and facts documents seldom settle.
  const unknown = (keys: string[]) => keys.map((k) => cardFields.find((f) => f.key === k)).filter((f): f is ProfileField => !!f && !f.value && f.band !== "user");
  const rfpUnknown = unknown(["proj.zone", "site.lot"]);
  const whileFields = unknown(["proj.budget_stated", "hdr.cond.procurement_route", "proj.program"]);

  return (
    <div className="profile" ref={root}>
      <ProfileFeedback target={feedbackTarget}>
      <ProfileStrip
        data={data}
        run={run}
        marks={receiptOf(marks)}
        marksTotal={marks.length}
        receiptOpen={receiptOpen}
        next={next}
        checks={checks}
        requestNote={requestNote}
        sourcesOpen={sourcesOpen}
        latestFind={run.phase === "running" ? latestFind : null}
        below={run.phase === "running" ? below : 0}
        rfpUnknown={rfpUnknown}
        whileFields={whileFields}
        docName={(d) => documentLabel?.(d.id) || d.filename}
        kindLabel={(k) => KIND_LABELS[k] ?? k.replace(/_/g, " ")}
        onSet={setField}
        onCheck={goToCheck}
        onShowNotRead={onShowNotRead}
        onSources={() => setSourcesOpen(!sourcesOpen)}
        onBelow={onBelow}
        onChip={onChip}
        onMarkSeen={markSeen}
        onCloseReceipt={closeReceipt}
        onOpenView={onOpenView}
        onFocusField={focusField}
      />
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

      <section className="pf-lede" aria-label="Summary">
        <ProjectSummary fields={cardFields} />
      </section>

      {/* Update sits in the title band (owner, 10 Nov 2026); the button's own
          refresh mark turns while busy, so no second spinner in the label. */}
      <PageHeaderAction surface="profile">
        <UpdateButton onClick={update} disabled={requesting} title={updateTitle(data)} busy={requesting || (data.failed_documents === 0 && data.unread_documents === 0 && data.active_documents > 0)}>
          {requesting ? "Requesting profile update…" : data.failed_documents > 0 ? "Retry project profile" : data.unread_documents > 0 ? "Update" : data.active_documents > 0 ? "Updating project profile…" : data.pending_documents > 0 ? "Profile update queued" : "Update"}
          {!requesting && data.failed_documents === 0 && data.unread_documents > 0 && <span className="update-new" aria-hidden="true">{data.unread_documents} new</span>}
        </UpdateButton>
        <MoreMenu label="Profile actions" items={[
          { label: "Download project profile", icon: <IconDownload />, onSelect: () => window.print() },
          { label: "Add part", icon: <IconPlus />, onSelect: () => setAddingPart(true) },
        ]} />
      </PageHeaderAction>

      {/* No "Project" title (owner, 8 Nov 2026): the card's blue head row says it. */}
      <section className="pf-section" aria-label="Project">
        {addingPart && (
          <div className="pf-head pf-head-bare">
            <div className="pf-head-actions">
              <AddPart projectId={projectId} onDone={() => { setAddingPart(false); profileApi.get(projectId).then(setData, fail); }} onCancel={() => setAddingPart(false)} onError={fail} />
            </div>
          </div>
        )}
        <ProjectCard card={card} parts={data.parts} onSet={setField} onJump={onJump} markOf={markOf} pendingOf={pendingOf} />
      </section>

      <ScopeWorks
        projectId={projectId}
        data={data}
        tick={tick + workTick}
        selected={selected}
        onSelect={setEntry}
        onScope={setScope}
        onSet={set}
        onSetMany={setMany}
        onJump={onJump}
        onSources={openSources}
        onSignedOut={onSignedOut}
        changes={scopeMarks}
      />

      <ConsultantTeam appointed={data.appointed ?? []} />

      {/* The spaces the building provides, below the consultant team (owner,
          11 Nov 2026; docs/plans/2026-11-11-accommodation-schedule-lines.md). */}
      {data.accommodation && (
        <Accommodation projectId={projectId} table={data.accommodation} onJump={onJump} onError={fail}
          markOf={(keys) => keys.map((k) => fieldMarks.get(`${k}|${wholePart}`)).find(Boolean)}
          onChange={(next) => setData((d) => (d ? { ...d, accommodation: next } : d))} />
      )}
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

// The Project card is a drawing title block in the three columns of RFP page
// one (internal/knowledge/rfp.go ProjectGrid), so a fact sits in the same
// place on both. Each row reads label then value, as the RFP grid does. Only
// the RFP's key fields show until the card is opened (plus any value the
// documents disagree on); open, the rest follow, the unknown tail folded.
// Listed keys come first, in this order; the rest of a column's tiers follow;
// a key no column claims falls to Delivery.
const COLUMNS: { title: string; key: string[]; keys: string[]; tiers: Tier[] }[] = [
  {
    title: "Site and planning",
    key: ["site.address", "site.lot", "proj.lga", "proj.zone", "hdr.scale.site_sqm", "hdr.cond.planning"],
    keys: [
      "det.state", "fact.consent_number", "fact.consent_date", "fact.consent_lapse_date",
      "hdr.cond.access_constraints", "proj.work_hours", "hdr.cond.environmental_sensitivity", "det.flood_hazard", "det.bushfire_prone_land", "det.heritage_status",
    ],
    tiers: ["place"],
  },
  {
    title: "The building",
    key: [
      "hdr.work_type", "hdr.building_class", "hdr.subclass",
      "hdr.scale.storeys", "hdr.scale.total_storeys", "hdr.scale.levels", "hdr.scale.gfa_sqm", "hdr.scale.total_gfa_sqm",
      ...UNIT_MIX, "hdr.scale.bedrooms", "hdr.scale.bathrooms", "hdr.scale.car_parks", "hdr.scale.garage_spaces", "hdr.cond.quality_level",
    ],
    keys: ["proj.works_levels", "hdr.scale.added_gfa_sqm", "det.occupant_load", "det.ncc_class", "det.ncc_edition"],
    tiers: ["class", "work", "subclass", "scale"],
  },
  {
    title: "Delivery",
    key: ["proj.client", "hdr.cond.procurement_route", "proj.program", "proj.sustainability", "proj.regulated_design", "proj.budget_stated"],
    keys: [
      "proj.landlord", "hdr.cond.stakeholder_complexity", "hdr.cond.operational_constraints",
      "proj.budget_aspiration", "proj.contributions_stated", "proj.preliminary_estimate",
      "proj.target_da_lodgement", "proj.target_construction_start", "proj.target_completion", "proj.preliminary_construction_months",
      "proj.start_conditions", "proj.tenderers",
    ],
    tiers: ["identity", "budget", "approvals", "complexity", "delivery"],
  },
];

const KEY_FIELDS = new Set(COLUMNS.flatMap((c) => c.key));
const listed = (i: number) => [...COLUMNS[i].key, ...COLUMNS[i].keys];

function columnOf(key: string): number {
  const at = COLUMNS.findIndex((_, i) => listed(i).includes(key));
  if (at >= 0) return at;
  const byTier = COLUMNS.findIndex((c) => c.tiers.includes(tierOf(key)));
  return byTier >= 0 ? byTier : COLUMNS.length - 1;
}

/** One column's fields: its listed keys in order, then the rest in tier order. */
function inColumn(fields: ProfileField[], i: number): ProfileField[] {
  const keys = listed(i);
  const mine = fields.filter((f) => columnOf(f.key) === i);
  const at = (f: ProfileField) => (keys.includes(f.key) ? keys.indexOf(f.key) : keys.length);
  return mine.map((f, n) => ({ f, n })).sort((a, b) => at(a.f) - at(b.f) || a.n - b.n).map(({ f }) => f);
}

function ProjectCard({ card, parts, onSet, onJump, markOf, pendingOf }: {
  card: Arranged; parts: ProfileData["parts"]; onSet: Setter; onJump: (id: string) => void;
  markOf: (f: ProfileField) => string | undefined; pendingOf: (f: ProfileField) => number;
}) {
  const [open, setOpen] = useState(false);
  const row = (f: ProfileField) => <FieldRow key={f.key + f.part_id} field={f} onSet={onSet} onJump={onJump} parts={parts} compact paired changed={markOf(f)} pending={pendingOf(f)} />;
  const all = (Object.keys(card) as Tier[]).flatMap((t) => [...card[t].shown, ...card[t].more]);
  // A conflict always shows, so "to check" can reach it with the card closed.
  // A mix count shows up front once known; the empty ones fold away.
  const always = (f: ProfileField) => (KEY_FIELDS.has(f.key) && !(UNIT_MIX.includes(f.key) && unanswered(f))) || f.band === "red";
  const rest = all.filter((f) => !always(f));
  // Parts sit in The building, after the class they divide. Bedrooms,
  // bathrooms and car parks share one Counts row where the first of them sat.
  const withParts = (fields: ProfileField[], col: number) => {
    if (COLUMNS[col].title !== "The building") return fields.map(row);
    const counts = fields.filter((f) => COUNT_KEYS.includes(f.key));
    const out: ReactNode[] = [];
    for (const f of fields) {
      if (!COUNT_KEYS.includes(f.key)) out.push(row(f));
      else if (f === counts[0]) out.push(<CountsRow key="counts" fields={counts} onSet={onSet} onJump={onJump} changed={markOf} pending={pendingOf} />);
      if (f.key === "hdr.building_class") out.push(<PartsRow key="parts" parts={parts} />);
    }
    if (!fields.some((f) => f.key === "hdr.building_class")) out.push(<PartsRow key="parts" parts={parts} />);
    return out;
  };
  return (
    <>
      <div className="pf-card pf-tiers pf-titleblock pf-paired">
        {COLUMNS.map((c, i) => {
          const fields = inColumn(all.filter(always), i);
          const extra = inColumn(rest, i);
          return (
            <div key={c.title} className="pf-tb-col" role="group" aria-label={c.title}>
              <h3 className="pf-tb-head">{c.title}</h3>
              {fields.length > 0 ? withParts(fields, i) : <p className="muted pf-tb-empty">Nothing read yet.</p>}
              {open && extra.filter((f) => !unanswered(f)).map(row)}
              {open && <NotYetKnown fields={extra.filter(unanswered)} render={row} className="pf-col-more" />}
            </div>
          );
        })}
      </div>
      {/* A chevron tab on the card's foot rule, not a line of its own (owner, 10 Nov 2026). */}
      {rest.length > 0 && (
        <button type="button" className="pf-card-toggle" aria-expanded={open} aria-label={open ? "View less" : "View more"} title={open ? "View less" : "View more"} onClick={() => setOpen(!open)} />
      )}
    </>
  );
}

// The project in one sentence, built in code from the card's values. Each
// value is underlined the way it was decided, by the origin rule (owner,
// 10 Nov 2026): green from the documents (dotted where Jev leans on them),
// dotted blue typical for the building, rust the user's own word.
const WORDS = ["", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine"];

function ProjectSummary({ fields }: { fields: ProfileField[] }) {
  const get = (...keys: string[]) => keys.map((k) => fields.find((f) => f.key === k && f.value !== "")).find(Boolean);
  const text = (f: ProfileField) => f.options?.find((o) => o.id === f.value)?.label ?? f.value;
  const soft = (t: string) => (/^[A-Z][a-z]/.test(t) ? t[0].toLowerCase() + t.slice(1) : t);
  const an = (t: string) => (/^[aeiou]/i.test(t) && !/^[A-Z]{2}/.test(t) ? "an" : "a");
  const token = (f: ProfileField | undefined, show: (t: string) => string = (t) => t) => {
    if (!f) return null;
    const how = f.band === "user" ? "pencil-user" : f.band === "amber" ? "pencil-jev" : f.band === "red" ? "mark-check" : f.band === "suggested" ? "pencil-typical" : "pencil-doc";
    const said = f.band === "user" ? "Set by you" : f.band === "amber" ? "Picked by Jev from the documents" : f.band === "red" ? "Sources differ" : f.band === "suggested" ? "Typical for this building" : "From the documents";
    return (
      <button type="button" className={`pf-t ${how}`} data-key={f.key} title={`${f.label}: ${said}`}
        onClick={() => document.getElementById(`pf-${f.key}-${f.part_id}`)?.focus()}>
        {show(text(f))}
      </button>
    );
  };
  const work = get("hdr.work_type"), sub = get("hdr.subclass"), storeys = get("hdr.scale.storeys", "hdr.scale.total_storeys", "hdr.scale.levels");
  const gfa = get("hdr.scale.gfa_sqm", "hdr.scale.total_gfa_sqm"), site = get("hdr.scale.site_sqm"), lga = get("proj.lga"), zone = get("proj.zone");
  const quality = get("hdr.cond.quality_level"), client = get("proj.client"), planning = get("hdr.cond.planning"), budget = get("proj.budget_stated");
  if (!work && !sub && !client && !gfa) return <p className="pf-summary pf-summary-empty">The summary appears once the documents are read.</p>;
  const mix = UNIT_MIX.map((k) => get(k)).filter((f): f is ProfileField => !!f && /^\d+$/.test(f.value));
  const n = storeys ? Number(storeys.value) : 0;
  const storeyWords = (t: string) => `${WORDS[n] ?? t} ${n === 1 ? "storey" : "storeys"}`;
  return (
    <p className="pf-summary">
      A {work ? token(work, soft) : "project"}
      {sub && <>: {an(soft(text(sub)))} {token(sub, soft)}</>}
      {storeys && <>, of {token(storeys, storeyWords)}</>}
      {gfa && <>{storeys ? " and " : ", "}{token(gfa, (t) => `${t} m²`)}</>}
      {mix.length > 0 && <>, with {mix.map((f, i) => (
        <span key={f.key}>{i === 0 ? "" : i === mix.length - 1 ? " and " : ", "}{token(f, (t) => `${t} ${UNIT_MIX_WORDS[f.key]}`)}</span>
      ))} {mix.length === 1 && mix[0].value === "1" ? "unit" : "units"}</>}
      {site && <> on a {token(site, (t) => `${t} m²`)} site</>}
      {lga && <> in {token(lga)}</>}
      {zone && <>, zoned {token(zone)}</>}.
      {(quality || client || planning || budget) && (
        <>
          {" "}{quality ? <>{an(soft(text(quality))) === "an" ? "An" : "A"} {token(quality, soft)}</> : "It is"}
          {client && <> for {token(client)}</>}
          {planning && <>, approved as {an(text(planning))} {token(planning)}</>}
          {budget && <>, with a construction budget of {token(budget)}</>}.
        </>
      )}
    </p>
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

function AddPart({ projectId, onDone, onCancel, onError }: { projectId: string; onDone: () => void; onCancel: () => void; onError: (e: unknown) => void }) {
  const [label, setLabel] = useState("");
  const [kind, setKind] = useState("building");
  async function add(e: FormEvent) {
    e.preventDefault();
    if (!label.trim()) return;
    try {
      await profileApi.addPart(projectId, label.trim(), kind);
      onDone();
    } catch (err) {
      onError(err);
    }
  }
  return (
    <form onSubmit={add} className="pf-inline pf-add-part">
      <input className="input" aria-label="Part label" placeholder="Part name" value={label} onChange={(e) => setLabel(e.target.value)} autoFocus />
      <select className="select" aria-label="Part kind" value={kind} onChange={(e) => setKind(e.target.value)}>
        {KINDS.map((k) => (
          <option key={k}>{k}</option>
        ))}
      </select>
      <button type="submit" className="btn btn-small">Add</button>
      <button type="button" className="cell-link" onClick={onCancel}>Cancel</button>
    </form>
  );
}

/** The project's parts, as a row of The building column. */
function PartsRow({ parts }: { parts: ProfileData["parts"] }) {
  return (
    <div className="pf-row pf-pair pf-parts-row">
      <div className="pf-pair-head"><span className="pf-label">Parts</span></div>
      <div className="pf-value pf-parts" aria-label="Parts">
        {parts.map((p) => (
          <span className="pf-part" key={p.id} data-kind={p.kind}>{p.label}</span>
        ))}
      </div>
    </div>
  );
}

/** Rows the consultant team always shows, so the table reads as a slot to fill. */
const TEAM_PLACEHOLDER_ROWS = 3;

/** Consultants appointed through Procurement, one per row, padded with blank discipline rows. */
function ConsultantTeam({ appointed }: { appointed: NonNullable<ProfileData["appointed"]> }) {
  const blanks = Array.from({ length: Math.max(0, TEAM_PLACEHOLDER_ROWS - appointed.length) }, (_, i) => appointed.length + i + 1);
  return (
    <section className="pf-section pf-team" aria-labelledby="pf-team">
      <div className="pf-head section-band">
        <h2 id="pf-team">Consultant team</h2>
      </div>
      <table className="pf-team-table" data-testid="profile-appointed">
        <thead>
          <tr><th scope="col">Discipline</th><th scope="col">Consultant</th><th scope="col">Contact</th><th scope="col">Email</th><th scope="col">Phone</th></tr>
        </thead>
        <tbody>
          {appointed.map((a) => (
            <tr key={`${a.package_id}-${a.firm}`}>
              <th scope="row">{a.package}</th>
              <td title={a.firm}>{a.firm}</td>
              <td>{a.contact || "—"}</td>
              <td title={a.email || undefined}>{a.email ? <a className="cell-link" href={`mailto:${a.email}`}>{a.email}</a> : "—"}</td>
              <td>{a.phone ? <a className="cell-link" href={`tel:${a.phone.replace(/\s/g, "")}`}>{a.phone}</a> : "—"}</td>
            </tr>
          ))}
          {blanks.map((n) => (
            <tr key={`blank-${n}`} className="pf-team-blank">
              <th scope="row">Discipline {n}</th>
              <td /><td /><td /><td />
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
