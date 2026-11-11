// The Projects page: pick the job that needs you. Each row carries the site
// address, the filing count, values to check and the last filing, all read in
// one query (GET /projects); nothing here calls Jev. Rows sort by the latest
// filing so live jobs sit on top. Archive puts a project away without
// touching it; delete stays behind its confirmation.

import { useEffect, useRef, useState, type FormEvent, type KeyboardEvent } from "react";
import { Pending } from "./Activity";
import { api, ApiError, type Project } from "./api";
import { ConfirmDialog } from "./ConfirmDialog";
import { GroundButton } from "./GroundButton";
import { TraceBand } from "./TracePage";

type Props = {
  projects: Project[] | null;
  loadError: string;
  onRetry: () => void;
  onOpen: (id: string) => void;
  onCreated: (id: string) => void;
  onChanged: (id: string, patch: Partial<Project>) => void;
  onDeleted: (id: string) => void;
  onSignedOut: () => void;
};

const DAY = 86_400_000;

/** "Today", "Yesterday", "3 days ago", "4 Nov", "24 Oct 2025". */
export function whenLabel(iso: string, now = new Date()): string {
  const at = new Date(iso);
  const startOf = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const days = Math.round((startOf(now) - startOf(at)) / DAY);
  if (days <= 0) return "Today";
  if (days === 1) return "Yesterday";
  if (days < 7) return `${days} days ago`;
  const sameYear = at.getFullYear() === now.getFullYear();
  return at.toLocaleDateString("en-AU", sameYear ? { day: "numeric", month: "short" } : { day: "numeric", month: "short", year: "numeric" });
}

const fullDate = (iso: string) => new Date(iso).toLocaleString("en-AU", { day: "numeric", month: "long", year: "numeric", hour: "numeric", minute: "2-digit" });

function matches(p: Project, query: string): boolean {
  const q = query.trim().toLowerCase();
  return !q || p.name.toLowerCase().includes(q) || (p.address ?? "").toLowerCase().includes(q);
}

export function ProjectsPage({ projects, loadError, onRetry, onOpen, onCreated, onChanged, onDeleted, onSignedOut }: Props) {
  const [query, setQuery] = useState("");
  const [showArchived, setShowArchived] = useState(false);
  const [error, setError] = useState("");
  const [confirm, setConfirm] = useState<Project | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [renaming, setRenaming] = useState<string | null>(null);
  const search = useRef<HTMLInputElement>(null);
  const table = useRef<HTMLTableSectionElement>(null);

  const all = projects ?? [];
  const archivedCount = all.filter((p) => p.archived).length;
  const activeCount = all.length - archivedCount;
  // With nothing archived left, go back to the active list.
  const viewingArchived = showArchived && archivedCount > 0;
  const inView = all.filter((p) => !!p.archived === viewingArchived);
  const shown = inView.filter((p) => matches(p, query));

  // "/" finds a project, as in most list pages.
  useEffect(() => {
    const onKey = (e: globalThis.KeyboardEvent) => {
      if (e.key !== "/" || e.ctrlKey || e.metaKey || e.altKey) return;
      const t = e.target as HTMLElement | null;
      if (t && (t.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(t.tagName))) return;
      e.preventDefault();
      search.current?.focus();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  function fail(err: unknown, fallback: string) {
    if (err instanceof ApiError && err.status === 401) return onSignedOut();
    setError(err instanceof Error && err.message ? err.message : fallback);
  }

  async function remove() {
    if (!confirm) return;
    setDeleting(true);
    try {
      await api.deleteProject(confirm.id);
      onDeleted(confirm.id);
      setConfirm(null);
    } catch (err) {
      fail(err, "Couldn't delete the project.");
      setConfirm(null);
    } finally {
      setDeleting(false);
    }
  }

  async function archive(p: Project, archived: boolean) {
    setError("");
    try {
      await api.archiveProject(p.id, archived);
      onChanged(p.id, { archived });
    } catch (err) {
      fail(err, archived ? "Couldn't archive the project." : "Couldn't restore the project.");
    }
  }

  async function rename(p: Project, name: string) {
    setRenaming(null);
    const value = name.trim();
    if (!value || value === p.name) return;
    setError("");
    try {
      await api.renameProject(p.id, value);
      onChanged(p.id, { name: value });
    } catch (err) {
      fail(err, "Couldn't rename the project.");
    }
  }

  // Up and down move between project names; Enter opens (the name is a link).
  function onRowsKey(e: KeyboardEvent<HTMLTableSectionElement>) {
    if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
    const links = Array.from(table.current?.querySelectorAll<HTMLElement>(".pl-open") ?? []);
    const at = links.indexOf(document.activeElement as HTMLElement);
    if (at < 0) return;
    e.preventDefault();
    links[Math.min(links.length - 1, Math.max(0, at + (e.key === "ArrowDown" ? 1 : -1)))]?.focus();
  }

  const meta = projects === null
    ? (loadError ? "Projects not loaded" : "Loading…")
    : `${activeCount} ${activeCount === 1 ? "project" : "projects"}${archivedCount ? ` · ${archivedCount} archived` : ""}`;

  return (
    <div className="projects-page">
      <TraceBand
        title="Projects"
        meta={meta}
        actions={<NewProject onCreated={onCreated} onSignedOut={onSignedOut} first={projects?.length === 0} />}
      />
      <div className="trace-home-body">
        {loadError && <div className="home-load-error" role="alert">{loadError} <button className="btn" onClick={onRetry}>Retry projects</button></div>}
        {error && <p className="error-text" role="alert">{error}</p>}

        {projects === null && !loadError ? (
          <p className="pl-note"><Pending delayed>Loading projects…</Pending></p>
        ) : projects?.length === 0 ? (
          <div className="pl-empty">
            <strong>No projects yet.</strong>
            <span>Name your first project above (a site address works well), then drop its documents in.</span>
          </div>
        ) : projects && (
          <>
            <div className="pl-toolbar">
              <label className="pl-search">
                <span className="sr-only">Find a project</span>
                <input ref={search} className="input" type="search" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Find by name or address" aria-keyshortcuts="/" />
                <kbd aria-hidden="true">/</kbd>
              </label>
              {archivedCount > 0 && (
                <div className="pl-filter" role="group" aria-label="Show">
                  <button type="button" className="btn btn-small" aria-pressed={!viewingArchived} onClick={() => setShowArchived(false)}>Active <span className="pl-filter-n">{activeCount}</span></button>
                  <button type="button" className="btn btn-small" aria-pressed={viewingArchived} onClick={() => setShowArchived(true)}>Archived <span className="pl-filter-n">{archivedCount}</span></button>
                </div>
              )}
            </div>

            {shown.length === 0 ? (
              <p className="pl-note" role="status">
                {query.trim() ? <>No {viewingArchived ? "archived " : ""}projects match “{query.trim()}”.</> : "Every project is archived. Show Archived to bring one back."}
              </p>
            ) : (
              <div className="pl-scroll">
                <table className="pl-table">
                  <thead>
                    <tr>
                      <th scope="col">Project</th>
                      <th scope="col" className="pl-num">Documents</th>
                      <th scope="col">To check</th>
                      <th scope="col">Last filing</th>
                      <th scope="col"><span className="sr-only">Actions</span></th>
                    </tr>
                  </thead>
                  <tbody ref={table} onKeyDown={onRowsKey}>
                    {shown.map((p) => (
                      <ProjectRow
                        key={p.id}
                        project={p}
                        renaming={renaming === p.id}
                        onOpen={() => onOpen(p.id)}
                        onRename={() => setRenaming(p.id)}
                        onRenamed={(name) => void rename(p, name)}
                        onRenameCancel={() => setRenaming(null)}
                        onArchive={(value) => void archive(p, value)}
                        onDelete={() => setConfirm(p)}
                      />
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </>
        )}
      </div>

      {confirm && (
        <ConfirmDialog
          title={`Delete ${confirm.name}?`}
          message="This permanently removes the project and everything in it: documents, files, the cost plan, specifications, procurement and issued records. This cannot be undone. Archive keeps it instead."
          confirmLabel="Delete project"
          busy={deleting}
          busyLabel="Deleting…"
          onConfirm={() => void remove()}
          onCancel={() => {
            if (!deleting) setConfirm(null);
          }}
        />
      )}
    </div>
  );
}

// Page-level action in the title band: name the project and create it in one
// step. The field is always there, so starting a job is one type and Enter.
function NewProject({ onCreated, onSignedOut, first }: { onCreated: (id: string) => void; onSignedOut: () => void; first: boolean }) {
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const input = useRef<HTMLInputElement>(null);

  // A first project starts with the cursor in the field.
  useEffect(() => {
    if (first) input.current?.focus();
  }, [first]);

  async function create(e: FormEvent) {
    e.preventDefault();
    const value = name.trim();
    if (!value) {
      setError("Give the project a name.");
      input.current?.focus();
      return;
    }
    setBusy(true);
    setError("");
    try {
      const { id } = await api.createProject(value);
      onCreated(id);
    } catch (err) {
      setBusy(false);
      if (err instanceof ApiError && err.status === 401) return onSignedOut();
      setError(err instanceof Error ? err.message : "Couldn't create the project.");
    }
  }

  return (
    <form className="pl-new" onSubmit={create} noValidate>
      <label htmlFor="new-project" className="sr-only">New project name or site address</label>
      <input
        id="new-project"
        ref={input}
        className="input"
        placeholder="Site address or name"
        value={name}
        maxLength={200}
        onChange={(e) => setName(e.target.value)}
        aria-invalid={error ? "true" : undefined}
        aria-describedby={error ? "new-project-error" : undefined}
      />
      <GroundButton type="submit" primary busy={busy} disabled={busy}>Create project</GroundButton>
      {error && (
        <span id="new-project-error" className="error-text pl-new-error" role="alert">
          {error}
        </span>
      )}
    </form>
  );
}

type RowProps = {
  project: Project;
  renaming: boolean;
  onOpen: () => void;
  onRename: () => void;
  onRenamed: (name: string) => void;
  onRenameCancel: () => void;
  onArchive: (archived: boolean) => void;
  onDelete: () => void;
};

function ProjectRow({ project: p, renaming, onOpen, onRename, onRenamed, onRenameCancel, onArchive, onDelete }: RowProps) {
  const checks = p.sources_differ ?? 0;
  const docs = p.documents ?? 0;
  const last = p.last_filed ? whenLabel(p.last_filed) : "";
  return (
    <tr
      className="pl-row"
      data-archived={p.archived || undefined}
      onClick={(e) => {
        // The whole row opens the project, except its menu and rename field.
        if (renaming || (e.target as HTMLElement).closest(".pl-menu, a, input")) return;
        onOpen();
      }}
    >
      <th scope="row" className="pl-project">
        {renaming ? (
          <RenameField name={p.name} onSave={onRenamed} onCancel={onRenameCancel} />
        ) : (
          <a
            className="pl-open"
            href={`/projects/${p.id}`}
            onClick={(e) => {
              e.preventDefault();
              onOpen();
            }}
          >
            {p.name}
          </a>
        )}
        <span className={p.address ? "pl-address" : "pl-address is-unknown"}>{p.address || "Site address not stated"}</span>
        {/* Narrow screens fold the figures under the address. */}
        <span className="pl-compact">
          {docs} {docs === 1 ? "document" : "documents"}
          {checks > 0 && <> · <span className="pl-check-text">{checks} to check</span></>}
          {last && <> · {last}</>}
        </span>
      </th>
      <td className="pl-num">{docs}</td>
      <td>
        {checks > 0 ? (
          <span className="pl-check" title="Profile values whose sources differ">
            <span aria-hidden="true">⚠</span> {checks}
          </span>
        ) : (
          <span className="pl-none" aria-label="None">—</span>
        )}
      </td>
      <td className="pl-when">
        {p.last_filed ? <time dateTime={p.last_filed} title={fullDate(p.last_filed)}>{last}</time> : <span className="pl-none">Nothing filed</span>}
      </td>
      <td className="pl-actions">
        <RowMenu name={p.name} archived={!!p.archived} onRename={onRename} onArchive={onArchive} onDelete={onDelete} />
      </td>
    </tr>
  );
}

function RenameField({ name, onSave, onCancel }: { name: string; onSave: (name: string) => void; onCancel: () => void }) {
  const [value, setValue] = useState(name);
  const done = useRef(false);
  const input = useRef<HTMLInputElement>(null);
  useEffect(() => input.current?.select(), []);
  const finish = (save: boolean) => {
    if (done.current) return;
    done.current = true;
    if (save) onSave(value);
    else onCancel();
  };
  return (
    <input
      ref={input}
      className="pl-rename"
      aria-label="Project name"
      value={value}
      maxLength={200}
      onChange={(e) => setValue(e.target.value)}
      onBlur={() => finish(true)}
      onKeyDown={(e) => {
        if (e.key === "Enter") finish(true);
        if (e.key === "Escape") finish(false);
      }}
    />
  );
}

function RowMenu({ name, archived, onRename, onArchive, onDelete }: { name: string; archived: boolean; onRename: () => void; onArchive: (archived: boolean) => void; onDelete: () => void }) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!open) return;
    root.current?.querySelector<HTMLElement>('[role="menuitem"]')?.focus();
    const close = (e: PointerEvent) => {
      if (!root.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("pointerdown", close);
    return () => document.removeEventListener("pointerdown", close);
  }, [open]);

  const pick = (action: () => void) => () => {
    setOpen(false);
    action();
  };

  function onKey(e: KeyboardEvent<HTMLDivElement>) {
    if (!open) return;
    if (e.key === "Escape") {
      e.stopPropagation();
      setOpen(false);
      trigger.current?.focus();
      return;
    }
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      e.stopPropagation();
      const items = Array.from(root.current?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? []);
      const at = items.indexOf(document.activeElement as HTMLElement);
      items[(at + (e.key === "ArrowDown" ? 1 : -1) + items.length) % items.length]?.focus();
    }
  }

  return (
    <div className="pl-menu" ref={root} onKeyDown={onKey}>
      <button ref={trigger} type="button" className="pl-menu-button" aria-label={`Actions for ${name}`} aria-haspopup="menu" aria-expanded={open} onClick={() => setOpen((v) => !v)}>
        <svg viewBox="0 0 20 20" width="18" height="18" aria-hidden="true"><circle cx="4.5" cy="10" r="1.6" /><circle cx="10" cy="10" r="1.6" /><circle cx="15.5" cy="10" r="1.6" /></svg>
      </button>
      {open && (
        <div className="pl-menu-pop" role="menu" aria-label={`Actions for ${name}`}>
          <button type="button" role="menuitem" className="account-item" onClick={pick(onRename)}>Rename</button>
          <button type="button" role="menuitem" className="account-item" onClick={pick(() => onArchive(!archived))}>{archived ? "Restore" : "Archive"}</button>
          <button type="button" role="menuitem" className="account-item pl-danger" onClick={pick(onDelete)}>Delete project…</button>
        </div>
      )}
    </div>
  );
}
