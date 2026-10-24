import { Pending } from "./Activity";
import { FILED_DOCUMENT_TYPE } from "./documentDrag";
// The document register: one compact line per filing, sortable, opening the
// source and filing details in the centre workspace.

import { useEffect, useMemo, useRef, useState, type KeyboardEvent, type MouseEvent, type ReactNode, type HTMLAttributes } from "react";
import type { Catalog, Doc, ReadSetting } from "./api";
import { FIELDS, notFiledReason, type RowModel } from "./DocumentRow";
import { ConfirmDialog } from "./ConfirmDialog";
import { IconBin, IconRetry, IconStored, IconUpload } from "./icons";
import { OCRStatus, ocrStage } from "./OCRStatus";
import { documentActivity } from "./documentActivity";
import { clickRow, emptySelection, prune, selectAll as selectEveryRow, type Selection } from "./table/selection";
import { InlineCellEditor, type CommitKey } from "./table/InlineCellEditor";
import "./register-edit.css";

type Col = "number" | "title" | "revision" | "date" | "discipline" | "kind" | "lifecycle";

interface Props {
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
  { col: "kind", label: "Kind", className: "reg-kind" },
  { col: "lifecycle", label: "Lifecycle", className: "reg-life" },
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

function RegisterCell({ doc, field, catalog, onCorrect, children, ...attrs }: {
  doc: Doc | undefined;
  field: Col;
  catalog: Catalog | null;
  onCorrect: Props["onCorrect"];
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
    tabIndex={editable ? 0 : undefined} aria-label={editable ? `${spec.label}: ${options?.find(option => option.id === fieldValue(doc, field))?.label || fieldValue(doc, field) || "Not set"}. Double-click or press F2 to edit` : undefined}
    onDoubleClick={e => { e.stopPropagation(); begin(); }}
    onKeyDown={e => {
      if (e.target !== e.currentTarget) return;
      if (e.key === "F2" || e.key === "Enter") { e.preventDefault(); e.stopPropagation(); begin(); }
    }}>
    {editing ? <>
      <InlineCellEditor className="reg-edit" value={fieldValue(doc, field)} label={`Edit ${spec.label}`} error={error} busy={busy}
        maxLength={500} options={options} onCommit={(value, key) => void commit(value, key)} onCancel={() => finish("cancel")} />
      {busy && <span className="sr-only" role="status">Saving…</span>}
      {error && <span className="reg-edit-error" role="alert">{error}</span>}
    </> : children}
  </td>;
}

export function Register({
  rows,
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
  const [selection, setSelection] = useState<Selection>(emptySelection);
  const selected = useMemo(() => new Set(selection.ids), [selection]);
  const [busy, setBusy] = useState(false);
  const [bulkField, setBulkField] = useState("discipline");
  const [bulkValue, setBulkValue] = useState<string | null>(null);
  const [bulkMessage, setBulkMessage] = useState("");
  const bulkSaving = useRef(false);
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
  useEffect(() => { setSelection(s => prune(s, docIds)); }, [docIds]);

  const select = (id: string, e: { shiftKey: boolean; ctrlKey: boolean; metaKey: boolean }) => {
    if (bulkSaving.current) return;
    setBulkMessage("");
    setSelection(s => clickRow(s, id, docIds, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey }));
  };
  const allSelected = docIds.length > 0 && docIds.every(id => selected.has(id));
  const someSelected = selected.size > 0 && !allSelected;
  const selectAll = () => { setBulkMessage(""); setSelection(allSelected ? emptySelection : selectEveryRow(docIds)); };

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
  const bulkOptions = catalog?.[bulkField === "discipline" ? "disciplines" : bulkField === "kind" ? "kinds" : "lifecycle"] ?? [];
  const allFiled = [...selected].every(id => rows.find(row => row.doc?.id === id)?.doc?.status === "filed");
  const correctSelected = async () => {
    if (bulkSaving.current || bulkValue === null || !allFiled) return;
    bulkSaving.current = true;
    setBusy(true); setBulkMessage("");
    const ids = [...selected];
    const failed: string[] = [];
    // Bound concurrent requests while retaining the validated correction path.
    let next = 0;
    await Promise.all(Array.from({ length: Math.min(4, ids.length) }, async () => {
      while (next < ids.length) {
        const id = ids[next++];
        try { await onCorrect(id, bulkField, bulkValue); }
        catch { failed.push(id); }
      }
    }));
    setBulkMessage(failed.length ? `${ids.length - failed.length} saved; ${failed.length} could not be saved. Failed documents remain selected. Apply again to retry.` : `Updated ${bulkField} for ${ids.length} documents.`);
    if (failed.length) setSelection(selectEveryRow(failed));
    bulkSaving.current = false;
    setBusy(false);
  };

  return (
    <div className="register file-drop" data-over={fileOver ? "true" : undefined}>
      <div className="reg-head">
        <h2>
          Documents <span className="muted">{rows.filter((r) => r.doc).length}</span>
        </h2>
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
        <table className="reg-table">
          <colgroup>
            <col className="reg-sel" />
            {COLUMNS.map((c) => (
              <col key={c.col} className={c.className} />
            ))}
            <col className="reg-prof" />
            <col className="reg-open" />
            <col className="reg-bin" />
          </colgroup>
          <thead>
            <tr>
              <th scope="col" className="reg-sel">
                <input
                  type="checkbox"
                  aria-label="Select all documents"
                  checked={allSelected}
                  ref={(el) => {
                    if (el) el.indeterminate = someSelected;
                  }}
                  onChange={selectAll}
                  disabled={busy || docIds.length === 0}
                />
              </th>
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
              <th scope="col" className="reg-prof" title="Included for profile reading">
                <span className="reg-th-text" aria-hidden>Use</span>
                <span className="sr-only">Included for profile reading</span>
              </th>
              <th scope="col" className="reg-open"><span className="sr-only">Open document</span></th>
              <th scope="col" className="reg-bin">
                <button
                  type="button"
                  className="reg-del"
                  disabled={selected.size === 0}
                  title={selected.size === 0 ? "Select documents to delete them" : `Delete ${selected.size} selected`}
                  onClick={() => setConfirm([...selected])}
                >
                  <IconBin />
                  <span className="sr-only">Delete selected documents</span>
                </button>
              </th>
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
              onToggleRead={(doc, reads) => apply([doc.id], reads ? "skip" : "read")}
              onDelete={(doc) => setConfirm([doc.id])}
            />
          ))}
        </table>
      )}
      {selected.size > 0 && (
        <div className="reg-bulk" role="toolbar" aria-label="Selected documents">
          <span className="reg-bulk-count">{selected.size} selected</span>
          <div className="reg-bulk-edit">
            <select className="select" aria-label="Bulk edit field" disabled={busy} value={bulkField} onChange={e => { setBulkField(e.target.value); setBulkValue(null); setBulkMessage(""); }}>
              <option value="discipline">Discipline</option><option value="kind">Kind</option><option value="lifecycle">Lifecycle</option>
            </select>
            <select className="select" aria-label="Bulk edit value" disabled={busy || !catalog} value={bulkValue ?? "__choose"} onChange={e => { setBulkValue(e.target.value); setBulkMessage(""); }}>
              <option value="__choose" disabled>Choose value…</option>
              <option value="">None (clear value)</option>
              {bulkOptions.map(option => <option key={option.id} value={option.id}>{option.label}</option>)}
            </select>
            <button type="button" className="btn btn-small" disabled={busy || bulkValue === null || !allFiled} onClick={() => void correctSelected()}>{busy ? "Saving…" : `Apply to ${selected.size}`}</button>
            {!allFiled && <span>Only filed documents can be edited together.</span>}
          </div>
          {bulkMessage && <p className="reg-bulk-message" role="status">{bulkMessage}</p>}
          <button type="button" className="btn btn-small" disabled={busy} onClick={() => apply([...selected], "read")}>
            Include for profile
          </button>
          <button type="button" className="btn btn-small" disabled={busy} onClick={() => apply([...selected], "skip")}>
            Exclude from profile
          </button>
          <button type="button" className="btn btn-small" disabled={busy} onClick={() => apply([...selected], "auto")}>
            Reset to automatic
          </button>
          <button type="button" className="btn btn-small btn-danger-quiet" disabled={busy} onClick={() => setConfirm([...selected])}>
            <IconBin />
            Delete
          </button>
          <button type="button" className="cell-link" disabled={busy} onClick={() => { setSelection(emptySelection); setBulkMessage(""); }}>
            Clear
          </button>
        </div>
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
  onToggleRead,
  onDelete,
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
  onToggleRead: (doc: Doc, reads: boolean) => void;
  onDelete: (doc: Doc) => void;
}) {
  const doc = row.doc;
  const suggested = suggestedTitle(doc);
  const [accepting, setAccepting] = useState(false);
  const [acceptError, setAcceptError] = useState("");
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
  const kind = fieldValue(doc, "kind");
  const kindLabel = catalog?.kinds.find((k) => k.id === kind)?.label ?? kind;
  const date = fieldValue(doc, "date");
  const userSet = (field: string) => doc?.fields.find((f) => f.field === field)?.decided_by === "user";
  const keys = (e: KeyboardEvent) => {
    if (e.target !== e.currentTarget && !(e.target instanceof HTMLTableCellElement)) return;
    if (doc && (e.key === "Enter" || e.key === " ")) {
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
  const readLabel = reads ? "Included for profile reading" : "Excluded from profile reading";
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
        onKeyDown={keys}
      >
        <td className="reg-sel" onClick={(e) => e.stopPropagation()}>
          {doc && (
            <input
              type="checkbox"
              aria-label={`Select ${title || row.filename}`}
              checked={selected}
              onChange={() => undefined}
              onClick={(e) => onSelect(doc.id, { shiftKey: e.shiftKey, ctrlKey: !e.shiftKey || e.ctrlKey, metaKey: e.metaKey })}
            />
          )}
        </td>
        <RegisterCell {...cell("number")} className="reg-no" title={fieldValue(doc, "number")} data-user={userSet("number") || undefined}>
          {fieldValue(doc, "number")}
        </RegisterCell>
        <RegisterCell {...cell("title")} className="reg-title" data-suggested={!!suggested || undefined} title={`${title || (titleNeedsReview ? "No title accepted automatically. Open this row to enter the title from the document." : missingTitle)}\nFile: ${row.filename}`} data-user={userSet("title") || undefined}>
          <RowMark row={row} />
          {row.uploadError ? <span className="error-text">{row.uploadError}</span> : title || (
            <>
              <span className={ocrStage(doc) ? "ocr-summary" : "muted"}>{activity ? <Pending queued={activity!.state === "queued"}>{activity!.label}</Pending> : missingTitle}</span>
              <span className="reg-file-hint"> · File: {row.filename}</span>
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
          {doc?.text_status && (
            <span className="reg-text-status" data-state={doc.text_status} title={doc.text_status === "done" ? "Extracted text is saved. Check source coverage in the profile for unreadable pages and unresolved requirements." : undefined}>
              {doc.text_status === "done" ? (doc.text_empty_pages ? "! Check text coverage" : doc.text_source_version ? `✓ Text ready${doc.text_pages ? ` · ${doc.text_pages} pages` : ""}` : "Text prepared · update profile") : doc.text_status === "failed" ? "! Text preparation failed" : <Pending queued={doc.text_status === "queued"}>{doc.text_status === "queued" ? "Text preparation queued" : "Preparing text…"}</Pending>}
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
        <RegisterCell {...cell("kind")} className="reg-kind" title={kindLabel || undefined} data-user={userSet("kind") || undefined}>
          {kindLabel}
        </RegisterCell>
        <RegisterCell {...cell("lifecycle")} className="reg-life" data-user={userSet("lifecycle") || undefined}>
          {catalog?.lifecycle.find(option => option.id === fieldValue(doc, "lifecycle"))?.label || fieldValue(doc, "lifecycle")}
        </RegisterCell>
        <td className="reg-prof">
          {doc && reads !== null && (
            <button
              type="button"
              className="reg-read"
              data-reads={reads ? "true" : undefined}
              data-override={override ? "true" : undefined}
              aria-pressed={reads}
              title={`${readLabel}${override ? " (your choice)" : " (automatic for this kind)"}. This is a selection, not reading progress. Click to ${reads ? "exclude it" : "include it"}; use Update project profile to start reading.`}
              onClick={(e) => {
                e.stopPropagation();
                onToggleRead(doc, reads);
              }}
            >
              <span aria-hidden>{reads ? "✓" : "–"}</span>
              <span className="sr-only">{readLabel}</span>
            </button>
          )}
        </td>
        <td className="reg-open">
          <button type="button" className="cell-link" aria-label={`Open document: ${title || row.filename}`} aria-controls={open ? "document-workspace" : undefined} aria-expanded={open} onClick={e => { e.stopPropagation(); onOpen(); }}>Open</button>
        </td>
        <td className="reg-bin">
          {doc && (
            <button
              type="button"
              className="reg-del"
              title="Delete this document"
              onClick={(e) => {
                e.stopPropagation();
                onDelete(doc);
              }}
            >
              <IconBin />
              <span className="sr-only">Delete {title || row.filename}</span>
            </button>
          )}
        </td>
      </tr>
    </tbody>
  );
}
