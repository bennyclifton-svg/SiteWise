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
}

export interface SystemRow {
  leaf: string;
  label: string;
  part_id: string;
  presence: Cell;
  provider: Cell;
  note: Cell;
  shown_by_default: boolean;
}

export interface Part {
  id: string;
  label: string;
  kind: string;
  ncc_class: string;
}

export interface Profile {
  project_id: string;
  built_at: string | null;
  pending_documents: number;
  thresholds: { version: string; provisional: boolean; applied: boolean };
  parts: Part[];
  header: ProfileField[];
  facts: ProfileField[];
  systems: { id: string; label: string; rows: SystemRow[] }[];
  compliance: { group: string; rows: ProfileField[] }[];
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
  get: (projectId: string) => call<Profile>("GET", `/projects/${projectId}/profile`),
  /** value null records "unknown" as the user's word; reset returns to the evidence. */
  set: (projectId: string, key: string, body: { part_id?: string; value?: string | null; note?: string; reset?: boolean }) =>
    call<Profile>("PUT", `/projects/${projectId}/profile/${encodeURIComponent(key)}`, body),
  addPart: (projectId: string, label: string, kind: string, ncc_class = "") =>
    call<Part>("POST", `/projects/${projectId}/parts`, { label, kind, ncc_class }),
  renamePart: (projectId: string, partId: string, label: string) =>
    call<Part>("PATCH", `/projects/${projectId}/parts/${partId}`, { label }),
};
