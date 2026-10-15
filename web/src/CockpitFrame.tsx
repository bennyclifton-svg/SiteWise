import { useEffect, useRef, useState, type DragEvent, type KeyboardEvent, type PointerEvent as ReactPointerEvent, type ReactNode } from "react";
import { ThemeButton } from "./ThemeButton";

interface Props {
  nav: ReactNode;
  title: string;
  subtitle: string;
  children: ReactNode;
  repository: ReactNode;
  wideWorkspace?: boolean;
  onDragEnter?: (e: DragEvent) => void;
  onDragOver?: (e: DragEvent) => void;
  onDragLeave?: () => void;
  onDrop?: (e: DragEvent) => void;
}

// Chosen width of the document rail. Null until the user drags it, so a
// window resize can still pick a width that fits the screen.
const REPO_WIDTH_KEY = "sitewise.cockpit.repoWidth.v1";
const REPO_MIN = 280;
const REPO_PREFERRED = 760;
const REPO_ABS_MAX = 920;
const MAIN_FLOOR = 200;
const MAIN_DEFAULT = 420;
const WIDE_SHELL = "(min-width: 1101px)";

function repoMax(viewport: number) {
  // The rail can be dragged until the working view is about 200px, which is
  // wide enough to open every document column.
  return Math.max(REPO_MIN, Math.min(REPO_ABS_MAX, viewport - 268 - MAIN_FLOOR - 56));
}

function repoDefault(viewport: number) {
  const room = viewport - 268 - MAIN_DEFAULT - 56;
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
export function CockpitFrame({ nav, title, subtitle, children, repository, wideWorkspace=false, onDragEnter, onDragOver, onDragLeave, onDrop }: Props) {
  const [showDocuments,setShowDocuments]=useState(false);
  const focusWorkspace=wideWorkspace&&!showDocuments;
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

  return (
    <div className={`cockpit-page${wideWorkspace?" specification-page":""}`}>
      <div
        className={dragging ? "cockpit-shell is-resizing" : "cockpit-shell"}
        style={wide ? { gridTemplateColumns: focusWorkspace?"minmax(220px, 268px) minmax(0, 1fr)":`minmax(220px, 268px) minmax(0, 1fr) ${width}px` } : undefined}
        onDragEnter={onDragEnter}
        onDragOver={onDragOver}
        onDragLeave={onDragLeave}
        onDrop={onDrop}
      >
        {nav}
        <header className="cockpit-shell-header">
          <div className="cockpit-title-block">
            <h1 className="cockpit-title">{title}</h1>
            <p className="cockpit-address">{subtitle}</p>
          </div>
          <ThemeButton />
          {wideWorkspace&&<button className="btn" aria-expanded={showDocuments} onClick={()=>setShowDocuments(v=>!v)}>{showDocuments?"Hide documents":"Show documents"}</button>}
        </header>
        <main className="project-main-panel" id="main">
          <div className="cockpit-scroll">{children}</div>
          <AskBar />
        </main>
        <aside className="project-side-panel" aria-label="Document register" style={focusWorkspace?{display:"none"}:undefined}>
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
          {repository}
        </aside>
      </div>
    </div>
  );
}

// Visual match for the Clerk composer. It does not send: SiteWise has no chat model.
function AskBar() {
  return (
    <div className="ask-bar" role="group" aria-label="Ask about your project documents. Not connected.">
      <input readOnly tabIndex={-1} placeholder="Ask about your project documents" aria-label="Ask about your project documents" />
      <div className="ask-modes">
        <button type="button" aria-pressed="true" disabled>
          Fast
        </button>
        <button type="button" aria-pressed="false" disabled>
          Thorough
        </button>
      </div>
      <button type="button" className="ask-icon" disabled aria-label="Voice input">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12 3a3 3 0 0 1 3 3v6a3 3 0 0 1-6 0V6a3 3 0 0 1 3-3z" fill="none" stroke="currentColor" strokeWidth="1.7" />
          <path d="M6 11a6 6 0 0 0 12 0M12 17v4" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
        </svg>
      </button>
      <button type="button" className="ask-icon ask-send" disabled aria-label="Send">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M5 12h14M13 6l6 6-6 6" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </button>
    </div>
  );
}
