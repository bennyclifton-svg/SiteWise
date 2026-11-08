import { Pending } from "./Activity";
import { FILED_DOCUMENT_TYPE } from "./documentDrag";
// The document register: one compact line per filing, sortable, opening the
// source and filing details in the centre workspace.

import { useEffect, useMemo, useRef, useState, type KeyboardEvent, type MouseEvent, type ReactNode, type HTMLAttributes } from "react";
import type { Catalog, Doc, ReadSetting } from "./api";
import { FIELDS, notFiledReason, type RowModel } from "./DocumentRow";
import { ConfirmDialog } from "./ConfirmDialog";
import { IconBin, IconEye, IconEyeOff, IconMore, IconOpen, IconRename, IconRetry, IconStored, IconTag, IconUndo, IconUpload } from "./icons";
import { OCRStatus, ocrStage } from "./OCRStatus";
import { documentActivity } from "./documentActivity";
import { clickRow, emptySelection, prune, selectAll as selectEveryRow, type Selection } from "./table/selection";
import { InlineCellEditor, type CommitKey } from "./table/InlineCellEditor";
import { RowMenu, type MenuItem, type RowMenuAt } from "./table/RowMenu";
import "./register-edit.css";

type Col = "number" | "title" | "revision" | "date" | "discipline";

interface Props {
  selection?: Selection;
  onSelectionChange?: React.Dispatch<React.SetStateAction<Selection>>;
  rows: RowModel[];
  catalog: Catalog | null;
  live: "live" | "reconnecting" | "offline";
  onCorrect: (docId: string, field: string, value: string) => Promise<void>;
  onAddFiles: () => void;
  /** Sets the profile reading setting for documents; resolves when saved. */
  onSetReading: (ids: string[], setting: ReadSetting) => Promise<void>;
  /** Show only documents the profile does not read. */
  notReadOnly: boolean;
  onClearFilter: () => void;
  /** Permanently deletes documents after the user confirmed; resolves true when done. */
  onDelete: (ids: string[]) => Promise<boolean>;
  /** A file is being dragged onto the page. Highlights this panel with the shared drop treatment. */
  fileOver?: boolean;
  openKey: string | null;
  onOpen: (key: string | null) => void;
}

const COLUMNS: { col: Col; label: string; className: string }[] = [
  { col: "number", label: "Doc No.", className: "reg-no" },
  { col: "title", label: "Title", className: "reg-title" },
  { col: "revision", label: "Rev", className: "reg-rev" },
  { col: "date", label: "Date", className: "reg-date" },
  { col: "discipline", label: "Discipline", className: "reg-disc" },
];

function fieldValue(doc: Doc | undefined, field: string): string {
  if (!doc) return "";
  const f = doc.fields.find((x) => x.field === field);
  if (f?.value) return f.value;
  if (field === "number") return doc.number ?? "";
  if (field === "revision") return doc.revision ?? "";
  return "";
}

/** Whether the profile reads a document: the user's choice, else its kind. */
export function profileReads(doc: Doc, readKinds: string[]): boolean {
  if (doc.profile_read === "read") return true;
  if (doc.profile_read === "skip") return false;
  return readKinds.includes(fieldValue(doc, "kind"));
}

function shortDate(iso: string): string {
  const m = iso.match(/^(\d{4})-(\d{2})-(\d{2})/);
  if (m) return `${m[3]}/${m[2]}/${m[1].slice(2)}`;
  const numeric = iso.match(/^(\d{1,2})[./-](\d{1,2})[./-](\d{4})$/);
  if (numeric) return `${numeric[1].padStart(2, "0")}/${numeric[2].padStart(2, "0")}/${numeric[3].slice(2)}`;
  const named = iso.match(/^(\d{1,2})\s+([A-Za-z]+),?\s+(\d{4})$/);
  const month = named ? ["jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"].indexOf(named[2].slice(0, 3).toLowerCase()) + 1 : 0;
  return named && month ? `${named[1].padStart(2, "0")}/${String(month).padStart(2, "0")}/${named[3].slice(2)}` : iso;
}

function sortValue(row: RowModel, col: Col): string {
  if (col === "title") return (fieldValue(row.doc, "title") || suggestedTitle(row.doc) || row.filename).toLowerCase();
  return fieldValue(row.doc, col).toLowerCase();
}

function suggestedTitle(doc: Doc | undefined): string {
	if (doc?.status !== "filed") return "";
  if (doc?.fields.some(f => f.field === "title" && (f.value || f.decided_by === "user"))) return "";
  return doc?.visual_title ?? "";
}

function RegisterCell({ doc, field, catalog, onCorrect, editRequest = 0, selected = false, children, ...attrs }: {
  doc: Doc | undefined;
  field: Col;
  catalog: Catalog | null;
  onCorrect: Props["onCorrect"];
  /** Bumped by the row menu's Rename to open this cell's editor. */
  editRequest?: number;
  /** Announced first, since a plain table row cannot carry a selected state. */
  selected?: boolean;
  children: ReactNode;
} & HTMLAttributes<HTMLTableCellElement>) {
  const [editing, setEditing] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const saving = useRef(false);
  const cell = useRef<HTMLTableCellElement>(null);
  const spec = FIELDS.find(spec => spec.field === field)!;
  const options = spec.vocab ? catalog?.[spec.vocab] : undefined;
  const editable = doc?.status === "filed" && (!spec.vocab || !!options);
  const begin = () => {
    if (!editable || editing) return;
    setError("");
    setEditing(true);
  };
  useEffect(() => { if (editRequest > 0) begin(); }, [editRequest]);
  const finish = (key: CommitKey | "cancel") => {
    setEditing(false);
    if (key === "blur") return;
    const cells = Array.from(cell.current?.parentElement?.querySelectorAll<HTMLElement>('td[data-editable="true"]') ?? [])
      .filter(el => el.getClientRects().length > 0);
    const index = cells.indexOf(cell.current!);
    const next = key === "tab" ? cells[index + 1] : key === "shift-tab" ? cells[index - 1] : cell.current;
    (next ?? cell.current)?.focus({ preventScroll: true });
  };
  const commit = async (value: string, key: CommitKey) => {
    if (!doc || saving.current) return;
    saving.current = true;
    setBusy(true); setError("");
    try {
      const next = value.trim();
      // Explicitly confirming the same rule/model value still makes it the
      // user's value, just as correction in Filing details does.
      if (next !== fieldValue(doc, field) || doc.fields.find(saved => saved.field === field)?.decided_by !== "user") {
        await onCorrect(doc.id, field, next);
      }
      finish(key);
    } catch (e) {
      setError(e instanceof Error ? `Couldn't save: ${e.message}` : "Couldn't save. Try again.");
    } finally {
      saving.current = false;
      setBusy(false);
    }
  };
  return <td {...attrs} ref={cell} data-editable={editable || undefined} data-editing={editing || undefined}
    tabIndex={editable ? 0 : undefined} aria-label={editable ? `${selected ? "Selected. " : ""}${spec.label}: ${options?.find(option => option.id === fieldValue(doc, field))?.label || fieldValue(doc, field) || "Not set"}. Press F2 to edit` : undefined}
    onKeyDown={e => {
      if (e.target !== e.currentTarget) return;
      if (e.key === "F2") { e.preventDefault(); e.stopPropagation(); begin(); }
    }}>
    {editing ? <>
      <InlineCellEditor className="reg-edit" value={fieldValue(doc, field)} label={`Edit ${spec.label}`} error={error} busy={busy}
        maxLength={500} options={options} onCommit={(value, key) => void commit(value, key)} onCancel={() => finish("cancel")} />
      {busy && <span className="sr-only" role="status">Saving…</span>}
      {error && <span className="reg-edit-error" role="alert">{error}</span>}
    </> : children}
  </td>;
}

// Kind and lifecycle stay editable in the document's filing details; the
// register keeps to what people scan for.
type FieldChoice = "discipline";
interface MenuState extends RowMenuAt { ids: string[]; rowKey: string }

export function Register({
  rows,
  selection: controlledSelection,
  onSelectionChange,
  catalog,
  live,
  onCorrect,
  onAddFiles,
  onSetReading,
  notReadOnly,
  onClearFilter,
  onDelete,
  fileOver = false,
  openKey,
  onOpen,
}: Props) {
  const [sort, setSort] = useState<{ col: Col; dir: 1 | -1 } | null>(null);
  const [localSelection, setLocalSelection] = useState<Selection>(emptySelection);
  const selection = controlledSelection ?? localSelection;
  const setSelection = onSelectionChange ?? setLocalSelection;
  const selected = useMemo(() => new Set(selection.ids), [selection]);
  const [busy, setBusy] = useState(false);
  const [fieldEdit, setFieldEdit] = useState<{ field: FieldChoice; ids: string[] } | null>(null);
  const [fieldChoice, setFieldChoice] = useState<string | null>(null);
  const [notice, setNotice] = useState("");
  const fieldSaving = useRef(false);
  const [menu, setMenu] = useState<MenuState | null>(null);
  const [rename, setRename] = useState<{ id: string; n: number } | null>(null);
  const [confirm, setConfirm] = useState<string[] | null>(null);
  const [deleting, setDeleting] = useState(false);
  const readKinds = useMemo(() => catalog?.profile_read_kinds ?? [], [catalog]);

  const ordered = useMemo(() => {
    const uploads = rows.filter((r) => !r.doc);
    const docs = rows.filter((r) => r.doc);
    const col = sort?.col ?? "number";
    const dir = sort?.dir ?? 1;
    docs.sort((a, b) => {
      const av = sortValue(a, col);
      const bv = sortValue(b, col);
      if (!av && bv) return 1; // blanks last
      if (av && !bv) return -1;
      return av.localeCompare(bv, undefined, { numeric: true }) * dir;
    });
    const shown = notReadOnly ? docs.filter((r) => !profileReads(r.doc!, readKinds)) : docs;
    return [...(notReadOnly ? [] : uploads), ...shown];
  }, [rows, sort, notReadOnly, readKinds]);

  const activity = ordered.map(row => {
    if (row.uploadError || row.interrupted) return null;
    if (row.doc?.text_status === "leased") return { state: "active" };
    return documentActivity(row) ?? (row.doc?.text_status === "queued" ? { state: "queued" } : null);
  }).filter(a => a !== null);
  const activeCount = activity.filter(a => a.state === "active").length;
  const queuedCount = activity.length - activeCount;

  // Selection is by document id, in display order; rows that leave the list
  // leave the selection.
  const docIds = useMemo(() => ordered.flatMap((r) => (r.doc ? [r.doc.id] : [])), [ordered]);
  useEffect(() => { setSelection(s => prune(s, docIds)); }, [docIds, setSelection]);

  const select = (id: string, e: { shiftKey: boolean; ctrlKey: boolean; metaKey: boolean }) => {
    if (fieldSaving.current) return;
    setNotice("");
    setSelection(s => clickRow(s, id, docIds, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey }));
  };

  const titleOf = (id: string) => {
    const doc = rows.find((r) => r.doc?.id === id)?.doc;
    if (!doc) return "";
    const number = fieldValue(doc, "number");
    const title = fieldValue(doc, "title") || doc.filename;
    return number ? `${number} ${title}` : title;
  };
  const confirmMessage = (ids: string[]) => {
    const names = ids.slice(0, 3).map(titleOf).filter(Boolean);
    const more = ids.length > names.length ? ", …" : "";
    const what = ids.length === 1 ? `Delete ${names[0] ?? "this document"}?` : `Delete ${ids.length} documents, including ${names.join(", ")}${more}?`;
    return `${what} The files, their filing and anything the profile read from them are removed. This can't be undone.`;
  };
  const remove = async () => {
    if (!confirm) return;
    setDeleting(true);
    const done = await onDelete(confirm);
    setDeleting(false);
    if (done) {
      setSelection(emptySelection);
      setConfirm(null);
    }
  };

  const apply = async (ids: string[], setting: ReadSetting) => {
    setBusy(true);
    try {
      await onSetReading(ids, setting);
    } finally {
      setBusy(false);
    }
  };

  const disciplineLabel = (id: string) => catalog?.disciplines.find((d) => d.id === id)?.label ?? id;
  const fieldOptions = fieldEdit ? catalog?.disciplines ?? [] : [];
  const saveField = async () => {
    if (!fieldEdit || fieldSaving.current || fieldChoice === null) return;
    fieldSaving.current = true;
    setBusy(true); setNotice("");
    const { field, ids } = fieldEdit;
    const failed: string[] = [];
    // Bound concurrent requests while retaining the validated correction path.
    let next = 0;
    await Promise.all(Array.from({ length: Math.min(4, ids.length) }, async () => {
      while (next < ids.length) {
        const id = ids[next++];
        try { await onCorrect(id, field, fieldChoice); }
        catch { failed.push(id); }
      }
    }));
    setNotice(failed.length ? `${ids.length - failed.length} saved; ${failed.length} could not be saved. They stay selected: set ${field} again to retry.` : `Updated ${field} for ${ids.length} document${ids.length === 1 ? "" : "s"}.`);
    if (failed.length) setSelection(selectEveryRow(failed));
    fieldSaving.current = false;
    setBusy(false);
    setFieldEdit(null);
  };

  // The menu acts on the selection when the row is part of it; otherwise the
  // row becomes the selection, as a file browser does.
  const openMenu = (row: RowModel, x: number, y: number, trigger: HTMLElement | null) => {
    const id = row.doc?.id;
    let ids: string[] = [];
    if (id) {
      if (selected.has(id)) ids = [...selection.ids];
      else { setSelection(clickRow(emptySelection, id, docIds)); ids = [id]; }
    }
    setNotice("");
    setMenu({ x, y, ids, rowKey: row.key, trigger });
  };
  const closeMenu = (restoreFocus: boolean) => {
    if (restoreFocus) menu?.trigger?.focus({ preventScroll: true });
    setMenu(null);
  };
  const menuItems = (m: MenuState): MenuItem[] => {
    const docs = m.ids.flatMap(id => rows.find(r => r.doc?.id === id)?.doc ?? []);
    const one = docs.length <= 1;
    const filed = docs.length > 0 && docs.every(d => d.status === "filed");
    const reads = docs.map(d => profileReads(d, readKinds));
    const items: MenuItem[] = [];
    if (one) items.push({ label: "Open", icon: <IconOpen />, onSelect: () => onOpen(m.rowKey) });
    if (!docs.length) return items;
    if (one) items.push({ label: "Rename", icon: <IconRename />, disabled: !filed, onSelect: () => setRename(r => ({ id: docs[0].id, n: (r?.n ?? 0) + 1 })) });
    items.push("sep", {
      label: "Set discipline…", icon: <IconTag />, disabled: !filed || !catalog,
      onSelect: () => { setFieldChoice(null); setFieldEdit({ field: "discipline", ids: m.ids }); },
    }, "sep");
    if (reads.some(r => !r)) items.push({ label: "Include in profile", icon: <IconEye />, disabled: busy, onSelect: () => void apply(m.ids, "read") });
    if (reads.some(r => r)) items.push({ label: "Exclude from profile", icon: <IconEyeOff />, disabled: busy, onSelect: () => void apply(m.ids, "skip") });
    if (docs.some(d => d.profile_read === "read" || d.profile_read === "skip")) {
      items.push({ label: "Reset profile reading to automatic", icon: <IconUndo />, disabled: busy, onSelect: () => void apply(m.ids, "auto") });
    }
    items.push("sep", { label: one ? "Delete…" : `Delete ${docs.length} documents…`, icon: <IconBin />, danger: true, onSelect: () => setConfirm(m.ids) });
    return items;
  };

  return (
    <div className="register file-drop" data-over={fileOver ? "true" : undefined}>
      <div className="reg-head">
        {/* Named for screen readers only: the panel already reads as the register (owner, 7 Nov 2026). */}
        <h2 className="sr-only">Documents</h2>
        <span className="reg-live" data-state={live} role="status">
          <span className={live === "live" ? "sr-only" : undefined}>
            {live === "live" ? "Live" : live === "reconnecting" ? "Reconnecting…" : "Offline, retrying"}
          </span>
        </span>
        <button type="button" className="reg-tool" onClick={onAddFiles} aria-label="Add files" title="Add files">
          <IconUpload />
        </button>
      </div>
      {activity.length > 0 && <p className="reg-activity" role="status" aria-atomic="true"><Pending queued={!activeCount}>{[activeCount > 0 && `${activeCount} processing`, queuedCount > 0 && `${queuedCount} waiting`].filter(Boolean).join(" · ")}</Pending></p>}
      <p className="reg-notice" role="status">{notice}</p>
      {notReadOnly && (
        <p className="reg-filter">
          Showing documents excluded from profile reading.{" "}
          <button type="button" className="cell-link" onClick={onClearFilter}>
            Show all
          </button>
        </p>
      )}
      {rows.length === 0 ? (
        <p className="reg-empty">Drop drawings, reports and schedules anywhere on this page. Each is filed in about a second.</p>
      ) : (
        <table
          className="reg-table"
          aria-label="Documents. Click to select, Ctrl or Shift to add, double-click or Enter to open, right-click for actions."
          onKeyDown={e => {
            if ((e.target as HTMLElement).closest("input,select,textarea")) return;
            if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "a") { e.preventDefault(); setNotice(""); setSelection(selectEveryRow(docIds)); }
            else if (e.key === "Escape" && selected.size > 0) { setSelection(emptySelection); }
          }}
        >
          <colgroup>
            {COLUMNS.map((c) => (
              <col key={c.col} className={c.className} />
            ))}
            <col className="reg-more" />
          </colgroup>
          <thead>
            <tr>
              {COLUMNS.map((c) => {
                const active = (sort?.col ?? "number") === c.col;
                return (
                  <th key={c.col} scope="col" className={c.className} aria-sort={active ? ((sort?.dir ?? 1) === 1 ? "ascending" : "descending") : "none"}>
                    <button
                      type="button"
                      onClick={() => setSort((s) => (s && s.col === c.col ? { col: c.col, dir: s.dir === 1 ? -1 : 1 } : { col: c.col, dir: 1 }))}
                    >
                      {c.label}
                    </button>
                  </th>
                );
              })}
              <th scope="col" className="reg-more"><span className="sr-only">Actions</span></th>
            </tr>
          </thead>
          {ordered.map((row) => (
            <RegisterRow
              key={row.key}
              row={row}
              open={openKey === row.key}
              onOpen={() => onOpen(row.key)}
              catalog={catalog}
              disciplineLabel={disciplineLabel}
              onCorrect={onCorrect}
              selected={!!row.doc && selected.has(row.doc.id)}
              onSelect={select}
              reads={row.doc ? profileReads(row.doc, readKinds) : null}
              menuOpen={menu?.rowKey === row.key}
              onMenu={(x, y, trigger) => menu && trigger instanceof HTMLButtonElement && trigger === menu.trigger ? closeMenu(false) : openMenu(row, x, y, trigger)}
              renameRequest={row.doc && rename?.id === row.doc.id ? rename.n : 0}
            />
          ))}
        </table>
      )}
      {menu && <RowMenu at={menu} items={menuItems(menu)} label={menu.ids.length > 1 ? `Actions for ${menu.ids.length} documents` : "Document actions"}
        head={menu.ids.length > 1 ? `${menu.ids.length} documents selected` : undefined} onClose={closeMenu} />}
      {fieldEdit && (
        <ConfirmDialog
          title={`Set ${fieldEdit.field}${fieldEdit.ids.length > 1 ? ` for ${fieldEdit.ids.length} documents` : ""}`}
          message="The value you choose replaces the current one."
          confirmLabel="Apply"
          busyLabel="Saving…"
          tone="plain"
          busy={busy}
          confirmDisabled={fieldChoice === null}
          onConfirm={() => void saveField()}
          onCancel={() => setFieldEdit(null)}
        >
          <select className="select reg-field-choice" aria-label={`New ${fieldEdit.field}`} disabled={busy} value={fieldChoice ?? "__choose"} onChange={e => setFieldChoice(e.target.value)}>
            <option value="__choose" disabled>Choose…</option>
            <option value="">None (clear value)</option>
            {fieldOptions.map(option => <option key={option.id} value={option.id}>{option.label}</option>)}
          </select>
        </ConfirmDialog>
      )}
      {confirm && (
        <ConfirmDialog
          title={confirm.length === 1 ? "Delete document" : `Delete ${confirm.length} documents`}
          message={confirmMessage(confirm)}
          confirmLabel="Delete"
          busy={deleting}
          onConfirm={remove}
          onCancel={() => setConfirm(null)}
        />
      )}
    </div>
  );
}

function RowMark({ row }: { row: RowModel }) {
  const doc = row.doc;
  if (row.uploadError || row.interrupted) {
    return (
      <span className="reg-flag" data-tone="alert" title="Stopped. Open to retry or dismiss.">
        <IconRetry />
        <span className="sr-only">Stopped</span>
      </span>
    );
  }
  if (documentActivity(row)) return null; // The status beside the title carries the activity.
  if (!doc) return null;
  if (doc.status === "split") {
    return (
      <span className="reg-flag" title="Drawing set: each sheet is listed separately; the original is retained">
        <span className="reg-set">Set</span>
      </span>
    );
  }
  if (doc.status === "not_filed") {
    return (
      <span className="reg-flag" data-tone="stored" title="Stored, not filed">
        <IconStored />
        <span className="sr-only">Stored, not filed</span>
      </span>
    );
  }
  return null;
}

function RegisterRow({
  row,
  open,
  onOpen,
  catalog,
  disciplineLabel,
  onCorrect,
  selected,
  onSelect,
  reads,
  menuOpen,
  onMenu,
  renameRequest,
}: {
  row: RowModel;
  open: boolean;
  onOpen: () => void;
  catalog: Catalog | null;
  disciplineLabel: (id: string) => string;
  onCorrect: (docId: string, field: string, value: string) => Promise<void>;
  selected: boolean;
  onSelect: (id: string, e: { shiftKey: boolean; ctrlKey: boolean; metaKey: boolean }) => void;
  reads: boolean | null;
  menuOpen: boolean;
  /** Opens (or, from the same trigger, closes) the row actions menu at x, y. */
  onMenu: (x: number, y: number, trigger: HTMLElement | null) => void;
  renameRequest: number;
}) {
  const doc = row.doc;
  const suggested = suggestedTitle(doc);
  const [accepting, setAccepting] = useState(false);
  const [acceptError, setAcceptError] = useState("");
  const more = useRef<HTMLButtonElement>(null);
  const acceptTitle = async () => {
    if (!doc || !suggested || accepting) return;
    setAccepting(true); setAcceptError("");
    try { await onCorrect(doc.id, "title", suggested); }
    catch { setAcceptError("Could not save title. Try Accept again."); }
    finally { setAccepting(false); }
  };
  const title = fieldValue(doc, "title") || suggested || (doc?.status === "split" ? row.filename : "");
  const titleNeedsReview = doc?.fields.some((f) => f.field === "title" && f.decided_by === "jev" && f.band === "blank");
  const activity = documentActivity(row);
  const missingTitle = titleNeedsReview ? "Title needs review" : "Title not extracted";
  const discipline = fieldValue(doc, "discipline");
  const date = fieldValue(doc, "date");
  const userSet = (field: string) => doc?.fields.find((f) => f.field === field)?.decided_by === "user";
  const inControl = (e: { target: EventTarget }) => !!(e.target as HTMLElement).closest("input,select,textarea,button");
  const menuFromRow = (el: HTMLElement) => {
    const r = (more.current ?? el).getBoundingClientRect();
    onMenu(r.right, r.bottom, more.current);
  };
  const keys = (e: KeyboardEvent) => {
    if (e.target !== e.currentTarget && !(e.target instanceof HTMLTableCellElement)) return;
    if (e.key === "Enter") { e.preventDefault(); onOpen(); return; }
    if (e.key === "ContextMenu" || (e.key === "F10" && e.shiftKey)) { e.preventDefault(); menuFromRow(e.currentTarget as HTMLElement); return; }
    if (doc && e.key === " ") {
      e.preventDefault();
      onSelect(doc.id, e);
    }
  };
  const click = (e: MouseEvent) => {
    if (doc) {
      if (e.shiftKey) e.preventDefault();
      onSelect(doc.id, e);
    }
  };
  const cell = (field: Col) => ({ doc, field, catalog, onCorrect });
  const override = doc?.profile_read === "read" || doc?.profile_read === "skip";
  // A boolean, so a zero page count never renders as "0" beside the title.
  const showTextStatus = !!doc?.text_status && (doc.text_status !== "done" || !!doc.text_empty_pages);
  return (
    <tbody
      className="reg-doc"
      id={doc ? `doc-${doc.id}` : undefined}
      data-filename={row.filename}
      data-activity={activity?.state ?? (doc?.text_status === "leased" ? "active" : undefined)}
      data-open={open ? "true" : undefined}
      data-flash={row.flash ? "true" : undefined}
      data-landed={row.landed ? "true" : undefined}
      data-selected={selected ? "true" : undefined}
      data-excluded={reads === false ? "true" : undefined}
      tabIndex={-1}
    >
      <tr
        className="reg-line"
        draggable={doc?.status === "filed"}
        onDragStart={e => {
          if (!doc || doc.status !== "filed" || (e.target as HTMLElement).closest("input,select,textarea,button")) { e.preventDefault(); return; }
          e.dataTransfer.setData(FILED_DOCUMENT_TYPE, doc.id);
          e.dataTransfer.effectAllowed = "link";
        }}
        tabIndex={0}
        onClick={click}
        onDoubleClick={e => { if (!inControl(e)) onOpen(); }}
        onContextMenu={e => {
          // An open editor keeps the browser's own menu for cut and paste.
          if ((e.target as HTMLElement).closest("input,select,textarea")) return;
          e.preventDefault();
          onMenu(e.clientX, e.clientY, e.currentTarget);
        }}
        onKeyDown={keys}
      >
        <RegisterCell {...cell("number")} className="reg-no" title={fieldValue(doc, "number")} data-user={userSet("number") || undefined}>
          {fieldValue(doc, "number")}
        </RegisterCell>
        <RegisterCell {...cell("title")} editRequest={renameRequest} selected={selected} className="reg-title" data-suggested={!!suggested || undefined} title={`${title || (titleNeedsReview ? "No title accepted automatically. Open this row to enter the title from the document." : missingTitle)}\nFile: ${row.filename}`} data-user={userSet("title") || undefined}>
          <RowMark row={row} />
          {reads === false && (
            <span className="reg-unread" title={`Not read for the project profile${override ? " (your choice)" : " (automatic for this kind)"}. Right-click to include it.`}>
              <IconEyeOff />
              <span className="sr-only">Not read for the project profile.</span>
            </span>
          )}
          {row.uploadError ? <span className="error-text">{row.uploadError}</span> : title || (
            <>
              <span>{row.filename}</span>
              <span className={ocrStage(doc) ? "ocr-summary" : "reg-title-review"}>{activity ? <Pending queued={activity.state === "queued"}>{activity.label}</Pending> : "Needs title review"}</span>
            </>
          )}
          {title && activity && !ocrStage(doc) && <span className="reg-file-hint"><Pending queued={activity!.state === "queued"}>{activity!.label}</Pending></span>}
          {doc && (title || !ocrStage(doc)) && <OCRStatus doc={doc} compact />}
          {suggested && <span className="reg-title-accept"><span>Suggested · review</span><button type="button" className="cell-link" aria-label="Accept suggested title" disabled={accepting} onClick={e => { e.stopPropagation(); void acceptTitle(); }}>{accepting ? <Pending delayed>Saving…</Pending> : "Accept"}</button></span>}
          {acceptError && <span className="error-text" role="alert">{acceptError}</span>}
          {doc?.status === "split" && <span className="reg-file-hint">Original drawing set</span>}
          {doc?.status === "not_filed" && (
            <span className="reg-file-hint" title={notFiledReason(doc)}>
              {doc.reason === "no_text_layer" ? "Not filed · no readable text" : "Not filed · open for details"}
            </span>
          )}
          {/* Text that is ready says nothing; only work in progress or a problem shows. */}
          {showTextStatus && (
            <span className="reg-text-status" data-state={doc!.text_status}>
              {doc!.text_status === "done" ? "! Check text coverage" : doc!.text_status === "failed" ? "! Text preparation failed" : <Pending queued={doc!.text_status === "queued"}>{doc!.text_status === "queued" ? "Text preparation queued" : "Preparing text…"}</Pending>}
            </span>
          )}
          {row.progress !== undefined && !doc && <span className="reg-progress" style={{ transform: `scaleX(${row.progress})` }} />}
        </RegisterCell>
        <RegisterCell {...cell("revision")} className="reg-rev" data-user={userSet("revision") || undefined}>
          {fieldValue(doc, "revision")}
        </RegisterCell>
        <RegisterCell {...cell("date")} className="reg-date" title={date}>{date ? shortDate(date) : ""}</RegisterCell>
        <RegisterCell {...cell("discipline")} className="reg-disc" title={discipline ? disciplineLabel(discipline) : undefined} data-user={userSet("discipline") || undefined}>
          {discipline ? disciplineLabel(discipline) : ""}
        </RegisterCell>
        <td className="reg-more">
          <button
            ref={more}
            type="button"
            className="reg-more-btn"
            aria-label={`More actions for ${title || row.filename}`}
            title="More actions"
            aria-haspopup="menu"
            aria-expanded={menuOpen}
            onClick={e => { e.stopPropagation(); menuFromRow(e.currentTarget); }}
            onDoubleClick={e => e.stopPropagation()}
          >
            <IconMore />
          </button>
        </td>
      </tr>
    </tbody>
  );
}
