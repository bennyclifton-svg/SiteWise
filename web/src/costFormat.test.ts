import { describe, expect, it } from "vitest";
import { fmt } from "./CostGrid";

describe("fmt", () => {
  it("keeps cents by default for every other screen", () => {
    expect(fmt("1234567.5")).toBe("1,234,567.50");
    expect(fmt("-12.34")).toBe("−12.34");
    expect(fmt(null)).toBe("—");
  });
  it("rounds half away from zero to whole dollars without floats", () => {
    expect(fmt("96500.00", false)).toBe("96,500");
    expect(fmt("999.49", false)).toBe("999");
    expect(fmt("999.50", false)).toBe("1,000");
    expect(fmt("-0.40", false)).toBe("0");
    expect(fmt("-2.5", false)).toBe("−3");
    expect(fmt("90071992547409919.99", false)).toBe("90,071,992,547,409,920");
  });
});
