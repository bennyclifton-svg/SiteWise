import { useEffect, useMemo, useRef, useState, type DragEvent } from "react";
import { ApiError } from "./api";
import { fmt } from "./CostGrid";
import { IconCheck, IconConfirmed, IconNotChecked, IconNotSet, IconUpload, IconYou } from "./icons";
import { invoiceApi, uploadInvoice, type Invoice, type InvoiceField, type InvoiceFilter, type Ledger } from "./invoiceApi";

export const STATUS_WORD: Record<string, string> = { reading: "Reading…", filed: "Filed", needs_review: "Needs review", not_read: "Not read", void: "Void", superseded: "Superseded" };
const KINDS: [string, string][] = [["invoice", "Tax invoice"], ["credit_note", "Credit note"], ["quote", "Quote"], ["statement", "Statement"], ["remittance", "Remittance"], ["receipt", "Receipt"], ["payment_schedule", "Payment schedule"], ["other", "Other"]];
const NOT_READ: Record<string, string> = {
  no_text_layer: "This PDF has no selectable text, so it was stored but not read. Type the fields from the PDF.",
  unreadable_text: "This PDF's text can't be read (its font has no usable character map). Type the fields from the PDF.",
  unsupported_format: "Only PDF, DOCX and XLSX invoices are read. The file is kept as uploaded; type the fields.",
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

interface Upload { key: string; name: string; progress: number; state: "uploading" | "reading" | "already" | "failed"; id?: string; error?: string }
const PARALLEL = 6;

export function InvoicesTab({ projectId, ledger, filter, onFilter, selected, onSelect, onLedger, onSignedOut, reload, incoming, onIncoming }: {
  projectId: string; ledger: Ledger | null; filter: InvoiceFilter; onFilter: (f: InvoiceFilter) => void;
  selected: string; onSelect: (id: string) => void; onLedger: (l: Ledger) => void; onSignedOut: () => void; reload: () => void;
  incoming?: File[] | null; onIncoming?: () => void;
}) {
  const [uploads, setUploads] = useState<Upload[]>([]);
  const [over, setOver] = useState(false);
  const [message, setMessage] = useState("");
  const queue = useRef<{ key: string; file: File }[]>([]);
  const active = useRef(0);
  const input = useRef<HTMLInputElement>(null);
  const depth = useRef(0);

  function pump() {
    while (active.current < PARALLEL && queue.current.length) {
      const next = queue.current.shift()!;
      active.current++;
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
    setUploads(u => [...items.map(i => ({ key: i.key, name: i.file.name, progress: 0, state: "uploading" as const })), ...u]);
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
    setUploads(u => u.filter(x => x.state === "uploading" || x.state === "failed" || x.state === "already" || (x.state === "reading" && (!x.id || status.get(x.id) === "reading" || !status.has(x.id)))));
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
      (!filter.line || (filter.line === "unallocated" ? !row.line_id : row.line_id === filter.line)) && (!filter.status || inv.status === filter.status)), [invoices, filter]);
  const months = useMemo(() => [...new Set(invoices.map(i => i.period.slice(0, 7)))].sort().reverse(), [invoices]);
  const filtered = !!(filter.supplier || filter.month || filter.line || filter.status);
  const reading = invoices.filter(i => i.status === "reading").length;

  return <section className="invoices" aria-label="Invoices" {...drag} data-over={over || undefined}>
    <div className="inv-drop" data-over={over || undefined}>
      <span className="drop-mark" aria-hidden="true"><IconUpload /></span>
      <div><p className="drop-title">Drop invoices here</p><p className="inv-drop-hint">PDF tax invoices from any consultant or contractor. Drop several at once; each is read on its own and lands in the ledger.</p></div>
      <button type="button" className="btn" onClick={() => input.current?.click()}>Choose invoices</button>
      <input ref={input} type="file" multiple accept=".pdf,.docx,.xlsx,application/pdf" hidden data-testid="invoice-input" onChange={e => { if (e.target.files) add(e.target.files); e.target.value = ""; }} />
    </div>
    <p className="sr-only" role="status" aria-live="polite">{message}{reading ? ` ${reading} still reading.` : ""}</p>
    {uploads.length > 0 && <ul className="inv-uploads" aria-label="Uploads">{uploads.map(u => <li key={u.key} data-state={u.state}>
      <span className="inv-upload-name">{u.name}</span>
      <span>{u.state === "uploading" ? `Uploading ${Math.round(u.progress * 100)}%` : u.state === "reading" ? "Reading…" : u.state === "already" ? "Already filed" : u.error}</span>
      {(u.state === "already" || u.state === "failed") && <button type="button" className="btn btn-small" onClick={() => setUploads(x => x.filter(y => y.key !== u.key))}>Dismiss</button>}
    </li>)}</ul>}
    <div className="inv-filters" role="group" aria-label="Filter invoices">
      <label>Supplier<select aria-label="Supplier filter" value={filter.supplier} onChange={e => onFilter({ ...filter, supplier: e.target.value })}><option value="">All suppliers</option>{ledger?.packages.map(p => <option key={p.id} value={p.id}>{p.title}</option>)}</select></label>
      <label>Month<select aria-label="Month filter" value={filter.month} onChange={e => onFilter({ ...filter, month: e.target.value })}><option value="">All months</option>{months.map(m => <option key={m} value={m}>{month(m)}</option>)}</select></label>
      <label>Activity line<select aria-label="Activity line filter" value={filter.line} onChange={e => onFilter({ ...filter, line: e.target.value })}><option value="">All lines</option><option value="unallocated">Unallocated</option>{ledger?.lines.map(l => <option key={l.id} value={l.id}>{l.label}{l.archived ? " (removed)" : ""}</option>)}</select></label>
      <label>Status<select aria-label="Status filter" value={filter.status} onChange={e => onFilter({ ...filter, status: e.target.value })}><option value="">All statuses</option>{Object.entries(STATUS_WORD).map(([k, v]) => <option key={k} value={k}>{v}</option>)}</select></label>
      {filtered && <button type="button" className="btn btn-small" onClick={() => onFilter({ supplier: "", month: "", line: "", status: "" })}>Clear filters</button>}
    </div>
    {!ledger ? <p role="status">Loading invoices…</p> : invoices.length === 0 ? <div className="inv-empty"><p><strong>No invoices yet.</strong> Drop the month's invoices above. Each one is read, mapped to a cost plan line, and counted in the Cost plan tab's invoiced to date.</p></div> :
      <div className="inv-table-scroll"><table className="inv-table">
        <caption className="sr-only">Invoice ledger, newest first, one row per activity line</caption>
        <thead><tr><th scope="col">Invoice no.</th><th scope="col">Supplier</th><th scope="col">Invoice date</th><th scope="col">Month</th><th scope="col">Activity line</th><th scope="col" className="num">Ex GST (AUD)</th><th scope="col" className="num">GST (AUD)</th><th scope="col" className="num">Total inc GST (AUD)</th><th scope="col">Status</th></tr></thead>
        <tbody>{rows.length === 0 ? <tr><td colSpan={9}>No invoices match these filters.</td></tr> : rows.map(({ inv, row }) => {
          const f = inv.fields;
          const lineField: InvoiceField | undefined = row.shares > 1 ? { value: row.line_id, band: row.band, decided_by: row.decided_by as InvoiceField["decided_by"] } : f.line ?? { value: "", band: "blank", decided_by: "rule" };
          return <tr key={`${inv.id}:${row.share}`} aria-selected={selected === inv.id} data-status={inv.status} onClick={() => onSelect(inv.id)}>
            <td><button type="button" className="inv-open" onClick={e => { e.stopPropagation(); onSelect(inv.id); }}>{inv.number || inv.filename}</button>{row.shares > 1 && <span className="inv-share">share {row.share + 1} of {row.shares}</span>}<Chip field={f.number} /></td>
            <td>{inv.supplier || "—"}<Chip field={f.supplier} /></td>
            <td>{day(inv.date)}<Chip field={f.date} /></td>
            <td>{month(inv.period)}<Chip field={f.period} /></td>
            <td>{row.line_id ? row.line : "Unallocated"}<Chip field={lineField} /></td>
            <td className="num">{fmt(row.ex_gst)}{row.share === 0 && <Chip field={f.ex_gst} />}</td>
            <td className="num">{fmt(row.gst)}{row.share === 0 && <Chip field={f.gst} />}</td>
            <td className="num">{fmt(row.total)}{row.share === 0 && <Chip field={f.total} />}</td>
            <td><span className="inv-status" data-status={inv.status}>{STATUS_WORD[inv.status] ?? inv.status}</span>{!inv.counted && inv.status !== "reading" && <span className="inv-out">not in totals</span>}{inv.flags.filter(x => !x.info).slice(0, 2).map(x => <span key={x.code} className="inv-flag">{x.reason}</span>)}</td>
          </tr>;
        })}</tbody>
      </table></div>}
    {selected && ledger && <InvoicePanel projectId={projectId} ledger={ledger} invoice={ledger.invoices.find(i => i.id === selected)} onClose={() => onSelect("")} onLedger={onLedger} onSignedOut={onSignedOut} />}
  </section>;
}

function InvoicePanel({ projectId, ledger, invoice, onClose, onLedger, onSignedOut }: { projectId: string; ledger: Ledger; invoice: Invoice | undefined; onClose: () => void; onLedger: (l: Ledger) => void; onSignedOut: () => void }) {
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
    <div className="inv-panel-head"><h2>{invoice.number || invoice.filename}</h2><span className="inv-status" data-status={invoice.status}>{STATUS_WORD[invoice.status]}</span><button type="button" className="btn btn-small" onClick={onClose}>Close</button></div>
    <div className="inv-panel-body">
      <div className="inv-pdf">{/\.pdf$/i.test(invoice.filename) ? <iframe title={`PDF of ${invoice.filename}`} src={invoiceApi.fileURL(projectId, invoice.id)} /> : <p><a href={invoiceApi.fileURL(projectId, invoice.id).replace("?view=1", "")}>Download {invoice.filename}</a></p>}</div>
      <form className="inv-fields" aria-busy={busy} onSubmit={e => e.preventDefault()}>
        {invoice.status === "not_read" && <p className="inv-note">{NOT_READ[invoice.not_read_reason ?? ""] ?? "Stored, not read. Type the fields from the PDF."}</p>}
        {error && <p role="alert" className="inv-error">{error}</p>}
        {invoice.flags.length > 0 && <ul className="inv-flags" aria-label="Checks">{invoice.flags.map(x => <li key={x.code} data-info={x.info || undefined}>{x.reason}</li>)}</ul>}
        <div className="inv-field"><label htmlFor="inv-kind">Document</label><select id="inv-kind" value={f.kind?.value ?? ""} disabled={busy} onChange={e => save("kind", e.target.value)}><option value="" disabled>Not set</option>{KINDS.map(([k, v]) => <option key={k} value={k}>{v}</option>)}</select><Chip field={f.kind} /></div>
        <div className="inv-field"><label htmlFor="inv-number">Invoice no.</label><input id="inv-number" key={`${invoice.id}:n:${f.number?.value}`} defaultValue={f.number?.value ?? ""} disabled={busy} maxLength={100} onBlur={e => save("number", e.target.value.trim())} onKeyDown={e => { if (e.key === "Enter") (e.target as HTMLInputElement).blur(); }} /><Chip field={f.number} /></div>
        <div className="inv-field"><label htmlFor="inv-supplier">Supplier</label><select id="inv-supplier" value={f.supplier?.value ?? ""} disabled={busy} onChange={e => save("supplier", e.target.value)}><option value="">No matching package</option>{ledger.packages.map(p => <option key={p.id} value={p.id}>{p.title}</option>)}</select><Chip field={f.supplier} /></div>
        <div className="inv-field"><label htmlFor="inv-abn">Supplier ABN</label><input id="inv-abn" key={`${invoice.id}:a:${f.issuer_abn?.value}`} defaultValue={f.issuer_abn?.value ?? ""} disabled={busy} inputMode="numeric" onBlur={e => save("issuer_abn", e.target.value.replace(/\s/g, ""))} onKeyDown={e => { if (e.key === "Enter") (e.target as HTMLInputElement).blur(); }} /><Chip field={f.issuer_abn} /></div>
        <div className="inv-field"><label htmlFor="inv-date">Invoice date</label><input id="inv-date" type="date" key={`${invoice.id}:d:${f.date?.value}`} defaultValue={f.date?.value ?? ""} disabled={busy} onBlur={e => save("date", e.target.value)} /><Chip field={f.date} /></div>
        <div className="inv-field"><label htmlFor="inv-period">Month certified</label><input id="inv-period" type="month" key={`${invoice.id}:p:${f.period?.value}`} defaultValue={(f.period?.value ?? invoice.period).slice(0, 7)} disabled={busy} onChange={e => { if (e.target.value) save("period", `${e.target.value}-01`); }} /><Chip field={f.period} /></div>
        {!splitNow && <div className="inv-field"><label htmlFor="inv-line">Activity line</label><select id="inv-line" value={invoice.allocations?.[0]?.line_id ?? ""} disabled={busy} onChange={e => save("line", e.target.value)}><option value="">Unallocated</option>{ledger.lines.filter(l => !l.archived || l.id === invoice.allocations?.[0]?.line_id).map(l => <option key={l.id} value={l.id}>{l.label}{l.category ? ` · ${l.category}` : ""}</option>)}</select><Chip field={f.line ?? { value: "", band: "blank", decided_by: "rule" }} /></div>}
        {money("ex_gst", "Ex GST (AUD)")}{money("gst", "GST (AUD)")}{money("total", "Total inc GST (AUD)")}
        <div className="inv-field"><label htmlFor="inv-currency">Currency</label><input id="inv-currency" key={`${invoice.id}:c:${f.currency?.value}`} defaultValue={f.currency?.value ?? "AUD"} maxLength={3} disabled={busy} onBlur={e => save("currency", e.target.value.trim().toUpperCase())} /><Chip field={f.currency} /></div>
        <fieldset className="inv-split"><legend>Activity lines</legend>
          {splitNow && !split && <ul>{invoice.allocations?.map(a => <li key={a.id}>{a.line_id ? a.line : "Unallocated"} · {fmt(a.amount)}</li>)}</ul>}
          {split ? <>{split.map((s, i) => <div key={i} className="inv-split-row"><select aria-label={`Share ${i + 1} line`} value={s.line_id} onChange={e => setSplit(split.map((x, j) => j === i ? { ...x, line_id: e.target.value } : x))}><option value="">Unallocated</option>{ledger.lines.filter(l => !l.archived).map(l => <option key={l.id} value={l.id}>{l.label}</option>)}</select><input aria-label={`Share ${i + 1} amount ex GST`} inputMode="decimal" value={s.amount} onChange={e => setSplit(split.map((x, j) => j === i ? { ...x, amount: e.target.value } : x))} />{split.length > 1 && <button type="button" className="btn btn-small" onClick={() => setSplit(split.filter((_, j) => j !== i))}>Remove</button>}</div>)}
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
