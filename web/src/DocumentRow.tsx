// One filing drawn as a title block: a strip of boxed fields, each stating its
// value, how sure SiteWise is, and who decided it. A box is corrected in place.

import { useEffect, useId, useRef, useState, type CSSProperties, type FormEvent, type KeyboardEvent } from "react";
import type { Catalog, Doc, Field, Option } from "./api";
import { OCRStatus, ocrStage } from "./OCRStatus";
import {
  IconCheck,
  IconConfirmed,
  IconNotChecked,
  IconNotSet,
  IconRetry,
  IconStored,
  IconSupersedes,
  IconYou,
} from "./icons";

export interface RowModel {
  key: string;
  filename: string;
  doc?: Doc;
  /** 0..1 while bytes are still going up. */
  progress?: number;
  uploadError?: string;
  duplicate?: boolean;
  interrupted?: boolean;
  stale?: boolean;
  filedInMs?: number;
  landed?: boolean;
  flash?: boolean;
}

interface Props {
  row: RowModel;
  catalog: Catalog | null;
  priorLabel: (id: string) => string | undefined;
  onCorrect: (docId: string, field: string, value: string) => Promise<void>;
  onRetry: (docId: string, missingOnly?: boolean) => Promise<boolean>;
  onJump: (docId: string) => void;
  onDismiss: (key: string) => void;
}

type CellState = "confirmed" | "check" | "unset" | "unchecked" | "user";

export const FIELDS: { field: string; label: string; vocab?: "kinds" | "disciplines" | "lifecycle"; editable: boolean }[] = [
  { field: "number", label: "Doc no.", editable: true },
  { field: "revision", label: "Rev", editable: true },
  { field: "title", label: "Title", editable: true },
  { field: "date", label: "Date", editable: true },
  { field: "kind", label: "Kind", vocab: "kinds", editable: true },
  { field: "discipline", label: "Discipline", vocab: "disciplines", editable: true },
  { field: "lifecycle", label: "Lifecycle", vocab: "lifecycle", editable: true },
  { field: "supersedes", label: "Supersedes", editable: false },
];

export const REASONS: Record<string, string> = {
  no_text_layer:
    "This PDF has no selectable text and has not been read with OCR. Your original is safe. Choose Retry OCR to recover its lettering and file it for review.",
  ocr_unavailable: "Text recovery is unavailable on this server. Your original is safe. Ask the administrator to check OCR, then retry.",
  ocr_failed: "OCR could not finish reading this PDF. Your original is safe. Retry, or upload a PDF with selectable text.",
  ocr_no_text: "OCR found no readable lettering on the first page. Your original is safe. Try a clearer scan or a PDF with selectable text.",
  ocr_limit: "This PDF exceeds the OCR reading limit. Your original is safe. Upload a smaller PDF or one with selectable text.",
  unsupported_format: "SiteWise files PDF, DOCX and XLSX. This file is kept exactly as uploaded but not filed.",
  unreadable:
    "The file couldn't be opened. It may be damaged or password protected. It's kept exactly as uploaded.",
  too_large: "Its identity pages are larger than intake reads. The file is kept exactly as uploaded.",
  empty: "The file is empty.",
};

export function stateOf(f: Field | undefined): CellState {
  if (!f) return "unset";
  if (f.decided_by === "user") return "user";
  if (f.band === "grey") return "unchecked";
  if (!f.value) return "unset";
  if (f.band === "amber") return "check";
  if (f.band === "green") return "confirmed";
  return "unset";
}

function Why({ field, state }: { field: Field | undefined; state: CellState }) {
  switch (state) {
    case "user":
      return (
        <span className="cell-why">
          <IconYou />
          {field?.value ? "Set by you" : "Cleared by you"}
        </span>
      );
    case "unchecked":
      return (
        <span className="cell-why" title="Jev didn't answer in time. Rule values stand; this field has not been judged.">
          <IconNotChecked />
          Not checked
        </span>
      );
    case "check":
      return (
        <span className="cell-why">
          <IconCheck />
          <span>
            Check · <span className="cell-who">{field?.decided_by === "jev" ? "Jev" : "Rule"}</span>
          </span>
        </span>
      );
    case "confirmed":
      return (
        <span className="cell-why">
          <IconConfirmed />
          {field?.decided_by === "jev" ? (
            <span>
              Confirmed · <span className="cell-who">Jev</span>
            </span>
          ) : (
            "From document"
          )}
        </span>
      );
    default:
      return (
        <span className="cell-why">
          <IconNotSet />
          Not set
        </span>
      );
  }
}

function labelFor(options: Option[] | undefined, id: string): string {
  return options?.find((o) => o.id === id)?.label ?? id;
}

export function DocumentRow({ row, catalog, priorLabel, onCorrect, onRetry, onJump, onDismiss }: Props) {
  const headingId = useId();
  const doc = row.doc;
  const filed = doc?.status === "filed";
  const notFiled = doc?.status === "not_filed";
  const waiting = !doc || doc.status === "pending";

  return (
    <article
      className="tb"
      id={doc ? `doc-${doc.id}` : undefined}
      aria-labelledby={headingId}
      tabIndex={-1}
      data-landed={row.landed ? "true" : undefined}
      data-flash={row.flash ? "true" : undefined}
      aria-busy={waiting && !row.uploadError && !row.interrupted ? "true" : undefined}
    >
      <header className="tb-head">
        <h2 className="tb-file" id={headingId} title={row.filename}>
          {row.filename}
        </h2>
        <HeadState row={row} onRetry={onRetry} onDismiss={onDismiss} />
        {doc && <a className="cell-link" href={`/api/documents/${doc.id}/file`}>Download</a>}
        {row.progress !== undefined && !doc && (
          <span className="tb-progress" style={{ transform: `scaleX(${row.progress})` }} />
        )}
      </header>
      {doc && <OCRStatus doc={doc} onReprocess={onRetry} />}

      {doc?.source_id && (
        <p className="tb-note">Sheet {doc.sheet_page} of {doc.sheet_total} · <button type="button" className="cell-link" onClick={() => onJump(doc.source_id!)}>{doc.source_filename}</button></p>
      )}
      {doc?.expansion && (
        <p className="tb-note">
          {doc.expansion.status === "complete" ? `${doc.expansion.page_count} sheets listed separately. Original drawing set retained.` :
            doc.expansion.status === "pending" ? `Checking ${doc.expansion.page_count} pages for separate drawing sheets…` : doc.expansion.reason}
          {doc.expansion.status === "review" && <button type="button" className="btn btn-small" onClick={() => onRetry(doc.id)}>Retry sheet processing</button>}
        </p>
      )}

      {notFiled && (
        <p className="tb-note">
          <strong>Kept, not filed.</strong> {REASONS[doc.reason ?? ""] ?? doc.reason}
        </p>
      )}
      {row.uploadError && !doc && <p className="tb-note error-text">{row.uploadError}</p>}

      {!notFiled && doc?.status !== "split" && !row.uploadError && (
        <div className="tb-grid">
          {FIELDS.map((spec, i) =>
            filed ? (
              <FieldCell
                key={spec.field}
                index={i}
                spec={spec}
                doc={doc}
                catalog={catalog}
                priorLabel={priorLabel}
                onCorrect={onCorrect}
                onJump={onJump}
              />
            ) : (
              <div className="cell" data-field={spec.field} key={spec.field}>
                <span className="cell-label">{spec.label}</span>
                <span className="skeleton" style={{ width: `${40 + ((i * 23) % 45)}%` }} />
              </div>
            ),
          )}
        </div>
      )}
    </article>
  );
}

export function HeadState({
  row,
  onRetry,
  onDismiss,
}: {
  row: RowModel;
  onRetry: (docId: string, missingOnly?: boolean) => Promise<boolean>;
  onDismiss: (key: string) => void;
}) {
  const doc = row.doc;
  if (doc?.status === "split") return <span className="tb-state">Drawing set · original retained</span>;
  if (!doc) {
    if (row.uploadError) {
      return (
        <>
          <span className="tb-state" data-tone="alert">
            Upload failed
          </span>
          <button type="button" className="btn btn-small" onClick={() => onDismiss(row.key)}>
            Dismiss
          </button>
        </>
      );
    }
    const pct = Math.round((row.progress ?? 0) * 100);
    return <span className="tb-state">{pct < 100 ? `Uploading ${pct}%` : "Stored, filing…"}</span>;
  }
  if (doc.status === "not_filed") {
    return (
      <><span className="tb-state" data-tone="stored">
        <IconStored />
        Stored · not filed
      </span>
      {(doc.reason === "no_text_layer" || doc.reason?.startsWith("ocr_")) && <button type="button" className="btn btn-small" onClick={() => onRetry(doc.id)}><IconRetry />Retry OCR</button>}</>
    );
  }
  if (doc.status === "pending") {
    if (ocrStage(doc)) return <span className="tb-state">{ocrStage(doc)!.title}</span>;
    if (row.interrupted || row.stale) {
      return (
        <>
          <span className="tb-state" data-tone={row.interrupted ? "alert" : undefined}>
            {row.interrupted ? "Filing stopped. The file is safe." : "Still filing…"}
          </span>
          <button type="button" className="btn btn-small" onClick={() => onRetry(doc.id)}>
            <IconRetry />
            Retry filing
          </button>
        </>
      );
    }
    return <span className="tb-state">Filing…</span>;
  }
  const seconds = row.filedInMs !== undefined ? ` in ${(row.filedInMs / 1000).toFixed(2)} s` : "";
  return (
    <span className="tb-state" data-tone="ok">
      <IconConfirmed />
      {row.duplicate ? "Already filed in this project" : `Filed${seconds}`}
    </span>
  );
}

interface CellProps {
  index: number;
  spec: (typeof FIELDS)[number];
  doc: Doc;
  catalog: Catalog | null;
  priorLabel: (id: string) => string | undefined;
  onCorrect: (docId: string, field: string, value: string) => Promise<void>;
  onJump: (docId: string) => void;
}

export function FieldCell({ index, spec, doc, catalog, priorLabel, onCorrect, onJump }: CellProps) {
  const field = doc.fields.find((f) => f.field === spec.field);
  const state = stateOf(field);
  const [editing, setEditing] = useState(false);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const returnFocus = useRef(false);

  useEffect(() => {
    if (!editing && returnFocus.current) {
      returnFocus.current = false;
      buttonRef.current?.focus();
    }
  }, [editing]);

  const options = spec.vocab ? catalog?.[spec.vocab] : undefined;
  const raw = field?.value ?? "";
  const display = raw ? (options ? labelFor(options, raw) : raw) : "—";
  const style = { "--i": index } as CSSProperties;

  if (spec.field === "supersedes") {
    const prior = raw ? (priorLabel(raw) ?? "Earlier filing") : "";
    return (
      <div className="cell" data-field={spec.field} data-state={state} style={style}>
        <span className="cell-label">{spec.label}</span>
        <span className="cell-value" data-empty={raw ? undefined : "true"}>
          {raw ? (
            <button type="button" className="cell-link" onClick={() => onJump(raw)}>
              <IconSupersedes /> {prior}
            </button>
          ) : (
            "—"
          )}
        </span>
        {field ? (
          <Why field={field} state={state} />
        ) : (
          <span className="cell-why">
            <IconNotSet />
            No earlier filing
          </span>
        )}
      </div>
    );
  }

  if (editing) {
    return (
      <CellEditor
        spec={spec}
        initial={raw}
        options={options}
        onCancel={() => {
          returnFocus.current = true;
          setEditing(false);
        }}
        onSave={async (value) => {
          if (value !== raw || state !== "user") {
            await onCorrect(doc.id, spec.field, value);
          }
          returnFocus.current = true;
          setEditing(false);
        }}
      />
    );
  }

  return (
    <button
      ref={buttonRef}
      type="button"
      className="cell"
      data-field={spec.field}
      data-state={state}
      style={style}
      onClick={() => setEditing(true)}
    >
      <span className="cell-label">{spec.label}</span>
      <span className="cell-value" data-empty={raw ? undefined : "true"}>
        {display}
      </span>
      <Why field={field} state={state} />
      <span className="sr-only">, edit {spec.label}</span>
    </button>
  );
}

function CellEditor({
  spec,
  initial,
  options,
  onCancel,
  onSave,
}: {
  spec: (typeof FIELDS)[number];
  initial: string;
  options: Option[] | undefined;
  onCancel: () => void;
  onSave: (value: string) => Promise<void>;
}) {
  const id = useId();
  const [value, setValue] = useState(initial);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const inputRef = useRef<HTMLInputElement & HTMLSelectElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
    inputRef.current?.select?.();
  }, []);

  async function save(next: string) {
    if (saving) return;
    setSaving(true);
    setError("");
    try {
      await onSave(next.trim());
    } catch (e) {
      setSaving(false);
      setError(e instanceof Error && e.message ? `Couldn't save: ${e.message}` : "Couldn't save. Try again.");
      inputRef.current?.focus();
    }
  }

  function submit(e: FormEvent) {
    e.preventDefault();
    void save(value);
  }

  function keys(e: KeyboardEvent) {
    if (e.key === "Escape") {
      e.preventDefault();
      onCancel();
    }
  }

  return (
    <form className="cell cell-edit" data-field={spec.field} onSubmit={submit} onKeyDown={keys}>
      <label className="cell-label" htmlFor={id}>
        {spec.label}
      </label>
      {options ? (
        <select
          id={id}
          ref={inputRef}
          className="select"
          value={value}
          disabled={saving}
          aria-invalid={error ? "true" : undefined}
          onChange={(e) => {
            setValue(e.target.value);
            void save(e.target.value);
          }}
        >
          <option value="">None</option>
          {options.map((o) => (
            <option key={o.id} value={o.id}>
              {o.label}
            </option>
          ))}
        </select>
      ) : (
        <input
          id={id}
          ref={inputRef}
          className="input"
          value={value}
          maxLength={500}
          disabled={saving}
          autoComplete="off"
          spellCheck={spec.field === "title"}
          aria-invalid={error ? "true" : undefined}
          aria-describedby={`${id}-hint`}
          onChange={(e) => setValue(e.target.value)}
        />
      )}
      <span className={error ? "cell-hint error-text" : "cell-hint"} id={`${id}-hint`} role={error ? "alert" : undefined}>
        {saving ? "Saving…" : error || (options ? "Esc to cancel" : "Enter to save · Esc to cancel")}
      </span>
    </form>
  );
}
