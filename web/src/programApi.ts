// Client for the Program routes (plan docs/plans/2026-10-18-program.md
// section 8). Every route answers with the full view (8.2), so the client
// replaces its state and never merges. No route calls Jev.

import { ApiError, request } from "./api";
import type { Input, Output } from "./program/types";

export interface ProgramHead {
  id: string;
  project_id: string;
  version: number;
  start_date: string | null;
  start_origin: string | null;
  skeleton_id: string | null;
  stage_revision: number;
  working_week: number;
}

export interface ProgramActivity {
  id: string;
  parent_id: string | null;
  ordinal: number;
  kind: string;
  label: string;
  catalogue_stage_id?: string | null;
  package_id?: string | null;
  interface_id?: string | null;
  duration_days: number | null;
  duration_low: number | null;
  duration_high: number | null;
  duration_origin: string | null;
  /** Where the duration came from; `assumed` says it rests on a default to adjust. */
  duration_source?: { assumed?: string } & Record<string, unknown>;
  constraint_kind: string;
  constraint_date: string | null;
  review_status: "proposed" | "accepted" | string;
  origin: string;
  version: number;
}

export interface ProgramLink {
  id: string;
  pred_id: string;
  succ_id: string;
  type: "FS" | "SS" | "FF" | "SF" | string;
  lag_days: number;
  lag_unit: "working" | "calendar" | string;
  origin: string;
  interface_id?: string | null;
  version: number;
}

export interface ProgramCheck {
  code: string;
  activity_id?: string;
  message: string;
}

export interface ProgramQuestion {
  key: string;
  kind: string;
  label: string;
  /** Labels shown; values sent, same order. An option past the end of values ("Not sure") dismisses. */
  options: string[];
  values: string[];
  number?: boolean;
  narrows_by_days: number;
}

/** A proposed document reading (plan 10.2, 10.3): a proposal until accepted. */
export interface ProgramReading {
  id: string;
  /** "" for an awarded proposal's R7 reading. */
  import_id: string;
  /** null when the stage it belongs to is gone (plan 7.7). */
  target_activity_id: string | null;
  catalogue_stage_id: string | null;
  field: "start" | "finish" | "duration" | string;
  /** The file's text, verbatim. */
  value: string;
  /** Code's parse: {row?, date?, low?, high?}. */
  parsed: { row?: string; date?: string; low?: number | null; high?: number | null } | null;
  band: string;
  decided_by: string;
  document_id: string;
  document_title: string;
  /** False when accepting would not change the target (a user value wins). */
  applies: boolean;
  source: "program_file" | "proposal_r7" | string;
  /** "From the awarded proposal: Design development 6 weeks" (R7 only). */
  text?: string;
}

export interface ProgramImport {
  id: string;
  document_id: string;
  filename: string;
  /** filing, reading, read or not_read. */
  status: string;
  not_read: string | null;
  /** "4 readings from Smith program.xlsx". */
  text: string;
  readings: number;
  /** Target ids no row matched, and ids Jev did not answer (grey, retried). */
  not_matched: string[] | null;
  grey: string[] | null;
  unmatched_rows: { row: string; label: string; start?: string; finish?: string; duration?: string }[] | null;
  unmatched: number;
  /** Links the file's predecessor column makes between two matched rows, not yet in the program (plan 10.2). */
  links?: ProgramProposedLink[] | null;
}

/** A link code read from a file's predecessor column, between two matched activities. */
export interface ProgramProposedLink {
  pred_id: string;
  succ_id: string;
  type: string;
  lag_days: number;
  lag_unit: string;
}

export interface ProgramHoldEvidence {
  reading_id: string;
  document_id: string;
  passage_id: string;
  document_title: string;
  page: number;
  band: string;
  /** "Evidence found: <title>, page <n>". */
  text: string;
  document_date: string;
  date_label: string;
}

export interface ProgramHoldPoint {
  activity_id: string;
  released: { by: string; document_id?: string; document_title?: string; page?: number; document_date?: string; date_label?: string } | null;
  evidence: ProgramHoldEvidence[] | null;
  /** The known-limit tooltip (plan 10.4). */
  limit: string;
}

/** One ask-first review item (plan 7.4). */
export interface ProgramChange {
  id: string;
  kind: "add" | "remove" | "rename" | "duration" | "link" | string;
  source: string;
  source_key: string;
  target_activity_id: string | null;
  summary: string;
  default_choice: "adopt" | "keep" | string;
  edited: boolean;
  edited_since: boolean;
}

export interface ProgramChangeList {
  program_id: string;
  version: number;
  changes: ProgramChange[];
  counts: Record<string, number>;
}

export interface ProgramView {
  program: ProgramHead;
  activities: ProgramActivity[];
  links: ProgramLink[];
  calendar: { days: { day: string; kind: string; label: string; origin: string }[] | null };
  computed: Output;
  checks: ProgramCheck[] | null;
  question: ProgramQuestion | null;
  start_suggestion: string | null;
  skeleton: { id: string | null; label: string; reason: string };
  readings_pending: number;
  changes_pending: number;
  readings?: ProgramReading[] | null;
  imports?: ProgramImport[] | null;
  hold_points?: ProgramHoldPoint[] | null;
  /** The delivery stage catalogue, for remapping a user-made stage (owner answer Q5). */
  catalogue_stages?: { id: string; label: string }[];
  /** Present on write responses only (plan 8.1 feedback). */
  toasts?: string[];
  temp_ids?: Record<string, string>;
  healed_links?: ProgramLink[];
  chain_added?: ProgramLink[];
  chain_removed?: ProgramLink[];
  backward_links?: ProgramLink[];
  linked?: ProgramLink[];
}

/** One edit of an ops batch (plan 8.1). Any id may name an earlier temp_id. */
export interface ProgramOp {
  op: string;
  temp_id?: string;
  id?: string;
  ids?: string[];
  parent_id?: string;
  after_id?: string;
  kind?: string;
  label?: string;
  /** null makes the duration unknown; absent keeps it. */
  duration_days?: number | null;
  constraint_kind?: string;
  constraint_date?: string;
  catalogue_stage_id?: string;
  days?: number;
  start_date?: string | null;
  reason?: string;
  pred_id?: string;
  succ_id?: string;
  type?: string;
  lag_days?: number;
  lag_unit?: string;
  day?: string;
}

/** A refused batch (422): the plain message plus what it names. */
export class ProgramRefused extends ApiError {
  constructor(
    message: string,
    readonly field = "",
    readonly opIndex = -1,
    readonly cycle: string[] = [],
    readonly held: string[] = [],
  ) {
    super(422, message);
  }
}

/** A stale write: the server's current view and its message (8.1). */
export class ProgramConflict extends Error {
  constructor(readonly current: ProgramView, message: string) {
    super(message);
  }
}

const base = (p: string) => `/projects/${p}/program`;

// A 409 carries {error, current}; hand the caller the newer view to show.
async function write(p: Promise<ProgramView>): Promise<ProgramView> {
  try {
    return await p;
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) {
      try {
        const body = JSON.parse(e.message) as { error: string; current: ProgramView };
        if (body.current) throw new ProgramConflict(body.current, body.error);
      } catch (inner) {
        if (inner instanceof ProgramConflict) throw inner;
      }
    }
    if (e instanceof ApiError && e.status === 422) {
      try {
        const body = JSON.parse(e.message) as { error?: string; field?: string; op_index?: number; cycle?: string[]; held?: string[] };
        if (body.error) throw new ProgramRefused(body.error, body.field ?? "", body.op_index ?? -1, body.cycle ?? [], body.held ?? []);
      } catch (inner) {
        if (inner instanceof ApiError) throw inner;
      }
    }
    throw e;
  }
}

export const programApi = {
  /** Null when the project has no program yet (404 {"empty": true}). */
  read: async (p: string): Promise<ProgramView | null> => {
    try {
      return await request<ProgramView>("GET", base(p));
    } catch (e) {
      if (e instanceof ApiError && e.status === 404 && e.message.includes('"empty"')) return null;
      throw e;
    }
  },
  prepare: (p: string) => write(request<ProgramView>("POST", `${base(p)}/prepare`, { surface: "program" })),
  acceptAll: (p: string, v: ProgramView) => write(request<ProgramView>("POST", `${base(p)}/accept`, { version: v.program.version, all: true })),
  answer: (p: string, v: ProgramView, key: string, value: string) =>
    write(request<ProgramView>("POST", `${base(p)}/question`, { version: v.program.version, key, value })),
  setStart: (p: string, v: ProgramView, start: string | null) =>
    write(
      request<ProgramView>("POST", `${base(p)}/ops`, {
        program_id: v.program.id,
        version: v.program.version,
        ops: [{ op: "set_start", start_date: start }],
      }),
    ),
  /** One batch = one undo step; pinned to the version the edit was made on. */
  ops: (p: string, v: ProgramView, ops: ProgramOp[]) =>
    write(request<ProgramView>("POST", `${base(p)}/ops`, { program_id: v.program.id, version: v.program.version, ops })),
  undo: (p: string, v: ProgramView) => write(request<ProgramView>("POST", `${base(p)}/undo`, { version: v.program.version })),
  redo: (p: string, v: ProgramView) => write(request<ProgramView>("POST", `${base(p)}/redo`, { version: v.program.version })),
  changes: (p: string) => request<ProgramChangeList>("GET", `${base(p)}/changes`),
  /** Adopted changes apply as one undoable batch. */
  decide: (p: string, v: ProgramView, decisions: { id: string; choice: string }[]) =>
    write(request<ProgramView>("POST", `${base(p)}/changes/decide`, { version: v.program.version, decisions })),
  adoptAll: (p: string, v: ProgramView) => write(request<ProgramView>("POST", `${base(p)}/changes/decide`, { version: v.program.version, all: "adopt" })),
  /** Readings (a target's start, finish and duration go together) or rows. */
  accept: (p: string, v: ProgramView, ids: string[]) => write(request<ProgramView>("POST", `${base(p)}/accept`, { version: v.program.version, ids })),
  dismissReadings: (p: string, v: ProgramView, ids: string[]) => write(request<ProgramView>("POST", `${base(p)}/readings/dismiss`, { version: v.program.version, ids })),
  /** File rows no target matched, added as activities (or stages with no parent). */
  addRows: (p: string, v: ProgramView, rows: { import_id: string; row: string; parent_id: string }[]) =>
    write(request<ProgramView>("POST", `${base(p)}/accept`, { version: v.program.version, add_rows: rows })),
  release: (p: string, v: ProgramView, id: string, readingId = "") =>
    write(request<ProgramView>("POST", `${base(p)}/hold-points/release`, { version: v.program.version, id, reading_id: readingId })),
  /** Files the upload through intake; the readings arrive as a program.readings event. */
  importFile: async (p: string, file: File): Promise<{ import_id: string; document_id: string }> => {
    let res: Response;
    try {
      res = await fetch(`/api${base(p)}/import?name=${encodeURIComponent(file.name)}`, { method: "POST", credentials: "same-origin", body: file });
    } catch {
      throw new ApiError(0, "Can't reach SiteWise. Check your connection and try again.");
    }
    if (!res.ok) throw new ApiError(res.status, (await res.text()).trim() || res.statusText);
    return (await res.json()) as { import_id: string; document_id: string };
  },
  exportHref: (p: string, view: "collapsed" | "expanded") => `/api${base(p)}/export?format=xlsx&view=${view}`,
};

/** The newest `program.changed` announcement (plan 8.3): its program and version. */
export interface RemoteProgram {
  programId: string;
  version: number;
}

/**
 * Keeps the newest announcement. Another program id (the program was
 * recreated) replaces it whatever its version, since versions restart.
 */
export function newerAnnouncement(cur: RemoteProgram, programId: string, version: number): RemoteProgram {
  if (programId && cur.programId && programId !== cur.programId) return { programId, version };
  if (version <= cur.version) return cur;
  return { programId: programId || cur.programId, version };
}

/**
 * Whether an announcement means the shown program is stale: another
 * program, or a newer version of this one. An announcement already
 * fetched for never refetches again, so a GET that answers with a lower
 * version cannot loop.
 */
export function needsRefetch(remote: RemoteProgram, shown: { id: string; version: number }, fetched: RemoteProgram | null): boolean {
  if (remote.version <= 0) return false;
  if (fetched && fetched.programId === remote.programId && fetched.version === remote.version) return false;
  if (remote.programId && remote.programId !== shown.id) return true;
  return remote.version > shown.version;
}

/** The raw engine input of a view, for the TypeScript drag preview (plan 3, 8.2). */
export function toInput(v: ProgramView): Input {
  return {
    calendar: {
      start: v.program.start_date ?? "",
      working_week: v.program.working_week,
      days: (v.calendar.days ?? []).map((d) => ({ day: d.day, kind: d.kind, label: d.label })),
    },
    activities: v.activities.map((a) => ({
      id: a.id,
      parent: a.parent_id ?? "",
      ordinal: a.ordinal,
      kind: a.kind,
      label: a.label,
      duration: a.duration_days,
      low: a.duration_low,
      high: a.duration_high,
      constraint: a.constraint_kind,
      constraint_date: a.constraint_date ?? "",
    })),
    links: v.links.map((l) => ({ id: l.id, pred: l.pred_id, succ: l.succ_id, type: l.type, lag: l.lag_days, lag_unit: l.lag_unit })),
  };
}
