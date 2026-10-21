import { Pending } from "./Activity";
import { useEffect, useState } from "react";
import { api, type Doc, type VisualReview } from "./api";
import "./visual-identity.css";

export function VisualIdentityReview({ doc, onCorrect }: {doc: Doc; onCorrect: (id: string, field: string, value: string) => Promise<void>}) {
  const [review, setReview] = useState<VisualReview | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [image, setImage] = useState(0);
  const [enlarged, setEnlarged] = useState(false);
  useEffect(() => {
    let stopped = false;
    let timer: ReturnType<typeof setTimeout>;
    const load = async () => {
      try {
        const next = await api.visualReview(doc.id);
        if (stopped) return;
        setReview(next); setError("");
        if (next.status === "queued" || next.status === "reading") timer = setTimeout(load, 3000);
      } catch { if (!stopped) setError("Could not load image review. Close and reopen this document to try again."); }
    };
    void load();
    return () => { stopped = true; clearTimeout(timer); };
  }, [doc.id, busy]);
  const request = async () => {
    setBusy("request"); setError("");
    try { setReview(await api.requestVisualReview(doc.id)); }
    catch { setError("Could not start image review. Try again."); }
    finally { setBusy(""); }
  };
  const use = async (field: string, value: string) => {
    setBusy(field); setError("");
    try { await onCorrect(doc.id, field, value); }
    catch { setError("Could not save this value. Check the filing and try again."); }
    finally { setBusy(""); }
  };
  const selected = review?.result.images?.[image];
  return <section className="visual-review" aria-label="Image-assisted filing review">
    <h3>Check against the document</h3>
    {error && <p role="alert">{error}</p>}
    {!review && !error && <p role="status"><Pending delayed>Loading image review…</Pending></p>}
    {review?.status === "reading" && <p role="status"><Pending>Reading document images…</Pending> You can keep working.</p>}
    {review?.status === "queued" && <p role="status"><Pending queued>Waiting for image reading…</Pending> You can keep working.</p>}
    {(review?.status === "not_requested" || review?.status === "unavailable") && <p>
      {review.status === "unavailable" ? "Image reading could not complete. Your saved details are unchanged. " : "Read the document images to suggest titles and revision labels. "}
      <button className="cell-link" disabled={!!busy} onClick={() => void request()}>{busy ? <Pending delayed>Starting…</Pending> : review.status === "unavailable" ? "Retry image reading" : "Read document images"}</button>
    </p>}
    {review?.status === "ready" && <>
      <p className="muted">Suggestions only · {review.result.pages === 1 ? "Page 1" : `First ${review.result.pages} pages`} checked. Compare the source before using a value.</p>
      <div className="visual-review-layout">
        <div>
          {(["title", "number", "revision"] as const).map(field => {
            const suggestion = review.result.suggestions?.[field];
            const current = doc.fields.find(f => f.field === field);
            const equal = !!suggestion && current?.value === suggestion.value;
            return <div className="visual-review-field" key={field}>
              <strong>{field === "title" ? "Title" : field === "number" ? "Document number" : "Revision"}</strong>
              <p>{suggestion?.value || "No clear suggestion"}</p>
              {suggestion && <span>Page {suggestion.page || 1} · {equal ? "Matches saved value" : current?.decided_by === "user" ? "You have already edited this field" : <button className="cell-link" disabled={!!busy} onClick={() => void use(field, suggestion.value)}>{busy === field ? <Pending delayed>Saving…</Pending> : `Use ${field}`}</button>}</span>}
            </div>;
          })}
        </div>
        <div className="visual-review-evidence">
          <div className="visual-review-tabs" aria-label="Source images">{review.result.images?.map((im, i) => <button key={i} className="cell-link" aria-pressed={image === i} onClick={() => setImage(i)}>Page {im.page}: {im.label}</button>)}</div>
          {selected && <>
            <button className="cell-link" aria-pressed={enlarged} onClick={() => setEnlarged(!enlarged)}>{enlarged ? "Fit image" : "Enlarge image"}</button>
            <div className="visual-review-image" data-enlarged={enlarged || undefined} tabIndex={enlarged ? 0 : undefined} aria-label="Source image; scroll to inspect when enlarged"><img src={`data:image/jpeg;base64,${selected.data}`} alt={`Page ${selected.page}: ${selected.label}`} /></div>
          </>}
          <a href={`/api/documents/${doc.id}/file?view=1`} target="_blank" rel="noreferrer">Open original PDF</a>
        </div>
      </div>
    </>}
  </section>;
}
