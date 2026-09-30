// Shell: sign in with an invite token, choose or create a project, then file.
// Routing is two paths, so it is done with the History API directly.

import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { api, ApiError, type Catalog, type Project as ProjectRow } from "./api";
import { Project } from "./Project";

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
  const [projects, setProjects] = useState<ProjectRow[] | null>(null);

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
      history.replaceState(null, "", location.pathname);
      api.signIn(m[1]).then(
        () => {
          setAuthError("");
          setAuth("in");
        },
        () => {
          setAuthError("That invite link has expired or was already used. Ask your project admin for a new one.");
          setAuth("out");
        },
      );
      return true;
    };
    const onHash = () => void consume();
    window.addEventListener("hashchange", onHash);
    if (!consume()) {
      api.checkSession().then(
        () => setAuth("in"),
        () => setAuth("out"),
      );
    }
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  const signedOut = useCallback(() => {
    setAuthError("Your session has ended. Sign in again with a new invite.");
    setAuth("out");
  }, []);

  const loadProjects = useCallback(() => {
    api.projects().then(setProjects, (e: unknown) => {
      if (e instanceof ApiError && e.status === 401) signedOut();
    });
  }, [signedOut]);

  useEffect(() => {
    if (auth !== "in") return;
    loadProjects();
    api.catalog().then(setCatalog, () => setCatalog(null));
  }, [auth, loadProjects]);

  if (auth === "checking") return null;
  if (auth === "out") return <SignIn initialError={authError} onSignedIn={() => setAuth("in")} />;

  return (
    <>
      <a className="sr-only" href="#main">
        Skip to content
      </a>
      <header className="bar">
        <a
          className="bar-logo-link"
          href="/"
          onClick={(e) => {
            e.preventDefault();
            navigate("/");
          }}
        >
          <img className="bar-logo" src="/sitewise-logo.png" alt="SiteWise, all projects" width="118" height="28" />
        </a>
        {route.name === "project" && projects && projects.length > 0 && (
          <>
            <span className="bar-divider" aria-hidden="true" />
            <label className="sr-only" htmlFor="project-switch">
              Project
            </label>
            <select
              id="project-switch"
              className="select"
              value={route.id}
              onChange={(e) => navigate(e.target.value ? `/projects/${e.target.value}` : "/")}
            >
              {!projects.some((p) => p.id === route.id) && <option value={route.id}>Unknown project</option>}
              {projects.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
              <option value="">All projects…</option>
            </select>
          </>
        )}
        <span className="bar-spacer" />
      </header>
      <div id="main">
        {route.name === "project" ? (
          <Project
            key={route.id}
            projectId={route.id}
            catalog={catalog}
            onSignedOut={signedOut}
            onHome={() => navigate("/")}
          />
        ) : (
          <Projects
            projects={projects}
            onOpen={(id) => navigate(`/projects/${id}`)}
            onCreated={(id) => {
              loadProjects();
              navigate(`/projects/${id}`);
            }}
            onSignedOut={signedOut}
          />
        )}
      </div>
    </>
  );
}

function SignIn({ initialError, onSignedIn }: { initialError: string; onSignedIn: () => void }) {
  const [token, setToken] = useState("");
  const [error, setError] = useState(initialError);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    const value = token.trim();
    if (!value) {
      setError("Paste the token from your invite email.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await api.signIn(value);
      onSignedIn();
    } catch (err) {
      setBusy(false);
      setError(
        err instanceof ApiError && err.status === 401
          ? "That token has expired or was already used. Ask your project admin for a new invite."
          : err instanceof Error
            ? err.message
            : "Sign in failed.",
      );
    }
  }

  return (
    <main className="entry">
      <div className="entry-panel">
        <img className="bar-logo" src="/sitewise-logo.png" alt="SiteWise" width="134" height="32" />
        <h1>Sign in with your invite</h1>
        <p>SiteWise is invite only. Open the link in your invite email, or paste its token here.</p>
        <form className="stack-form" onSubmit={submit} noValidate>
          <label htmlFor="token">Invite token</label>
          <input
            id="token"
            className="input"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            autoComplete="one-time-code"
            spellCheck={false}
            aria-invalid={error ? "true" : undefined}
            aria-describedby={error ? "token-error" : undefined}
          />
          {error && (
            <span id="token-error" className="error-text" role="alert">
              {error}
            </span>
          )}
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? "Signing in…" : "Sign in"}
          </button>
        </form>
      </div>
    </main>
  );
}

function Projects({
  projects,
  onOpen,
  onCreated,
  onSignedOut,
}: {
  projects: ProjectRow[] | null;
  onOpen: (id: string) => void;
  onCreated: (id: string) => void;
  onSignedOut: () => void;
}) {
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

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
      {projects === null ? (
        <p className="drop-sub">Loading projects…</p>
      ) : projects.length === 0 ? (
        <div className="empty" style={{ marginBottom: 28 }}>
          <strong>No projects yet.</strong>
          Name your first project below, then drop its documents in.
        </div>
      ) : (
        <ul className="project-list">
          {projects.map((p) => (
            <li key={p.id}>
              <a
                className="project-link"
                href={`/projects/${p.id}`}
                onClick={(e) => {
                  e.preventDefault();
                  onOpen(p.id);
                }}
              >
                {p.name}
                <span>Open</span>
              </a>
            </li>
          ))}
        </ul>
      )}
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
            {busy ? "Creating…" : "Create project"}
          </button>
        </div>
        {error && (
          <span id="new-project-error" className="error-text" role="alert">
            {error}
          </span>
        )}
      </form>
    </main>
  );
}
