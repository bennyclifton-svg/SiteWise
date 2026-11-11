// What an update changed on the profile, worked out in the browser from two
// saved profiles (docs/design/2026-11-10-run-time-feedback.md). The user's
// own edits and resets are never reported: they know what they did. Values,
// bands and source counts come from the server as they are; nothing here is
// judged or guessed.
import type { Profile, ProfileField, SystemRow } from "./profileApi";

/** One value: what it is, how it was decided, and how many passages back it. */
export interface FieldState { v: string; b: string; n: number }
/** One system: in scope, flagged to check, or ticked by the user. */
export interface ScopeState { in: boolean; check: boolean; user: boolean; entry: string }
export interface Snapshot { revision: number; fields: Record<string, FieldState>; scope: Record<string, ScopeState> }

export type FieldHow = "filled" | "changed" | "cleared" | "conflict" | "held" | "sources";
export type Change =
  | { kind: "field"; id: string; key: string; part: string; how: FieldHow; was: string; now: string; added: number }
  | { kind: "scope"; leaf: string; entry: string; how: "in" | "out" | "check" };

/** Every value the page shows from the profile: the card, details, site and compliance inputs. */
export function profileFields(p: Profile): ProfileField[] {
  return [...p.header, ...p.facts, ...p.details, ...p.site_details, ...p.compliance.flatMap((g) => g.rows)];
}

/** Evidence that the completed project will not have the system (as ScopeWorks reads it). */
const saysOut = (r: SystemRow) => r.presence.value === "not_included" && !["", "blank", "suggested"].includes(r.presence.band);

export function snapshotOf(p: Profile): Snapshot {
  const fields: Record<string, FieldState> = {};
  for (const f of profileFields(p)) fields[`${f.key}|${f.part_id}`] = { v: f.value, b: f.band, n: f.sources.length };
  const scope: Record<string, ScopeState> = {};
  for (const g of p.systems) {
    for (const r of g.rows) {
      scope[r.leaf] = { in: r.in_scope, check: r.presence.band === "red" || (r.in_scope && saysOut(r)), user: r.scope_origin === "user", entry: g.id };
    }
  }
  return { revision: p.revision ?? 0, fields, scope };
}

const EMPTY: FieldState = { v: "", b: "", n: 0 };

function fieldHow(b: FieldState, a: FieldState): FieldHow | null {
  // The user's word: a document speaking to it again is noted, nothing else.
  if (a.b === "user") return b.b === "user" && a.n > b.n ? "held" : null;
  if (b.b === "user") return null; // the user handed it back
  if (a.b === "red" && b.b !== "red") return "conflict";
  if (!b.v && a.v) return "filled";
  if (b.v && !a.v) return "cleared";
  if (b.v !== a.v) return "changed";
  if (a.n > b.n || a.b !== b.b) return "sources";
  return null;
}

export function diffSnapshots(before: Snapshot, after: Snapshot): Change[] {
  const out: Change[] = [];
  for (const [id, a] of Object.entries(after.fields)) {
    const b = before.fields[id] ?? EMPTY;
    const how = fieldHow(b, a);
    if (!how) continue;
    const at = id.lastIndexOf("|");
    out.push({ kind: "field", id, key: id.slice(0, at), part: id.slice(at + 1), how, was: b.v, now: a.v, added: Math.max(0, a.n - b.n) });
  }
  for (const [leaf, a] of Object.entries(after.scope)) {
    const b = before.scope[leaf] ?? { in: false, check: false, user: false, entry: a.entry };
    // A document flagging a ticked system is news even though the tick is the user's.
    if (a.check && !b.check) out.push({ kind: "scope", leaf, entry: a.entry, how: "check" });
    else if (!a.user && a.in !== b.in) out.push({ kind: "scope", leaf, entry: a.entry, how: a.in ? "in" : "out" });
  }
  return out;
}

export interface Receipt { filled: number; changed: number; check: number; held: number; added: number; removed: number }

export function receiptOf(changes: Change[]): Receipt {
  const r: Receipt = { filled: 0, changed: 0, check: 0, held: 0, added: 0, removed: 0 };
  for (const c of changes) {
    if (c.kind === "field") {
      if (c.how === "filled") r.filled++;
      else if (c.how === "changed" || c.how === "cleared") r.changed++;
      else if (c.how === "conflict") r.check++;
      else if (c.how === "held") r.held++;
    } else if (c.how === "in") r.added++;
    else if (c.how === "out") r.removed++;
    else r.check++;
  }
  return r;
}
