import { Pending } from "./Activity";
import { useEffect, useMemo, useRef, useState, type DragEvent } from "react";
import { ApiError } from "./api";
import { fmt } from "./CostGrid";
import { IconCheck, IconConfirmed, IconNotChecked, IconNotSet, IconUpload, IconYou } from "./icons";
import { invoiceApi, uploadInvoice, type Invoice, type InvoiceField, type InvoiceFilter, type Ledger } from "./invoiceApi";

export const STATUS_WORD: Record<string, string> = { reading: "Reading…", filed: "Filed", needs_review: "Needs review", not_read: "Not read", void: "Void", superseded: "Superseded" };
const activityLabel = (line: { label: string; category: string }) => line.category && line.category !== line.label ? `${line.category} → ${line.label}` : line.label;
const KINDS: [string, string][] = [["invoice", "Tax invoice"], ["credit_note", "Credit note"], ["quote", "Quote"], ["statement", "Statement"], ["remittance", "Remittance"], ["receipt", "Receipt"], ["payment_schedule", "Payment schedule"], ["other", "Other"]];
const NOT_READ: Record<string, string> = {
  no_text_layer: "This PDF has no selectable text, so it was stored but not read. Type the fields from the PDF.",
  unreadable_text: "This file's text can't be read. It is kept as uploaded; type the fields from the original.",
  unsupported_format: "Only PDF, DOCX, XLSX and Markdown invoices are read. The file is kept as uploaded; type the fields.",
  too_large: "This file is larger than invoice reading allows. It is kept as uploaded; type the fields.",
};
const month = (iso: string) => {
  if (!iso) return "—";
  const [y, m] = iso.split("-");
  return `${["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"][Number(m) - 1]} ${y}`;
};
const day = (iso: string) => {
  if (!iso) return "—";
  const [y, m, d] = iso.split("-");
  return `${Number(d)} ${month(`${y}-${m}`).split(" ")[0]} ${y}`;
};

type ChipState = "confirmed" | "check" | "unchecked" | "user" | "unset";
function chipState(f: InvoiceField | undefined): ChipState {
  if (!f) return "unset";
  if (f.decided_by === "user") return "user";
  if (f.band === "grey") return "unchecked";
  if (!f.value) return "unset";
  if (f.band === "green") return "confirmed";
  if (f.band === "amber") return "check";
  return "unset";
}

/** Chip says how a field was decided, in words and an icon, never colour alone. */
export function Chip({ field }: { field: InvoiceField | undefined }) {
  const state = chipState(field);
  const who = field?.decided_by === "jev" ? "Jev" : "Rule";
  const body = {
    user: <><IconYou />{field?.value ? "Set by you" : "Cleared by you"}</>,
    unchecked: <><IconNotChecked />Not checked</>,
    check: <><IconCheck />Check · {who}</>,
    confirmed: <><IconConfirmed />{field?.decided_by === "jev" ? "Confirmed · Jev" : "From document"}</>,
    unset: <><IconNotSet />Not set</>,
  }[state];
  return <span className="inv-chip" data-state={state} title={state === "unchecked" ? "Jev didn't answer in time. Rule values stand; this is re-checked in the background." : undefined}>{body}</span>;
}

// Header clicks rank the ledger (owner direction, 7 Nov 2026): one key at a
// time, ascending then descending then the ledger's own newest-first order.
type SortKey = "date" | "number" | "discipline" | "line" | "month" | "ex_gst" | "status";
const COLUMNS: [SortKey, string][] = [["date", "Invoice date"], ["number", "Invoice no."], ["discipline", "Discipline"], ["line", "Activity line"], ["month", "Month"], ["ex_gst", "Ex GST (AUD)"], ["status", "Status"]];
// Money stays a decimal string; compare by sign, then digits, never as a float.
function compareMoney(a: string | null | undefined, b: string | null | undefined) {
  if (a == null || b == null) return a == null ? (b == null ? 0 : 1) : -1;
  const neg = (m: string) => m.startsWith("-"), abs = (m: string) => m.replace("-", "");
  if (neg(a) !== neg(b)) return neg(a) ? -1 : 1;
  const [aw, af = ""] = abs(a).split("."), [bw, bf = ""] = abs(b).split(".");
  const c = aw.length - bw.length || aw.localeCompare(bw) || af.padEnd(2, "0").localeCompare(bf.padEnd(2, "0"));
  return neg(a) ? -c : c;
}

interface Upload { key: string; name: string; progress: number; state: "queued" | "uploading" | "reading" | "already" | "failed"; id?: string; error?: string }
const PARALLEL = 6;

export function InvoicesTab({ onCostLine, costLineIds, projectId, ledger, filter, onFilter, selected, onSelect, onLedger, onSignedOut, reload, incoming, onIncoming }: {
  projectId: string; ledger: Ledger | null; filter: InvoiceFilter; onFilter: (f: InvoiceFilter) => void;
  selected: string; onSelect: (id: string) => void; onLedger: (l: Ledger) => void; onSignedOut: () => void; reload: () => void;
  onCostLine?: (id: string) => void; costLineIds?: string[];
  incoming?: File[] | null; onIncoming?: () => void;
}) {
  const [uploads, setUploads] = useState<Upload[]>([]);
  const [sort, setSort] = useState<{ key: SortKey; dir: 1 | -1 } | null>(null);
  const [over, setOver] = useState(false);
  const [message, setMessage] = useState("");
  const [savingDiscipline, setSavingDiscipline] = useState("");
  const [disciplineError, setDisciplineError] = useState("");
  const disciplineSaving = useRef(false);

  async function saveDiscipline(invoice: Invoice, supplier: string) {
    if (disciplineSaving.current || supplier === invoice.supplier_id) return;
    disciplineSaving.current = true;
    setSavingDiscipline(invoice.id);
    setDisciplineError("");
    try {
      onLedger(await invoiceApi.edit(projectId, invoice.id, { supplier }));
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
      setDisciplineError(`Could not save the discipline for ${invoice.number || invoice.filename}. Choose it again to retry.`);
    } finally {
      disciplineSaving.current = false;
      setSavingDiscipline("");
    }
  }
  const queue = useRef<{ key: string; file: File }[]>([]);
  const active = useRef(0);
  const input = useRef<HTMLInputElement>(null);
  const depth = useRef(0);

  function pump() {
    while (active.current < PARALLEL && queue.current.length) {
      const next = queue.current.shift()!;
      active.current++;
      setUploads(u => u.map(x => x.key === next.key ? { ...x, state: "uploading" } : x));
      const { done } = uploadInvoice(projectId, next.file, p => setUploads(u => u.map(x => x.key === next.key ? { ...x, progress: p } : x)));
      done.then(r => {
        setUploads(u => u.map(x => x.key === next.key ? { ...x, progress: 1, id: r.id, state: r.created ? "reading" : "already" } : x));
        reload();
      }).catch(e => {
        if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; }
        setUploads(u => u.map(x => x.key === next.key ? { ...x, state: "failed", error: e instanceof Error ? e.message : "Upload failed" } : x));
      }).finally(() => { active.current--; pump(); });
    }
  }
  function add(files: FileList | File[]) {
    const list = Array.from(files);
    if (!list.length) return;
    const items = list.map(f => ({ key: `${f.name}:${f.size}:${Math.random().toString(36).slice(2)}`, file: f }));
    setUploads(u => [...items.map(i => ({ key: i.key, name: i.file.name, progress: 0, state: "queued" as const })), ...u]);
    queue.current.push(...items);
    setMessage(`${list.length} ${list.length === 1 ? "invoice" : "invoices"} added. Each row appears as it is read.`);
    pump();
  }
  // Files dropped on another cost plan tab arrive here.
  useEffect(() => {
    if (incoming?.length) { add(incoming); onIncoming?.(); }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [incoming]);
  // Rows that finished reading leave the upload list; the ledger shows them.
  useEffect(() => {
    if (!ledger) return;
    const status = new Map(ledger.invoices.map(i => [i.id, i.status]));
    setUploads(u => u.filter(x => x.state === "queued" || x.state === "uploading" || x.state === "failed" || x.state === "already" || (x.state === "reading" && (!x.id || status.get(x.id) === "reading" || !status.has(x.id)))));
  }, [ledger]);

  // The panel is its own drop target: invoices never fall through to the
  // document register's page-wide drop.
  const drag = {
    onDragEnter: (e: DragEvent) => { if (!e.dataTransfer.types.includes("Files")) return; e.preventDefault(); e.stopPropagation(); depth.current++; setOver(true); },
    onDragOver: (e: DragEvent) => { if (!e.dataTransfer.types.includes("Files")) return; e.preventDefault(); e.stopPropagation(); e.dataTransfer.dropEffect = "copy"; },
    onDragLeave: (e: DragEvent) => { e.stopPropagation(); depth.current = Math.max(0, depth.current - 1); if (!depth.current) setOver(false); },
    onDrop: (e: DragEvent) => { if (!e.dataTransfer.types.includes("Files")) return; e.preventDefault(); e.stopPropagation(); depth.current = 0; setOver(false); add(e.dataTransfer.files); },
  };

  const invoices = ledger?.invoices ?? [];
  const rows = useMemo(() => invoices.flatMap(inv => (inv.rows?.length ? inv.rows : [{ share: 0, shares: 1, line_id: "", line: "Unallocated", ex_gst: inv.ex_gst, gst: inv.gst, total: inv.total, band: "blank" as const, decided_by: "rule" }]).map(r => ({ inv, row: r })))
    .filter(({ inv, row }) => (!filter.supplier || inv.supplier_id === filter.supplier) && (!filter.month || inv.period.startsWith(filter.month)) &&
      (!filter.line || (filter.line === "unallocated" ? !row.line_id : (row.line_id === filter.line || !!ledger?.groups?.[filter.line]?.line_ids.includes(row.line_id)))) && (!filter.status || inv.status === filter.status)), [invoices, filter, ledger?.groups]);
  // The table names the discipline in its own column, so the line shows its stage alone.
  const lineLabel = (id: string, fallback: string) => ledger?.lines.find(l => l.id === id)?.label ?? fallback;
  const discipline = (inv: Invoice) => ledger?.packages.find(p => p.id === inv.supplier_id)?.title ?? inv.supplier ?? "";
  const sorted = useMemo(() => {
    if (!sort) return rows;
    const text = (a: string, b: string) => a.localeCompare(b, undefined, { numeric: true, sensitivity: "base" });
    const by: Record<SortKey, (a: typeof rows[number], b: typeof rows[number]) => number> = {
      date: (a, b) => text(a.inv.date, b.inv.date),
      number: (a, b) => text(a.inv.number || a.inv.filename, b.inv.number || b.inv.filename),
      discipline: (a, b) => text(discipline(a.inv), discipline(b.inv)),
      line: (a, b) => text(a.row.line_id ? lineLabel(a.row.line_id, a.row.line) : "", b.row.line_id ? lineLabel(b.row.line_id, b.row.line) : ""),
      month: (a, b) => text(a.inv.period, b.inv.period),
      ex_gst: (a, b) => compareMoney(a.row.ex_gst, b.row.ex_gst),
      status: (a, b) => text(STATUS_WORD[a.inv.status] ?? a.inv.status, STATUS_WORD[b.inv.status] ?? b.inv.status),
    };
    return [...rows].sort((a, b) => sort.dir * by[sort.key](a, b));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, sort, ledger]);
  const filtered = !!(filter.supplier || filter.month || filter.line || filter.status);
  const filterName = filter.line === "unallocated" ? "no activity line" : filter.line ? (ledger?.groups?.[filter.line]?.label ?? ledger?.lines.find(l => l.id === filter.line)?.label ?? "one line") : "a filter";
  const reading = invoices.filter(i => i.status === "reading").length;

  return <section className="invoices" aria-label="Invoices" {...drag} data-over={over || undefined}>
    {/* The whole tab is the drop zone; the button opens the file picker. */}
    <div className="costs-toolbar inv-toolbar">
      {filtered && <p className="inv-filtered">Showing invoices on {filterName} · <button type="button" className="inv-open" onClick={() => onFilter({ supplier: "", month: "", line: "", status: "" })}>Show all</button></p>}
      <button type="button" className="btn inv-upload" aria-label="Upload invoices" title="Upload invoices, or drop them on the table" onClick={() => input.current?.click()}><IconUpload /></button>
      <input ref={input} type="file" multiple accept=".pdf,.docx,.xlsx,.md,.markdown,application/pdf" hidden data-testid="invoice-input" onChange={e => { if (e.target.files) add(e.target.files); e.target.value = ""; }} />
    </div>
    <p className="sr-only" role="status" aria-live="polite">{message}{reading ? ` ${reading} still reading.` : ""}</p>
    {uploads.length > 0 && <ul className="inv-uploads" aria-label="Uploads">{uploads.map(u => <li key={u.key} data-state={u.state}>
      <span className="inv-upload-name">{u.name}</span>
      <span>{u.state === "queued" ? <Pending queued>Queued for upload</Pending> : u.state === "uploading" ? <Pending>{u.progress < 1 ? `Uploading ${Math.round(u.progress * 100)}%` : "Finishing upload…"}</Pending> : u.state === "reading" ? <Pending>Reading invoice…</Pending> : u.state === "already" ? "Already filed" : u.error}</span>
      {(u.state === "already" || u.state === "failed") && <button type="button" className="btn btn-small" onClick={() => setUploads(x => x.filter(y => y.key !== u.key))}>Dismiss</button>}
    </li>)}</ul>}
    {disciplineError && <p role="alert" className="inv-error">{disciplineError}</p>}
    <span className="sr-only" role="status">{savingDiscipline && "Saving discipline…"}</span>
    {!ledger ? <p role="status"><Pending delayed>Loading invoices…</Pending></p> : invoices.length === 0 ? <div className="inv-empty"><p><strong>No invoices yet.</strong> Drop the month's invoices here, or use the upload button. Each one is read, mapped to a cost plan line, and counted in the Cost plan tab's invoiced to date.</p></div> :
      <div className="inv-table-scroll"><table className="inv-table">
        <caption className="sr-only">Invoice ledger, newest first, one row per activity line</caption>
        <thead><tr>{COLUMNS.map(([key, label]) => <th key={key} scope="col" className={key === "ex_gst" ? "num" : undefined} aria-sort={sort?.key === key ? (sort.dir === 1 ? "ascending" : "descending") : undefined}>
          <button type="button" onClick={() => setSort(sort?.key === key ? (sort.dir === 1 ? { key, dir: -1 } : null) : { key, dir: 1 })}>{label}</button></th>)}</tr></thead>
        <tbody>{sorted.length === 0 ? <tr><td colSpan={7}>No invoices on {filterName}.</td></tr> : sorted.map(({ inv, row }) => {
          return <tr key={`${inv.id}:${row.share}`} aria-selected={selected === inv.id} data-status={inv.status}>
            <td>{day(inv.date)}</td>
            <td><button type="button" className="inv-open" onClick={() => onSelect(inv.id)}>{inv.number || inv.filename}</button>{row.shares > 1 && <span className="inv-share"> · {row.share + 1}/{row.shares}</span>}</td>
            <td><select className="field-plain inv-discipline" aria-label={`Discipline for ${inv.number || inv.filename}${row.shares > 1 ? ` share ${row.share + 1}` : ""}`} value={inv.supplier_id} disabled={!!savingDiscipline} onChange={e => void saveDiscipline(inv, e.target.value)}>
              <option value="">Not set</option>
              {inv.supplier_id && !ledger.packages.some(p => p.id === inv.supplier_id) && <option value={inv.supplier_id}>{inv.supplier || "Unavailable discipline"}</option>}
              {ledger.packages.map(p => <option key={p.id} value={p.id}>{p.title}</option>)}
            </select></td>
            <td>{row.line_id && onCostLine && costLineIds?.includes(row.line_id) ? <button type="button" className="inv-open" title="Show activity in cost plan" onClick={() => onCostLine(row.line_id)}>{lineLabel(row.line_id, row.line)}</button> : row.line_id ? lineLabel(row.line_id, row.line) : "Unallocated"}</td>
            <td>{month(inv.period)}</td>
            <td className="num">{fmt(row.ex_gst)}</td>
            <td><span className="inv-status" data-status={inv.status} title={inv.flags.map(x => x.reason).join("; ")}>{inv.status === "reading" ? <Pending>Reading invoice…</Pending> : STATUS_WORD[inv.status] ?? inv.status}</span>{!inv.counted && inv.status !== "reading" && <span className="inv-out"> · not in totals</span>}</td>
          </tr>;
        })}</tbody>
      </table></div>}
    {selected && ledger && <InvoicePanel onCostLine={onCostLine} costLineIds={costLineIds} projectId={projectId} ledger={ledger} invoice={ledger.invoices.find(i => i.id === selected)} onClose={() => onSelect("")} onLedger={onLedger} onSignedOut={onSignedOut} />}
  </section>;
}

function InvoicePanel({ onCostLine, costLineIds, projectId, ledger, invoice, onClose, onLedger, onSignedOut }: { onCostLine?: (id: string) => void; costLineIds?: string[]; projectId: string; ledger: Ledger; invoice: Invoice | undefined; onClose: () => void; onLedger: (l: Ledger) => void; onSignedOut: () => void }) {
  const [busy, setBusy] = useState(false), [error, setError] = useState(""), [split, setSplit] = useState<{ line_id: string; amount: string }[] | null>(null);
  useEffect(() => { const k = (e: KeyboardEvent) => { if (e.key === "Escape" && !(e.target instanceof HTMLInputElement)) onClose(); }; window.addEventListener("keydown", k); return () => window.removeEventListener("keydown", k); }, [onClose]);
  useEffect(() => { setSplit(null); setError(""); }, [invoice?.id]);
  if (!invoice) return <aside className="inv-panel" aria-label="Invoice"><button type="button" className="btn btn-small" onClick={onClose}>Close</button><p>That invoice is no longer in the ledger.</p></aside>;
  async function run(action: () => Promise<Ledger>) {
    setBusy(true); setError("");
    try { onLedger(await action()); }
    catch (e) { if (e instanceof ApiError && e.status === 401) { onSignedOut(); return; } setError(e instanceof ApiError && e.status === 422 ? `Check the value. ${e.message}` : e instanceof ApiError && e.status === 409 ? e.message : "Could not save. Your entry is kept; try again."); }
    finally { setBusy(false); }
  }
  const save = (field: string, value: string) => { if ((invoice.fields[field]?.value ?? "") === value) return; void run(() => invoiceApi.edit(projectId, invoice.id, { [field]: value })); };
  const f = invoice.fields;
  const splitNow = (invoice.allocations?.length ?? 0) > 1 || invoice.allocations?.some(a => a.amount != null);
  const dup = invoice.flags.find(x => x.code === "duplicate" || x.code === "look_alike");
  const twin = dup ? ledger.invoices.find(o => o.id !== invoice.id && o.status !== "void" && o.status !== "superseded" && (o.supplier_id && o.supplier_id === invoice.supplier_id || o.abn && o.abn === invoice.abn) && (o.number === dup.reason.replace(/^(Possible duplicate of|Looks like) /, "") || o.filename === dup.reason.replace(/^(Possible duplicate of|Looks like) /, ""))) : undefined;
  const money = (field: string, label: string) => <div className="inv-field"><label htmlFor={`inv-${field}`}>{label}</label><input id={`inv-${field}`} key={`${invoice.id}:${field}:${f[field]?.value}`} inputMode="decimal" defaultValue={f[field]?.value ? fmt(centsToDecimal(f[field].value)) : ""} disabled={busy} onBlur={e => save(field, e.target.value.replace(/[,$\s]/g, ""))} onKeyDown={e => { if (e.key === "Enter") (e.target as HTMLInputElement).blur(); }} /><Chip field={f[field]} /></div>;
  return <aside className="inv-panel" aria-label={`Invoice ${invoice.number || invoice.filename}`}>
    <span role="status">{busy && <Pending delayed>Saving…</Pending>}</span>
    <div className="inv-panel-head"><h2>{invoice.number || invoice.filename}</h2><span className="inv-status" data-status={invoice.status}>{invoice.status === "reading" ? <Pending>Reading invoice…</Pending> : STATUS_WORD[invoice.status]}</span><button type="button" className="btn btn-small" onClick={onClose}>Close</button></div>
    <div className="inv-panel-body">
      <div className="inv-pdf">{/\.pdf$/i.test(invoice.filename) ? <iframe title={`PDF of ${invoice.filename}`} src={invoiceApi.fileURL(projectId, invoice.id)} /> : <p><a href={invoiceApi.fileURL(projectId, invoice.id).replace("?view=1", "")}>Download {invoice.filename}</a></p>}</div>
      <form className="inv-fields" aria-busy={busy} onSubmit={e => e.preventDefault()}>
        {invoice.status === "not_read" && <p className="inv-note">{NOT_READ[invoice.not_read_reason ?? ""] ?? "Stored, not read. Type the fields from the original file."}</p>}
        {error && <p role="alert" className="inv-error">{error}</p>}
        {invoice.flags.length > 0 && <ul className="inv-flags" aria-label="Checks">{invoice.flags.map(x => <li key={x.code} data-info={x.info || undefined}>{x.reason}</li>)}</ul>}
        <div className="inv-field"><label htmlFor="inv-kind">Document</label><select id="inv-kind" value={f.kind?.value ?? ""} disabled={busy} onChange={e => save("kind", e.target.value)}><option value="" disabled>Not set</option>{KINDS.map(([k, v]) => <option key={k} value={k}>{v}</option>)}</select><Chip field={f.kind} /></div>
        <div className="inv-field"><label htmlFor="inv-number">Invoice no.</label><input id="inv-number" key={`${invoice.id}:n:${f.number?.value}`} defaultValue={f.number?.value ?? ""} disabled={busy} maxLength={100} onBlur={e => save("number", e.target.value.trim())} onKeyDown={e => { if (e.key === "Enter") (e.target as HTMLInputElement).blur(); }} /><Chip field={f.number} /></div>
        <div className="inv-field"><label htmlFor="inv-supplier">Discipline</label><select id="inv-supplier" value={f.supplier?.value ?? ""} disabled={busy} onChange={e => save("supplier", e.target.value)}><option value="">No matching package</option>{ledger.packages.map(p => <option key={p.id} value={p.id}>{p.title}</option>)}</select><Chip field={f.supplier} /></div>
        <div className="inv-field"><label htmlFor="inv-abn">Supplier ABN</label><input id="inv-abn" key={`${invoice.id}:a:${f.issuer_abn?.value}`} defaultValue={f.issuer_abn?.value ?? ""} disabled={busy} inputMode="numeric" onBlur={e => save("issuer_abn", e.target.value.replace(/\s/g, ""))} onKeyDown={e => { if (e.key === "Enter") (e.target as HTMLInputElement).blur(); }} /><Chip field={f.issuer_abn} /></div>
        <div className="inv-field"><label htmlFor="inv-date">Invoice date</label><input id="inv-date" type="date" key={`${invoice.id}:d:${f.date?.value}`} defaultValue={f.date?.value ?? ""} disabled={busy} onBlur={e => save("date", e.target.value)} /><Chip field={f.date} /></div>
        <div className="inv-field"><label htmlFor="inv-period">Month certified</label><input id="inv-period" type="month" key={`${invoice.id}:p:${f.period?.value}`} defaultValue={(f.period?.value ?? invoice.period).slice(0, 7)} disabled={busy} onChange={e => { if (e.target.value) save("period", `${e.target.value}-01`); }} /><Chip field={f.period} /></div>
        {!splitNow && <div className="inv-field"><label htmlFor="inv-line">Activity line</label><select id="inv-line" value={invoice.allocations?.[0]?.line_id ?? ""} disabled={busy} onChange={e => save("line", e.target.value)}><option value="">Unallocated</option>{ledger.lines.filter(l => !ledger.groups?.[l.id] && (!l.archived || l.id === invoice.allocations?.[0]?.line_id)).map(l => <option key={l.id} value={l.id}>{activityLabel(l)}</option>)}</select><Chip field={f.line ?? { value: "", band: "blank", decided_by: "rule" }} /></div>}
        {money("ex_gst", "Ex GST (AUD)")}{money("gst", "GST (AUD)")}{money("total", "Total inc GST (AUD)")}
        <div className="inv-field"><label htmlFor="inv-currency">Currency</label><input id="inv-currency" key={`${invoice.id}:c:${f.currency?.value}`} defaultValue={f.currency?.value ?? "AUD"} maxLength={3} disabled={busy} onBlur={e => save("currency", e.target.value.trim().toUpperCase())} /><Chip field={f.currency} /></div>
        <fieldset className="inv-split"><legend>Activity lines</legend>
          {onCostLine && invoice.allocations?.filter(a => costLineIds?.includes(a.line_id)).map(a => <button key={a.id} type="button" className="inv-open" onClick={() => onCostLine(a.line_id)}>Show {activityLabel(ledger.lines.find(l => l.id === a.line_id) ?? { label: a.line, category: "" })} in cost plan</button>)}
          {splitNow && !split && <ul>{invoice.allocations?.map(a => <li key={a.id}>{a.line_id ? a.line : "Unallocated"} · {fmt(a.amount)}</li>)}</ul>}
          {split ? <>{split.map((s, i) => <div key={i} className="inv-split-row"><select aria-label={`Share ${i + 1} line`} value={s.line_id} onChange={e => setSplit(split.map((x, j) => j === i ? { ...x, line_id: e.target.value } : x))}><option value="">Unallocated</option>{ledger.lines.filter(l => !l.archived && !ledger.groups?.[l.id]).map(l => <option key={l.id} value={l.id}>{activityLabel(l)}</option>)}</select><input aria-label={`Share ${i + 1} amount ex GST`} inputMode="decimal" value={s.amount} onChange={e => setSplit(split.map((x, j) => j === i ? { ...x, amount: e.target.value } : x))} />{split.length > 1 && <button type="button" className="btn btn-small" onClick={() => setSplit(split.filter((_, j) => j !== i))}>Remove</button>}</div>)}
            <button type="button" className="btn btn-small" onClick={() => setSplit([...split, { line_id: "", amount: "" }])}>Add a line</button>
            <button type="button" className="btn btn-small" disabled={busy} onClick={() => void run(() => invoiceApi.split(projectId, invoice.id, split.length === 1 && !split[0].amount ? split : split.map(s => ({ ...s, amount: s.amount.replace(/[,$\s]/g, "") })))).then(() => setSplit(null))}>Save split</button>
            <button type="button" className="btn btn-small" onClick={() => setSplit(null)}>Cancel</button></> :
            <button type="button" className="btn btn-small" disabled={busy || invoice.status === "void"} onClick={() => setSplit(invoice.allocations?.length ? invoice.allocations.map(a => ({ line_id: a.line_id, amount: a.amount ?? invoice.ex_gst ?? "" })) : [{ line_id: "", amount: invoice.ex_gst ?? "" }])}>{splitNow ? "Change split" : "Split across lines"}</button>}
        </fieldset>
        <div className="inv-actions">
          {dup && twin && <><button type="button" className="btn" disabled={busy} onClick={() => void run(() => invoiceApi.edit(projectId, invoice.id, {}, "replaces", twin.id))}>Replaces {twin.number || twin.filename}</button><button type="button" className="btn" disabled={busy} onClick={() => void run(() => invoiceApi.edit(projectId, invoice.id, {}, "keep_both"))}>Keep both</button></>}
          {invoice.kind === "not_invoice" && <button type="button" className="btn" disabled={busy} onClick={() => save("kind", "invoice")}>Count it as an invoice</button>}
          {invoice.status === "void" ? <button type="button" className="btn" disabled={busy} onClick={() => void run(() => invoiceApi.edit(projectId, invoice.id, {}, "restore"))}>Restore</button> :
            invoice.status !== "superseded" && <button type="button" className="btn" disabled={busy} onClick={() => void run(() => invoiceApi.edit(projectId, invoice.id, {}, "void"))}>Void</button>}
        </div>
        <p className="inv-file">Source file: {invoice.filename}</p>
      </form>
    </div>
  </aside>;
}

// Decisions store amounts in cents; the inputs show dollars.
function centsToDecimal(c: string): string {
  const neg = c.startsWith("-"), digits = (neg ? c.slice(1) : c).padStart(3, "0");
  return `${neg ? "-" : ""}${digits.slice(0, -2)}.${digits.slice(-2)}`;
}

export function VariationsTab() {
  return <section className="inv-empty" aria-label="Variations"><h2>Variations are coming</h2><p>Approved variations will adjust each line's budget here. This tab holds no data yet; invoices and the cost plan are unaffected.</p></section>;
}
