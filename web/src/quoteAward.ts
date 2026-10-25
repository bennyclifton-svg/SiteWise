import type { Column, QuoteFee } from "./submissionApi";

// The offers a quote headlines: package totals and alternatives, not their
// breakdowns or earlier offers.
export function headlineFees(fees: QuoteFee[]) {
  const offers = fees.filter(f => (f.role.includes("Package") || f.role.includes("Alternative")) && !f.suggestion?.startsWith("Earlier"));
  // A bare repeated figure adds no scope information. Keep the named offer
  // prominent; every occurrence remains available in the source fee lines.
  return offers.filter(f => f.label !== "Quoted fee" || !offers.some(other => other.id !== f.id && (f.amount !== null ? other.amount === f.amount : other.printed === f.printed) && other.label !== "Quoted fee"));
}

// The one fee a proposal offers, when it offers exactly one amount ex GST:
// an appointment letter's "Fee: $6,800 + GST". Two different offers, a
// conditional fee or an unclear tax basis leave the PM to choose.
export function soleQuotedFee(col: Column): QuoteFee | null {
  if (!col.quote) return null;
  const offers = headlineFees(col.quote.fees).filter(f => f.amount !== null && !f.role.startsWith("Conditional") && !f.suggestion?.startsWith("Conditional"));
  return offers.length > 0 && offers.every(f => f.amount === offers[0].amount) ? offers[0] : null;
}

// The confirmation note for a fee taken from the source, as the PM's record.
export const feeNote = (f: QuoteFee) => `${f.label}. Source page ${f.page}: ${f.source}`.slice(0, 500);
