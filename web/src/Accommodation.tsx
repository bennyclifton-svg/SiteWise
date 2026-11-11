// The Accommodation schedule: the spaces the building provides after the
// works, one line per room, for any building type
// (docs/plans/2026-11-11-accommodation-schedule-lines.md). The server builds
// the lines from the building type's typical spaces (blue), the documents'
// readings (green, amber to check, red where they disagree), the card's
// counts and the user's edits (rust, final). Selection, editing and the row
// menu work as in the document register and the RFP scope table. No Jev.

import { useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";
import { ConflictPop, Sources } from "./ProfileFields";
import { IconBin, IconMore, IconPlus, IconRename, IconSubdivide, IconTag, IconUndo } from "./icons";
import { profileApi, type Accommodation as Table, type AccommodationEdit, type LineField, type ScheduleLine } from "./profileApi";
import { RevisionMark } from "./RunMarks";
import { InlineCellEditor, type CommitKey } from "./table/InlineCellEditor";
import { RowHandle, startRowDrag } from "./table/RowHandle";
import { RowMenu, type MenuItem, type RowMenuAt } from "./table/RowMenu";
import { clickRow, emptySelection, prune, selectAll, type Selection } from "./table/selection";
import "./accommodation.css";

const FIELDS: { field: LineField; label: string; numeric?: boolean }[] = [
  { field: "name", label: "Space" },
  { field: "level", label: "Level" },
  { field: "area", label: "Area m²", numeric: true },
  { field: "comment", label: "Comment" },
];

const ORIGIN: Record<ScheduleLine["origin"], string> = {
  typical: "Typical for this building type; not read from a document",
  document: "Read from the documents",
  card: "From the count on the Project card",
  user: "Added by you",
};

/** A typed value the server would refuse, said before it is sent. */
function invalid(field: LineField, text: string): string {
  const v = text.trim();
  if (field === "name" && !v) return "A space needs a name";
  if (field === "area" && v && !/^\d[\d,]*(\.\d+)?\s*(m²|m2|sqm)?$/i.test(v)) return "Square metres, such as 24 or 12.5";
  return "";
}

export function Accommodation({ projectId, table, markOf, onChange, onJump, onError }: {
  projectId: string;
  table: Table;
  /** What the last update changed under these readings, if anything. */
  markOf: (keys: string[]) => string | undefined;
  onChange: (next: Table) => void;
  onJump: (id: string) => void;
  onError: (e: unknown) => void;
}) {
  const lines = table.lines;
  const order = lines.map((l) => l.id);
  const [selection, setSelection] = useState<Selection>(emptySelection);
  const [edit, setEdit] = useState<{ id: string; field: LineField; many?: string[]; error?: string } | null>(null);
  const [menu, setMenu] = useState<(RowMenuAt & { ids: string[] }) | null>(null);
  const [busy, setBusy] = useState(false);
  const [drop, setDrop] = useState<{ id: string; after: boolean } | null>(null);
  const [ghost, setGhost] = useState("");
  const dragging = useRef(false);
  const tableRef = useRef<HTMLTableElement>(null);
  const orderKey = order.join("|");
  useEffect(() => setSelection((s) => prune(s, orderKey ? orderKey.split("|") : [])), [orderKey]);
  const selected = new Set(selection.ids);
  const lineOf = (id: string) => lines.find((l) => l.id === id);

  const run = async (e: AccommodationEdit) => {
    setBusy(true);
    try {
      onChange(await profileApi.editAccommodation(projectId, e));
      return true;
    } catch (err) {
      onError(err);
      return false;
    } finally {
      setBusy(false);
    }
  };
  // Restores keyboard focus to the table after a menu action or an edit.
  const refocus = () => requestAnimationFrame(() => tableRef.current?.focus({ preventScroll: true }));

  const commit = async (id: string, field: LineField, text: string, key: CommitKey, many?: string[]) => {
    const line = lineOf(id);
    const error = invalid(field, text);
    if (error) {
      setEdit({ id, field, many, error });
      return;
    }
    const value = text.trim();
    const ids = many ?? [id];
    const same = ids.length === 1 && line?.[field].value === value;
    if (!same && !(await run({ op: "set", lines: ids, field, value }))) {
      setEdit({ id, field, many, error: "Not saved. Try again." });
      return;
    }
    // Tab moves along the line, as in the register; Enter and leaving stop.
    const at = FIELDS.findIndex((f) => f.field === field);
    const next = key === "tab" ? FIELDS[at + 1] : key === "shift-tab" ? FIELDS[at - 1] : undefined;
    setEdit(next && !many ? { id, field: next.field } : null);
    if (!next) refocus();
  };

  const insertAt = (index: number, name = "New space") => {
    const next = [...order];
    next.splice(index, 0, "new:0");
    void run({ op: "insert", order: next, value: name });
  };
  const remove = (ids: string[]) => {
    setSelection(emptySelection);
    void run({ op: "delete", lines: ids });
    refocus();
  };

  const menuItems = (ids: string[]): MenuItem[] => {
    const one = ids.length === 1 ? lineOf(ids[0]) : undefined;
    const at = one ? order.indexOf(one.id) : -1;
    const edited = ids.some((id) => {
      const l = lineOf(id);
      return id.includes("#") && l && FIELDS.some((f) => l[f.field].band === "user");
    });
    const out: MenuItem[] = [];
    if (one) out.push({ label: "Rename", icon: <IconRename />, disabled: busy, onSelect: () => setEdit({ id: one.id, field: "name" }) });
    out.push({ label: ids.length === 1 ? "Set level" : `Set level for ${ids.length} lines`, icon: <IconTag />, disabled: busy,
      onSelect: () => setEdit({ id: ids.find((id) => order.includes(id)) ?? ids[0], field: "level", many: ids.length > 1 ? ids : undefined }) });
    out.push("sep");
    if (one) {
      out.push({ label: "Insert line above", icon: <IconPlus />, disabled: busy, onSelect: () => insertAt(at) });
      out.push({ label: "Insert line below", icon: <IconPlus />, disabled: busy, onSelect: () => insertAt(at + 1) });
    }
    out.push({ label: ids.length === 1 ? "Duplicate" : `Duplicate ${ids.length} lines`, icon: <IconPlus />, disabled: busy, onSelect: () => void run({ op: "duplicate", lines: ids }) });
    if (one?.many) out.push({ label: `Split into ${one.many} lines`, icon: <IconSubdivide />, disabled: busy, onSelect: () => void run({ op: "split", lines: [one.id] }) });
    if (edited) out.push({ label: "Reset to documents", icon: <IconUndo />, disabled: busy, onSelect: () => void run({ op: "reset", lines: ids.filter((id) => id.includes("#")) }) });
    out.push("sep");
    out.push({ label: ids.length === 1 ? "Delete line" : `Delete ${ids.length} lines`, icon: <IconBin />, danger: true, disabled: busy, onSelect: () => remove(ids) });
    return out;
  };

  // A click selects the line; Ctrl/Cmd adds or removes it, Shift a range.
  const rowClick = (id: string) => (e: React.MouseEvent<HTMLTableRowElement>) => {
    if (dragging.current || (e.target as HTMLElement).closest("input,button,select,details,textarea")) return;
    if (e.shiftKey) {
      e.preventDefault();
      window.getSelection()?.removeAllRanges();
    }
    setSelection((s) => clickRow(s, id, order, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey }));
  };
  // Right-click acts on the selection when the line is in it; otherwise the
  // line becomes the selection, as the document register does.
  const openMenu = (id: string, x: number, y: number, trigger: HTMLElement | null) => {
    const ids = selected.has(id) ? [...selection.ids] : [id];
    if (!selected.has(id)) setSelection(clickRow(emptySelection, id, order));
    setMenu({ x, y, trigger, ids });
  };
  const keys = (e: React.KeyboardEvent) => {
    if ((e.target as HTMLElement).closest("input,select,textarea,details")) return;
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "a") {
      e.preventDefault();
      setSelection(selectAll(order));
    } else if ((e.key === "Delete" || e.key === "Backspace") && selection.ids.length && !busy) {
      e.preventDefault();
      remove([...selection.ids]);
    } else if (e.key === "Escape" && selection.ids.length) {
      setSelection(emptySelection);
    } else if ((e.key === "Enter" || e.key === "F2") && selection.ids.length === 1 && !busy && !edit) {
      e.preventDefault();
      setEdit({ id: selection.ids[0], field: "name" });
    } else if ((e.key === "ContextMenu" || (e.shiftKey && e.key === "F10")) && selection.ids.length) {
      e.preventDefault();
      const row = tableRef.current?.querySelector<HTMLElement>(`tr[data-id="${CSS.escape(selection.ids[0])}"]`);
      const r = row?.getBoundingClientRect();
      openMenu(selection.ids[0], r ? r.left + 24 : 0, r ? r.bottom : 0, row ?? null);
    }
  };
  // Drag a line by its handle; a selected line takes the selection with it.
  const handleDown = (e: ReactPointerEvent<HTMLElement>, id: string) => {
    if (busy) return;
    const moving = selected.has(id) ? order.filter((x) => selected.has(x)) : [id];
    if (!selected.has(id)) setSelection(clickRow(emptySelection, id, order));
    let target: { id: string; after: boolean } | null = null;
    startRowDrag(e, {
      onStart: () => {
        dragging.current = true;
      },
      onMove: (ev) => {
        const row = document.elementFromPoint(ev.clientX, ev.clientY)?.closest<HTMLElement>(".pf-accom-table tr[data-id]");
        const over = row?.dataset.id;
        target = row && over && !moving.includes(over) ? { id: over, after: ev.clientY > row.getBoundingClientRect().top + row.offsetHeight / 2 } : null;
        setDrop(target);
      },
      onDrop: () => {
        if (!target) return;
        const next = order.filter((x) => !moving.includes(x));
        next.splice(next.indexOf(target.id) + Number(target.after), 0, ...moving);
        void run({ op: "order", order: next });
      },
      onStop: () => {
        setDrop(null);
        window.setTimeout(() => {
          dragging.current = false;
        }, 0);
      },
    });
  };

  const addLine = () => {
    const name = ghost.trim();
    if (!name) return;
    setGhost("");
    insertAt(order.length, name);
  };

  const cell = (l: ScheduleLine, field: LineField, numeric?: boolean) => {
    const c = l[field];
    const editing = edit?.id === l.id && edit.field === field;
    const shown = c.value;
    const changed = markOf(c.keys);
    const why = (
      <span className="pf-why">
        {changed && <RevisionMark text={changed} />}
        <Sources cell={c} onJump={onJump} />
        <ConflictPop cell={c} onPick={(v) => void run({ op: "set", lines: [l.id], field, value: v })} />
      </span>
    );
    const body = editing ? (
      <InlineCellEditor value={c.value} label={`${l.name.value} ${FIELDS.find((f) => f.field === field)?.label}`} error={edit.error ?? ""} busy={busy}
        maxLength={field === "comment" ? 500 : field === "name" ? 120 : 80}
        onCommit={(text, key) => void commit(l.id, field, text, key, edit.many)} onCancel={() => { setEdit(null); refocus(); }} />
    ) : (
      <div className="pf-accom-cell">
        <span className="pf-accom-value" title={shown || undefined}>{shown}</span>
        {why}
      </div>
    );
    const start = () => {
      if (busy) return;
      setSelection(clickRow(emptySelection, l.id, order));
      setEdit({ id: l.id, field });
    };
    if (field === "name") {
      return (
        <th scope="row" key={field} data-field={field} data-band={c.band || undefined} onDoubleClick={start}>
          <RowHandle disabled={busy} onPointerDown={(e) => handleDown(e, l.id)} />
          {body}
          {l.flag && <p className="pf-accom-flag">{l.flag}</p>}
        </th>
      );
    }
    return (
      <td key={field} data-field={field} data-band={c.band || undefined} className={numeric ? "num" : undefined} onDoubleClick={start}>
        {body}
      </td>
    );
  };

  return (
    <section className="pf-section pf-accom" aria-labelledby="pf-accom">
      <div className="pf-head section-band">
        <h2 id="pf-accom">Accommodation</h2>
        {selection.ids.length > 0 && (
          <button type="button" className="pf-accom-bin-many" disabled={busy} onClick={() => remove([...selection.ids])}
            aria-label={`Delete ${selection.ids.length} selected ${selection.ids.length === 1 ? "line" : "lines"}`} title={`Delete ${selection.ids.length} selected`}>
            <IconBin /><span aria-hidden="true">{selection.ids.length}</span>
          </button>
        )}
      </div>
      <div className="pf-accom-scroll">
        <table ref={tableRef} className="pf-accom-table" data-testid="profile-accommodation" tabIndex={0} onKeyDown={keys} aria-busy={busy || undefined}
          aria-label="Accommodation schedule. Click to select, Shift or Ctrl to select more, double-click to edit, right-click for actions.">
          <thead>
            <tr>
              {FIELDS.map((f) => (
                <th scope="col" key={f.field} className={f.numeric ? "num" : undefined}>{f.label}</th>
              ))}
              <th scope="col" className="pf-accom-more"><span className="sr-only">Actions</span></th>
            </tr>
          </thead>
          <tbody>
            {lines.map((l) => (
              <tr key={l.id} data-id={l.id} data-origin={l.origin} data-flag={l.flag ? "true" : undefined} tabIndex={-1}
                aria-selected={selected.has(l.id)} title={ORIGIN[l.origin]}
                data-drop={drop?.id === l.id ? (drop.after ? "after" : "before") : undefined}
                onMouseDown={(e) => {
                  // A modifier click selects lines, never the page's text.
                  if ((e.shiftKey || e.ctrlKey || e.metaKey) && !(e.target as HTMLElement).closest("input,select,textarea")) {
                    e.preventDefault();
                    e.currentTarget.focus({ preventScroll: true });
                  }
                }}
                onClick={rowClick(l.id)}
                onContextMenu={(e) => {
                  if ((e.target as HTMLElement).closest("input")) return;
                  e.preventDefault();
                  openMenu(l.id, e.clientX, e.clientY, e.currentTarget);
                }}>
                {FIELDS.map((f) => cell(l, f.field, f.numeric))}
                <td className="pf-accom-more">
                  <button type="button" className="reg-more-btn" aria-label={`Actions for ${l.name.value}`} title="Actions" aria-haspopup="menu"
                    onClick={(e) => {
                      e.stopPropagation();
                      const r = e.currentTarget.getBoundingClientRect();
                      openMenu(l.id, r.left, r.bottom, e.currentTarget);
                    }}>
                    <IconMore />
                  </button>
                </td>
              </tr>
            ))}
            {/* A grey line to add a space, as the Cost Plan adds a line. */}
            <tr className="pf-accom-ghost">
              <th scope="row" colSpan={FIELDS.length + 1}>
                <input className="pf-accom-add" list="pf-accom-names" value={ghost} disabled={busy} maxLength={120}
                  placeholder={lines.length === 0 ? "No spaces yet. Type a space to add it…" : "Add a space…"} aria-label="Add a space"
                  onChange={(e) => setGhost(e.target.value)}
                  onKeyDown={(e) => {
                    e.stopPropagation();
                    if (e.key === "Enter") {
                      e.preventDefault();
                      addLine();
                    } else if (e.key === "Escape") setGhost("");
                  }}
                  onBlur={addLine} />
                <datalist id="pf-accom-names">
                  {table.names.map((n) => <option key={n} value={n} />)}
                </datalist>
              </th>
            </tr>
          </tbody>
        </table>
      </div>
      {table.notices.map((n) => (
        <p key={n} className="pf-accom-notice">{n}</p>
      ))}
      {menu && (
        <RowMenu at={menu} items={menuItems(menu.ids)} label="Line actions" head={menu.ids.length > 1 ? `${menu.ids.length} lines` : lineOf(menu.ids[0])?.name.value}
          onClose={(restore) => {
            setMenu(null);
            if (restore) menu.trigger?.focus({ preventScroll: true });
          }} />
      )}
    </section>
  );
}
