import { PageActionsContext } from "./PageActions";
import { useCallback, useEffect, useRef, useState, type DragEvent, type KeyboardEvent, type PointerEvent as ReactPointerEvent, type ReactNode } from "react";
import { IconDocumentsPanel } from "./icons";
import { ProjectName } from "./ProjectName";
interface Props {
  nav: ReactNode;
  actionSurface?: string;
  title: string;
  onRename?: (name: string) => Promise<void>;
  subtitle: string;
  children: ReactNode;
  repository: ReactNode;
  wideWorkspace?: boolean;
  repositoryOpenRequest?: number;
  profileFeedbackRef?: (element: HTMLDivElement | null) => void;
  showProfileFeedback?: boolean;
  onDragEnter?: (e: DragEvent) => void;
  onDragOver?: (e: DragEvent) => void;
  onDragOverCapture?: (e: DragEvent) => void;
  onDropCapture?: (e: DragEvent) => void;
  onDragEndCapture?: (e: DragEvent) => void;
  onDragLeave?: () => void;
  onDrop?: (e: DragEvent) => void;
}

// Chosen width of the document rail. Null until the user drags it, so a
// window resize can still pick a width that fits the screen.
const REPO_WIDTH_KEY = "sitewise.cockpit.repoWidth.v1";
const REPO_MIN = 280;
// Wide enough for every register column; the working view keeps the rest.
const REPO_PREFERRED = 540;
const REPO_ABS_MAX = 920;
const MAIN_FLOOR = 200;
const MAIN_DEFAULT = 420;
const WIDE_SHELL = "(min-width: 1101px)";
// The left nav, about 15% narrower than its old 236px.
const NAV_WIDTH = 200;
const NAV_COLUMN = `minmax(170px, ${NAV_WIDTH}px)`;

function repoMax(viewport: number) {
  // The rail can be dragged until the working view is about 200px, which is
  // wide enough to open every document column.
  return Math.max(REPO_MIN, Math.min(REPO_ABS_MAX, viewport - NAV_WIDTH - MAIN_FLOOR - 56));
}

function repoDefault(viewport: number) {
  const room = viewport - NAV_WIDTH - MAIN_DEFAULT - 56;
  return Math.min(repoMax(viewport), Math.max(REPO_MIN, Math.min(REPO_PREFERRED, room)));
}

function clampRepo(px: number, viewport: number) {
  return Math.round(Math.min(repoMax(viewport), Math.max(REPO_MIN, px)));
}

function readRepoWidth(): number | null {
  try {
    const raw = localStorage.getItem(REPO_WIDTH_KEY);
    if (!raw) return null;
    const n = Number(raw);
    return Number.isFinite(n) ? n : null;
  } catch {
    return null;
  }
}

function storeRepoWidth(px: number) {
  try {
    localStorage.setItem(REPO_WIDTH_KEY, String(px));
  } catch {
    // A private window can refuse storage. The width still applies for this visit.
  }
}

// The project page is three columns on a wide screen: navigation, the working
// view, and the document repository. The repository stays up on every view.
// Its width is dragged from the left edge; columns fold in cockpit.css when
// the rail is too narrow to read them.
export function CockpitFrame({ actionSurface = "", nav, title, onRename, subtitle, children, repository, wideWorkspace=false, repositoryOpenRequest=0, profileFeedbackRef, showProfileFeedback=false, onDragEnter, onDragOver, onDragLeave, onDrop, onDragOverCapture, onDropCapture, onDragEndCapture }: Props) {
  const [showDocuments,setShowDocuments]=useState(true);
  useEffect(() => { if (repositoryOpenRequest) setShowDocuments(true); }, [repositoryOpenRequest]);
  const focusWorkspace=!showDocuments;
  const [wide, setWide] = useState(() => window.matchMedia(WIDE_SHELL).matches);
  const [viewport, setViewport] = useState(() => window.innerWidth);
  const [chosen, setChosen] = useState<number | null>(readRepoWidth);
  const [dragging, setDragging] = useState(false);
  const drag = useRef<{ x: number; w: number } | null>(null);
  const width = clampRepo(chosen ?? repoDefault(viewport), viewport);
  const max = repoMax(viewport);

  useEffect(() => {
    const mq = window.matchMedia(WIDE_SHELL);
    const onChange = () => {
      setWide(mq.matches);
      setViewport(window.innerWidth);
    };
    onChange();
    mq.addEventListener("change", onChange);
    window.addEventListener("resize", onChange);
    return () => {
      mq.removeEventListener("change", onChange);
      window.removeEventListener("resize", onChange);
    };
  }, []);

  function setWidth(next: number) {
    const px = clampRepo(next, window.innerWidth);
    setChosen(px);
    storeRepoWidth(px);
  }

  function onResizeDown(e: ReactPointerEvent<HTMLDivElement>) {
    if (e.button !== 0) return;
    e.preventDefault();
    drag.current = { x: e.clientX, w: width };
    setDragging(true);
    try {
      e.currentTarget.setPointerCapture(e.pointerId);
    } catch {
      // Capture fails if the pointer is already gone. Moves still update the width.
    }
  }

  function onResizeMove(e: ReactPointerEvent<HTMLDivElement>) {
    if (!drag.current) return;
    setWidth(drag.current.w - (e.clientX - drag.current.x));
  }

  function onResizeUp(e: ReactPointerEvent<HTMLDivElement>) {
    if (!drag.current) return;
    setWidth(drag.current.w - (e.clientX - drag.current.x));
    drag.current = null;
    setDragging(false);
  }

  function onResizeKey(e: KeyboardEvent<HTMLDivElement>) {
    const step = e.shiftKey ? 48 : 16;
    if (e.key === "ArrowLeft") {
      e.preventDefault();
      setWidth(width + step);
    } else if (e.key === "ArrowRight") {
      e.preventDefault();
      setWidth(width - step);
    } else if (e.key === "Home") {
      e.preventDefault();
      setWidth(REPO_MIN);
    } else if (e.key === "End") {
      e.preventDefault();
      setWidth(max);
    }
  }

  // The perspex band covers the header rows. Its height is the space the
  // working view leaves at its top, so scrolled content passes under it.
  const shell = useRef<HTMLDivElement>(null);
  const paper = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = paper.current, host = shell.current;
    if (!el || !host || typeof ResizeObserver === "undefined") return;
    const sync = () => host.style.setProperty("--trace-top", `${Math.round(el.getBoundingClientRect().height)}px`);
    sync();
    const ro = new ResizeObserver(sync);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const [actionTarget, setActionTarget] = useState<HTMLDivElement | null>(null);
  const [besideTarget, setBesideTarget] = useState<HTMLDivElement | null>(null);
  const [primaryTarget, setPrimaryTarget] = useState<HTMLDivElement | null>(null);
  const [sideTarget, setSideTarget] = useState<HTMLDivElement | null>(null);
  const openSide = useCallback(() => setShowDocuments(true), []);
  return (
    <PageActionsContext.Provider value={{ surface: actionSurface, target: actionTarget, primaryTarget, besideTarget, sideTarget, openSide }}>
    <div className={`cockpit-page${wideWorkspace?" specification-page":""}`}>
      <div
        ref={shell}
        className={dragging ? "cockpit-shell is-resizing" : "cockpit-shell"}
        style={wide ? { gridTemplateColumns: focusWorkspace?`${NAV_COLUMN} minmax(0, 1fr)`:`${NAV_COLUMN} minmax(0, 1fr) ${width}px` } : undefined}
        onDragEnter={onDragEnter}
        onDragOver={onDragOver}
        onDragOverCapture={onDragOverCapture}
        onDropCapture={onDropCapture}
        onDragEndCapture={onDragEndCapture}
        onDragLeave={onDragLeave}
        onDrop={onDrop}
      >
        {nav}
        <div className="trace-sheet" aria-hidden="true" />
        <div className="trace-paper" ref={paper} aria-hidden="true">
          <div className="trace-paper-sheet" />
        </div>
        <header className="cockpit-shell-header">
          <div className="cockpit-title-block">
            {onRename ? <ProjectName name={title} onSave={onRename} /> : <h1 className="cockpit-title">{title}</h1>}
            <p className="cockpit-address">{subtitle}</p>
          </div>
          <div className="cockpit-header-actions">
            <button
              type="button"
              className="documents-toggle"
              aria-expanded={showDocuments}
              aria-controls="project-documents"
              aria-label={showDocuments ? "Hide documents" : "Show documents"}
              title={showDocuments ? "Hide documents" : "Show documents"}
              onClick={() => setShowDocuments(open => !open)}
            >
              <IconDocumentsPanel />
            </button>
          </div>
        </header>
        <main className="project-main-panel" id="main">
          <div className="cockpit-scroll">
            {/* Page actions sit on the sheet, under the title band, so the ribbon stays the title only. */}
            <div className="panel-actions" aria-label="Workspace actions">
              <div className="workspace-primary-actions" ref={setPrimaryTarget} />
              <div className="page-action-rail">
                <div className="page-action-beside" ref={setBesideTarget} />
                <div ref={setActionTarget} />
              </div>
            </div>
            {children}
          </div>
          <div ref={profileFeedbackRef} hidden={!showProfileFeedback} className="profile-feedback-slot" />
        </main>
        <aside id="project-documents" className="project-side-panel" aria-label="Document register" style={focusWorkspace?{display:"none"}:undefined}>
          {wide && (
            <div
              className="repo-resize"
              role="separator"
              aria-orientation="vertical"
              aria-label="Resize documents panel"
              aria-valuemin={REPO_MIN}
              aria-valuemax={max}
              aria-valuenow={width}
              tabIndex={0}
              onPointerDown={onResizeDown}
              onPointerMove={onResizeMove}
              onPointerUp={onResizeUp}
              onPointerCancel={onResizeUp}
              onKeyDown={onResizeKey}
            />
          )}
          <div className="side-panel-slot" ref={setSideTarget} />
          {repository}
        </aside>
      </div>
    </div>
    </PageActionsContext.Provider>
  );
}
