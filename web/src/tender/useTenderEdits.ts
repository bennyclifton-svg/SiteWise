// Every edit to the tender tables goes through here: one call with the
// table's version, the answer replaces the table, and each change that can
// be undone is remembered in order. No edit calls a model.
import { useCallback, useRef, useState } from "react";
import { ApiError } from "../api";
import { submissionApi, type PriceTable } from "../submissionApi";
import { tenderApi, type TenderOp, type TenderTable } from "./tenderApi";

export interface TenderEdits {
  busy: boolean;
  /** Words for the manager: a refused edit, a reloaded table, a refused undo. */
  note: string;
  clearNote: () => void;
  canUndo: boolean;
  run: (ops: TenderOp[]) => Promise<boolean>;
  undo: () => Promise<void>;
  readAgain: (sub: string) => Promise<void>;
}

/** What is being dragged, shared by the two tables. The browser does not let
 * a drop target read the dragged data until the drop. */
// What is being dragged: a firm's lines, from a row (its key; "" for "Not
// broken down") and slot when it is one line of an open row. Dropped on
// another row's cell it moves there; on a slot of its own column it changes
// places. Lines out of the count only change places.
export const tenderDrag: { current: { sub: string; lines: string[]; row?: string; slot?: number; counted?: boolean } | null } = { current: null };

function words(e: unknown, fallback: string): string {
  if (!(e instanceof ApiError) || !e.message) return fallback;
  try {
    const body = JSON.parse(e.message) as { message?: string };
    if (body.message) return body.message;
  } catch { /* plain text */ }
  return e.message;
}

export function useTenderEdits(projectId: string, pkg: string, tender: TenderTable | undefined, setTable: (t: PriceTable) => void, onSignedOut: () => void): TenderEdits {
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState("");
  const [changes, setChanges] = useState<string[]>([]);
  const version = useRef(0);
  version.current = tender?.version ?? 0;

  const reload = useCallback(async () => setTable(await submissionApi.table(projectId, pkg)), [projectId, pkg, setTable]);

  const run = useCallback(async (ops: TenderOp[]) => {
    setBusy(true);
    try {
      const t = await tenderApi.ops(projectId, pkg, version.current, ops);
      setTable(t);
      const change = t.change;
      if (change) setChanges(c => [...c, change]);
      setNote("");
      return true;
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) { onSignedOut(); return false; }
      if (e instanceof ApiError && e.status === 409) {
        // Someone else changed the table: show theirs, and say so.
        await reload().catch(() => undefined);
        setNote("This table changed. It has been reloaded.");
        return false;
      }
      setNote(words(e, "Could not save. Try again."));
      return false;
    } finally {
      setBusy(false);
    }
  }, [projectId, pkg, setTable, onSignedOut, reload]);

  const undo = useCallback(async () => {
    const id = changes[changes.length - 1];
    if (!id) return;
    setBusy(true);
    try {
      await tenderApi.undo(projectId, id);
      setNote("");
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
      setNote(words(e, "Could not undo. Try again."));
    } finally {
      // Undone, or refused for good: either way that change is spent.
      setChanges(c => c.slice(0, -1));
      await reload().catch(() => undefined);
      setBusy(false);
    }
  }, [changes, projectId, onSignedOut, reload]);

  const readAgain = useCallback(async (sub: string) => {
    setBusy(true);
    try {
      setTable(await tenderApi.readAgain(projectId, sub));
      setNote("");
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
      setNote(words(e, "Could not start the reading. Try again."));
    } finally {
      setBusy(false);
    }
  }, [projectId, setTable, onSignedOut]);

  return { busy, note, clearNote: () => setNote(""), canUndo: changes.length > 0, run, undo, readAgain };
}
