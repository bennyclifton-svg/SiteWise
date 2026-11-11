// Shell: sign in, choose or create a project, then file.
// Routing is two paths, so it is done with the History API directly.

import { useCallback, useEffect, useRef, useState } from "react";
import { api, ApiError, type Catalog, type Project as ProjectRow } from "./api";
import { Account } from "./Account";
import { Project } from "./Project";
import { ProjectsPage } from "./ProjectsPage";
import { ShellPreview } from "./ShellPreview";
import { SignIn } from "./SignIn";
import { TracePage } from "./TracePage";

type Route = { name: "home" } | { name: "account" } | { name: "project"; id: string };

function parseRoute(path: string): Route {
  if (/^\/account\/?$/.test(path)) return { name: "account" };
  const m = path.match(/^\/projects\/([0-9a-fA-F-]{36})\/?$/);
  return m ? { name: "project", id: m[1] } : { name: "home" };
}

// An email confirmation link carries its token in the fragment. It is used
// once, removed from the address bar, and works signed in or out.
function takeVerifyToken(): string {
  const m = location.hash.match(/verify=([0-9a-fA-F]+)/);
  if (!m) return "";
  history.replaceState(null, "", location.pathname + location.search);
  return m[1];
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
  const [notice, setNotice] = useState("");
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

  // The Account link sits inside shared header controls, so it asks the
  // shell to navigate rather than holding the router itself.
  useEffect(() => {
    const onNavigate = (e: Event) => navigate(String((e as CustomEvent).detail));
    window.addEventListener("sitewise:navigate", onNavigate);
    return () => window.removeEventListener("sitewise:navigate", onNavigate);
  }, [navigate]);

  useEffect(() => {
    const token = takeVerifyToken();
    if (!token) return;
    api.verifyEmail(token).then(
      () => setNotice("Your email address is confirmed. You can upload documents now."),
      (e: unknown) => setNotice(e instanceof ApiError && e.message ? e.message : "That confirmation link did not work. Send a new one from the Account page."),
    );
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
    // A password reset or team invite link opens the sign-in panel.
    if (/(reset|invite)=/.test(location.hash)) {
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
      {notice && (
        <div className="app-notice" role="status">
          {notice} <button className="link-button" onClick={() => setNotice("")}>Dismiss</button>
        </div>
      )}
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
        <TracePage onHome={() => navigate("/")}>
          {route.name === "account" ? <Account onSignedOut={signedOut} /> : <ProjectsPage
            projects={projects}
            loadError={projectsError}
            onRetry={loadProjects}
            onOpen={(id) => navigate(`/projects/${id}`)}
            onCreated={(id) => {
              loadProjects();
              navigate(`/projects/${id}`);
            }}
            onChanged={(id, patch) => setProjects((cur) => cur?.map((p) => (p.id === id ? { ...p, ...patch } : p)) ?? cur)}
            onDeleted={(id) => setProjects((cur) => cur?.filter((p) => p.id !== id) ?? cur)}
            onSignedOut={signedOut}
          />}
        </TracePage>
      )}
    </>
  );
}

