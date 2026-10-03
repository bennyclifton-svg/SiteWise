// Left navigation on a project page: the wordmark and a project switcher,
// after the Clerk project nav. More destinations join here later.

import { useEffect, useRef, useState } from "react";

interface Props {
  projects: { id: string; name: string }[] | null;
  currentId: string;
  navigate: (path: string) => void;
}

export function LeftNav({ projects, currentId, navigate }: Props) {
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
        onClick={(e) => {
          e.preventDefault();
          go("/");
        }}
      >
        <img src="/sitewise-logo.png" alt="SiteWise, all projects" width="118" height="28" />
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
    </nav>
  );
}
