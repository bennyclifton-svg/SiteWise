import { expect, test } from "vitest";
import { basisWords, cellWords, columnState, dollars, gapWords, markWords, printedWords, remainderWords, signed, sourceWords, suggestedWords } from "./words";

const cell = (state: "" | "priced" | "included" | "excluded" | "not_priced", amount: string | null = null) => ({ state, amount, lines: 1, mark: "" as const, to_check: false });

test("a cell says its amount, or what the firm said", () => {
  expect(cellWords(cell("priced", "96000.00"))).toBe("$96,000.00");
  expect(cellWords(cell("priced"))).toBe("Amount to type");
  expect(cellWords(cell("included"))).toBe("Included");
  expect(cellWords(cell("excluded"))).toBe("Excluded");
  expect(cellWords(cell("not_priced"))).toBe("Not priced");
  expect(cellWords(cell(""))).toBe("—");
  expect(cellWords(undefined)).toBe("—");
});

test("marks are one plain word or phrase", () => {
  // High and low marks were removed (owner, 15 Nov 2026).
  expect(markWords("high")).toBe("");
  expect(markWords("gap")).toBe("Others priced this");
  expect(markWords("")).toBe("");
});

test("the remainder says why it is there", () => {
  expect(remainderWords("")).toBe("");
  expect(remainderWords("rounding")).toBe("Rounding");
  expect(remainderWords("margin:15")).toBe("Margin not shown at line level (15%)");
  expect(remainderWords("margin:7.5")).toBe("Margin not shown at line level (7.5%)");
  expect(remainderWords("unbroken")).toBe("The firm did not break this down");
  expect(remainderWords("differs")).toBe("Lines differ from the tender sum");
});

test("a column says where its reading stands", () => {
  expect(columnState({ state: "pending", error: "", checked: false })).toBe("Waiting to read prices");
  expect(columnState({ state: "reading", error: "", checked: false })).toBe("Reading prices…");
  expect(columnState({ state: "failed", error: "The model declined to read this file.", checked: false })).toBe("The model declined to read this file.");
  expect(columnState({ state: "failed", error: "", checked: false })).toBe("Prices not read");
  // A read column carries no "to check" word: Award is the acceptance.
  expect(columnState({ state: "read", error: "", checked: false })).toBe("");
});

test("a price that is not fixed says so", () => {
  expect(basisWords("fixed")).toBe("");
  expect(basisWords("unknown")).toBe("");
  expect(basisWords("estimate")).toBe("Estimate, not a fixed price");
  expect(basisWords("cost_plus")).toBe("Cost plus, not a fixed price");
  expect(basisWords("rates_only")).toBe("Rates only");
});

test("adds and subs carry their sign and where the figure came from", () => {
  expect(signed("15000.00")).toBe("+$15,000.00");
  expect(signed("-34000.00")).toBe("−$34,000.00");
  expect(signed(null)).toBe("—");
  expect(sourceWords({ source: "firm", page: 34 })).toBe("the firm's figure, p. 34");
  expect(sourceWords({ source: "firm", page: 0 })).toBe("the firm's figure");
  expect(sourceWords({ source: "others_highest", page: 0 })).toBe("highest of the others");
  expect(sourceWords({ source: "others_average", page: 0 })).toBe("average of the others");
  expect(sourceWords({ source: "typed", page: 0 })).toBe("typed by you");
  expect(gapWords({ low: "24000.00", high: "126000.00", average: "75000.00", count: 2 })).toBe("Not priced. Others $24,000.00 to $126,000.00");
  expect(gapWords({ low: "5000.00", high: "5000.00", average: "5000.00", count: 1 })).toBe("Not priced. Others $5,000.00");
  expect(gapWords(undefined)).toBe("Not priced");
  expect(suggestedWords(3)).toBe("3 suggestions not applied");
  expect(suggestedWords(1)).toBe("1 suggestion not applied");
  expect(suggestedWords(0)).toBe("");
});

test("whole dollars round half up and keep the grouping", () => {
  expect(dollars("1234567.49", false)).toBe("$1,234,567");
  expect(dollars("999.50", false)).toBe("$1,000");
  expect(dollars("-12.5", false)).toBe("-$13");
  expect(signed("15000.00", false)).toBe("+$15,000");
  expect(cellWords(cell("priced", "96000.00"), false)).toBe("$96,000");
  expect(gapWords({ low: "5000.40", high: "5000.40", average: "5000.40", count: 1 }, false)).toBe("Not priced. Others $5,000");
  expect(dollars("1234.5")).toBe("$1,234.50");
});

test("a line printed with GST says what was printed", () => {
  expect(printedWords({ printed: "$105,600.00", gst: "inc", amount: "96000.00" })).toBe("printed $105,600.00 inc GST");
  expect(printedWords({ printed: "$96,000.00", gst: "ex", amount: "96000.00" })).toBe("");
  expect(printedWords({ printed: "", gst: "inc", amount: null })).toBe("");
});
