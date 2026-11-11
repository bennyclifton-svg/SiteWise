// One profile update as the browser follows it: each document's passes, the
// passages just read, and what the update changes elsewhere
// (docs/design/2026-11-10-run-time-feedback.md). It polls only while an
// update runs, shows only what the server measured, and never calls Jev.
import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "./api";

export interface ProgressDoc {
  document_id: string;
  filename: string;
  state: "queued" | "reading" | "failed";
  pass: "label" | "evidence";
  units: number;
  labelled: number;
  evidence: number;
}
export interface Progress { revision: number; preparing: number; documents: ProgressDoc[] }
export interface Find {
  passage_id: string;
  document_id: string;
  stage: "label" | "evidence";
  page: number;
  keys: string[];
  systems: string[];
  excerpt: string;
  at: string;
}
/** What the other surfaces now propose or flag: each is that surface's own count. */
export interface Next { procurement: number; costs: number; program: number; drafts: number }

async function get<T>(path: string): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`/api${path}`, { credentials: "same-origin" });
  } catch {
    throw new ApiError(0, "Can't reach SiteWise. Check your connection and try again.");
  }
  if (!res.ok) throw new ApiError(res.status, (await res.text()).trim() || res.statusText);
  return (await res.json()) as T;
}

export const runApi = {
  progress: (projectId: string) => get<Progress>(`/projects/${projectId}/profile/progress`),
  finds: (projectId: string, docs: string[], after: string) =>
    get<{ finds: Find[] }>(`/projects/${projectId}/profile/finds?${new URLSearchParams({ docs: docs.join(","), after })}`),
  next: (projectId: string) => get<Next>(`/projects/${projectId}/profile/next`),
};

/** A document as the strip draws it: a sheet hatched as labelled, inked as read for evidence. */
export interface RunDoc {
  id: string;
  filename: string;
  state: "queued" | "reading" | "failed" | "done";
  pass: "label" | "evidence";
  units: number;
  labelled: number;
  evidence: number;
  /** Passes committed: the label pass once evidence is the pass, both when done. */
  ticks: number;
}

const fromProgress = (d: ProgressDoc, prev?: RunDoc): RunDoc => ({
  id: d.document_id, filename: d.filename, state: d.state, pass: d.pass,
  units: d.units, labelled: d.labelled, evidence: d.evidence,
  ticks: Math.max(d.pass === "evidence" ? 1 : 0, prev?.ticks ?? 0),
});

const finished = (r: RunDoc): RunDoc => (r.state === "failed" || r.state === "done" ? r : { ...r, state: "done", ticks: 2, labelled: r.units, evidence: r.units });

/** Folds one progress read into the documents seen this run; one that left the list is done. */
export function mergeRun(prev: RunDoc[], docs: ProgressDoc[]): RunDoc[] {
  const now = new Map(docs.map((d) => [d.document_id, d]));
  const out = prev.map((r) => {
    const d = now.get(r.id);
    return d ? fromProgress(d, r) : finished(r);
  });
  for (const d of docs) if (!prev.some((r) => r.id === d.document_id)) out.push(fromProgress(d));
  return out;
}

/** What a passage spoke to: card and detail fields from the label pass, systems from both. */
export function findTargets(f: Pick<Find, "stage" | "keys" | "systems">): { fields: string[]; leaves: string[]; entries: string[] } {
  const fields: string[] = [], leaves: string[] = [], entries: string[] = [];
  const addLeaf = (leaf: string) => {
    if (!leaves.includes(leaf)) leaves.push(leaf);
    const entry = leaf.split(".")[0];
    if (!entries.includes(entry)) entries.push(entry);
  };
  for (const k of f.keys) {
    if (k.startsWith("sys.")) {
      const id = k.slice(4, k.lastIndexOf("."));
      if (f.stage === "evidence" && id.includes(".")) addLeaf(id);
    } else if (f.stage === "label" && /^(hdr|det|fact|proj|site)\./.test(k) && !fields.includes(k)) {
      fields.push(k);
    }
  }
  for (const s of f.systems) {
    if (s.includes(".")) addLeaf(s);
    else if (!entries.includes(s)) entries.push(s);
  }
  return { fields, leaves, entries };
}

export interface Run {
  /** idle: nothing followed yet; running; done: the last update finished. */
  phase: "idle" | "running" | "done";
  /** Pressed here: the elapsed time is the update's own, not since joining. */
  started: boolean;
  docs: RunDoc[];
  preparing: number;
  startedAt: number;
  finishedAt: number;
}

const IDLE: Run = { phase: "idle", started: false, docs: [], preparing: 0, startedAt: 0, finishedAt: 0 };
const POLL_MS = 1000;
const HIDDEN_POLL_MS = 5000;

/**
 * Follows a profile update. pending is the profile's own "documents still to
 * read"; onRevision asks the page to fetch the profile again; onFinds hands
 * over passages read since the last poll, each once.
 */
export function useProfileRun({ projectId, pending, revision, onRevision, onFinds, onSignedOut }: {
  projectId: string;
  pending: boolean;
  revision: number;
  onRevision: () => void;
  onFinds: (finds: Find[]) => void;
  onSignedOut: () => void;
}) {
  const [run, setRun] = useState<Run>(IDLE);
  const latest = useRef({ revision, pending, onRevision, onFinds, onSignedOut });
  latest.current = { revision, pending, onRevision, onFinds, onSignedOut };
  const seen = useRef(new Set<string>());
  const cursor = useRef("");
  // The revision last asked for, and when the profile was last asked again.
  const askedRevision = useRef(0);
  const askedAt = useRef(0);

  const begin = useCallback((started: boolean) => {
    seen.current.clear();
    cursor.current = "";
    // The missed-event check waits its 2 s from the start, as the old poll did.
    askedAt.current = Date.now();
    setRun({ phase: "running", started, docs: [], preparing: 0, startedAt: Date.now(), finishedAt: 0 });
  }, []);
  /** The user pressed Update and documents were queued. */
  const start = useCallback(() => begin(true), [begin]);
  const dismiss = useCallback(() => setRun((r) => (r.phase === "done" ? IDLE : r)), []);

  useEffect(() => setRun(IDLE), [projectId]);
  // Join an update already going: another tab, or a reload mid-run.
  useEffect(() => {
    if (!pending) return;
    setRun((r) => {
      if (r.phase === "running") return r;
      askedAt.current = Date.now();
      return { ...IDLE, phase: "running", startedAt: Date.now() };
    });
  }, [pending]);

  const running = run.phase === "running";
  useEffect(() => {
    if (!running) return;
    let live = true;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const poll = async () => {
      try {
        const p = await runApi.progress(projectId);
        if (!live) return;
        if (p.revision > latest.current.revision && askedRevision.current !== p.revision) {
          askedRevision.current = p.revision;
          latest.current.onRevision();
        }
        const reading = p.documents.filter((d) => d.state === "reading").map((d) => d.document_id).slice(0, 20);
        const quiet = p.preparing === 0 && p.documents.every((d) => d.state === "failed");
        setRun((r) => {
          if (r.phase !== "running") return r;
          const docs = mergeRun(r.docs, p.documents);
          // Done when nothing is read or waiting and the profile agrees.
          if (quiet && !latest.current.pending) return { ...r, docs: docs.map(finished), preparing: 0, phase: "done", finishedAt: Date.now() };
          return { ...r, docs, preparing: p.preparing };
        });
        // The profile still says pending though nothing is: a completion event
        // was missed, so read the profile again (at most every 2 s).
        if (quiet && latest.current.pending && Date.now() - askedAt.current > 2000) {
          askedAt.current = Date.now();
          latest.current.onRevision();
        }
        if (reading.length > 0) {
          const { finds } = await runApi.finds(projectId, reading, cursor.current);
          if (!live) return;
          if (finds.length > 0) cursor.current = new Date(Date.parse(finds[finds.length - 1].at) - 2000).toISOString();
          const fresh = finds.filter((f) => {
            const key = `${f.passage_id}|${f.stage}`;
            if (seen.current.has(key)) return false;
            seen.current.add(key);
            return true;
          });
          if (fresh.length > 0) latest.current.onFinds(fresh);
        }
      } catch (e) {
        // A dropped poll is retried; only a lost session stops the page.
        if (e instanceof ApiError && e.status === 401) return latest.current.onSignedOut();
      }
      if (live) timer = setTimeout(poll, document.hidden ? HIDDEN_POLL_MS : POLL_MS);
    };
    void poll();
    return () => {
      live = false;
      clearTimeout(timer);
    };
  }, [running, projectId]);

  return { run, start, dismiss };
}
