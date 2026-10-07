// Thin client for the Go API. Every call is same-origin with the session
// cookie; the browser adds the Origin header the server checks on mutations.

export type Band = "green" | "amber" | "blank" | "grey";
export type DecidedBy = "rule" | "jev" | "user";
export type Status = "pending" | "filed" | "not_filed" | "split";
/** Whether the project profile reads a document: automatic follows its kind. */
export type ReadSetting = "auto" | "read" | "skip";

export interface Field {
  field: string;
  value: string;
  band: Band;
  decided_by: DecidedBy;
  confidence?: number;
}

export interface Doc {
  id: string;
  project_id: string;
  filename: string;
  status: Status;
  text_pages?: number;
  text_empty_pages?: number;
  text_source_version?: string;
  text_status?: "queued" | "leased" | "done" | "failed";
  profile_read?: ReadSetting;
  reason?: string;
  number?: string;
  revision?: string;
  supersedes_id?: string;
  created_at: string;
  fields: Field[];
  source_id?: string;
  source_filename?: string;
  sheet_page?: number;
  sheet_total?: number;
  expansion?: { source_id: string; page_count: number; status: "pending" | "complete" | "review"; reason?: string };
}

export interface Project {
  id: string;
  name: string;
}

export interface DocumentList {
  cursor: number;
  project: Project;
  documents: Doc[];
}

export interface Option {
  id: string;
  label: string;
}

export interface Catalog {
  kinds: Option[];
  disciplines: Option[];
  lifecycle: Option[];
  /** Kinds the profile reads when a document's setting is automatic. */
  profile_read_kinds?: string[];
}

/** Event payload for filing, correction and not_filed. */
export interface DocEvent {
	project_id?: string;
  document_id: string;
  status: Status;
  reason?: string;
  fields?: Field[];
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message);
  }
}

export async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
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
  if (!res.ok) {
    const text = (await res.text()).trim();
    throw new ApiError(res.status, text || res.statusText);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

export const api = {
  checkSession: () => request<void>("GET", "/session"),
  signIn: (token: string) => request<void>("POST", "/session", { token }),
  projects: () => request<Project[]>("GET", "/projects"),
  createProject: (name: string) => request<{ id: string }>("POST", "/projects", { name }),
  documents: (projectId: string) => request<DocumentList>("GET", `/projects/${projectId}/documents`),
  catalog: () => request<Catalog>("GET", "/catalog"),
  correct: (docId: string, field: string, value: string) =>
    request<Doc>("PUT", `/documents/${docId}/fields/${field}`, { value }),
  retry: (docId: string, missingOnly = false) => request<Doc>("POST", `/documents/${docId}/${missingOnly ? "details/reprocess" : "filing"}`),
  document: (docId: string) => request<Doc>("GET", `/documents/${docId}`),
  /** Returns the rebuilt profile; the server applies a set's setting to its sheets. */
  setProfileReading: (projectId: string, ids: string[], setting: ReadSetting) =>
    request<unknown>("PUT", `/projects/${projectId}/documents/profile-read`, { document_ids: ids, setting }),
  /** Permanent. A drawing set takes its sheets; returns every id removed. */
  deleteDocuments: (projectId: string, ids: string[]) =>
    request<{ deleted: string[] }>("POST", `/projects/${projectId}/documents/delete`, { document_ids: ids }),
};

/** created is false when these bytes were already filed in the project. */
export interface Uploaded {
  doc: Doc;
  created: boolean;
}

/**
 * Upload streams the file as the raw body. XHR is used instead of fetch
 * because fetch has no upload progress events.
 */
export function upload(
  projectId: string,
  file: File,
  onProgress: (fraction: number) => void,
): { done: Promise<Uploaded>; abort: () => void } {
  const xhr = new XMLHttpRequest();
  const done = new Promise<Uploaded>((resolve, reject) => {
    xhr.open("POST", `/api/projects/${projectId}/files?name=${encodeURIComponent(file.name)}`);
    xhr.withCredentials = true;
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded / e.total);
    };
    xhr.onload = () => {
      if (xhr.status === 200 || xhr.status === 201) {
        resolve({ doc: JSON.parse(xhr.responseText) as Doc, created: xhr.status === 201 });
      } else {
        reject(new ApiError(xhr.status, xhr.responseText.trim() || "Upload failed"));
      }
    };
    xhr.onerror = () => reject(new ApiError(0, "Upload interrupted. Drop the file again; nothing is lost."));
    xhr.onabort = () => reject(new ApiError(0, "Upload cancelled"));
    xhr.send(file);
  });
  return { done, abort: () => xhr.abort() };
}

export const EVENT_KINDS = ["filing", "correction", "not_filed", "filing_failed", "sheets", "profile", "job", "ocr", "deleted", "report", "works", "packages", "delivery"] as const;
export type EventKind = (typeof EVENT_KINDS)[number];

export interface StreamEvent {
  id: number;
  kind: EventKind;
  document_id?: string;
  payload: DocEvent;
}
