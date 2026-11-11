import { describe, expect, it } from "vitest";
import { whenLabel } from "./ProjectsPage";

describe("whenLabel", () => {
  const now = new Date(2026, 10, 11, 18, 0);
  it("names recent days in words", () => {
    expect(whenLabel(new Date(2026, 10, 11, 7, 30).toISOString(), now)).toBe("Today");
    expect(whenLabel(new Date(2026, 10, 10, 23, 59).toISOString(), now)).toBe("Yesterday");
    expect(whenLabel(new Date(2026, 10, 8, 12, 0).toISOString(), now)).toBe("3 days ago");
  });
  it("gives a date after a week, with the year only when it differs", () => {
    expect(whenLabel(new Date(2026, 9, 28, 12, 0).toISOString(), now)).toBe("28 Oct");
    expect(whenLabel(new Date(2025, 9, 24, 12, 0).toISOString(), now)).toBe("24 Oct 2025");
  });
});
