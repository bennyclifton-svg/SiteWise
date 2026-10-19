// Mirror of internal/program/calendar.go, rule for rule. Working-day indices
// map to calendar days; index 0 is the first working day on or after the
// start date. Days are integers (days since 1970-01-01, or offsets from the
// start date in method arguments). No JavaScript Date is used: dates are
// parsed and formatted with integer civil-calendar arithmetic, so no time
// zone can move a date (plan section 2, product rule 9).
//
// The calendar reduces to count(d): the number of working days in [0, d) for
// d ≥ 0, and minus the number in [d, 0) for d < 0. Every lookup is weekly
// arithmetic plus one binary search over the sorted exceptions.

import {
  type Calendar,
  type CalendarDay,
  DayPublicHoliday,
  DayRDO,
  DayShutdown,
  DayWorkingOverride,
  DefaultWorkingWeek,
} from "./types";

export function floorDiv(a: number, b: number): number {
  return Math.floor(a / b);
}

function isLeap(y: number): boolean {
  return y % 4 === 0 && (y % 100 !== 0 || y % 400 === 0);
}

const monthDays = [0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];

export function daysInMonth(y: number, m: number): number {
  return m === 2 && isLeap(y) ? 29 : monthDays[m];
}

/** Days since 1970-01-01 of a civil date (H. Hinnant's days_from_civil). */
export function daysFromCivil(y: number, m: number, d: number): number {
  const yy = m <= 2 ? y - 1 : y;
  const era = floorDiv(yy, 400);
  const yoe = yy - era * 400;
  const mp = m > 2 ? m - 3 : m + 9;
  const doy = Math.floor((153 * mp + 2) / 5) + d - 1;
  const doe = yoe * 365 + Math.floor(yoe / 4) - Math.floor(yoe / 100) + doy;
  return era * 146097 + doe - 719468;
}

const isoDay = /^([0-9]{4})-([0-9]{2})-([0-9]{2})$/;

/**
 * dayNumber parses a strict YYYY-MM-DD date, as Go's
 * time.Parse("2006-01-02") does: four-digit year, two-digit month 01–12 and a
 * two-digit day that exists in that month.
 */
export function dayNumber(s: string | undefined | null): number | null {
  if (typeof s !== "string" || s.length !== 10) return null;
  const m = isoDay.exec(s);
  if (!m) return null;
  const y = +m[1];
  const mo = +m[2];
  const d = +m[3];
  if (mo < 1 || mo > 12 || d < 1 || d > daysInMonth(y, mo)) return null;
  return daysFromCivil(y, mo, d);
}

const two: string[] = [];
for (let k = 0; k < 100; k++) two.push(k < 10 ? "0" + k : String(k));

function formatYMD(y: number, m: number, d: number): string {
  // Go writes the low four digits of the year; years outside 0–9999 never
  // reach here from a valid program.
  const yy = ((y % 10000) + 10000) % 10000;
  return two[Math.floor(yy / 100)] + two[yy % 100] + "-" + two[m] + "-" + two[d];
}

/** Formats a day number as YYYY-MM-DD (civil_from_days). */
export function dayString(n: number): string {
  const z = n + 719468;
  const era = floorDiv(z, 146097);
  const doe = z - era * 146097;
  const yoe = Math.floor((doe - Math.floor(doe / 1460) + Math.floor(doe / 36524) - Math.floor(doe / 146096)) / 365);
  let y = yoe + era * 400;
  const doy = doe - (365 * yoe + Math.floor(yoe / 4) - Math.floor(yoe / 100));
  const mp = Math.floor((5 * doy + 2) / 153);
  const day = doy - Math.floor((153 * mp + 2) / 5) + 1;
  const m = mp >= 10 ? mp - 9 : mp + 3;
  if (m <= 2) y++;
  return formatYMD(y, m, day);
}

/** Bitmask bit for an absolute day: Mon=1 … Sun=64 (1 Jan 1970 was a Thursday). */
export function weekdayBit(d: number): number {
  return 1 << (((d % 7) + 7 + 3) % 7);
}

/**
 * The Monday a relative program (no start date) anchors its working-day
 * indices to (plan 6.1, 13.1), as in Go. Its dates are never shown; they let
 * stored snet/mso constraint dates mean "working day N" on a relative program.
 */
export const RelativeEpoch = "2001-01-01";
const relativeOrigin = daysFromCivil(2001, 1, 1);

function validDayKind(k: string): boolean {
  return k === DayPublicHoliday || k === DayRDO || k === DayShutdown || k === DayWorkingOverride;
}

export class Cal {
  readonly relative: boolean;
  readonly origin: number;
  readonly perWeek: number;
  private readonly pre: Int32Array = new Int32Array(8);
  private readonly works: boolean[] = [false, false, false, false, false, false, false];
  private exDays: Int32Array = new Int32Array(0);
  private exSum: Int32Array = new Int32Array(1);
  private p0 = 0;
  private cacheLo = 0;
  private cache: string[] = [];

  /**
   * Tolerant like Go's NewCalendar: a malformed start date makes the program
   * relative and a malformed day is skipped, because Compute has no error path.
   */
  constructor(c: Calendar) {
    let mask = c.working_week ?? 0;
    if (mask <= 0 || mask > 127) mask = DefaultWorkingWeek;
    const start = dayNumber(c.start);
    this.relative = start === null;
    this.origin = start === null ? relativeOrigin : start;
    let perWeek = 0;
    for (let r = 0; r < 7; r++) {
      this.works[r] = (weekdayBit(this.origin + r) & mask) !== 0;
      this.pre[r + 1] = this.pre[r] + (this.works[r] ? 1 : 0);
      if (this.works[r]) perWeek++;
    }
    this.perWeek = perWeek;
    // A relative program has no dates, so holidays cannot apply to it.
    if (!this.relative) this.setExceptions(c.days ?? []);
  }

  // Keeps only days that change the pattern: a holiday on a working weekday
  // (−1) or an override on an idle weekday (+1). The last row for a day wins.
  private setExceptions(days: CalendarDay[]): void {
    const kinds = new Map<number, string>();
    for (const d of days) {
      const n = dayNumber(d.day);
      if (n === null || !validDayKind(d.kind)) continue;
      kinds.set(n - this.origin, d.kind);
    }
    const delta = new Map<number, number>();
    for (const [off, kind] of kinds) {
      const works = this.works[off - 7 * floorDiv(off, 7)];
      if (kind === DayWorkingOverride && !works) delta.set(off, 1);
      else if (kind !== DayWorkingOverride && works) delta.set(off, -1);
    }
    const offs = Int32Array.from(delta.keys()).sort();
    this.exDays = offs;
    this.exSum = new Int32Array(offs.length + 1);
    for (let k = 0; k < offs.length; k++) this.exSum[k + 1] = this.exSum[k] + (delta.get(offs[k]) as number);
    this.p0 = this.exBefore(0);
  }

  private exBefore(off: number): number {
    let lo = 0;
    let hi = this.exDays.length; // first k with exDays[k] >= off
    while (lo < hi) {
      const m = (lo + hi) >>> 1;
      if (this.exDays[m] < off) lo = m + 1;
      else hi = m;
    }
    return this.exSum[lo];
  }

  /** Signed number of working days in [0, off). */
  count(off: number): number {
    const q = floorDiv(off, 7);
    return q * this.perWeek + this.pre[off - 7 * q] + this.exBefore(off) - this.p0;
  }

  /** Offset of working day i: the smallest d with count(d+1) ≥ i+1. */
  dayOf(i: number): number {
    const g = floorDiv(i, this.perWeek) * 7;
    let lo = g - 7;
    let hi = g + 7;
    for (let step = 7; this.count(lo + 1) >= i + 1; step *= 2) lo -= step;
    for (let step = 7; this.count(hi + 1) < i + 1; step *= 2) hi += step;
    while (hi - lo > 1) {
      const m = lo + Math.floor((hi - lo) / 2);
      if (this.count(m + 1) >= i + 1) hi = m;
      else lo = m;
    }
    return hi;
  }

  // Boundary helpers for calendar-day lag (plan 6.2). An activity starts at
  // the beginning of its first day and finishes at the beginning of the day
  // after its last; a zero-length row finishes when it starts. Each "on or
  // before" helper is the exact inverse of its "on or after" pair.

  startDay(s: number): number {
    return this.dayOf(s);
  }

  finishDay(f: number, d: number): number {
    return d === 0 ? this.dayOf(f) : this.dayOf(f - 1) + 1;
  }

  startOnOrAfter(m: number): number {
    return this.count(m);
  }

  startOnOrBefore(m: number): number {
    return this.count(m + 1) - 1;
  }

  finishOnOrAfter(m: number, d: number): number {
    return d === 0 ? this.count(m) : this.count(m - 1) + 1;
  }

  finishOnOrBefore(m: number, d: number): number {
    return d === 0 ? this.count(m + 1) - 1 : this.count(m);
  }

  /** What "3w" multiplies by. */
  workingDaysPerWeek(): number {
    return this.perWeek;
  }

  /**
   * Date of working day i. On a relative program it counts from RelativeEpoch:
   * for storing a constraint, never for display.
   */
  dateOf(i: number): string {
    const k = i - this.cacheLo;
    if (k >= 0 && k < this.cache.length) return this.cache[k];
    return dayString(this.origin + this.dayOf(i));
  }

  /**
   * Precomputes the dates of indices lo..hi in one walk, so formatting a whole
   * program costs one pass over its span instead of a search per date. Spans
   * far longer than the rows they serve are skipped, as in Go.
   */
  warm(lo: number, hi: number, rows: number): void {
    if (this.relative || hi < lo || hi - lo > 8 * rows + 400) return;
    const n = hi - lo + 1;
    const out = new Array<string>(n);
    const ex = this.exDays;
    let k = 0;
    let d = this.dayOf(lo);
    while (k < ex.length && ex[k] < d) k++;
    // Walk y/m/d alongside the day number; converting each day from scratch
    // would cost more.
    const first = dayString(this.origin + d);
    let y = +first.slice(0, 4);
    let mo = +first.slice(5, 7);
    let day = +first.slice(8, 10);
    let dim = daysInMonth(y, mo);
    for (let i = 0; i < n; i++) {
      out[i] = formatYMD(y, mo, day);
      for (;;) {
        d++;
        if (++day > dim) {
          day = 1;
          if (++mo > 12) {
            mo = 1;
            y++;
          }
          dim = daysInMonth(y, mo);
        }
        while (k < ex.length && ex[k] < d) k++;
        let works = this.works[d - 7 * floorDiv(d, 7)];
        if (k < ex.length && ex[k] === d) works = !works;
        if (works) break;
      }
    }
    this.cache = out;
    this.cacheLo = lo;
  }

  /** Index of the first working day on or after date, or null (relative: from RelativeEpoch). */
  indexOnOrAfter(date: string | undefined | null): number | null {
    const d = dayNumber(date);
    if (d === null) return null;
    return this.count(d - this.origin);
  }

  /** Index of the last working day on or before date, or null. */
  indexOnOrBefore(date: string | undefined | null): number | null {
    const d = dayNumber(date);
    if (d === null) return null;
    return this.count(d - this.origin + 1) - 1;
  }

  /** Moves a start index by calendar days to the first working day on or after. */
  shiftCalendar(i: number, days: number): number {
    return this.startOnOrAfter(this.dayOf(i) + days);
  }

  /**
   * The one conversion from a half-open interval to shown dates (plan 6.1):
   * start = date(start); finish = date(start+duration−1) when duration > 0,
   * "" for a milestone. Both "" for a relative program.
   */
  display(start: number, duration: number): [string, string] {
    if (this.relative) return ["", ""];
    if (duration <= 0) return [this.dateOf(start), ""];
    return [this.dateOf(start), this.dateOf(start + duration - 1)];
  }

  /** "Week 1" for the first seven calendar days from the start. */
  weekLabel(i: number): string {
    return "Week " + (floorDiv(this.dayOf(i), 7) + 1);
  }

  /** "Week 2, day 3": WeekLabel plus the working day within that week. */
  dayLabel(i: number): string {
    const off = this.dayOf(i);
    const week = floorDiv(off, 7);
    const n = this.count(off + 1) - this.count(week * 7);
    return `Week ${week + 1}, day ${n}`;
  }
}

export function newCalendar(c: Calendar): Cal {
  return new Cal(c);
}
