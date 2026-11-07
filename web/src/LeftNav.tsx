// Left navigation: wordmark, project switcher, and the working views.
// Accessible names stay the short labels the tests and existing calls use
// ("Profile", "Costs"); the visible labels match the Clerk shell.

import { useEffect, useRef, useState } from "react";
import { AccountMenu } from "./AccountMenu";
import { requestCostSettings } from "./costSettingsRequest";
import { DevStatus } from "./DevStatus";
import { Wordmark } from "./Wordmark";

export type WorkbenchView = "profile" | "specifications" | "procurement" | "costs" | "program";

interface Props {
  projects: { id: string; name: string }[] | null;
  currentId: string;
  navigate: (path: string) => void;
  view: WorkbenchView;
  onView: (view: WorkbenchView) => void;
}

// Each view is drawn as a small architectural line drawing in one of two
// drawing colours, brass or blue (green and rose are kept for status). The
// solid part of each drawing takes a pale wash; the page's tables share it.
type NavIcon = { line: string; wash?: string };
const VIEWS: { id: WorkbenchView | "plan"; label: string; name?: string; icon: NavIcon; tone: string; idle?: boolean }[] = [
  // An elevation: gable, door and ground line.
  { id: "profile", label: "Project Profile", name: "Profile", tone: "blue", icon: { line: "M2.5 20h19M5 20V10.5L12 5l7 5.5V20M10 20v-5h4v5", wash: "M5 20V10.5L12 5l7 5.5V20z" } },
  // Elemental cost bars on a baseline.
  { id: "costs", label: "Cost Plan", name: "Costs", tone: "brass", icon: { line: "M3 20h18M6 20v-6h3v6M10.5 20V9h3v11M15 20V5h3v15", wash: "M15 20V5h3v15z" } },
  // A Gantt chart: three bars off a time axis.
  { id: "program", label: "Program", tone: "blue", icon: { line: "M4 3.5v17M6.5 5.5h7v3h-7zM9.5 10.5h9v3h-9zM12.5 15.5h6v3h-6z", wash: "M9.5 10.5h9v3h-9z" } },
  // A document set going out: two sheets and a tick.
  { id: "procurement", label: "Procurement", tone: "brass", icon: { line: "M9 3.5h10v13M5 7.5h10v13H5zM7.5 14l2 2 3.5-4", wash: "M5 7.5h10v13H5z" } },
  // A drawing sheet with its title block.
  { id: "specifications", label: "Specifications", tone: "blue", icon: { line: "M3.5 4.5h17v15h-17zM13 15h7.5M13 15v4.5M6.5 8h8M6.5 11h5", wash: "M13 15h7.5v4.5H13z" } },
  // The old Project Plan placeholder: still not built, so it stays disabled and marked Planned.
  { id: "plan", label: "Report", tone: "report", icon: { line: "M6 3h12v18H6zM9 3v18M12.5 16v-2M15 16v-5" }, idle: true },
];

export function LeftNav({ projects, currentId, navigate, view, onView }: Props) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [mobileOpen, setMobileOpen] = useState(false);
  const menuButton = useRef<HTMLButtonElement>(null);
  const box = useRef<HTMLDivElement>(null);
  const current = projects?.find((p) => p.id === currentId);

  useEffect(() => {
    if (!open) return;
    const close = (e: MouseEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, [open]);

  const go = (path: string) => {
    setOpen(false);
    navigate(path);
  };

  return (
    <nav className="leftnav" aria-label="Projects" data-mobile-open={mobileOpen || undefined} onKeyDown={e => { if (e.key === "Escape" && !open) { setMobileOpen(false); menuButton.current?.focus(); } }}>
      <button ref={menuButton} type="button" className="btn mobile-nav-toggle" aria-expanded={mobileOpen} aria-controls="project-navigation" onClick={() => setMobileOpen(value => !value)}>Project navigation</button>
      <div id="project-navigation" className="navigation-content">
      <a
        className="leftnav-logo"
        href="/"
        aria-label="SiteWise, all projects"
        onClick={(e) => {
          e.preventDefault();
          go("/");
        }}
      >
        <Wordmark height={32} />
      </a>
      <div className="switcher" ref={box} onKeyDown={(e) => {
        if (e.key === "Escape" && open) {
          e.stopPropagation(); setOpen(false);
          box.current?.querySelector<HTMLButtonElement>(".switcher-button")?.focus();
        }
      }}>
        <span className="switcher-label">Project</span>
        <button
          type="button"
          className="switcher-button"
          aria-controls="project-switcher"
          aria-expanded={open}
          onClick={() => setOpen((o) => !o)}
          title={current?.name}
        >
          <FolderIcon />
          <span className="switcher-name">{current?.name ?? "Project"}</span>
          <span aria-hidden="true">▾</span>
        </button>
        {open && (
          <div id="project-switcher" className="switcher-list" aria-label="Switch project">
            <input className="input" aria-label="Find project" placeholder="Find project" value={query} onChange={e => setQuery(e.target.value)} />
            <ul className="switcher-results">
              {(projects ?? []).filter(p => p.name.toLowerCase().includes(query.toLowerCase())).map(p => (
                <li key={p.id}>
                  <button type="button" onClick={() => go(`/projects/${p.id}`)} aria-current={p.id === currentId ? "true" : undefined} data-active={p.id === currentId || undefined}>{p.name}</button>
                </li>
              ))}
              {projects && !projects.some(p => p.name.toLowerCase().includes(query.toLowerCase())) && <li className="muted">No matching projects.</li>}
            </ul>
            <button type="button" className="switcher-home" onClick={() => go("/")}>All projects and new project…</button>
          </div>
        )}
      </div>
      <div className="nav-views" role="group" aria-label="Project views">
        {VIEWS.map((item) => {
          const active = !item.idle && item.id === view;
          return (
            <button
              key={item.id}
              type="button"
              className="nav-item"
              aria-current={active ? "page" : undefined}
              aria-label={item.name}
              aria-disabled={item.idle || undefined}
              disabled={item.idle}
              title={item.idle ? "Not in this build yet" : undefined}
              onClick={() => {
                if (!item.idle) {
                  onView(item.id as WorkbenchView); setMobileOpen(false);
                  if (menuButton.current?.offsetParent) menuButton.current.focus();
                }
              }}
            >
              <svg className="nav-icon" data-tone={item.tone} viewBox="0 0 24 24" aria-hidden="true">
                {item.icon.wash && <path className="nav-icon-wash" d={item.icon.wash} />}
                <path d={item.icon.line} fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
              <span>{item.label}{item.idle && <small className="nav-planned">Planned</small>}</span>
            </button>
          );
        })}
      </div>
      <div className="nav-spacer" />
      <DevStatus />
      <AccountMenu onCostSettings={() => { onView("costs"); setMobileOpen(false); requestCostSettings(); }} />
      </div>
    </nav>
  );
}

function FolderIcon() {
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M3 7h6l2 2h10v9H3z" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinejoin="round" />
    </svg>
  );
}
