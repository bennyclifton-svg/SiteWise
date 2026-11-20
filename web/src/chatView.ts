import { useEffect, useRef } from "react";
import type { WorkbenchView } from "./LeftNav";

// One registry between the chat strip and the tables and dialogs it can move
// (docs/plans/2026-11-18-chat-agency.md, Task 2). A mounted table offers its
// setters here; the strip looks a table up by name when a `view` event
// arrives, and reads the selection from here when it sends a turn. Wiring
// each table into Chat.tsx by hand would grow with every plan.
//
// Every setter only sets state: none moves focus, so an open cell editor
// keeps it while the selection changes under it.

/** The tables chat can name. They match viewTables in internal/httpapi/chat_screen.go. */
export type ChatTableName = "documents" | "cost_lines" | "program" | "procurement" | "register" | "messages" | "variations" | "claims" | "claim_rows";

/** Where a table lives: a workbench view, or the repository panel beside every view. */
export type ChatPlace = WorkbenchView | "documents";

export interface ChatTable {
  place: ChatPlace;
  /** Selects exactly these rows (ids not in the table are ignored). */
  select(ids: readonly string[]): void;
  /** Hides every other row under a "Showing N … · Clear" line; `shown` is the named filter's words, when one chose them. */
  only(ids: readonly string[], shown?: string): void;
  /** Puts the table back: all rows, nothing hidden. */
  clear(): void;
  /** Opens one row as the table's own Open would. */
  open?(id: string): void;
  /** The ids selected now, in the table's order. */
  selected(): readonly string[];
}

/** Shown before the project report opens: the right-hand panel's Register side, whose band holds "Project report" (monthly plan, Task 7). The reader itself opens in the middle panel. */
export const REPORT_PLACE: WorkbenchView = "register";

export interface ChatDialog {
  open(fields: Readonly<Record<string, string>>): void;
}

interface Entry { table: ChatTable; touched: number }

const tables = new Map<string, Entry>();
const dialogs = new Map<string, ChatDialog>();
// Brings a table on screen when it is not mounted (a tab of its view).
const hosts = new Map<string, () => void>();
const listeners = new Set<() => void>();
let clock = 0;

function changed() {
  for (const l of [...listeners]) l();
}

export function registerTable(name: ChatTableName, table: ChatTable): () => void {
  const entry: Entry = { table, touched: tables.get(name)?.touched ?? 0 };
  tables.set(name, entry);
  changed();
  return () => { if (tables.get(name) === entry) tables.delete(name); };
}

/** A dialog's name says what it is for, e.g. "award:<package id>". */
export function registerDialog(name: string, dialog: ChatDialog): () => void {
  dialogs.set(name, dialog);
  changed();
  return () => { if (dialogs.get(name) === dialog) dialogs.delete(name); };
}

export function registerHost(name: ChatTableName, show: () => void): () => void {
  hosts.set(name, show);
  return () => { if (hosts.get(name) === show) hosts.delete(name); };
}

/** Marks a table's selection as the newest, so "these" means its rows. */
export function noteSelection(name: ChatTableName) {
  const e = tables.get(name);
  if (e) e.touched = ++clock;
}

export const tableOf = (name: string) => tables.get(name)?.table;
export const dialogOf = (name: string) => dialogs.get(name);
export const hostOf = (name: string) => hosts.get(name);

/** Resolves once `get` finds something, or with undefined after `ms`. */
export function whenReady<T>(get: () => T | undefined, ms = 4000): Promise<T | undefined> {
  const now = get();
  if (now) return Promise.resolve(now);
  return new Promise(resolve => {
    const done = (v: T | undefined) => { listeners.delete(check); clearTimeout(timer); resolve(v); };
    const check = () => { const v = get(); if (v) done(v); };
    const timer = setTimeout(() => done(undefined), ms);
    listeners.add(check);
  });
}

export const MAX_SCREEN_IDS = 200;

export interface ScreenSelection { table: ChatTableName; ids: string[]; count: number }

/** The newest non-empty selection among the tables on screen: at most 200 ids and the full count. */
export function screenSelection(view: WorkbenchView): ScreenSelection | undefined {
  let best: [string, Entry] | undefined;
  for (const [name, e] of tables) {
    if (e.table.place !== view && e.table.place !== "documents") continue;
    if (e.table.selected().length === 0) continue;
    if (!best || e.touched > best[1].touched) best = [name, e];
  }
  if (!best) return undefined;
  const ids = best[1].table.selected();
  return { table: best[0] as ChatTableName, ids: ids.slice(0, MAX_SCREEN_IDS), count: ids.length };
}

/**
 * Registers a mounted table for as long as it is mounted. The handlers are
 * read through a ref, so they always see the table's current rows without
 * registering again on every render. `selected` is the table's selection,
 * watched so the newest one is what "these" means.
 */
export function useChatTable(name: ChatTableName, place: ChatPlace, handlers: Omit<ChatTable, "place" | "selected">, selected: readonly string[], ready = true) {
  const latest = useRef({ handlers, selected });
  latest.current = { handlers, selected };
  // A table registers once its rows have loaded: a move that arrives
  // earlier waits for it rather than acting on an empty table.
  useEffect(() => ready ? registerTable(name, {
    place,
    select: ids => latest.current.handlers.select(ids),
    only: (ids, shown) => latest.current.handlers.only(ids, shown),
    clear: () => latest.current.handlers.clear(),
    open: id => latest.current.handlers.open?.(id),
    selected: () => latest.current.selected,
  }) : undefined, [name, place, ready]);
  const key = selected.join(",");
  useEffect(() => { if (key) noteSelection(name); }, [name, key]);
}

/** Words for "Showing N …" on each table. */
export const ROW_WORDS: Record<ChatTableName, [string, string]> = {
  documents: ["document", "documents"],
  cost_lines: ["cost line", "cost lines"],
  program: ["Program row", "Program rows"],
  procurement: ["package", "packages"],
  register: ["register row", "register rows"],
  messages: ["message", "messages"],
  variations: ["variation", "variations"],
  claims: ["claim", "claims"],
  claim_rows: ["claim row", "claim rows"],
};

/** The "Showing …" line. A named filter's words (Go writes them to follow the count) replace "chosen in chat". */
export function showingLine(name: ChatTableName, n: number, shown?: string) {
  const [one, many] = ROW_WORDS[name];
  return `Showing ${n} ${n === 1 ? one : many} ${shown || "chosen in chat"}.`;
}

/** Scrolls a row into view unless the PM is typing in an editor, so a view never pulls the page from under a cell being edited. */
export function revealQuietly(row: Element | null | undefined) {
  const active = document.activeElement;
  if (!row || (active && active.closest("input,textarea,select,[contenteditable=true]"))) return;
  row.scrollIntoView({ block: "nearest", inline: "nearest" });
}
