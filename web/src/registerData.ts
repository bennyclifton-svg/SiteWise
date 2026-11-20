import { useCallback, useEffect, useState } from "react";
import { ApiError, request } from "./api";
import { deliveryApi, type AlsoOpen, type DeliveryRecord, type OpCall, type RegisterView } from "./deliveryApi";
import { opMessage } from "./registerKinds";

// One read of the register for the two places that show it (owner,
// 20 November 2026): the list in the right-hand panel and the opened row's
// sheet in the middle. Both write through POST /ops (area delivery), so each
// action has one Undo, and both read the same rows, so an edit in one shows
// in the other. No read or write here calls Jev or the model.

export interface RegisterData {
  /** Every row; the list filters by system itself, so the open sheet keeps its row. */
  view: RegisterView | null;
  /** The also-open line for the System filter, from the filtered read. */
  alsoOpen?: AlsoOpen;
  system: string;
  setSystem: (system: string) => void;
  error: string;
  setError: (error: string) => void;
  busy: boolean;
  /** The last change, for its Undo line, and where it was made. */
  last: { change: string; label: string; from: "list" | "sheet" } | null;
  clearLast: () => void;
  undo: () => void;
  run: (ops: OpCall[], label: string, from?: "list" | "sheet") => Promise<boolean>;
  reload: () => void;
  rowOf: (id: string) => DeliveryRecord | undefined;
  /** Add to RFP writes the RFP draft's setting, which has no Undo yet. */
  addToRFP: (row: DeliveryRecord) => Promise<void>;
  /** Read conditions: Go checks the notice and queues the background read. */
  readNotice: (documentId: string) => Promise<boolean>;
}

/** The register's rows, read once `enabled` and again on each tick or write. */
export function useRegisterData(projectId: string, tick: number, enabled: boolean, onSignedOut: () => void): RegisterData {
  const [view, setView] = useState<RegisterView | null>(null);
  const [alsoOpen, setAlsoOpen] = useState<AlsoOpen | undefined>();
  const [system, setSystem] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [n, setN] = useState(0);
  const [last, setLast] = useState<RegisterData["last"]>(null);
  const reload = useCallback(() => setN(x => x + 1), []);

  useEffect(() => {
    if (!enabled) return;
    let stopped = false;
    deliveryApi.view(projectId).then(v => { if (!stopped) { setView(v); setError(""); } }).catch(e => {
      if (stopped) return;
      if (e instanceof ApiError && e.status === 401) onSignedOut();
      else setError("The register could not be read. Reload to try again.");
    });
    return () => { stopped = true; };
  }, [projectId, tick, n, enabled, onSignedOut]);

  // The also-open line is Go's for one system; it is read only while that
  // filter is chosen.
  useEffect(() => {
    if (!enabled || !system) { setAlsoOpen(undefined); return; }
    let stopped = false;
    deliveryApi.view(projectId, system).then(v => { if (!stopped) setAlsoOpen(v.also_open); }).catch(() => { if (!stopped) setAlsoOpen(undefined); });
    return () => { stopped = true; };
  }, [projectId, system, tick, n, enabled]);

  // One batch, one Undo. A 409 means the row changed under the PM: the
  // register is read again and the PM tries once more.
  const run = useCallback(async (ops: OpCall[], label: string, from: "list" | "sheet" = "list") => {
    setBusy(true); setError("");
    try {
      const out = await deliveryApi.ops(projectId, ops);
      setLast(out.change_id ? { change: out.change_id, label, from } : null);
      setN(x => x + 1);
      return true;
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) onSignedOut();
      else if (e instanceof ApiError && e.status === 409) setError("This row changed while you were editing. The register was read again; try once more.");
      else setError(e instanceof ApiError && e.message ? opMessage(e.message) : "Not saved. Try again.");
      setN(x => x + 1);
      return false;
    } finally { setBusy(false); }
  }, [projectId, onSignedOut]);

  const undo = useCallback(() => {
    if (!last) return;
    const c = last.change;
    setLast(null);
    void deliveryApi.undo(projectId, c).then(() => setN(x => x + 1)).catch(() => setError("Can't undo: this has changed since."));
  }, [projectId, last]);

  const addToRFP = useCallback(async (r: DeliveryRecord) => {
    setBusy(true); setError("");
    try {
      await request("POST", `/projects/${projectId}/ops`, { area: "reports", ops: [{ op: "reports.add_scope_line", fields: { item: r.id } }] });
      setLast(null); setN(x => x + 1);
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) onSignedOut();
      else setError(e instanceof ApiError && e.message ? opMessage(e.message) : "Not added. Try again.");
    } finally { setBusy(false); }
  }, [projectId, onSignedOut]);

  const readNotice = useCallback(async (documentId: string) => {
    setBusy(true); setError("");
    try { await deliveryApi.readNotice(projectId, documentId); setN(x => x + 1); return true; }
    catch (e) {
      if (e instanceof ApiError && e.status === 401) onSignedOut();
      else setError(e instanceof ApiError && e.message ? e.message : "The conditions could not be read. Try again.");
      return false;
    } finally { setBusy(false); }
  }, [projectId, onSignedOut]);

  const rowOf = useCallback((id: string) => view?.items.find(r => r.id === id), [view]);
  return { view, alsoOpen, system, setSystem, error, setError, busy, last, clearLast: () => setLast(null), undo, run, reload, rowOf, addToRFP, readNotice };
}

/** delivery.edit for each row, with the version the screen holds. */
export function editOps(rows: (DeliveryRecord | undefined)[], fields: Record<string, unknown>): OpCall[] {
  return rows.flatMap(r => r ? [{ op: "delivery.edit", fields: { item: r.id, version: r.version, ...fields } }] : []);
}
