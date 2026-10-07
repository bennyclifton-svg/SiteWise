// A chosen project: drop files anywhere on the page, watch each become a title
// block, correct any box. Live updates come from the durable event stream and
// never move focus.

import { useCallback, useEffect, useMemo, useReducer, useRef, useState, type DragEvent } from "react";
import {
  api,
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
import { Profile } from "./Profile";
import { Register } from "./Register";
import { Reports } from "./Reports";
import { Packages } from "./Packages";
import { Works } from "./Works";
import { Delivery } from "./Delivery";
import { Costs, CostSummary } from "./Costs";
import { Proposals } from "./Proposals";
import { ocrStage } from "./OCRStatus";

const STALE_MS = 8000;
const UPLOAD_CONCURRENCY = 4;
const ACCEPT = ".pdf,.docx,.xlsx";

interface Upload {
  key: string;
  filename: string;
  progress: number;
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
  | { type: "loaded"; list: DocumentList }
  | { type: "failed"; missing: boolean; error: string }
  | { type: "uploadStart"; items: { key: string; filename: string }[] }
  | { type: "uploadProgress"; key: string; progress: number }
  | { type: "uploadDone"; key: string; doc: Doc; created: boolean; at: number }
  | { type: "uploadFailed"; key: string; error: string }
  | { type: "dismiss"; key: string }
  | { type: "event"; ev: StreamEvent; at: number }
  | { type: "doc"; doc: Doc }
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
      for (const it of action.items) uploads[it.key] = { key: it.key, filename: it.filename, progress: 0 };
      return { ...state, uploads, order: [...action.items.map((it) => it.key), ...state.order] };
    }
    case "uploadProgress": {
      const up = state.uploads[action.key];
      if (!up) return state;
      return { ...state, uploads: { ...state.uploads, [action.key]: { ...up, progress: action.progress } } };
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
  projectId: string;
  catalog: Catalog | null;
  onSignedOut: () => void;
  onHome: () => void;
}

export function Project({ projectId, catalog, onSignedOut, onHome }: Props) {
  const [state, dispatch] = useReducer(reducer, initial);
  const [cursor, setCursor] = useState<number | null>(null);
  const [live, setLive] = useState<"live" | "reconnecting" | "offline">("reconnecting");
  const [over, setOver] = useState(false);
  const [announcement, setAnnouncement] = useState("");
  const [flash, setFlash] = useState<string | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const [profileTick, setProfileTick] = useState(0);
  const [showReports, setShowReports] = useState(false);
  const [showWorks, setShowWorks] = useState(false);
  const [worksOpened, setWorksOpened] = useState(false);
  const [showProposals, setShowProposals] = useState(false);
  const [proposalsOpened, setProposalsOpened] = useState(false);
  const [showDelivery, setShowDelivery] = useState(false);
  const [showCosts, setShowCosts] = useState(false);
  const [profileView, setProfileView] = useState<"all" | "summary" | "systems">("all");
  const [costsOpened, setCostsOpened] = useState(false);
  const [deliveryOpened, setDeliveryOpened] = useState(false);
  const [showPackages, setShowPackages] = useState(false);
  const [packagesOpened, setPackagesOpened] = useState(false);
  const [reportsOpened, setReportsOpened] = useState(false);
	const [reportTick, setReportTick] = useState(0);
  const [notReadOnly, setNotReadOnly] = useState(false);
  const lastId = useRef(0);
  const fileInput = useRef<HTMLInputElement>(null);
  const queue = useRef<{ key: string; file: File }[]>([]);
  const active = useRef(0);
  const dragDepth = useRef(0);
  const docsRef = useRef(state.docs);
  docsRef.current = state.docs;

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
		  if (ev.kind === "report" || ev.kind === "works" || ev.kind === "packages" || ev.kind === "delivery" || ev.kind === "costs") {
		    if (ev.payload?.project_id === projectId) setReportTick(n => n + 1);
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
          const doc = docsRef.current[ev.payload?.document_id ?? ""];
          if (!doc) return;
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

  // The whole page is a drop target once a project is chosen.
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
        if (up) return [{ key, filename: up.filename, progress: up.progress, uploadError: up.error }];
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

  if (state.phase === "loading") {
    return (
      <main className="page" aria-busy="true">
        <p className="drop-sub">Opening project…</p>
      </main>
    );
  }
  if (state.phase === "missing") {
    return (
      <main className="page">
        <div className="empty">
          <strong>This project isn't available.</strong>
          It may belong to another organisation, or the link is wrong.
          <p style={{ marginTop: 14 }}>
            <button type="button" className="btn" onClick={onHome}>
              Back to projects
            </button>
          </p>
        </div>
      </main>
    );
  }
  if (state.phase === "error") {
    return (
      <main className="page">
        <div className="banner" role="alert">
          {state.error}
          <button type="button" className="btn btn-small" onClick={() => location.reload()}>
            Try again
          </button>
        </div>
      </main>
    );
  }

  return (
    <main className="workspace workspace-with-views" onDragEnter={onDragEnter} onDragOver={onDragOver} onDragLeave={onDragLeave} onDrop={onDrop}>
      <nav className="project-views" aria-label="Project views">
        <button type="button" className="btn btn-small" aria-pressed={!showReports && !showPackages && !showWorks && !showDelivery && !showProposals && !showCosts && profileView === "all"} onClick={() => { setProfileView("all"); setShowCosts(false); setShowReports(false); setShowPackages(false); setShowWorks(false); setShowDelivery(false); setShowProposals(false); }}>Filing</button>
        <button type="button" className="btn btn-small" aria-pressed={showWorks} onClick={() => { setShowCosts(false); setWorksOpened(true); setShowWorks(true); setShowReports(false); setShowPackages(false); setShowDelivery(false); setShowProposals(false); }}>Works</button>
        <button type="button" className="btn btn-small" aria-pressed={showProposals} onClick={() => { setShowCosts(false); setProposalsOpened(true); setShowProposals(true); setShowWorks(false); setShowReports(false); setShowPackages(false); setShowDelivery(false); }}>Proposals</button>
        <button type="button" className="btn btn-small" aria-pressed={showPackages} onClick={() => { setShowCosts(false); setPackagesOpened(true); setShowPackages(true); setShowWorks(false); setShowReports(false); setShowDelivery(false); setShowProposals(false); }}>Packages</button>
        <button type="button" className="btn btn-small" aria-pressed={showDelivery} onClick={() => { setShowCosts(false); setShowProposals(false); setDeliveryOpened(true); setShowDelivery(true); setShowWorks(false); setShowReports(false); setShowPackages(false); }}>Delivery</button>
        <button type="button" className="btn btn-small" aria-pressed={showReports} onClick={() => { setShowCosts(false); setReportsOpened(true); setShowReports(true); setShowPackages(false); setShowWorks(false); setShowDelivery(false); setShowProposals(false); }}>Reports</button>
        <button type="button" className="btn btn-small" aria-pressed={showCosts} onClick={() => { setCostsOpened(true); setShowCosts(true); setShowWorks(false); setShowReports(false); setShowPackages(false); setShowDelivery(false); setShowProposals(false); }}>Costs</button>
        <button type="button" className="btn btn-small" aria-pressed={profileView === "summary" && !showCosts && !showReports && !showPackages && !showWorks && !showDelivery && !showProposals} onClick={() => { setProfileView("summary"); setShowCosts(false); setShowReports(false); setShowPackages(false); setShowWorks(false); setShowDelivery(false); setShowProposals(false); }}>Summary</button>
        <button type="button" className="btn btn-small" aria-pressed={profileView === "systems" && !showCosts && !showReports && !showPackages && !showWorks && !showDelivery && !showProposals} onClick={() => { setProfileView("systems"); setShowCosts(false); setShowReports(false); setShowPackages(false); setShowWorks(false); setShowDelivery(false); setShowProposals(false); }}>Systems</button>
      </nav>
      <section className={`profile-col ${profileView !== "all" ? "profile-focused" : ""}`} aria-label="Project profile" hidden={showReports || showPackages || showWorks || showDelivery || showProposals || showCosts}>
        <h1 className="project-title">{state.projectName}</h1>
        {profileView === "summary" && <CostSummary projectId={projectId} tick={profileTick + reportTick} onSignedOut={onSignedOut} />}
        <Profile view={profileView} projectId={projectId} tick={profileTick} onJump={id => { setProfileView("all"); jump(id); }} onSignedOut={onSignedOut} onShowNotRead={() => { setProfileView("all"); setNotReadOnly(true); }} />
      </section>
      <aside className="register-col" aria-label="Document register" hidden={profileView !== "all" || showReports || showPackages || showWorks || showDelivery || showProposals || showCosts}>
        <Register
          rows={rows}
          catalog={catalog}
          live={live}
          priorLabel={priorLabel}
          onCorrect={correct}
          onRetry={retry}
          onJump={jump}
          onDismiss={(key) => dispatch({ type: "dismiss", key })}
          onAddFiles={() => fileInput.current?.click()}
          onSetReading={setReading}
          notReadOnly={notReadOnly}
          onClearFilter={() => setNotReadOnly(false)}
          onDelete={remove}
        />
      </aside>
      {worksOpened && <div className="project-reports" hidden={!showWorks}><Works projectId={projectId} tick={profileTick + reportTick} onSignedOut={onSignedOut} /></div>}
      {proposalsOpened && <div className="project-reports" hidden={!showProposals}><Proposals projectId={projectId} tick={profileTick + reportTick} onSignedOut={onSignedOut} /></div>}
      {costsOpened && <div className="project-reports" hidden={!showCosts}><Costs projectId={projectId} tick={profileTick + reportTick} onSignedOut={onSignedOut} /></div>}
      {deliveryOpened && <div className="project-reports" hidden={!showDelivery}><Delivery projectId={projectId} tick={profileTick + reportTick} onSignedOut={onSignedOut} /></div>}
      {reportsOpened && <div className="project-reports" hidden={!showReports}><Reports projectId={projectId} tick={profileTick + reportTick} onSignedOut={onSignedOut} /></div>}
      {packagesOpened && <div className="project-reports" hidden={!showPackages}><Packages projectId={projectId} tick={profileTick + reportTick} onSignedOut={onSignedOut} /></div>}
      {/* Drop overlay: invisible until files are dragged over the page. */}
      <div className="drop" data-over={over ? "true" : undefined} aria-hidden={!over}>
        <p className="drop-title">Release to file into {state.projectName}</p>
      </div>
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
    </main>
  );
}
