// #shell-preview opens a real project in the cockpit. The profile, scope and
// document register are the same components as the project page.

import { useEffect, useState } from "react";
import { api, type Catalog, type Project as ProjectRow } from "./api";
import { Project } from "./Project";

export function ShellPreview() {
  const [phase, setPhase] = useState<"checking" | "out" | "empty" | "ready">("checking");
  const [projects, setProjects] = useState<ProjectRow[]>([]);
  const [catalog, setCatalog] = useState<Catalog | null>(null);
  const [projectId, setProjectId] = useState<string | null>(null);

  useEffect(() => {
    let live = true;
    api.checkSession().then(
      async () => {
        const [list, cat] = await Promise.all([api.projects(), api.catalog().catch(() => null)]);
        if (!live) return;
        setProjects(list);
        setCatalog(cat);
        const named = list.find((p) => p.name.toLowerCase() === "custom home 2");
        setProjectId((named ?? list[0])?.id ?? null);
        setPhase(list.length ? "ready" : "empty");
      },
      () => {
        if (live) setPhase("out");
      },
    );
    return () => {
      live = false;
    };
  }, []);

  if (phase === "checking") return null;
  if (phase !== "ready" || !projectId) {
    return <p className="reg-empty">{phase === "out" ? "Sign in to open the project profile." : "No project to open."}</p>;
  }

  return (
    <Project
      key={projectId}
      projectId={projectId}
      catalog={catalog}
      projects={projects}
      navigate={(path) => {
        const m = path.match(/\/projects\/([0-9a-fA-F-]{36})/);
        if (m) setProjectId(m[1]);
      }}
      onSignedOut={() => setPhase("out")}
      onHome={() => undefined}
    />
  );
}
