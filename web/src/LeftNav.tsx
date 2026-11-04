// Left navigation: wordmark, project switcher, and the working views.
// Accessible names stay the short labels the tests and existing calls use
// ("Profile", "Costs"); the visible labels match the Clerk shell.

import { useEffect, useRef, useState } from "react";
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

const VIEWS: { id: WorkbenchView | "plan"; label: string; name?: string; icon: string; idle?: boolean }[] = [
  { id: "profile", label: "Project Profile", name: "Profile", icon: "M4 8h16M4 16h16M8 8v8M16 8v8" },
  { id: "plan", label: "Project Plan", icon: "M7 3h8l4 4v14H7zM15 3v5h5", idle: true },
  { id: "costs", label: "Cost Plan", name: "Costs", icon: "M12 3v18M7 8h7a3 3 0 0 1 0 6H9a3 3 0 0 0 0 6h8" },
  { id: "program", label: "Program", icon: "M4 6h16M4 12h10M4 18h13" },
  { id: "procurement", label: "Procurement", icon: "M8 4h8v3H8zM6 7h12v13H6z" },
  { id: "specifications", label: "Specifications", icon: "M7 3h8l4 4v14H7zM9 13h6M9 17h6" },
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
        <Wordmark height={30} />
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
              <svg className="nav-icon" viewBox="0 0 24 24" aria-hidden="true">
                <path d={item.icon} fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
              <span>{item.label}{item.idle && <small className="nav-planned">Planned</small>}</span>
            </button>
          );
        })}
      </div>
      <div className="nav-spacer" />
      <DevStatus />
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
