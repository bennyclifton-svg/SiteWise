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
    <nav className="leftnav" aria-label="Projects">
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
      <div className="switcher" ref={box} onKeyDown={(e) => e.key === "Escape" && setOpen(false)}>
        <span className="switcher-label">Project</span>
        <button
          type="button"
          className="switcher-button"
          aria-haspopup="listbox"
          aria-expanded={open}
          onClick={() => setOpen((o) => !o)}
          title={current?.name}
        >
          <FolderIcon />
          <span className="switcher-name">{current?.name ?? "Project"}</span>
          <span aria-hidden="true">▾</span>
        </button>
        {open && (
          <ul className="switcher-list" role="listbox" aria-label="Switch project">
            {(projects ?? []).map((p) => (
              <li key={p.id} role="option" aria-selected={p.id === currentId}>
                <button type="button" onClick={() => go(`/projects/${p.id}`)} data-active={p.id === currentId || undefined}>
                  <span>{p.name}</span>
                  {p.id === currentId && <span aria-hidden="true">✓</span>}
                </button>
              </li>
            ))}
            <li className="switcher-sep" role="presentation" />
            <li role="option" aria-selected={false}>
              <button type="button" onClick={() => go("/")}>
                All projects and new project…
              </button>
            </li>
          </ul>
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
                if (!item.idle) onView(item.id as WorkbenchView);
              }}
            >
              <svg className="nav-icon" viewBox="0 0 24 24" aria-hidden="true">
                <path d={item.icon} fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
              <span>{item.label}</span>
            </button>
          );
        })}
      </div>
      <div className="nav-spacer" />
      <section className="nav-chats" aria-label="Chats">
        <div className="nav-chats-label">
          <svg className="nav-icon" viewBox="0 0 24 24" aria-hidden="true">
            <path d="M5 6h14v9H8l-3 3z" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinejoin="round" />
          </svg>
          <span>Chats</span>
        </div>
        <p>No chats in this build.</p>
      </section>
      <DevStatus />
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
