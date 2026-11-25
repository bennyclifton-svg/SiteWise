import { useEffect, useState } from "react";
import { Pending } from "./Activity";
import { ApiError } from "./api";
import { reportApi } from "./reportApi";
import { Reports } from "./Reports";

// Report (owner, 25 November 2026): the project report as its own view on
// the left nav, under Correspondence. Opening it finds the project's report,
// or creates it: the server returns the one there is, so a second open
// never makes two. Every figure in it is Go's, from the screens' rollups
// (docs/plans/2026-11-17-monthly-report.md); nothing here calls Jev or the
// model.
export function ProjectReport({ projectId, reportId, onReport, tick, onSignedOut }: {
  projectId: string;
  /** The report on screen, once known; chat may name it. */
  reportId: string | null;
  onReport: (id: string) => void;
  tick: number;
  onSignedOut: () => void;
}) {
  const [error, setError] = useState("");
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    if (reportId) return;
    let stopped = false;
    setError("");
    reportApi.list(projectId)
      .then(async list => (list.find(r => r.kind === "monthly") ?? await reportApi.create(projectId, "", "monthly")).id)
      .then(id => { if (!stopped) onReport(id); })
      .catch(e => {
        if (stopped) return;
        if (e instanceof ApiError && e.status === 401) onSignedOut();
        else setError("The project report could not be opened.");
      });
    return () => { stopped = true; };
  }, [projectId, reportId, attempt, onReport, onSignedOut]);
  if (error) return <div className="banner" role="alert"><p>{error}</p><button type="button" className="btn btn-small" onClick={() => setAttempt(n => n + 1)}>Try again</button></div>;
  if (!reportId) return <p role="status"><Pending delayed>Opening the project report…</Pending></p>;
  return <Reports key={reportId} reportId={reportId} tick={tick} surface="report" onSignedOut={onSignedOut} />;
}
