// Client for the project profile routes. Reads are precomputed on the server;
// a write returns the whole profile so the page stays consistent.
import { ApiError } from "./api";

export type ProfileBand = "green" | "amber" | "red" | "blank" | "suggested" | "user" | "unchecked" | "";

export interface ProfileSource {
  document_id: string;
  passage_id?: string;
  excerpt?: string;
  confidence?: number;
}

export interface Alternative {
  value: string;
  sources: ProfileSource[];
}

export interface Cell {
  value: string;
  band: ProfileBand;
  assertion?: string;
  tenders?: "consistent" | "differ" | "";
  note?: string;
  sources: ProfileSource[];
  alternatives: Alternative[];
  derived?: { rule: string; state: string; reason?: string };
}

export interface ProfileOption {
  id: string;
  label: string;
}

export interface ProfileField extends Cell {
  key: string;
  label: string;
  part_id: string;
  kind: "choice" | "multi" | "number" | "text";
  unit?: string;
  options?: ProfileOption[];
  stated_in?: string[];
  /** Compliance rows: relevant to the scope of works (or conflicting). */
  relevant?: boolean;
}

export interface SystemRow {
  leaf: string;
  label: string;
  part_id: string;
  presence: Cell;
  provider: Cell;
  note: Cell;
  shown_by_default: boolean;
  /** The works touch this system; origin is default, document or user. */
  in_scope: boolean;
  scope_origin?: "default" | "document" | "user";
  scope_note?: string;
}

/** A scope choice: in, out, or null to hand it back to defaults and documents. */
export type ScopeChoice = "in" | "out" | null;

export interface Preset {
  id: string;
  label: string;
  systems: string[];
}

export interface Part {
  id: string;
  label: string;
  kind: string;
  ncc_class: string;
}

export interface SourceCoverage {
 document_id: string; filename: string; pages: number; empty_pages: number[];
 current: boolean; units: number; labelled: number; evidence: number;
 needs_mapping: number; mapped: number; background: number;
}
export interface SourceRecord {
 id: string; document_id: string; filename: string; ordinal: number; text: string;
 page: number; location: string; section: string; context: string; category: string;
 provider: string; scope: string; outcome: string; confidence?: number;
 keys: string[]; unresolved: string[]; systems: string[];
}
export interface SourceRecords { records: SourceRecord[]; more: boolean }
export interface Profile {
 coverage?: SourceCoverage[];
  project_id: string;
  built_at: string | null;
  pending_documents: number;
  active_documents: number;
  /** Text split, not yet read by Jev; "Update project profile" reads them. */
  unread_documents: number;
  /** Documents the profile reads and does not read; skipped_kind is the most common kind not read. */
  read_documents: number;
  skipped_documents: number;
  skipped_kind: string;
  failed_documents: number;
  payment_required: boolean;
  /** Set only on an update request: documents it sent to Jev. */
  queued?: number;
  thresholds: { version: string; provisional: boolean; applied: boolean };
  parts: Part[];
  header: ProfileField[];
  facts: ProfileField[];
  systems: { id: string; label: string; rows: SystemRow[] }[];
  compliance: { group: string; rows: ProfileField[] }[];
  presets: Preset[];
}

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`/api${path}`, {
      method,
      credentials: "same-origin",
      headers: body === undefined ? undefined : { "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, "Can't reach SiteWise. Check your connection and try again.");
  }
  if (!res.ok) throw new ApiError(res.status, (await res.text()).trim() || res.statusText);
  return (await res.json()) as T;
}

export const profileApi = {
 sources: (projectId: string, system = "", outcome = "", offset = 0) => call<SourceRecords>("GET", `/projects/${projectId}/profile/sources?${new URLSearchParams({ system, outcome, offset: String(offset) })}`),
  get: (projectId: string) => call<Profile>("GET", `/projects/${projectId}/profile`),
  /** Queue Jev reading for unread documents; the profile fills in as each finishes. */
  read: (projectId: string) => call<Profile>("POST", `/projects/${projectId}/profile/read`),
  /** value null records "unknown" as the user's word; reset returns to the evidence. */
  set: (projectId: string, key: string, body: { part_id?: string; value?: string | null; note?: string; reset?: boolean }) =>
    call<Profile>("PUT", `/projects/${projectId}/profile/${encodeURIComponent(key)}`, body),
  /** Records scope choices in one write and returns the rebuilt profile. */
  setScope: (projectId: string, systems: Record<string, ScopeChoice>) =>
    call<Profile>("PUT", `/projects/${projectId}/profile/scope`, { systems }),
  addPart: (projectId: string, label: string, kind: string, ncc_class = "") =>
    call<Part>("POST", `/projects/${projectId}/parts`, { label, kind, ncc_class }),
  renamePart: (projectId: string, partId: string, label: string) =>
    call<Part>("PATCH", `/projects/${projectId}/parts/${partId}`, { label }),
};
