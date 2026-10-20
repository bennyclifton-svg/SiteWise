// Readings and review as the Program view shows them (plan
// docs/plans/2026-10-18-program.md 7.4, 7.7, 10.2 to 10.4). Pure functions
// over the GET /program view: grouping a target's readings, the words the
// banner and toolbar use, and the evidence that lost its stage. Accepting,
// dismissing and deciding are server routes; nothing here changes a value.

import type { ProgramReading } from "../programApi";

/** One target's proposed values from one document: accepted or dismissed together. */
export interface ReadingGroup {
  key: string;
  /** null when the stage the evidence belongs to is gone (7.7). */
  targetId: string | null;
  catalogueStageId: string | null;
  ids: string[];
  documentId: string;
  documentTitle: string;
  importId: string;
  source: string;
  /** Parsed dates (YYYY-MM-DD) and the verbatim text they came from. */
  start: { date: string; raw: string } | null;
  finish: { date: string; raw: string } | null;
  duration: string | null;
  /** False when accepting changes nothing: the user's own value is kept. */
  applies: boolean;
  /** An awarded proposal's line: "From the awarded proposal: …". */
  text: string;
}

/**
 * Groups readings the way the server accepts them: by document, target and
 * catalogue stage, in the order they arrive. A start that did not parse
 * keeps its raw text and no date.
 */
export function groupReadings(readings: readonly ProgramReading[] | null | undefined): ReadingGroup[] {
  const out: ReadingGroup[] = [];
  const byKey = new Map<string, ReadingGroup>();
  for (const r of readings ?? []) {
    const key = `${r.document_id}|${r.target_activity_id ?? ""}|${r.catalogue_stage_id ?? ""}`;
    let g = byKey.get(key);
    if (!g) {
      g = {
        key,
        targetId: r.target_activity_id || null,
        catalogueStageId: r.catalogue_stage_id || null,
        ids: [],
        documentId: r.document_id,
        documentTitle: r.document_title,
        importId: r.import_id,
        source: r.source,
        start: null,
        finish: null,
        duration: null,
        applies: false,
        text: "",
      };
      byKey.set(key, g);
      out.push(g);
    }
    g.ids.push(r.id);
    g.applies ||= r.applies;
    if (r.text) g.text = r.text;
    if (r.field === "start") g.start = { date: r.parsed?.date ?? "", raw: r.value };
    else if (r.field === "finish") g.finish = { date: r.parsed?.date ?? "", raw: r.value };
    else if (r.field === "duration") g.duration = r.value;
  }
  return out;
}

/** "15 Mar 2027 – 26 Mar 2027", "from 15 Mar 2027", "6 weeks"; raw text where a date did not parse. */
export function proposedText(g: ReadingGroup, fmt: (iso: string) => string): string {
  const day = (d: { date: string; raw: string } | null) => (d ? (d.date ? fmt(d.date) : d.raw) : "");
  const s = day(g.start);
  const f = day(g.finish);
  if (s && f) return `${s} – ${f}`;
  if (s) return `from ${s}`;
  if (f) return `to ${f}`;
  return g.duration ?? "";
}

/** Readings that belong to a stage no longer in the program (plan 7.7). */
export function orphanEvidence(groups: readonly ReadingGroup[]): ReadingGroup[] {
  return groups.filter((g) => !g.targetId && !!g.catalogueStageId);
}

const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

/**
 * The review banner (plan 7.4): "Scope and procurement changed: 3 to add,
 * 1 to remove, 2 renamed." Without counts yet, the pending total.
 */
export function reviewWords(counts: Record<string, number> | null | undefined, pending: number): string {
  const parts: string[] = [];
  if (counts) {
    if (counts.add) parts.push(`${counts.add} to add`);
    if (counts.remove) parts.push(`${counts.remove} to remove`);
    if (counts.rename) parts.push(`${counts.rename} renamed`);
    if (counts.duration) parts.push(plural(counts.duration, "duration changed", "durations changed"));
    if (counts.link) parts.push(plural(counts.link, "link changed", "links changed"));
  }
  if (!parts.length) parts.push(plural(pending, "change", "changes"));
  return `Scope and procurement changed: ${parts.join(", ")}.`;
}

/** Why a file could not be read, in plain words. */
export function notReadWords(reason: string | null | undefined): string {
  switch (reason) {
    case "no_text_layer":
      return "The PDF has no text layer (a scan), so no rows could be read.";
    case "unreadable_text":
      return "The file is too large or damaged to read.";
    case "unsupported_format":
      return "Only PDF, DOCX and XLSX program files can be read.";
    case "no_program_rows":
      return "No program rows with dates were found in it.";
    case "not_filed":
      return "The file could not be filed.";
  }
  return "It could not be read.";
}

/** Program files the Program view takes: PDF, DOCX and XLSX (plan 10.2). */
export function importable(name: string): boolean {
  return /\.(pdf|docx|xlsx)$/i.test(name.trim());
}
