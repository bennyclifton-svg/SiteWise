import { useEffect, useRef, useState } from "react";
import { ApiError } from "./api";
import { registerDialog } from "./chatView";
import { reportApi } from "./reportApi";

// The project report's one entry point (docs/plans/2026-11-17-monthly-report.md,
// Task 7): finds the project's report, or creates it, and asks the host
// panel to show it in the shared reader. The host keeps its own view state,
// as Procurement does for an RFP. A double click never makes two reports:
// the server returns the one there is.
export function ProjectReportButton({ projectId, onOpen, onSignedOut }: { projectId: string; onOpen: (reportId: string) => void; onSignedOut?: () => void }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const latest = useRef(onOpen);
  latest.current = onOpen;
  async function open() {
    setBusy(true); setError("");
    try {
      const existing = (await reportApi.list(projectId)).find(r => r.kind === "monthly");
      latest.current((existing ?? await reportApi.create(projectId, "", "monthly")).id);
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) onSignedOut?.();
      else setError("Could not open the project report. Try again.");
    } finally { setBusy(false); }
  }
  return <>
    <button type="button" className="btn btn-small" disabled={busy} aria-busy={busy} onClick={() => void open()}>Project report</button>
    {error && <span role="alert" className="report-muted">{error}</span>}
  </>;
}

/**
 * Lets chat open the project report in this host panel: a "report" dialog
 * from chat calls onOpen with the report's id (and the reader then opens
 * its issue form when asked; the PM presses Issue). The host calls it once,
 * whatever it is showing, so chat finds it from any of the host's views.
 */
export function useProjectReportDialog(onOpen: (reportId: string) => void) {
  const latest = useRef(onOpen);
  latest.current = onOpen;
  useEffect(() => registerDialog("report", { open: fields => { if (fields.report) latest.current(fields.report); } }), []);
}
