import { useEffect, useState } from "react";
import type { Catalog, Doc } from "./api";
import { FIELDS, FieldCell, HeadState, notFiledReason, type RowModel } from "./DocumentRow";
import { OCRStatus } from "./OCRStatus";
import { VisualIdentityReview } from "./VisualIdentityReview";
import "./document-workspace.css";

interface Props {
  row: RowModel;
  catalog: Catalog | null;
  priorLabel: (id: string) => string | undefined;
  onCorrect: (id: string, field: string, value: string) => Promise<void>;
  onRetry: (id: string, missingOnly?: boolean) => Promise<boolean>;
  onJump: (id: string) => void;
  onDismiss: (key: string) => void;
  onClose: () => void;
}

export function DocumentWorkspace({ row, catalog, priorLabel, onCorrect, onRetry, onJump, onDismiss, onClose }: Props) {
  const doc = row.doc;
  const [details, setDetails] = useState(!doc);
  return <section className="document-workspace" id="document-workspace" aria-label="Document viewer" data-filename={row.filename}>
    <header className="document-workspace-head">
      <h2>{doc?.fields.find(f => f.field === "title")?.value || row.filename}</h2>
      <button type="button" className="btn btn-small" onClick={onClose}>Close document</button>
    </header>
    <div className="document-workspace-tools">
      <div role="group" aria-label="Document view">
        <button className="btn btn-small" aria-pressed={!details} onClick={() => setDetails(false)}>Document</button>
        <button className="btn btn-small" aria-pressed={details} onClick={() => setDetails(true)}>Filing details</button>
      </div>
      {doc && <a className="cell-link" href={`/api/documents/${doc.id}/file`}>Download original</a>}
    </div>
    {details ? <div className="document-filing">
      <div className="reg-detail-head"><span className="reg-file">{row.filename}</span><HeadState row={row} onRetry={onRetry} onDismiss={onDismiss} /></div>
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
          <strong>Kept, not filed.</strong> {notFiledReason(doc)}
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
      {doc?.status === "filed" && doc.filename.toLowerCase().endsWith(".pdf") && <VisualIdentityReview doc={doc} onCorrect={onCorrect} />}
    </div> : doc ? <DocumentPreview key={doc.id} doc={doc} /> : <p role="status">The document will be available when its upload completes.</p>}
  </section>;
}

function DocumentPreview({ doc }: { doc: Doc }) {
  const markdown = /\.(md|markdown)$/i.test(doc.filename);
  const pdf = /\.pdf$/i.test(doc.filename);
  const [text, setText] = useState<string | null>(null);
  const [error, setError] = useState(false);
  useEffect(() => {
    if (!markdown) return;
    const controller = new AbortController();
    fetch(`/api/documents/${doc.id}/file`, { signal: controller.signal })
      .then(async response => {
        if (!response.ok) throw new Error("File unavailable");
        const value = await response.text();
        if (!controller.signal.aborted) setText(value);
      })
      .catch(() => { if (!controller.signal.aborted) setError(true); });
    return () => controller.abort();
  }, [doc.id, markdown]);
  if (pdf) return <div className="document-preview">
    <iframe title={`Document: ${doc.filename}`} src={`/api/documents/${doc.id}/file?view=1#view=FitH`} />
    <p className="document-preview-help">If the PDF does not display, <a className="cell-link" href={`/api/documents/${doc.id}/file?view=1`} target="_blank" rel="noreferrer">open it in a new tab</a> or download the original.</p>
  </div>;
  if (markdown) return error ? <p role="alert">Could not load this document. Close and reopen it to retry, or download the original.</p>
    : text === null ? <p role="status">Loading document…</p> : <pre className="document-text">{text}</pre>;
  return <p className="document-preview-help">An in-app preview is not available for this file type yet. Download the original to open it. Its filing details are available above.</p>;
}
