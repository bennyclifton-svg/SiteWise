// Development only: whether the local server is up and whether the running
// build is behind the source tree. /dev/build exists only when serve runs
// with -dev-login, so in production this renders nothing.

import { useEffect, useState } from "react";

interface Build {
  server_stale: boolean;
  server_changed?: string;
  web_stale: boolean;
  web_changed?: string;
}

type State = { kind: "unknown" } | { kind: "absent" } | { kind: "offline" } | { kind: "up"; build: Build };

const POLL_MS = 5000;

export function DevStatus() {
  const [state, setState] = useState<State>({ kind: "unknown" });

  useEffect(() => {
    let live = true;
    const check = async () => {
      try {
        const res = await fetch("/dev/build", { cache: "no-store" });
        if (!live) return;
        // A server without -dev-login answers with the app shell: not dev.
        if (!res.ok || !res.headers.get("content-type")?.includes("json")) {
          setState((s) => (s.kind === "up" || s.kind === "offline" ? { kind: "offline" } : { kind: "absent" }));
          return;
        }
        setState({ kind: "up", build: (await res.json()) as Build });
      } catch {
        if (live) setState((s) => (s.kind === "absent" ? s : { kind: "offline" }));
      }
    };
    check();
    const id = setInterval(check, POLL_MS);
    return () => {
      live = false;
      clearInterval(id);
    };
  }, []);

  if (state.kind === "unknown" || state.kind === "absent") return null;
  if (state.kind === "offline") {
    return (
      <div className="devstatus" data-state="offline" role="status">
        <span className="devstatus-dot" aria-hidden="true" />
        <span>
          Dev server offline
          <small>Run tools\dev.ps1</small>
        </span>
      </div>
    );
  }
  const { build } = state;
  const stale = build.server_stale || build.web_stale;
  return (
    <div className="devstatus" data-state={stale ? "stale" : "up"} role="status">
      <span className="devstatus-dot" aria-hidden="true" />
      <span>
        {stale ? "Dev server behind code" : "Dev server running"}
        {build.web_stale ? (
          <small title={build.web_changed}>UI changed: rerun dev.ps1 -Build</small>
        ) : build.server_stale ? (
          <small title={build.server_changed}>Backend changed: rerun dev.ps1</small>
        ) : null}
      </span>
    </div>
  );
}
