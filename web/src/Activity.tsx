import type { ReactNode } from "react";
import "./activity.css";

/** Motion describes active work; a clock means waiting, never progress. */
export function Activity({ state = "active", delayed = false }: { state?: "active" | "queued"; delayed?: boolean }) {
  return <span className="activity-icon" data-state={state} data-delayed={delayed || undefined} aria-hidden="true">
    {state === "queued"
      ? <svg viewBox="0 0 16 16" fill="none"><circle cx="8" cy="8" r="6" /><path d="M8 4.5V8l2.5 1.5" /></svg>
      : <svg viewBox="0 0 16 16" fill="none"><circle className="activity-track" cx="8" cy="8" r="6" /><path d="M8 2a6 6 0 0 1 6 6" /></svg>}
  </span>;
}

export function Pending({ children, queued = false, delayed = false }: { children: ReactNode; queued?: boolean; delayed?: boolean }) {
  return <span className="activity" data-state={queued ? "queued" : "active"}><Activity state={queued ? "queued" : "active"} delayed={delayed} /><span>{children}</span></span>;
}

export function ImportStatus({ status }: { status: string }) {
  if (status === "queued") return <Pending queued>Queued for reading</Pending>;
  if (status === "leased") return <Pending>Reading document…</Pending>;
  return <>{status === "done" ? "Reading complete" : status === "failed" ? "Reading failed" : status}</>;
}
