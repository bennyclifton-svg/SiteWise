import { Pending } from "./Activity";
// Shell: sign in, choose or create a project, then file.
// Routing is two paths, so it is done with the History API directly.

import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { api, ApiError, type Catalog, type Project as ProjectRow } from "./api";
import { ConfirmDialog } from "./ConfirmDialog";
import { SignOutButton } from "./SignOutButton";
import { ThemeButton } from "./ThemeButton";
import { Wordmark } from "./Wordmark";
import { Project } from "./Project";
import { ShellPreview } from "./ShellPreview";
import { SignIn } from "./SignIn";

type Route = { name: "home" } | { name: "project"; id: string };

function parseRoute(path: string): Route {
  const m = path.match(/^\/projects\/([0-9a-fA-F-]{36})\/?$/);
  return m ? { name: "project", id: m[1] } : { name: "home" };
}

type Auth = "checking" | "in" | "out";

export function App() {
  const [auth, setAuth] = useState<Auth>("checking");
  const [authError, setAuthError] = useState("");
  const [route, setRoute] = useState<Route>(() => parseRoute(location.pathname));
  const [catalog, setCatalog] = useState<Catalog | null>(null);
  const [projectsError, setProjectsError] = useState("");
  const [catalogError, setCatalogError] = useState(false);
  const [projects, setProjects] = useState<ProjectRow[] | null>(null);
  const sessionGeneration = useRef(0);

  const navigate = useCallback((path: string) => {
    history.pushState(null, "", path);
    setRoute(parseRoute(path));
    window.scrollTo(0, 0);
  }, []);

  useEffect(() => {
    const onPop = () => setRoute(parseRoute(location.pathname));
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, []);

  // An invite link carries its token in the fragment, which never reaches
  // server logs. It is consumed once and removed from the address bar, on
  // first load or when the link is pasted into an open tab.
  useEffect(() => {
    const consume = (): boolean => {
      const m = location.hash.match(/token=([0-9a-fA-F]+)/);
      if (!m) return false;
      history.replaceState(null, "", location.pathname + location.search);
      api.signIn(m[1]).then(
        () => {
          setAuthError("");
          setAuth("in");
        },
        () => {
          setAuthError("That sign-in link has expired or was already used. Sign in with your email and password.");
          setAuth("out");
        },
      );
      return true;
    };
    const onHash = () => void consume();
    window.addEventListener("hashchange", onHash);
    // A password reset link opens the sign-in panel in reset mode.
    if (/reset=/.test(location.hash)) {
      setAuth("out");
    } else if (!consume()) {
      api.checkSession().then(
        () => setAuth("in"),
        () => setAuth("out"),
      );
    }
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  const signedOut = useCallback(() => {
    sessionGeneration.current++;
    setProjects(null);
    setCatalog(null);
    setAuthError("Your session has ended. Sign in again.");
    setAuth("out");
  }, []);

  useEffect(() => {
    const done = () => {
      sessionGeneration.current++;
      setProjects(null);
      setCatalog(null);
      setProjectsError("");
      setCatalogError(false);
      setAuthError("");
      setAuth("out");
      navigate("/");
    };
    window.addEventListener("sitewise:signed-out", done);
    return () => window.removeEventListener("sitewise:signed-out", done);
  }, [navigate]);

  const loadProjects = useCallback(() => {
    const generation = sessionGeneration.current;
    setProjectsError("");
    api.projects().then(value => { if (generation === sessionGeneration.current) setProjects(value); }, (e: unknown) => {
      if (generation !== sessionGeneration.current) return;
      setProjectsError("Could not load projects. Check your connection and retry.");
      if (e instanceof ApiError && e.status === 401) signedOut();
    });
  }, [signedOut]);

  const loadCatalog = useCallback(() => {
    const generation = sessionGeneration.current;
    setCatalogError(false);
    api.catalog().then(value => { if (generation === sessionGeneration.current) setCatalog(value); }, () => { if (generation === sessionGeneration.current) setCatalogError(true); });
  }, []);

  useEffect(() => {
    if (auth !== "in") return;
    loadProjects();
    loadCatalog();
  }, [auth, loadProjects, loadCatalog]);

  if (location.hash === "#shell-preview") return <ShellPreview />;
  if (auth === "checking") return null;
  if (auth === "out") return <SignIn initialError={authError} onSignedIn={() => setAuth("in")} />;

  return (
    <>
      <a className="sr-only skip-link" href="#main">
        Skip to content
      </a>
      {catalogError && <div className="app-load-error" role="alert">Could not load filing choices. <button className="btn" onClick={loadCatalog}>Retry choices</button></div>}
      {route.name === "project" ? (
        <Project
          key={route.id}
          onRenamed={(id, name) => setProjects(current => current?.map(p => p.id === id ? { ...p, name } : p) ?? null)}
          projectId={route.id}
          catalog={catalog}
          projects={projects}
          navigate={navigate}
          onSignedOut={signedOut}
          onHome={() => navigate("/")}
        />
      ) : (
        <>
          <header className="bar home-bar">
            <a
              className="bar-logo-link"
              href="/"
              aria-label="SiteWise, all projects"
              onClick={(e) => {
                e.preventDefault();
                navigate("/");
              }}
            >
              <Wordmark height={28} />
            </a>
            <span className="bar-spacer" />
            <ThemeButton />
            <SignOutButton />
          </header>
          <div id="main">
            <Projects
              projects={projects}
              loadError={projectsError}
              onRetry={loadProjects}
              onOpen={(id) => navigate(`/projects/${id}`)}
              onCreated={(id) => {
                loadProjects();
                navigate(`/projects/${id}`);
              }}
              onDeleted={(id) => setProjects((cur) => cur?.filter((p) => p.id !== id) ?? cur)}
              onSignedOut={signedOut}
            />
          </div>
        </>
      )}
    </>
  );
}

function Projects({
  projects,
  loadError,
  onRetry,
  onOpen,
  onCreated,
  onDeleted,
  onSignedOut,
}: {
  projects: ProjectRow[] | null;
  loadError: string;
  onRetry: () => void;
  onOpen: (id: string) => void;
  onCreated: (id: string) => void;
  onDeleted: (id: string) => void;
  onSignedOut: () => void;
}) {
  const [name, setName] = useState("");
  const [query, setQuery] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [confirm, setConfirm] = useState<ProjectRow | null>(null);
  const [deleting, setDeleting] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  async function remove() {
    if (!confirm) return;
    setDeleting(true);
    setError("");
    try {
      await api.deleteProject(confirm.id);
      onDeleted(confirm.id);
      setConfirm(null);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) return onSignedOut();
      setError(err instanceof Error ? err.message : "Couldn't delete the project.");
      setConfirm(null);
    } finally {
      setDeleting(false);
    }
  }

  async function create(e: FormEvent) {
    e.preventDefault();
    const value = name.trim();
    if (!value) {
      setError("Give the project a name.");
      inputRef.current?.focus();
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
    <main className="projects">
      <h1>Projects</h1>
      <form className="stack-form" onSubmit={create} noValidate>
        <label htmlFor="new-project">New project</label>
        <div className="inline-form">
          <input
            id="new-project"
            ref={inputRef}
            className="input"
            placeholder="e.g. 14 Hale Street, Petersham"
            value={name}
            maxLength={200}
            onChange={(e) => setName(e.target.value)}
            aria-invalid={error ? "true" : undefined}
            aria-describedby={error ? "new-project-error" : undefined}
          />
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? <Pending delayed>Creating…</Pending> : "Create project"}
          </button>
        </div>
        {error && (
          <span id="new-project-error" className="error-text" role="alert">
            {error}
          </span>
        )}
      </form>
      {loadError && <div className="home-load-error" role="alert">{loadError} <button className="btn" onClick={onRetry}>Retry projects</button></div>}
      {projects && projects.length > 0 && <label className="project-search">Find a project<input className="input" type="search" value={query} onChange={e => setQuery(e.target.value)} placeholder="Search project names" /></label>}
      {projects === null && !loadError ? (
        <p className="drop-sub"><Pending delayed>Loading projects…</Pending></p>
      ) : projects?.length === 0 ? (
        <div className="empty">
          <strong>No projects yet.</strong>
          Name your first project above, then drop its documents in.
        </div>
      ) : (
        <ul className="project-list">
          {(projects ?? []).filter(p => p.name.toLowerCase().includes(query.toLowerCase())).map((p) => (
            <li key={p.id}>
              <a
                className="project-link"
                href={`/projects/${p.id}`}
                onClick={(e) => {
                  e.preventDefault();
                  onOpen(p.id);
                }}
              >
                <span className="project-name">{p.name}</span>
                <span className="project-open">Open</span>
              </a>
              <details className="project-actions">
                <summary aria-label={`Actions for ${p.name}`}>Actions</summary>
                <button type="button" className="btn" aria-label={`Delete ${p.name}`} onClick={e => { e.currentTarget.closest("details")?.removeAttribute("open"); setConfirm(p); }}>Delete project</button>
              </details>
            </li>
          ))}
        </ul>
      )}
      {projects && projects.length > 0 && !projects.some(p => p.name.toLowerCase().includes(query.toLowerCase())) && <p role="status">No projects match “{query}”.</p>}
      {confirm && (
        <ConfirmDialog
          title={`Delete ${confirm.name}?`}
          message="This permanently removes the project and everything in it: documents, files, the cost plan, specifications, procurement and issued records. This cannot be undone."
          confirmLabel="Delete project"
          busy={deleting}
          busyLabel="Deleting…"
          onConfirm={() => void remove()}
          onCancel={() => {
            if (!deleting) setConfirm(null);
          }}
        />
      )}
    </main>
  );
}
