/** A piece of a term: its wording, or a field slot with the value printed there. */
export type TermPart = { text: string } | { slot: string; value: string };

const SLOT = /\{([a-z_]+(?::[a-z_0-9.]+)?)\}/g;

/**
 * A term split at its {field:x} slots, each with the value its printed text
 * holds there (internal/rfp fills the template; "___" marks a blank). Null
 * when the text is not the template filled, so the term shows as plain text.
 */
export function termParts(template: string, text: string): TermPart[] | null {
  const pieces = template.split(SLOT); // literal, slot, literal, slot, …, literal
  if (pieces.length === 1) return template === text ? [{ text }] : null;
  if (!text.startsWith(pieces[0])) return null;
  const out: TermPart[] = pieces[0] ? [{ text: pieces[0] }] : [];
  let at = pieces[0].length;
  for (let i = 1; i < pieces.length; i += 2) {
    const after = pieces[i + 1];
    const last = i + 2 >= pieces.length;
    // Two slots side by side cannot be told apart.
    if (!after && !last) return null;
    const end = last ? text.length - after.length : text.indexOf(after, at);
    if (end < at || (last && !text.endsWith(after))) return null;
    out.push({ slot: pieces[i], value: text.slice(at, end) });
    if (after) out.push({ text: after });
    at = end + after.length;
  }
  return at === text.length ? out : null;
}

/**
 * Reworded text with its fields put back: each field's value, found in
 * order, becomes its slot again, so the field stays editable where it reads.
 * A value the user changed or removed stays as they wrote it.
 */
export function withSlots(text: string, parts: TermPart[]): string {
  let out = "", rest = text;
  for (const p of parts) {
    if (!("slot" in p) || !p.value) continue;
    const at = rest.indexOf(p.value);
    if (at < 0) continue;
    out += rest.slice(0, at) + `{${p.slot}}`;
    rest = rest.slice(at + p.value.length);
  }
  return out + rest;
}
