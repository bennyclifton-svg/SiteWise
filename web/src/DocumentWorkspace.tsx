import { useEffect, useId, useLayoutEffect, useRef, useState, type RefObject } from "react";
import { Pending } from "./Activity";
import type { Catalog, Doc } from "./api";
import { IconCheck, IconDownload, IconOpen, IconPrint, IconRetry, IconTag } from "./icons";
import { BackButton, MoreMenu, type MenuItem } from "./PageActions";
import { FIELDS, FieldCell, HeadState, notFiledReason, stateOf, type RowModel } from "./DocumentRow";
import { OCRStatus } from "./OCRStatus";
import { VisualIdentityReview } from "./VisualIdentityReview";
import "./document-workspace.css";

interface Props {
  row: RowModel;
  catalog: Catalog | null;
  /** The view this document was opened over, and returns to. */
  backTo: string;
  priorLabel: (id: string) => string | undefined;
  onCorrect: (id: string, field: string, value: string) => Promise<void>;
  onRetry: (id: string, missingOnly?: boolean) => Promise<boolean>;
  /** Splits a drawing set the automatic split stopped on. */
  onSplit: (id: string) => Promise<boolean>;
  onJump: (id: string) => void;
  onDismiss: (key: string) => void;
  onClose: () => void;
}

// Filing sits beside the page once the pane can hold both: a readable page
// (about 520px) and the filing column. Narrower, it takes the page's place.
const BESIDE_MIN = 860;
const FILING_KEY = "sitewise.document.filing.v1";

function readFilingClosed(): boolean {
  try { return localStorage.getItem(FILING_KEY) === "closed"; } catch { return false; }
}

function storeFilingClosed(closed: boolean) {
  try { localStorage.setItem(FILING_KEY, closed ? "closed" : "open"); } catch { /* A private window can refuse storage; the choice still holds for this visit. */ }
}

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

/** A stored ISO date in words; anything else exactly as it was filed. */
function longDate(value: string): string {
  const m = value.match(/^(\d{4})-(\d{2})-(\d{2})/);
  const month = m ? MONTHS[Number(m[2]) - 1] : undefined;
  return m && month ? `${Number(m[3])} ${month} ${m[1]}` : value;
}

/** The identity under the title: only what is filed, so a blank stays out. */
function identity(doc: Doc, catalog: Catalog | null, titled: boolean): string[] {
  const value = (field: string) => doc.fields.find(f => f.field === field)?.value ?? "";
  const revision = value("revision");
  const discipline = value("discipline");
  return [
    value("number"),
    revision && (/^rev/i.test(revision) ? revision : `Rev ${revision}`),
    value("date") && longDate(value("date")),
    discipline && (catalog?.disciplines.find(o => o.id === discipline)?.label ?? discipline),
    titled ? doc.filename : "",
  ].filter(Boolean);
}

export function DocumentWorkspace({ row, catalog, backTo, priorLabel, onCorrect, onRetry, onSplit, onJump, onDismiss, onClose }: Props) {
  const doc = row.doc;
  const pdf = !!doc && /\.pdf$/i.test(doc.filename);
  const markdown = !!doc && /\.(md|markdown)$/i.test(doc.filename);
  const preview = pdf || markdown;
  const filed = doc?.status === "filed";
  const title = doc?.fields.find(f => f.field === "title")?.value ?? "";
  const toCheck = filed ? FIELDS.filter(spec => ["check", "unchecked"].includes(stateOf(doc.fields.find(f => f.field === spec.field)))).length : 0;

  const root = useRef<HTMLElement>(null);
  const frame = useRef<HTMLIFrameElement>(null);
  const filingId = useId();
  // Measured before the first paint, so the filing column never jumps in.
  const [roomy, setRoomy] = useState(false);
  useLayoutEffect(() => {
    const el = root.current;
    if (!el) return;
    const measure = () => setRoomy(el.getBoundingClientRect().width >= BESIDE_MIN);
    measure();
    if (typeof ResizeObserver === "undefined") return;
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);
  const layout = !preview ? "alone" : roomy ? "beside" : "over";
  // Beside the page it stays as the user last left it; over the page it starts
  // closed for each document, so opening one always shows the document.
  const [besideClosed, setBesideClosed] = useState(readFilingClosed);
  const [overOpen, setOverOpen] = useState(false);
  const filing = layout === "alone" || (layout === "beside" ? !besideClosed : overOpen);
  const toggleFiling = () => {
    if (layout === "beside") { storeFilingClosed(!besideClosed); setBesideClosed(!besideClosed); }
    else setOverOpen(open => !open);
  };

  // A suggestion's page: the browser's viewer only takes a page when it loads,
  // so each request loads the frame again at that page.
  const [at, setAt] = useState({ page: 1, request: 0 });
  const showPage = (page: number) => { setAt(now => ({ page, request: now.request + 1 })); setOverOpen(false); };

  // Escape returns to the view behind, as Back does. It is taken before the
  // register sees it, so the selection there is left as it was; an open editor
  // or menu keeps its own Escape. Keys pressed inside the PDF never reach here.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape" || e.defaultPrevented) return;
      const target = e.target instanceof Element ? e.target : null;
      if (target?.closest('input, select, textarea, [contenteditable="true"], [role="menu"], [role="listbox"], [role="dialog"], dialog, [aria-haspopup][aria-expanded="true"]')) return;
      e.preventDefault();
      e.stopPropagation();
      onClose();
    };
    document.addEventListener("keydown", onKey, true);
    return () => document.removeEventListener("keydown", onKey, true);
  }, [onClose]);

  // Stacked on a narrow screen the working view sits above the register, so
  // bring the document that was just opened into view.
  useEffect(() => {
    if (!window.matchMedia("(min-width: 1101px)").matches) root.current?.scrollIntoView({ block: "start" });
  }, []);

  const viewURL = doc ? `/api/documents/${doc.id}/file?view=1` : "";
  const print = () => {
    // The frame is the same origin, so the browser prints the PDF itself.
    try {
      const view = frame.current?.contentWindow;
      if (!view) throw new Error("no frame");
      view.focus();
      view.print();
    } catch {
      window.open(viewURL, "_blank", "noopener");
    }
  };
  const more: MenuItem[] = pdf ? [
    { label: "Open in new tab", icon: <IconOpen />, href: viewURL, newTab: true },
    { label: "Print", icon: <IconPrint />, onSelect: print },
  ] : [];

  return <section ref={root} className="document-workspace" id="document-workspace" aria-label="Document viewer" data-filename={row.filename} data-layout={layout} data-filing={filing || undefined}>
    <header className="document-head">
      <BackButton to={backTo} onClick={onClose} title={`Back to ${backTo} (Esc)`} />
      <div className="document-title">
        <h2 title={title || row.filename}>{title || row.filename}</h2>
        {doc && <p className="document-meta">{identity(doc, catalog, !!title).map((part, i) => <span key={i}>{part}</span>)}</p>}
      </div>
      <div className="document-actions">
        {preview && <button type="button" className="page-download document-filing-toggle" aria-expanded={filing} aria-controls={filingId} title={filing ? "Hide filing details" : "Show filing details"} onClick={toggleFiling}>
          <IconTag /><span>Filing</span>
          {toCheck > 0 && <span className="document-tocheck" title={`${toCheck} to check`}><IconCheck />{toCheck}<span className="sr-only"> to check</span></span>}
        </button>}
        {doc && <a className="page-download document-download" href={`/api/documents/${doc.id}/file`} aria-label="Download original" title="Download original"><IconDownload /><span>Download</span></a>}
        {more.length > 0 && <MoreMenu label="Document actions" items={more} />}
      </div>
    </header>
    <div className="document-body">
      {preview && <div className="document-page">
        {pdf ? <PdfPage key={`${doc.id}:${at.request}`} doc={doc} page={at.page} frame={frame} /> : <MarkdownPage key={doc.id} doc={doc} />}
      </div>}
      {filing && <aside className="document-filing" id={filingId} aria-label="Filing details">
        <div className="section-band"><h3>Filing</h3><span className="document-filing-state"><HeadState row={row} onRetry={onRetry} onDismiss={onDismiss} /></span></div>
        {!doc && <p className="tb-note" role="status">The document will be available when its upload completes.</p>}
        {doc && !preview && <p className="tb-note">There is no preview for this file type yet. Download the original to open it.</p>}
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
            {doc.expansion.can_split && (
              <button type="button" className="btn btn-small" onClick={() => onSplit(doc.id)}>
                Split into pages
              </button>
            )}
          </p>
        )}
        {doc?.status === "not_filed" && (
          <p className="tb-note">
            <strong>Kept, not filed.</strong> {notFiledReason(doc)}
          </p>
        )}
        {filed && (
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
        {filed && pdf && <VisualIdentityReview doc={doc} onCorrect={onCorrect} onShowPage={showPage} />}
      </aside>}
    </div>
  </section>;
}

/** The whole PDF in the browser's own viewer, without the browser's toolbar. */
function PdfPage({ doc, page, frame }: { doc: Doc; page: number; frame: RefObject<HTMLIFrameElement | null> }) {
  const [loaded, setLoaded] = useState(false);
  const open = `${page > 1 ? `page=${page}&` : ""}toolbar=0&navpanes=0&view=FitH`;
  return <div className="document-frame">
    {!loaded && <p className="document-wait" role="status"><Pending delayed>Loading document…</Pending></p>}
    <iframe ref={frame} title={`Document: ${doc.filename}`} src={`/api/documents/${doc.id}/file?view=1#${open}`} onLoad={() => setLoaded(true)} />
  </div>;
}

/** Markdown as literal text, never as HTML. */
function MarkdownPage({ doc }: { doc: Doc }) {
  const [text, setText] = useState<string | null>(null);
  const [error, setError] = useState(false);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    fetch(`/api/documents/${doc.id}/file`, { signal: controller.signal })
      .then(async response => {
        if (!response.ok) throw new Error("File unavailable");
        const value = await response.text();
        if (!controller.signal.aborted) setText(value);
      })
      .catch(() => { if (!controller.signal.aborted) setError(true); });
    return () => controller.abort();
  }, [doc.id, attempt]);
  if (error) return <div className="document-wait" role="alert">
    <p>Could not load this document. Retry, or download the original.</p>
    <button type="button" className="btn btn-small" onClick={() => { setError(false); setAttempt(n => n + 1); }}><IconRetry />Retry</button>
  </div>;
  if (text === null) return <p className="document-wait" role="status"><Pending delayed>Loading document…</Pending></p>;
  return <pre className="document-text" tabIndex={0}>{text}</pre>;
}
