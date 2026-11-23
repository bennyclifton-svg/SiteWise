// The sheet of a received item: an EOT claim, a notice of delay or a letter
// (docs/plans/2026-11-23-correspondence-between-parties.md, section 18).
// The PM records what arrived and, on an EOT claim, the result: an outcome,
// the days granted and the reasons. Every field saves through POST /ops
// (delivery.edit) with Undo. Dates, lateness and totals are Go's; nothing
// here calls Jev or the model.

import { useState, type ReactNode } from "react";
import { longDate } from "./Delivery";
import { numberLabel, type ClaimDetails, type DeliveryRecord, type EOTResult } from "./deliveryApi";
import type { RegisterData } from "./registerData";
import { resultWords } from "./registerKinds";

interface Props {
  row: DeliveryRecord;
  data: RegisterData;
  onOpenMessage: (id: string) => void;
}

const OUTCOMES: [EOTResult["outcome"], string][] = [["granted", "Granted"], ["granted_in_part", "Granted in part"], ["rejected", "Rejected"], ["more_information", "More information needed"]];

// Standard words for each result, to keep or type over.
const WORDING: Record<EOTResult["outcome"], string> = {
  granted: "The claim is granted for the days stated.",
  granted_in_part: "The claim is granted in part, for the days stated, for these reasons:",
  rejected: "The claim is rejected, for these reasons:",
  more_information: "Before the claim can be assessed, please provide:",
};

export function ClaimDetail({ row, data, onOpenMessage }: Props) {
  const d = row.details as ClaimDetails;
  const save = (fields: Record<string, unknown>, label = "Saved.") =>
    data.run([{ op: "delivery.edit", fields: { item: row.id, ...fields } }], label, "sheet");
  const k = (name: string) => `${row.id}-${row.version}-${name}`;
  const text = (name: keyof ClaimDetails, label: string, opts: { long?: boolean; hint?: string; max?: number } = {}) => {
    const value = (d[name] as string | undefined) ?? "";
    const commit = (v: string) => { if (v !== value) void save({ [name]: v }); };
    return <label className="pf-row"><span className="pf-label">{label}</span><span className="pf-value">
      {opts.long
        ? <textarea className="input" key={k(name)} defaultValue={value} rows={3} maxLength={opts.max ?? 8000} placeholder={opts.hint} disabled={data.busy} aria-label={label} onBlur={e => commit(e.target.value)} />
        : <input className="input" key={k(name)} defaultValue={value} maxLength={opts.max ?? 200} placeholder={opts.hint} disabled={data.busy} aria-label={label} onBlur={e => commit(e.target.value)} />}
    </span></label>;
  };
  const day = (name: "received_on" | "delay_from" | "delay_to", label: string, after?: ReactNode) =>
    <label className="pf-row"><span className="pf-label">{label}</span><span className="pf-value">
      <input type="date" className="input register-date" key={k(name)} defaultValue={d[name] ?? ""} disabled={data.busy} aria-label={label}
        onBlur={e => { if (e.target.value !== (d[name] ?? "")) void save({ [name]: e.target.value || null }); }} />{after}</span></label>;

  const notices = (data.view?.items ?? []).filter(r => r.kind === "delay_notice");
  const period = row.kind === "eot_claim"
    ? `${data.view?.eot_days ?? 28} days to respond${data.view?.eot_typical ? ", SiteWise's default; check your contract" : ", the project's period"}`
    : row.kind === "letter" ? `${data.view?.response_days ?? 5} working days to respond${data.view?.response_typical ? " (typical)" : ""}` : "No response is owed; it is recorded";
  const dueLine = row.target_date ? `Response due ${longDate(row.target_date)}` : row.kind === "delay_notice" ? "" : "No due date: add the date it was received";

  return <>
    <p className="register-send-what" role="status">{dueLine ? `${dueLine}. ` : ""}{row.target_date && row.due_basis === "typed" ? "You typed this date." : period}.</p>
    <div className="pf-grid register-grid">
      {day("received_on", "Received")}
      {text("raised_by", "From", { hint: "Who sent it: a firm or a person" })}
      {text("ref", "Their ref", { max: 100, hint: "Their own number, if any" })}
      {row.kind !== "letter" && text("cause", "Cause", { long: true, hint: "What caused the delay, in their words" })}
      {row.kind !== "letter" && day("delay_from", "Delay from")}
      {row.kind === "eot_claim" && day("delay_to", "Delay to")}
      {row.kind === "eot_claim" && <label className="pf-row"><span className="pf-label">Days claimed</span><span className="pf-value">
        <input type="number" min={0} max={3650} className="input register-days-input" key={k("days_claimed")} defaultValue={d.days_claimed ?? ""} disabled={data.busy} aria-label="Days claimed"
          onBlur={e => { const v = e.target.value === "" ? null : Number(e.target.value); if (v !== (d.days_claimed ?? null)) void save({ days_claimed: v }); }} /></span></label>}
      {row.kind === "eot_claim" && <label className="pf-row"><span className="pf-label">Notice of delay</span><span className="pf-value">
        <select className="input" value={d.notice_id ?? ""} disabled={data.busy} aria-label="Notice of delay"
          onChange={e => void save({ details: { notice_id: e.target.value } }, e.target.value ? "Linked the notice." : "Unlinked the notice.")}>
          <option value="">{notices.length ? "None linked" : "No notice of delay is recorded"}</option>
          {notices.map(n => <option key={n.id} value={n.id}>{numberLabel(n.kind, n.number)} · {n.title}</option>)}
        </select></span></label>}
      {d.message_id && <div className="pf-row"><span className="pf-label">Message</span><span className="pf-value">
        <button type="button" className="cell-link" onClick={() => onOpenMessage(d.message_id!)}>Open the message</button></span></div>}
    </div>
    {row.kind === "letter" && <label className="register-field register-question">Letter
      <textarea aria-label="Letter" key={k("body")} defaultValue={d.body ?? ""} maxLength={8000} rows={4} disabled={data.busy}
        onBlur={e => { if (e.target.value !== (d.body ?? "")) void save({ body: e.target.value }); }} /></label>}
    {row.kind === "letter" && <LetterAnswer row={row} data={data} />}
    {row.kind === "eot_claim" && <ResultForm row={row} data={data} />}
  </>;
}

// LetterAnswer records the answer; saving one makes the letter answered.
function LetterAnswer({ row, data }: { row: DeliveryRecord; data: RegisterData }) {
  const d = row.details as ClaimDetails;
  const [text, setText] = useState(d.answer ?? "");
  const answered = row.status === "answered" || row.status === "closed";
  return <form className="register-answer-form" onSubmit={e => {
    e.preventDefault();
    const fields: Record<string, unknown> = { answer: text };
    if (text.trim() && row.status === "open") fields.status = "answered";
    void data.run([{ op: "delivery.edit", fields: { item: row.id, ...fields } }], "Saved the answer.", "sheet");
  }}>
    <label className="register-field">Our answer<textarea aria-label="Our answer" value={text} maxLength={8000} rows={3} onChange={e => setText(e.target.value)} /></label>
    <p><button className="btn btn-small" disabled={data.busy || text === (d.answer ?? "")}>{answered ? "Save" : "Save as answered"}</button></p>
  </form>;
}

// ResultForm records an EOT claim's result. A formal result asks for the
// outcome before it saves, so nobody decides a claim by accident; fewer
// days than claimed, or a rejection, needs reasons (Go checks both).
function ResultForm({ row, data }: { row: DeliveryRecord; data: RegisterData }) {
  const d = row.details as ClaimDetails;
  const r = d.result;
  const [outcome, setOutcome] = useState<EOTResult["outcome"] | "">(r?.outcome ?? "");
  const [days, setDays] = useState(r?.days_granted != null ? String(r.days_granted) : "");
  const [reasons, setReasons] = useState(r?.reasons ?? "");
  const [editing, setEditing] = useState(!r);
  const needsDays = outcome === "granted" || outcome === "granted_in_part";
  const needsReasons = outcome === "rejected" || outcome === "granted_in_part";
  const n = days.trim() === "" ? null : Number(days);
  const why = !outcome ? "Choose the result."
    : needsDays && (n === null || !Number.isInteger(n) || n < 1) ? "Give the days granted."
      : outcome === "granted" && d.days_claimed != null && n !== null && n < d.days_claimed ? "Fewer days than claimed is granted in part."
        : outcome === "granted_in_part" && d.days_claimed != null && n !== null && n >= d.days_claimed ? "Granted in part is fewer days than claimed."
          : needsReasons && !reasons.trim() ? "Give the reasons." : "";
  const decided = outcome !== "" && outcome !== "more_information";

  if (r && !editing) {
    return <section className="report-section register-result">
      <div className="report-section-head section-band"><h3>Result</h3></div>
      <p><span className="register-chip register-result-chip" data-outcome={r.outcome}>{resultWords(r, d.days_claimed)}</span>
        {r.assessed_on ? <span className="register-muted"> · {longDate(r.assessed_on)}</span> : null}</p>
      {r.reasons && <blockquote className="register-quote">{r.reasons}</blockquote>}
      <p><button type="button" className="cell-link" disabled={data.busy} onClick={() => setEditing(true)}>Change the result</button></p>
    </section>;
  }
  return <form className="report-section register-result" onSubmit={e => {
    e.preventDefault();
    if (why || !outcome) return;
    const result: Record<string, unknown> = { outcome, reasons: reasons.trim() };
    if (needsDays) result.days_granted = n;
    else if (outcome === "rejected") result.days_granted = 0;
    const fields: Record<string, unknown> = { outcome, reasons: result.reasons };
    if ("days_granted" in result) fields.days_granted = result.days_granted;
    if (decided && row.status === "open") fields.status = "assessed";
    if (!decided && row.status === "assessed") fields.status = "open";
    setEditing(false);
    void data.run([{ op: "delivery.edit", fields: { item: row.id, ...fields } }], `Recorded the result: ${resultWords(result as unknown as EOTResult, d.days_claimed).toLowerCase()}.`, "sheet");
  }}>
    <div className="report-section-head section-band"><h3>Result</h3></div>
    <fieldset className="register-outcomes"><legend className="sr-only">Outcome</legend>
      {OUTCOMES.map(([id, label]) => <label key={id} className="register-toggle"><input type="radio" name={`outcome-${row.id}`} value={id} checked={outcome === id}
        onChange={() => { setOutcome(id); if (!reasons.trim() || Object.values(WORDING).includes(reasons)) setReasons(WORDING[id]); }} /> {label}</label>)}
    </fieldset>
    {needsDays && <label className="register-field">Days granted{d.days_claimed != null ? ` (of ${d.days_claimed} claimed)` : ""}
      <input type="number" min={1} max={3650} className="input register-days-input" value={days} onChange={e => setDays(e.target.value)} aria-label="Days granted" /></label>}
    {outcome && <label className="register-field">{needsReasons ? "Reasons" : "Words"}
      <textarea aria-label={needsReasons ? "Reasons" : "Words"} value={reasons} maxLength={8000} rows={3} onChange={e => setReasons(e.target.value)} /></label>}
    <p><button className="btn btn-primary btn-small" disabled={data.busy || !!why} title={why || undefined}>Record result</button>
      {r && <> <button type="button" className="cell-link" onClick={() => setEditing(false)}>Cancel</button></>}
      {why && outcome ? <span className="register-muted"> {why}</span> : null}</p>
  </form>;
}
