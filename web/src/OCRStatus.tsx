import type { Doc } from "./api";
import { useState } from "react";
import { IconCheck, IconConfirmed, IconRetry } from "./icons";

const stages = [
  { reason: "ocr_queued", label: "Queued", title: "OCR queued", detail: "No selectable text found. Your file is saved and will be read automatically." },
  { reason: "ocr_reading", label: "Read lettering", title: "Reading lettering", detail: "Recovering text from the first page. You can keep working while this finishes." },
  { reason: "ocr_classifying", label: "File", title: "Filing recovered text", detail: "Checking document details and classification. Recovered values will be marked for review." },
];

export function ocrStage(doc?: Doc) {
  if (doc?.status === "filed" && doc.reason?.startsWith("ocr_details_")) {
    const stage = stages.find(s => s.reason === doc.reason!.replace("ocr_details_", "ocr_"));
    if (stage) return { ...stage, title: stage.reason === "ocr_queued" ? "Detail recovery queued" : stage.reason === "ocr_reading" ? "Reading drawing details" : "Checking missing details" };
  }
  return doc?.status === "pending" ? stages.find((s) => s.reason === doc.reason) : undefined;
}

const detailFields = [["title", "drawing title"], ["number", "drawing number"], ["discipline", "discipline"], ["revision", "revision"], ["kind", "document kind"]];

export function OCRStatus({ doc, compact = false, onReprocess }: { doc: Doc; compact?: boolean; onReprocess?: (id: string, missingOnly: boolean) => Promise<boolean> }) {
  const [requesting, setRequesting] = useState(false);
  const [error, setError] = useState("");
  const stage = ocrStage(doc);
  const recovery = doc.status === "filed" && !!doc.reason?.startsWith("ocr_details_");
  const complete = doc.status === "filed" && (doc.reason === "ocr_review" || doc.reason === "ocr_details_review" || doc.reason === "ocr_details_unchanged");
  const failed = recovery && !stage && !complete;
  const missing = detailFields.filter(([key]) => !doc.fields.some(f => f.field === key && (!!f.value || f.decided_by === "user"))).map(([, label]) => label);
  if (!stage && !complete && !failed) return null;
  const summary = stage?.title ?? (failed ? "Detail recovery stopped" : doc.reason === "ocr_details_unchanged" ? "No new details recovered" : recovery ? "Details recovered · check fields" : "OCR complete · check fields");
  const request = async () => {
    if (!onReprocess || requesting) return;
    setRequesting(true); setError("");
    try { if (!await onReprocess(doc.id, true)) setError("Recovery did not start. Check your connection and try again."); }
    catch { setError("Recovery did not start. Check your connection and try again."); }
    finally { setRequesting(false); }
  };
  if (compact) return (
    <span className="ocr-summary" data-complete={complete || undefined}>
      {!stage ? <IconCheck /> : <span className="reg-spin" aria-hidden="true" />}
      {summary}
    </span>
  );
  const active = stage ? stages.findIndex(s => s.reason === stage.reason) : stages.length;
  return (
    <section className="ocr-status" aria-label="Text recovery" role="status" aria-live="polite" aria-atomic="true">
      <div className="ocr-status-copy">
        <strong>{recovery ? summary : complete ? "Text recovered · review the fields" : stage!.title}</strong>
        <p>{recovery ? stage ? "Reading the first page for missing details. Saved values and your edits stay in place; you can keep working." : failed ? "Recovery could not finish. Your document is still filed and its saved details are unchanged. Try again, or check the original and edit the missing fields." : "Recovery finished. Check new values against the original; OCR can misread letters and numbers. Saved values and your edits were kept." : complete ? "OCR can misread letters and numbers. Check the recovered values against the original. Only the first page was read for filing." : stage!.detail}</p>
        {!stage && missing.length > 0 && <p><strong>Still missing:</strong> {missing.join(", ")}. {recovery && !failed ? "Check the original and edit any details that could not be recovered." : "Reprocess the first page to look for these details."}</p>}
      </div>
      {!failed && <ol className="ocr-steps" aria-label="OCR progress">
        {stages.map((s, i) => (
          <li key={s.reason} data-state={i < active ? "done" : i === active ? "active" : "waiting"} aria-current={i === active ? "step" : undefined}>
            <span className="ocr-step-mark" aria-hidden="true">{i < active ? <IconConfirmed /> : <span />}</span>
            {recovery && i === 2 ? "Check details" : s.label}<span className="sr-only">{i < active ? ", complete" : i === active ? ", in progress" : ", waiting"}</span>
          </li>
        ))}
      </ol>}
      {!stage && missing.length > 0 && onReprocess && <button type="button" className="btn btn-small" disabled={requesting} onClick={request}><IconRetry />{requesting ? "Queuing recovery…" : "Reprocess missing details"}</button>}
      {error && <p className="error-text" role="alert">{error}</p>}
    </section>
  );
}
