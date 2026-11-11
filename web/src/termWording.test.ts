import { describe, expect, it } from "vitest";
import { termParts, withSlots } from "./termWording";

describe("termParts", () => {
  it("splits a term at its fields, each with the value printed there", () => {
    expect(termParts("Your proposal stays valid for {field:validity_days} days.", "Your proposal stays valid for 90 days.")).toEqual([
      { text: "Your proposal stays valid for " }, { slot: "field:validity_days", value: "90" }, { text: " days." },
    ]);
  });
  it("reads two fields and a blank", () => {
    expect(termParts("Hold PI of {field:pi} and PL of {field:pl}.", "Hold PI of $10 million and PL of ___.")).toEqual([
      { text: "Hold PI of " }, { slot: "field:pi", value: "$10 million" }, { text: " and PL of " }, { slot: "field:pl", value: "___" }, { text: "." },
    ]);
  });
  it("keeps a term with no fields as one piece", () => {
    expect(termParts("No mobilisation payment.", "No mobilisation payment.")).toEqual([{ text: "No mobilisation payment." }]);
  });
  it("gives up when the text is not the template filled", () => {
    expect(termParts("Valid for {field:validity_days} days.", "Something else.")).toBeNull();
    expect(termParts("{field:a}{field:b}", "12")).toBeNull();
  });
});

describe("withSlots", () => {
  const parts = termParts("Your proposal stays valid for {field:validity_days} days.", "Your proposal stays valid for 90 days.")!;
  it("puts each field back where its value still reads", () => {
    expect(withSlots("Proposals stay open for 90 days from closing.", parts)).toBe("Proposals stay open for {field:validity_days} days from closing.");
  });
  it("leaves a sentence that dropped the value as written", () => {
    expect(withSlots("Proposals stay open for sixty days.", parts)).toBe("Proposals stay open for sixty days.");
  });
  it("puts a blank back as its field", () => {
    const blank = termParts("Practical completion by {field:completion}.", "Practical completion by ___.")!;
    expect(withSlots("Complete the works by ___.", blank)).toBe("Complete the works by {field:completion}.");
  });
  it("matches values in order", () => {
    const two = termParts("Pay in {field:a} days, hold {field:b} days.", "Pay in 30 days, hold 30 days.")!;
    expect(withSlots("Pay within 30 days; hold for 30 days.", two)).toBe("Pay within {field:a} days; hold for {field:b} days.");
  });
});
