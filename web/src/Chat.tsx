import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import { ApiError, request } from "./api";
import type { WorkbenchView } from "./LeftNav";
import { REPORT_PLACE, dialogOf, hostOf, screenSelection, tableOf, whenReady, type ChatPlace } from "./chatView";
import "./chat.css";

// The chat strip: one input at the foot of the middle panel. A typed
// instruction is carried out at once and shown as a card with Undo; a
// change that needs the PM's word waits on a card with Apply. This file is
// loaded after the workbench has rendered, so opening a project costs
// nothing for it (docs/plans/2026-11-15-chat-agent.md, Task 9).

interface Mark { start: number; end: number; kind: "fact" | "quote" | "source" | "seen" | "check" | "verdict"; note?: string; document_id?: string; page?: number }
interface Answer { text: string; marks?: Mark[]; question?: boolean; coverage?: string; unchecked?: number }
interface Card { kind: "change" | "prepared"; area: string; lines: string[] | null; change_id?: string; undo?: string; prepared_id?: string; why?: string; refs?: string[]; program_version?: number }
/** A screen move from the view tool: ids are full ids Go checked against the project. */
interface View { do: "show" | "open" | "select" | "only" | "clear"; table?: string; ids?: string[]; dialog?: string; fields?: Record<string, string>; line: string; shown?: string }
interface ChatEvent { type: string; text?: string; tool?: string; card?: Card; answer?: Answer; view?: View }
interface StoredTurn { id: string; text: string; status: string; answer?: { events?: ChatEvent[] } }

interface Turn {
  key: string;
  text: string;
  status: "running" | "done" | "failed";
  doing: string;
  streamed: string;
  answer?: Answer;
  cards: CardState[];
  /** What each screen move did, in Go's words. */
  moved: string[];
  error?: string;
}
interface CardState { card: Card; state: "" | "busy" | "undone" | "dismissed"; note?: string }

const VIEW: Record<string, WorkbenchView> = { cost_plan: "costs", costs: "costs", profile: "profile", program: "program", procurement: "procurement", specifications: "specifications", register: "register", delivery: "register" };
const AREA: Record<string, string> = { cost_plan: "Cost Plan", profile: "Profile", program: "Program", procurement: "Procurement" };
const DOING: Record<string, string> = {
  read: "Reading the records…", find: "Looking it up…", apply: "Making the change…", calc: "Working it out…",
  documents: "Looking through the documents…", passages: "Reading the documents…", messages: "Checking the inbox…", view: "Moving the screen…",
};
// Where each table chat can name lives (internal/httpapi/chat_screen.go).
const PLACE: Record<string, ChatPlace> = { documents: "documents", cost_lines: "costs", program: "program", procurement: "procurement", register: "register", variations: "costs", claims: "costs", claim_rows: "costs" };

/** The key Go's undo names a change by: its change id, or the Program version it left. */
const undoKey = (c: Card) => c.undo === "program" ? `program:${c.program_version ?? ""}` : c.change_id ?? "";
const markUndone = (turns: Turn[], key: string): Turn[] => turns.map(t => ({ ...t, cards: t.cards.map(c => c.card.kind === "change" && undoKey(c.card) === key ? { ...c, state: "undone" } : c) }));

function fromStored(t: StoredTurn): Turn {
  const turn: Turn = { key: t.id, text: t.text, status: t.status === "failed" ? "failed" : "done", doing: "", streamed: "", cards: [], moved: [] };
  for (const e of t.answer?.events ?? []) {
    // A stored move is shown, never made again: the screen is the PM's now.
    if (e.type === "view" && e.view) turn.moved.push(e.view.line);
    if (e.card) turn.cards.push({ card: e.card, state: "" });
    if (e.type === "answer" && e.answer) turn.answer = e.answer;
    if (e.type === "error") turn.error = e.text;
  }
  return turn;
}

// Marked renders the checked answer: Go's figures, quotations that link to
// their page, and anything the assistant typed itself under the squared
// highlighter ("check it").
function Marked({ answer, onOpenDocument }: { answer: Answer; onOpenDocument: (id: string) => void }) {
  const out: ReactNode[] = [];
  let at = 0;
  const marks = (answer.marks ?? []).filter(m => m.kind !== "seen").sort((a, b) => a.start - b.start);
  marks.forEach((m, i) => {
    if (m.start < at) return;
    out.push(answer.text.slice(at, m.start));
    const words = answer.text.slice(m.start, m.end);
    if ((m.kind === "quote" || m.kind === "source") && m.document_id) {
      const id = m.document_id;
      out.push(<button key={i} type="button" className="chat-quote" title={m.note} onClick={() => onOpenDocument(id)}>{words}</button>);
    } else if (m.kind === "fact") {
      out.push(<span key={i} className="chat-fact" title={m.note}>{words}</span>);
    } else {
      out.push(<mark key={i} className="mark-check chat-check" title={m.note}>{words}</mark>);
    }
    at = m.end;
  });
  out.push(answer.text.slice(at));
  return <p className="chat-answer">{out}</p>;
}

// moveScreen makes one view event happen: the table's view is shown first,
// then its setter is called once the table has registered. It resolves with
// a note when the move could not be made, or "".
async function moveScreen(v: View, onShow: (view: WorkbenchView) => void, onShowDocuments: () => void): Promise<string> {
  if (v.dialog === "award") {
    const pkg = v.fields?.package ?? "", column = v.fields?.column ?? "";
    onShow("procurement");
    const table = await whenReady(() => tableOf("procurement"));
    table?.open?.(pkg);
    const dialog = await whenReady(() => dialogOf(`award:${pkg}`));
    if (!dialog) return "The award dialog could not be opened. Open the package's prices and press Award.";
    dialog.open({ package: pkg, column });
    return "";
  }
  if (v.dialog === "share") {
    // Share is the project's one dialog; chat opens it filled in, the PM presses Create link.
    const dialog = await whenReady(() => dialogOf("share"));
    if (!dialog) return "Share could not be opened. Use Share… on the document, report or invited firm.";
    dialog.open(v.fields ?? {});
    return "";
  }
  if (v.dialog === "claim") {
    // A claim's panel opens filled in; the PM presses Issue there. Chat never issues.
    onShow("costs");
    const dialog = await whenReady(() => dialogOf("claim"));
    if (!dialog) return "The claim could not be opened. Open it on the Variations and claims tab.";
    dialog.open(v.fields ?? {});
    return "";
  }
  if (v.dialog === "report") {
    // The project report opens in its reader; with issue, its issue form
    // opens filled in and the PM presses Issue. Chat never issues.
    const report = v.fields?.report ?? "";
    onShow(REPORT_PLACE);
    const entry = await whenReady(() => dialogOf("report"));
    if (!entry) return "The project report could not be opened. Open it with Project report.";
    entry.open({ report });
    if (v.fields?.issue !== "yes") return "";
    const form = await whenReady(() => dialogOf(`report-issue:${report}`));
    if (!form) return "The issue form could not be opened. Press Issue report on the report.";
    form.open({});
    return "";
  }
  const place = PLACE[v.table ?? ""];
  if (!place) return "";
  if (place === "documents") onShowDocuments(); else onShow(place);
  hostOf(v.table ?? "")?.();
  if (v.do === "show") return "";
  const table = await whenReady(() => tableOf(v.table ?? ""));
  if (!table) return "That table isn't on screen yet. Open it and ask again.";
  const ids = v.ids ?? [];
  if (v.do === "select") table.select(ids);
  else if (v.do === "only") table.only(ids, v.shown);
  else if (v.do === "clear") table.clear();
  else if (v.do === "open" && ids[0]) table.open?.(ids[0]);
  return "";
}

export default function Chat({ projectId, view, openDocument, onShow, onShowDocuments, onOpenDocument }: {
  projectId: string; view: WorkbenchView; openDocument: string | null;
  onShow: (view: WorkbenchView) => void; onShowDocuments: () => void; onOpenDocument: (id: string) => void;
}) {
  const [turns, setTurns] = useState<Turn[]>([]);
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [open, setOpen] = useState(false);
  const [unavailable, setUnavailable] = useState("");
  const conversation = useRef("");
  const warmed = useRef(false);
  const log = useRef<HTMLDivElement>(null);
  const input = useRef<HTMLInputElement>(null);
  const base = `/projects/${projectId}`;

  useEffect(() => {
    let live = true;
    conversation.current = "";
    request<{ available: boolean; unavailable: string; turns: (StoredTurn & { conversation_id: string })[] }>("GET", `${base}/chat`).then(v => {
      if (!live) return;
      setUnavailable(v.available ? "" : v.unavailable);
      let shown = v.turns.map(fromStored);
      for (const t of v.turns) for (const e of t.answer?.events ?? []) if (e.type === "undone" && e.text) shown = markUndone(shown, e.text);
      setTurns(shown);
      conversation.current = v.turns.at(-1)?.conversation_id ?? "";
    }, () => { if (live) setUnavailable("Chat can't be reached right now."); });
    return () => { live = false; };
  }, [base]);

  useEffect(() => { log.current?.scrollTo({ top: log.current.scrollHeight }); }, [turns, open]);

  const patch = useCallback((key: string, change: (t: Turn) => Turn) => setTurns(all => all.map(t => t.key === key ? change(t) : t)), []);
  const patchCard = (key: string, index: number, change: Partial<CardState>) => patch(key, t => ({ ...t, cards: t.cards.map((c, i) => i === index ? { ...c, ...change } : c) }));

  // Focusing the strip warms the model's cache, once, so the first command is quick.
  const warm = () => {
    if (warmed.current || unavailable) return;
    warmed.current = true;
    request("POST", `${base}/chat/warm`, {}).catch(() => undefined);
  };

  async function send(e: FormEvent) {
    e.preventDefault();
    const typed = text.trim();
    if (!typed || busy) return;
    const key = `new-${Date.now()}`;
    setText("");
    setBusy(true);
    setTurns(all => [...all, { key, text: typed, status: "running", doing: "Thinking…", streamed: "", cards: [], moved: [] }]);
    const fail = (message: string) => patch(key, t => ({ ...t, status: "failed", doing: "", error: message }));
    // What is on screen as Enter is pressed: ids and counts only, never titles.
    const screen = { view, open: openDocument ? { kind: "document", id: openDocument } : undefined, selected: screenSelection(view) };
    try {
      const res = await fetch(`/api${base}/chat`, {
        method: "POST", credentials: "same-origin", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ conversation_id: conversation.current, text: typed, screen }),
      });
      if (!res.ok || !res.body) {
        fail(res.status === 404 ? "This conversation is no longer available. Nothing was changed." : "Chat is unavailable. Nothing was changed.");
        return;
      }
      const reader = res.body.pipeThrough(new TextDecoderStream()).getReader();
      let pending = "";
      for (;;) {
        const { value, done } = await reader.read();
        if (done) break;
        pending += value;
        const frames = pending.split("\n\n");
        pending = frames.pop() ?? "";
        for (const frame of frames) {
          if (!frame.startsWith("data: ")) continue;
          const ev = JSON.parse(frame.slice(6)) as ChatEvent & { conversation_id?: string };
          if (ev.type === "text") patch(key, t => ({ ...t, doing: "", streamed: t.streamed + (ev.text ?? "") }));
          else if (ev.type === "tool") patch(key, t => ({ ...t, doing: DOING[ev.tool ?? ""] ?? "Working…", streamed: "" }));
          else if (ev.card) patch(key, t => ({ ...t, doing: "", cards: [...t.cards, { card: ev.card!, state: "" }] }));
          else if (ev.type === "answer") patch(key, t => ({ ...t, doing: "", streamed: "", answer: ev.answer }));
          else if (ev.type === "view" && ev.view) {
            const v = ev.view;
            patch(key, t => ({ ...t, doing: "", moved: [...t.moved, v.line] }));
            void moveScreen(v, onShow, onShowDocuments).then(note => { if (note) patch(key, t => ({ ...t, error: note })); });
          }
          else if (ev.type === "undone" && ev.text) { const undone = ev.text; setTurns(all => markUndone(all, undone)); }
          else if (ev.type === "go") { const view = VIEW[ev.text ?? ""]; if (view) onShow(view); patch(key, t => ({ ...t, doing: "", answer: { text: `Opened ${AREA[ev.text ?? ""] ?? ev.text}.` } })); }
          else if (ev.type === "error") fail(ev.text ?? "Chat is unavailable. Nothing was changed.");
          else if (ev.type === "done") { conversation.current = ev.conversation_id ?? conversation.current; patch(key, t => t.status === "running" ? { ...t, status: "done", doing: "" } : t); }
        }
      }
      patch(key, t => t.status === "running" ? { ...t, status: "failed", doing: "", error: "The connection dropped. Check the screen before trying again." } : t);
    } catch {
      fail("Can't reach SiteWise. Check the screen before trying again.");
    } finally {
      setBusy(false);
      // Back to the strip, unless the PM has moved on to a cell editor or
      // another control while the turn ran: a view never takes focus.
      const active = document.activeElement;
      if (!active || active === document.body || active.closest(".chat")) input.current?.focus();
    }
  }

  async function undo(key: string, index: number, card: Card) {
    patchCard(key, index, { state: "busy", note: "" });
    try {
      if (card.undo === "program") {
        const program = await request<{ program: { version: number } }>("GET", `${base}/program`);
        await request("POST", `${base}/program/undo`, { version: program.program.version });
      } else {
        await request("POST", `${base}/changes/${card.change_id}/undo`, {});
      }
      patchCard(key, index, { state: "undone" });
    } catch (err) {
      const changed = err instanceof ApiError && (err.status === 422 || err.status === 409);
      patchCard(key, index, { state: "", note: changed ? "Can't undo: this has changed since." : "Undo didn't go through. Try again." });
    }
  }

  async function decide(key: string, index: number, card: Card, apply: boolean) {
    patchCard(key, index, { state: "busy", note: "" });
    try {
      const done = await request<Card>("POST", `${base}/chat/prepared/${card.prepared_id}/${apply ? "apply" : "dismiss"}`, {});
      if (apply) patchCard(key, index, { card: done, state: "" });
      else patchCard(key, index, { state: "dismissed" });
    } catch (err) {
      let note = "That didn't go through. Try again.";
      if (err instanceof ApiError && err.status === 422) { try { note = (JSON.parse(err.message) as { message: string }).message; } catch { /* keep the plain note */ } }
      if (err instanceof ApiError && err.status === 409) note = "This was already decided.";
      patchCard(key, index, { state: "", note });
    }
  }

  const shown = open ? turns : turns.slice(-1);
  return (
    <section className={`chat${open ? " chat-open" : ""}`} aria-label="Chat">
      {shown.length > 0 && (
        <div className="chat-log" ref={log} role="log" aria-live="polite">
          {shown.map(t => (
            <article key={t.key} className="chat-turn">
              <p className="chat-typed">{t.text}</p>
              {t.cards.map(({ card, state, note }, i) => (
                <div key={i} className={`chat-card chat-card-${card.kind}${state === "undone" || state === "dismissed" ? " chat-card-spent" : ""}`}>
                  <p className="chat-card-head">{card.kind === "prepared" ? "Ready to apply" : state === "undone" ? "Undone" : "Changed"} · {AREA[card.area] ?? card.area}</p>
                  <ul>{(card.lines ?? []).map((line, n) => <li key={n}>{line}</li>)}</ul>
                  {card.kind === "prepared" && card.why && state !== "dismissed" && <p className="chat-why">{card.why}</p>}
                  {note && <p className="chat-note" role="alert">{note}</p>}
                  <div className="chat-card-actions">
                    {card.kind === "prepared" && state !== "dismissed" && <>
                      <button type="button" className="btn btn-small btn-primary" disabled={state === "busy"} onClick={() => decide(t.key, i, card, true)}>Apply</button>
                      <button type="button" className="btn btn-small" disabled={state === "busy"} onClick={() => decide(t.key, i, card, false)}>Dismiss</button>
                    </>}
                    {state === "dismissed" && <span className="chat-muted">Dismissed. Nothing was changed.</span>}
                    {card.kind === "change" && VIEW[card.area] && <button type="button" className="btn btn-small" onClick={() => { onShow(VIEW[card.area]); setOpen(false); }}>Show</button>}
                    {card.kind === "change" && card.undo && card.undo !== "none" && state !== "undone" && <button type="button" className="btn btn-small" disabled={state === "busy"} onClick={() => undo(t.key, i, card)}>Undo</button>}
                  </div>
                </div>
              ))}
              {t.moved.map((line, i) => <p key={`moved-${i}`} className="chat-answer">{line}</p>)}
              {t.doing && <p className="chat-doing">{t.doing}</p>}
              {t.streamed && !t.answer && <p className="chat-answer">{t.streamed}</p>}
              {t.answer?.text && <Marked answer={t.answer} onOpenDocument={onOpenDocument} />}
              {t.answer?.coverage && <p className="chat-coverage">{t.answer.coverage}</p>}
              {!!t.answer?.unchecked && <p className="chat-coverage">Highlighted words are the assistant's own, not from your records. Check them.</p>}
              {t.error && <p className="chat-note" role="alert">{t.error}</p>}
            </article>
          ))}
        </div>
      )}
      <form className="chat-strip" onSubmit={send}>
        <label className="sr-only" htmlFor="chat-input">Tell SiteWise what to change, or ask about this project</label>
        <input id="chat-input" ref={input} value={text} onChange={e => setText(e.target.value)} onFocus={warm} disabled={!!unavailable} autoComplete="off" maxLength={4000}
          placeholder={unavailable || "Tell SiteWise what to change, or ask about this project"} />
        {unavailable.includes("Account") && <a className="chat-account" href="/account">Open Account</a>}
        <button type="submit" className="btn btn-small" disabled={busy || !text.trim() || !!unavailable}>{busy ? "Working…" : "Send"}</button>
        <button type="button" className="chat-expand" aria-expanded={open} aria-label={open ? "Collapse chat" : "Expand chat"} title={open ? "Collapse chat" : "Expand chat"} onClick={() => setOpen(o => !o)}>
          <svg viewBox="0 0 16 16" width="16" height="16" aria-hidden="true"><path d={open ? "M3 6l5 5 5-5" : "M3 10l5-5 5 5"} fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" /></svg>
        </button>
      </form>
    </section>
  );
}
