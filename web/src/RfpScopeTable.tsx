import { Fragment, useEffect, useRef, useState, type ReactNode } from "react";
import type { ReportBlock, RfpControls } from "./reportApi";
import { ControlRow, type SettingChange, type Write } from "./RfpEditor";
import { InlineText } from "./InlineText";
import { IconBin, IconUndo } from "./icons";
import { clickRow, emptySelection, prune, type Selection } from "./table/selection";
import { RowMenu, type MenuItem, type RowMenuAt } from "./table/RowMenu";

type Service = { key: string; control?: string; label: string; value: string; group?: string; stage?: string; detail?: string; assumption?: string; party?: string; source: string; inactive?: boolean; notes?: {label:string;text:string}[] };
type Stage = { key: string; name: string; summary?: string; state: string; source?: string; by?: string; text: string; assumption?: string; design_responsibility?: string; fee_basis?: string };
type Props = { block: ReportBlock; controls: RfpControls; busy: boolean; onWrite: Write };

// The letters match the printed key above the issued table.
const positionChoices = [["I", "I", "Included"], ["O", "O", "Optional price"], ["X", "X", "Excluded"], ["ask", "?", "Unknown"]] as const;

export function scopeTable(block: ReportBlock): boolean {
  return !block.source_missing && !!block.table
    && ["hydraulic_services", "hydraulic_stages", "scope_services", "scope_stages", "stage_scope"].includes(String(block.basis?.layout));
}

export function RfpScopeTable({ block, controls, busy, onWrite }: Props) {
  const [pending, setPending] = useState<{ key: string; value: string } | null>(null);
  useEffect(() => { if (!busy) setPending(null); }, [busy]);
  if (block.basis?.layout === "stage_scope") return <StageScope block={block} controls={controls} busy={busy} onWrite={onWrite} />;
  if (["hydraulic_stages", "scope_stages"].includes(String(block.basis?.layout))) return <StageFees block={block} controls={controls} busy={busy} onWrite={onWrite} />;
  const services = block.basis?.positions as Service[];
  const positions = controls.groups.flatMap(g => g.controls);
  const controlKey = (s: Service) => s.control ?? `position:${s.key}`;
  const writeChoice = (s: Service, value: string) => onWrite(controlKey(s), !controlKey(s).startsWith("position:") ? ({ I: "include", O: "option", X: "exclude", ask: "ask" }[value]) : value);
  return <div className="hydraulic-scope">
    {block.edited && <p className="report-wording">{block.text}</p>}
    <p className="rfp-scope-note">One click sets each service: I included · O optional, priced on its own · X excluded · ? unknown, to resolve before issue.</p>
    <div className="report-table-scroll"><table className="report-table hydraulic-service-table">
      <thead><tr><th scope="col">Scope of services</th><th scope="col">Position</th></tr></thead>
      <tbody>{services.map((s, i) => <Fragment key={controlKey(s)}> 
        {s.group && s.group !== services[i - 1]?.group && <tr className="hydraulic-group"><th colSpan={2} scope="colgroup">{s.group}</th></tr>}
        <tr data-control={controlKey(s)}>
          <th scope="row"><span>{s.label}</span>{s.detail && <p className="hydraulic-detail">{s.detail}</p>}
            {s.notes?.map((n,i) => <p className="hydraulic-detail" key={i}>{n.label && `${n.label}: `}{n.text}</p>)}
            {s.inactive && <p className="hydraulic-detail">Not required by the current brief. Select a position to add it to this RFP.</p>}
            {s.assumption && <p className="hydraulic-detail">Pricing basis: {s.assumption}</p>}
            {s.value === "ask" && <p className="hydraulic-unresolved">Scope not decided. Choose a position before issue.</p>}
            {s.value === "X" && controlKey(s).startsWith("position:") && !s.inactive && <Responsible service={s} busy={busy} onWrite={onWrite} />}
            {positions.find(p => p.key === controlKey(s))?.set && <button type="button" className="cell-link" disabled={busy} onClick={() => void onWrite(controlKey(s), null)}>Reset selection</button>}
            <ScopeChecks controls={controls} controlKey={controlKey(s)} label={positions.find(p => p.key === controlKey(s))?.label ?? s.label} busy={busy} onWrite={onWrite} />
            {!controlKey(s).startsWith("stage:") && <ScopeInputs controls={controls} target={controlKey(s)} busy={busy} onWrite={onWrite} />}
          </th>
          <td className="rfp-scope-position"><div className="rfp-position-choice" role="radiogroup" aria-label={`${s.label}: position`}>
            {positionChoices.map(([value, short, label]) => <label key={value} title={label}>
              <input type="radio" name={`scope-${controlKey(s)}`} value={value} aria-label={`${s.label}: ${label}`} checked={(pending?.key === controlKey(s) ? pending.value : s.value) === value} disabled={busy} onChange={() => { setPending({ key: controlKey(s), value }); void writeChoice(s, value); }} />
              <span aria-hidden="true">{short}</span>
            </label>)}
          </div></td>
        </tr>
      </Fragment>)}<tr><td colSpan={2}><ScopeInputs controls={controls} target="scope_basis" available={[...services.map(controlKey), ...controls.boundary.stages.map(s => `stage:${s.value}`)]} busy={busy} onWrite={onWrite} /></td></tr></tbody>
    </table></div>
    <p className="rfp-scope-note">Tell us in R6 if you propose a departure from these selections.</p>
  </div>;
}

type Line = { key: string; text: string; value: string; source: string; custom?: boolean; edited?: boolean };
type ScopeStage = Stage & { lines?: Line[]; by_element?: boolean };
type Element = { key: string; code: string; name: string; state: string };

const upperFirst = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

/** One In and one Excluded tick box; ticking one clears the other. */
function InOut({ label, value, disabled, onChoose }: { label: string; value: "I" | "X"; disabled: boolean; onChoose: (v: "I" | "X") => void }) {
  return <>
    <td className="rfp-inout"><label className="rfp-tick"><input type="checkbox" aria-label={`${label}: included`} checked={value === "I"} disabled={disabled} onChange={() => { if (value !== "I") onChoose("I"); }} /></label></td>
    <td className="rfp-inout"><label className="rfp-tick"><input type="checkbox" aria-label={`${label}: excluded`} checked={value === "X"} disabled={disabled} onChange={() => { if (value !== "X") onChoose("X"); }} /></label></td>
  </>;
}

const newLineID = () => "u" + Date.now().toString(36);
/** A line's wording as saved: one line, no {slot} braces. */
const cleanLine = (text: string) => text.replace(/\s+/g, " ").trim().replace(/[{}]/g, "");
/** Selection ids name a line within its stage. */
const lineID = (stage: string, line: string) => `${stage}/${line}`;
const splitID = (id: string) => id.split("/") as [string, string];
const items = (n: number) => `${n} ${n === 1 ? "item" : "items"}`;

/**
 * Scope by stage (owner direction, 3 and 8 November 2026). A closed stage is
 * one row: its name with a count, a preview of its lines, an In and an
 * Excluded tick for the whole stage, and its one fee cell. Opened, the name
 * and the fee cell run down beside its lines; every line is edited where it
 * reads, ticked in or out, and deleted alone or with Ctrl and Shift
 * selections. "Add item…" at the foot of a stage adds a line.
 */
function StageScope({ block, controls, busy, onWrite }: Props) {
  const [open, setOpen] = useState<Set<string>>(new Set());
  const [showAll, setShowAll] = useState(false);
  const [edit, setEdit] = useState<{ id: string; text: string } | null>(null);
  const [ghost, setGhost] = useState<Record<string, string>>({});
  const [selection, setSelection] = useState<Selection>(emptySelection);
  const [menu, setMenu] = useState<(RowMenuAt & { ids: string[] }) | null>(null);
  // The tick shows at once while it saves; the saved draft replaces it.
  const [pending, setPending] = useState<Record<string, "I" | "X">>({});
  useEffect(() => { if (!busy) setPending({}); }, [busy]);
  const tick = (key: string, value: "I" | "X") => pending[key] ?? value;
  const choose = (key: string, value: "I" | "X", write: () => Promise<void>) => { setPending(p => ({ ...p, [key]: value })); void write(); };
  const stages = (block.basis?.stages ?? []) as ScopeStage[];
  const services = (block.basis?.services ?? []) as Service[];
  const elements = (block.basis?.elements ?? []) as Element[];
  const table = block.table!;
  // A discipline's part of a combined engagement prices nothing itself: its
  // fees are asked once, in the engagement's fee schedule.
  const priced = table.columns.length > 3;
  const stageControls = controls.groups.find(g => g.id === "stages")?.controls ?? [];
  const applies = (s: ScopeStage) => !(s.state === "exclude" && s.source === "rule");
  const shown = stages.filter(s => showAll || applies(s));
  const hidden = stages.length - stages.filter(applies).length;
  const held = (key: string) => services.filter(s => (s.stage ?? "") === key && !s.inactive);
  const toggle = (key: string) => setOpen(old => { const next = new Set(old); if (next.has(key)) next.delete(key); else next.add(key); return next; });
  const own = (s: ScopeStage) => (s.lines ?? []).filter(l => l.custom).map(l => ({ id: l.key, text: l.text, value: l.value }));
  const saveOwn = (s: ScopeStage, lines: { id: string; text: string; value: string }[]) => onWrite(`custom:${s.key}`, lines.length ? lines : null);
  // Selection runs over the lines of open stages, in the order they show.
  const order = shown.filter(s => open.has(s.key)).flatMap(s => [...(s.lines ?? []).filter(l => !l.custom), ...(s.lines ?? []).filter(l => l.custom)].map(l => lineID(s.key, l.key)));
  const orderKey = order.join("|");
  useEffect(() => setSelection(s => prune(s, orderKey ? orderKey.split("|") : [])), [orderKey]);
  const selected = new Set(selection.ids);

  // A printed RFP shows every stage open, whatever the screen shows.
  const beforePrint = useRef<Set<string> | null>(null);
  const allKeys = stages.map(s => s.key).join("|");
  useEffect(() => {
    const before = () => setOpen(now => { beforePrint.current = now; return new Set(["", ...allKeys.split("|")]); });
    const after = () => { if (beforePrint.current) setOpen(beforePrint.current); beforePrint.current = null; };
    window.addEventListener("beforeprint", before);
    window.addEventListener("afterprint", after);
    return () => { window.removeEventListener("beforeprint", before); window.removeEventListener("afterprint", after); };
  }, [allKeys]);

  const findLine = (id: string) => {
    const [stage, key] = splitID(id);
    const s = stages.find(x => x.key === stage);
    return { s, l: s?.lines?.find(x => x.key === key) };
  };
  const saveWording = (s: ScopeStage, l: Line, raw: string) => {
    const text = cleanLine(raw);
    setEdit(null);
    if (!text || text === upperFirst(l.text)) return;
    if (l.custom) void saveOwn(s, own(s).map(o => o.id === l.key ? { ...o, text } : o));
    else void onWrite(`wording:${s.key}.${l.key}`, text);
  };
  const addLine = (s: ScopeStage) => {
    const text = cleanLine(ghost[s.key] ?? "");
    if (!text) return;
    setGhost(g => ({ ...g, [s.key]: "" }));
    void saveOwn(s, [...own(s), { id: newLineID(), text, value: "I" }]);
  };
  // Deleting is final (owner, 8 November 2026): a standard line leaves this
  // RFP and the user's own line is removed, all in one write.
  const remove = (ids: string[]) => {
    const changes: SettingChange[] = [];
    const customs = new Map<string, Set<string>>();
    for (const id of ids) {
      const { s, l } = findLine(id);
      if (!s || !l) continue;
      if (l.custom) { customs.set(s.key, (customs.get(s.key) ?? new Set()).add(l.key)); continue; }
      changes.push({ key: `line:${s.key}.${l.key}`, value: "D" });
      if (l.edited) changes.push({ key: `wording:${s.key}.${l.key}`, value: null });
    }
    for (const [stage, gone] of customs) {
      const s = stages.find(x => x.key === stage)!;
      const kept = own(s).filter(o => !gone.has(o.id));
      changes.push({ key: `custom:${stage}`, value: kept.length ? kept : null });
    }
    setSelection(emptySelection);
    if (!changes.length) return;
    if (onWrite.many) void onWrite.many(changes);
    else void (async () => { for (const c of changes) await onWrite(c.key, c.value); })();
  };
  const menuItems = (ids: string[]): MenuItem[] => {
    const out: MenuItem[] = [];
    const one = ids.length === 1 ? findLine(ids[0]) : null;
    if (one?.s && one.l?.edited) {
      const key = `wording:${one.s.key}.${one.l.key}`;
      out.push({ label: "Reset to standard wording", icon: <IconUndo />, disabled: busy, onSelect: () => void onWrite(key, null) }, "sep");
    }
    out.push({ label: ids.length === 1 ? "Delete line" : `Delete ${ids.length} lines`, icon: <IconBin />, danger: true, disabled: busy, onSelect: () => remove(ids) });
    return out;
  };
  const rowClick = (id: string) => (e: React.MouseEvent<HTMLTableRowElement>) => {
    if ((e.target as HTMLElement).closest("input,button,textarea,select,label")) return;
    if (e.ctrlKey || e.metaKey || e.shiftKey) {
      e.preventDefault();
      window.getSelection()?.removeAllRanges();
      setSelection(s => clickRow(s, id, order, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey }));
      e.currentTarget.focus({ preventScroll: true });
    } else if (selection.ids.length) setSelection(emptySelection);
  };
  // Right-click acts on the selection when the line is in it; otherwise the
  // line becomes the selection, as the document register does.
  const rowMenu = (id: string) => (e: React.MouseEvent<HTMLTableRowElement>) => {
    if ((e.target as HTMLElement).closest("textarea,input")) return;
    e.preventDefault();
    const ids = selected.has(id) ? [...selection.ids] : [id];
    if (!selected.has(id)) setSelection(clickRow(emptySelection, id, order));
    setMenu({ x: e.clientX, y: e.clientY, trigger: e.currentTarget, ids });
  };
  const keys = (e: React.KeyboardEvent) => {
    if ((e.target as HTMLElement).closest("textarea,input")) return;
    if ((e.key === "Delete" || e.key === "Backspace") && selection.ids.length && !busy) { e.preventDefault(); remove([...selection.ids]); }
    if (e.key === "Escape" && selection.ids.length) setSelection(emptySelection);
  };

  // The stage cell: its name opens and closes it (owner direction, 7 November
  // 2026: no button box); a small caret shows which way, and the count of
  // its items sits in a circle beside it.
  const stageCell = (key: string, name: string, count: number, isOpen: boolean, rows = 1, extra?: ReactNode) => <th scope="row" className="rfp-stage-name" rowSpan={rows > 1 ? rows : undefined}>
    <button type="button" className="rfp-stage-toggle" aria-expanded={isOpen} aria-controls={`rfp-stage-${key || "all"}`} onClick={() => toggle(key)}>
      <span className="rfp-stage-caret" aria-hidden="true" /><span>{name}</span>
      <span className="rfp-stage-badge" title={items(count)} aria-hidden="true">{count}</span>
    </button>
    <span className="sr-only">{items(count)}</span>
    {extra}
  </th>;
  const feeCell = (text: string, rows = 1) => priced ? <td className="rfp-fee rfp-fee-stage" rowSpan={rows > 1 ? rows : undefined}>{text}</td> : null;
  const serviceRow = (s: Service, stageOut: boolean, start: ReactNode = null, end: ReactNode = null) => <tr key={s.key} className="rfp-stage-line" data-control={s.control ?? `position:${s.key}`}>
    {start}
    <td>{s.label}{s.detail && <p className="hydraulic-detail">{s.detail}</p>}
      {s.notes?.map((n, i) => <p className="hydraulic-detail" key={i}>{n.label && `${n.label}: `}{n.text}</p>)}
      <ScopeChecks controls={controls} controlKey={s.control ?? `position:${s.key}`} label={s.label} busy={busy} onWrite={onWrite} />
      <ScopeInputs controls={controls} target={s.control ?? `position:${s.key}`} busy={busy} onWrite={onWrite} /></td>
    <InOut label={s.label} value={stageOut ? "X" : tick(s.key, s.value === "X" ? "X" : "I")} disabled={busy || stageOut} onChoose={v => choose(s.key, v, () => onWrite(s.control ?? `position:${s.key}`, v))} />
    {end}
  </tr>;

  return <>
    {controls.decisions.filter(c => c.control === "boundary:range").map(c => <p key={c.id} data-control="boundary:range" className="hydraulic-unresolved">{c.label}. {c.text}</p>)}
    {block.edited && <p className="report-wording">{block.text}</p>}
    <div className="report-table-scroll" onKeyDown={keys}><table className="report-table rfp-stage-scope">
      <colgroup><col className="rfp-col-stage" /><col /><col className="rfp-col-tick" /><col className="rfp-col-tick" />{priced && <col className="rfp-col-fee" />}</colgroup>
      <thead><tr><th scope="col">Stage</th><th scope="col">Scope</th><th scope="col" className="rfp-inout">In</th><th scope="col" className="rfp-inout" aria-label="Excluded"><abbr title="Excluded">Excl.</abbr></th>{priced && <th scope="col" className="rfp-fee">Fee $</th>}</tr></thead>
      {shown.map(s => {
        const control = stageControls.find(c => c.key === `stage:${s.key}`);
        const out = tick(`stage:${s.key}`, s.state === "exclude" ? "X" : "I") === "X";
        const isOpen = open.has(s.key);
        const lines = s.lines ?? [];
        const stageElements = s.by_element ? elements : [];
        const stageServices = held(s.key);
        const count = lines.length + stageElements.length + stageServices.length;
        if (!isOpen) {
          const preview = out ? "Excluded" : lines.filter(l => l.value !== "X").map(l => upperFirst(l.text).replace(/[.;]+$/, "")).join("; ") || (count ? "" : "No items");
          return <tbody key={s.key} id={`rfp-stage-${s.key}`} data-stage-state={s.state}>
            <tr className="rfp-stage-row" data-control={`stage:${s.key}`}>
              {stageCell(s.key, s.name, count, false)}
              <td className="rfp-stage-summary" onClick={e => { if (!(e.target as HTMLElement).closest("button,input,select,textarea,a,label")) toggle(s.key); }}>
                <span className="rfp-stage-preview" title={preview}>{preview}</span>
                <ScopeChecks controls={controls} controlKey={`stage:${s.key}`} label={s.name} busy={busy} onWrite={onWrite} /></td>
              <InOut label={s.name} value={out ? "X" : "I"} disabled={busy || !!control?.locked} onChoose={v => choose(`stage:${s.key}`, v, () => onWrite(`stage:${s.key}`, v === "I" ? "include" : "exclude"))} />
              {feeCell(out ? "—" : "$")}
            </tr>
          </tbody>;
        }
        // Open: the stage cell and the fee cell run down beside every row;
        // the first row carries both.
        const note = [s.design_responsibility, s.assumption].filter(Boolean).join(" ");
        const extra = <>
          {note && <p className="hydraulic-detail">{note}</p>}
          <ScopeChecks controls={controls} controlKey={`stage:${s.key}`} label={s.name} busy={busy} onWrite={onWrite} />
          <ScopeInputs controls={controls} target={`stage:${s.key}`} busy={busy} onWrite={onWrite} />
        </>;
        const rows = count + (out ? 0 : 1);
        let first = true;
        const lead = (): [ReactNode, ReactNode] => {
          if (!first) return [null, null];
          first = false;
          return [stageCell(s.key, s.name, count, true, rows, extra), feeCell(out ? "—" : "$", rows)];
        };
        const lineRow = (l: Line) => {
          const id = lineID(s.key, l.key);
          const shownText = upperFirst(l.text);
          const editing = edit?.id === id;
          const [start, end] = lead();
          return <tr key={l.key} className="rfp-stage-line rfp-line" data-line={l.key} data-edited={l.edited || undefined} data-selected={selected.has(id) || undefined}
            aria-selected={selected.has(id)} tabIndex={-1} onClick={rowClick(id)} onContextMenu={rowMenu(id)}>
            {start}
            <td className="rfp-line-text">
              <InlineText compact text={shownText} label={`Scope line: ${shownText}`} editing={editing} value={editing ? edit.text : shownText} maxLength={300}
                disabled={busy || out || (!!edit && !editing)} busy={busy}
                onStart={() => { setSelection(emptySelection); setEdit({ id, text: shownText }); }}
                onChange={text => setEdit({ id, text })} onSave={() => saveWording(s, l, edit?.text ?? "")} onCancel={() => setEdit(null)} />
            </td>
            <InOut label={shownText} value={out ? "X" : tick(`${s.key}.${l.key}`, l.value === "X" ? "X" : "I")} disabled={busy || out}
              onChoose={v => choose(`${s.key}.${l.key}`, v, () => l.custom ? saveOwn(s, own(s).map(o => o.id === l.key ? { ...o, value: v } : o)) : onWrite(`line:${s.key}.${l.key}`, v))} />
            {end}
          </tr>;
        };
        const body: ReactNode[] = [
          ...lines.filter(l => !l.custom).map(lineRow),
          ...stageElements.map(e => {
            const [start, end] = lead();
            return <tr key={`element-${e.key}`} className="rfp-stage-line" data-control={`element:${e.key}`}>
              {start}<td>{e.code} {e.name}</td>
              <InOut label={`${e.code} ${e.name}`} value={out ? "X" : tick(`element:${e.key}`, e.state === "exclude" ? "X" : "I")} disabled={busy || out}
                onChoose={v => choose(`element:${e.key}`, v, () => onWrite(`element:${e.key}`, v === "I" ? "include" : "exclude"))} />
              {end}
            </tr>;
          }),
          ...stageServices.map(p => { const [start, end] = lead(); return serviceRow(p, out, start, end); }),
          ...lines.filter(l => l.custom).map(lineRow),
        ];
        if (!out) {
          // The ghost line: a quiet "Add item…" that becomes a line on Enter
          // or when clicked away from (owner direction, 8 November 2026, as
          // the Cost Plan's).
          const [start, end] = lead();
          body.push(<tr key="add" className="rfp-stage-add">
            {start}
            <td colSpan={3}><textarea className="rfp-ghost" rows={1} aria-label={`New scope line for ${s.name}`} placeholder="Add item…" maxLength={300} disabled={busy}
              value={ghost[s.key] ?? ""} onChange={e => { setGhost(g => ({ ...g, [s.key]: e.target.value.replace(/\n/g, " ") })); grow(e.target); }}
              onKeyDown={e => { if (e.key === "Enter") { e.preventDefault(); addLine(s); } if (e.key === "Escape") setGhost(g => ({ ...g, [s.key]: "" })); }}
              onBlur={() => addLine(s)} /></td>
            {end}
          </tr>);
        }
        return <tbody key={s.key} id={`rfp-stage-${s.key}`} data-stage-state={s.state} data-open="true">
          {body.length ? body : <tr className="rfp-stage-row">{stageCell(s.key, s.name, count, true, 1, extra)}<td className="rfp-stage-summary">{out ? "Excluded" : "No items"}</td><td /><td />{feeCell(out ? "—" : "$")}</tr>}
        </tbody>;
      })}
      <tbody id="rfp-stage-all">
        {(() => {
          const across = held("");
          const isOpen = open.has("") && across.length > 0;
          if (!isOpen) return <tr className="rfp-stage-row">{stageCell("", "Across all stages", across.length, false)}
            <td className="rfp-stage-summary" onClick={() => toggle("")}><span className="rfp-stage-preview">{across.map(p => p.label).join("; ") || "No items"}</span></td><td /><td />{feeCell("In stage fees")}</tr>;
          return across.map((p, i) => i === 0
            ? serviceRow(p, false, stageCell("", "Across all stages", across.length, true, across.length), feeCell("In stage fees", across.length))
            : serviceRow(p, false));
        })()}
      </tbody>
      <tbody>{table.rows.slice(Number(block.basis?.stage_rows ?? 0)).map((row, i) => <tr key={`total-${i}`} className="rfp-stage-total"><th scope="row">{row[0]}</th><td>{row[1]}</td><td /><td />{priced && <td className="rfp-fee" />}</tr>)}</tbody>
    </table></div>
    {selection.ids.length > 0 && <div className="rfp-line-selection" role="toolbar" aria-label="Selected scope lines">
      <span>{selection.ids.length} {selection.ids.length === 1 ? "line" : "lines"} selected</span>
      <button type="button" className="btn btn-small rfp-line-delete" disabled={busy} onClick={() => remove([...selection.ids])}><IconBin /> Delete</button>
      <button type="button" className="btn btn-small" onClick={() => setSelection(emptySelection)}>Clear</button>
    </div>}
    {menu && <RowMenu at={menu} items={menuItems(menu.ids)} label={menu.ids.length > 1 ? `Actions for ${menu.ids.length} scope lines` : "Scope line actions"}
      head={menu.ids.length > 1 ? `${menu.ids.length} lines selected` : undefined} onClose={restore => { if (restore) menu.trigger?.focus({ preventScroll: true }); setMenu(null); }} />}
    {hidden > 0 && <button type="button" className="btn btn-small rfp-stage-more" aria-expanded={showAll} onClick={() => setShowAll(!showAll)}>{showAll ? "Hide stages that do not apply" : `Show ${hidden} ${hidden === 1 ? "stage" : "stages"} that do not apply to this project`}</button>}
    {/* Inputs whose row is not shown (a service the brief does not need) stay reachable here. */}
    <ScopeInputs controls={controls} target="scope_basis" available={[...services.filter(s => !s.inactive).map(s => s.control ?? `position:${s.key}`), ...stages.filter(applies).map(s => `stage:${s.key}`)]} busy={busy} onWrite={onWrite} />
  </>;
}

/** The ghost line grows with its text: no inner scroll bar. */
function grow(el: HTMLTextAreaElement) {
  el.style.height = "auto";
  el.style.height = `${el.scrollHeight + el.offsetHeight - el.clientHeight}px`;
}

function Responsible({ service, busy, onWrite }: { service: Service; busy: boolean; onWrite: Write }) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState(service.party ?? "");
  useEffect(() => { setValue(service.party ?? ""); }, [service.party]);
  return editing ? <form className="hydraulic-owner" onSubmit={e => { e.preventDefault(); if (value.trim()) void onWrite(`responsible:${service.key}`, value.trim()).then(() => setEditing(false)); }}>
    <label>Responsible party<input value={value} onChange={e => setValue(e.target.value)} disabled={busy} maxLength={500} required /></label>
    <button className="btn btn-small" disabled={busy || !value.trim()}>Save party</button>
    <button type="button" className="cell-link" disabled={busy} onClick={() => setEditing(false)}>Cancel</button>
  </form> : <p className="hydraulic-detail">By {service.party || "party not named"}. <button type="button" className="cell-link" disabled={busy} onClick={() => setEditing(true)}>Change party</button></p>;
}

function StageFees({ block, controls, busy, onWrite }: Props) {
  const [pending, setPending] = useState<{ key: string; included: boolean } | null>(null);
  useEffect(() => { if (!busy) setPending(null); }, [busy]);
  const stages = block.basis?.stages as Stage[];
  const table = block.table!;
  const pricingRows = (block.basis?.pricing_rows ?? {}) as Record<string,string[][]>;
  const stageControls = controls.groups.find(g => g.id === "stages")?.controls ?? [];
  return <>
    {controls.decisions.filter(c => c.control === "boundary:range").map(c => <p key={c.id} data-control="boundary:range" className="hydraulic-unresolved">{c.label}. {c.text}</p>)}
    <p className="report-wording">Checked stages form the base fee. Unchecked options are priced separately. {block.text}</p>
    <div className="report-table-scroll"><table className="report-table hydraulic-stage-table">
      <thead><tr>{table.columns.map(c => <th scope="col" key={c}>{c}</th>)}</tr></thead>
      <tbody>{stages.map(s => <Fragment key={s.key}><tr data-control={`stage:${s.key}`}>
        <th scope="row"><label className="hydraulic-stage-check"><input type="checkbox" checked={pending?.key === s.key ? pending.included : s.state === "include"} aria-label={`Include ${s.name} in base fee`} disabled={busy || stageControls.find(c => c.key === `stage:${s.key}`)?.locked} onChange={e => { setPending({ key: s.key, included: e.target.checked }); void onWrite(`stage:${s.key}`, e.target.checked ? "include" : "option"); }} />{s.name}</label>
          {s.state !== "include" && <select aria-label={`${s.name}: unchecked stage`} value={s.state} disabled={busy || stageControls.find(c => c.key === `stage:${s.key}`)?.locked} onChange={e => void onWrite(`stage:${s.key}`, e.target.value)}>
            <option value="ask">Unknown</option><option value="option">Optional price</option><option value="exclude">Not in this RFP</option><option value="done">Done by others</option>
          </select>}
          {stageControls.find(c => c.key === `stage:${s.key}`)?.set && <button type="button" className="cell-link" disabled={busy} onClick={() => void onWrite(`stage:${s.key}`, null)}>Reset stage</button>}
        </th>
        <td><ScopeChecks controls={controls} controlKey={`stage:${s.key}`} label={s.name} busy={busy} onWrite={onWrite} />{s.text}{s.design_responsibility && ` ${s.design_responsibility}`}{s.assumption && ` ${s.assumption}`} {s.fee_basis === "time_to_ceiling" && "Fee ceiling: hours × R3 rate."}{s.fee_basis === "per_unit" && "Fee: rate per unit × quantity."}<ScopeInputs controls={controls} target={`stage:${s.key}`} busy={busy} onWrite={onWrite} /></td>
        {table.columns.slice(2).map(h => h.startsWith("Fee") ? "Fee" : h).map(c => <td key={c} className="hydraulic-fee-blank" aria-label={`${c}: ${s.state === "exclude" || s.state === "done" ? "not applicable" : "consultant to complete"}`}>{s.state === "exclude" || s.state === "done" ? "—" : c === "Fee" && pricingRows[s.key]?.length ? "By element below" : ""}</td>)}
      </tr>{(pricingRows[s.key] ?? []).map((row,i) => <tr key={i} className="rfp-element-fee">{row.map((cell,j) => j === 0 ? <th scope="row" key={j}>{cell}</th> : <td key={j}>{cell}</td>)}</tr>)}</Fragment>)}
      {table.rows.slice(Number(block.basis?.stage_rows ?? stages.length)).map((row, i) => <tr key={`total-${i}`}>{row.map((cell, j) => j === 0 ? <th scope="row" key={j}>{cell}</th> : <td key={j}>{cell}</td>)}</tr>)}
      </tbody>
    </table></div>
  </>;
}


// Scope choices do not rewrite project facts. Conflicting evidence can still
// be corrected beside the affected service, without a second questionnaire.
function ScopeChecks({ controls, controlKey, label, busy, onWrite }: { controls: RfpControls; controlKey: string; label: string; busy: boolean; onWrite: Write }) {
  const checks = controls.decisions.filter(c => c.control === controlKey || c.changes?.includes(label)
    || (controlKey === "position:rainwater" && c.id === "legacy:flag.hyd_rainwater")
    || (controlKey === "position:alternative_sources" && c.id === "legacy:flag.hyd_alternative_sources"));
  const facts = controls.groups.flatMap(g => g.controls);
  return <>{checks.filter(c => c.label.includes("sources disagree") || c.id.startsWith("legacy:") || (c.control === controlKey && !c.id.startsWith("scope:"))).map(c => {
    const fact = facts.find(f => f.key === c.control);
    return <div className="hydraulic-unresolved" key={c.id} data-check={c.id}>
      <p>{c.label}. {c.id.startsWith("legacy:") ? c.text : "Resolve the conflicting project information before issue."}</p>
      {fact && !fact.locked && <div role="group" aria-label={`Resolve project information for ${label}`}>
        <span>Is this system or service required by the project? </span>
        {[['yes', 'Yes'], ['no', 'No']].map(([value, text]) => <button type="button" className="btn btn-small" key={value} disabled={busy} onClick={() => void onWrite(fact.key, value)}>{text}</button>)}
      </div>}
    </div>;
  })}</>;
}

function ScopeInputs({ controls, target, available, busy, onWrite }: { controls: RfpControls; target: string; available?: string[]; busy: boolean; onWrite: Write }) {
  const inputs = (controls.groups.find(g => g.id === "appointment")?.controls ?? []).filter(c => c.target === target || (target === "scope_basis" && !available?.includes(c.target ?? "scope_basis")));
  if (!inputs.length) return null;
  return <details className="rfp-scope-inputs"><summary>{target === "scope_basis" ? (controls.kind === "works" ? "Tender inputs" : "Appointment and service-level inputs") : "Scope basis"}</summary><p className="hydraulic-detail">These inputs describe the project. The scope selection above sets what this {controls.kind === "works" ? "tenderer" : "consultant"} is asked to price.</p>{inputs.map(c => <ControlRow key={c.key} control={c} busy={busy} showSource={false} blocked={controls.decisions.some(d => d.control === c.key)} onWrite={onWrite} />)}</details>;
}
