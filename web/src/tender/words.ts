// Every word the tender tables show, as pure functions, so the wording is
// tested apart from the layout. Plain words, no jargon.
import type { TenderAmount, TenderCell, TenderColumn, TenderLine, TenderRange } from "./tenderApi";

/** Groups the thousands of an exact decimal string. It formats; it never
 * computes. */
export function fmt(m: string): string {
  const [whole, cents = "00"] = m.split(".");
  let out = "";
  for (let i = 0; i < whole.length; i++) {
    if (i > 0 && (whole.length - i) % 3 === 0) out += ",";
    out += whole[i];
  }
  return `${out}.${cents.padEnd(2, "0")}`;
}

export const dollars = (m: string | null | undefined) => (m == null ? "—" : m.startsWith("-") ? `-$${fmt(m.slice(1))}` : `$${fmt(m)}`);

/** A signed amount for the second table: "+$15,000.00", "−$34,000.00". */
export function signed(m: string | null | undefined): string {
  if (m == null) return "—";
  return m.startsWith("-") ? `−$${fmt(m.slice(1))}` : `+$${fmt(m)}`;
}

export function cellWords(cell: TenderCell | undefined): string {
  if (!cell || cell.state === "") return "—";
  if (cell.state === "priced") return cell.amount == null ? "Amount to type" : dollars(cell.amount);
  return { included: "Included", excluded: "Excluded", not_priced: "Not priced" }[cell.state];
}

export function markWords(mark: string): string {
  return { high: "High", low: "Low", gap: "Others priced this" }[mark] ?? "";
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
  return col.checked ? "Checked" : "To check";
}

export function basisWords(basis: string): string {
  return { estimate: "Estimate, not a fixed price", cost_plus: "Cost plus, not a fixed price", rates_only: "Rates only" }[basis] ?? "";
}

/** What a line shows after its amount when the printed figure differs. */
export function printedWords(line: Pick<TenderLine, "printed" | "gst" | "amount">): string {
  if (!line.printed || line.gst !== "inc") return "";
  return `printed ${line.printed.startsWith("$") ? line.printed : `$${line.printed}`} inc GST`;
}

export function sourceWords(a: Pick<TenderAmount, "source" | "page">): string {
  if (a.source === "firm") return a.page ? `the firm's figure, p. ${a.page}` : "the firm's figure";
  if (a.source === "others_highest") return "highest of the others";
  if (a.source === "others_average") return "average of the others";
  if (a.source === "typed") return "typed by you";
  return "";
}

/** What a gap says before the manager chooses a figure for it. */
export function gapWords(others: TenderRange | undefined): string {
  if (!others || others.count === 0) return "Not priced";
  if (others.low === others.high) return `Not priced. Others ${dollars(others.low)}`;
  return `Not priced. Others ${dollars(others.low)} to ${dollars(others.high)}`;
}

export function suggestedWords(n: number): string {
  if (n === 0) return "";
  return `${n} suggestion${n === 1 ? "" : "s"} not applied`;
}
