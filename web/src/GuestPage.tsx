import { useEffect, useRef, useState, type DragEvent } from "react";
import { TraceBand } from "./TracePage";
import { Wordmark } from "./Wordmark";

// The one page a guest sees (docs/plans/2026-11-17-guest-access.md, Task 9):
// what was shared, nothing else. No navigation, no account, no chat, no
// sign-up. The token rides in the link's fragment, which the browser never
// sends to a server; it is posted once for the guest cookie and cleared
// from the address bar.

interface GuestDocument { id: string; number: string; title: string; revision: string }
interface GuestIssue { id: string; number: number; reporting_date: string }
interface Page {
  kind: "documents" | "report" | "package";
  title: string;
  project: string;
  from: string;
  to: string;
  works_until: string;
  documents: GuestDocument[];
  issues: GuestIssue[];
  quote?: { closed?: string; received: { filename: string; received_on: string }[] };
}

const GONE = "This link has expired or been withdrawn. Ask the person who sent it for a new one.";
const QUOTE_TYPES = [".pdf", ".docx", ".xlsx"];
const MAX_QUOTE = 50 << 20;

function shown(iso: string) {
  return new Date(`${iso.slice(0, 10)}T00:00:00`).toLocaleDateString("en-AU", { day: "numeric", month: "short", year: "numeric" });
}

async function guestCall(method: string, path: string, body?: BodyInit, json = false): Promise<Response> {
  const res = await fetch(`/api/guest${path}`, {
    method, credentials: "same-origin", body,
    headers: json ? { "Content-Type": "application/json" } : undefined,
  });
  if (!res.ok) {
    const text = (await res.text()).trim();
    throw new Error(res.status === 401 ? GONE : text || "Something went wrong. Try again.");
  }
  return res;
}

export default function GuestPage() {
  const [page, setPage] = useState<Page | null>(null);
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");
  const [sending, setSending] = useState(false);
  const [over, setOver] = useState(false);
  const input = useRef<HTMLInputElement>(null);

  const load = async () => {
    try { setPage(await (await guestCall("GET", "")).json() as Page); setError(""); }
    catch (e) { setPage(null); setError(e instanceof Error ? e.message : GONE); }
  };

  useEffect(() => {
    document.title = "Shared with you · SiteWise";
    const token = new URLSearchParams(window.location.hash.slice(1)).get("t");
    // Clear the token from the address bar and history before anything else.
    if (window.location.hash) window.history.replaceState(null, "", "/g");
    void (async () => {
      if (token) {
        try { await guestCall("POST", "/session", JSON.stringify({ token }), true); }
        catch (e) { setError(e instanceof Error ? e.message : GONE); return; }
      }
      await load();
    })();
  }, []);

  const send = async (file: File) => {
    const ext = file.name.slice(file.name.lastIndexOf(".")).toLowerCase();
    if (!QUOTE_TYPES.includes(ext)) { setStatus("Send a PDF, Word or Excel file."); return; }
    if (file.size > MAX_QUOTE) { setStatus("The file is larger than 50 MB."); return; }
    setSending(true); setStatus("");
    try {
      const res = await guestCall("POST", `/quote?name=${encodeURIComponent(file.name)}`, file);
      const got = await res.json() as { filename: string; received_on: string };
      setStatus(`Received ${got.filename}, ${shown(got.received_on)}.`);
      await load();
    } catch (e) {
      setStatus(e instanceof Error ? e.message : "The file could not be sent. Try again.");
      if (e instanceof Error && e.message === GONE) { setPage(null); setError(GONE); }
    } finally { setSending(false); }
  };
  const drop = (e: DragEvent) => {
    e.preventDefault(); setOver(false);
    const file = e.dataTransfer.files[0];
    if (file) void send(file);
  };

  return (
    <div className="trace-home guest-page">
      <header className="trace-home-bar"><Wordmark height={28} /></header>
      <main id="main" className="trace-home-sheet">
        {!page ? <div className="guest-gone">
          {error ? <p role="alert">{error}</p> : <p role="status">Opening…</p>}
        </div> : <>
          <TraceBand title={page.title} meta={<>{page.project} · from {page.from} to {page.to} · works until {shown(page.works_until)}</>} />
          {page.issues.length > 0 && <section className="guest-section" aria-label="Issues">
            <h2 className="guest-head">{page.kind === "package" ? "Request" : "Report"}</h2>
            <table className="pl-table"><tbody>
              {page.issues.map((i, n) => <tr key={i.id}>
                <th scope="row">Issue {i.number}{n === 0 && page.issues.length > 1 ? " (latest)" : ""}</th>
                <td>{shown(i.reporting_date)}</td>
                <td className="pl-num"><a className="btn btn-small" href={`/api/guest/issues/${i.id}/file`} download>Download</a></td>
              </tr>)}
            </tbody></table>
          </section>}
          {page.documents.length > 0 && <section className="guest-section" aria-label="Documents">
            <h2 className="guest-head">Documents</h2>
            <table className="pl-table">
              <thead><tr><th scope="col">Number</th><th scope="col">Title</th><th scope="col">Revision</th><th scope="col"><span className="sr-only">Download</span></th></tr></thead>
              <tbody>{page.documents.map(d => <tr key={d.id}>
                <th scope="row">{d.number}</th><td>{d.title}</td><td>{d.revision}</td>
                <td className="pl-num"><a className="btn btn-small" href={`/api/guest/documents/${d.id}/file`} download>Download</a></td>
              </tr>)}</tbody>
            </table>
          </section>}
          {page.kind === "documents" && page.documents.length === 0 && <p className="guest-note">These documents are no longer shared.</p>}
          {page.quote && <section className="guest-section" aria-label="Your quote">
            <h2 className="guest-head">Your quote</h2>
            {page.quote.closed ? <p className="guest-note">{page.quote.closed}</p> : <div className="guest-drop file-drop" data-over={over ? "true" : undefined}
              onDragOver={e => { if (e.dataTransfer.types.includes("Files")) { e.preventDefault(); setOver(true); } }}
              onDragLeave={() => setOver(false)} onDrop={drop}>
              <p>Drop your quote here, or</p>
              <button type="button" className="btn" disabled={sending} onClick={() => input.current?.click()}>{sending ? "Sending…" : "Choose a file"}</button>
              <p className="guest-note">PDF, Word or Excel, up to 50 MB</p>
              <input ref={input} type="file" hidden accept={QUOTE_TYPES.join(",")} onChange={e => { const f = e.target.files?.[0]; e.target.value = ""; if (f) void send(f); }} />
            </div>}
            {status && <p className="guest-note" role="status">{status}</p>}
            {page.quote.received.length > 0 && <ul className="guest-received">
              {page.quote.received.map((r, i) => <li key={i}>Received {r.filename}, {shown(r.received_on)}</li>)}
            </ul>}
          </section>}
        </>}
      </main>
    </div>
  );
}
