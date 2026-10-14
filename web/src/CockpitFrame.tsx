import type { DragEvent, ReactNode } from "react";
import { ThemeButton } from "./ThemeButton";

interface Props {
  nav: ReactNode;
  title: string;
  subtitle: string;
  children: ReactNode;
  repository: ReactNode;
  onDragEnter?: (e: DragEvent) => void;
  onDragOver?: (e: DragEvent) => void;
  onDragLeave?: () => void;
  onDrop?: (e: DragEvent) => void;
}

// The project page is three columns on a wide screen: navigation, the working
// view, and the document repository. The repository stays up on every view.
export function CockpitFrame({ nav, title, subtitle, children, repository, onDragEnter, onDragOver, onDragLeave, onDrop }: Props) {
  return (
    <div className="cockpit-page">
      <div className="cockpit-shell" onDragEnter={onDragEnter} onDragOver={onDragOver} onDragLeave={onDragLeave} onDrop={onDrop}>
        {nav}
        <header className="cockpit-shell-header">
          <div className="cockpit-title-block">
            <h1 className="cockpit-title">{title}</h1>
            <p className="cockpit-address">{subtitle}</p>
          </div>
          <ThemeButton />
        </header>
        <main className="project-main-panel" id="main">
          <div className="cockpit-scroll">{children}</div>
          <AskBar />
        </main>
        <aside className="project-side-panel" aria-label="Document register">
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
