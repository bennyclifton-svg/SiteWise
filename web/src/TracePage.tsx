import type { ReactNode } from "react";
import { AccountMenu } from "./AccountMenu";
import { Wordmark } from "./Wordmark";

// The pages outside a project (Projects, Account) in the Trace grammar
// (web/DESIGN.md): the drafting table, one white sheet, and a band of
// butter paper holding the title and the page's own actions. Wide screens
// get the sheet and the paper; narrow ones stack, as the workbench does.

export function TracePage({ onHome, children }: { onHome: () => void; children: ReactNode }) {
  return (
    <div className="trace-home">
      <header className="trace-home-bar">
        <a
          className="bar-logo-link"
          href="/"
          aria-label="SiteWise, all projects"
          onClick={(e) => {
            e.preventDefault();
            onHome();
          }}
        >
          <Wordmark height={28} />
        </a>
        <span className="bar-spacer" />
        <AccountMenu placement="bar" />
      </header>
      <main id="main" className="trace-home-sheet">
        {children}
      </main>
    </div>
  );
}

/** The title band: title, one line under it, and page-level actions (ground-line buttons). */
export function TraceBand({ title, meta, actions }: { title: string; meta?: ReactNode; actions?: ReactNode }) {
  return (
    <div className="trace-band">
      <div className="trace-band-title">
        <h1>{title}</h1>
        {meta && <p className="trace-band-meta">{meta}</p>}
      </div>
      {actions && <div className="trace-band-actions">{actions}</div>}
    </div>
  );
}
