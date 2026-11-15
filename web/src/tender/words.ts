// Every word the tender tables show, as pure functions, so the wording is
// tested apart from the layout. Plain words, no jargon.
import type { TenderAmount, TenderCell, TenderColumn, TenderLine, TenderRange } from "./tenderApi";

/** Groups the thousands of an exact decimal string. With cents off it
 * shows whole dollars, half a dollar rounding up (the Cost Plan's rule). It
 * formats; it never computes a figure that is stored. */
export function fmt(m: string, cents = true): string {
  const [w, c = ""] = m.split(".");
  const part = c.padEnd(2, "0");
  const whole = !cents && part[0] >= "5" ? (BigInt(w || "0") + 1n).toString() : w;
  let out = "";
  for (let i = 0; i < whole.length; i++) {
    if (i > 0 && (whole.length - i) % 3 === 0) out += ",";
    out += whole[i];
  }
  return cents ? `${out}.${part}` : out;
}

export const dollars = (m: string | null | undefined, cents = true) => (m == null ? "—" : m.startsWith("-") ? `-$${fmt(m.slice(1), cents)}` : `$${fmt(m, cents)}`);

/** A signed amount for the second table: "+$15,000.00", "−$34,000.00". */
export function signed(m: string | null | undefined, cents = true): string {
  if (m == null) return "—";
  return m.startsWith("-") ? `−$${fmt(m.slice(1), cents)}` : `+$${fmt(m, cents)}`;
}

export function cellWords(cell: TenderCell | undefined, cents = true): string {
  if (!cell || cell.state === "") return "—";
  if (cell.state === "priced") return cell.amount == null ? "Amount to type" : dollars(cell.amount, cents);
  return { included: "Included", excluded: "Excluded", not_priced: "Not priced" }[cell.state];
}

// High and low marks were removed (owner, 15 Nov 2026); only a gap is marked.
export function markWords(mark: string): string {
  return { gap: "Others priced this" }[mark] ?? "";
}

export function remainderWords(why: string): string {
  if (why === "rounding") return "Rounding";
  if (why.startsWith("margin:")) return `Margin not shown at line level (${why.slice("margin:".length)}%)`;
  if (why === "unbroken") return "The firm did not break this down";
  if (why === "differs") return "Lines differ from the tender sum";
  return "";
}

export function columnState(col: Pick<TenderColumn, "state" | "error" | "checked">): string {
  if (col.state === "pending") return "Waiting to read prices";
  if (col.state === "reading") return "Reading prices…";
  if (col.state === "failed") return col.error || "Prices not read";
  // A read column needs no word: pressing Award is the manager's acceptance.
  return "";
}

export function basisWords(basis: string): string {
  return { estimate: "Estimate, not a fixed price", cost_plus: "Cost plus, not a fixed price", rates_only: "Rates only" }[basis] ?? "";
}

/** What a line shows after its amount when the printed figure differs. */
export function printedWords(line: Pick<TenderLine, "printed" | "gst" | "amount">): string {
  if (!line.printed || line.gst !== "inc") return "";
  return `printed ${line.printed.startsWith("$") ? line.printed : `$${line.printed}`} inc GST`;
}

/** A line's hover text: everything its compact row leaves out — the full
 * wording, the firm's heading, the printed inc GST figure, warnings and notes. */
export function lineTip(l: Pick<TenderLine, "label" | "heading" | "printed" | "gst" | "amount" | "flags" | "note" | "counted">): string {
  return [
    l.label,
    l.heading ? `Under "${l.heading}" in the firm's document` : "",
    printedWords(l) ? `${printedWords(l).replace(/^p/, "P")}` : "",
    ...l.flags.map(f => f.text),
    l.note,
    l.counted ? "" : "Not counted",
  ].filter(Boolean).join("\n");
}

export function sourceWords(a: Pick<TenderAmount, "source" | "page">): string {
  if (a.source === "firm") return a.page ? `the firm's figure, p. ${a.page}` : "the firm's figure";
  if (a.source === "others_highest") return "highest of the others";
  if (a.source === "others_average") return "average of the others";
  if (a.source === "typed") return "typed by you";
  return "";
}

/** What a gap says before the manager chooses a figure for it. */
export function gapWords(others: TenderRange | undefined, cents = true): string {
  if (!others || others.count === 0) return "Not priced";
  if (others.low === others.high) return `Not priced. Others ${dollars(others.low, cents)}`;
  return `Not priced. Others ${dollars(others.low, cents)} to ${dollars(others.high, cents)}`;
}

export function suggestedWords(n: number): string {
  if (n === 0) return "";
  return `${n} suggestion${n === 1 ? "" : "s"} not applied`;
}
