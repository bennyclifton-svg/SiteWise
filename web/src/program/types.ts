// Shapes shared with the Go engine (internal/program/model.go). Input uses
// the shared fixture names; Output is the `computed` object of GET /program
// (plan 8.2), field for field, so the drag preview and the server answer can
// be compared as plain JSON.

export const KindStage = "stage";
export const KindActivity = "activity";
export const KindMilestone = "milestone";
export const KindHoldPoint = "hold_point";

export const ConstraintASAP = "asap";
export const ConstraintSNET = "snet";
export const ConstraintMSO = "mso";

export const LinkFS = "FS";
export const LinkSS = "SS";
export const LinkFF = "FF";
export const LinkSF = "SF";
export const LagWorking = "working";
export const LagCalendar = "calendar";

export const DayPublicHoliday = "public_holiday";
export const DayRDO = "rdo";
export const DayShutdown = "shutdown";
export const DayWorkingOverride = "working_override";

/** Monday to Friday in the Mon=1 … Sun=64 bitmask. */
export const DefaultWorkingWeek = 31;

export interface CalendarDay {
  day: string;
  kind: string;
  label?: string;
}

export interface Calendar {
  /** YYYY-MM-DD, or "" for a relative program. */
  start?: string;
  /** Mon=1, Tue=2 … Sun=64; 0 or absent means Monday to Friday. */
  working_week?: number;
  days?: CalendarDay[] | null;
}

export interface Activity {
  id: string;
  parent?: string;
  ordinal?: number;
  kind?: string;
  label?: string;
  /** Working days; null or absent is unknown and stays unknown. */
  duration?: number | null;
  low?: number | null;
  high?: number | null;
  constraint?: string;
  constraint_date?: string;
}

export interface Link {
  id: string;
  pred: string;
  succ: string;
  type?: string;
  lag?: number;
  lag_unit?: string;
}

export interface Input {
  calendar: Calendar;
  activities: Activity[] | null;
  links: Link[] | null;
}

export interface Status {
  known: number;
  total: number;
  words: string;
}

export interface Finish {
  typical: string | null;
  low: string | null;
  high: string | null;
  typical_index: number | null;
  low_index: number | null;
  high_index: number | null;
  complete: boolean;
}

/** One row's times; every number and date is null when unknown. */
export interface Times {
  es: number | null;
  ef: number | null;
  ls: number | null;
  lf: number | null;
  tf: number | null;
  ff: number | null;
  critical: boolean;
  start: string | null;
  finish: string | null;
  unknown: boolean;
}

export interface Conflict {
  activity_id: string;
  index: number;
  held_index: number;
  held_by: string;
  /** Absent when the hold comes through a stage tie, as in Go (omitempty). */
  link_id?: string;
}

export interface Output {
  status: Status;
  finish: Finish;
  activities: Record<string, Times>;
  driving_links: string[];
  cycle: string[] | null;
  conflicts: Conflict[];
}
