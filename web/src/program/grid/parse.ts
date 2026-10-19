// Typed formats for the grid (plan 9.5). Mirror of internal/program/parse.go
// rule for rule, so a value typed in the grid and the same value read from a
// program file mean the same thing. Go's \s is ASCII whitespace only, so the
// patterns spell it out as [\t\n\f\r ] instead of JavaScript's wider \s.

import { daysInMonth } from "../calendar";
import { LagCalendar, LagWorking, LinkFS } from "../types";

/** The schema's limit on duration_days. */
export const MaxDuration = 3650;

export type Parsed<T> = { ok: true; value: T } | { ok: false; error: string };

const ok = <T,>(value: T): Parsed<T> => ({ ok: true, value });
const fail = <T,>(error: string): Parsed<T> => ({ ok: false, error });

/** Go's %q for the values these messages quote. */
const q = (s: string): string => JSON.stringify(s);

// Go's strings.TrimSpace: Unicode white space, which JavaScript's trim()
// matches for every character a typed cell can hold.
const trim = (s: string): string => s.trim();

/** Working days; a single value has low === high, a range low < high. */
export interface Duration {
  low: number;
  high: number;
}

/** "0" makes a milestone. */
export function isMilestone(d: Duration): boolean {
  return d.high === 0;
}

// "4–6 wks", "4 - 6 weeks", "4 to 6 wks".
const rangeSplit = /^[\t\n\f\r ]*([0-9.]+[\t\n\f\r ]*[a-z]*)[\t\n\f\r ]*(?:–|—|-|to)[\t\n\f\r ]*([0-9.]+[\t\n\f\r ]*[a-z]*)[\t\n\f\r ]*$/;
// A number and an optional unit: "5", "5d", "3 wks".
const durationPart = /^([0-9]+(?:\.[0-9]+)?)[\t\n\f\r ]*([a-z]*)$/;

function durationUnit(u: string, perWeek: number): number | null {
  switch (u) {
    case "":
    case "d":
    case "dy":
    case "dys":
    case "day":
    case "days":
    case "wd":
    case "wds":
      return 1;
    case "w":
    case "wk":
    case "wks":
    case "week":
    case "weeks":
      return perWeek;
  }
  return null;
}

/**
 * Reads "5", "5d", "3w", "0", "3 wks", "10 days" and ranges such as
 * "4–6 wks". Weeks are working weeks of perWeek working days.
 */
export function parseDuration(s: string, perWeek: number): Parsed<Duration> {
  if (perWeek <= 0) perWeek = 5;
  const t = trim(s).toLowerCase();
  if (t === "") return fail("Enter a duration, for example 5, 5d or 3w.");
  const m = rangeSplit.exec(t);
  if (m) {
    let lowText = trim(m[1]);
    const highText = trim(m[2]);
    // "4–6 wks": the low end borrows the high end's unit.
    const hm = durationPart.exec(highText);
    if (hm) {
      const lm = durationPart.exec(lowText);
      if (lm && lm[2] === "") lowText = lm[1] + hm[2];
    }
    const low = durationValue(lowText, perWeek);
    if (!low.ok) return low;
    const high = durationValue(highText, perWeek);
    if (!high.ok) return high;
    if (low.value > high.value) return fail("A range runs from low to high, for example 4–6 wks.");
    return ok({ low: low.value, high: high.value });
  }
  const d = durationValue(t, perWeek);
  if (!d.ok) return d;
  return ok({ low: d.value, high: d.value });
}

function durationValue(t: string, perWeek: number): Parsed<number> {
  const m = durationPart.exec(trim(t));
  if (!m) return fail(`${q(t)} is not a duration. Use days or weeks, for example 5, 5d or 3w.`);
  const per = durationUnit(m[2], perWeek);
  if (per === null) return fail(`${q(m[2])} is not a unit. Use d for working days or w for weeks.`);
  // Decimals only when they land on whole working days ("1.5w" in a 4-day
  // week is 6 days); integer arithmetic avoids floating-point rounding.
  const dot = m[1].indexOf(".");
  const whole = dot < 0 ? m[1] : m[1].slice(0, dot);
  const frac = dot < 0 ? "" : m[1].slice(dot + 1);
  const digits = whole + frac;
  if (digits.length > 6) return fail(`Duration must be at most ${MaxDuration} working days.`);
  const num = parseInt(digits, 10);
  const den = 10 ** frac.length;
  if ((num * per) % den !== 0) return fail(`${trim(t)} is not a whole number of working days.`);
  const days = (num * per) / den;
  if (days > MaxDuration) return fail(`Duration must be at most ${MaxDuration} working days.`);
  return ok(days);
}

/** One entry of a predecessor cell: a grid row number, a link type and a lag. */
export interface PredRef {
  row: number;
  type: string;
  lag: number;
  lag_unit: string;
}

// "12", "12FS", "12SS+2d", "8FF-1d", "4SF", "7FS+3cd", "12+2d". Lag units:
// d (working days, the default), w (working weeks), cd or ed (calendar days).
const predPart =
  /^([0-9]{1,6})[\t\n\f\r ]*(fs|ss|ff|sf)?[\t\n\f\r ]*(?:([+-])[\t\n\f\r ]*([0-9]{1,4})[\t\n\f\r ]*(d|days?|w|wks?|weeks?|cd|ed|edays?)?)?$/;

/** Reads a comma- or semicolon-separated predecessor cell; empty is none. */
export function parsePredecessors(s: string, perWeek: number): Parsed<PredRef[]> {
  if (perWeek <= 0) perWeek = 5;
  const refs: PredRef[] = [];
  const seen = new Set<number>();
  for (const part of s.split(/[,;]/)) {
    const t = trim(part).toLowerCase();
    if (t === "") continue;
    const m = predPart.exec(t);
    if (!m) {
      return fail(
        `${q(trim(part))} is not a predecessor. Use a row number with an optional type and lag, for example 12, 12SS+2d or 7FS+3cd.`,
      );
    }
    const row = parseInt(m[1], 10);
    if (row < 1) return fail("Row numbers start at 1.");
    if (seen.has(row)) return fail(`Row ${row} is listed twice.`);
    seen.add(row);
    const ref: PredRef = { row, type: LinkFS, lag: 0, lag_unit: LagWorking };
    if (m[2]) ref.type = m[2].toUpperCase();
    if (m[3]) {
      let lag = parseInt(m[4], 10);
      switch (m[5] ?? "") {
        case "w":
        case "wk":
        case "wks":
        case "week":
        case "weeks":
          lag *= perWeek;
          break;
        case "cd":
        case "ed":
        case "eday":
        case "edays":
          ref.lag_unit = LagCalendar;
          break;
      }
      if (m[3] === "-") lag = -lag;
      ref.lag = lag;
    }
    if (ref.lag < -365 || ref.lag > 365) return fail("Lag must be between −365 and 365 days.");
    refs.push(ref);
  }
  return ok(refs);
}

/** Writes a PredRef back in the typed format: "12", "12SS+2d", "7FS+3cd". */
export function formatPredecessor(r: PredRef): string {
  let s = String(r.row);
  if (r.type !== LinkFS || r.lag !== 0) s += r.type;
  if (r.lag !== 0) {
    const unit = r.lag_unit === LagCalendar ? "cd" : "d";
    s += (r.lag > 0 ? "+" : "") + r.lag + unit;
  }
  return s;
}

/** How a column writes numeric dates. */
export type DayOrder = "day_month" | "month_day";
export const DayMonth: DayOrder = "day_month"; // d/m/yyyy, Australian
export const MonthDay: DayOrder = "month_day"; // m/d/yyyy

const numericDate = /^([0-9]{1,2})[/.-]([0-9]{1,2})[/.-]([0-9]{2}|[0-9]{4})$/;
const isoDate = /^([0-9]{4})-([0-9]{1,2})-([0-9]{1,2})$/;
const namedDate = /^([0-9]{1,2})[ -]([a-z]+)\.?,?[ -]([0-9]{2}|[0-9]{4})$/;
// Program files often prefix a weekday ("Mon 12/03/27") or suffix a time
// ("12/03/2027 8:00 AM"); neither changes the date.
const weekdayPrefix = /^(mon|tue|tues|wed|thu|thur|thurs|fri|sat|sun)[a-z]*\.?,?[\t\n\f\r ]+/;
const timeSuffix = /[\t\n\f\r ]+[0-9]{1,2}:[0-9]{2}.*$/;

const months: Record<string, number> = {
  jan: 1, january: 1, feb: 2, february: 2, mar: 3, march: 3,
  apr: 4, april: 4, may: 5, jun: 6, june: 6, jul: 7, july: 7,
  aug: 8, august: 8, sep: 9, sept: 9, september: 9, oct: 10,
  october: 10, nov: 11, november: 11, dec: 12, december: 12,
};

function cleanDate(s: string): string {
  return trim(s).toLowerCase().replace(weekdayPrefix, "").replace(timeSuffix, "");
}

/**
 * Picks one order for a whole column (plan 10.2 step 3): any first part over
 * 12 means day/month; otherwise any second part over 12 means month/day;
 * otherwise day/month, with assumed true when some value reads either way.
 */
export function decideDayOrder(values: readonly string[] | null): { order: DayOrder; assumed: boolean } {
  let firstOver = false;
  let secondOver = false;
  let ambiguous = false;
  for (const v of values ?? []) {
    const m = numericDate.exec(cleanDate(v));
    if (!m) continue;
    const a = +m[1];
    const b = +m[2];
    if (a > 12) firstOver = true;
    else if (b > 12) secondOver = true;
    else if (a !== b) ambiguous = true;
  }
  if (firstOver) return { order: DayMonth, assumed: false };
  if (secondOver) return { order: MonthDay, assumed: false };
  return { order: DayMonth, assumed: ambiguous };
}

const year = (s: string): number => (s.length === 2 ? 2000 + +s : +s);

const pad = (n: number, w: number): string => String(n).padStart(w, "0");

/**
 * Reads d/m/yyyy, d/m/yy, d Mon yyyy and yyyy-mm-dd and returns YYYY-MM-DD.
 * order applies to the numeric form only. Two-digit years are 2000–2099.
 */
export function parseDate(s: string, order: DayOrder): Parsed<string> {
  const t = cleanDate(s);
  let y: number;
  let mo: number;
  let d: number;
  let m: RegExpExecArray | null;
  if ((m = isoDate.exec(t))) {
    y = +m[1];
    mo = +m[2];
    d = +m[3];
  } else if ((m = numericDate.exec(t))) {
    d = +m[1];
    mo = +m[2];
    y = year(m[3]);
    if (order === MonthDay) [d, mo] = [mo, d];
  } else if ((m = namedDate.exec(t))) {
    const n = Object.hasOwn(months, m[2]) ? months[m[2]] : undefined;
    if (n === undefined) return fail(`${q(m[2])} is not a month.`);
    d = +m[1];
    mo = n;
    y = year(m[3]);
  } else {
    return fail(`${q(trim(s))} is not a date. Use d/m/yyyy, d Mon yyyy or yyyy-mm-dd.`);
  }
  // Go normalises 31 April to 1 May and refuses a date that moved.
  if (mo < 1 || mo > 12 || d < 1 || d > daysInMonth(y, mo)) return fail(`${q(trim(s))} is not a real date.`);
  return ok(`${pad(y, 4)}-${pad(mo, 2)}-${pad(d, 2)}`);
}
