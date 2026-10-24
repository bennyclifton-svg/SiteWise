// Shared ordered row selection for Program and the document register.
export interface Selection {
  /** Selected ids in the order they were added. */
  ids: readonly string[];
  anchor: string | null;
  /** Rows kept when a Shift range is replaced: the selection before the range. */
  base: readonly string[];
}

export const emptySelection: Selection = { ids: [], anchor: null, base: [] };

export type Mods = { ctrl?: boolean; shift?: boolean };

/** Applies a click with modifiers. `order` is the visible grid order. */
export function clickRow(s: Selection, id: string, order: readonly string[], m: Mods = {}): Selection {
  if (m.shift && s.anchor !== null && order.includes(s.anchor)) {
    const range = rangeOf(order, s.anchor, id);
    if (m.ctrl) {
      // Ctrl+Shift adds a range to what is selected.
      return { ids: union(s.ids, range), anchor: s.anchor, base: s.ids };
    }
    return { ids: union(s.base, range), anchor: s.anchor, base: s.base };
  }
  if (m.ctrl) {
    const on = s.ids.includes(id);
    const ids = on ? s.ids.filter((x) => x !== id) : [...s.ids, id];
    return { ids, anchor: id, base: ids.filter((x) => x !== id) };
  }
  return { ids: [id], anchor: id, base: [] };
}

/** Every visible row (Ctrl/Cmd+A). */
export function selectAll(order: readonly string[]): Selection {
  return { ids: [...order], anchor: order[0] ?? null, base: [] };
}

/** Drops ids no longer visible or no longer in the program. */
export function prune(s: Selection, order: readonly string[]): Selection {
  const keep = new Set(order);
  if (s.ids.every((x) => keep.has(x)) && (s.anchor === null || keep.has(s.anchor))) return s;
  return {
    ids: s.ids.filter((x) => keep.has(x)),
    anchor: s.anchor !== null && keep.has(s.anchor) ? s.anchor : null,
    base: s.base.filter((x) => keep.has(x)),
  };
}

function rangeOf(order: readonly string[], a: string, b: string): string[] {
  const i = order.indexOf(a);
  const j = order.indexOf(b);
  if (i < 0 || j < 0) return [b];
  return order.slice(Math.min(i, j), Math.max(i, j) + 1) as string[];
}

function union(a: readonly string[], b: readonly string[]): string[] {
  const seen = new Set(a);
  const out = [...a];
  for (const x of b) if (!seen.has(x)) (seen.add(x), out.push(x));
  return out;
}
