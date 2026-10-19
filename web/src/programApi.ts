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
  package_id?: string | null;
  interface_id?: string | null;
  duration_days: number | null;
  duration_low: number | null;
  duration_high: number | null;
  duration_origin: string | null;
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
  /** Present on write responses only. */
  toasts?: string[];
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
        const body = JSON.parse(e.message) as { error?: string };
        if (body.error) throw new ApiError(422, body.error);
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
  exportHref: (p: string, view: "collapsed" | "expanded") => `/api${base(p)}/export?format=xlsx&view=${view}`,
};

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
