import { expect, test } from "vitest";
import { soleQuotedFee } from "./quoteAward";
import type { Column, QuoteFee } from "./submissionApi";

const fee = (id: string, amount: string | null, role = "Package / fee — scope awaiting alignment", extra: Partial<QuoteFee> = {}): QuoteFee =>
  ({ id, label: `Fee ${id}`, printed: amount ? `$${amount}` : "$?", amount, role, page: 0, source: `Fee ${id}`, heading: "", suggestion: "Current offer", ...extra });
const col = (fees: QuoteFee[] | null) => ({ quote: fees ? { fees, coverage: {}, issues: [] } : undefined } as unknown as Column);

test("an appointment's single fee is the one to award", () => {
  expect(soleQuotedFee(col([fee("a1", "6800.00")]))?.id).toBe("a1");
});

test("the same fee printed twice is still one offer", () => {
  expect(soleQuotedFee(col([fee("a1", "6800.00"), fee("a2", "6800.00", "Package total")]))?.amount).toBe("6800.00");
});

test("two different offers leave the choice to the PM", () => {
  expect(soleQuotedFee(col([fee("a1", "6800.00"), fee("a2", "9200.00", "Alternative package")]))).toBeNull();
});

test("no quote, an unclear tax basis, a conditional fee or stage lines only offer nothing", () => {
  expect(soleQuotedFee(col(null))).toBeNull();
  expect(soleQuotedFee(col([fee("a1", null)]))).toBeNull();
  expect(soleQuotedFee(col([fee("a1", "500.00", "Package / fee", { suggestion: "Conditional / optional" })]))).toBeNull();
  expect(soleQuotedFee(col([fee("a1", "400.00", "Breakdown component"), fee("a2", "600.00", "Breakdown component")]))).toBeNull();
});
