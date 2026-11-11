import { describe, expect, it } from "vitest";
import { optionNames, shortLabel } from "./QuickField";
import type { ProfileField } from "./profileApi";

const field = (label: string, extra: Partial<ProfileField> = {}): ProfileField => ({
  key: "k", label, part_id: "whole", kind: "choice", value: "", band: "", sources: [], alternatives: [], ...extra,
});
const opts = (...labels: string[]) => labels.map((label, i) => ({ id: String(i), label }));

describe("shortLabel", () => {
  it("drops 'selected' and 'allocated' from appointment choices only", () => {
    expect(shortLabel(field("Declarations selected", { scope_decision: true }))).toBe("Declarations");
    expect(shortLabel(field("Dry design allocated", { scope_decision: true }))).toBe("Dry design");
    expect(shortLabel(field("Upgrade selected"))).toBe("Upgrade selected");
  });
  it("moves a bracketed standard to the hover", () => {
    expect(shortLabel(field("Bushfire attack level (AS 3959)"))).toBe("Bushfire attack level");
  });
});

describe("optionNames", () => {
  it("strips a prefix every option repeats", () => {
    expect(optionNames(opts("BAL-LOW", "BAL-12.5", "BAL-FZ"))).toEqual(["LOW", "12.5", "FZ"]);
    expect(optionNames(opts("Zone 1", "Zone 2"))).toEqual(["1", "2"]);
  });
  it("keeps names with no shared prefix, or where one would empty", () => {
    expect(optionNames(opts("Yes", "No"))).toEqual(["Yes", "No"]);
    expect(optionNames(opts("CC1", "CC2"))).toEqual(["CC1", "CC2"]);
    expect(optionNames(opts("HIL-A", "HIL-"))).toEqual(["HIL-A", "HIL-"]);
  });
});
