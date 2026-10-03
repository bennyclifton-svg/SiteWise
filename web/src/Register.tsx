// The document register: one compact line per filing, sortable, opening to
// the full title block for correction. Modelled on the Clerk register.

import { useEffect, useMemo, useState, type KeyboardEvent } from "react";
import type { Catalog, Doc } from "./api";
import { FIELDS, FieldCell, HeadState, REASONS, stateOf, type RowModel } from "./DocumentRow";
import { IconCheck, IconNotChecked, IconRetry, IconStored, IconUpload } from "./icons";
import { OCRStatus, ocrStage } from "./OCRStatus";

type Col = "number" | "title" | "revision" | "date" | "discipline" | "kind";

interface Props {
  rows: RowModel[];
  catalog: Catalog | null;
  live: "live" | "reconnecting" | "offline";
  priorLabel: (id: string) => string | undefined;
  onCorrect: (docId: string, field: string, value: string) => Promise<void>;
  onRetry: (docId: string, missingOnly?: boolean) => Promise<boolean>;
  onJump: (docId: string) => void;
  onDismiss: (key: string) => void;
  onAddFiles: () => void;
}

const COLUMNS: { col: Col; label: string; className: string }[] = [
  { col: "number", label: "DWG No.", className: "reg-no" },
  { col: "title", label: "Title", className: "reg-title" },
  { col: "revision", label: "Rev", className: "reg-rev" },
  { col: "date", label: "Date", className: "reg-date" },
  { col: "discipline", label: "Discipline", className: "reg-disc" },
  { col: "kind", label: "Kind", className: "reg-kind" },
];

function fieldValue(doc: Doc | undefined, field: string): string {
  if (!doc) return "";
  const f = doc.fields.find((x) => x.field === field);
  if (f?.value) return f.value;
  if (field === "number") return doc.number ?? "";
  if (field === "revision") return doc.revision ?? "";
  return "";
}

function shortDate(iso: string): string {
  const m = iso.match(/^(\d{4})-(\d{2})-(\d{2})/);
  return m ? `${m[3]}/${m[2]}/${m[1].slice(2)}` : iso;
}

function sortValue(row: RowModel, col: Col): string {
  if (col === "title") return (fieldValue(row.doc, "title") || row.filename).toLowerCase();
  return fieldValue(row.doc, col).toLowerCase();
}

export function Register({ rows, catalog, live, priorLabel, onCorrect, onRetry, onJump, onDismiss, onAddFiles }: Props) {
  const [sort, setSort] = useState<{ col: Col; dir: 1 | -1 } | null>(null);
  const [open, setOpen] = useState<Record<string, boolean>>({});

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
    return [...uploads, ...docs];
  }, [rows, sort]);

  const toggle = (key: string) => setOpen((o) => ({ ...o, [key]: !o[key] }));
  // A jump from the profile flashes a row; open it so its fields show.
  useEffect(() => {
    const flashed = rows.find((r) => r.flash);
    if (flashed) setOpen((o) => (o[flashed.key] ? o : { ...o, [flashed.key]: true }));
  }, [rows]);
  const disciplineLabel = (id: string) => catalog?.disciplines.find((d) => d.id === id)?.label ?? id;

  return (
    <div className="register">
      <div className="reg-head">
        <h2>
          Documents <span className="muted">{rows.filter((r) => r.doc).length}</span>
        </h2>
        <span className="reg-live" data-state={live} role="status">
          <span className={live === "live" ? "sr-only" : undefined}>
            {live === "live" ? "Live" : live === "reconnecting" ? "Reconnecting…" : "Offline, retrying"}
          </span>
        </span>
        <button type="button" className="btn btn-small" onClick={onAddFiles}>
          <IconUpload />
          Add files
        </button>
      </div>
      {rows.length === 0 ? (
        <p className="reg-empty">Drop drawings, reports and schedules anywhere on this page. Each is filed in about a second.</p>
      ) : (
        <table className="reg-table">
          <colgroup>
            {COLUMNS.map((c) => (
              <col key={c.col} className={c.className} />
            ))}
            <col className="reg-mark" />
          </colgroup>
          <thead>
            <tr>
              {COLUMNS.map((c) => {
                const active = (sort?.col ?? "number") === c.col;
                return (
                  <th key={c.col} scope="col" aria-sort={active ? ((sort?.dir ?? 1) === 1 ? "ascending" : "descending") : "none"}>
                    <button
                      type="button"
                      onClick={() => setSort((s) => (s && s.col === c.col ? { col: c.col, dir: s.dir === 1 ? -1 : 1 } : { col: c.col, dir: 1 }))}
                    >
                      {c.label}
                    </button>
                  </th>
                );
              })}
              <th scope="col">
                <span className="sr-only">State</span>
              </th>
            </tr>
          </thead>
          {ordered.map((row) => (
            <RegisterRow
              key={row.key}
              row={row}
              open={!!open[row.key]}
              onToggle={() => toggle(row.key)}
              catalog={catalog}
              disciplineLabel={disciplineLabel}
              priorLabel={priorLabel}
              onCorrect={onCorrect}
              onRetry={onRetry}
              onJump={onJump}
              onDismiss={onDismiss}
            />
          ))}
        </table>
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
  if (!doc || doc.status === "pending") {
    return (
      <span className="reg-flag" title="Filing…">
        <span className="reg-spin" aria-hidden />
        <span className="sr-only">Filing</span>
      </span>
    );
  }
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
  const states = doc.fields.filter((f) => f.field !== "supersedes").map(stateOf);
  const check = states.filter((s) => s === "check").length;
  if (check > 0) {
    const text = `${check} field${check === 1 ? "" : "s"} to check`;
    return (
      <span className="reg-flag" data-tone="warn" title={text}>
        <IconCheck />
        <span className="sr-only">{text}</span>
      </span>
    );
  }
  if (states.includes("unchecked")) {
    return (
      <span className="reg-flag" title="Not checked: Jev didn't answer in time">
        <IconNotChecked />
        <span className="sr-only">Not checked</span>
      </span>
    );
  }
  return null;
}

function RegisterRow({
  row,
  open,
  onToggle,
  catalog,
  disciplineLabel,
  priorLabel,
  onCorrect,
  onRetry,
  onJump,
  onDismiss,
}: {
  row: RowModel;
  open: boolean;
  onToggle: () => void;
  catalog: Catalog | null;
  disciplineLabel: (id: string) => string;
  priorLabel: (id: string) => string | undefined;
  onCorrect: (docId: string, field: string, value: string) => Promise<void>;
  onRetry: (docId: string, missingOnly?: boolean) => Promise<boolean>;
  onJump: (docId: string) => void;
  onDismiss: (key: string) => void;
}) {
  const doc = row.doc;
  const title = fieldValue(doc, "title");
  const discipline = fieldValue(doc, "discipline");
  const kind = fieldValue(doc, "kind");
  const kindLabel = catalog?.kinds.find((k) => k.id === kind)?.label ?? kind;
  const date = fieldValue(doc, "date");
  const userSet = (field: string) => doc?.fields.find((f) => f.field === field)?.decided_by === "user";
  const keys = (e: KeyboardEvent) => {
    if (e.target !== e.currentTarget) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      onToggle();
    }
    if (e.key === "Escape" && open) onToggle();
  };
  const detailId = `reg-detail-${row.key}`;
  return (
    <tbody
      className="reg-doc"
      id={doc ? `doc-${doc.id}` : undefined}
      data-filename={row.filename}
      data-open={open ? "true" : undefined}
      data-flash={row.flash ? "true" : undefined}
      data-landed={row.landed ? "true" : undefined}
      tabIndex={-1}
    >
      <tr
        className="reg-line"
        tabIndex={0}
        aria-expanded={open}
        aria-controls={open ? detailId : undefined}
        onClick={onToggle}
        onKeyDown={keys}
      >
        <td className="reg-no" title={fieldValue(doc, "number")} data-user={userSet("number") || undefined}>
          {fieldValue(doc, "number")}
        </td>
        <td className="reg-title" title={`${title || "Title not extracted"}\nFile: ${row.filename}`} data-user={userSet("title") || undefined}>
          {row.uploadError ? <span className="error-text">{row.uploadError}</span> : title || (
            <>
              <span className="muted">{ocrStage(doc) ? "Recovering document details" : !doc || doc.status === "pending" ? "Reading title…" : "Title not extracted"}</span>
              <span className="reg-file-hint">File: {row.filename}</span>
            </>
          )}
          {doc && <OCRStatus doc={doc} compact />}
          {doc?.status === "not_filed" && (
            <span className="reg-file-hint" title={REASONS[doc.reason ?? ""] ?? doc.reason}>
              {doc.reason === "no_text_layer" ? "Not filed · no readable text" : "Not filed · open for details"}
            </span>
          )}
          {doc?.text_status && (
            <span className="reg-text-status" data-state={doc.text_status} title={doc.text_status === "done" ? "Extracted text is saved. Check source coverage in the profile for unreadable pages and unresolved requirements." : undefined}>
              {doc.text_status === "done" ? (doc.text_empty_pages ? "! Check text coverage" : doc.text_source_version ? `✓ Text ready${doc.text_pages ? ` · ${doc.text_pages} pages` : ""}` : "Text prepared · update profile") : doc.text_status === "failed" ? "! Text preparation failed" : "◷ Preparing text…"}
            </span>
          )}
          {row.progress !== undefined && !doc && <span className="reg-progress" style={{ transform: `scaleX(${row.progress})` }} />}
        </td>
        <td className="reg-rev" data-user={userSet("revision") || undefined}>
          {fieldValue(doc, "revision")}
        </td>
        <td className="reg-date">{date ? shortDate(date) : ""}</td>
        <td className="reg-disc" title={discipline ? disciplineLabel(discipline) : undefined} data-user={userSet("discipline") || undefined}>
          {discipline ? disciplineLabel(discipline) : ""}
        </td>
        <td className="reg-kind" title={kindLabel || undefined} data-user={userSet("kind") || undefined}>
          {kindLabel}
        </td>
        <td className="reg-mark">
          <RowMark row={row} />
        </td>
      </tr>
      {open && (
        <tr className="reg-detail" id={detailId}>
          <td colSpan={COLUMNS.length + 1}>
            <div className="reg-detail-head">
              <span className="reg-file" title={row.filename}>
                {row.filename}
              </span>
              <HeadState row={row} onRetry={onRetry} onDismiss={onDismiss} />
              {doc && (
                <a className="cell-link" href={`/api/documents/${doc.id}/file`}>
                  Download
                </a>
              )}
            </div>
            {doc && <OCRStatus doc={doc} onReprocess={onRetry} />}
            {doc?.source_id && (
              <p className="tb-note">
                Sheet {doc.sheet_page} of {doc.sheet_total} ·{" "}
                <button type="button" className="cell-link" onClick={() => onJump(doc.source_id!)}>
                  {doc.source_filename}
                </button>
              </p>
            )}
            {doc?.expansion && (
              <p className="tb-note">
                {doc.expansion.status === "complete"
                  ? `${doc.expansion.page_count} sheets listed separately. Original drawing set retained.`
                  : doc.expansion.status === "pending"
                    ? `Checking ${doc.expansion.page_count} pages for separate drawing sheets…`
                    : doc.expansion.reason}
                {doc.expansion.status === "review" && (
                  <button type="button" className="btn btn-small" onClick={() => onRetry(doc.id)}>
                    Retry sheet processing
                  </button>
                )}
              </p>
            )}
            {doc?.status === "not_filed" && (
              <p className="tb-note">
                <strong>Kept, not filed.</strong> {REASONS[doc.reason ?? ""] ?? doc.reason}
              </p>
            )}
            {doc?.status === "filed" && (
              <div className="reg-grid">
                {FIELDS.map((spec, i) => (
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
                ))}
              </div>
            )}
          </td>
        </tr>
      )}
    </tbody>
  );
}
