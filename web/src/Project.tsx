import { PageActions } from "./PageActions";
import { specificationApi } from "./specificationApi";
import { emptySelection, selectAll as selectDocumentRows, type Selection } from "./table/selection";
import { Activity, Pending } from "./Activity";
import { readLocationChoice, writeLocationChoice } from "./workbenchLocation";
// A chosen project: drop files anywhere on the page, watch each become a title
// block, correct any box. Live updates come from the durable event stream and
// never move focus.

import { useCallback, useEffect, useMemo, useReducer, useRef, useState, type DragEvent, type ReactNode } from "react";
import {
  api,
  request,
  ApiError,
  EVENT_KINDS,
  upload,
  type Catalog,
  type Doc,
  type DocumentList,
  type ReadSetting,
  type StreamEvent,
} from "./api";
import { type RowModel } from "./DocumentRow";
import { CockpitFrame } from "./CockpitFrame";
import { LeftNav, viewLabel, type WorkbenchView } from "./LeftNav";
import { Profile, type ProfileRunState } from "./Profile";
import { Register } from "./Register";
import { DocumentWorkspace } from "./DocumentWorkspace";
import { Procurement } from "./Procurement";
import { Costs } from "./Costs";
import { Program } from "./Program";
import { newerAnnouncement, type RemoteProgram } from "./programApi";
import { SpecificationSchedule as Specifications } from "./SpecificationSchedule";
import { ocrStage } from "./OCRStatus";

const STALE_MS = 8000;
const UPLOAD_CONCURRENCY = 4;
const ACCEPT = ".pdf,.docx,.xlsx,.md,.markdown";

interface Upload {
  key: string;
  filename: string;
  progress: number;
  queued?: boolean;
  error?: string;
  docId?: string;
}

interface State {
  phase: "loading" | "ready" | "missing" | "error";
  error?: string;
  projectName: string;
  docs: Record<string, Doc>;
  /** Row keys, newest first. A key is a document id or an upload key. */
  order: string[];
  uploads: Record<string, Upload>;
  /** When each document's bytes finished uploading in this tab (performance clock). */
  sentAt: Record<string, number>;
  /** Wall-clock time each pending filing was started from this tab. */
  pendingSince: Record<string, number>;
  filedIn: Record<string, number>;
  duplicate: Record<string, boolean>;
  interrupted: Record<string, boolean>;
  landed: Record<string, boolean>;
  /** Events that arrived before their upload's response: a fast filing can
   *  beat the HTTP reply. Applied when the document joins the list. */
  early: Record<string, StreamEvent>;
  loadedAt: number;
}

type Action =
  | { type: "renamed"; name: string }
  | { type: "loaded"; list: DocumentList }
  | { type: "failed"; missing: boolean; error: string }
  | { type: "uploadStart"; items: { key: string; filename: string }[] }
  | { type: "uploadProgress"; key: string; progress: number }
  | { type: "uploadDone"; key: string; doc: Doc; created: boolean; at: number }
  | { type: "uploadFailed"; key: string; error: string }
  | { type: "dismiss"; key: string }
  | { type: "event"; ev: StreamEvent; at: number }
  | { type: "doc"; doc: Doc }
  | { type: "externalDoc"; doc: Doc }
  | { type: "retrying"; id: string }
  | { type: "unland"; id: string }
  | { type: "reading"; ids: string[]; setting: ReadSetting }
  | { type: "removed"; ids: string[] };

const initial: State = {
  phase: "loading",
  projectName: "",
  docs: {},
  order: [],
  uploads: {},
  sentAt: {},
  pendingSince: {},
  filedIn: {},
  duplicate: {},
  interrupted: {},
  landed: {},
  early: {},
  loadedAt: 0,
};

const MAX_EARLY = 200;

function without<T>(rec: Record<string, T>, key: string): Record<string, T> {
  const next = { ...rec };
  delete next[key];
  return next;
}

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case "renamed": return { ...state, projectName: action.name };
    case "loaded": {
      const docs: Record<string, Doc> = {};
      for (const d of action.list.documents) docs[d.id] = d;
      return {
        ...state,
        phase: "ready",
        projectName: action.list.project.name,
        docs,
        order: action.list.documents.map((d) => d.id),
        loadedAt: Date.now(),
      };
    }
    case "failed":
      return { ...state, phase: action.missing ? "missing" : "error", error: action.error };
    case "uploadStart": {
      // A batch goes on top in the order it was chosen.
      const uploads = { ...state.uploads };
      for (const it of action.items) uploads[it.key] = { key: it.key, filename: it.filename, progress: 0, queued: true };
      return { ...state, uploads, order: [...action.items.map((it) => it.key), ...state.order] };
    }
    case "uploadProgress": {
      const up = state.uploads[action.key];
      if (!up) return state;
      return { ...state, uploads: { ...state.uploads, [action.key]: { ...up, progress: action.progress, queued: false } } };
    }
    case "uploadDone": {
      const { doc, key } = action;
      // An event may have landed before the upload response; keep the newer status.
      const known = state.docs[doc.id];
      const merged = known && known.status !== "pending" ? known : doc;
      const order = state.order.filter((k) => k !== doc.id).map((k) => (k === key ? doc.id : k));
      const next: State = {
        ...state,
        docs: { ...state.docs, [doc.id]: merged },
        order,
        uploads: without(state.uploads, key),
        sentAt: doc.status === "pending" ? { ...state.sentAt, [doc.id]: action.at } : state.sentAt,
        pendingSince: doc.status === "pending" ? { ...state.pendingSince, [doc.id]: Date.now() } : state.pendingSince,
        duplicate: action.created ? without(state.duplicate, doc.id) : { ...state.duplicate, [doc.id]: true },
        early: without(state.early, doc.id),
      };
      const early = state.early[doc.id];
      return early ? reducer(next, { type: "event", ev: early, at: action.at }) : next;
    }
    case "uploadFailed": {
      const up = state.uploads[action.key];
      if (!up) return state;
      return { ...state, uploads: { ...state.uploads, [action.key]: { ...up, error: action.error } } };
    }
    case "dismiss":
      return { ...state, uploads: without(state.uploads, action.key), order: state.order.filter((k) => k !== action.key) };
    case "event": {
      const { ev } = action;
      const id = ev.payload?.document_id ?? ev.document_id;
      if (!id) return state;
      const doc = state.docs[id];
      if (!doc) {
        // Not listed yet: keep the newest event for when its upload returns.
        // Events for other projects share the org stream, so the store is capped.
        if (!(id in state.early) && Object.keys(state.early).length >= MAX_EARLY) return state;
        return { ...state, early: { ...state.early, [id]: ev } };
      }
      if (ev.kind === "filing_failed") {
        return doc.status === "pending" ? { ...state, interrupted: { ...state.interrupted, [id]: true } } : state;
      }
      const p = ev.payload;
      const next: Doc = {
        ...doc,
        status: p.status ?? doc.status,
        reason: p.reason ?? doc.reason,
        fields: p.fields ?? doc.fields,
      };
      const landing = doc.status === "pending" && next.status !== "pending";
      const sent = state.sentAt[id];
      return {
        ...state,
        docs: { ...state.docs, [id]: next },
        interrupted: landing ? without(state.interrupted, id) : state.interrupted,
        landed: landing ? { ...state.landed, [id]: true } : state.landed,
        filedIn: landing && sent !== undefined ? { ...state.filedIn, [id]: action.at - sent } : state.filedIn,
      };
    }
    case "externalDoc": {
      if (state.docs[action.doc.id]) return state;
      const early = state.early[action.doc.id];
      const next = { ...state, docs: { ...state.docs, [action.doc.id]: action.doc },
        order: [action.doc.id, ...state.order], early: without(state.early, action.doc.id) };
      return early ? reducer(next, { type: "event", ev: early, at: performance.now() }) : next;
    }
    case "doc":
      return { ...state, docs: { ...state.docs, [action.doc.id]: action.doc } };
    case "retrying":
      return {
        ...state,
        interrupted: without(state.interrupted, action.id),
        sentAt: { ...state.sentAt, [action.id]: performance.now() },
        pendingSince: { ...state.pendingSince, [action.id]: Date.now() },
      };
    case "unland":
      return { ...state, landed: without(state.landed, action.id) };
    case "removed": {
      const gone = new Set(action.ids);
      const docs = { ...state.docs };
      for (const id of gone) delete docs[id];
      return { ...state, docs, order: state.order.filter((k) => !gone.has(k)) };
    }
    case "reading": {
      // The server applies a drawing set's setting to its sheets too.
      const ids = new Set(action.ids);
      const docs = { ...state.docs };
      for (const d of Object.values(state.docs)) {
        if (ids.has(d.id) || (d.source_id && ids.has(d.source_id))) docs[d.id] = { ...d, profile_read: action.setting };
      }
      return { ...state, docs };
    }
  }
}

interface Props {
  onRenamed: (id: string, name: string) => void;
  projectId: string;
  catalog: Catalog | null;
  projects: { id: string; name: string }[] | null;
  navigate: (path: string) => void;
  onSignedOut: () => void;
  onHome: () => void;
}

export function Project({ projectId, catalog, projects, navigate, onSignedOut, onHome, onRenamed }: Props) {
  const [state, dispatch] = useReducer(reducer, initial);
  const [cursor, setCursor] = useState<number | null>(null);
  const [live, setLive] = useState<"live" | "reconnecting" | "offline">("reconnecting");
  const [over, setOver] = useState(false);
  const [announcement, setAnnouncement] = useState("");
  const [flash, setFlash] = useState<string | null>(null);
  const [openDocument, setOpenDocument] = useState<string | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const [profileTick, setProfileTick] = useState(0);
  // The profile's update as the nav and tab title show it; views opened since it finished.
  const [profileRun, setProfileRun] = useState<ProfileRunState | null>(null);
  const [visitedSinceRun, setVisitedSinceRun] = useState<Partial<Record<WorkbenchView, boolean>>>({});
  const [profileFeedbackTarget, setProfileFeedbackTarget] = useState<HTMLDivElement | null>(null);
  const [initialView] = useState(() => readLocationChoice<WorkbenchView>("view", ["profile", "costs", "procurement", "specifications", "program"], "profile"));
  const [repositoryOpenRequest, setRepositoryOpenRequest] = useState(0);
  const [documentSelection, setDocumentSelection] = useState<Selection>(emptySelection);
  const [showReports, setShowReports] = useState(initialView === "procurement");
  const [showCosts, setShowCosts] = useState(initialView === "costs");
  const [showSpecifications, setShowSpecifications] = useState(initialView === "specifications");
  const [specificationsOpened, setSpecificationsOpened] = useState(initialView === "specifications");
  const [specificationTick, setSpecificationTick] = useState(0);
  const [invoiceTick, setInvoiceTick] = useState(0);
  const [costsOpened, setCostsOpened] = useState(initialView === "costs");
  const [reportsOpened, setReportsOpened] = useState(initialView === "procurement");
  const [showProgram, setShowProgram] = useState(initialView === "program");
  const [programOpened, setProgramOpened] = useState(initialView === "program");
  // Newest program version announced on the stream (plan 8.3), with the
  // program it belongs to: a recreated program starts its versions again.
  const [programVersion, setProgramVersion] = useState<RemoteProgram>({ programId: "", version: 0 });
  // Bumped when the program's review or readings change (no new version).
  const [programTick, setProgramTick] = useState(0);
	const [reportTick, setReportTick] = useState(0);
  const [notReadOnly, setNotReadOnly] = useState(false);
  const [address, setAddress] = useState("");
  const lastId = useRef(0);
  const fileInput = useRef<HTMLInputElement>(null);
  const queue = useRef<{ key: string; file: File }[]>([]);
  const active = useRef(0);
  const dragDepth = useRef(0);
  const docsRef = useRef(state.docs);
  docsRef.current = state.docs;

  // SSE is the fast path. A small list refresh recovers a lost completion
  // event without downloading images, restarting jobs or spending AI calls.
  useEffect(() => {
    let stopped = false;
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      const before = docsRef.current;
      const waiting = Object.values(before).some(doc => doc.status === "filed" && /\.pdf$/i.test(doc.filename)
        && (!doc.visual_status || doc.visual_status === "queued" || doc.visual_status === "reading")
        && !doc.fields.some(f => f.field === "title" && (f.value || f.decided_by === "user")));
      if (waiting) {
        try {
          const list = await api.documents(projectId);
          if (!stopped) for (const doc of list.documents) {
            // Do not overwrite a correction or deletion that overtook this read.
            if (before[doc.id] && docsRef.current[doc.id] === before[doc.id]) dispatch({type: "doc", doc});
          }
        } catch { /* Retry transient failures on the next tick. */ }
      }
      if (!stopped) timer = setTimeout(poll, 3000);
    };
    timer = setTimeout(poll, 1000);
    return () => { stopped = true; clearTimeout(timer); };
  }, [projectId]);

  // Load the list, then stream from the cursor read before it.
  useEffect(() => {
    let cancelled = false;
    api
      .documents(projectId)
      .then((list) => {
        if (cancelled) return;
        lastId.current = list.cursor;
        dispatch({ type: "loaded", list });
        setCursor(list.cursor);
      })
      .catch((e: unknown) => {
        if (cancelled) return;
        if (e instanceof ApiError && e.status === 401) return onSignedOut();
        const missing = e instanceof ApiError && e.status === 404;
        dispatch({ type: "failed", missing, error: e instanceof Error ? e.message : "Couldn't load this project." });
      });
    return () => {
      cancelled = true;
    };
  }, [projectId, onSignedOut]);

  useEffect(() => {
    if (cursor === null) return;
    let es: EventSource | null = null;
    let retry: ReturnType<typeof setTimeout> | undefined;
    let stopped = false;
    const open = () => {
      if (stopped) return;
      es = new EventSource(`/api/events?after=${lastId.current}`);
      es.onopen = () => setLive("live");
      es.onerror = () => {
        if (!es || es.readyState !== EventSource.CLOSED) {
          setLive("reconnecting");
          return;
        }
        setLive("offline");
        api.checkSession().then(
          () => {
            retry = setTimeout(open, 3000);
          },
          (e: unknown) => {
            if (e instanceof ApiError && e.status === 401) onSignedOut();
            else retry = setTimeout(open, 3000);
          },
        );
      };
      for (const kind of EVENT_KINDS) {
        es.addEventListener(kind, (msg) => {
          const ev = JSON.parse((msg as MessageEvent<string>).data) as StreamEvent;
          if (ev.id <= lastId.current) return;
          lastId.current = ev.id;
          if (ev.kind === "invoices") {
            if (ev.payload?.project_id === projectId) setInvoiceTick(n => n + 1);
            return;
          }
          if (ev.kind === "program.review" || ev.kind === "program.readings") {
            const p = ev.payload as unknown as { project_id?: string } | undefined;
            if (p?.project_id === projectId) setProgramTick((n) => n + 1);
            return;
          }
          if (ev.kind === "program.changed") {
            const p = ev.payload as unknown as { project_id?: string; program_id?: string; version?: number } | undefined;
            if (p?.project_id === projectId && typeof p.version === "number") setProgramVersion((v) => newerAnnouncement(v, p.program_id ?? "", p.version as number));
            return;
          }
          if (ev.kind === "specification") {
            if (ev.payload?.project_id === projectId) setSpecificationTick(n => n + 1);
            return;
          }
		  if (ev.kind === "report" || ev.kind === "works" || ev.kind === "packages" || ev.kind === "delivery" || ev.kind === "costs" || ev.kind === "procurement") {
		    if (ev.payload?.project_id === projectId) setReportTick(n => n + 1);
		    return;
		  }
          if (ev.kind === "visual") {
            if (ev.document_id) api.document(ev.document_id).then(doc => {
              if (!stopped && doc.project_id === projectId) dispatch({type: "doc", doc});
            }).catch(() => { /* A deletion can overtake the reading. */ });
            return;
          }
          if (ev.kind === "sheets") {
            api.documents(projectId).then((list) => {
              if (!stopped) dispatch({ type: "loaded", list });
            }).catch(() => setAnnouncement("Could not refresh drawing sheets. Reload this project."));
            return;
          }
          if (ev.kind === "deleted") {
            // Another tab or person deleted it; this tab's own delete already removed it.
            if (ev.document_id) dispatch({ type: "removed", ids: [ev.document_id] });
            return;
          }
          if (ev.kind === "profile" || ev.kind === "job") {
            setProfileTick((t) => t + 1);
            if (ev.kind === "job") {
              api.documents(projectId).then((list) => {
                if (!stopped) dispatch({ type: "loaded", list });
              }).catch(() => setAnnouncement("Could not refresh document preparation status. Reload this project."));
            }
            return;
          }
          dispatch({ type: "event", ev, at: performance.now() });
          const documentId = ev.payload?.document_id ?? ev.document_id;
          const doc = docsRef.current[documentId ?? ""];
          if (!doc) {
            // Procurement and other tabs upload through the same repository.
            // Their events arrive without this component's upload response.
            if (documentId && (!ev.payload?.project_id || ev.payload.project_id === projectId)) api.document(documentId).then(d => {
              if (!stopped && d.project_id === projectId) dispatch({ type: "externalDoc", doc: d });
            }).catch(() => { /* A later deletion may already have removed it. */ });
            return;
          }
          if (ev.kind === "filing") setAnnouncement(`${doc.filename} filed.`);
          if (ev.kind === "not_filed") setAnnouncement(`${doc.filename} stored but not filed.`);
          if (ev.kind === "filing_failed") setAnnouncement(`Filing stopped for ${doc.filename}. Retry is available.`);
        });
      }
    };
    open();
    return () => {
      stopped = true;
      es?.close();
      clearTimeout(retry);
    };
  }, [cursor, onSignedOut]);

  // Clear the ink-in flag once its animation has played.
  useEffect(() => {
    const ids = Object.keys(state.landed);
    if (ids.length === 0) return;
    const t = setTimeout(() => ids.forEach((id) => dispatch({ type: "unland", id })), 900);
    return () => clearTimeout(t);
  }, [state.landed]);

  const anyPending = Object.values(state.docs).some((d) => d.status === "pending");

  // Poll every active OCR pass, including initial filing and Retry OCR, so
  // a missed completion event cannot strand a progress indicator.
  const recovering = Object.values(state.docs).filter(d => ocrStage(d)).map(d => d.id).sort().join(",");
  useEffect(() => {
    if (!recovering) return;
    let live = true;
    let timer: ReturnType<typeof setTimeout>;
    const refresh = async () => {
      await Promise.all(recovering.split(",").map(async id => {
        try { const doc = await api.document(id); if (live) dispatch({ type: "doc", doc }); }
        catch (e) { if (live && e instanceof ApiError && e.status === 401) onSignedOut(); }
      }));
      if (live) timer = setTimeout(refresh, 2000);
    };
    timer = setTimeout(refresh, 2000);
    return () => { live = false; clearTimeout(timer); };
  }, [recovering, onSignedOut]);
  useEffect(() => {
    if (!anyPending) return;
    const t = setInterval(() => setNow(Date.now()), 2000);
    return () => clearInterval(t);
  }, [anyPending]);

  const pump = useCallback(() => {
    while (active.current < UPLOAD_CONCURRENCY && queue.current.length > 0) {
      const { key, file } = queue.current.shift()!;
      active.current++;
      dispatch({ type: "uploadProgress", key, progress: 0 });
      const job = upload(projectId, file, (progress) => dispatch({ type: "uploadProgress", key, progress }));
      job.done
        .then(({ doc, created }) => {
          dispatch({ type: "uploadDone", key, doc, created, at: performance.now() });
          if (!created) setAnnouncement(`${file.name} is already in this project.`);
        })
        .catch((e: unknown) => {
          if (e instanceof ApiError && e.status === 401) onSignedOut();
          dispatch({ type: "uploadFailed", key, error: e instanceof Error ? e.message : "Upload failed." });
        })
        .finally(() => {
          active.current--;
          pump();
        });
    }
  }, [projectId, onSignedOut]);

  const addFiles = useCallback(
    (files: FileList | File[]) => {
      const list = Array.from(files);
      if (list.length === 0) return;
      const items = list.map((file) => ({ key: `upload-${crypto.randomUUID()}`, filename: file.name, file }));
      dispatch({ type: "uploadStart", items });
      for (const { key, file } of items) queue.current.push({ key, file });
      setAnnouncement(list.length === 1 ? `Uploading ${list[0].name}.` : `Uploading ${list.length} files.`);
      pump();
    },
    [pump],
  );

  // The page accepts a drop once a project is chosen. Only the repository highlights.
  function onDragEnter(e: DragEvent) {
    if (!e.dataTransfer.types.includes("Files")) return;
    e.preventDefault();
    dragDepth.current++;
    setOver(true);
  }
  function onDragOver(e: DragEvent) {
    if (!e.dataTransfer.types.includes("Files")) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "copy";
  }
  function onDragLeave() {
    dragDepth.current = Math.max(0, dragDepth.current - 1);
    if (dragDepth.current === 0) setOver(false);
  }
  function onDrop(e: DragEvent) {
    e.preventDefault();
    dragDepth.current = 0;
    setOver(false);
    addFiles(e.dataTransfer.files);
  }

  const correct = useCallback(
    async (docId: string, field: string, value: string) => {
      try {
        const doc = await api.correct(docId, field, value);
        dispatch({ type: "doc", doc });
        setAnnouncement(`Saved ${field}.`);
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) onSignedOut();
        throw e;
      }
    },
    [onSignedOut],
  );

  const retry = useCallback(
    (docId: string, missingOnly = false) => {
      dispatch({ type: "retrying", id: docId });
      return api.retry(docId, missingOnly).then(
        (doc) => { dispatch({ type: "doc", doc }); return true; },
        (e: unknown) => {
          if (e instanceof ApiError && e.status === 401) onSignedOut();
          setAnnouncement("Retry didn't start. Check your connection and try again.");
          return false;
        },
      );
    },
    [onSignedOut],
  );

  const split = useCallback(
    (docId: string) =>
      api.split(docId).then(
        (doc) => { dispatch({ type: "doc", doc }); setAnnouncement("Splitting into pages. Each page is listed as it is read."); return true; },
        (e: unknown) => {
          if (e instanceof ApiError && e.status === 401) onSignedOut();
          setAnnouncement(e instanceof ApiError && e.status === 409 ? "This document can't be split into pages." : "The split didn't start. Check your connection and try again.");
          return false;
        },
      ),
    [onSignedOut],
  );

  const setReading = useCallback(
    async (ids: string[], setting: ReadSetting) => {
      try {
        await api.setProfileReading(projectId, ids, setting);
        dispatch({ type: "reading", ids, setting });
        setProfileTick((t) => t + 1);
        const n = ids.length === 1 ? "1 document" : `${ids.length} documents`;
        setAnnouncement(setting === "read" ? `${n} will be read for the profile.` : setting === "skip" ? `${n} will not be read for the profile.` : `${n} reset to automatic.`);
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) onSignedOut();
        setAnnouncement("The reading setting wasn't saved. Check your connection and try again.");
      }
    },
    [projectId, onSignedOut],
  );

  const remove = useCallback(
    async (ids: string[]) => {
      try {
        const { deleted } = await api.deleteDocuments(projectId, ids);
        dispatch({ type: "removed", ids: deleted });
        setProfileTick((t) => t + 1);
        setAnnouncement(deleted.length === 1 ? "1 document deleted." : `${deleted.length} documents deleted.`);
        return true;
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) onSignedOut();
        setAnnouncement(e instanceof ApiError && e.status === 404 ? "Nothing was deleted: a document was already gone. The list has been refreshed." : "Nothing was deleted. Check your connection and try again.");
        if (e instanceof ApiError && e.status === 404) api.documents(projectId).then((list) => dispatch({ type: "loaded", list }), () => undefined);
        return false;
      }
    },
    [projectId, onSignedOut],
  );

  const jump = useCallback((docId: string) => {
    setOpenDocument(docId);
    const el = document.getElementById(`doc-${docId}`);
    if (!el) return;
    el.scrollIntoView({ block: "center", behavior: "smooth" });
    el.focus({ preventScroll: true });
    setFlash(docId);
    setTimeout(() => setFlash((f) => (f === docId ? null : f)), 1200);
  }, []);

  const priorLabel = useCallback(
    (id: string) => {
      const d = state.docs[id];
      if (!d) return undefined;
      const number = d.fields.find((f) => f.field === "number")?.value || d.number;
      const rev = d.fields.find((f) => f.field === "revision")?.value || d.revision;
      return [number, rev].filter(Boolean).join(" · ") || d.filename;
    },
    [state.docs],
  );

  const rows: RowModel[] = useMemo(
    () =>
      state.order.flatMap((key): RowModel[] => {
        const up = state.uploads[key];
        if (up) return [{ key, filename: up.filename, progress: up.progress, uploadQueued: up.queued, uploadError: up.error }];
        const doc = state.docs[key];
        if (!doc) return [];
        const since = state.pendingSince[key] ?? state.loadedAt;
        return [
          {
            key,
            filename: doc.filename,
            doc,
            duplicate: state.duplicate[key],
            interrupted: state.interrupted[key],
            stale: doc.status === "pending" && now - since > STALE_MS,
            filedInMs: state.filedIn[key],
            landed: state.landed[key],
            flash: flash === key,
          },
        ];
      }),
    [state, now, flash],
  );

  const viewingRow = rows.find(row => row.key === openDocument);

  const view: WorkbenchView = showProgram
    ? "program"
    : showSpecifications
    ? "specifications"
    : showCosts
      ? "costs"
      : showReports
        ? "procurement"
        : "profile";

  const showView = (next: WorkbenchView) => {
    setVisitedSinceRun((seen) => ({ ...seen, [next]: true }));
    writeLocationChoice("view", next);
    setOpenDocument(null);
    setShowProgram(next === "program");
    if (next === "program") {
      setProgramOpened(true);
      setShowSpecifications(false);
      setShowCosts(false);
      setShowReports(false);
      return;
    }
    if (next === "specifications") {
      setSpecificationsOpened(true);
      setShowSpecifications(true);
      setShowCosts(false);
      setShowReports(false);
      return;
    }
    setShowSpecifications(false);
    if (next === "procurement") {
      setShowCosts(false);
      setReportsOpened(true);
      setShowReports(true);
      return;
    }
    if (next === "costs") {
      setCostsOpened(true);
      setShowCosts(true);
      setShowReports(false);
      return;
    }
    setShowCosts(false);
    setShowReports(false);
  };

  // The profile's update on the nav (docs/design/2026-11-10-run-time-feedback.md):
  // a ring on Project Profile while it reads and you are elsewhere, then its
  // count of changes; and on the views it affects, their own counts until opened.
  const runNext = profileRun?.next ?? null;
  useEffect(() => setVisitedSinceRun({}), [runNext]);
  const navMarks: Partial<Record<WorkbenchView, ReactNode>> = {};
  const count = (n: number, words: string) => <><span className="nav-count" title={`${n} ${words}`} aria-hidden="true">{n}</span><span className="sr-only">{n} {words}</span></>;
  if (profileRun && view !== "profile") {
    if (profileRun.reading) navMarks.profile = <span title="Reading documents for the profile"><Activity /><span className="sr-only">Reading documents for the profile</span></span>;
    else if (profileRun.changes > 0) navMarks.profile = count(profileRun.changes, "changes marked on the profile");
  }
  if (runNext) {
    const mark = (v: WorkbenchView, n: number, words: string) => {
      if (n > 0 && view !== v && !visitedSinceRun[v]) navMarks[v] = count(n, words);
    };
    mark("procurement", runNext.procurement + runNext.drafts, "proposed since the profile update");
    mark("costs", runNext.costs, "cost lines proposed since the profile update");
    mark("program", runNext.program, "program changes waiting for review");
  }
  // While the profile reads, the tab says so, for a user working in another tab.
  const reading = !!profileRun?.reading, readDocs = profileRun?.read ?? 0, runDocs = profileRun?.total ?? 0;
  useEffect(() => {
    if (!reading || !state.projectName) return;
    const before = document.title;
    document.title = `Reading ${runDocs > 0 ? `${Math.min(readDocs + 1, runDocs)} of ${runDocs}` : "documents"} · ${state.projectName}`;
    return () => { document.title = before; };
  }, [reading, readDocs, runDocs, state.projectName]);
  const documentLabel = useCallback((id: string) => docsRef.current[id]?.number || docsRef.current[id]?.filename || "", []);

  const nav = (
    <LeftNav projects={projects} currentId={projectId} navigate={navigate} view={view} onView={showView} marks={navMarks} />
  );
  const repository = (
    <Register
      rows={rows}
      selection={documentSelection}
      onSelectionChange={setDocumentSelection}
      catalog={catalog}
      live={live}
      onCorrect={correct}
      onAddFiles={() => fileInput.current?.click()}
      onSetReading={setReading}
      notReadOnly={notReadOnly}
      onClearFilter={() => setNotReadOnly(false)}
      onDelete={remove}
      onSplit={split}
      fileOver={over}
      openKey={openDocument}
      onOpen={setOpenDocument}
    />
  );

  if (state.phase === "loading") {
    return (
      <CockpitFrame nav={nav} title="Opening project…" subtitle="Site address TBC" repository={<p className="reg-empty"><Pending delayed>Opening documents…</Pending></p>}>
        <p className="drop-sub"><Pending delayed>Opening project…</Pending></p>
      </CockpitFrame>
    );
  }
  if (state.phase === "missing") {
    return (
      <CockpitFrame nav={nav} title="Project" subtitle="This project isn't available." repository={repository}>
        <div className="empty">
          <strong>This project isn't available.</strong>
          It may belong to another organisation, or the link is wrong.
          <p style={{ marginTop: 14 }}>
            <button type="button" className="btn" onClick={onHome}>
              Back to projects
            </button>
          </p>
        </div>
      </CockpitFrame>
    );
  }
  if (state.phase === "error") {
    return (
      <CockpitFrame nav={nav} title={state.projectName || "Project"} subtitle={address || "Site address TBC"} repository={repository}>
        <div className="banner" role="alert">
          {state.error}
          <button type="button" className="btn btn-small" onClick={() => location.reload()}>
            Try again
          </button>
        </div>
      </CockpitFrame>
    );
  }

  return (
    <CockpitFrame
      nav={nav}
      actionSurface={viewingRow ? "document" : showProgram ? "program" : showCosts ? "costs" : showReports ? "procurement" : showSpecifications ? "specifications" : "profile"}
      onRename={async name => { await request("PATCH", `/projects/${projectId}`, { name }); dispatch({ type: "renamed", name }); onRenamed(projectId, name); }}
      title={state.projectName}
      subtitle={address || "Site address TBC"}
      repository={repository}
      repositoryOpenRequest={repositoryOpenRequest}
      wideWorkspace={!viewingRow && (showSpecifications || showReports || showCosts || showProgram)}
      profileFeedbackRef={setProfileFeedbackTarget}
      showProfileFeedback={!viewingRow && !showSpecifications && !showReports && !showCosts && !showProgram}
      onDragEnter={onDragEnter}
      onDragOverCapture={e => { if ((e.target as HTMLElement).closest("[data-proposal-drop]")) { dragDepth.current = 0; setOver(false); } }}
      onDropCapture={() => { dragDepth.current = 0; setOver(false); }}
      onDragEndCapture={() => { dragDepth.current = 0; setOver(false); }}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
    >
      <PageActions surface="specifications"><a href={specificationApi.exportURL(projectId)} download>Export draft CSV</a><button type="button" onClick={() => window.print()}>Print specification</button></PageActions>
      <section className="profile-col" aria-label="Project profile" hidden={!!viewingRow || showSpecifications || showReports || showCosts || showProgram}>
        <Profile projectId={projectId} tick={profileTick} workTick={reportTick} onJump={jump} onSignedOut={onSignedOut} onShowNotRead={() => setNotReadOnly(true)} onIdentity={setAddress} feedbackTarget={profileFeedbackTarget}
          documentLabel={documentLabel} onRunState={setProfileRun} onOpenView={showView} />
      </section>
      {costsOpened && <div className="project-reports" hidden={!!viewingRow || !showCosts}><Costs projectId={projectId} tick={profileTick + reportTick} invoiceTick={invoiceTick} onSignedOut={onSignedOut} /></div>}
      {specificationsOpened && <div className="project-reports" hidden={!!viewingRow || !showSpecifications}><Specifications projectId={projectId} tick={specificationTick+profileTick+reportTick} documents={Object.values(state.docs)} onSignedOut={onSignedOut} onOpenProfile={()=>showView("profile")} /></div>}
      {programOpened && <div className="project-reports" hidden={!!viewingRow || !showProgram}><Program projectId={projectId} remoteVersion={programVersion} remoteTick={programTick} onProfileChanged={() => setProfileTick((t) => t + 1)} active={showProgram && !viewingRow} onSignedOut={onSignedOut} /></div>}
      {reportsOpened && <div className="project-reports" hidden={!!viewingRow || !showReports}><Procurement transmittal={{ selectedIds: documentSelection.ids, onSelectDocuments: ids => setDocumentSelection(selectDocumentRows(ids)), onShowDocuments: ids => { setNotReadOnly(false); setRepositoryOpenRequest(n => n + 1); setDocumentSelection(selectDocumentRows(ids.filter(id => state.docs[id]))); } }} projectId={projectId} tick={profileTick + reportTick} onSignedOut={onSignedOut} onOpenCostPlan={() => showView("costs")} onOpenDocument={id => { if (!state.docs[id]) return false; setOpenDocument(id); return true; }} /></div>}
      {viewingRow && <DocumentWorkspace key={viewingRow.key} row={viewingRow} catalog={catalog} backTo={viewLabel(view)} priorLabel={priorLabel} onCorrect={correct} onRetry={retry} onSplit={split} onJump={jump} onDismiss={key => dispatch({ type: "dismiss", key })} onClose={() => {
        setOpenDocument(null);
        document.querySelector<HTMLElement>(`#doc-${viewingRow.doc?.id} .reg-line`)?.focus({ preventScroll: true });
      }} />}
      <input
        ref={fileInput}
        type="file"
        multiple
        accept={ACCEPT}
        hidden
        data-testid="file-input"
        onChange={(e) => {
          if (e.target.files) addFiles(e.target.files);
          e.target.value = "";
        }}
      />
      <p className="sr-only" aria-live="polite">
        {announcement}
      </p>
    </CockpitFrame>
  );
}
